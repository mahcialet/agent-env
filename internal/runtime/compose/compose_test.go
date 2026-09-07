package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
)

type step struct {
	args   []string
	output string
	err    error
}
type fakeRunner struct {
	t        *testing.T
	steps    []step
	commands []execx.Command
}

func (f *fakeRunner) Run(_ context.Context, c execx.Command) (execx.Result, error) {
	f.t.Helper()
	f.commands = append(f.commands, c)
	if len(f.steps) == 0 {
		f.t.Fatalf("unexpected command: %#v", c)
		return execx.Result{}, errors.New("unexpected")
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	if c.Name != "docker" || !reflect.DeepEqual(c.Args, s.args) {
		f.t.Fatalf("wanted docker %#v; got %s %#v", s.args, c.Name, c.Args)
	}
	if c.Timeout <= 0 {
		f.t.Fatal("unbounded command")
	}
	return execx.Result{Stdout: s.output}, s.err
}
func (f *fakeRunner) done() {
	f.t.Helper()
	if len(f.steps) > 0 {
		f.t.Fatalf("%d expected commands not run", len(f.steps))
	}
}
func runtimeFixture(t *testing.T) domain.Runtime {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "source with spaces 日本語")
	return domain.Runtime{Name: "backend", Type: "compose", Project: "ae_test_123", Context: "test-context", Directory: dir, Files: []string{filepath.Join(dir, "compose.yaml"), filepath.Join(dir, "compose.agent.yaml")}, Services: []string{"api"}}
}
func base(r domain.Runtime) []string {
	args := []string{"--context", r.Context, "compose", "-p", r.Project, "--project-directory", r.Directory}
	for _, p := range r.Files {
		args = append(args, "-f", p)
	}
	return args
}
func with(args []string, rest ...string) []string {
	return append(append([]string{}, args...), rest...)
}
func testClient(f *fakeRunner) Client { return Client{Runner: f, Policy: policy.Defaults()} }

func TestDoctor(t *testing.T) {
	f := &fakeRunner{t: t, steps: []step{{args: []string{"context", "show"}, output: "desktop-linux\r\n"}, {args: []string{"--context", "desktop-linux", "compose", "version", "--short"}, output: "5.5.0\n"}, {args: []string{"--context", "desktop-linux", "version", "--format", "{{json .}}"}, output: `{"Client":{"Version":"29.7.2"},"Server":{"Version":"29.7.2"}}`}}}
	got, err := testClient(f).Doctor(context.Background())
	if err != nil || got["context"] != "desktop-linux" || got["compose_version"] != "5.5.0" {
		t.Fatalf("got %v %v", got, err)
	}
	f.done()
	for _, kind := range []string{"missing-docker", "old-compose", "missing-daemon"} {
		t.Run(kind, func(t *testing.T) {
			f := &fakeRunner{t: t}
			switch kind {
			case "missing-docker":
				f.steps = []step{{args: []string{"context", "show"}, err: errors.New("executable not found")}}
			case "old-compose":
				f.steps = []step{{args: []string{"context", "show"}, output: "default"}, {args: []string{"--context", "default", "compose", "version", "--short"}, output: "1.29.0"}}
			case "missing-daemon":
				f.steps = []step{{args: []string{"context", "show"}, output: "default"}, {args: []string{"--context", "default", "compose", "version", "--short"}, output: "2.40.0"}, {args: []string{"--context", "default", "version", "--format", "{{json .}}"}, output: `{"Client":{"Version":"29"}}`}}
			}
			if _, err := testClient(f).Doctor(context.Background()); err == nil {
				t.Fatal("missing prerequisite succeeded")
			}
			f.done()
		})
	}
}

