package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type lifecycleProcess struct {
	store                Store
	alive                map[string]bool
	startErr, inspectErr error
	starts, destroys     int
	cancel               context.CancelCauseFunc
}

func (p *lifecycleProcess) Prepare(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	saved, err := p.store.Get(ctx, r.LeaseID)
	if err != nil {
		return r, err
	}
	var found bool
	for _, v := range saved.Runtimes {
		if v.Name == r.Name && v.Process != nil && v.Process.Directory == r.Process.Directory {
			found = true
		}
	}
	if !found {
		return r, errors.New("process preparation before durable directory intent")
	}
	snapshot := *r.Process
	r.Process = &snapshot
	snapshot.Executable = "fixture-native"
	snapshot.State = "prepared"
	return r, nil
}
func (p *lifecycleProcess) Start(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	saved, err := p.store.Get(ctx, r.LeaseID)
	if err != nil {
		return r, err
	}
	found := false
	for _, v := range saved.Runtimes {
		if v.Name == r.Name && v.Started && v.Process.State == "launching" && v.Process.Executable != "" {
			found = true
		}
	}
	if !found {
		return r, errors.New("process start before durable prepared launch intent")
	}
	p.starts++
	p.alive[r.LeaseID+":"+r.Name] = true
	snapshot := *r.Process
	r.Process = &snapshot
	snapshot.ProcessID = 9000 + p.starts
	snapshot.ProcessStart = "owned-birth"
	snapshot.State = "running"
	if p.cancel != nil {
		p.cancel(domain.ErrLockLost)
	}
	return r, p.startErr
}
func (p *lifecycleProcess) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeObservation{}, err
	}
	if p.inspectErr != nil {
		return RuntimeObservation{}, p.inspectErr
	}
	alive := p.alive[r.LeaseID+":"+r.Name]
	o := RuntimeObservation{Exists: alive, Ready: alive, Endpoints: map[string]string{}}
	if alive {
		o.Resources = []domain.Resource{{Kind: "process", ID: r.Name, Runtime: r.Name, Metadata: map[string]string{"lease": r.LeaseID, "runtime": r.Name}}}
		for name, port := range r.Process.Ports {
			o.Endpoints[name] = fmt.Sprintf("127.0.0.1:%d", port)
		}
	}
	return o, nil
}
func (p *lifecycleProcess) Logs(context.Context, domain.Runtime) (string, error) {
	return "persistent process log", nil
}
func (p *lifecycleProcess) Destroy(ctx context.Context, r domain.Runtime) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.inspectErr != nil {
		return p.inspectErr
	}
	p.destroys++
	delete(p.alive, r.LeaseID+":"+r.Name)
	return nil
}

