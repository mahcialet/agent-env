package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

func TestAppCommandHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_COMMAND_HELPER") != "1" {
		return
	}
	if os.Getenv("AGENT_ENV_COMMAND_MODE") == "sleep" {
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}
	secret := os.Getenv("AGENT_ENV_COMMAND_TOKEN")
	for _, b := range []byte("output " + secret + " 日本語\n") {
		_, _ = os.Stdout.Write([]byte{b})
	}
	_, _ = fmt.Fprint(os.Stderr, "diagnostic "+secret)
	cwd, _ := os.Getwd()
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"cwd": cwd, "args": os.Args})
	_ = os.MkdirAll("reports", 0700)
	_ = os.WriteFile(filepath.Join("reports", "result.txt"), []byte("artifact "+secret), 0600)
	if os.Getenv("AGENT_ENV_COMMAND_MODE") == "fail" {
		os.Exit(3)
	}
	os.Exit(0)
}

type commandSource struct{ calls, dirtyAfter int }

func (*commandSource) Resolve(context.Context, string, string) (domain.Source, error) {
	return domain.Source{}, errors.New("not expected")
}
func (*commandSource) Materialize(context.Context, domain.Source) error {
	return errors.New("not expected")
}
func (s *commandSource) Inspect(_ context.Context, source domain.Source) (SourceObservation, error) {
	s.calls++
	return SourceObservation{Exists: true, Registered: true, Commit: source.Commit, TrackedDirty: s.dirtyAfter > 0 && s.calls >= s.dirtyAfter}, nil
}
func (*commandSource) Remove(context.Context, domain.Source, bool) error {
	return errors.New("not expected")
}

func commandFixture(t *testing.T, management ...*domain.Management) (*Service, *sqlite.Store, domain.Lease) {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, "source 日本語 with spaces")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	m := config.Manifest{Version: 1, Sources: map[string]config.Source{"self": {Repository: ".", DefaultRef: "HEAD"}}, Runtimes: map[string]config.Runtime{"local": {Type: "compose", Source: "self", ProjectDirectory: ".", Files: []string{"compose.yaml"}}}, Components: map[string]config.Component{"api": {Runtime: "local", ComposeServices: []string{"api"}}, "extra": {Runtime: "local", ComposeServices: []string{"extra"}}}, Stacks: map[string]config.Stack{"small": {Roots: []string{"api"}}, "large": {Roots: []string{"api", "extra"}}}, Tests: map[string]config.Test{"check": {Stack: "small", Source: "self", WorkingDirectory: ".", Command: []string{exe, "-test.run=^TestAppCommandHelper$", "--", "space 日本語", "${env:AGENT_ENV_COMMAND_TOKEN}"}, Env: map[string]string{"AGENT_ENV_COMMAND_HELPER": "1", "AGENT_ENV_COMMAND_TOKEN": "${env:AGENT_ENV_COMMAND_TOKEN}"}, Timeout: "5s", Artifacts: []string{"reports"}}}}
	if err := config.Validate(&m); err != nil {
		t.Fatal(err)
	}
	data, _ := config.CanonicalJSON(&m)
	now := time.Now().UTC()
	lease := domain.Lease{ID: newID(), Owner: "fixture", Mode: "review", Stack: "small", Desired: "active", Observed: "ready", CreatedAt: now, HeartbeatAt: now, ExpiresAt: now.Add(time.Hour), Manifest: data, ManifestDigest: config.Digest(&m), Sources: []domain.Source{{Alias: "self", RepositoryID: root, RepositoryPath: root, WorktreePath: root, Commit: strings.Repeat("a", 40), CheckoutMode: "detached"}}, Components: []domain.Component{{Name: "api", Runtime: "local", Services: []string{"api"}}}}
	db, err := sqlite.Open(filepath.Join(home, "registry.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if len(management) > 0 {
		lease.Management = copyManagement(management[0])
	}
	if err := db.Reserve(context.Background(), lease, 0); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_ENV_COMMAND_TOKEN", "private-value-日本語-42")
	return &Service{Home: home, Store: db, Source: &commandSource{}, Runner: execx.OSRunner{}}, db, lease
}