func TestRenderSelectedClosureAndPolicy(t *testing.T) {
	r := runtimeFixture(t)
	good := `{"name":"ae_test_123","services":{"api":{"depends_on":{"db":{"condition":"service_healthy"}},"networks":{"default":{}},"ports":[{"target":8080}],"volumes":[{"type":"volume","source":"data","target":"/data"}]},"db":{},"dashboard":{"privileged":true}},"networks":{"default":{"name":"ae_test_123_default"}},"volumes":{"data":{"name":"ae_test_123_data"}}}`
	f := &fakeRunner{t: t, steps: []step{{args: with(base(r), "config", "--format", "json"), output: good}}}
	got, err := testClient(f).Render(context.Background(), r, []string{r.Directory})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Services, []string{"db", "api"}) || len(got.Digest) != 64 || !json.Valid(got.JSON) {
		t.Fatalf("invalid render: %+v", got)
	}
	if f.commands[0].Dir != r.Directory {
		t.Fatal("missing explicit cwd")
	}
	f.done()
	for _, tc := range []struct{ name, body, want string }{
		{"fixed port", strings.Replace(good, `"target":8080`, `"target":8080,"published":"8080"`, 1), "POLICY_FIXED_PORT"},
		{"global volume", strings.Replace(good, "ae_test_123_data", "global_data", 1), "globally shared"},
		{"global network", strings.Replace(good, "ae_test_123_default", "global_net", 1), "globally shared"},
		{"device", strings.Replace(good, `"api":{`, `"api":{"devices":[{"source":"/dev/kvm","target":"/dev/kvm"}],`, 1), "device passthrough"},
		{"missing service", `{"services":{"unrelated":{}}}`, "missing"},
		{"invalid JSON", `not JSON`, "invalid rendered"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRunner{t: t, steps: []step{{args: with(base(r), "config", "--format", "json"), output: tc.body}}}
			_, err := testClient(f).Render(context.Background(), r, []string{r.Directory})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %s got %v", tc.want, err)
			}
			f.done()
		})
	}
}

