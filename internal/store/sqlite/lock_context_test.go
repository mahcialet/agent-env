package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestOperationContextFencesAllLeaseMutations(t *testing.T) {
	s, _ := database(t)
	ctx := context.Background()
	l := lease("one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	op, release, err := s.AcquireContext(ctx, l.ID, "owner", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	l.Purpose = "owned change"
	if err := s.Save(op, l); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, l); !errors.Is(err, ErrBusy) {
		t.Fatalf("unbound write bypassed lock: %v", err)
	}
	if _, err := s.db.Exec("UPDATE operation_locks SET token=? WHERE lease_id=?", "replacement", l.ID); err != nil {
		t.Fatal(err)
	}
	without := context.WithoutCancel(op)
	for _, tc := range []struct {
		name  string
		write func() error
	}{
		{"save", func() error { changed := l; changed.Purpose = "stale overwrite"; return s.Save(without, changed) }},
		{"event", func() error {
			return s.Event(without, domain.Event{ID: "event", LeaseID: l.ID, Type: "stale", Time: time.Now()})
		}},
		{"run", func() error { return s.SaveRun(without, domain.CommandRun{ID: "run", LeaseID: l.ID, Name: "stale"}) }},
		{"artifact", func() error {
			return s.SaveArtifact(without, domain.Artifact{ID: "artifact", LeaseID: l.ID, Path: "stale"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.write(); !errors.Is(err, domain.ErrLockLost) {
				t.Fatalf("stale mutation not fenced: %v", err)
			}
		})
	}
	if !errors.Is(context.Cause(op), domain.ErrLockLost) || !op.(interface{ LockLost() bool }).LockLost() {
		t.Fatalf("ownership loss not propagated: %v", context.Cause(op))
	}
	got, err := s.Get(ctx, l.ID)
	if err != nil || got.Purpose != "owned change" {
		t.Fatalf("snapshot overwritten: %+v %v", got, err)
	}
	for _, table := range []string{"events", "command_runs", "artifacts"} {
		var count int
		if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s changed: count=%d err=%v", table, count, err)
		}
	}
	if err := release(); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("release lost failure evidence: %v", err)
	}
	var token string
	if err := s.db.QueryRow("SELECT token FROM operation_locks WHERE lease_id=?", l.ID).Scan(&token); err != nil || token != "replacement" {
		t.Fatalf("release removed another owner's token: %s %v", token, err)
	}
}

func TestRenewalFailureCancelsBeforeExpiry(t *testing.T) {
	s, _ := database(t)
	ctx := context.Background()
	l := lease("one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("CREATE TRIGGER reject_lock_renewal BEFORE UPDATE ON operation_locks BEGIN SELECT RAISE(FAIL, 'injected renewal failure'); END"); err != nil {
		t.Fatal(err)
	}
	op, release, err := s.AcquireContext(ctx, l.ID, "owner", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-op.Done():
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("active operation not canceled before lock expiration")
	}
	if !errors.Is(context.Cause(op), domain.ErrLockLost) {
		t.Fatalf("wrong cancellation cause: %v", context.Cause(op))
	}
	if err := s.Save(context.WithoutCancel(op), l); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("WithoutCancel bypassed renewal failure: %v", err)
	}
	if err := release(); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("release failure %v", err)
	}
}

func TestCallerCancellationDoesNotMaskLaterLockLoss(t *testing.T) {
	s, _ := database(t)
	ctx, cancel := context.WithCancel(context.Background())
	l := lease("one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	op, release, err := s.AcquireContext(ctx, l.ID, "owner", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	// Ordinary request cancellation still permits compensation under valid ownership.
	if err := s.Save(context.WithoutCancel(op), l); err != nil {
		t.Fatalf("valid compensation rejected: %v", err)
	}
	if _, err := s.db.Exec("UPDATE operation_locks SET expires_at=? WHERE lease_id=?", time.Now().Add(-time.Second).UnixNano(), l.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(context.WithoutCancel(op), l); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("expired compensation accepted: %v", err)
	}
	if !op.(interface{ LockLost() bool }).LockLost() {
		t.Fatal("prior caller cancellation masked lock loss")
	}
	if !errors.Is(context.Cause(op), context.Canceled) {
		t.Fatalf("unexpected caller cancellation cause %v", context.Cause(op))
	}
	if err := release(); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("release failure %v", err)
	}
}

func TestReleasedCapabilityCannotWrite(t *testing.T) {
	s, _ := database(t)
	ctx := context.Background()
	l := lease("one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	op, release, err := s.AcquireContext(ctx, l.ID, "owner", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(context.WithoutCancel(op), l); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("released capability accepted: %v", err)
	}
}
