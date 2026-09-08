package compose

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type podmanRunnerFunc func(context.Context, execx.Command) (execx.Result, error)

func (f podmanRunnerFunc) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	return f(ctx, c)
}
func testPodmanIdentity(t *testing.T) podmanIdentity {
	t.Helper()
	return podmanIdentity{Version: 1, Executable: filepath.Join(t.TempDir(), "podman"), ComposeExecutable: filepath.Join(t.TempDir(), "podman-compose"), Fingerprint: "fixture"}
}
func TestPodmanIdentityPinsLocalAndRemote(t *testing.T) {
	p := testPodmanIdentity(t)
	for _, remote := range []bool{false, true} {
		if remote {
			p.URL = "ssh://user@machine/run/user/1000/podman/podman.sock"
			p.Identity = filepath.Join(t.TempDir(), "key")
		}
		got, err := decodePodmanIdentity(p.encode())
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("round trip: %#v %v", got, err)
		}
		flags := got.flags()
		if flags[0] != "--remote="+map[bool]string{true: "true", false: "false"}[remote] {
			t.Fatal(flags)
		}
	}
	if _, err := decodePodmanIdentity("default"); err == nil {
		t.Fatal("accepted ambient identity")
	}
}
func TestPodmanEnvironmentRejectsInheritedRouting(t *testing.T) {
	for _, key := range []string{"CONTAINER_HOST", "CONTAINER_CONNECTION", "CONTAINER_SSHKEY", "PODMAN_COMPOSE_IN_POD", "DOCKER_HOST", "COMPOSE_FILE", podmanBridgeEnv} {
		t.Setenv(key, "unexpected")
	}
	for _, value := range podmanEnvironment() {
		if strings.Contains(value, "=unexpected") {
			t.Fatal(value)
		}
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("export PODMAN_COMPOSE_IN_POD=true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rejectPodmanDotenv(directory); err == nil {
		t.Fatal("accepted reserved dotenv")
	}
}
func TestPodmanNormalizeComposeModel(t *testing.T) {
	directory := t.TempDir()
	data, err := normalizePodmanConfig(`services:
  web:
    image: example
    ports: ["127.0.0.1::80"]
    depends_on: [db]
    environment: ["LITERAL=a=b"]
    volumes: ["./src:/app:ro", "/cache", "data:/data"]
  db:
    image: example
volumes:
  data:
`, directory, "ae_fixture")
	if err != nil {
		t.Fatal(err)
	}
	var model map[string]any
	if err = json.Unmarshal([]byte(data), &model); err != nil {
		t.Fatal(err)
	}
	services := model["services"].(map[string]any)
	web := services["web"].(map[string]any)
	mounts := web["volumes"].([]any)
	if mounts[0].(map[string]any)["source"] != filepath.Join(directory, "src") {
		t.Fatal(mounts)
	}
	anon := mounts[1].(map[string]any)["source"].(string)
	if model["volumes"].(map[string]any)[anon] == nil {
		t.Fatal("anonymous volume lacks owned declaration")
	}
	if web["ports"].([]any)[0].(map[string]any)["host_ip"] != "127.0.0.1" {
		t.Fatal(web)
	}
	if _, err := normalizePodmanConfig("x-podman: {in_pod: true}\nservices: {web: {image: example}}", directory, "ae_fixture"); err == nil {
		t.Fatal("accepted provider extension")
	}
}
func TestPodmanNativeLabelsRequired(t *testing.T) {
	for _, labels := range []string{`{"com.docker.compose.project":"ae_fixture"}`, `{"io.podman.compose.project":"ae_other","com.docker.compose.project":"ae_fixture"}`, `{}`} {
		if _, err := normalizePodmanInspection(`[{"Config":{"Labels":`+labels+`}}]`, "ae_fixture"); err == nil {
			t.Fatalf("accepted %s", labels)
		}
	}
	got, err := normalizePodmanInspection(`[{"Config":{"Labels":{"io.podman.compose.project":"ae_fixture","com.docker.compose.project":"ae_fixture","io.podman.compose.service":"web"}},"State":{"Healthcheck":{"Status":"unhealthy"}}}]`, "ae_fixture")
	if err != nil || !strings.Contains(got, `"Health":{"Status":"unhealthy"}`) {
		t.Fatalf("%s %v", got, err)
	}
}
func TestPodmanAdapterNeverCallsDockerAndPinsNativeFlags(t *testing.T) {
	p := testPodmanIdentity(t)
	called := false
	c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		called = true
		if q.Name != p.Executable || !reflect.DeepEqual(q.Args, []string{"--remote=false", "ps", "--filter", "label=io.podman.compose.project=ae_fixture"}) {
			t.Fatalf("un-pinned native request: %#v", q)
		}
		return execx.Result{}, nil
	})}
	_, err := (podmanAdapter{client: c, identity: p, project: "ae_fixture"}).Run(context.Background(), execx.Command{Name: "docker", Args: []string{"--context", p.encode(), "ps", "--filter", "label=com.docker.compose.project=ae_fixture"}})
	if err != nil || !called {
		t.Fatalf("%v %v", called, err)
	}
}
func TestPodmanChangedEngineRefusesMutation(t *testing.T) {
	p := testPodmanIdentity(t)
	calls := 0
	c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		calls++
		if !reflect.DeepEqual(q.Args, []string{"--remote=false", "info", "--format", "json"}) {
			t.Fatalf("mutation before fingerprint: %#v", q)
		}
		return execx.Result{Stdout: `{"host":{"hostname":"different","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
	})}
	if err := c.Up(context.Background(), domain.Runtime{Context: p.encode()}); err == nil || !strings.Contains(err.Error(), "fingerprint changed") {
		t.Fatalf("%v", err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestPodmanBridgeNativeRoundTrip(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "helper.go")
	binary := filepath.Join(directory, "podman-helper")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	sourceCode := `package main
import("encoding/json";"os")
func main(){ b,_:=json.Marshal(map[string]any{"args":os.Args[1:],"host":os.Getenv("CONTAINER_HOST"),"bridge":os.Getenv("AGENT_ENV_PODMAN_BRIDGE_V1")});_ = os.WriteFile(os.Getenv("AGENT_ENV_TEST_PODMAN_OUT"),b,0600);os.Exit(23) }
`
	if err := os.WriteFile(source, []byte(sourceCode), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", binary, source)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build native helper: %s %v", out, err)
	}
	output := filepath.Join(directory, "result.json")
	t.Setenv("AGENT_ENV_TEST_PODMAN_OUT", output)
	t.Setenv("CONTAINER_HOST", "ssh://wrong-host")
	p := podmanIdentity{Version: 1, Executable: binary, ComposeExecutable: binary, Fingerprint: "fixture"}
	for _, remote := range []bool{false, true} {
		if remote {
			p.URL = "ssh://user@expected-host/socket"
			p.Identity = filepath.Join(directory, "ssh key")
		}
		t.Setenv(podmanBridgeEnv, p.encode())
		handled, code := RunPodmanBridge([]string{"info", "--format", "json"})
		if !handled || code != 23 {
			t.Fatalf("bridge result %v %d", handled, code)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			Args         []string
			Host, Bridge string
		}
		if err = json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result.Args, append(p.flags(), "info", "--format", "json")) || result.Host != "" || result.Bridge != "" {
			t.Fatalf("unpinned child %#v", result)
		}
	}
	handled, code := RunPodmanBridge([]string{"--version", ""})
	if !handled || code != 23 {
		t.Fatalf("version bridge %v %d", handled, code)
	}
	handled, code = RunPodmanBridge([]string{"--url=ssh://wrong", "info"})
	if !handled || code != 125 {
		t.Fatalf("routing override accepted %v %d", handled, code)
	}
}

func TestPodmanAnonymousVolumeProofRejectsAmbiguity(t *testing.T) {
	valid := podmanAnonymousVolume{Name: "anonymous", Mountpoint: "/volume", CreatedAt: "2026-09-09T00:00:00Z", Driver: "local", Anonymous: true}
	proof, err := valid.proof()
	if err != nil || proof == "" {
		t.Fatal(proof, err)
	}
	for _, mutation := range []func(*podmanAnonymousVolume){func(v *podmanAnonymousVolume) { v.Anonymous = false }, func(v *podmanAnonymousVolume) { v.Labels = map[string]string{"foreign": "owner"} }, func(v *podmanAnonymousVolume) { v.CreatedAt = "" }, func(v *podmanAnonymousVolume) { v.Mountpoint = "" }} {
		candidate := valid
		mutation(&candidate)
		if _, err := candidate.proof(); err == nil {
			t.Fatalf("ambiguous proof accepted %#v", candidate)
		}
	}
}
func TestPodmanAnonymousVolumeDoesNotDeleteReplacement(t *testing.T) {
	p := testPodmanIdentity(t)
	old := podmanAnonymousVolume{Name: "volume", Mountpoint: "/volume", CreatedAt: "old", Driver: "local", Anonymous: true}
	proof, _ := old.proof()
	replacement := old
	replacement.CreatedAt = "new"
	data, _ := json.Marshal([]podmanAnonymousVolume{replacement})
	calls := 0
	c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		calls++
		switch {
		case reflect.DeepEqual(q.Args, []string{"--remote=false", "volume", "ls", "--quiet"}):
			return execx.Result{Stdout: "volume\n"}, nil
		case reflect.DeepEqual(q.Args, []string{"--remote=false", "volume", "inspect", "volume"}):
			return execx.Result{Stdout: string(data)}, nil
		default:
			t.Fatalf("unsafe request %#v", q)
			return execx.Result{}, nil
		}
	})}
	err := c.removeAnonymousVolumes(context.Background(), domain.Runtime{Context: p.encode()}, []domain.Resource{{ExternalID: "volume", Metadata: map[string]string{"volume_fingerprint": proof}}})
	if err == nil || !strings.Contains(err.Error(), "identity changed") || calls != 2 {
		t.Fatalf("%v calls=%d", err, calls)
	}
}

func TestPodmanCleanupEvidenceScope(t *testing.T) {
	r := domain.Runtime{Name: "runtime", LeaseID: "lease", Project: "ae_fixture", Context: testPodmanIdentity(t).encode()}
	v := podmanAnonymousVolume{Name: "volume", Mountpoint: "/volume", CreatedAt: "now", Anonymous: true}
	fingerprint, _ := v.proof()
	proof := resource(r, "volume", "volume", map[string]string{"anonymous": "true", "provider": "podman-compose", "lease": r.LeaseID, "runtime": r.Name, "project": r.Project, "context": r.Context, "attached_container": "container-id", "volume_fingerprint": fingerprint})
	if err := validatePodmanCleanupEvidence(r, proof); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"provider", "lease", "runtime", "project", "context", "attached_container", "volume_fingerprint"} {
		old := proof.Metadata[key]
		proof.Metadata[key] = ""
		if err := validatePodmanCleanupEvidence(r, proof); err == nil {
			t.Fatal("missing proof accepted:", key)
		}
		proof.Metadata[key] = old
	}
}

func TestPodmanDoctorVersionFloorAndSnapshot(t *testing.T) {
	directory := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	for _, name := range []string{"podman", "podman-compose"} {
		if err := os.WriteFile(filepath.Join(directory, name+suffix), []byte("fake executable"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", directory)
	t.Setenv("CONTAINER_CONNECTION", "")
	t.Setenv("CONTAINER_HOST", "unix:///fixture/podman.sock")
	t.Setenv("CONTAINER_SSHKEY", "")
	for _, version := range []string{"1.3.0", "2.0.0", "1.6.0"} {
		t.Run(version, func(t *testing.T) {
			c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
				if strings.Contains(filepath.Base(q.Name), "podman-compose") {
					if len(q.Args) != 3 || q.Args[0] != "--podman-path" || q.Args[2] != "--version" || q.Env[podmanBridgeEnv] == "" {
						t.Fatalf("unpinned version probe %#v", q)
					}
					return execx.Result{Stdout: "podman-compose version " + version + "\npodman version 5.4.2\n"}, nil
				}
				if q.Args[len(q.Args)-1] == "--version" {
					return execx.Result{Stdout: "podman version 5.4.2"}, nil
				}
				return execx.Result{Stdout: `{"host":{"hostname":"host","os":"linux","arch":"amd64","security":{"rootless":true}},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
			})}
			details, err := c.Doctor(context.Background())
			if version != "1.6.0" {
				if err == nil || !strings.Contains(err.Error(), ">=1.6.0") {
					t.Fatal(err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"client_version", "server_version", "compose_version", "context", "engine_fingerprint", "mode", "rootless"} {
				if details[key] == "" {
					t.Fatalf("missing %s in %#v", key, details)
				}
			}
			identity, err := decodePodmanIdentity(details["context"])
			if err != nil || identity.URL != "unix:///fixture/podman.sock" || !filepath.IsAbs(identity.ComposeExecutable) {
				t.Fatal(identity, err)
			}
		})
	}
}
func TestPodmanSnapshotKeepsLiteralDollars(t *testing.T) {
	input := []byte(`{"services":{"web":{"environment":{"LITERAL":"$HOME ${TOKEN} $$"}}}}`)
	data, err := escapePodmanSnapshot(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "$$HOME $${TOKEN} $$$$") {
		t.Fatal(string(data))
	}
}

