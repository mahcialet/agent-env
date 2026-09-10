//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"gopkg.in/yaml.v3"
)

func TestIntegrationCommandHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_FIXTURE_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	secret := os.Getenv("TEST_TOKEN")
	fmt.Fprintln(os.Stdout, "fixture stdout "+secret)
	fmt.Fprintln(os.Stderr, "fixture stderr "+secret)
	if err := os.WriteFile("report.txt", []byte("fixture artifact "+secret), 0600); err != nil {
		os.Exit(11)
	}
	if mode == "fail" {
		os.Exit(9)
	}
	os.Exit(0)
}

func integrationCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, b)
	}
	return strings.TrimSpace(string(b))
}

func invokeIntegration(args ...string) (json.RawMessage, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := New(&stdout, &stderr)
	cmd.SetArgs(append([]string{"--output=json", "--owner=integration"}, args...))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	err := cmd.ExecuteContext(ctx)
	var envelope struct {
		SchemaVersion int             `json:"schema_version"`
		Data          json.RawMessage `json:"data"`
	}
	if stdout.Len() > 0 {
		if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
			return nil, stderr.String(), fmt.Errorf("invalid CLI JSON %q: %w", stdout.String(), decodeErr)
		}
		if envelope.SchemaVersion != 1 {
			return nil, stderr.String(), fmt.Errorf("schema version %d", envelope.SchemaVersion)
		}
	}
	return envelope.Data, stderr.String(), err
}
func integrationJSON[T any](t *testing.T, args ...string) T {
	t.Helper()
	b, stderr, err := invokeIntegration(args...)
	if err != nil {
		t.Fatalf("CLI %v: %v\n%s\n%s", args, err, stderr, b)
	}
	var value T
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func integrationFixture(t *testing.T) (string, string, *config.Manifest) {
	t.Helper()
	if os.Getenv("AGENT_ENV_INTEGRATION") != "1" {
		t.Skip("explicit Docker integration: set AGENT_ENV_INTEGRATION=1")
	}
	integrationCommand(t, "", "docker", "info", "--format", "{{.ServerVersion}}")
	integrationCommand(t, "", "docker", "compose", "version", "--short")
	base, err := os.MkdirTemp("", "agent-env-docker-integration-")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "state space 日本語")
	repo := filepath.Join(base, "repository space 日本語")
	t.Setenv("AGENT_ENV_HOME", home)
	t.Setenv("FIXTURE_SECRET_TOKEN", "integration-credential-value-7f032")
	foreignVolume := fmt.Sprintf("agent-env-integration-foreign-%d-%d", os.Getpid(), time.Now().UnixNano())
	t.Setenv("AGENT_ENV_TEST_FOREIGN_VOLUME", foreignVolume)
	// Register recovery before any daemon resource exists, independently of the
	// create response. Keep registry and sources when cleanup cannot be proved.
	t.Cleanup(func() {
		if os.Getenv("AGENT_ENV_HOME") != home {
			t.Errorf("fixture home changed; preserving %s", base)
			return
		}
		err := cleanupIntegrationRegistry(home, func(id string) (domain.Lease, error) {
			raw, stderr, err := invokeIntegration("destroy", id, "--force")
			var released domain.Lease
			if err == nil {
				err = json.Unmarshal(raw, &released)
			}
			if err != nil {
				return released, fmt.Errorf("destroy fixture lease %s: %w: %s", id, err, stderr)
			}
			return released, nil
		})
		if err != nil {
			t.Errorf("fixture recovery failed; preserving %s: %v", base, err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		// Delete only this explicitly named fixture volume, never daemon-wide data.
		out, err := exec.CommandContext(ctx, "docker", "volume", "rm", foreignVolume).CombinedOutput()
		if err != nil {
			t.Errorf("fixture volume cleanup failed; preserving %s: %v %s", base, err, out)
			return
		}
		if t.Failed() {
			t.Logf("preserving failed fixture evidence: %s", base)
			return
		}
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("remove released fixture: %v", err)
		}
	})
	integrationCommand(t, "", "docker", "volume", "create", "--label", "io.agent-env.integration-fixture="+foreignVolume, foreignVolume)
	for _, p := range []string{repo, filepath.Join(repo, "www")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"compose.yaml", "www/index.html"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "compose", filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(p)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("tracked fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := &config.Manifest{Version: 1, Sources: map[string]config.Source{"self": {Repository: ".", DefaultRef: "main"}}, Runtimes: map[string]config.Runtime{"compose": {Type: "compose", Source: "self", ProjectDirectory: ".", Files: []string{"compose.yaml"}}}, Components: map[string]config.Component{"database": {Runtime: "compose", ComposeServices: []string{"database"}}, "api": {Runtime: "compose", ComposeServices: []string{"api"}, DependsOn: []string{"database"}, Provides: []string{"api"}}, "dashboard": {Runtime: "compose", ComposeServices: []string{"dashboard"}, DependsOn: []string{"api"}, Provides: []string{"dashboard"}}}, Stacks: map[string]config.Stack{"api": {Roots: []string{"api"}}, "dashboard": {Roots: []string{"dashboard"}}}, Tests: map[string]config.Test{}}
	for name, target := range map[string]int{"api": 8080, "dashboard": 8081} {
		component := m.Components[name]
		component.Endpoints = map[string]config.Endpoint{"http": {Service: name, Target: target, Protocol: "tcp"}}
		m.Components[name] = component
	}
	for _, mode := range []string{"pass", "fail"} {
		m.Tests[mode] = config.Test{Stack: "api", Source: "self", WorkingDirectory: ".", Command: []string{os.Args[0], "-test.run=^TestIntegrationCommandHelper$", "--", mode}, Env: map[string]string{"AGENT_ENV_FIXTURE_HELPER": "1", "TEST_TOKEN": "${env:FIXTURE_SECRET_TOKEN}"}, Timeout: "20s", Artifacts: []string{"report.txt"}}
	}
	writeIntegrationManifest(t, repo, m)
	integrationCommand(t, repo, "git", "-c", "init.templateDir=", "init", "--initial-branch=main")
	integrationCommit(t, repo)
	return repo, home, m
}
func integrationCommit(t *testing.T, repo string) {
	t.Helper()
	integrationCommand(t, repo, "git", "add", ".")
	integrationCommand(t, repo, "git", "-c", "user.name=Integration Fixture", "-c", "user.email=integration@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Integration fixture")
}
func writeIntegrationManifest(t *testing.T, repo string, m *config.Manifest) {
	t.Helper()
	b, err := yaml.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), b, 0600); err != nil {
		t.Fatal(err)
	}
}

func cleanupIntegrationLease(t *testing.T, l domain.Lease) {
	t.Helper()
	if l.ID == "" {
		return
	}
	t.Cleanup(func() {
		b, stderr, err := invokeIntegration("destroy", l.ID, "--force")
		if err != nil {
			t.Errorf("cleanup lease %s: %v\n%s\n%s", l.ID, err, stderr, b)
		}
	})
}
func createIntegration(t *testing.T, repo, stack string, extra ...string) domain.Lease {
	t.Helper()
	args := append([]string{"create", repo, "--stack", stack}, extra...)
	b, stderr, err := invokeIntegration(args...)
	var l domain.Lease
	if len(b) > 0 {
		if e := json.Unmarshal(b, &l); e != nil {
			t.Fatal(e)
		}
	}
	cleanupIntegrationLease(t, l)
	if err != nil {
		t.Fatalf("create %s: %v\n%s\n%s", stack, err, stderr, b)
	}
	if l.Observed != "ready" {
		t.Fatalf("create state %s", l.Observed)
	}
	return l
}

type integrationShow struct {
	Lease     domain.Lease        `json:"lease"`
	Runs      []domain.CommandRun `json:"runs"`
	Artifacts []domain.Artifact   `json:"artifacts"`
	Events    []domain.Event      `json:"events"`
}

func resourceIDs(l domain.Lease) []string {
	ids := make([]string, 0, len(l.Resources))
	for _, r := range l.Resources {
		ids = append(ids, r.Kind+":"+r.ExternalID)
	}
	sort.Strings(ids)
	return ids
}

func checkComponentLogs(t *testing.T, id, component, excluded string, nonempty bool) {
	t.Helper()
	result := integrationJSON[struct {
		Logs map[string]string `json:"logs"`
	}](t, "logs", id, "--component", component)
	var data strings.Builder
	for _, log := range result.Logs {
		data.WriteString(log)
	}
	if strings.Contains(data.String(), excluded+"-1") {
		t.Fatalf("component %s logs contain %s service", component, excluded)
	}
	if nonempty && !strings.Contains(data.String(), component+"-1") {
		t.Fatalf("component %s logs have no own service evidence: %q", component, data.String())
	}
}

func TestIntegrationConcurrentLeasesClosureAndEvidence(t *testing.T) {
	repo, home, _ := integrationFixture(t)
	plan := integrationJSON[struct {
		Sources    []domain.Source    `json:"sources"`
		Components []domain.Component `json:"components"`
	}](t, "plan", repo, "--stack", "api", "--ref", "main")
	if len(plan.Sources) != 1 || len(plan.Components) != 2 {
		t.Fatalf("plan closure %+v", plan)
	}
	type result struct {
		lease  domain.Lease
		err    error
		stderr string
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, stack := range []string{"api", "dashboard"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, stderr, err := invokeIntegration("create", repo, "--stack", stack)
			var l domain.Lease
			if len(b) > 0 {
				if e := json.Unmarshal(b, &l); e != nil {
					err = e
				}
			}
			results <- result{l, err, stderr}
		}()
	}
	wg.Wait()
	close(results)
	leases := map[string]domain.Lease{}
	for result := range results {
		cleanupIntegrationLease(t, result.lease)
		if result.err != nil {
			t.Errorf("concurrent create: %v\n%s", result.err, result.stderr)
		}
		leases[result.lease.Stack] = result.lease
	}
	if t.Failed() {
		t.FailNow()
	}
	api, dashboard := leases["api"], leases["dashboard"]
	t.Logf("created real concurrent leases %s and %s", api.ID, dashboard.ID)
	if api.Observed != "ready" || dashboard.Observed != "ready" || api.Sources[0].Commit != dashboard.Sources[0].Commit || api.Sources[0].WorktreePath == dashboard.Sources[0].WorktreePath || api.Runtimes[0].Project == dashboard.Runtimes[0].Project {
		t.Fatalf("concurrency identity %+v %+v", api, dashboard)
	}
	if len(api.Runtimes[0].Services) != 2 || len(dashboard.Runtimes[0].Services) != 3 {
		t.Fatalf("closure %+v %+v", api.Runtimes, dashboard.Runtimes)
	}
	for _, l := range []domain.Lease{api, dashboard} {
		b, err := os.ReadFile(filepath.Join(home, "leases", l.ID, "environment.json"))
		if err != nil {
			t.Fatal(err)
		}
		var descriptor struct {
			SchemaVersion int          `json:"schema_version"`
			Lease         domain.Lease `json:"lease"`
		}
		if err := json.Unmarshal(b, &descriptor); err != nil {
			t.Fatal(err)
		}
		if descriptor.SchemaVersion != 1 || descriptor.Lease.ID != l.ID || descriptor.Lease.Observed != "ready" || len(descriptor.Lease.Resources) == 0 {
			t.Fatalf("descriptor not current: %+v", descriptor)
		}
		if diff := integrationCommand(t, l.Sources[0].WorktreePath, "git", "diff", "HEAD", "--", "compose.yaml"); diff != "" {
			t.Fatalf("generated endpoint changed source: %s", diff)
		}
		original, err := os.ReadFile(filepath.Join(l.Sources[0].WorktreePath, "compose.yaml"))
		if err != nil || strings.Contains(string(original), "ports:") {
			t.Fatalf("fixture already publishes host ports: %v", err)
		}
	}
	capabilities := integrationJSON[struct {
		Endpoints map[string]string `json:"endpoints"`
	}](t, "capabilities", dashboard.ID)
	for _, name := range []string{"api.http", "dashboard.http"} {
		address := capabilities.Endpoints[name]
		if !strings.HasPrefix(address, "127.0.0.1:") || strings.HasSuffix(address, ":0") {
			t.Fatalf("dynamic endpoint %s = %q", name, address)
		}
		response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://" + address + "/")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("endpoint %s status %d", name, response.StatusCode)
		}
	}
	checkComponentLogs(t, dashboard.ID, "api", "dashboard", true)
	checkComponentLogs(t, dashboard.ID, "dashboard", "api", true)
	for _, r := range api.Resources {
		if r.Metadata["service"] == "dashboard" {
			t.Fatal("unselected dashboard started")
		}
	}
	before := resourceIDs(dashboard)
	hasVolume, hasNetwork := false, false
	for _, r := range dashboard.Resources {
		hasVolume = hasVolume || r.Kind == "volume"
		hasNetwork = hasNetwork || r.Kind == "network"
	}
	if !hasVolume || !hasNetwork {
		t.Fatal("fixture lacks volume/network evidence")
	}
	for _, r := range dashboard.Resources {
		if r.Metadata["service"] == "api" {
			host := integrationCommand(t, "", "docker", "--context", dashboard.Runtimes[0].Context, "port", r.ExternalID, "8080/tcp")
			response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://" + host + "/")
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatalf("HTTP %d", response.StatusCode)
			}
		}
	}
	run := integrationJSON[domain.CommandRun](t, "test", api.ID, "pass")
	if run.Status != "passed" || run.ExitCode != 0 {
		t.Fatalf("run %+v", run)
	}
	for _, path := range []string{run.StdoutPath, run.StderrPath} {
		b, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(b), "[REDACTED]") || strings.Contains(string(b), os.Getenv("FIXTURE_SECRET_TOKEN")) {
			t.Fatalf("redacted evidence %s: %q %v", path, b, err)
		}
	}
	b, _, err := invokeIntegration("test", api.ID, "fail")
	if err == nil || ExitCode(err) != 5 {
		t.Fatalf("failing named test exit: %v", err)
	}
	var failed domain.CommandRun
	if err := json.Unmarshal(b, &failed); err != nil {
		t.Fatal(err)
	}
	if failed.ExitCode != 9 || failed.Status != "failed" {
		t.Fatalf("failure run %+v", failed)
	}
	show := integrationJSON[integrationShow](t, "show", api.ID)
	if len(show.Runs) != 2 || len(show.Artifacts) < 8 || len(show.Events) == 0 {
		t.Fatalf("show evidence runs=%d artifacts=%d events=%d", len(show.Runs), len(show.Artifacts), len(show.Events))
	}
	integrationJSON[domain.Lease](t, "destroy", api.ID)
	checkComponentLogs(t, api.ID, "api", "database", true)
	descriptorData, err := os.ReadFile(filepath.Join(home, "leases", api.ID, "environment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var releasedDescriptor struct {
		Lease domain.Lease `json:"lease"`
	}
	if err := json.Unmarshal(descriptorData, &releasedDescriptor); err != nil {
		t.Fatal(err)
	}
	if releasedDescriptor.Lease.Observed != "released" {
		t.Fatal("descriptor did not record release")
	}
	if name := integrationCommand(t, "", "docker", "volume", "inspect", os.Getenv("AGENT_ENV_TEST_FOREIGN_VOLUME"), "--format", "{{.Name}}"); name != os.Getenv("AGENT_ENV_TEST_FOREIGN_VOLUME") {
		t.Fatal("unselected foreign volume was changed by destroy")
	}
	after := integrationJSON[integrationShow](t, "show", dashboard.ID).Lease
	if strings.Join(before, "\n") != strings.Join(resourceIDs(after), "\n") || after.Observed != "ready" {
		t.Fatalf("destroy changed sibling resources before=%v after=%v", before, resourceIDs(after))
	}
	if _, err := os.Stat(dashboard.Sources[0].WorktreePath); err != nil {
		t.Fatal("sibling worktree removed")
	}
	r := dashboard.Runtimes[0]
	integrationCommand(t, r.Directory, "docker", "--context", r.Context, "compose", "-p", r.Project, "--project-directory", r.Directory, "-f", r.ConfigPath, "down", "--volumes", "--remove-orphans")
	listed := integrationJSON[[]domain.Lease](t, "list")
	found := false
	for _, l := range listed {
		if l.ID == dashboard.ID {
			found = true
			if l.Observed != "degraded" {
				t.Fatalf("manual deletion invisible: %s", l.Observed)
			}
		}
	}
	if !found {
		t.Fatal("lease missing from list")
	}
}

