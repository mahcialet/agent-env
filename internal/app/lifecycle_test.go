package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

type lifecycleSource struct {
	states     map[string]SourceObservation
	operations *[]string
	failAlias  string
}

func (f *lifecycleSource) Resolve(_ context.Context, repo, ref string) (domain.Source, error) {
	return domain.Source{RepositoryID: repo, RepositoryPath: repo, RequestedRef: ref, Commit: "0123456789012345678901234567890123456789", ResolvedAt: time.Now()}, nil
}
func (f *lifecycleSource) Materialize(_ context.Context, s domain.Source) error {
	*f.operations = append(*f.operations, "materialize:"+s.Alias)
	if err := os.MkdirAll(s.WorktreePath, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.WorktreePath, "compose.yaml"), []byte("services: {}\n"), 0600); err != nil {
		return err
	}
	f.states[s.WorktreePath] = SourceObservation{Exists: true, Registered: true, Commit: s.Commit}
	if s.Alias == f.failAlias {
		return errors.New("materialization partial failure")
	}
	return nil
}
func (f *lifecycleSource) Inspect(_ context.Context, s domain.Source) (SourceObservation, error) {
	return f.states[s.WorktreePath], nil
}
func (f *lifecycleSource) Remove(_ context.Context, s domain.Source, force bool) error {
	*f.operations = append(*f.operations, "remove:"+s.Alias)
	if f.states[s.WorktreePath].TrackedDirty && !force {
		return errors.New("dirty")
	}
	delete(f.states, s.WorktreePath)
	return nil
}
func (f *lifecycleSource) Diff(context.Context, domain.Source) (string, error) {
	return "tracked patch evidence", nil
}

type lifecycleRuntime struct {
	projects                    map[string]bool
	operations                  *[]string
	upCount, failUp             int
	failDown, notReady, foreign bool
	store                       Store
	cancelUp                    context.CancelFunc
}

func (f *lifecycleRuntime) Doctor(context.Context) (map[string]string, error) {
	return map[string]string{"context": "test-context"}, nil
}
func (f *lifecycleRuntime) Render(_ context.Context, r domain.Runtime, _ []string) (Rendered, error) {
	services := map[string]any{}
	for _, s := range r.Services {
		services[s] = map[string]any{"image": "fixture"}
	}
	b, _ := json.Marshal(map[string]any{"services": services})
	return Rendered{JSON: b, Services: r.Services}, nil
}
func (f *lifecycleRuntime) Up(ctx context.Context, r domain.Runtime) error {
	*f.operations = append(*f.operations, "up:"+r.Name)
	f.upCount++
	f.projects[r.Project] = true
	if r.ConfigPath == "" || r.ConfigDigest == "" {
		return errors.New("up without persisted config")
	}
	leases, err := f.store.List(ctx)
	if err != nil {
		return err
	}
	saved := false
	for _, l := range leases {
		for _, stored := range l.Runtimes {
			if stored.Project == r.Project && stored.Started {
				saved = true
			}
		}
	}
	if !saved {
		return errors.New("up before saved runtime intent")
	}
	if f.upCount == f.failUp {
		if f.cancelUp != nil {
			f.cancelUp()
		}
		return errors.New("up partially allocated then failed")
	}
	return nil
}

