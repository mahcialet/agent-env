package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestCancellationRequestsCrossConnectionsWithoutBypassingFence(t *testing.T) {
	ctx := context.Background()
	db, path := database(t)
	l := lease("cancel-test")
	if err := db.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	operation, release, err := db.AcquireContext(ctx, l.ID, "test-owner", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	run := domain.CommandRun{ID: "exact-run-one", LeaseID: l.ID, Status: "running", StartedAt: time.Now()}
	if err := db.SaveRun(operation, run); err != nil {
		t.Fatal(err)
	}
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := other.Save(ctx, l); !errors.Is(err, ErrBusy) {
		t.Fatalf("state fence bypassed: %v", err)
	}
	if err := other.RequestRunCancel(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if err := other.RequestRunCancel(ctx, run.ID); err != nil {
		t.Fatalf("repeat request: %v", err)
	}
	requested, err := db.RunCancellationRequested(operation, run.ID)
	if err != nil || !requested {
		t.Fatalf("request not visible: %t %v", requested, err)
	}
	var count int
	if err := db.db.QueryRow("SELECT count(*) FROM command_run_cancellations WHERE run_id=?", run.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate requests %d %v", count, err)
	}
	run.Status = "canceled"
	run.FinishedAt = time.Now()
	if err := db.SaveRun(operation, run); err != nil {
		t.Fatal(err)
	}
	next := run
	next.ID = "exact-run-two"
	next.Status = "running"
	next.FinishedAt = time.Time{}
	if err := db.SaveRun(operation, next); err != nil {
		t.Fatal(err)
	}
	if requested, err := db.RunCancellationRequested(operation, next.ID); err != nil || requested {
		t.Fatalf("old cancellation affected next run: %t %v", requested, err)
	}
	got, err := db.Get(ctx, l.ID)
	if err != nil || got.Observed != l.Observed || got.Desired != l.Desired {
		t.Fatalf("request altered lease state: %+v %v", got, err)
	}
}

func TestCancellationDoesNotCreateRequestsForMissingOrCompletedRuns(t *testing.T) {
	ctx := context.Background()
	db, _ := database(t)
	l := lease("finished-test")
	if err := db.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.RequestRunCancel(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing run: %v", err)
	}
	if err := db.RequestRunCancel(ctx, ""); err == nil {
		t.Fatal("empty ID accepted")
	}
	run := domain.CommandRun{ID: "completed", LeaseID: l.ID, Status: "passed", StartedAt: time.Now(), FinishedAt: time.Now()}
	if err := db.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := db.RequestRunCancel(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if requested, err := db.RunCancellationRequested(ctx, run.ID); err != nil || requested {
		t.Fatalf("completed request created: %t %v", requested, err)
	}
}
