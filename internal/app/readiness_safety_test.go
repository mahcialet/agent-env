package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

type readinessSafetyRunner struct {
	calls             int
	err               error
	recoverAfterFirst bool
}

func (r *readinessSafetyRunner) Run(context.Context, execx.Command) (execx.Result, error) {
	r.calls++
	if r.recoverAfterFirst && r.calls > 1 {
		return execx.Result{ExitCode: 0}, nil
	}
	return execx.Result{ExitCode: -1, Stderr: "readiness-secret"}, r.err
}

func TestReadinessRetainsUnconfirmedCommand(t *testing.T) {
	for _, unsafe := range []error{execx.ErrProcessTreeUnconfirmed, execx.ErrOutputIncomplete} {
		t.Run(unsafe.Error(), func(t *testing.T) {
			t.Setenv("AUDIT_READINESS_TOKEN", "readiness-secret")
			s, o, source, _, ops := lifecycleFixture(t)
			p := filepath.Join(o.Repository, ".agent-env.yaml")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			b = []byte(strings.Replace(string(b), "database: {runtime: first, compose_services: [db]}", "database: {runtime: first, compose_services: [db], readiness: [{type: command, source: self, working_directory: ., command: [probe], timeout: 5s, interval: 1ms}]}", 1))
			if err := os.WriteFile(p, b, 0600); err != nil {
				t.Fatal(err)
			}
			runner := &readinessSafetyRunner{err: fmt.Errorf("readiness-secret: %w", unsafe)}
			s.Runner = runner
			l, err := s.Create(context.Background(), o, CreateOptions{Owner: "audit"})
			if !errors.Is(err, unsafe) || strings.Contains(fmt.Sprint(err), "readiness-secret") {
				t.Fatalf("lost safety classification or leaked secret: %v", err)
			}
			if runner.calls != 1 || l.Observed != "quarantined" || len(source.states) != 1 {
				t.Fatalf("unsafe retry/cleanup: calls=%d state=%s sources=%d ops=%v", runner.calls, l.Observed, len(source.states), *ops)
			}
			runs, err := s.Store.Runs(context.Background(), l.ID)
			if err != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("missing persistent command barrier: %+v, %v", runs, err)
			}
			_, err = s.Destroy(context.Background(), l.ID, true, false)
			if err == nil || len(source.states) != 1 {
				t.Fatalf("later forced destroy bypassed unresolved command: %v, sources=%d", err, len(source.states))
			}
		})
	}
}

func TestReadinessOrdinaryFailureMayRetryAndRelease(t *testing.T) {
	s, o, source, _, _ := lifecycleFixture(t)
	p := filepath.Join(o.Repository, ".agent-env.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), "database: {runtime: first, compose_services: [db]}", "database: {runtime: first, compose_services: [db], readiness: [{type: command, source: self, working_directory: ., command: [probe], timeout: 5s, interval: 1ms}]}", 1))
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	runner := &readinessSafetyRunner{err: errors.New("ordinary probe failure"), recoverAfterFirst: true}
	s.Runner = runner
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "audit"})
	if err != nil || l.Observed != "ready" || runner.calls != 2 {
		t.Fatalf("ordinary retry changed: state=%s calls=%d err=%v", l.Observed, runner.calls, err)
	}
	runs, err := s.Store.Runs(context.Background(), l.ID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("missing attempts: %+v %v", runs, err)
	}
	statuses := map[string]int{}
	for _, run := range runs {
		statuses[run.Status]++
	}
	if statuses["passed"] != 1 || statuses["failed"] != 1 {
		t.Fatalf("attempts not terminal: %+v", runs)
	}
	l, err = s.Destroy(context.Background(), l.ID, true, false)
	if err != nil || l.Observed != "released" || len(source.states) != 0 {
		t.Fatalf("completed probes block release: %+v %v", l, err)
	}
}

type reviewCancellationRunner struct{ started chan struct{} }

func (r *reviewCancellationRunner) Run(ctx context.Context, _ execx.Command) (execx.Result, error) {
	r.started <- struct{}{}
	<-ctx.Done()
	return execx.Result{ExitCode: -1}, ctx.Err()
}
func TestReviewReadinessCancellationDoesNotRetry(t *testing.T) {
	s, o, _, _, _ := lifecycleFixture(t)
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "review"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, release, err := s.Store.AcquireContext(context.Background(), l.ID, "review", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Errorf("release readiness fence: %v", err)
		}
	})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	runner := &reviewCancellationRunner{started: make(chan struct{}, 3)}
	s.Runner = runner
	done := startFixtureOperation(t, ctx, func(ctx context.Context) error {
		return s.runProbe(ctx, l, "database", 0, config.Probe{Type: "command", Source: "self", WorkingDirectory: ".", Command: []string{"probe"}, Timeout: "10s", Interval: "1ms"})
	})
	select {
	case <-runner.started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe never started")
	}
	runs, err := s.Store.Runs(context.Background(), l.ID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs %v %v", runs, err)
	}
	if err = s.Store.RequestRunCancel(context.Background(), runs[0].ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runner.started:
		cancel()
		<-done
		t.Fatal("durably canceled readiness command started a new attempt")
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation sentinel lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatal("cancellation not acknowledged")
	}
}
