package sqlite

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestWindowsDatabaseURI(t *testing.T) {
	u := url.URL{Scheme: "file", Path: uriPath("C:/Users/space 日本語/db.sqlite")}
	parsed, err := url.Parse(u.String())
	if err != nil || parsed.Host != "" || parsed.Path != "/C:/Users/space 日本語/db.sqlite" {
		t.Fatalf("drive became host: %s %v", u.String(), err)
	}
}

func database(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "space 日本語", "registry.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}
func lease(id string) domain.Lease {
	return domain.Lease{ID: id, Owner: "owner", Observed: "requested", Desired: "ready", CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC(), Sources: []domain.Source{{Alias: "self", RepositoryID: "repo", RepositoryPath: "/repo 日本語", RequestedRef: "main", Commit: "abcd", WorktreePath: "/work tree/" + id, CheckoutMode: "detached"}}, Components: []domain.Component{{Name: "api", Runtime: "compose", Services: []string{"api"}}}, Runtimes: []domain.Runtime{{Name: "compose", Type: "compose", Project: "project-" + id}}, Resources: []domain.Resource{{ID: "resource-" + id, LeaseID: id, Runtime: "compose", Kind: "project", ExternalID: "project-" + id}}}
}

func TestPersistenceNormalizedAndMigrations(t *testing.T) {
	ctx := context.Background()
	s, path := database(t)
	want := lease("one")
	if err := s.Reserve(ctx, want, 2); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"repositories", "leases", "lease_sources", "lease_components", "lease_runtimes", "runtime_resources", "schema_migrations"} {
		var n int
		if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s count %d: %v", table, n, err)
		}
	}
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	got, err := other.Get(ctx, want.ID)
	if err != nil || got.Sources[0].Commit != "abcd" || got.Resources[0].ExternalID != want.Resources[0].ExternalID {
		t.Fatalf("got %+v: %v", got, err)
	}
	want.Observed = "ready"
	want.Sources[0].Commit = "efgh"
	if err := other.Save(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get(ctx, want.ID)
	if err != nil || got.Observed != "ready" || got.Sources[0].Commit != "efgh" {
		t.Fatalf("save %+v: %v", got, err)
	}
	if err := s.Save(ctx, lease("missing")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing %v", err)
	}
}

func TestConcurrentCapacityAndProjectReservation(t *testing.T) {
	ctx := context.Background()
	s, path := database(t)
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	var wg sync.WaitGroup
	results := make(chan error, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db := s
			if i%2 == 0 {
				db = other
			}
			results <- db.Reserve(ctx, lease(fmt.Sprint(i)), 3)
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrCapacity) {
			t.Fatal(err)
		}
	}
	if success != 3 {
		t.Fatalf("reserved %d", success)
	}
	list, err := s.List(ctx)
	if err != nil || len(list) != 3 {
		t.Fatalf("list %d %v", len(list), err)
	}
	duplicate := lease("duplicate")
	duplicate.Runtimes[0].Project = list[0].Runtimes[0].Project
	if err := s.Reserve(ctx, duplicate, 0); err == nil {
		t.Fatal("duplicate active project accepted")
	}
	if _, err := s.Get(ctx, "duplicate"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial reservation: %v", err)
	}
	released := list[0]
	released.Observed = "released"
	released.Desired = "released"
	if err := s.Save(ctx, released); err != nil {
		t.Fatal(err)
	}
	if err := s.Reserve(ctx, duplicate, 3); err != nil {
		t.Fatalf("released project not reusable: %v", err)
	}
}

func TestChildFailureRollsBackSnapshot(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	original := lease("one")
	if err := s.Reserve(ctx, original, 0); err != nil {
		t.Fatal(err)
	}
	broken := original
	broken.Observed = "ready"
	broken.Sources = append(broken.Sources, broken.Sources[0])
	if err := s.Save(ctx, broken); err == nil {
		t.Fatal("duplicate child accepted")
	}
	got, err := s.Get(ctx, "one")
	if err != nil || got.Observed != original.Observed || len(got.Sources) != 1 {
		t.Fatalf("partial save: %+v %v", got, err)
	}
	if err := s.Event(ctx, domain.Event{ID: "orphan", LeaseID: "none"}); err == nil {
		t.Fatal("foreign key not enforced")
	}
	s.db.SetMaxIdleConns(0)
	if err := s.Event(ctx, domain.Event{ID: "orphan2", LeaseID: "none"}); err == nil {
		t.Fatal("foreign key not enforced on replacement connection")
	}
}

func TestEvidence(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	if err := s.Reserve(ctx, lease("one"), 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Event(ctx, domain.Event{ID: "event", LeaseID: "one", Time: time.Now(), Type: "allocated", Message: "日本語"}); err != nil {
		t.Fatal(err)
	}
	run := domain.CommandRun{ID: "run", LeaseID: "one", Name: "test", Argv: []string{"echo", "with spaces"}, StartedAt: time.Now(), Status: "running"}
	if err := s.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	run.Status = "succeeded"
	run.ExitCode = 0
	if err := s.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	artifact := domain.Artifact{ID: "artifact", LeaseID: "one", RunID: "run", Path: "output 日本語.log", CreatedAt: time.Now()}
	if err := s.SaveArtifact(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	events, err := s.Events(ctx, "one")
	if err != nil || len(events) != 1 || events[0].Message != "日本語" {
		t.Fatalf("events %+v %v", events, err)
	}
	runs, err := s.Runs(ctx, "one")
	if err != nil || len(runs) != 1 || runs[0].Status != "succeeded" || runs[0].Argv[1] != "with spaces" {
		t.Fatalf("runs %+v %v", runs, err)
	}
	artifacts, err := s.Artifacts(ctx, "one")
	if err != nil || len(artifacts) != 1 || artifacts[0].Path != artifact.Path {
		t.Fatalf("artifacts %+v %v", artifacts, err)
	}
	if err := s.Reserve(ctx, lease("two"), 0); err != nil {
		t.Fatal(err)
	}
	run.LeaseID = "two"
	if err := s.SaveRun(ctx, run); err == nil {
		t.Fatal("cross-lease run overwrite accepted")
	}
	artifact.ID = "foreign-artifact"
	artifact.LeaseID = "two"
	if err := s.SaveArtifact(ctx, artifact); err == nil {
		t.Fatal("cross-lease run artifact accepted")
	}
}

func TestOperationLockAcrossConnectionsAndExpiry(t *testing.T) {
	ctx := context.Background()
	s, path := database(t)
	if err := s.Reserve(ctx, lease("one"), 0); err != nil {
		t.Fatal(err)
	}
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	release, err := s.Acquire(ctx, "one", "owner-one", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Acquire(ctx, "one", "owner-two", time.Second); !errors.Is(err, ErrBusy) {
		t.Fatalf("concurrent lock %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := other.Acquire(ctx, "one", "owner-two", time.Second); !errors.Is(err, ErrBusy) {
		t.Fatalf("renewal lost lock %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("INSERT INTO operation_locks(lease_id,token,expires_at) VALUES(?,?,?)", "one", "crashed", time.Now().Add(-time.Second).UnixNano()); err != nil {
		t.Fatal(err)
	}
	release, err = other.Acquire(ctx, "one", "recovered", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
}
