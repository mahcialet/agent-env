package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type siblingCleanupAndroid struct {
	*androidLifecycleFake
	operations     *[]string
	failures       map[string]error
	onDestroy      func(context.Context, domain.Runtime) error
	inspectFailure bool
	destroyed      map[string]bool
}

func (a *siblingCleanupAndroid) Destroy(ctx context.Context, r domain.Runtime) error {
	*a.operations = append(*a.operations, "down:"+r.Name)
	if a.onDestroy != nil {
		if err := a.onDestroy(ctx, r); err != nil {
			return err
		}
	}
	if err := a.failures[r.Name]; err != nil && !a.inspectFailure {
		return err
	}
	if err := a.androidLifecycleFake.Destroy(ctx, r); err != nil {
		return err
	}
	a.destroyed[r.Name] = true
	return nil
}

func (a *siblingCleanupAndroid) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if a.inspectFailure && a.destroyed[r.Name] && a.failures[r.Name] != nil {
		return RuntimeObservation{}, a.failures[r.Name]
	}
	return a.androidLifecycleFake.Inspect(ctx, r)
}

func siblingCleanupFixture(t *testing.T) (*Service, domain.Lease, *lifecycleSource, *lifecycleRuntime, *siblingCleanupAndroid) {
	t.Helper()
	s, options, source, compose, operations := lifecycleFixture(t)
	s.ReadinessTimeout = 5 * time.Second
	android := &siblingCleanupAndroid{androidLifecycleFake: &androidLifecycleFake{store: s.Store, live: map[string]domain.Runtime{}}, operations: operations, failures: map[string]error{}, destroyed: map[string]bool{}}
	s.Android = android
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  a-compose: {type: compose, source: self, files: [compose.yaml]}
  b-good: {type: android-emulator, source: self, avd: pixel}
  c-failing: {type: android-emulator, source: self, avd: pixel}
  d-failing: {type: android-emulator, source: self, avd: pixel}
components:
  a-compose: {runtime: a-compose, compose_services: [api]}
  b-good: {runtime: b-good}
  c-failing: {runtime: c-failing}
  d-failing: {runtime: d-failing}
stacks:
  review: {roots: [a-compose, b-good, c-failing, d-failing]}
`
	if err := os.WriteFile(filepath.Join(options.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	l, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	*operations = nil
	return s, l, source, compose, android
}

func assertSiblingReservations(t *testing.T, s *Service, id string, want int) {
	t.Helper()
	raw, err := sql.Open("sqlite", filepath.Join(s.Home, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var active int
	if err := raw.QueryRow("SELECT count(*) FROM android_reservations WHERE lease_id=? AND active=1", id).Scan(&active); err != nil || active != want {
		t.Fatalf("active reservations=%d want=%d err=%v", active, want, err)
	}
}

func TestCleanupContinuesIndependentRuntimesAndRetainsSourcesUntilRetry(t *testing.T) {
	for _, postInspect := range []bool{false, true} {
		name := "destroy-failure"
		if postInspect {
			name = "post-destroy-inspection-failure"
		}
		t.Run(name, func(t *testing.T) {
			s, l, source, compose, android := siblingCleanupFixture(t)
			one, two := errors.New("first cleanup uncertainty"), errors.New("second cleanup uncertainty")
			android.failures["c-failing"], android.failures["d-failing"] = one, two
			android.inspectFailure = postInspect
			l, err := s.Destroy(context.Background(), l.ID, false, false)
			if !errors.Is(err, one) || !errors.Is(err, two) || l.Observed != "quarantined" {
				t.Fatalf("lost aggregated cleanup errors: %+v %v", l, err)
			}
			downs := []string{}
			for _, op := range *android.operations {
				if strings.HasPrefix(op, "down:") {
					downs = append(downs, op)
				}
				if strings.HasPrefix(op, "remove:") {
					t.Fatalf("source removed before all runtimes confirmed: %v", *android.operations)
				}
			}
			want := []string{"down:d-failing", "down:c-failing", "down:b-good", "down:a-compose"}
			if !reflect.DeepEqual(downs, want) {
				t.Fatalf("independent reverse cleanup=%v want=%v", downs, want)
			}
			if len(compose.projects) != 0 || len(source.states) != 1 {
				t.Fatalf("independent Compose cleanup/source retention: projects=%d sources=%d", len(compose.projects), len(source.states))
			}
			stored, err := s.Store.Get(context.Background(), l.ID)
			if err != nil || stored.Observed != "quarantined" {
				t.Fatalf("durable quarantine: %+v %v", stored, err)
			}
			for _, r := range stored.Runtimes {
				failed := android.failures[r.Name] != nil
				if r.Started != failed {
					t.Fatalf("runtime completion not durable: %+v", r)
				}
				if r.Name == "b-good" {
					if _, err := os.Stat(r.Android.AVDPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("healthy sibling AVD not removed: %v", err)
					}
				}
			}
			assertSiblingReservations(t, s, l.ID, 3)
			android.failures = map[string]error{}
			l, err = s.Destroy(context.Background(), l.ID, false, false)
			if err != nil || l.Observed != "released" || len(source.states) != 0 || len(android.live) != 0 || len(compose.projects) != 0 {
				t.Fatalf("retry did not finish cleanup: %+v %v", l, err)
			}
			assertSiblingReservations(t, s, l.ID, 0)
			if _, err = s.Destroy(context.Background(), l.ID, false, false); err != nil {
				t.Fatalf("repeated cleanup: %v", err)
			}
		})
	}
}

func TestCleanupStopsIndependentEffectsAfterOperationLockLoss(t *testing.T) {
	s, l, source, compose, android := siblingCleanupFixture(t)
	store := lossStore(t, s)
	android.onDestroy = func(context.Context, domain.Runtime) error { return store.replaceOwner() }
	_, err := s.Destroy(context.Background(), l.ID, false, false)
	if !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("lost operation fencing error: %v", err)
	}
	if !reflect.DeepEqual(*android.operations, []string{"down:d-failing"}) || len(compose.projects) != 1 || len(android.live) != 3 || len(source.states) != 1 {
		t.Fatalf("continued effects after lock loss: %v", *android.operations)
	}
	stored, err := store.Get(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Observed != "releasing" {
		t.Fatalf("lost owner wrote final state: %s", stored.Observed)
	}
	assertStaleAppWritesFail(t, store, stored)
	assertSiblingReservations(t, s, l.ID, 3)
}

func TestCleanupStopsIndependentEffectsAfterCancellation(t *testing.T) {
	s, l, source, compose, android := siblingCleanupFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	android.onDestroy = func(context.Context, domain.Runtime) error { cancel(); return context.Canceled }
	_, err := s.Destroy(ctx, l.ID, false, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
	if !reflect.DeepEqual(*android.operations, []string{"down:d-failing"}) || len(compose.projects) != 1 || len(android.live) != 3 || len(source.states) != 1 {
		t.Fatalf("continued effects after cancellation: %v", *android.operations)
	}
	assertSiblingReservations(t, s, l.ID, 3)
}

func TestCleanupContinuesAfterRuntimeLocalTimeout(t *testing.T) {
	s, l, source, compose, android := siblingCleanupFixture(t)
	android.failures["d-failing"] = context.DeadlineExceeded
	l, err := s.Destroy(context.Background(), l.ID, false, false)
	if !errors.Is(err, context.DeadlineExceeded) || l.Observed != "quarantined" {
		t.Fatalf("timeout evidence lost: %+v %v", l, err)
	}
	if len(android.live) != 1 || len(compose.projects) != 0 || len(source.states) != 1 {
		t.Fatalf("local timeout stranded independent runtimes: Android=%d Compose=%d sources=%d", len(android.live), len(compose.projects), len(source.states))
	}
	assertSiblingReservations(t, s, l.ID, 3)
}
