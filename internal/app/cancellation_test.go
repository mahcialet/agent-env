package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// The source refuses deletion unless cancellation evidence was finalized first.
type cleanupAfterRunSource struct {
	SourceProvider
	store   Store
	leaseID string
	removed atomic.Bool
}

func (s *cleanupAfterRunSource) Inspect(ctx context.Context, source domain.Source) (SourceObservation, error) {
	if s.removed.Load() {
		return SourceObservation{}, nil
	}
	return s.SourceProvider.Inspect(ctx, source)
}
func (s *cleanupAfterRunSource) Remove(ctx context.Context, source domain.Source, _ bool) error {
	runs, err := s.store.Runs(ctx, s.leaseID)
	if err != nil {
		return err
	}
	if len(runs) != 1 || runs[0].Status != "canceled" || runs[0].FinishedAt.IsZero() {
		return fmt.Errorf("source deletion preceded final cancellation evidence: %+v", runs)
	}
	artifacts, err := s.store.Artifacts(ctx, runs[0].LeaseID)
	if err != nil {
		return err
	}
	if len(artifacts) < 3 {
		return errors.New("source deletion preceded final artifact records")
	}
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact.Path); err != nil {
			return fmt.Errorf("source deletion preceded artifact file: %w", err)
		}
	}
	if err := os.RemoveAll(source.WorktreePath); err != nil {
		return err
	}
	s.removed.Store(true)
	return nil
}

func TestCancellationProcessHelper(t *testing.T) {
	mode := os.Getenv("AGENT_ENV_CANCELLATION_HELPER")
	if mode == "" {
		return
	}
	marker := os.Getenv("AGENT_ENV_CANCELLATION_MARKER")
	if mode == "leaf" {
		if err := os.WriteFile(marker+".ready", nil, 0600); err != nil {
			os.Exit(8)
		}
		time.Sleep(time.Second)
		_ = os.WriteFile(marker, []byte("unsafe surviving descendant"), 0600)
		os.Exit(0)
	}
	exe, _ := os.Executable()
	child := exec.Command(exe, "-test.run=^TestCancellationProcessHelper$")
	child.Env = append(os.Environ(), "AGENT_ENV_CANCELLATION_HELPER=leaf")
	if err := child.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
	_ = child.Process.Release()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker + ".ready"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			os.Exit(10)
		}
		time.Sleep(5 * time.Millisecond)
	}
	fmt.Fprint(os.Stdout, "started "+os.Getenv("AGENT_ENV_COMMAND_TOKEN"))
	fmt.Fprint(os.Stderr, "diagnostic "+os.Getenv("AGENT_ENV_COMMAND_TOKEN"))
	time.Sleep(20 * time.Second)
	os.Exit(0)
}

type readySignalWriter struct {
	ready chan struct{}
	once  sync.Once
}

func (w *readySignalWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.ready) })
	return len(p), nil
}

func TestDestroyCancelsActualCommandTreeBeforeSourceCleanup(t *testing.T) {
	s, db, lease := commandFixture(t)
	source := &cleanupAfterRunSource{SourceProvider: s.Source, store: db, leaseID: lease.ID}
	s.Source = source
	marker := filepath.Join(t.TempDir(), "descendant late write 日本語")
	setCommandSpec(t, db, &lease, func(spec *config.Test) {
		spec.Command = []string{os.Args[0], "-test.run=^TestCancellationProcessHelper$"}
		spec.Timeout = "15s"
		spec.Artifacts = nil
		spec.Env["AGENT_ENV_CANCELLATION_HELPER"] = "parent"
		spec.Env["AGENT_ENV_CANCELLATION_MARKER"] = marker
		spec.Env["GORACE"] = "atexit_sleep_ms=0"
	})
	signal := &readySignalWriter{ready: make(chan struct{})}
	s.Stdout = signal
	type outcome struct {
		run domain.CommandRun
		err error
	}
	done := make(chan outcome, 1)
	go func() { run, err := s.Test(context.Background(), lease.ID, "check"); done <- outcome{run, err} }()
	select {
	case <-signal.ready:
	case <-time.After(8 * time.Second):
		t.Fatal("command/descendant did not start")
	}
	other, err := sqlite.Open(filepath.Join(s.Home, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	destroyService := *s
	destroyService.Store = other
	destroyed, err := destroyService.Destroy(context.Background(), lease.ID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if destroyed.Observed != "released" || !source.removed.Load() {
		t.Fatalf("cleanup %+v", destroyed)
	}
	result := <-done
	if !errors.Is(result.err, ErrTestFailed) || result.run.Status != "canceled" {
		t.Fatalf("command result %+v: %v", result.run, result.err)
	}
	if requested, err := db.RunCancellationRequested(context.Background(), result.run.ID); err != nil || !requested {
		t.Fatalf("exact run cancellation missing: %t %v", requested, err)
	}
	for _, path := range []string{result.run.StdoutPath, result.run.StderrPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), os.Getenv("AGENT_ENV_COMMAND_TOKEN")) {
			t.Fatal("canceled evidence leaked a secret")
		}
	}
	select {
	case <-time.After(1200 * time.Millisecond):
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("descendant survived cleanup: %v", err)
	}
}

func TestDestroyRefusesStaleRunningEvidenceEvenForce(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			s, db, lease := commandFixture(t)
			if err := db.SaveRun(context.Background(), domain.CommandRun{ID: "stale-run", LeaseID: lease.ID, Status: "running", StartedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Destroy(context.Background(), lease.ID, force, false); err == nil || !strings.Contains(err.Error(), "inspect the prior process") {
				t.Fatalf("stale run permitted cleanup: %v", err)
			}
			if _, err := os.Stat(lease.Sources[0].WorktreePath); err != nil {
				t.Fatal("source lost", err)
			}
			got, err := db.Get(context.Background(), lease.ID)
			if err != nil || got.Observed != "ready" {
				t.Fatalf("stale run changed lease state: %+v %v", got, err)
			}
		})
	}
}