func TestPodmanInventoryDoctorDoesNotRequireCompose(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "podman")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := os.WriteFile(binary, []byte("fake executable"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("CONTAINER_CONNECTION", "")
	t.Setenv("CONTAINER_HOST", "unix:///fixture/podman.sock")
	t.Setenv("CONTAINER_SSHKEY", "")
	c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		if q.Name != binary {
			t.Fatalf("inventory executed compose: %#v", q)
		}
		return execx.Result{Stdout: `{"host":{"hostname":"host","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
	})}
	result, err := c.InventoryDoctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := decodePodmanIdentity(result["context"])
	if err != nil || identity.ComposeExecutable != "" {
		t.Fatal(identity, err)
	}
	if _, err = c.Doctor(context.Background()); err == nil || !strings.Contains(err.Error(), "podman-compose prerequisite missing") {
		t.Fatal(err)
	}
}

func TestPodmanRejectsNestedExecutionExtensions(t *testing.T) {
	for _, fixture := range []string{
		"services: {web: {image: fixture}}\nnetworks: {default: {x-podman.disable_dns: true}}",
		"services: {web: {image: fixture, networks: {default: {x-podman.interface_name: eth0}}}}",
		"services: {web: {image: fixture}}\nsecrets: {token: {file: token, x-podman.relabel: z}}",
		"services: {web: {image: fixture}}\nnetworks: {default: {X-PODMAN.dns: [127.0.0.1]}}",
		"services: {web: {image: fixture, secrets: [{source: token, x-podman.relabel: z}]}}",
	} {
		if _, err := normalizePodmanConfig(fixture, t.TempDir(), "ae_fixture"); err == nil || !strings.Contains(err.Error(), "provider execution extension") {
			t.Fatalf("accepted nested extension %s: %v", fixture, err)
		}
	}
}
func TestPodmanInspectRetainedAnonymousVolumeExists(t *testing.T) {
	p := testPodmanIdentity(t)
	volume := podmanAnonymousVolume{Name: "retained-volume", Mountpoint: "/volume", CreatedAt: "now", Driver: "local", Anonymous: true}
	proof, _ := volume.proof()
	volumeJSON, _ := json.Marshal([]podmanAnonymousVolume{volume})
	c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		args := q.Args[1:]
		switch {
		case reflect.DeepEqual(args, []string{"info", "--format", "json"}):
			return execx.Result{Stdout: `{"host":{"hostname":"host","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
		case reflect.DeepEqual(args, []string{"volume", "ls", "--quiet"}):
			return execx.Result{Stdout: "retained-volume\n"}, nil
		case reflect.DeepEqual(args, []string{"volume", "inspect", "retained-volume"}):
			return execx.Result{Stdout: string(volumeJSON)}, nil
		case len(args) > 0 && args[0] == "ps":
			return execx.Result{}, nil
		case len(args) > 1 && (args[0] == "volume" || args[0] == "network") && args[1] == "ls":
			return execx.Result{}, nil
		default:
			t.Fatalf("unexpected command %#v", q)
			return execx.Result{}, nil
		}
	})}
	var err error
	p.Fingerprint, _, err = c.fingerprint(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	r := domain.Runtime{Name: "runtime", LeaseID: "lease", Project: "ae_fixture", Context: p.encode(), Directory: t.TempDir()}
	r.ConfigPath = filepath.Join(r.Directory, "compose.json")
	if err = os.WriteFile(r.ConfigPath, []byte(`{"services":{"web":{"image":"fixture"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	r.CleanupEvidence = []domain.Resource{resource(r, "volume", volume.Name, map[string]string{"anonymous": "true", "provider": "podman-compose", "lease": r.LeaseID, "runtime": r.Name, "project": r.Project, "context": r.Context, "attached_container": "deleted-container", "volume_fingerprint": proof})}
	observation, err := c.Inspect(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Exists || len(observation.Resources) != 1 || observation.Resources[0].ExternalID != volume.Name {
		t.Fatalf("retained volume lost: %#v", observation)
	}
}

func TestPodmanRenderRejectsProviderSpecificHostAccess(t *testing.T) {
	fixtures := []string{
		"services: {web: {image: fixture, environment: {TOKEN: null}}}",
		"services: {web: {image: fixture, environment: [TOKEN]}}",
		"services: {web: {image: fixture, volumes: [{type: glob, source: '/etc/*', target: /host}]}}",
		"services: {web: {image: fixture, volumes: [{type: 7, source: /etc, target: /host}]}}",
		"services: {web: {image: fixture, volumes: [{type: bind, source: 7, target: /host}]}}",
		"services: {web: {image: fixture, volumes: {type: bind, source: /etc, target: /host}}}",
		"services: {web: {image: fixture, network_mode: 'ns:/proc/1/ns/net'}}",
		"services: {web: {image: fixture, network_mode: pasta}}",
		"services: {web: {image: fixture, network_mode: 'slirp4netns:allow_host_loopback=true'}}",
		"services: {web: {image: fixture, network_mode: 'container:foreign'}}",
		"services: {web: {image: fixture, network_mode: 'service:foreign'}}",
		"services: {web: {image: fixture, network_mode: 7}}",
		"services: {web: {image: fixture}}\nnetworks: {default: {x-podman.routes: [{destination: default}]}}",
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			p := testPodmanIdentity(t)
			configCalls := 0
			c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
				if q.Name == p.ComposeExecutable {
					configCalls++
					return execx.Result{Stdout: fixture}, nil
				}
				if !reflect.DeepEqual(q.Args, []string{"--remote=false", "info", "--format", "json"}) {
					t.Fatalf("unexpected native command %#v", q)
				}
				return execx.Result{Stdout: `{"host":{"hostname":"host","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
			})}
			var err error
			p.Fingerprint, _, err = c.fingerprint(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			r := domain.Runtime{Name: "runtime", LeaseID: "lease", Project: "ae_fixture", Context: p.encode(), Directory: directory, Files: []string{filepath.Join(directory, "compose.yaml")}}
			if _, err = c.Render(context.Background(), r, []string{directory}); err == nil {
				t.Fatal("accepted unsafe raw provider config")
			}
			if configCalls != 1 {
				t.Fatalf("did not exercise raw render path: %d", configCalls)
			}
		})
	}
}