func processLifecycleFixture(t *testing.T, mixed bool) (*Service, PlanOptions, *lifecycleProcess, *lifecycleSource) {
	t.Helper()
	s, o, source, _, _ := lifecycleFixture(t)
	manifest := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  api: {type: process, source: self, working_directory: ., command: [fixture-native, "${port:http}"], ports: {http: {protocol: tcp}}}
components:
  api: {runtime: api, endpoints: {http: {runtime_port: http}}}
stacks:
  review: {roots: [api]}
`
	if mixed {
		manifest = strings.Replace(manifest, "components:", "  db: {type: compose, source: self, project_directory: ., files: [compose.yaml]}\ncomponents:", 1)
		manifest = strings.Replace(manifest, "  api: {runtime: api,", "  database: {runtime: db, compose_services: [db]}\n  api: {runtime: api, depends_on: [database],", 1)
	} else {
		s.Runtime = nil
	}
	if err := os.WriteFile(filepath.Join(o.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	p := &lifecycleProcess{store: s.Store, alive: map[string]bool{}}
	s.Process = p
	return s, o, p, source
}

func TestProcessLifecyclePersistedIntentMixedRoutingAndIsolation(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(fmt.Sprint(mixed), func(t *testing.T) {
			s, o, p, source := processLifecycleFixture(t, mixed)
			ctx := context.Background()
			one, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
			if err != nil {
				t.Fatal(err)
			}
			two, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
			if err != nil {
				t.Fatal(err)
			}
			get := func(l domain.Lease) domain.Runtime {
				for _, r := range l.Runtimes {
					if r.Type == "process" {
						return r
					}
				}
				t.Fatal("missing process")
				return domain.Runtime{}
			}
			a, b := get(one), get(two)
			if a.Process.ProcessID == 0 || a.Process.Ports["http"] <= 0 || a.Process.Ports["http"] == b.Process.Ports["http"] || a.Process.Directory == b.Process.Directory {
				t.Fatalf("isolation: %+v %+v", a, b)
			}
			endpoints, err := s.Endpoints(ctx, one)
			if err != nil || endpoints["api.http"] != fmt.Sprintf("127.0.0.1:%d", a.Process.Ports["http"]) {
				t.Fatalf("endpoints=%v err=%v", endpoints, err)
			}
			one, err = s.Destroy(ctx, one.ID, false, false)
			if err != nil || one.Observed != "released" {
				t.Fatalf("destroy=%s %v", one.Observed, err)
			}
			if len(p.alive) != 1 || len(source.states) != 1 {
				t.Fatal("sibling process or source affected")
			}
			if get(one).Process.ProcessID == 0 || get(one).Process.State != "released" {
				t.Fatal("cleanup discarded identity evidence")
			}
			artifacts, err := s.Store.Artifacts(ctx, one.ID)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, a := range artifacts {
				found = found || a.Kind == "process-log/api"
			}
			if !found {
				t.Fatal("process logs not captured")
			}
			one, err = s.Reconcile(ctx, one.ID)
			if err != nil || one.Observed != "released" {
				t.Fatalf("released reconciliation=%s %v", one.Observed, err)
			}
			two, err = s.Reconcile(ctx, two.ID)
			if err != nil || two.Observed != "ready" {
				t.Fatalf("sibling reconcile=%s %v", two.Observed, err)
			}
			if _, err = s.Destroy(ctx, two.ID, false, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type processIdentityFailStore struct {
	Store
	failed bool
}

func (s *processIdentityFailStore) Save(ctx context.Context, l domain.Lease) error {
	for _, r := range l.Runtimes {
		if r.Process != nil && r.Process.ProcessID > 0 && !s.failed {
			s.failed = true
			return errors.New("fixture identity persistence failed")
		}
	}
	return s.Store.Save(ctx, l)
}
func TestProcessIdentitySaveFailureAndPartialStartCompensate(t *testing.T) {
	for _, failure := range []string{"save", "start"} {
		t.Run(failure, func(t *testing.T) {
			s, o, p, source := processLifecycleFixture(t, false)
			if failure == "save" {
				s.Store = &processIdentityFailStore{Store: s.Store}
			} else {
				p.startErr = errors.New("fixture failed after process launch")
			}
			l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
			if err == nil || l.Observed != "released" || p.starts != 1 || p.destroys != 1 || len(p.alive) != 0 || len(source.states) != 0 {
				t.Fatalf("failed process saga=%+v %v starts=%d destroy=%d", l, err, p.starts, p.destroys)
			}
			saved, e := s.Store.Get(context.Background(), l.ID)
			if e != nil || saved.Runtimes[0].Process.ProcessID == 0 {
				t.Fatalf("lost returned identity: %+v %v", saved, e)
			}
		})
	}
}

func TestProcessUnknownOwnershipQuarantinesAndRecovers(t *testing.T) {
	s, o, p, source := processLifecycleFixture(t, false)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	p.inspectErr = errors.Join(domain.ErrResourceIdentity, errors.New("identity mismatch"))
	l, err = s.Destroy(ctx, l.ID, false, false)
	if err == nil || l.Observed != "quarantined" || p.destroys != 0 || len(source.states) != 1 {
		t.Fatalf("uncertain cleanup=%+v %v", l, err)
	}
	p.inspectErr = nil
	l, err = s.Destroy(ctx, l.ID, false, false)
	if err != nil || l.Observed != "released" {
		t.Fatalf("recovery=%s %v", l.Observed, err)
	}
}

type processFenceStore struct {
	Store
	cancel context.CancelCauseFunc
}

func (s *processFenceStore) AcquireContext(ctx context.Context, id, owner string, d time.Duration) (context.Context, func() error, error) {
	owned, release, err := s.Store.AcquireContext(ctx, id, owner, d)
	if err != nil {
		return owned, release, err
	}
	wrapped, cancel := context.WithCancelCause(owned)
	s.cancel = cancel
	return wrapped, release, nil
}
func TestProcessFenceLossRetainsLaunchForLaterRecovery(t *testing.T) {
	s, o, p, source := processLifecycleFixture(t, false)
	base := s.Store
	fence := &processFenceStore{Store: base}
	s.Store = fence
	// Attach the cancellation hook only after the operation context exists.
	wrapped := &processFenceProvider{lifecycleProcess: p, fence: fence}
	s.Process = wrapped
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if !errors.Is(err, domain.ErrLockLost) || p.starts != 1 || p.destroys != 0 || len(p.alive) != 1 || len(source.states) != 1 {
		t.Fatalf("fence loss had compensation effects: %+v %v", l, err)
	}
	s.Store = base
	s.Process = p
	p.cancel = nil
	l, err = s.Destroy(context.Background(), l.ID, false, false)
	if err != nil || l.Observed != "released" || len(p.alive) != 0 {
		t.Fatalf("later recovery=%s %v", l.Observed, err)
	}
}

type processFenceProvider struct {
	*lifecycleProcess
	fence *processFenceStore
}

func (p *processFenceProvider) Start(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	p.cancel = p.fence.cancel
	return p.lifecycleProcess.Start(ctx, r)
}

type processProbeRunner struct {
	got    execx.Command
	output string
}

func (r *processProbeRunner) Run(_ context.Context, c execx.Command) (execx.Result, error) {
	r.got = c
	return execx.Result{Stdout: r.output}, nil
}
func TestProcessReadinessUsesRecordedNumericEndpoint(t *testing.T) {
	s, o, _, _ := processLifecycleFixture(t, false)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := s.Destroy(ctx, l.ID, false, false); err != nil {
			t.Errorf("destroy process fixture: %v", err)
		}
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("ready"))
	}))
	defer server.Close()
	_, portText, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portText)
	l.Runtimes[0].Process.Ports["http"] = port
	if err := s.runProbe(ctx, l, "api", 0, config.Probe{Type: "http", URL: "http://127.0.0.1:${endpoint:http}/health", Timeout: "1s"}); err != nil {
		t.Fatal(err)
	}
	runner := &processProbeRunner{}
	s.Runner = runner
	probe := config.Probe{Type: "command", Source: "self", Command: []string{"fixture-check", "--port", "${endpoint:http}"}, Timeout: "1s"}
	if err := s.runProbe(ctx, l, "api", 1, probe); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.got.Args, []string{"--port", portText}) {
		t.Fatalf("readiness argv=%v", runner.got.Args)
	}
	if probe.Command[2] != "${endpoint:http}" {
		t.Fatal("readiness expansion mutated manifest command")
	}
	if _, err := s.processProbe(l, "api", config.Probe{URL: "http://127.0.0.1:${endpoint:missing}"}); err == nil {
		t.Fatal("unknown endpoint accepted")
	}
	t.Setenv("PROCESS_READINESS_VALUE", "private-${literal}-value")
	expanded, err := s.processProbe(l, "api", config.Probe{Command: []string{"fixture", "${runtime_dir}", "${lease_id}", "${port:http}", "${env:PROCESS_READINESS_VALUE}"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"fixture", l.Runtimes[0].Process.StateDirectory, l.ID, portText, "private-${literal}-value"}
	if !reflect.DeepEqual(expanded.Command, want) {
		t.Fatalf("process command expansion=%v", expanded.Command)
	}
	if _, err := s.processProbe(l, "api", config.Probe{Command: []string{"fixture", "${env:AGENT_ENV_DEFINITELY_MISSING_READINESS_VALUE}"}}); err == nil {
		t.Fatal("missing readiness environment accepted")
	}
	secrets := processProbeSecrets(config.Probe{Command: []string{"fixture", "${env:PROCESS_READINESS_VALUE}"}})
	if !reflect.DeepEqual(secrets, []string{"private-${literal}-value"}) {
		t.Fatalf("readiness secret redaction inputs=%v", secrets)
	}
	runner.output = "echo private-${literal}-value"
	if err := s.runProbe(ctx, l, "api", 2, config.Probe{Type: "command", Source: "self", Command: []string{"fixture", "${env:PROCESS_READINESS_VALUE}"}}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := s.Store.Artifacts(ctx, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	redacted := false
	for _, a := range artifacts {
		if a.Kind != "readiness" {
			continue
		}
		data, err := os.ReadFile(a.Path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), runner.output) {
			t.Fatal("referenced readiness environment leaked into artifact")
		}
		redacted = redacted || strings.Contains(string(data), "echo [REDACTED]")
	}
	if !redacted {
		t.Fatal("readiness output did not redact explicitly referenced environment")
	}

}

func TestProcessCrashDegradesWithoutRestartAndReleasedEffectsQuarantine(t *testing.T) {
	s, o, p, _ := processLifecycleFixture(t, false)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	key := l.ID + ":api"
	delete(p.alive, key)
	l, err = s.Reconcile(ctx, l.ID)
	if err != nil || l.Observed != "degraded" || p.starts != 1 {
		t.Fatalf("crash reconcile=%s starts=%d %v", l.Observed, p.starts, err)
	}
	l, err = s.Destroy(ctx, l.ID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	p.alive[key] = true
	l, err = s.Reconcile(ctx, l.ID)
	if err != nil || l.Observed != "quarantined" || p.starts != 1 {
		t.Fatalf("released effects reconcile=%s starts=%d %v", l.Observed, p.starts, err)
	}
	delete(p.alive, key)
}

type exitDuringReadiness struct{ p *lifecycleProcess }

func (r exitDuringReadiness) Run(context.Context, execx.Command) (execx.Result, error) {
	clear(r.p.alive)
	return execx.Result{}, nil
}
func TestProcessExitDuringSuccessfulProbeCannotBecomeReady(t *testing.T) {
	s, o, p, source := processLifecycleFixture(t, false)
	file := filepath.Join(o.Repository, ".agent-env.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "endpoints: {http: {runtime_port: http}}}", "endpoints: {http: {runtime_port: http}}, readiness: [{type: command, source: self, command: [fixture-exit-check], timeout: 1s}]}", 1)
	if err := os.WriteFile(file, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	s.Runner = exitDuringReadiness{p: p}
	lease, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if err == nil || !strings.Contains(err.Error(), "stopped before readiness") || lease.Observed != "released" || len(source.states) != 0 {
		t.Fatalf("dead process accepted successful readiness: %s %v", lease.Observed, err)
	}
}

type knownNoSpawnProvider struct{ *lifecycleProcess }

func (p knownNoSpawnProvider) Start(_ context.Context, r domain.Runtime) (domain.Runtime, error) {
	snapshot := *r.Process
	r.Process = &snapshot
	r.Process.State = "prepared"
	return r, errors.Join(execx.ErrProcessNotStarted, errors.New("fixture native exec denied"))
}
func TestProcessKnownNoSpawnCompensatesPreparedState(t *testing.T) {
	s, o, p, source := processLifecycleFixture(t, false)
	s.Process = knownNoSpawnProvider{p}
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
	if !errors.Is(err, execx.ErrProcessNotStarted) || l.Observed != "released" || len(source.states) != 0 || p.starts != 0 || p.destroys != 1 {
		t.Fatalf("known no-spawn compensation=%+v %v", l, err)
	}
	if l.Runtimes[0].Process.ProcessID != 0 || l.Runtimes[0].Process.State != "released" {
		t.Fatal("known no-spawn state became ambiguous")
	}
}

func TestProcessDestroyPreviewReportsCleanupWithoutEffects(t *testing.T) {
	for _, state := range []string{"live", "exited", "uncertain"} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", state, force), func(t *testing.T) {
				s, options, process, source := processLifecycleFixture(t, false)
				ctx := context.Background()
				lease, err := s.Create(ctx, options, CreateOptions{Owner: "tester"})
				if err != nil {
					t.Fatal(err)
				}
				r := lease.Runtimes[0]
				if state == "exited" {
					delete(process.alive, lease.ID+":"+r.Name)
				}
				if state == "uncertain" {
					process.inspectErr = errors.New("process ownership is uncertain")
				}
				if err := os.MkdirAll(r.Process.StateDirectory, 0700); err != nil {
					t.Fatal(err)
				}
				privateFile := filepath.Join(r.Process.StateDirectory, "profile")
				if err := os.WriteFile(privateFile, []byte("retain private state"), 0600); err != nil {
					t.Fatal(err)
				}
				before, err := s.Store.Get(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				events, err := s.Store.Events(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				artifacts, err := s.Store.Artifacts(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				preview, err := s.Destroy(ctx, lease.ID, force, true)
				if err != nil {
					t.Fatal(err)
				}
				text := strings.Join(preview.Diagnostics, "\n")
				if strings.Contains(text, "Compose") {
					t.Errorf("process preview describes Compose: %s", text)
				}
				if state == "uncertain" {
					if !strings.Contains(text, "would quarantine runtime "+r.Name) || strings.Contains(text, "stop process runtime") {
						t.Errorf("unsafe uncertain preview: %s", text)
					}
				} else {
					for _, want := range []string{"verify native identity", "stop process runtime " + r.Name, "if running", "retain process logs", "whole-tree absence", "remove private state at " + r.Process.StateDirectory} {
						if !strings.Contains(text, want) {
							t.Errorf("preview missing %q: %s", want, text)
						}
					}
				}
				after, err := s.Store.Get(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				afterEvents, err := s.Store.Events(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				afterArtifacts, err := s.Store.Artifacts(ctx, lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				wantLive := 1
				if state == "exited" {
					wantLive = 0
				}
				if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, afterEvents) || !reflect.DeepEqual(artifacts, afterArtifacts) || process.starts != 1 || process.destroys != 0 || len(process.alive) != wantLive || len(source.states) != 1 {
					t.Fatal("dry-run changed registry, evidence or resources")
				}
				data, err := os.ReadFile(privateFile)
				if err != nil || string(data) != "retain private state" {
					t.Fatalf("private state changed: %q %v", data, err)
				}
			})
		}
	}
}