func TestIntegrationMultiRepositoryPinsAndDirtyGC(t *testing.T) {
	repo, home, m := integrationFixture(t)
	second := filepath.Join(filepath.Dir(repo), "second local 日本語")
	if err := os.MkdirAll(second, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "README.md"), []byte("second repository\n"), 0600); err != nil {
		t.Fatal(err)
	}
	integrationCommand(t, second, "git", "-c", "init.templateDir=", "init", "--initial-branch=main")
	integrationCommit(t, second)
	firstCommit := integrationCommand(t, second, "git", "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(second, "README.md"), []byte("new second repository\n"), 0600); err != nil {
		t.Fatal(err)
	}
	integrationCommit(t, second)
	m.Sources["tools"] = config.Source{Repository: second, DefaultRef: "main"}
	writeIntegrationManifest(t, repo, m)
	integrationCommit(t, repo)
	l := createIntegration(t, repo, "api", "--source", "tools="+firstCommit)
	if len(l.Sources) != 2 {
		t.Fatalf("sources %+v", l.Sources)
	}
	for _, source := range l.Sources {
		if source.Alias == "tools" && source.Commit != firstCommit {
			t.Fatalf("override ignored %+v", source)
		}
		head := integrationCommand(t, source.WorktreePath, "git", "rev-parse", "HEAD")
		if head != source.Commit {
			t.Fatalf("unpin %s != %s", head, source.Commit)
		}
	}
	tracked := filepath.Join(l.Sources[0].WorktreePath, "README.md")
	if err := os.WriteFile(tracked, []byte("precious tracked changes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	l.ExpiresAt = time.Now().Add(-10 * time.Minute)
	l.HeartbeatAt = time.Now().Add(-2 * time.Minute)
	if err := store.Save(context.Background(), l); err != nil {
		store.Close()
		t.Fatal(err)
	}
	store.Close()
	preview := integrationJSON[struct {
		Apply  bool           `json:"apply"`
		Leases []domain.Lease `json:"leases"`
	}](t, "gc")
	if preview.Apply || len(preview.Leases) != 1 {
		t.Fatalf("default GC %+v", preview)
	}
	if b, err := os.ReadFile(tracked); err != nil || string(b) != "precious tracked changes\n" {
		t.Fatalf("dryrun changed tracked source: %q %v", b, err)
	}
	_, _, err = invokeIntegration("gc", "--apply")
	if err == nil || ExitCode(err) != 6 {
		t.Fatalf("unsafe GC succeeded: %v", err)
	}
	show := integrationJSON[integrationShow](t, "show", l.ID)
	if show.Lease.Observed != "quarantined" {
		t.Fatalf("dirty GC not quarantined: %s", show.Lease.Observed)
	}
	if b, err := os.ReadFile(tracked); err != nil || string(b) != "precious tracked changes\n" {
		t.Fatalf("GC discarded tracked changes: %q %v", b, err)
	}
	integrationJSON[domain.Lease](t, "destroy", l.ID, "--force")
	show = integrationJSON[integrationShow](t, "show", l.ID)
	found := false
	for _, a := range show.Artifacts {
		if a.Kind == "tracked-diff" {
			b, err := os.ReadFile(a.Path)
			if err != nil || !strings.Contains(string(b), "precious tracked changes") {
				t.Fatalf("diff evidence %q %v", b, err)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("force missing tracked diff")
	}
}

func TestIntegrationReadinessFailureRollsBack(t *testing.T) {
	repo, _, m := integrationFixture(t)
	c := m.Components["api"]
	c.Readiness = []config.Probe{{Type: "command", Source: "self", WorkingDirectory: ".", Command: []string{"git", "rev-parse", "--verify", "refs/heads/missing-fixture-ref"}, Timeout: "100ms", Interval: "20ms"}}
	m.Components["api"] = c
	writeIntegrationManifest(t, repo, m)
	integrationCommit(t, repo)
	b, stderr, err := invokeIntegration("create", repo, "--stack", "api")
	var l domain.Lease
	if e := json.Unmarshal(b, &l); e != nil {
		t.Fatalf("failed create JSON %q: %v %s", b, e, stderr)
	}
	cleanupIntegrationLease(t, l)
	if err == nil || l.Observed != "released" {
		t.Fatalf("failed readiness rollback state=%s error=%v %s", l.Observed, err, stderr)
	}
	for _, source := range l.Sources {
		if _, err := os.Stat(source.WorktreePath); !os.IsNotExist(err) {
			t.Fatalf("rollback worktree remains %s %v", source.WorktreePath, err)
		}
	}
	for _, r := range l.Runtimes {
		ids := integrationCommand(t, "", "docker", "--context", r.Context, "ps", "-aq", "--filter", "label=com.docker.compose.project="+r.Project)
		if ids != "" {
			t.Fatalf("rollback containers remain %s", ids)
		}
		for _, kind := range []string{"network", "volume"} {
			ids := integrationCommand(t, "", "docker", "--context", r.Context, kind, "ls", "-q", "--filter", "label=com.docker.compose.project="+r.Project)
			if ids != "" {
				t.Fatalf("rollback %s remain %s", kind, ids)
			}
		}
	}
	show := integrationJSON[integrationShow](t, "show", l.ID)
	if len(show.Artifacts) == 0 {
		t.Fatal("failed startup logs not retained")
	}
}

func TestIntegrationRegistryRecoversLostCreateResponse(t *testing.T) {
	repo, home, _ := integrationFixture(t)
	// Create real resources but intentionally discard its entire response. Only
	// durable fixture registry state may supply identities for recovery.
	if _, stderr, err := invokeIntegration("create", repo, "--stack", "api"); err != nil {
		t.Fatalf("effect setup failed: %v %s", err, stderr)
	}
	before := integrationJSON[[]domain.Lease](t, "list", "--cached")
	if len(before) != 1 || before[0].Observed != "ready" || len(before[0].Resources) == 0 || len(before[0].Runtimes) == 0 {
		t.Fatalf("lost-response fixture did not create effects: %+v", before)
	}
	// Prove daemon-side effects existed independently of cached registry rows,
	// so an empty runtime list or a never-started project cannot satisfy cleanup.
	for _, r := range before[0].Runtimes {
		if r.Project == "" {
			t.Fatal("fixture runtime has no Compose project identity")
		}
		ids := integrationCommand(t, "", "docker", "--context", r.Context, "ps", "-aq", "--filter", "label=com.docker.compose.project="+r.Project)
		if ids == "" {
			t.Fatalf("fixture project %s never created daemon containers", r.Project)
		}
	}
	if err := cleanupIntegrationRegistry(home, func(id string) (domain.Lease, error) {
		raw, stderr, err := invokeIntegration("destroy", id, "--force")
		var released domain.Lease
		if err != nil {
			return released, fmt.Errorf("recovery destroy: %w %s", err, stderr)
		}
		err = json.Unmarshal(raw, &released)
		return released, err
	}); err != nil {
		t.Fatal(err)
	}
	after := integrationJSON[[]domain.Lease](t, "list", "--cached")
	if len(after) != 1 || after[0].ID != before[0].ID || after[0].Observed != "released" {
		t.Fatalf("registry recovery not complete: %+v", after)
	}
	for _, r := range before[0].Runtimes {
		if ids := integrationCommand(t, "", "docker", "--context", r.Context, "ps", "-aq", "--filter", "label=com.docker.compose.project="+r.Project); ids != "" {
			t.Fatalf("owned containers remain: %s", ids)
		}
		for _, kind := range []string{"network", "volume"} {
			if ids := integrationCommand(t, "", "docker", "--context", r.Context, kind, "ls", "-q", "--filter", "label=com.docker.compose.project="+r.Project); ids != "" {
				t.Fatalf("owned %s remain: %s", kind, ids)
			}
		}
	}
	if name := integrationCommand(t, "", "docker", "volume", "inspect", os.Getenv("AGENT_ENV_TEST_FOREIGN_VOLUME"), "--format", "{{.Name}}"); name != os.Getenv("AGENT_ENV_TEST_FOREIGN_VOLUME") {
		t.Fatal("foreign volume changed")
	}
}
