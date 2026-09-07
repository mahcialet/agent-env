package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

type perServiceLogRuntime struct {
	RuntimeProvider
	calls [][]string
}

func (r *perServiceLogRuntime) Logs(_ context.Context, v domain.Runtime) (string, error) {
	r.calls = append(r.calls, append([]string(nil), v.Services...))
	if len(v.Services) != 1 {
		return "", fmt.Errorf("cleanup logs require exact service scope")
	}
	return "only-" + v.Services[0], nil
}

func TestCleanupRetainsSeparatelyScopedLogsForSharedRuntime(t *testing.T) {
	s, options, _, runtime, _ := lifecycleFixture(t)
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  shared: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
components:
  api: {runtime: shared, compose_services: [db, api]}
  dashboard: {runtime: shared, compose_services: [dashboard], depends_on: [api]}
stacks:
  review: {roots: [dashboard]}
`
	if err := os.WriteFile(filepath.Join(options.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	logs := &perServiceLogRuntime{RuntimeProvider: runtime}
	s.Runtime = logs
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Destroy(context.Background(), lease.ID, false, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(logs.calls, [][]string{{"db"}, {"api"}, {"dashboard"}}) {
		t.Fatalf("aggregate or duplicate collection: %v", logs.calls)
	}
	artifacts, err := s.Store.Artifacts(context.Background(), lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{}
	for _, a := range artifacts {
		switch a.Kind {
		case "compose-log/shared/db", "compose-log/shared/api", "compose-log/shared/dashboard":
			kinds = append(kinds, a.Kind)
			data, err := os.ReadFile(a.Path)
			if err != nil {
				t.Fatal(err)
			}
			name := filepath.Base(a.Kind)
			if string(data) != "only-"+name {
				t.Fatalf("archive mixes services: %s %s", a.Kind, data)
			}
		case "compose-log":
			t.Fatal("duplicated aggregate retained")
		}
	}
	sort.Strings(kinds)
	if !reflect.DeepEqual(kinds, []string{"compose-log/shared/api", "compose-log/shared/dashboard", "compose-log/shared/db"}) {
		t.Fatalf("missing service archives: %v", kinds)
	}
}
