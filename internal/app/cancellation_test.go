package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// startFixtureOperation owns cancellation and completion before the caller can
// fail an assertion. Register it after the operation's database/temp resources so
// LIFO cleanup joins their last user before those resources are removed.
func startFixtureOperation[T any](t *testing.T, parent context.Context, run func(context.Context) T) <-chan T {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	result := make(chan T, 1)
	finished := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-finished
	})
	go func() {
		defer close(finished)
		result <- run(ctx)
	}()
	return result
}

func TestFixtureOperationJoinsBeforeResourceCleanupOnEarlyExit(t *testing.T) {
	entered, completed := make(chan struct{}), make(chan struct{})
	// Fallback also bounds the intentionally broken-helper control: no worker
	// remains behind if cleanup fails to cancel/join the operation.
	parent, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); <-completed }()
	t.Run("early-exit", func(t *testing.T) {
		t.Cleanup(func() {
			select {
			case <-completed:
			default:
				t.Error("resource cleanup preceded operation completion")
			}
		})
		startFixtureOperation(t, parent, func(ctx context.Context) error {
			close(entered)
			<-ctx.Done()
			close(completed)
			return ctx.Err()
		})
		<-entered
		t.Skip("exercise early test exit after the operation has entered")
	})
}

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
		conn, err := net.DialTimeout("tcp", os.Getenv("AGENT_ENV_CANCELLATION_OBSERVER"), 5*time.Second)
		if err != nil {
			os.Exit(7)
		}
		if _, err := conn.Write([]byte{'R'}); err != nil {
			os.Exit(8)
		}
		// A connected socket must be accepted and observed before the parent
		// may finish or trigger cancellation. Windows discards pending accepts
		// when a process exits, so connect/write alone is not startup evidence.
		var ack [1]byte
		if _, err := io.ReadFull(conn, ack[:]); err != nil || ack[0] != 'A' {
			os.Exit(8)
		}
		if err := os.WriteFile(marker+".ready", nil, 0600); err != nil {
			os.Exit(8)
		}
		// The leaf owns this socket for its entire remaining lifetime. Echoes
		// support an independent live control; EOF follows termination, not a
		// delayed marker write or production ownership bookkeeping.
		var b [1]byte
		for {
			_, err := io.ReadFull(conn, b[:])
			if err == io.EOF {
				os.Exit(0)
			}
			if err != nil {
				os.Exit(8)
			}
			if _, err := conn.Write(b[:]); err != nil {
				os.Exit(8)
			}
		}
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
	marker := filepath.Join(t.TempDir(), "descendant startup 日本語")
	listener := descendantObserver(t)
	setCommandSpec(t, db, &lease, func(spec *config.Test) {
		spec.Command = []string{os.Args[0], "-test.run=^TestCancellationProcessHelper$"}
		spec.Timeout = "15s"
		spec.Artifacts = nil
		spec.Env["AGENT_ENV_CANCELLATION_HELPER"] = "parent"
		spec.Env["AGENT_ENV_CANCELLATION_MARKER"] = marker
		spec.Env["AGENT_ENV_CANCELLATION_OBSERVER"] = listener.Addr().String()
		spec.Env["GORACE"] = "atexit_sleep_ms=0"
	})
	signal := &readySignalWriter{ready: make(chan struct{})}
	s.Stdout = signal
	type outcome struct {
		run domain.CommandRun
		err error
	}
	done := startFixtureOperation(t, context.Background(), func(ctx context.Context) outcome {
		run, err := s.Test(ctx, lease.ID, "check")
		return outcome{run, err}
	})
	conn := observeDescendant(t, listener)
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
		select {
		case result := <-done:
			t.Fatalf("destroy: %v; command status %s: %v", err, result.run.Status, result.err)
		case <-time.After(5 * time.Second):
			t.Fatalf("destroy: %v; command did not return", err)
		}
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
	requireDescendantExit(t, conn)
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
			done := startFixtureOperation(t, context.Background(), func(ctx context.Context) error {
				_, err := s.Test(ctx, lease.ID, "check")
				return err
			})
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

func descendantObserver(t *testing.T) *net.TCPListener {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close observer: %v", err)
		}
	})
	return listener
}

func observeDescendant(t *testing.T, listener *net.TCPListener) net.Conn {
	t.Helper()
	if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal("descendant connection missing:", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close descendant observer: %v", err)
		}
	})
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil || b[0] != 'R' {
		t.Fatalf("descendant startup evidence %q: %v", b, err)
	}
	if _, err := conn.Write([]byte{'A'}); err != nil {
		t.Fatal(err)
	}
	return conn
}

func requireDescendantExit(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if n, err := conn.Read(b[:]); n != 0 || !descendantTerminalRead(err) {
		t.Fatalf("descendant lifetime connection did not end: n=%d err=%v", n, err)
	}
}

func TestDescendantObserverPositiveControl(t *testing.T) {
	listener := descendantObserver(t)
	marker := filepath.Join(t.TempDir(), "observer 日本語")
	ctx, cancel := context.WithCancel(context.Background())
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCancellationProcessHelper$")
	child.Env = append(os.Environ(), "AGENT_ENV_CANCELLATION_HELPER=leaf", "AGENT_ENV_CANCELLATION_MARKER="+marker, "AGENT_ENV_CANCELLATION_OBSERVER="+listener.Addr().String(), "GORACE=atexit_sleep_ms=0")
	if err := child.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		cancel()
		if !waited {
			_ = child.Wait()
		}
	})
	conn := observeDescendant(t, listener)
	if _, err := conn.Write([]byte{'L'}); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil || b[0] != 'L' {
		t.Fatalf("live descendant did not answer: %q %v", b, err)
	}
	// Closing only our write side explicitly asks the positive-control leaf to
	// exit. Its read side must then report EOF or a remote reset and native Wait must succeed.
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	requireDescendantExit(t, conn)
	err := child.Wait()
	waited = true
	if err != nil {
		t.Fatal("positive-control leaf exit:", err)
	}
}

// A process-owned TCP socket can end with EOF or a remote connection reset on
// native Windows. No timeout, local close, refusal, or unrelated network error
// establishes descendant termination.
func descendantTerminalRead(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	var op *net.OpError
	if !errors.As(err, &op) || op.Op != "read" {
		return false
	}
	if runtime.GOOS == "windows" {
		return errors.Is(err, syscall.Errno(10054))
	}
	return errors.Is(err, syscall.ECONNRESET)
}

func TestDescendantTerminalReadClassification(t *testing.T) {
	reset := syscall.ECONNRESET
	if runtime.GOOS == "windows" {
		reset = syscall.Errno(10054)
	}
	for _, tc := range []struct {
		name     string
		err      error
		terminal bool
	}{
		{"eof", io.EOF, true},
		{"remote-reset", &net.OpError{Op: "read", Net: "tcp", Err: &os.SyscallError{Syscall: "recv", Err: reset}}, true},
		{"timeout", &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}, false},
		{"local-close", &net.OpError{Op: "read", Net: "tcp", Err: net.ErrClosed}, false},
		{"refused", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNREFUSED}, false},
		{"write-reset", &net.OpError{Op: "write", Net: "tcp", Err: reset}, false},
		{"bare-reset", reset, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := descendantTerminalRead(tc.err); got != tc.terminal {
				t.Fatalf("terminal=%v want %v for %v", got, tc.terminal, tc.err)
			}
		})
	}
}