func TestLifecycleCanceledRequestStillCompensates(t *testing.T) {
	s, o, source, runtime, _ := lifecycleFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime.failUp = 2
	runtime.cancelUp = cancel
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err == nil || l.Observed != "released" || len(source.states) != 0 || len(runtime.projects) != 0 {
		t.Fatalf("cancellation stranded saga %+v %v", l, err)
	}
}
func (f *lifecycleRuntime) Down(_ context.Context, r domain.Runtime) error {
	*f.operations = append(*f.operations, "down:"+r.Name)
	if f.failDown {
		return errors.New("down failed")
	}
	delete(f.projects, r.Project)
	return nil
}
func (f *lifecycleRuntime) Inspect(_ context.Context, r domain.Runtime) (RuntimeObservation, error) {
	exists := f.projects[r.Project]
	resources := []domain.Resource{}
	if exists {
		metadata := map[string]string{"project": r.Project, "lease": r.LeaseID, "runtime": r.Name}
		if f.foreign {
			metadata["lease"] = "foreign"
		}
		resources = append(resources, domain.Resource{ID: r.Project, Runtime: r.Name, Kind: "container", ExternalID: r.Project, Metadata: metadata})
	}
	return RuntimeObservation{Ready: exists && !f.notReady, Exists: exists, Resources: resources, Diagnostics: []string{}}, nil
}
func (f *lifecycleRuntime) Logs(_ context.Context, r domain.Runtime) (string, error) {
	*f.operations = append(*f.operations, "logs:"+r.Name)
	return "runtime logs", nil
}

func TestOwnedResourcesRejectMissingLeaseLabel(t *testing.T) {
	resource := domain.Resource{Kind: "volume", ExternalID: "same-project-volume", Metadata: map[string]string{"project": "ae_project"}}
	if err := ownedResources("expected-lease", []domain.Resource{resource}); err == nil {
		t.Fatal("missing ownership label was accepted")
	}
	resource.Metadata["lease"] = "expected-lease"
	if err := ownedResources("expected-lease", []domain.Resource{resource}); err != nil {
		t.Fatal(err)
	}
}

func TestOwnedConfigurationLabelsSelectedNamedResources(t *testing.T) {
	r := domain.Runtime{Name: "backend", Project: "ae_project", Services: []string{"api"}, LeaseID: "lease123"}
	data, err := ownedConfiguration([]byte(`{"services":{"api":{"labels":{"user":"kept"}}},"networks":{"default":{"name":"ae_project_default","labels":{"user":"kept"}}},"volumes":{"data":{"name":"ae_project_data"}}}`), r, r.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]map[string]struct {
		Labels map[string]string `json:"labels"`
	}
	// The top-level name is scalar; read only the resource maps under test.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	parsed = make(map[string]map[string]struct {
		Labels map[string]string `json:"labels"`
	})
	for _, kind := range []string{"services", "networks", "volumes"} {
		var resources map[string]struct {
			Labels map[string]string `json:"labels"`
		}
		if err := json.Unmarshal(raw[kind], &resources); err != nil {
			t.Fatal(err)
		}
		parsed[kind] = resources
		for name, resource := range resources {
			if resource.Labels["io.agent-env.lease"] != r.LeaseID || resource.Labels["io.agent-env.runtime"] != r.Name {
				t.Fatalf("%s/%s lacks ownership: %v", kind, name, resource.Labels)
			}
		}
	}
	if parsed["networks"]["default"].Labels["user"] != "kept" {
		t.Fatal("user labels overwritten")
	}
}

