package app

import (
	"context"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanProcessPinsSourceWithoutRuntimeEffects(t *testing.T) {
	repo := t.TempDir()
	body := `version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  server:
    type: process
    source: app
    working_directory: .
    command: [server, '--port=${port:http}']
    env: {AUTH_TOKEN: '${env:PROCESS_PLAN_TEST_TOKEN}'}
    ports: {http: {protocol: tcp}}
components:
  api:
    runtime: server
    endpoints: {http: {runtime_port: http}}
    readiness: [{type: http, url: 'http://127.0.0.1:${endpoint:http}/health'}]
stacks:
  api: {roots: [api]}
`
	path := filepath.Join(repo, ".agent-env.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	source := &planningSource{}
	plan, err := BuildPlan(context.Background(), PlanOptions{Repository: repo, Stack: "api"}, source)
	if err != nil {
		t.Fatal(err)
	}
	r := plan.Runtimes[0]
	if r.Provider != "" || r.Process == nil || r.Process.SourceCommit != plan.Sources[0].Commit || r.Process.Ports["http"] != 0 || r.Process.ProcessID != 0 || r.Process.Directory != "" || r.Process.State != "planned" {
		t.Fatalf("unexpected process plan: %+v", r)
	}
	if len(source.calls) != 1 || r.Process.Env["AUTH_TOKEN"] != "${env:PROCESS_PLAN_TEST_TOKEN}" {
		t.Fatalf("lost declarative evidence: %+v", r.Process)
	}
	entries, err := os.ReadDir(repo)
	if err != nil || len(entries) != 1 {
		t.Fatalf("plan allocated state: %v %v", entries, err)
	}
	for _, literal := range []string{"literal-credential", "inherited-private-value"} {
		t.Setenv("PROCESS_PLAN_TEST_TOKEN", "inherited-private-value")
		bad := strings.Replace(body, "${env:PROCESS_PLAN_TEST_TOKEN}", literal, 1)
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		source := &planningSource{}
		if _, err := BuildPlan(context.Background(), PlanOptions{Repository: repo, Stack: "api"}, source); err == nil || len(source.calls) != 0 {
			t.Fatalf("literal secret accepted or reached source resolution: %v", err)
		}
	}
}

func TestProcessEndpointAliasWinsOverRawRuntimePort(t *testing.T) {
	p := &lifecycleProcess{alive: map[string]bool{"lease:api": true}}
	service := &Service{Process: p}
	lease := domain.Lease{ID: "lease", Manifest: []byte(`{"components":{"api":{"endpoints":{"http":{"runtime_port":"metrics"}}}}}`), Components: []domain.Component{{Name: "api", Runtime: "api"}}, Runtimes: []domain.Runtime{{Name: "api", Type: "process", LeaseID: "lease", Process: &domain.PersistentProcess{Ports: map[string]int{"http": 12001, "metrics": 12002}}}}}
	endpoints, err := service.Endpoints(context.Background(), lease)
	if err != nil || endpoints["api.http"] != "127.0.0.1:12002" {
		t.Fatalf("declared alias overwritten: %v %v", endpoints, err)
	}
}

func TestProcessReadinessRejectsLiteralInheritedSecretBeforeSnapshot(t *testing.T) {
	t.Setenv("PROCESS_READINESS_TEST_TOKEN", "readiness-private-token")
	manifest := config.Manifest{Runtimes: map[string]config.Runtime{"api": {Type: "process"}}, Components: map[string]config.Component{"api": {Runtime: "api", Readiness: []config.Probe{{Type: "command", Command: []string{"checker", "readiness-private-token"}}}}}}
	if err := validateProcessSecrets(manifest); err == nil {
		t.Fatal("literal readiness secret accepted")
	}
	component := manifest.Components["api"]
	component.Readiness[0].Command[1] = "${env:PROCESS_READINESS_TEST_TOKEN}"
	manifest.Components["api"] = component
	if err := validateProcessSecrets(manifest); err != nil {
		t.Fatal(err)
	}
}