func TestUpUsesPinnedConfigurationAndSelectedServices(t *testing.T) {
	r := runtimeFixture(t)
	r.ConfigPath = filepath.Join(t.TempDir(), "validated config.json")
	if err := os.WriteFile(r.ConfigPath, []byte(`{"services":{"db":{},"api":{}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	r.Services = []string{"db", "api"}
	f := &fakeRunner{t: t, steps: []step{{args: []string{"--context", r.Context, "compose", "-p", r.Project, "--project-directory", r.Directory, "-f", r.ConfigPath, "up", "-d", "db", "api"}}}}
	if err := testClient(f).Up(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	f.done()
	r.Context = ""
	if err := testClient(f).Up(context.Background(), r); err == nil {
		t.Fatal("missing context accepted")
	}
	r.Context = "default"
	r.ConfigPath = "relative.json"
	if err := testClient(f).Up(context.Background(), r); err == nil {
		t.Fatal("relative config accepted")
	}
}

func inspectSteps(r domain.Runtime, containers, containerJSON, networks, networkJSON, volumes, volumeJSON string) []step {
	b := []string{"--context", r.Context}
	filter := "label=" + projectLabel + "=" + r.Project
	steps := []step{{args: with(b, "ps", "--all", "--quiet", "--filter", filter), output: containers}}
	if containers != "" {
		steps = append(steps, step{args: with(with(b, "inspect", "--type", "container"), ids(containers)...), output: containerJSON})
	}
	steps = append(steps, step{args: with(b, "network", "ls", "--quiet", "--filter", filter), output: networks})
	if networks != "" {
		steps = append(steps, step{args: with(with(b, "network", "inspect"), ids(networks)...), output: networkJSON})
	}
	steps = append(steps, step{args: with(b, "volume", "ls", "--quiet", "--filter", filter), output: volumes})
	if volumes != "" {
		steps = append(steps, step{args: with(with(b, "volume", "inspect"), ids(volumes)...), output: volumeJSON})
	}
	return steps
}
func containerJSON(r domain.Runtime, status string) string {
	labels := map[string]string{projectLabel: r.Project, "com.docker.compose.service": "api", runtimeLabel: r.Name}
	if r.LeaseID != "" {
		labels[leaseLabel] = r.LeaseID
	}
	data := []map[string]any{{"Id": "c123", "Image": "sha256:image", "Config": map[string]any{"Image": "example:latest", "Labels": labels}, "State": map[string]any{"Status": status, "Running": status == "running", "Health": map[string]string{"Status": "healthy"}}, "NetworkSettings": map[string]any{"Ports": map[string]any{"8080/tcp": []map[string]string{{"HostIp": "127.0.0.1", "HostPort": "43210"}}}}}}
	b, _ := json.Marshal(data)
	return string(b)
}

func TestInspectObservesResourcesHealthAndEndpoints(t *testing.T) {
	r := runtimeFixture(t)
	networkJSON := `[{"Id":"net123","Name":"ae_test_123_default","Labels":{"com.docker.compose.project":"ae_test_123"}}]`
	volumeJSON := `[{"Name":"ae_test_123_data","Labels":{"com.docker.compose.project":"ae_test_123"}}]`
	f := &fakeRunner{t: t, steps: inspectSteps(r, "c123", containerJSON(r, "running"), "net123", networkJSON, "ae_test_123_data", volumeJSON)}
	o, err := testClient(f).Inspect(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !o.Exists || !o.Ready || len(o.Resources) != 3 || o.Endpoints["api/8080/tcp"] != "127.0.0.1:43210" {
		t.Fatalf("unexpected observation %+v", o)
	}
	f.done()
	for _, status := range []string{"exited", "missing"} {
		t.Run(status, func(t *testing.T) {
			steps := inspectSteps(r, "c123", containerJSON(r, status), "", "", "", "")
			if status == "missing" {
				steps = inspectSteps(r, "", "", "", "", "", "")
			}
			f := &fakeRunner{t: t, steps: steps}
			o, err := testClient(f).Inspect(context.Background(), r)
			if err != nil || o.Ready || len(o.Diagnostics) == 0 {
				t.Fatalf("stale ready state: %+v %v", o, err)
			}
			f.done()
		})
	}
}

func TestDownVerifiesIdentityAndRemainingResources(t *testing.T) {
	r := runtimeFixture(t)
	before := inspectSteps(r, "c123", containerJSON(r, "running"), "", "", "", "")
	after := inspectSteps(r, "", "", "", "", "", "")
	steps := append(before, step{args: with(base(r), "down", "--volumes", "--remove-orphans")})
	steps = append(steps, after...)
	f := &fakeRunner{t: t, steps: steps}
	if err := testClient(f).Down(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	f.done()
	bad := strings.Replace(containerJSON(r, "running"), r.Project, "another_project", 1)
	f = &fakeRunner{t: t, steps: inspectSteps(r, "c123", bad, "", "", "", "")[:2]}
	if err := testClient(f).Down(context.Background(), r); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("unsafe cleanup: %v", err)
	}
	f.done()
	steps = append(inspectSteps(r, "", "", "", "", "", ""), step{args: with(base(r), "down", "--volumes", "--remove-orphans")})
	steps = append(steps, inspectSteps(r, "c123", containerJSON(r, "exited"), "", "", "", "")...)
	f = &fakeRunner{t: t, steps: steps}
	if err := testClient(f).Down(context.Background(), r); err == nil || !strings.Contains(err.Error(), "still has resources") {
		t.Fatalf("cleanup residue succeeded: %v", err)
	}
	f.done()
}

func TestGenerateOverride(t *testing.T) {
	r := runtimeFixture(t)
	p, err := GenerateOverride(r, "lease123", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p) || !json.Valid(b) || !strings.Contains(string(b), leaseLabel) || !strings.Contains(string(b), "lease123") {
		t.Fatalf("invalid ownership override %s", b)
	}
}

func TestRefuseTamperedRecordedConfig(t *testing.T) {
	r := runtimeFixture(t)
	r.ConfigPath = filepath.Join(t.TempDir(), "normalized.json")
	data := []byte("{\"services\":{}}\n")
	sum := sha256.Sum256(data)
	r.ConfigDigest = hex.EncodeToString(sum[:])
	if err := os.WriteFile(r.ConfigPath, []byte("changed instructions"), 0600); err != nil {
		t.Fatal(err)
	}
	f := &fakeRunner{t: t}
	if err := testClient(f).Up(context.Background(), r); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tampered instructions accepted: %v", err)
	}
	f.done()
}
