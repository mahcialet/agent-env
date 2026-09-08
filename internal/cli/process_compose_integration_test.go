//go:build integration

package cli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"gopkg.in/yaml.v3"
)

func TestIntegrationPersistentProcessHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_PERSISTENT_FIXTURE") != "1" {
		return
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:"+os.Getenv("PROCESS_FIXTURE_PORT"))
	if err != nil {
		os.Exit(21)
	}
	if err := os.MkdirAll(filepath.Join(os.Getenv("PROCESS_FIXTURE_STATE"), "profile"), 0700); err != nil {
		os.Exit(22)
	}
	fmt.Fprintln(os.Stdout, "persistent fixture ready")
	http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "process fixture") }))
	os.Exit(0)
}

func TestIntegrationPersistentProcessComposeCoexistence(t *testing.T) {
	repo, _, manifest := integrationFixture(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(executable)+string(os.PathListSeparator)+os.Getenv("PATH"))
	manifest.Runtimes["host"] = config.Runtime{Type: "process", Source: "self", WorkingDirectory: ".", Command: []string{filepath.Base(executable), "-test.run=^TestIntegrationPersistentProcessHelper$"}, Env: map[string]string{"AGENT_ENV_PERSISTENT_FIXTURE": "1", "PROCESS_FIXTURE_PORT": "${port:http}", "PROCESS_FIXTURE_STATE": "${runtime_dir}"}, Ports: map[string]config.ProcessPort{"http": {Protocol: "tcp"}}}
	manifest.Components["host"] = config.Component{Runtime: "host", Endpoints: map[string]config.Endpoint{"http": {RuntimePort: "http"}}, Readiness: []config.Probe{{Type: "http", URL: "http://127.0.0.1:${endpoint:http}/", Timeout: "10s", Interval: "100ms"}}}
	manifest.Stacks["mixed"] = config.Stack{Roots: []string{"api", "host"}}
	writeIntegrationManifest(t, repo, manifest)
	// The fixture serializer retains legacy Compose zero-value fields; process
	// declarations reject their presence, including empty values.
	manifestPath := filepath.Join(repo, ".agent-env.yaml")
	encoded, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	runtime := document["runtimes"].(map[string]any)["host"].(map[string]any)
	allowed := map[string]bool{"type": true, "source": true, "working_directory": true, "command": true, "env": true, "ports": true}
	for key := range runtime {
		if !allowed[key] {
			delete(runtime, key)
		}
	}
	delete(document["components"].(map[string]any)["host"].(map[string]any), "compose_services")
	endpoint := document["components"].(map[string]any)["host"].(map[string]any)["endpoints"].(map[string]any)["http"].(map[string]any)
	for key := range endpoint {
		if key != "runtime_port" {
			delete(endpoint, key)
		}
	}
	encoded, err = yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	integrationCommit(t, repo)
	mixed := createIntegration(t, repo, "mixed")
	sibling := createIntegration(t, repo, "mixed")
	process := func(l domain.Lease) *domain.PersistentProcess {
		for _, r := range l.Runtimes {
			if r.Type == "process" {
				return r.Process
			}
		}
		t.Fatal("missing process")
		return nil
	}
	first, second := process(mixed), process(sibling)
	if first.Ports["http"] == second.Ports["http"] || first.Directory == second.Directory {
		t.Fatal("shared process port or state")
	}
	endpoints := integrationJSON[struct {
		Endpoints map[string]string `json:"endpoints"`
	}](t, "capabilities", mixed.ID).Endpoints
	for _, name := range []string{"api.http", "host.http"} {
		response, err := (&http.Client{Timeout: 5 * time.Second}).Get("http://" + endpoints[name] + "/")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("%s: %d", name, response.StatusCode)
		}
	}
	logs := integrationJSON[struct {
		Logs map[string]string `json:"logs"`
	}](t, "logs", mixed.ID, "--component", "host")
	if len(logs.Logs) != 1 || !strings.Contains(logs.Logs["host"], "persistent fixture ready") {
		t.Fatalf("process logs: %+v", logs)
	}
	integrationJSON[domain.Lease](t, "destroy", mixed.ID)
	if _, err := os.Stat(first.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("mutable state survived: %v", err)
	}
	remaining := integrationJSON[integrationShow](t, "show", sibling.ID).Lease
	if remaining.Observed != "ready" || process(remaining).ProcessID != second.ProcessID {
		t.Fatalf("sibling disturbed: %+v", remaining)
	}
	if diff := integrationCommand(t, sibling.Sources[0].WorktreePath, "git", "status", "--porcelain"); diff != "" {
		t.Fatalf("source polluted: %s", diff)
	}
	retained := integrationJSON[struct {
		Logs map[string]string `json:"logs"`
	}](t, "logs", mixed.ID, "--component", "host")
	found := false
	for _, log := range retained.Logs {
		found = found || strings.Contains(log, "persistent fixture ready")
	}
	if !found {
		t.Fatal("missing retained process logs")
	}
}