func lifecycleFixture(t *testing.T) (*Service, PlanOptions, *lifecycleSource, *lifecycleRuntime, *[]string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo 日本語 spaces")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  first: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
  second: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
components:
  database: {runtime: first, compose_services: [db]}
  api: {runtime: second, compose_services: [api], depends_on: [database]}
stacks:
  review: {roots: [api]}
`
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := sqlite.Open(filepath.Join(root, "state", "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	operations := []string{}
	source := &lifecycleSource{states: map[string]SourceObservation{}, operations: &operations}
	runtime := &lifecycleRuntime{projects: map[string]bool{}, operations: &operations, store: store}
	service := &Service{Store: store, Source: source, Runtime: runtime, Home: filepath.Join(root, "state"), Policy: policy.Defaults(), ReadinessTimeout: 5 * time.Second, ReadinessInterval: time.Millisecond}
	return service, PlanOptions{Repository: repo, Stack: "review"}, source, runtime, &operations
}

func TestLifecycleCreatePersistedIntentAndUniqueIsolation(t *testing.T) {
	s, o, _, runtime, _ := lifecycleFixture(t)
	ctx := context.Background()
	one, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	two, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if one.Observed != "ready" || one.Desired != "active" || one.SourceSetDigest != two.SourceSetDigest || one.ID == two.ID || one.Sources[0].WorktreePath == two.Sources[0].WorktreePath || one.Runtimes[0].Project == two.Runtimes[0].Project {
		t.Fatalf("isolation/identity: %+v %+v", one, two)
	}
	if len(runtime.projects) != 4 {
		t.Fatalf("project count %d", len(runtime.projects))
	}
	for _, r := range one.Runtimes {
		b, err := os.ReadFile(r.ConfigPath)
		if err != nil || !strings.Contains(string(b), one.ID) || !strings.Contains(string(b), "io.agent-env.lease") {
			t.Fatalf("ownership config: %s %v", b, err)
		}
	}
	if _, err := s.Destroy(ctx, one.ID, false, false); err != nil {
		t.Fatal(err)
	}
	if len(runtime.projects) != 2 {
		t.Fatal("destroy crossed lease boundary")
	}
	two, err = s.Reconcile(ctx, two.ID)
	if err != nil || two.Observed != "ready" {
		t.Fatalf("other lease harmed: %+v %v", two, err)
	}
}

func TestLifecyclePartialStartupCompensatesReverseAndKeepsEvidence(t *testing.T) {
	s, o, source, runtime, operations := lifecycleFixture(t)
	runtime.failUp = 2
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if err == nil || l.Observed != "released" {
		t.Fatalf("failed saga %+v %v", l, err)
	}
	if len(runtime.projects) != 0 || len(source.states) != 0 {
		t.Fatal("partial allocation leaked")
	}
	joined := strings.Join(*operations, ",")
	if !strings.Contains(joined, "logs:second,down:second,logs:first,down:first,remove:self") {
		t.Fatalf("wrong compensation order %s", joined)
	}
	events, err := s.Store.Events(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		found = found || e.Type == "allocation_failed"
	}
	if !found {
		t.Fatal("failure evidence missing")
	}
	artifacts, err := s.Store.Artifacts(context.Background(), l.ID)
	logs := 0
	doctorInfo := 0
	for _, a := range artifacts {
		if strings.HasPrefix(a.Kind, "compose-log/") {
			logs++
		}
		if a.Kind == "docker-info" {
			doctorInfo++
		}
	}
	if err != nil || logs != 2 || doctorInfo != 1 {
		t.Fatalf("logs %d %v", len(artifacts), err)
	}
}

func TestLifecycleCleanupFailureAndDirtySourceQuarantine(t *testing.T) {
	for _, failure := range []string{"dirty", "down", "foreign"} {
		t.Run(failure, func(t *testing.T) {
			s, o, source, runtime, _ := lifecycleFixture(t)
			ctx := context.Background()
			l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
			if err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "dirty":
				state := source.states[l.Sources[0].WorktreePath]
				state.TrackedDirty = true
				source.states[l.Sources[0].WorktreePath] = state
			case "down":
				runtime.failDown = true
			case "foreign":
				runtime.foreign = true
			}
			l, err = s.Destroy(ctx, l.ID, false, false)
			if err == nil || l.Observed != "quarantined" {
				t.Fatalf("quarantine %+v %v", l, err)
			}
			if len(source.states) != 1 || len(runtime.projects) != 2 {
				t.Fatal("unsafe cleanup proceeded")
			}
			if failure == "dirty" {
				l, err = s.Destroy(ctx, l.ID, true, false)
				if err != nil || l.Observed != "released" {
					t.Fatalf("force %+v %v", l, err)
				}
				artifacts, _ := s.Store.Artifacts(ctx, l.ID)
				found := false
				for _, a := range artifacts {
					found = found || a.Kind == "tracked-diff"
				}
				if !found {
					t.Fatal("forced cleanup missing diff")
				}
			}
		})
	}
}

func TestLifecycleReadinessTimeoutRollsBack(t *testing.T) {
	s, o, source, runtime, _ := lifecycleFixture(t)
	s.ReadinessTimeout = 50 * time.Millisecond
	runtime.notReady = true
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if err == nil || l.Observed != "released" || len(runtime.projects) != 0 || len(source.states) != 0 {
		t.Fatalf("timeout saga %+v %v", l, err)
	}
}

func TestLifecyclePartialSourceFailureCompensates(t *testing.T) {
	s, o, source, runtime, _ := lifecycleFixture(t)
	source.failAlias = "self"
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if err == nil || l.Observed != "released" || len(source.states) != 0 || len(runtime.projects) != 0 {
		t.Fatalf("source partial rollback %+v %v", l, err)
	}
}

func TestLifecycleReconcileMissingAndReleasedLeftovers(t *testing.T) {
	s, o, _, runtime, _ := lifecycleFixture(t)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	delete(runtime.projects, l.Runtimes[0].Project)
	l, err = s.Reconcile(ctx, l.ID)
	if err != nil || l.Observed != "degraded" {
		t.Fatalf("missing project %+v %v", l, err)
	}
	l, err = s.Destroy(ctx, l.ID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	runtime.projects[l.Runtimes[0].Project] = true
	l, err = s.Reconcile(ctx, l.ID)
	if err != nil || l.Observed != "quarantined" {
		t.Fatalf("released leftovers %+v %v", l, err)
	}
	l, err = s.Destroy(ctx, l.ID, false, false)
	if err != nil || l.Observed != "released" || len(runtime.projects) != 0 {
		t.Fatalf("orphan recovery %+v %v", l, err)
	}
}

func TestLifecycleGCPreviewRenewAndQuarantine(t *testing.T) {
	s, o, _, runtime, _ := lifecycleFixture(t)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	l.ExpiresAt = time.Now().Add(-time.Hour)
	l.HeartbeatAt = time.Now().Add(-2 * time.Minute)
	if err := s.Store.Save(ctx, l); err != nil {
		t.Fatal(err)
	}
	preview, err := s.GC(ctx, false)
	if err != nil || len(preview) != 1 || len(runtime.projects) != 2 {
		t.Fatalf("preview %+v %v", preview, err)
	}
	l, err = s.Renew(ctx, l.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	preview, err = s.GC(ctx, true)
	if err != nil || len(preview) != 0 || len(runtime.projects) != 2 {
		t.Fatalf("renewed collected %+v %v", preview, err)
	}
	l.ExpiresAt = time.Now().Add(-time.Hour)
	l.Observed = "quarantined"
	if err := s.Store.Save(ctx, l); err != nil {
		t.Fatal(err)
	}
	preview, err = s.GC(ctx, true)
	if err != nil || len(preview) != 0 || len(runtime.projects) != 2 {
		t.Fatalf("quarantine collected %+v %v", preview, err)
	}
}

func TestLifecycleRejectsSecretSnapshots(t *testing.T) {
	r := domain.Runtime{Project: "project", Name: "runtime", Services: []string{"api"}}
	if _, err := ownedConfiguration([]byte(`{"services":{"api":{"environment":{"POSTGRES_PASSWORD_FILE":"/run/secrets/password"}}}}`), r, "lease"); err != nil {
		t.Fatalf("secret file reference rejected: %v", err)
	}
	if _, err := ownedConfiguration([]byte(`{"services":{"api":{"environment":{"API_TOKEN":"credential-value"}}}}`), r, "lease"); err == nil || strings.Contains(err.Error(), "credential-value") {
		t.Fatalf("credential config accepted or disclosed: %v", err)
	}
	s, o, _, _, _ := lifecycleFixture(t)
	path := filepath.Join(o.Repository, ".agent-env.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, []byte("tests:\n  test:\n    stack: review\n    source: self\n    command: [test]\n    env: {API_TOKEN: credential-value}\n")...)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"}); err == nil || strings.Contains(err.Error(), "credential-value") {
		t.Fatalf("credential manifest accepted/disclosed %v", err)
	}
	leases, _ := s.Store.List(context.Background())
	if len(leases) != 0 {
		t.Fatal("secret rejection persisted a lease")
	}
}