func TestPodmanNativeServiceOwnership(t *testing.T) {
	for _, labels := range []string{
		`{"io.podman.compose.project":"ae_fixture","com.docker.compose.service":"web"}`,
		`{"io.podman.compose.project":"ae_fixture","io.podman.compose.service":"web","com.docker.compose.service":"other"}`,
	} {
		if _, err := normalizePodmanInspection(`[{"Config":{"Labels":`+labels+`}}]`, "ae_fixture"); err == nil {
			t.Fatal("accepted missing/conflicting native service", labels)
		}
	}
	output, err := normalizePodmanInspection(`[{"Config":{"Labels":{"io.podman.compose.project":"ae_fixture","io.podman.compose.service":"web"}}}]`, "ae_fixture")
	if err != nil || !strings.Contains(output, `"com.docker.compose.service":"web"`) {
		t.Fatal(output, err)
	}
}
func TestPodmanRelativeFileReferencesSurviveSnapshotRelocation(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"service.env", "config.txt", "secret.txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("KEY=value\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	output, err := normalizePodmanConfig(`services:
  web:
    image: fixture
    env_file: [service.env]
    configs: [settings]
    secrets: [token]
configs:
  settings: {file: config.txt}
secrets:
  token: {file: secret.txt}
`, directory, "ae_fixture")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err = json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatal(err)
	}
	env := raw["services"].(map[string]any)["web"].(map[string]any)["env_file"].([]any)[0].(string)
	config := raw["configs"].(map[string]any)["settings"].(map[string]any)["file"].(string)
	secret := raw["secrets"].(map[string]any)["token"].(map[string]any)["file"].(string)
	for _, path := range []string{env, config, secret} {
		if !filepath.IsAbs(path) {
			t.Fatal("relative reference survived", path)
		}
		if _, err = os.ReadFile(path); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.env")
	if err = os.WriteFile(outside, []byte("KEY=secret"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{outside, directory, filepath.Join(directory, "missing.env")} {
		quoted, _ := json.Marshal(path)
		if _, err = normalizePodmanConfig("services: {web: {image: fixture, env_file: ["+string(quoted)+"]}}", directory, "ae_fixture"); err == nil {
			t.Fatal("unsafe file accepted", path)
		}
	}
}
func TestPodmanRejectsInitialFileDirectoryMismatchBeforeProvider(t *testing.T) {
	p := testPodmanIdentity(t)
	called := false
	c := podmanClient{Runner: podmanRunnerFunc(func(context.Context, execx.Command) (execx.Result, error) { called = true; return execx.Result{}, nil })}
	directory := t.TempDir()
	_, err := (podmanAdapter{client: c, identity: p, project: "ae_fixture"}).compose(context.Background(), execx.Command{}, []string{"--project-directory", directory, "-f", filepath.Join(directory, "nested", "compose.yaml"), "config", "--format", "json"})
	if err == nil || !strings.Contains(err.Error(), "must share project_directory") || called {
		t.Fatal(err, called)
	}
}

func TestPodmanEnvironmentRequiresExplicitPassThroughValues(t *testing.T) {
	t.Setenv("TOKEN", "host-secret-that-must-not-be-captured")
	for _, environment := range []string{"{TOKEN: null}", "[TOKEN]"} {
		_, err := normalizePodmanConfig("services: {web: {image: fixture, environment: "+environment+"}}", t.TempDir(), "ae_fixture")
		if err == nil || !strings.Contains(err.Error(), "unresolved environment pass-through TOKEN") {
			t.Fatal(err)
		}
		if strings.Contains(err.Error(), "host-secret-that-must-not-be-captured") {
			t.Fatal("diagnostic exposed ambient secret")
		}
	}
	output, err := normalizePodmanConfig(`services: {web: {image: fixture, environment: {TOKEN: "", LITERAL: "$TOKEN"}, labels: [empty-label]}}`, t.TempDir(), "ae_fixture")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"TOKEN":""`) || !strings.Contains(output, `"LITERAL":"$TOKEN"`) || !strings.Contains(output, `"empty-label":""`) {
		t.Fatal(output)
	}
}

func TestPodmanProviderFailurePreservesRedactedNativeDiagnostic(t *testing.T) {
	t.Setenv("API_TOKEN", "sensitive-fixture-token")
	p := testPodmanIdentity(t)
	c := podmanClient{Runner: podmanRunnerFunc(func(context.Context, execx.Command) (execx.Result, error) {
		return execx.Result{ExitCode: 125, Stderr: "Error: invalid published port 0; API_TOKEN=sensitive-fixture-token"}, nil
	})}
	_, err := (podmanAdapter{client: c, identity: p}).compose(context.Background(), execx.Command{}, []string{"--project-directory", t.TempDir(), "config", "--format", "json"})
	if err == nil || !strings.Contains(err.Error(), "podman-compose failed (exit 125)") || !strings.Contains(err.Error(), "invalid published port 0") || strings.Contains(err.Error(), "sensitive-fixture-token") {
		t.Fatal(err)
	}
}

func TestPodmanUpUsesDynamicPortWithoutChangingCanonicalSnapshot(t *testing.T) {
	for _, published := range []string{`"0"`, `0`} {
		t.Run(published, func(t *testing.T) {
			p := testPodmanIdentity(t)
			directory := t.TempDir()
			snapshot := filepath.Join(directory, "compose.json")
			original := []byte(`{"services":{"web":{"image":"fixture","ports":[{"host_ip":"127.0.0.1","published":` + published + `,"target":8080,"protocol":"tcp"}]}}}`)
			if err := os.WriteFile(snapshot, original, 0600); err != nil {
				t.Fatal(err)
			}
			providerCalls := 0
			c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
				if q.Name == p.ComposeExecutable {
					providerCalls++
					var path string
					for i, arg := range q.Args {
						if arg == "-f" {
							path = q.Args[i+1]
						}
					}
					if path == "" || path == snapshot {
						t.Fatalf("provider did not receive private snapshot: %#v", q)
					}
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					var root map[string]any
					if err = json.Unmarshal(data, &root); err != nil {
						t.Fatal(err)
					}
					port := root["services"].(map[string]any)["web"].(map[string]any)["ports"].([]any)[0].(map[string]any)
					if _, exists := port["published"]; exists {
						t.Fatal("zero host port reached Podman", port)
					}
					if port["host_ip"] != "127.0.0.1" || port["target"] != float64(8080) || port["protocol"] != "tcp" {
						t.Fatal("port scope changed", port)
					}
					return execx.Result{}, nil
				}
				args := q.Args[1:]
				if reflect.DeepEqual(args, []string{"info", "--format", "json"}) {
					return execx.Result{Stdout: `{"host":{"hostname":"host","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`}, nil
				}
				if len(args) > 0 && args[0] == "ps" {
					return execx.Result{}, nil
				}
				if len(args) > 1 && (args[0] == "network" || args[0] == "volume") && args[1] == "ls" {
					return execx.Result{}, nil
				}
				t.Fatalf("unexpected native request %#v", q)
				return execx.Result{}, nil
			})}
			var err error
			p.Fingerprint, _, err = c.fingerprint(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			r := domain.Runtime{Name: "runtime", LeaseID: "lease", Project: "ae_fixture", Context: p.encode(), Directory: directory, ConfigPath: snapshot, Services: []string{"web"}}
			if err = c.Up(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(snapshot)
			if err != nil || string(after) != string(original) {
				t.Fatal("canonical snapshot modified", err)
			}
			if providerCalls != 1 {
				t.Fatal(providerCalls)
			}
		})
	}
}
