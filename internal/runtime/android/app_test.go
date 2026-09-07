package android

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

type prerequisiteRunner struct {
	base    execx.Runner
	failArg string
	failed  bool
}

func (r *prerequisiteRunner) Run(ctx context.Context, cmd execx.Command) (execx.Result, error) {
	if len(cmd.Args) == 1 && cmd.Args[0] == r.failArg {
		r.failed = true
		if r.failArg == "-accel-check" {
			return execx.Result{}, errors.New("host acceleration unavailable")
		}
		// The fixture SDK tools are regular files, but not runnable binaries.
		// Exercise the real native execution boundary for both tools on every OS.
		return (execx.OSRunner{}).Run(ctx, cmd)
	}
	return r.base.Run(ctx, cmd)
}

func TestSDKPrerequisitesFailBeforeAppAllocation(t *testing.T) {
	for _, arg := range []string{"-version", "version", "-accel-check"} {
		t.Run(arg, func(t *testing.T) {
			a, _, p := fixture(t)
			runner := &prerequisiteRunner{base: a.Runner, failArg: arg}
			a.Runner = runner
			s, options, db := serviceFixture(t, t.TempDir(), "Pixel", a)
			defer db.Close()
			// Planning remains SDK-independent even with an unusable SDK.
			if _, err := app.BuildPlan(context.Background(), options, s.Source); err != nil || runner.failed {
				t.Fatalf("planning used SDK prerequisite: %v", err)
			}
			_, err := s.Create(context.Background(), options, app.CreateOptions{Owner: "prerequisite-fixture"})
			if !errors.Is(err, app.ErrPrerequisite) || !runner.failed {
				t.Fatalf("wanted runnable prerequisite failure, got %v (checked=%t)", err, runner.failed)
			}
			leases, err := db.List(context.Background())
			if err != nil || len(leases) != 0 {
				t.Fatalf("prerequisite failure reserved leases: %+v %v", leases, err)
			}
			if len(s.Source.(*appSource).live) != 0 || p.command.Name != "" {
				t.Fatal("prerequisite failure materialized sources or started a detached process")
			}
			if _, err := os.Stat(filepath.Join(s.Home, "leases")); !os.IsNotExist(err) {
				t.Fatalf("prerequisite failure created lease artifacts: %v", err)
			}
		})
	}
}

// Source is synthetic here so native Android adapter tests need neither Git nor
// Docker. App orchestration, runtime adapter and durable SQLite are real.
type appSource struct {
	mu   sync.Mutex
	live map[string]bool
}

func (s *appSource) Resolve(_ context.Context, repo, ref string) (domain.Source, error) {
	return domain.Source{RepositoryID: repo, RepositoryPath: repo, RequestedRef: ref, Commit: strings.Repeat("a", 40), ResolvedAt: time.Now()}, nil
}
func (s *appSource) Materialize(_ context.Context, source domain.Source) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(source.WorktreePath, 0700); err != nil {
		return err
	}
	s.live[source.WorktreePath] = true
	return nil
}
func (s *appSource) Inspect(_ context.Context, source domain.Source) (app.SourceObservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return app.SourceObservation{Exists: s.live[source.WorktreePath], Registered: s.live[source.WorktreePath], Commit: source.Commit}, nil
}
func (s *appSource) Remove(_ context.Context, source domain.Source, _ bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.live, source.WorktreePath)
	return os.RemoveAll(source.WorktreePath)
}
func (s *appSource) Diff(context.Context, domain.Source) (string, error) { return "", nil }

func serviceFixture(t *testing.T, root, template string, adapter Adapter) (*app.Service, app.PlanOptions, *sqlite.Store) {
	t.Helper()
	repo := filepath.Join(root, "source 日本語 spaces")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("version: 1\nsources:\n  self: {repository: ., default_ref: main}\nruntimes:\n  phone: {type: android-emulator, source: self, avd: %q}\ncomponents:\n  device: {runtime: phone}\nstacks:\n  android-runtime: {roots: [device]}\n", template)
	writeFixture(t, filepath.Join(repo, ".agent-env.yaml"), manifest)
	home := filepath.Join(root, "state")
	db, err := sqlite.Open(filepath.Join(home, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	s := &app.Service{Home: home, Store: db, Source: &appSource{live: map[string]bool{}}, Android: adapter, Policy: policy.Defaults(), ReadinessTimeout: time.Second, ReadinessInterval: time.Millisecond}
	return s, app.PlanOptions{Repository: repo, Stack: "android-runtime"}, db
}

func TestRealAdapterThroughAppPersistsOwnedResource(t *testing.T) {
	a, _, p := fixture(t)
	s, options, db := serviceFixture(t, t.TempDir(), "Pixel", a)
	defer db.Close()
	// Native unit tests must coexist with developer-owned emulators. Reserve a
	// synthetic fixture prefix in this isolated registry, without observing,
	// stopping or claiming ownership of those external processes.
	for port := 5554; port <= 5682; port += 2 {
		if portAvailable(port) && portAvailable(port+1) {
			break
		}
		id := fmt.Sprintf("fixture-slot-%d", port)
		home := filepath.Join(s.Home, "fixture-reservations", id, "avd")
		reserved := domain.Lease{ID: id, Observed: "quarantined", Desired: "ready", CreatedAt: time.Now(), HeartbeatAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Runtimes: []domain.Runtime{{LeaseID: id, Name: "fixture", Type: "android-emulator", Android: &domain.AndroidEmulator{Template: "Pixel", AVDName: id, AVDHome: home, AVDPath: filepath.Join(home, id+".avd")}}}}
		if err := db.Reserve(context.Background(), reserved, 0); err != nil {
			t.Fatal(err)
		}
	}
	s.Policy.MaxActive = 65
	lease, err := s.Create(context.Background(), options, app.CreateOptions{Owner: "adapter-fixture"})
	if err != nil {
		t.Fatalf("real app rejected adapter observation: %+v %v", lease, err)
	}
	if lease.Observed != "ready" || len(lease.Resources) != 1 {
		t.Fatalf("lease not ready with owned resource: %+v", lease)
	}
	resource := lease.Resources[0]
	if resource.Metadata["lease"] != lease.ID || resource.ID != lease.ID+":phone:emulator" {
		t.Fatalf("resource ownership mapping: %+v", resource)
	}
	stored, err := db.Get(context.Background(), lease.ID)
	if err != nil || len(stored.Resources) != 1 || stored.Runtimes[0].Android.ProcessID == 0 {
		t.Fatalf("durable adapter identity: %+v %v", stored, err)
	}
	if released, err := s.Destroy(context.Background(), lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("real app cleanup: %+v %v", released, err)
	}
	if p.alive.Load() {
		t.Fatal("app cleanup retained emulator")
	}
}
