//go:build browserintegration

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

// Every invocation is a separate native executable, including observation and
// stale actions. Browser installation is an explicit integration prerequisite.
func TestBrowserNativeCLI(t *testing.T) {
	browser := os.Getenv("AGENT_ENV_BROWSER_EXECUTABLE")
	if browser == "" {
		t.Fatal("browserintegration requires AGENT_ENV_BROWSER_EXECUTABLE pointing to a native Chromium executable")
	}
	browser, err := filepath.Abs(browser)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(browser); err != nil || info.IsDir() {
		t.Fatalf("browser executable %q: %v", browser, err)
	}
	t.Setenv("PATH", filepath.Dir(browser)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_COUNT", "0")
	base, err := os.MkdirTemp("", "agent-env-browser-日本語-")
	if err != nil {
		t.Fatal(err)
	}
	retain := false
	t.Cleanup(func() {
		if retain {
			t.Logf("retained uncertain browser fixture: %s", base)
		} else if err := os.RemoveAll(base); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("AGENT_ENV_HOME", filepath.Join(base, "state home"))
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir, name string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture %s %v: %v\n%s", name, args, err, out)
		}
		return string(out)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	cliPath := filepath.Join(base, "agent-env"+suffix)
	run(root, "go", "build", "-o", cliPath, "./cmd/agent-env")
	repo := filepath.Join(base, "source repository")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	backendName := "fixture-backend" + suffix
	run(root, "go", "build", "-o", filepath.Join(repo, backendName), "./internal/cli/testdata/browser")
	browserArg, _ := json.Marshal(filepath.Base(browser))
	backendArg, _ := json.Marshal("./" + backendName)
	extra := ""
	if os.Getenv("AGENT_ENV_BROWSER_NO_SANDBOX") == "1" {
		extra = ", \"--no-sandbox\""
		t.Log("explicit fixture-only --no-sandbox requested")
	}
	manifest := fmt.Sprintf(`version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  backend:
    type: process
    source: self
    working_directory: .
    command: [%s, "--port", "${port:http}"]
    ports: {http: {protocol: tcp}}
  browser-process:
    type: process
    source: self
    working_directory: .
    command: [%s, "--headless=new", "--enable-automation", "--user-data-dir=${runtime_dir}/profile", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=${port:cdp}", "--no-first-run", "--window-size=1200,900", "--no-default-browser-check"%s, "about:blank"]
    ports: {cdp: {protocol: tcp}}
browsers:
  web: {type: chromium-cdp, runtime: browser-process, cdp_port: cdp}
components:
  backend:
    runtime: backend
    endpoints: {http: {runtime_port: http}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:http}/health", timeout: 30s}
  browser:
    runtime: browser-process
    depends_on: [backend]
    endpoints: {cdp: {runtime_port: cdp}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:cdp}/json/version", timeout: 30s}
stacks:
  review: {roots: [browser]}
`, backendArg, browserArg, extra)
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	run(repo, "git", "-c", "init.templateDir=", "init", "-b", "main")
	run(repo, "git", "add", ".agent-env.yaml", backendName)
	run(repo, "git", "-c", "commit.gpgsign=false", "-c", "user.name=Browser Fixture", "-c", "user.email=browser@example.invalid", "commit", "-m", "Native browser fixture")
	invoke := func(args ...string) (json.RawMessage, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, cliPath, append([]string{"--output=json", "--owner=browser-fixture"}, args...)...)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if out.Len() > 0 {
			if e := json.Unmarshal(out.Bytes(), &envelope); e != nil {
				return nil, fmt.Errorf("invalid JSON: %w: %s", e, &out)
			}
		}
		if err != nil {
			return envelope.Data, fmt.Errorf("CLI %v: %w: %s %s", args, err, &stderr, &out)
		}
		return envelope.Data, nil
	}
	must := func(args ...string) json.RawMessage {
		t.Helper()
		data, err := invoke(args...)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	t.Cleanup(func() {
		data, err := invoke("list", "--cached")
		if err != nil {
			retain = true
			t.Errorf("list cleanup: %v", err)
			return
		}
		var leases []domain.Lease
		if err := json.Unmarshal(data, &leases); err != nil {
			retain = true
			t.Error(err)
			return
		}
		for _, l := range leases {
			if _, err := invoke("destroy", l.ID, "--force"); err != nil {
				retain = true
				t.Errorf("cleanup %s: %v", l.ID, err)
			}
		}
	})
	var leases []domain.Lease
	for range 2 {
		var l domain.Lease
		if err := json.Unmarshal(must("create", repo, "--stack", "review"), &l); err != nil {
			t.Fatal(err)
		}
		if l.Observed != "ready" {
			t.Fatalf("create state %s", l.Observed)
		}
		leases = append(leases, l)
	}
	process := func(l domain.Lease, name string) *domain.PersistentProcess {
		t.Helper()
		for _, r := range l.Runtimes {
			if r.Name == name {
				return r.Process
			}
		}
		t.Fatalf("runtime %s missing", name)
		return nil
	}
	a, b := process(leases[0], "browser-process"), process(leases[1], "browser-process")
	if a.StateDirectory == b.StateDirectory || a.Ports["cdp"] == b.Ports["cdp"] || a.ProcessID == b.ProcessID {
		t.Fatal("leases share browser identity/profile/port")
	}
	type result struct {
		Observation domain.BrowserObservation `json:"observation"`
		Snapshot    *domain.BrowserSnapshot   `json:"snapshot"`
		Artifacts   []domain.Artifact         `json:"artifacts"`
	}
	call := func(op string, args ...string) result {
		t.Helper()
		raw := must(append([]string{"browser", op, leases[0].ID, "--browser", "web"}, args...)...)
		var r result
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	cap := call("capabilities")
	if cap.Observation.Identity.Product == "" || cap.Observation.Identity.Protocol == "" {
		t.Fatalf("missing browser identity: %+v", cap)
	}
	t.Logf("native OS=%s arch=%s product=%s protocol=%s", runtime.GOOS, runtime.GOARCH, cap.Observation.Identity.Product, cap.Observation.Identity.Protocol)
	pages := call("pages").Observation.Pages
	if len(pages) != 1 {
		t.Fatalf("initial pages: %+v", pages)
	}
	if pages[0].URL != "about:blank" {
		t.Fatal("capabilities navigated original page")
	}
	page := pages[0].ID
	backend := process(leases[0], "backend")
	url := fmt.Sprintf("http://127.0.0.1:%d/page.html", backend.Ports["http"])
	call("navigate", "--page", page, "--url", url)
	call("wait", "--page", page, "--wait-for", "text", "--contains", "Browser fixture", "--timeout", "10s")
	snapshot := func() *domain.BrowserSnapshot {
		t.Helper()
		r := call("snapshot", "--page", page)
		s := r.Snapshot
		if s == nil {
			s = r.Observation.Snapshot
		}
		if s == nil || len(s.Nodes) == 0 {
			t.Fatalf("empty snapshot: %+v", r)
		}
		return s
	}
	find := func(s *domain.BrowserSnapshot, role, name string) domain.BrowserNode {
		t.Helper()
		for _, n := range s.Nodes {
			if n.Role == role && n.Name == name {
				return n
			}
		}
		t.Fatalf("node %s %q absent: %+v", role, name, s.Nodes)
		return domain.BrowserNode{}
	}
	deadlineStart := time.Now()
	if _, err := invoke("browser", "wait", leases[0].ID, "--browser", "web", "--page", page, "--wait-for", "text", "--contains", "not-present-timeout-fixture", "--timeout", "250ms"); err == nil {
		t.Fatal("missing text wait unexpectedly succeeded")
	}
	if time.Since(deadlineStart) > 5*time.Second {
		t.Fatal("wait did not respect bounded timeout")
	}
	s := snapshot()
	find(s, "heading", "Browser fixture")
	find(s, "button", "Shadow action")
	find(s, "heading", "Iframe heading")
	for _, n := range s.Nodes {
		if strings.Contains(n.Value, "native-password-secret") {
			t.Fatal("password leaked in AX snapshot")
		}
	}
	action := func(op string, s *domain.BrowserSnapshot, n domain.BrowserNode, args ...string) result {
		t.Helper()
		return call(op, append([]string{"--page", page, "--snapshot", s.ID, "--node", n.Ref}, args...)...)
	}
	unicode := "日本語 🧪 café ' \\\""
	typed := action("set-text", s, find(s, "textbox", "Unicode text"), "--text", unicode)
	if !typed.Observation.ReadbackEqual {
		t.Fatalf("Unicode readback failed: %+v", typed)
	}
	s = snapshot()
	if n := find(s, "textbox", "Unicode text"); n.Value != "" {
		t.Fatalf("editable value persisted: %q", n.Value)
	}
	cleared := action("set-text", s, find(s, "textbox", "Unicode text"), "--text", "")
	if !cleared.Observation.ReadbackEqual {
		t.Fatal("empty-string replacement did not clear input")
	}
	s = snapshot()
	named := action("set-text", s, find(s, "textbox", "Unicode text"), "--text", "web")
	if !named.Observation.ReadbackEqual {
		t.Fatal("browser-name text readback failed")
	}
	s = snapshot()
	if s.Browser != "web" {
		t.Fatal("typed text corrupted browser authority")
	}
	reentered := action("set-text", s, find(s, "textbox", "Unicode text"), "--text", unicode)
	if !reentered.Observation.ReadbackEqual {
		t.Fatal("Unicode reentry failed")
	}
	s = snapshot()
	action("key", s, find(s, "textbox", "Unicode text"), "--key", "Enter")
	call("wait", "--page", page, "--wait-for", "text", "--contains", "Enter received", "--timeout", "5s")
	s = snapshot()
	action("click", s, find(s, "button", "Send request"))
	call("wait", "--page", page, "--wait-for", "text", "--contains", "Request count 1", "--timeout", "5s")
	client := http.Client{Timeout: 3 * time.Second}
	lastResponse, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/last", backend.Ports["http"]))
	if err != nil {
		t.Fatal(err)
	}
	lastBody, err := io.ReadAll(lastResponse.Body)
	lastResponse.Body.Close()
	if err != nil || string(lastBody) != unicode {
		t.Fatalf("Unicode backend round trip: %q %v", lastBody, err)
	}
	logs := must("logs", leases[0].ID)
	if !bytes.Contains(logs, []byte("browser-fixture click=1")) {
		t.Fatalf("backend request uncorrelated: %s", logs)
	}
	s = snapshot()
	victim := find(s, "button", "Replaceable action")
	action("click", s, find(s, "button", "Replace node"))
	if _, err := invoke("browser", "click", leases[0].ID, "--browser", "web", "--page", page, "--snapshot", s.ID, "--node", victim.Ref); err == nil {
		t.Fatal("replaced node accepted")
	}
	staleResponse, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/count", backend.Ports["http"]))
	if err != nil {
		t.Fatal(err)
	}
	staleCount, err := io.ReadAll(staleResponse.Body)
	staleResponse.Body.Close()
	if err != nil || string(staleCount) != "1" {
		t.Fatalf("stale click caused backend effect: %q %v", staleCount, err)
	}
	s = snapshot()
	action("scroll", s, find(s, "region", "Scrollable area"), "--delta-y", "300")
	call("wait", "--page", page, "--wait-for", "text", "--contains", "Scrolled", "--timeout", "5s")
	dom := call("dom-snapshot", "--page", page)
	domFound := false
	for _, artifact := range dom.Artifacts {
		if artifact.Kind != "browser-dom" {
			continue
		}
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		var evidence struct {
			Version int               `json:"version"`
			Nodes   []json.RawMessage `json:"nodes"`
		}
		if err := json.Unmarshal(data, &evidence); err != nil {
			t.Fatal(err)
		}
		if evidence.Version != 1 || len(evidence.Nodes) == 0 || len(evidence.Nodes) > 2048 {
			t.Fatalf("invalid bounded DOM evidence: version=%d nodes=%d", evidence.Version, len(evidence.Nodes))
		}
		if bytes.Contains(data, []byte("native-password-secret")) {
			t.Fatal("password leaked in DOM")
		}
		domFound = true
	}
	if !domFound {
		t.Fatal("DOM evidence absent")
	}
	shot := call("screenshot", "--page", page)
	if shot.Observation.Page.ID != page || shot.Observation.Identity != cap.Observation.Identity {
		t.Fatal("screenshot identity attribution mismatch")
	}
	pngFound := false
	for _, artifact := range shot.Artifacts {
		if strings.HasSuffix(artifact.Path, ".png") {
			if artifact.LeaseID != leases[0].ID || artifact.RunID == "" {
				t.Fatal("screenshot artifact attribution missing")
			}
			data, err := os.ReadFile(artifact.Path)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
				t.Fatalf("invalid PNG: %v", err)
			}
			sum := sha256.Sum256(data)
			if artifact.Digest != hex.EncodeToString(sum[:]) {
				t.Fatal("PNG digest differs")
			}
			pngFound = true
		}
	}
	if !pngFound {
		t.Fatal("PNG artifact absent")
	}
	console := call("console", "--page", page, "--duration", "1s")
	consoleFound := false
	for _, event := range console.Observation.Console {
		if strings.Contains(event.Text, "fixture-console-heartbeat") {
			consoleFound = true
		}
	}
	if !consoleFound {
		t.Fatalf("console heartbeat absent: %+v", console.Observation.Console)
	}
	redactedEcho := false
	for _, event := range console.Observation.Console {
		if strings.Contains(event.Text, unicode) {
			t.Fatal("later console exposed prior text")
		}
		if event.Text == "[REDACTED]" {
			redactedEcho = true
		}
	}
	if !redactedEcho {
		t.Fatalf("fixture did not produce redacted later text echo: %+v", console.Observation.Console)
	}
	for _, artifact := range console.Artifacts {
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		var body any
		if err = json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		normalized, _ := json.Marshal(body)
		if bytes.Contains(normalized, []byte("日本語")) {
			t.Fatal("prior text persisted in console evidence")
		}
	}

	network := call("network", "--page", page, "--duration", "1s")
	networkFound := false
	requests := map[string]bool{}
	for _, event := range network.Observation.Network {
		if strings.Contains(event.URL, "/tick") && event.Method == "GET" {
			requests[event.ID] = true
		}
	}
	for _, event := range network.Observation.Network {
		if requests[event.ID] && event.Status == 200 {
			networkFound = true
		}
	}
	if !networkFound {
		t.Fatalf("network heartbeat absent: %+v", network.Observation.Network)
	}
	for _, artifact := range network.Artifacts {
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("native-header-secret")) || bytes.Contains(data, []byte("never-record")) {
			t.Fatal("network credential leaked")
		}
	}
	if _, err := invoke("browser", "click", leases[1].ID, "--browser", "web", "--snapshot", s.ID, "--node", find(s, "button", "Send request").Ref); err == nil {
		t.Fatal("cross-lease snapshot accepted")
	}
	created := call("page-create")
	newPage := created.Observation.Page.ID
	if newPage == "" {
		t.Fatal("created page missing")
	}
	if _, err := invoke("browser", "snapshot", leases[0].ID, "--browser", "web"); err == nil {
		t.Fatal("multiple pages selected implicitly")
	}
	if _, err := invoke("browser", "click", leases[0].ID, "--browser", "web", "--page", newPage, "--snapshot", s.ID, "--node", find(s, "button", "Send request").Ref); err == nil {
		t.Fatal("cross-page snapshot accepted")
	}
	call("page-close", "--page", newPage)
	call("navigate", "--page", page, "--url", fmt.Sprintf("http://127.0.0.1:%d/next", backend.Ports["http"]))
	if _, err := invoke("browser", "click", leases[0].ID, "--browser", "web", "--page", page, "--snapshot", s.ID, "--node", find(s, "button", "Send request").Ref); err == nil {
		t.Fatal("pre-navigation snapshot accepted")
	}
	call("wait", "--page", page, "--wait-for", "url", "--contains", "/next", "--timeout", "5s")
	// Independent second lease stays on its original blank page and backend state.
	raw := must("browser", "pages", leases[1].ID, "--browser", "web")
	var other result
	if err := json.Unmarshal(raw, &other); err != nil {
		t.Fatal(err)
	}
	if len(other.Observation.Pages) != 1 || other.Observation.Pages[0].URL != "about:blank" {
		t.Fatalf("second lease changed: %+v", other)
	}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/count", process(leases[1], "backend").Ports["http"]))
	if err != nil {
		t.Fatal(err)
	}
	count, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(count) != "0" {
		t.Fatalf("second backend changed: %q %v", count, err)
	}
	// Simulate manual browser death without involving browser-layer cleanup.
	killed, err := os.FindProcess(b.ProcessID)
	if err != nil {
		t.Fatal(err)
	}
	if err := killed.Kill(); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke("browser", "pages", leases[1].ID, "--browser", "web"); err == nil {
		t.Fatal("browser operation accepted after manual process death")
	}
	var shown struct {
		Lease domain.Lease `json:"lease"`
	}
	if err := json.Unmarshal(must("show", leases[1].ID), &shown); err != nil {
		t.Fatal(err)
	}
	if process(shown.Lease, "browser-process").ProcessID != b.ProcessID || shown.Lease.Observed == "ready" {
		t.Fatal("manual death restarted browser or retained READY")
	}

	for _, l := range leases {
		must("destroy", l.ID)
		for _, r := range l.Runtimes {
			if _, err := os.Stat(r.Process.StateDirectory); !os.IsNotExist(err) {
				t.Fatalf("profile/state retained after confirmed destroy: %s %v", r.Name, err)
			}
		}
	}
}