func setCommandSpec(t *testing.T, db *sqlite.Store, lease *domain.Lease, change func(*config.Test)) {
	t.Helper()
	var m config.Manifest
	if err := json.Unmarshal(lease.Manifest, &m); err != nil {
		t.Fatal(err)
	}
	spec := m.Tests["check"]
	change(&spec)
	m.Tests["check"] = spec
	data, _ := config.CanonicalJSON(&m)
	lease.Manifest = data
	lease.ManifestDigest = config.Digest(&m)
	if err := db.Save(context.Background(), *lease); err != nil {
		t.Fatal(err)
	}
}

func TestNamedCommandPersistsRedactedEvidence(t *testing.T) {
	s, db, lease := commandFixture(t)
	var stdout, stderr bytes.Buffer
	s.Stdout = &stdout
	s.Stderr = &stderr
	run, err := s.Test(context.Background(), lease.ID, "check")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "passed" || run.ExitCode != 0 || run.FinishedAt.Before(run.StartedAt) {
		t.Fatalf("run %+v", run)
	}
	secret := os.Getenv("AGENT_ENV_COMMAND_TOKEN")
	if strings.Contains(stdout.String()+stderr.String(), secret) || !strings.Contains(stdout.String(), "[REDACTED]") {
		t.Fatal("stream exposed secret or lacked replacement")
	}
	if !strings.Contains(stdout.String(), "source 日本語 with spaces") || !strings.Contains(stdout.String(), "space 日本語") {
		t.Fatal("cwd/argv did not round trip")
	}
	runs, err := db.Runs(context.Background(), lease.ID)
	if err != nil || len(runs) != 1 || runs[0].Status != "passed" {
		t.Fatalf("runs %+v %v", runs, err)
	}
	artifacts, err := db.Artifacts(context.Background(), lease.ID)
	if err != nil || len(artifacts) != 4 {
		t.Fatalf("artifacts %+v %v", artifacts, err)
	}
	for _, artifact := range artifacts {
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(secret)) {
			t.Fatalf("secret in %s", artifact.Kind)
		}
		digest, err := evidence.FileDigest(artifact.Path)
		if err != nil || digest != artifact.Digest {
			t.Fatalf("digest %s %v", artifact.Kind, err)
		}
	}
	encoded, _ := json.Marshal(runs)
	if bytes.Contains(encoded, []byte(secret)) {
		t.Fatal("secret in stored argv")
	}
	got, err := db.Get(context.Background(), lease.ID)
	if err != nil || got.Desired != "active" || !got.HeartbeatAt.After(lease.HeartbeatAt) {
		t.Fatalf("heartbeat/state %+v %v", got, err)
	}
}

