package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestGCProtectsExpiryGraceAndRecentHeartbeat(t *testing.T) {
	for _, scenario := range []string{"recent-expiry", "recent-heartbeat", "running-record"} {
		t.Run(scenario, func(t *testing.T) {
			s, options, source, runtime, _ := lifecycleFixture(t)
			ctx := context.Background()
			lease, err := s.Create(ctx, options, CreateOptions{Owner: "tester"})
			if err != nil {
				t.Fatal(err)
			}
			lease.ExpiresAt = time.Now().Add(-10 * time.Minute)
			lease.HeartbeatAt = time.Now().Add(-2 * time.Minute)
			switch scenario {
			case "recent-expiry":
				lease.ExpiresAt = time.Now().Add(-time.Minute)
			case "recent-heartbeat":
				lease.HeartbeatAt = time.Now()
			case "running-record":
				if err := s.Store.SaveRun(ctx, domain.CommandRun{ID: "unfinished", LeaseID: lease.ID, Status: "running", StartedAt: time.Now().Add(-time.Hour)}); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Store.Save(ctx, lease); err != nil {
				t.Fatal(err)
			}
			for _, apply := range []bool{false, true} {
				candidates, err := s.GC(ctx, apply)
				if err != nil || len(candidates) != 0 {
					t.Fatalf("protected lease offered to GC apply=%v: %+v %v", apply, candidates, err)
				}
			}
			if len(source.states) != 1 || len(runtime.projects) != 2 {
				t.Fatal("protected resources were cleaned up")
			}
		})
	}
}

type gcRunAppearsStore struct {
	Store
	id string
}

func (s gcRunAppearsStore) AcquireContext(ctx context.Context, id, token string, ttl time.Duration) (context.Context, func() error, error) {
	op, release, err := s.Store.AcquireContext(ctx, id, token, ttl)
	if err != nil {
		return nil, nil, err
	}
	if err := s.Store.SaveRun(op, domain.CommandRun{ID: s.id, LeaseID: id, Status: "running", StartedAt: time.Now()}); err != nil {
		_ = release()
		return nil, nil, err
	}
	return op, release, nil
}

func TestGCRechecksRunningRecordsAfterAcquiringLock(t *testing.T) {
	s, options, source, runtime, _ := lifecycleFixture(t)
	ctx := context.Background()
	lease, err := s.Create(ctx, options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	lease.ExpiresAt = time.Now().Add(-10 * time.Minute)
	lease.HeartbeatAt = time.Now().Add(-2 * time.Minute)
	if err := s.Store.Save(ctx, lease); err != nil {
		t.Fatal(err)
	}
	s.Store = gcRunAppearsStore{Store: s.Store, id: "raced-command"}
	if _, err := s.GC(ctx, true); err != nil {
		t.Fatal(err)
	}
	if len(source.states) != 1 || len(runtime.projects) != 2 {
		t.Fatal("GC ignored a command record created between preview and lock")
	}
	saved, err := s.Store.Get(ctx, lease.ID)
	if err != nil || saved.Observed == "released" {
		t.Fatalf("protected lease released: %+v %v", saved, err)
	}
}

func TestReconcileReportsExpiryAndUnfinishedCommandWithoutClaimingStopped(t *testing.T) {
	s, options, _, _, _ := lifecycleFixture(t)
	ctx := context.Background()
	lease, err := s.Create(ctx, options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	lease.ExpiresAt = time.Now().Add(-10 * time.Minute)
	lease.HeartbeatAt = time.Now().Add(-2 * time.Minute)
	if err := s.Store.Save(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveRun(ctx, domain.CommandRun{ID: "unfinished", LeaseID: lease.ID, Status: "running", StartedAt: time.Now().Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	observed, err := s.Reconcile(ctx, lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := strings.Join(observed.Diagnostics, "\n")
	if !strings.Contains(diagnostics, "lease expired at") || !strings.Contains(diagnostics, "stale command run unfinished") || !strings.Contains(diagnostics, "completion is unverified") {
		t.Fatalf("missing expiry/unfinished observation: %s", diagnostics)
	}
	runs, err := s.Store.Runs(ctx, lease.ID)
	if err != nil || len(runs) != 1 || runs[0].Status != "running" {
		t.Fatalf("reconcile claimed command completion: %+v %v", runs, err)
	}
}

func TestGCUsesHostPolicyGrace(t *testing.T) {
	s, options, _, _, _ := lifecycleFixture(t)
	ctx := context.Background()
	lease, err := s.Create(ctx, options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	lease.ExpiresAt = time.Now().Add(-2 * time.Minute)
	lease.HeartbeatAt = time.Now().Add(-30 * time.Second)
	if err := s.Store.Save(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if candidates, err := s.GC(ctx, false); err != nil || len(candidates) != 0 {
		t.Fatalf("defaults not applied: %+v %v", candidates, err)
	}
	s.Policy.GCGrace = time.Minute
	s.Policy.HeartbeatGrace = 10 * time.Second
	if candidates, err := s.GC(ctx, false); err != nil || len(candidates) != 1 {
		t.Fatalf("configured grace not applied: %+v %v", candidates, err)
	}
}
