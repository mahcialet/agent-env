package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

// This fixture deliberately uses real SQLite and observes persisted intent at
// the adapter boundary; no fake reservation implementation can mask collisions.
type androidLifecycleFake struct {
	mu                    sync.Mutex
	store                 Store
	live                  map[string]domain.Runtime
	validationErr         error
	notReady, identityErr bool
	creates, destroys     int
}

func (f *androidLifecycleFake) Doctor(context.Context) (map[string]string, error) {
	return map[string]string{"emulator": "fixture emulator", "adb": "fixture adb"}, nil
}
func (f *androidLifecycleFake) Validate(_ context.Context, template string) (domain.AndroidEmulator, error) {
	return domain.AndroidEmulator{Template: template, SDKPath: "fixture SDK", TemplatePath: "fixture AVD", SystemImage: "system-images/android-35/default/x86_64"}, f.validationErr
}
func (f *androidLifecycleFake) Create(ctx context.Context, runtime domain.Runtime) (domain.Runtime, error) {
	saved, err := f.store.Get(ctx, runtime.LeaseID)
	if err != nil {
		return runtime, err
	}
	intent := false
	for _, r := range saved.Runtimes {
		if r.Name == runtime.Name && r.Started && r.Android != nil && r.Android.AVDPath == runtime.Android.AVDPath && r.Android.ConsolePort == runtime.Android.ConsolePort {
			intent = true
		}
	}
	if !intent {
		return runtime, errors.New("Android launch preceded durable identity and started intent")
	}
	a := *runtime.Android
	if a.AVDPath == "" || a.AVDName == "" || a.Serial != fmt.Sprintf("emulator-%d", a.ConsolePort) || a.ConsolePort%2 != 0 || a.ADBPort != a.ConsolePort+1 {
		return runtime, errors.New("incomplete Android allocation")
	}
	if err := os.MkdirAll(a.AVDPath, 0700); err != nil {
		return runtime, err
	}
	if err := os.WriteFile(filepath.Join(a.AVDPath, "userdata-qemu.img"), []byte(runtime.LeaseID), 0600); err != nil {
		return runtime, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	a.ProcessID = 10000 + f.creates
	a.ProcessStart = fmt.Sprintf("birth-%d", f.creates)
	a.State = "running"
	runtime.Android = &a
	f.live[a.AVDName] = runtime
	return runtime, nil
}
func (f *androidLifecycleFake) Inspect(_ context.Context, runtime domain.Runtime) (RuntimeObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.identityErr {
		return RuntimeObservation{}, domain.ErrResourceIdentity
	}
	_, exists := f.live[runtime.Android.AVDName]
	o := RuntimeObservation{Exists: exists, Ready: exists && !f.notReady}
	if exists {
		o.Resources = []domain.Resource{{ID: runtime.Android.AVDName, Runtime: runtime.Name, Kind: "android-emulator", ExternalID: runtime.Android.Serial, Metadata: map[string]string{"lease": runtime.LeaseID, "runtime": runtime.Name}}}
	}
	return o, nil
}
func (f *androidLifecycleFake) Destroy(_ context.Context, runtime domain.Runtime) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.identityErr {
		return domain.ErrResourceIdentity
	}
	f.destroys++
	delete(f.live, runtime.Android.AVDName)
	return os.RemoveAll(runtime.Android.AVDPath)
}

type androidLifecycleSource struct {
	*lifecycleSource
	mu sync.Mutex
}

func (f *androidLifecycleSource) Materialize(ctx context.Context, s domain.Source) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lifecycleSource.Materialize(ctx, s)
}
func (f *androidLifecycleSource) Inspect(ctx context.Context, s domain.Source) (SourceObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lifecycleSource.Inspect(ctx, s)
}
func (f *androidLifecycleSource) Remove(ctx context.Context, s domain.Source, force bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lifecycleSource.Remove(ctx, s, force)
}

