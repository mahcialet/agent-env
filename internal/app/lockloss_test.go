package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

// This wrapper uses the real SQLite capability and renewal loop. A separate
// connection replaces its token after a simulated external effect completes.
type lockLossStore struct {
	Store
	raw     *sql.DB
	op      context.Context
	leaseID string
}

func (s *lockLossStore) AcquireContext(ctx context.Context, id, token string, _ time.Duration) (context.Context, func() error, error) {
	op, release, err := s.Store.AcquireContext(ctx, id, token, time.Second)
	if err == nil {
		s.op, s.leaseID = op, id
	}
	return op, release, err
}

func (s *lockLossStore) replaceOwner() error {
	if _, err := s.raw.Exec("UPDATE operation_locks SET token=? WHERE lease_id=?", "replacement-owner", s.leaseID); err != nil {
		return err
	}
	select {
	case <-s.op.Done():
	case <-time.After(2 * time.Second):
		return errors.New("real renewal loop did not cancel lost operation")
	}
	if !operationLost(s.op) {
		return fmt.Errorf("replacement token did not report ownership loss: %v", context.Cause(s.op))
	}
	return domain.ErrLockLost
}

func lossStore(t *testing.T, s *Service) *lockLossStore {
	t.Helper()
	raw, err := sql.Open("sqlite", filepath.Join(s.Home, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	wrapped := &lockLossStore{Store: s.Store, raw: raw}
	s.Store = wrapped
	return wrapped
}

type lockLossRuntime struct {
	RuntimeProvider
	store *lockLossStore
}

func (r lockLossRuntime) Up(ctx context.Context, v domain.Runtime) error {
	if err := r.RuntimeProvider.Up(ctx, v); err != nil {
		return err
	}
	return r.store.replaceOwner()
}

func TestCreateLockLossDoesNotCompensateOrOverwriteNewOwner(t *testing.T) {
	s, options, source, runtime, operations := lifecycleFixture(t)
	store := lossStore(t, s)
	s.Runtime = lockLossRuntime{RuntimeProvider: runtime, store: store}
	l, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("loss not returned: %+v %v", l, err)
	}
	if len(source.states) != 1 || len(runtime.projects) != 1 {
		t.Fatalf("old owner compensated after loss: sources=%d projects=%d operations=%v", len(source.states), len(runtime.projects), *operations)
	}
	for _, op := range *operations {
		if strings.HasPrefix(op, "down:") || strings.HasPrefix(op, "remove:") {
			t.Fatalf("cleanup ran under lost ownership: %v", *operations)
		}
	}
	saved, err := store.Get(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Observed != "starting" || !saved.Runtimes[0].Started {
		t.Fatalf("durable in-progress intent overwritten: %+v", saved)
	}
	events, err := store.Events(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "allocation_failed" || event.Type == "lease_released" || event.Type == "lease_quarantined" {
			t.Fatalf("old owner wrote final event: %+v", event)
		}
	}
	assertStaleAppWritesFail(t, store, saved)
}

type lockLossRunner struct{ store *lockLossStore }

func (r lockLossRunner) Run(_ context.Context, command execx.Command) (execx.Result, error) {
	if command.Stdout != nil {
		_, _ = command.Stdout.Write([]byte("output before ownership loss\n"))
	}
	err := r.store.replaceOwner()
	return execx.Result{ExitCode: -1}, err
}

func TestNamedTestLockLossRetainsLocalRunWithoutFinalRegistryWrites(t *testing.T) {
	s, db, lease := commandFixture(t)
	store := lossStore(t, s)
	s.Runner = lockLossRunner{store: store}
	run, err := s.Test(context.Background(), lease.ID, "check")
	if !errors.Is(err, domain.ErrLockLost) || run.ID == "" {
		t.Fatalf("loss result %+v %v", run, err)
	}
	local := filepath.Join(filepath.Dir(run.StdoutPath), "run.json")
	data, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	var descriptor domain.CommandRun
	if err := json.Unmarshal(data, &descriptor); err != nil {
		t.Fatal(err)
	}
	if descriptor.ID != run.ID || descriptor.FinishedAt.IsZero() || descriptor.Status == "running" {
		t.Fatalf("unfinished local evidence: %+v", descriptor)
	}
	logs, err := os.ReadFile(run.StdoutPath)
	if err != nil || !strings.Contains(string(logs), "output before ownership loss") {
		t.Fatalf("missing preserved output: %s %v", logs, err)
	}
	runs, err := db.Runs(context.Background(), lease.ID)
	if err != nil || len(runs) != 1 || runs[0].Status != "running" {
		t.Fatalf("old owner finalized registry run: %+v %v", runs, err)
	}
	artifacts, err := db.Artifacts(context.Background(), lease.ID)
	if err != nil || len(artifacts) != 0 {
		t.Fatalf("old owner registered artifacts: %+v %v", artifacts, err)
	}
	events, err := db.Events(context.Background(), lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == "test_finished" || event.Type == "test_command_error" {
			t.Fatalf("old owner wrote final test event: %+v", event)
		}
	}
	if source := s.Source.(*commandSource); source.calls != 1 {
		t.Fatalf("source inspection continued after ownership loss: %d", source.calls)
	}
	assertStaleAppWritesFail(t, store, lease)
}

func assertStaleAppWritesFail(t *testing.T, s *lockLossStore, lease domain.Lease) {
	t.Helper()
	ctx := context.WithoutCancel(s.op)
	lease.Observed = "released"
	if err := s.Save(ctx, lease); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("old app Save not fenced: %v", err)
	}
	if err := s.Event(ctx, domain.Event{ID: "stale-event", LeaseID: lease.ID, Time: time.Now(), Type: "stale"}); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("old app Event not fenced: %v", err)
	}
	var token string
	if err := s.raw.QueryRow("SELECT token FROM operation_locks WHERE lease_id=?", lease.ID).Scan(&token); err != nil || token != "replacement-owner" {
		t.Fatalf("old release deleted new owner's token: %s %v", token, err)
	}
}