func TestCancellationBeforeInvocationDoesNotStartRunner(t *testing.T) {
	s, db, lease := commandFixture(t)
	run := domain.CommandRun{ID: "not-started", LeaseID: lease.ID, Status: "running", StartedAt: time.Now()}
	if err := db.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	if err := db.RequestRunCancel(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}
	s.Runner = neverCancellationRunner{t}
	if _, err := s.runWithCancellation(context.Background(), run.ID, execx.Command{Name: "must-not-start"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("prestart cancellation: %v", err)
	}
}

type neverCancellationRunner struct{ t *testing.T }

func (r neverCancellationRunner) Run(context.Context, execx.Command) (execx.Result, error) {
	r.t.Error("canceled command started")
	return execx.Result{}, nil
}

// Failed evidence or ambiguous process cleanup must retain the durable running
// barrier even when the command acknowledged cancellation.
func TestDestroyRetainsSourceWhenCancellationFinalizationFails(t *testing.T) {
	for _, failure := range []string{"artifact-index", "process-tree", "output"} {
		t.Run(failure, func(t *testing.T) {
			s, db, lease := commandFixture(t)
			report := filepath.Join(lease.Sources[0].WorktreePath, "retained-report.txt")
			if err := os.WriteFile(report, []byte("only original report"), 0600); err != nil {
				t.Fatal(err)
			}
			setCommandSpec(t, db, &lease, func(spec *config.Test) { spec.Artifacts = []string{"retained-report.txt"} })
			var stopErr error
			switch failure {
			case "artifact-index":
				s.Store = rejectOutputArtifactStore{Store: db}
			case "process-tree":
				stopErr = execx.ErrProcessTreeUnconfirmed
			case "output":
				stopErr = execx.ErrOutputIncomplete
			}
			runner := &incompleteCancellationRunner{entered: make(chan struct{}), stopErr: stopErr}
			s.Runner = runner
			done := make(chan error, 1)
			go func() { _, err := s.Test(context.Background(), lease.ID, "check"); done <- err }()
			select {
			case <-runner.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("command did not start")
			}
			destroyService := *s
			destroyService.Store = db
			if _, err := destroyService.Destroy(context.Background(), lease.ID, true, false); err == nil || !strings.Contains(err.Error(), "inspect the prior process") {
				t.Fatalf("incomplete run permitted cleanup: %v", err)
			}
			if err := <-done; err == nil {
				t.Fatal("finalization failure was lost")
			}
			runs, err := db.Runs(context.Background(), lease.ID)
			if err != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("completion barrier lost: %+v %v", runs, err)
			}
			if data, err := os.ReadFile(report); err != nil || string(data) != "only original report" {
				t.Fatalf("original report lost: %q %v", data, err)
			}
		})
	}
}

type rejectOutputArtifactStore struct{ Store }

func (s rejectOutputArtifactStore) SaveArtifact(ctx context.Context, artifact domain.Artifact) error {
	if artifact.Kind == "test-output" {
		return errors.New("injected artifact index failure")
	}
	return s.Store.SaveArtifact(ctx, artifact)
}

type incompleteCancellationRunner struct {
	entered chan struct{}
	stopErr error
}

func (r *incompleteCancellationRunner) Run(ctx context.Context, _ execx.Command) (execx.Result, error) {
	close(r.entered)
	<-ctx.Done()
	return execx.Result{ExitCode: -1}, errors.Join(ctx.Err(), r.stopErr)
}