func androidAppFixture(t *testing.T) (*Service, PlanOptions, *androidLifecycleSource, *androidLifecycleFake) {
	t.Helper()
	s, options, source, _, _ := lifecycleFixture(t)
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  phone: {type: android-emulator, source: self, avd: pixel}
components:
  device: {runtime: phone}
stacks:
  review: {roots: [device]}
`
	if err := os.WriteFile(filepath.Join(options.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	android := &androidLifecycleFake{store: s.Store, live: map[string]domain.Runtime{}}
	lockedSource := &androidLifecycleSource{lifecycleSource: source}
	s.Android, s.Source = android, lockedSource
	// Android-only allocations must not accidentally require Docker.
	s.Runtime = nil
	// Identity/concurrency assertions do not test wall-clock boot deadlines.
	// Allow real SQLite persistence under race instrumentation and host load.
	s.ReadinessTimeout = 5 * time.Second
	return s, options, lockedSource, android
}

func TestAndroidPrerequisitesFailBeforeAllocation(t *testing.T) {
	for _, prerequisite := range []string{"SDK unavailable", "AVD template pixel unavailable"} {
		t.Run(prerequisite, func(t *testing.T) {
			s, options, source, android := androidAppFixture(t)
			android.validationErr = errors.New(prerequisite)
			lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
			if !errors.Is(err, ErrPrerequisite) {
				t.Fatalf("prerequisite error=%v lease=%+v", err, lease)
			}
			leases, err := s.Store.List(context.Background())
			if err != nil || len(leases) != 0 || len(source.states) != 0 || android.creates != 0 {
				t.Fatalf("prerequisite allocated resources: leases=%d source=%d starts=%d err=%v", len(leases), len(source.states), android.creates, err)
			}
		})
	}
}

func TestAndroidConcurrentLeasesAndSiblingCleanup(t *testing.T) {
	s, options, source, android := androidAppFixture(t)
	type result struct {
		lease domain.Lease
		err   error
	}
	done := make(chan result, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			l, e := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
			done <- result{l, e}
		}()
	}
	close(start)
	one, two := <-done, <-done
	if one.err != nil || two.err != nil {
		t.Fatalf("concurrent allocation: %v %v", one.err, two.err)
	}
	a, b := one.lease.Runtimes[0].Android, two.lease.Runtimes[0].Android
	if one.lease.Observed != "ready" || two.lease.Observed != "ready" || a.AVDPath == b.AVDPath || a.AVDHome == b.AVDHome || a.AVDName == b.AVDName || a.ConsolePort == b.ConsolePort || a.ADBPort == b.ADBPort || a.Serial == b.Serial {
		t.Fatalf("colliding/incomplete leases: %+v %+v", one.lease, two.lease)
	}
	for _, l := range []domain.Lease{one.lease, two.lease} {
		stored, err := s.Store.Get(context.Background(), l.ID)
		if err != nil || stored.Observed != "ready" || stored.Runtimes[0].Android.ProcessStart == "" || stored.Runtimes[0].Android.ProcessID == 0 || len(stored.Resources) != 1 {
			t.Fatalf("durable Android identity: %+v %v", stored, err)
		}
	}
	if released, err := s.Destroy(context.Background(), one.lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("destroy: %+v %v", released, err)
	}
	if released, err := s.Destroy(context.Background(), one.lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("repeat destroy: %+v %v", released, err)
	}
	if sibling, err := s.Reconcile(context.Background(), two.lease.ID); err != nil || sibling.Observed != "ready" {
		t.Fatalf("sibling disrupted: %+v %v", sibling, err)
	}
	if len(android.live) != 1 || len(source.states) != 1 {
		t.Fatal("destroy crossed ownership boundary")
	}
	data, err := os.ReadFile(filepath.Join(b.AVDPath, "userdata-qemu.img"))
	if err != nil || string(data) != two.lease.ID {
		t.Fatalf("sibling userdata altered: %q %v", data, err)
	}
}

func TestAndroidMissingProcessReconcilesDegraded(t *testing.T) {
	s, options, _, android := androidAppFixture(t)
	l, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	delete(android.live, l.Runtimes[0].Android.AVDName)
	l, err = s.Reconcile(context.Background(), l.ID)
	if err != nil || l.Observed != "degraded" {
		t.Fatalf("missing emulator reported healthy: %+v %v", l, err)
	}
}

func TestAndroidBootTimeoutCompensates(t *testing.T) {
	s, options, source, android := androidAppFixture(t)
	s.ReadinessTimeout = 50 * time.Millisecond
	android.notReady = true
	l, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err == nil || l.Observed != "released" || len(android.live) != 0 || len(source.states) != 0 || android.destroys == 0 {
		t.Fatalf("timeout failed compensation: %+v %v", l, err)
	}
	events, err := s.Store.Events(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		found = found || event.Type == "allocation_failed"
	}
	if !found {
		t.Fatal("boot timeout failure evidence absent")
	}
}

func TestAndroidUncertainIdentityQuarantinesEvenForce(t *testing.T) {
	s, options, source, android := androidAppFixture(t)
	l, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	a := *l.Runtimes[0].Android
	android.identityErr = true
	for _, force := range []bool{false, true} {
		l, err = s.Destroy(context.Background(), l.ID, force, false)
		if err == nil || l.Observed != "quarantined" || len(source.states) != 1 || len(android.live) != 1 || android.destroys != 0 {
			t.Fatalf("force=%v unsafe cleanup: %+v %v", force, l, err)
		}
	}
	if _, err := os.Stat(filepath.Join(a.AVDPath, "userdata-qemu.img")); err != nil {
		t.Fatalf("quarantine lost writable state: %v", err)
	}
	android.identityErr = false
	next, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	b := next.Runtimes[0].Android
	if a.ConsolePort == b.ConsolePort || a.ADBPort == b.ADBPort || a.Serial == b.Serial || a.AVDPath == b.AVDPath {
		t.Fatal("quarantined resources reassigned")
	}
}
