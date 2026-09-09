//go:build multihostintegration

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/worker"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type remoteRuntimeFixture struct {
	base   string
	remote func(...string) json.RawMessage
	local  func(string, string, ...string) string
}

func newRemoteRuntimeFixture(t *testing.T) remoteRuntimeFixture {
	t.Helper()
	base, createErr := os.MkdirTemp("", "agent-env-multihost-native-")
	if createErr != nil {
		t.Fatal(createErr)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retained uncertain fixture state: %s", base)
		} else if err := os.RemoveAll(base); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	ctx := context.Background()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_COUNT", "0")
	root, e := filepath.Abs(filepath.Join("..", ".."))
	if e != nil {
		t.Fatal(e)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	binary := filepath.Join(base, "agent-env"+suffix)
	run := func(dir, name string, args ...string) string {
		t.Helper()
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(cctx, name, args...)
		cmd.Dir = dir
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, e, out)
		}
		return string(out)
	}
	run(root, "go", "build", "-o", binary, "./cmd/agent-env")
	ca, credentials := multiHostCertificates(t, base)
	controllerHome := filepath.Join(base, "controller")
	clientHome := filepath.Join(base, "client")
	workerHomes := []string{filepath.Join(base, "worker-a"), filepath.Join(base, "worker-b")}
	for _, home := range append([]string{controllerHome, clientHome}, workerHomes...) {
		if e = os.MkdirAll(home, 0700); e != nil {
			t.Fatal(e)
		}
	}
	env := func(home string) []string {
		result := []string{}
		for _, v := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(v), "AGENT_ENV_HOME=") {
				result = append(result, v)
			}
		}
		return append(result, "AGENT_ENV_HOME="+home)
	}
	invoke := func(home string, args ...string) (json.RawMessage, error) {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(cctx, binary, append([]string{"--output=json", "--owner=multihost-fixture"}, args...)...)
		cmd.Env = env(home)
		var out, errOut bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errOut
		e := cmd.Run()
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		decodeErr := json.Unmarshal(out.Bytes(), &envelope)
		if e != nil {
			return envelope.Data, fmt.Errorf("CLI %v: %w\n%s\n%s", args, e, errOut.String(), out.String())
		}
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", out.String(), decodeErr)
		}
		return envelope.Data, nil
	}
	must := func(home string, args ...string) json.RawMessage {
		t.Helper()
		raw, e := invoke(home, args...)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	for _, identity := range []string{"client", "worker-a", "worker-b"} {
		args := []string{"control-plane", "enroll", "--certificate", credentials[identity][0], "--role", protocol.RoleWorker, "--host-id", identity}
		if identity == "client" {
			args = []string{"control-plane", "enroll", "--certificate", credentials[identity][0], "--role", protocol.RoleClient}
		}
		must(controllerHome, args...)
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	url := "https://" + address
	tlsArgs := func(identity string) []string {
		return []string{"--tls-ca", ca, "--tls-cert", credentials[identity][0], "--tls-key", credentials[identity][1]}
	}
	remoteArgs := append(tlsArgs("client"), "--controller", url, "--wait", "90s")
	remote := func(args ...string) json.RawMessage {
		t.Helper()
		return must(clientHome, append(append([]string{}, remoteArgs...), args...)...)
	}
	processes := []*multiHostProcess{}
	t.Cleanup(func() {
		for i := len(processes) - 1; i >= 0; i-- {
			p := processes[i]
			p.stop()
			if t.Failed() {
				content, _ := os.ReadFile(p.log.Name())
				t.Logf("service %s output:\n%s", filepath.Base(p.log.Name()), content)
			}
		}
	})
	start := func(home, label string, args ...string) *multiHostProcess {
		t.Helper()
		log, e := os.CreateTemp(base, label+"-*.log")
		if e != nil {
			t.Fatal(e)
		}
		cmd := exec.Command(binary, args...)
		cmd.Env = env(home)
		cmd.Stdout = log
		cmd.Stderr = log
		if e = cmd.Start(); e != nil {
			t.Fatal(e)
		}
		p := &multiHostProcess{cmd: cmd, log: log}
		processes = append(processes, p)
		return p
	}
	startController := func() *multiHostProcess {
		return start(controllerHome, "controller", append(tlsArgs("server"), "control-plane", "serve", "--listen", address)...)
	}
	startController()
	startWorker := func(index int) *multiHostProcess {
		identity := []string{"worker-a", "worker-b"}[index]
		return start(workerHomes[index], identity, append(tlsArgs(identity), "--controller", url, "worker", "serve", "--host-id", identity, "--max-leases", "2", "--android-slots", "0")...)
	}
	startWorker(0)
	startWorker(1)
	waitHosts := func() []protocol.Host {
		t.Helper()
		deadline := time.Now().Add(90 * time.Second)
		var last error
		for time.Now().Before(deadline) {
			raw, e := invoke(clientHome, append(append([]string{}, remoteArgs...), "hosts", "list")...)
			last = e
			if e == nil {
				var hosts []protocol.Host
				if e = json.Unmarshal(raw, &hosts); e == nil && len(hosts) == 2 && hosts[0].Online && hosts[1].Online && hosts[0].Compatible && hosts[1].Compatible {
					return hosts
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("two TLS workers did not register: %v", last)
		return nil
	}

	waitHosts()
	return remoteRuntimeFixture{base, remote, run}
}
func (f remoteRuntimeFixture) commit(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := filepath.Join(f.base, "target")
	for name, content := range files {
		p := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	f.local(repo, "git", "init")
	f.local(repo, "git", "config", "user.email", "fixture@example.test")
	f.local(repo, "git", "config", "user.name", "Fixture")
	f.local(repo, "git", "add", ".")
	f.local(repo, "git", "commit", "-m", "Committed runtime fixture")
	return repo
}
func (f remoteRuntimeFixture) create(t *testing.T, repo string) string {
	t.Helper()
	var op protocol.Operation
	if err := json.Unmarshal(f.remote("create", repo, "--stack", "app", "--host", "worker-a"), &op); err != nil {
		t.Fatal(err)
	}
	if op.LeaseID != "" {
		t.Cleanup(func() {
			released := f.operation(t, "destroy", op.LeaseID)
			if released.Lease == nil || released.Lease.Observed != "released" {
				t.Fatalf("cleanup missing released evidence: %+v", released)
			}
		})
	}
	if op.Result == nil || op.Result.State != "completed" {
		t.Fatalf("create result: %+v", op)
	}
	var response worker.Response
	if err := json.Unmarshal(op.Result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if response.Lease == nil || response.Lease.Observed != "ready" {
		t.Fatalf("create: %+v", response)
	}
	lease := *response.Lease
	return lease.ID
}
func (f remoteRuntimeFixture) download(t *testing.T, id, run string) {
	t.Helper()
	response := f.operation(t, "artifacts", id)
	raw, _ := json.Marshal(response.Value)
	var refs []worker.ArtifactReference
	if err := json.Unmarshal(raw, &refs); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, ref := range refs {
		if ref.Artifact.RunID != run {
			continue
		}
		destination := filepath.Join(f.base, fmt.Sprintf("download-%d", count))
		f.remote("artifact-download", ref.Blob.Digest, "--destination", destination)
		content, err := os.ReadFile(destination)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(content)) != ref.Blob.Digest {
			t.Fatalf("download digest: %v", err)
		}
		count++
	}
	if count == 0 {
		t.Fatal("no registered run artifact downloaded")
	}

}
func TestMultiHostRemoteBrowser(t *testing.T) {
	browser, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Fatal("selected remote browser acceptance requires google-chrome:", err)
	}
	extra := ""
	if os.Getenv("AGENT_ENV_BROWSER_NO_SANDBOX") == "1" {
		extra = `, "--no-sandbox"`
		t.Log("explicit fixture-only --no-sandbox requested")
	}
	f := newRemoteRuntimeFixture(t)
	manifest := fmt.Sprintf(`version: 1
sources:
  primary: {repository: ., default_ref: HEAD}
components:
  app:
    runtime: browser
    endpoints: {cdp: {runtime_port: cdp}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:cdp}/json/version", timeout: 30s}
stacks:
  app: {roots: [app]}
runtimes:
  browser:
    type: process
    source: primary
    working_directory: .
    command: [%q, "--headless=new", "--enable-automation"%s, "--disable-dev-shm-usage", "--no-first-run", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=${port:cdp}", "--user-data-dir=${runtime_dir}/profile", "about:blank"]
    ports: {cdp: {protocol: tcp}}
browsers:
  page: {type: chromium-cdp, runtime: browser, cdp_port: cdp}
`, filepath.Base(browser), extra)
	id := f.create(t, f.commit(t, map[string]string{".agent-env.yaml": manifest}))
	var snapshot app.BrowserResult
	response := f.operation(t, "browser", "snapshot", id, "--browser", "page")
	value, _ := json.Marshal(response.Value)
	if err := json.Unmarshal(value, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Run.ID == "" || snapshot.Snapshot == nil {
		t.Fatalf("snapshot: %+v", snapshot)
	}
	f.remote("browser", "pages", id, "--browser", "page")
	f.download(t, id, snapshot.Run.ID)
}
func TestMultiHostRemoteCompose(t *testing.T) {
	for _, provider := range []string{"docker", "podman"} {
		t.Run(provider, func(t *testing.T) {
			if _, err := exec.LookPath(provider); err != nil {
				t.Fatalf("selected remote acceptance requires %s: %v", provider, err)
			}
			f := newRemoteRuntimeFixture(t)
			f.local(f.base, provider, "info")
			manifest := fmt.Sprintf(`version: 1
sources:
  primary: {repository: ., default_ref: HEAD}
components:
  app:
    runtime: compose
    compose_services: [web]
    endpoints: {http: {service: web, target: 80, protocol: tcp}}
stacks:
  app: {roots: [app]}
runtimes:
  compose:
    type: compose
    provider: %s
    source: primary
    project_directory: .
    files: [compose.yaml]
tests:
  web:
    stack: app
    source: primary
    working_directory: .
    command: [go, version]
`, provider+"-compose")
			compose := `services:
  web:
    image: docker.io/library/nginx:alpine
    ports: ["80"]
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://127.0.0.1:80"]
      interval: 1s
      timeout: 1s
      retries: 30
`
			id := f.create(t, f.commit(t, map[string]string{".agent-env.yaml": manifest, "compose.yaml": compose}))
			response := f.operation(t, "test", id, "web")
			if response.Run == nil || response.Run.Status != "passed" {
				t.Fatalf("compose test: %+v", response)
			}
			run := *response.Run
			f.remote("logs", id)
			f.remote("logs", id, "--run", run.ID)
			f.download(t, id, run.ID)
		})
	}
}

func (f remoteRuntimeFixture) operation(t *testing.T, args ...string) worker.Response {
	t.Helper()
	var op protocol.Operation
	if err := json.Unmarshal(f.remote(args...), &op); err != nil {
		t.Fatal(err)
	}
	if op.Result == nil || op.Result.State != "completed" {
		t.Fatalf("operation %v: %+v", args, op)
	}
	var response worker.Response
	if err := json.Unmarshal(op.Result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	return response
}
