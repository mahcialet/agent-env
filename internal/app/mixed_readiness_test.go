package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type delayedBootAndroid struct {
	*androidLifecycleFake
	firstInspection time.Time
	delay           time.Duration
}

func (f *delayedBootAndroid) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if f.firstInspection.IsZero() {
		f.firstInspection = time.Now()
	}
	o, err := f.androidLifecycleFake.Inspect(ctx, r)
	o.Ready = o.Ready && time.Since(f.firstInspection) >= f.delay
	return o, err
}

func mixedReadinessFixture(t *testing.T) (*Service, PlanOptions, *lifecycleRuntime, *delayedBootAndroid) {
	t.Helper()
	s, options, _, compose, _ := lifecycleFixture(t)
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  backend: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
  phone: {type: android-emulator, source: self, avd: pixel}
components:
  api:
    runtime: backend
    compose_services: [api]
    readiness: [{type: compose, timeout: 200ms, interval: 5ms}]
  device: {runtime: phone}
stacks:
  review: {roots: [api, device]}
`
	if err := os.WriteFile(filepath.Join(options.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	android := &delayedBootAndroid{androidLifecycleFake: &androidLifecycleFake{store: s.Store, live: map[string]domain.Runtime{}}, delay: 400 * time.Millisecond}
	s.Android = android
	s.ReadinessTimeout = 5 * time.Second
	s.ReadinessInterval = 5 * time.Millisecond
	return s, options, compose, android
}

func TestMixedReadinessSatisfiedComposeDoesNotShortenAndroidBoot(t *testing.T) {
	s, options, _, android := mixedReadinessFixture(t)
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil || lease.Observed != "ready" {
		t.Fatalf("Android boot was truncated by satisfied Compose timeout: state=%s err=%v", lease.Observed, err)
	}
	if time.Since(android.firstInspection) < android.delay {
		t.Fatal("Android was accepted before boot completed")
	}
	if _, err := s.Destroy(context.Background(), lease.ID, false, false); err != nil {
		t.Fatal(err)
	}
}

func TestMixedReadinessOverdueComposeStillFails(t *testing.T) {
	s, options, compose, android := mixedReadinessFixture(t)
	compose.notReady = true
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "backend") || lease.Observed != "released" {
		t.Fatalf("Compose timeout lost in Android budget: state=%s err=%v", lease.Observed, err)
	}
	if len(compose.projects) != 0 || len(android.live) != 0 {
		t.Fatal("mixed readiness failure did not compensate both runtimes")
	}
}

// An SDK boot budget must not leak into a blocked Compose inspection either.
type blockingReadinessCompose struct {
	*lifecycleRuntime
	blocked          bool
	inspectionBudget time.Duration
}

func (f *blockingReadinessCompose) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if !f.blocked {
		f.blocked = true
		deadline, ok := ctx.Deadline()
		if !ok {
			return RuntimeObservation{}, errors.New("missing readiness deadline")
		}
		f.inspectionBudget = time.Until(deadline)
		<-ctx.Done()
		return RuntimeObservation{}, ctx.Err()
	}
	return f.lifecycleRuntime.Inspect(ctx, r)
}

func TestMixedReadinessBlockedComposeUsesItsOwnDeadline(t *testing.T) {
	s, options, compose, _ := mixedReadinessFixture(t)
	blocking := &blockingReadinessCompose{lifecycleRuntime: compose}
	s.Runtime = blocking
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if !errors.Is(err, context.DeadlineExceeded) || lease.Observed != "released" {
		t.Fatalf("blocked Compose inspection: state=%s err=%v", lease.Observed, err)
	}
	if blocking.inspectionBudget <= 0 || blocking.inspectionBudget > 200*time.Millisecond {
		t.Fatalf("Compose received Android boot budget: %s", blocking.inspectionBudget)
	}
}

type blockingReadinessAndroid struct {
	*androidLifecycleFake
	blocked          bool
	inspectionBudget time.Duration
}

func (f *blockingReadinessAndroid) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if !f.blocked {
		f.blocked = true
		deadline, ok := ctx.Deadline()
		if !ok {
			return RuntimeObservation{}, errors.New("missing readiness deadline")
		}
		f.inspectionBudget = time.Until(deadline)
		<-ctx.Done()
		return RuntimeObservation{}, ctx.Err()
	}
	return f.androidLifecycleFake.Inspect(ctx, r)
}

func TestMixedReadinessPendingComposeBoundsEarlierAndroidInspection(t *testing.T) {
	s, options, compose, android := mixedReadinessFixture(t)
	manifestPath := filepath.Join(options.Repository, ".agent-env.yaml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// Runtime order follows the component closure; sort the Android root first.
	if err := os.WriteFile(manifestPath, []byte(strings.ReplaceAll(string(manifest), "device", "adevice")), 0600); err != nil {
		t.Fatal(err)
	}
	compose.notReady = true
	blocking := &blockingReadinessAndroid{androidLifecycleFake: android.androidLifecycleFake}
	s.Android = blocking
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if len(lease.Runtimes) != 2 || lease.Runtimes[0].Name != "phone" {
		t.Fatalf("fixture did not inspect Android first: %+v", lease.Runtimes)
	}
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "backend") || lease.Observed != "released" {
		t.Fatalf("pending Compose was delayed behind Android: state=%s err=%v", lease.Observed, err)
	}
	if blocking.inspectionBudget <= 0 || blocking.inspectionBudget > 200*time.Millisecond {
		t.Fatalf("Android inspection exceeded pending Compose budget: %s", blocking.inspectionBudget)
	}
}