func TestNamedCommandFailuresRetainEvidence(t *testing.T) {
	for _, mode := range []string{"fail", "sleep"} {
		t.Run(mode, func(t *testing.T) {
			s, db, lease := commandFixture(t)
			setCommandSpec(t, db, &lease, func(spec *config.Test) {
				spec.Env["AGENT_ENV_COMMAND_MODE"] = mode
				if mode == "sleep" {
					spec.Timeout = "50ms"
					spec.Artifacts = nil
				}
			})
			run, err := s.Test(context.Background(), lease.ID, "check")
			if !errors.Is(err, ErrTestFailed) {
				t.Fatalf("expected test failure, got %v", err)
			}
			if mode == "fail" && (run.ExitCode != 3 || run.Status != "failed") {
				t.Fatalf("exit %+v", run)
			}
			if mode == "sleep" && run.Status != "timed_out" {
				t.Fatalf("timeout %+v", run)
			}
			runs, e := db.Runs(context.Background(), lease.ID)
			if e != nil || len(runs) != 1 || runs[0].FinishedAt.IsZero() {
				t.Fatalf("missing final run: %+v %v", runs, e)
			}
			if _, e := os.Stat(filepath.Join(filepath.Dir(run.StdoutPath), "run.json")); e != nil {
				t.Fatal(e)
			}
			// This probes lock release, not renewal timing; allow native DB/scheduler latency.
			release, e := db.Acquire(context.Background(), lease.ID, newID(), time.Minute)
			if e != nil {
				t.Fatalf("lock not released: %v", e)
			}
			if e = release(); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestNamedCommandRejectsUnknownAndWrongStack(t *testing.T) {
	s, db, lease := commandFixture(t)
	if _, err := s.Test(context.Background(), lease.ID, "unknown"); err == nil {
		t.Fatal("unknown test accepted")
	}
	setCommandSpec(t, db, &lease, func(spec *config.Test) { spec.Stack = "large" })
	if _, err := s.Test(context.Background(), lease.ID, "check"); err == nil || !strings.Contains(err.Error(), "component extra") {
		t.Fatalf("wrong stack: %v", err)
	}
	runs, err := db.Runs(context.Background(), lease.ID)
	if err != nil || len(runs) != 0 {
		t.Fatalf("rejected command started: %v %v", runs, err)
	}
}

func TestNamedCommandQuarantinesTrackedChanges(t *testing.T) {
	for _, after := range []int{1, 2} {
		t.Run(fmt.Sprint(after), func(t *testing.T) {
			s, db, lease := commandFixture(t)
			s.Source = &commandSource{dirtyAfter: after}
			_, err := s.Test(context.Background(), lease.ID, "check")
			if err == nil {
				t.Fatal("tracked changes accepted")
			}
			got, e := db.Get(context.Background(), lease.ID)
			if e != nil || got.Observed != "quarantined" {
				t.Fatalf("state %+v %v", got, e)
			}
			if _, e := os.Stat(lease.Sources[0].WorktreePath); e != nil {
				t.Fatalf("source was deleted: %v", e)
			}
		})
	}
}

type blockingCommandRunner struct{ entered, finish chan struct{} }

func (r blockingCommandRunner) Run(ctx context.Context, _ execx.Command) (execx.Result, error) {
	close(r.entered)
	select {
	case <-r.finish:
		return execx.Result{ExitCode: 0}, nil
	case <-ctx.Done():
		return execx.Result{ExitCode: -1}, ctx.Err()
	}
}
func TestNamedCommandSerializesDestroy(t *testing.T) {
	s, db, lease := commandFixture(t)
	s.Source = &cleanupAfterRunSource{SourceProvider: s.Source, store: db, leaseID: lease.ID}
	setCommandSpec(t, db, &lease, func(spec *config.Test) { spec.Artifacts = nil })
	runner := blockingCommandRunner{make(chan struct{}), make(chan struct{})}
	s.Runner = runner
	done := startFixtureOperation(t, context.Background(), func(ctx context.Context) error {
		_, err := s.Test(ctx, lease.ID, "check")
		return err
	})
	select {
	case <-runner.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("test did not start")
	}
	if _, err := s.Test(context.Background(), lease.ID, "check"); !errors.Is(err, sqlite.ErrBusy) {
		t.Errorf("concurrent test: %v", err)
	}
	if destroyed, err := s.Destroy(context.Background(), lease.ID, false, false); err != nil || destroyed.Observed != "released" {
		t.Errorf("concurrent destroy failed to cancel and finalize before cleanup: %+v %v", destroyed, err)
	}
	if err := <-done; !errors.Is(err, ErrTestFailed) {
		t.Fatalf("active test did not report cancellation: %v", err)
	}
}

type failingFinalRunStore struct {
	Store
	calls int
}

func (s *failingFinalRunStore) SaveRun(ctx context.Context, run domain.CommandRun) error {
	s.calls++
	if s.calls > 1 {
		return errors.New("registry final write failed")
	}
	return s.Store.SaveRun(ctx, run)
}
func TestNamedCommandKeepsFilesOnRegistryFailure(t *testing.T) {
	s, _, lease := commandFixture(t)
	s.Store = &failingFinalRunStore{Store: s.Store}
	run, err := s.Test(context.Background(), lease.ID, "check")
	if err == nil || !strings.Contains(err.Error(), "registry final write failed") {
		t.Fatalf("%v", err)
	}
	for _, path := range []string{run.StdoutPath, run.StderrPath, filepath.Join(filepath.Dir(run.StdoutPath), "run.json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNamedCommandRejectsArtifactEscape(t *testing.T) {
	s, db, lease := commandFixture(t)
	outside := filepath.Join(t.TempDir(), "external")
	if err := os.WriteFile(outside, []byte("external private content"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(lease.Sources[0].WorktreePath, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	setCommandSpec(t, db, &lease, func(spec *config.Test) { spec.Artifacts = []string{"escape"} })
	if _, err := s.Test(context.Background(), lease.ID, "check"); err == nil {
		t.Fatal("artifact escape accepted")
	}
	artifacts, err := db.Artifacts(context.Background(), lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range artifacts {
		data, _ := os.ReadFile(a.Path)
		if bytes.Contains(data, []byte("external private content")) {
			t.Fatal("outside file copied")
		}
	}
}

func TestNamedCommandLookupFailureFinalizesWithoutUnproducedArtifact(t *testing.T) {
	s, db, lease := commandFixture(t)
	setCommandSpec(t, db, &lease, func(spec *config.Test) { spec.Command = []string{"agent-env-fixture-nonexistent-executable-817eb93"} })
	run, err := s.Test(context.Background(), lease.ID, "check")
	if !errors.Is(err, ErrTestFailed) || run.Status != "failed" || run.ExitCode != -1 {
		t.Fatalf("lookup failure: %+v %v", run, err)
	}
	if !strings.Contains(err.Error(), "test artifact reports") {
		t.Fatalf("unproduced output not reported: %v", err)
	}
	runs, e := db.Runs(context.Background(), lease.ID)
	if e != nil || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatalf("no-process launch must finalize failed run: %+v %v", runs, e)
	}
	artifacts, e := db.Artifacts(context.Background(), lease.ID)
	if e != nil || len(artifacts) != 3 {
		t.Fatalf("retained stdout/stderr/run evidence: %+v %v", artifacts, e)
	}
	counts := map[string]int{}
	for _, artifact := range artifacts {
		counts[artifact.Kind]++
		if _, err := os.Stat(artifact.Path); err != nil {
			t.Fatalf("registered failure evidence missing: %s %v", artifact.Kind, err)
		}
	}
	for _, kind := range []string{"test-stdout", "test-stderr", "test-run"} {
		if counts[kind] != 1 {
			t.Fatalf("failure evidence kinds: %+v", counts)
		}
	}
	_, release, e := s.acquireDestroyAfterCancellation(context.Background(), lease.ID)
	if e != nil {
		t.Fatalf("proven no-process failure blocked cleanup: %v", e)
	}
	if e = release(); e != nil {
		t.Fatal(e)
	}
}

type lookupBarrierRunner struct{ marker error }

func (r lookupBarrierRunner) Run(context.Context, execx.Command) (execx.Result, error) {
	return execx.Result{ExitCode: -1}, errors.Join(&exec.Error{Name: "missing", Err: exec.ErrNotFound}, r.marker)
}
func TestNamedCommandLookupFailureDoesNotBypassUnconfirmedEvidence(t *testing.T) {
	for _, marker := range []error{execx.ErrProcessTreeUnconfirmed, execx.ErrOutputIncomplete} {
		t.Run(marker.Error(), func(t *testing.T) {
			s, db, lease := commandFixture(t)
			s.Runner = lookupBarrierRunner{marker}
			if _, err := s.Test(context.Background(), lease.ID, "check"); err == nil {
				t.Fatal("unsafe execution accepted")
			}
			runs, err := db.Runs(context.Background(), lease.ID)
			if err != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("unsafe cleanup barrier lost: %+v %v", runs, err)
			}
			if _, release, err := s.acquireDestroyAfterCancellation(context.Background(), lease.ID); err == nil {
				release()
				t.Fatal("unconfirmed run allowed cleanup")
			}
		})
	}
}
