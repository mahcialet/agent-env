package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type fixtureBrowser struct {
	calls int
	fn    func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error)
}

func (p *fixtureBrowser) Observe(ctx context.Context, r domain.Runtime, b domain.BrowserBinding, q domain.BrowserRequest, verify func(context.Context) error) (domain.BrowserObservation, error) {
	p.calls++
	if e := verify(ctx); e != nil {
		return domain.BrowserObservation{Confirmed: true}, e
	}
	if p.fn != nil {
		return p.fn(ctx, q)
	}
	return domain.BrowserObservation{Confirmed: true, Identity: domain.BrowserIdentity{Runtime: r.Name, PID: r.Process.ProcessID, Birth: r.Process.ProcessStart, Port: r.Process.Ports[b.CDPPort], WebSocket: "ws://127.0.0.1/browser", Product: "Chrome/fixture", Protocol: "1.3"}, Snapshot: &domain.BrowserSnapshot{Page: domain.BrowserPage{ID: "page"}, Document: "document", Nodes: []domain.BrowserNode{{Ref: "n1", BackendID: 1, Frame: "frame", Name: "Name", Role: "textbox", Editable: true, Fingerprint: "digest"}}}}, nil
}
func browserFixture(t *testing.T, management ...*domain.Management) (*Service, domain.Lease, *fixtureBrowser, *lifecycleProcess) {
	t.Helper()
	s, o, p, _ := processLifecycleFixture(t, false)
	data, e := os.ReadFile(filepath.Join(o.Repository, ".agent-env.yaml"))
	if e != nil {
		t.Fatal(e)
	}
	manifest := strings.Replace(string(data), `command: [fixture-native, "${port:http}"]`, `command: [fixture-native, "--headless=new", "--enable-automation", "--user-data-dir=${runtime_dir}/profile", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=${port:http}"]`, 1)
	manifest += "\nbrowsers:\n  web: {type: chromium-cdp, runtime: api, cdp_port: http}\n"
	if e = os.WriteFile(filepath.Join(o.Repository, ".agent-env.yaml"), []byte(manifest), 0600); e != nil {
		t.Fatal(e)
	}
	// Fixture creation is not a readiness-deadline test. The lifecycle helper's
	// 50 ms deadline can cancel SQLite setup on loaded race runners before any
	// browser assertion; retain that setting for all subsequent operations.
	readinessTimeout := s.ReadinessTimeout
	s.ReadinessTimeout = 5 * time.Second
	options := CreateOptions{Owner: "tester"}
	if len(management) > 0 {
		s.Management = copyManagement(management[0])
		options.Management = copyManagement(management[0])
		options.LeaseID = newID()
	}
	l, e := s.Create(context.Background(), o, options)
	s.Management = nil
	s.ReadinessTimeout = readinessTimeout
	if e != nil {
		t.Fatal(e)
	}
	provider := &fixtureBrowser{}
	s.BrowserProvider = provider
	return s, l, provider, p
}
func TestBrowserSnapshotRegistrationAndSemanticInput(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	ctx := context.Background()
	snapshot, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
	if e != nil {
		t.Fatal(e)
	}
	if snapshot.Snapshot == nil || snapshot.Snapshot.ID != snapshot.Run.ID || len(snapshot.Artifacts) != 3 {
		t.Fatalf("missing evidence: %+v", snapshot)
	}
	p.fn = func(ctx context.Context, q domain.BrowserRequest) (domain.BrowserObservation, error) {
		if q.Prior == nil || q.Prior.ID != snapshot.Run.ID || q.Node != "n1" {
			t.Fatal("registered snapshot not routed")
		}
		return domain.BrowserObservation{Confirmed: true, ReadbackEqual: true, Detail: q.Text}, nil
	}
	result, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "set-text", Text: "日本語 and café", Node: "n1"}, Snapshot: snapshot.Run.ID})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "日本語") || !strings.Contains(string(raw), "[REDACTED]") {
		t.Fatalf("text disclosure: %s", raw)
	}
	for _, a := range result.Artifacts {
		data, e := os.ReadFile(a.Path)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(data), "日本語") {
			t.Fatal("text in artifact")
		}
	}
	for _, a := range snapshot.Artifacts {
		if a.Kind == "browser-snapshot" {
			if e = os.WriteFile(a.Path, []byte("{}"), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	calls := p.calls
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "click", Node: "n1"}, Snapshot: snapshot.Run.ID})
	if e == nil || p.calls != calls {
		t.Fatal("tampered snapshot reached provider")
	}
}
func TestBrowserLifecycleGuards(t *testing.T) {
	for _, state := range []string{"quarantined", "released", "expired", "unfinished", "dead", "uncertain"} {
		t.Run(state, func(t *testing.T) {
			s, l, p, process := browserFixture(t)
			ctx := context.Background()
			switch state {
			case "quarantined":
				l.Observed = "quarantined"
			case "released":
				l.Desired = "released"
				l.Observed = "released"
			case "expired":
				l.ExpiresAt = time.Now().Add(-time.Second)
			case "unfinished":
				if e := s.Store.SaveRun(ctx, domain.CommandRun{ID: newID(), LeaseID: l.ID, Name: "browser-click", Argv: []string{"browser", "click"}, Status: "running", StartedAt: time.Now()}); e != nil {
					t.Fatal(e)
				}
			case "dead":
				delete(process.alive, l.ID+":api")
			case "uncertain":
				process.inspectErr = errors.New("identity unavailable")
			}
			if state == "quarantined" || state == "released" || state == "expired" {
				if e := s.Store.Save(ctx, l); e != nil {
					t.Fatal(e)
				}
			}
			_, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
			if e == nil || p.calls != 0 || process.starts != 1 {
				t.Fatal("unsafe browser attach or restart")
			}
		})
	}
}
func TestBrowserUncertainMutationRetainsCleanupBarrier(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	ctx := context.Background()
	p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
		return domain.BrowserObservation{ActionPerformed: true, Confirmed: false}, errors.New("connection lost")
	}
	result, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/"}})
	if e == nil || result.Run.Status != "running" {
		t.Fatal("uncertain input finalized")
	}
	_, e = s.Destroy(ctx, l.ID, true, false)
	if e == nil {
		t.Fatal("destroy bypassed uncertain browser operation")
	}
	calls := p.calls
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}})
	if e == nil || p.calls != calls {
		t.Fatal("unfinished operation replayed")
	}
}
func TestBrowserRejectedMutationDoesNotPoisonLease(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	ctx := context.Background()
	p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
		return domain.BrowserObservation{Confirmed: true}, errors.New("unsupported page")
	}
	result, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/"}})
	if e == nil || result.Run.Status != "failed" {
		t.Fatal("rejected request did not finalize")
	}
	p.fn = nil
	if _, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}}); e != nil {
		t.Fatal(e)
	}
}
func TestBrowserEvidenceFailureRetainsBarrier(t *testing.T) {
	s, l, _, _ := browserFixture(t)
	ctx := context.Background()
	directory := filepath.Join(s.Home, "leases", l.ID, "artifacts")
	if e := os.MkdirAll(filepath.Dir(directory), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(directory, []byte("block artifact directory"), 0600); e != nil {
		t.Fatal(e)
	}
	_, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}})
	if e == nil {
		t.Fatal("evidence error ignored")
	}
	runs, e := s.Store.Runs(ctx, l.ID)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, run := range runs {
		found = found || run.Status == "running"
	}
	if !found {
		t.Fatal("evidence failure lost run barrier")
	}
}

func TestBrowserPriorTextRedactionPreservesAuthority(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	ctx := context.Background()
	snap, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "set-text", Text: "web", Node: "n1"}, Snapshot: snap.Run.ID})
	if e != nil {
		t.Fatal(e)
	}
	p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
		return domain.BrowserObservation{Confirmed: true, Snapshot: &domain.BrowserSnapshot{Page: domain.BrowserPage{ID: "page"}, Nodes: []domain.BrowserNode{{Ref: "n1", BackendID: 1, Fingerprint: "digest", Name: "web echoed"}}}, Console: []domain.BrowserConsole{{Text: "web echoed"}}}, nil
	}
	// A provider receives the already verified runtime; supply its exact identity for this fixture.
	identity := snap.Observation.Identity
	fn := p.fn
	p.fn = func(c context.Context, q domain.BrowserRequest) (domain.BrowserObservation, error) {
		v, e := fn(c, q)
		v.Identity = identity
		return v, e
	}
	next, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
	if e != nil {
		t.Fatal(e)
	}
	if next.Snapshot.Browser != "web" || next.Snapshot.Nodes[0].Name != "[REDACTED] echoed" || next.Observation.Console[0].Text != "[REDACTED] echoed" {
		t.Fatalf("bad selective redaction: %+v", next)
	}
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "click", Node: "n1"}, Snapshot: next.Run.ID})
	if e != nil {
		t.Fatal(e)
	}
	artifacts, e := s.Store.Artifacts(ctx, l.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range artifacts {
		if a.Kind == "browser-redaction" {
			if e = os.WriteFile(a.Path, []byte("{}"), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	calls := p.calls
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}})
	if e == nil || p.calls != calls {
		t.Fatal("corrupt redaction evidence ignored")
	}
}
func TestBrowserLockLossDoesNotExposeObservation(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	base := s.Store
	var lose context.CancelCauseFunc
	s.Store = &uiLostFenceStore{Store: base, lose: &lose}
	p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
		lose(domain.ErrLockLost)
		return domain.BrowserObservation{Detail: "private secret", Console: []domain.BrowserConsole{{Text: "private secret"}}, DOM: []byte("private secret")}, nil
	}
	result, e := s.Browser(context.Background(), l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}})
	if !errors.Is(e, domain.ErrLockLost) {
		t.Fatalf("expected fence loss: %v", e)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "private secret") || len(result.Observation.DOM) > 0 {
		t.Fatal("unredacted observation returned after lock loss")
	}
	runs, e := base.Runs(context.Background(), l.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(runs) != 1 || runs[0].Status != "running" {
		t.Fatal("lock lost run incorrectly finalized")
	}
}

func TestBrowserMutationFenceBlocksDestroy(t *testing.T) {
	s, l, p, process := browserFixture(t)
	entered, finish := make(chan struct{}), make(chan struct{})
	p.fn = func(ctx context.Context, q domain.BrowserRequest) (domain.BrowserObservation, error) {
		close(entered)
		select {
		case <-finish:
			return domain.BrowserObservation{Confirmed: true, ActionPerformed: true}, nil
		case <-ctx.Done():
			return domain.BrowserObservation{}, ctx.Err()
		}
	}
	done := startFixtureOperation(t, context.Background(), func(ctx context.Context) error {
		_, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/"}})
		return err
	})
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("browser operation returned before entering provider: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("browser operation did not enter provider")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	_, err := s.Destroy(ctx, l.ID, true, false)
	cancel()
	if err == nil || process.destroys != 0 {
		t.Error("destroy crossed active browser fence")
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Destroy(context.Background(), l.ID, false, false); err != nil {
		t.Fatal(err)
	}
}
func TestBrowserDegradedAllowsOnlyDiagnostics(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	l.Observed = "degraded"
	ctx := context.Background()
	if e := s.Store.Save(ctx, l); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "pages"}}); e != nil {
		t.Fatal(e)
	}
	calls := p.calls
	if _, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/"}}); e == nil || p.calls != calls {
		t.Fatal("degraded input allowed")
	}
}

func TestBrowserNavigateInvalidatesSameDocumentSnapshot(t *testing.T) {
	s, l, p, _ := browserFixture(t)
	ctx := context.Background()
	snap, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/#fragment"}})
	if e != nil {
		t.Fatal(e)
	}
	calls := p.calls
	_, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "click", Node: "n1"}, Snapshot: snap.Run.ID})
	if e == nil || p.calls != calls {
		t.Fatal("pre-navigation snapshot authorized input")
	}
}

func TestBrowserURLWaitRequiresSubstring(t *testing.T) {
	for _, tc := range []struct {
		name, contains, role string
		valid                bool
	}{
		{"missing", "", "", false},
		{"role-only", "", "heading", false},
		{"role-with-substring", "/ready", "heading", false},
		{"substring", "/ready", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "wait", WaitFor: "url", Contains: tc.contains, Role: tc.role}}
			if err := validateBrowserOptions(&o); (err == nil) != tc.valid {
				t.Fatalf("URL predicate validation: %v, valid=%t", err, tc.valid)
			}
			if !tc.valid {
				// Invalid URL predicates must fail before lease lookup or CDP attachment.
				_, err := (&Service{}).Browser(context.Background(), "lease", o)
				if err == nil || !strings.Contains(err.Error(), "AGENTENV-BROWSER-INPUT") {
					t.Fatalf("invalid URL wait reached service dependencies: %v", err)
				}
			}
		})
	}
}

func TestBrowserSemanticProvenanceSurvivesUncertainInput(t *testing.T) {
	for _, operation := range []string{"click", "set-text", "key", "scroll"} {
		for _, uncertain := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/uncertain=%t", operation, uncertain), func(t *testing.T) {
				s, l, p, _ := browserFixture(t)
				ctx := context.Background()
				snapshot, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
				if err != nil {
					t.Fatal(err)
				}
				const secret = "provenance-secret-日本語"
				q := domain.BrowserRequest{Operation: operation, Node: "n1"}
				if operation == "set-text" {
					q.Text = secret
				}
				if operation == "key" {
					q.Key = "Enter"
				}
				if operation == "scroll" {
					q.DeltaY = 123
				}
				check := func(run domain.CommandRun) {
					t.Helper()
					for flag, want := range map[string]string{"--browser": "web", "--page": snapshot.Snapshot.Page.ID, "--snapshot": snapshot.Run.ID, "--node": "n1"} {
						found := false
						for i := 0; i+1 < len(run.Argv); i++ {
							if run.Argv[i] == flag && run.Argv[i+1] == want {
								found = true
							}
						}
						if !found {
							t.Fatalf("missing provenance %s=%s in %v", flag, want, run.Argv)
						}
					}
					data, err := json.Marshal(run)
					if err != nil || strings.Contains(string(data), secret) {
						t.Fatal("run contains input or invalid JSON")
					}
				}
				p.fn = func(ctx context.Context, request domain.BrowserRequest) (domain.BrowserObservation, error) {
					runs, err := s.Store.Runs(ctx, l.ID)
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, run := range runs {
						if run.Name == "browser-"+operation && run.Status == "running" {
							check(run)
							found = true
						}
					}
					if !found {
						t.Fatal("input happened before intent and provenance persisted")
					}
					if uncertain {
						return domain.BrowserObservation{ActionPerformed: true}, errors.New("CDP disconnected after input")
					}
					return domain.BrowserObservation{Confirmed: true, ActionPerformed: true}, nil
				}
				result, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: q, Snapshot: snapshot.Run.ID})
				if (err != nil) != uncertain {
					t.Fatalf("unexpected action outcome: %v", err)
				}
				check(result.Run)
				wantStatus := "passed"
				if uncertain {
					wantStatus = "running"
				}
				if result.Run.Status != wantStatus {
					t.Fatalf("status=%s", result.Run.Status)
				}
				found := false
				for _, a := range result.Artifacts {
					if a.Kind != "browser-result" {
						continue
					}
					data, err := os.ReadFile(a.Path)
					if err != nil {
						t.Fatal(err)
					}
					var saved BrowserResult
					if err = json.Unmarshal(data, &saved); err != nil {
						t.Fatal(err)
					}
					check(saved.Run)
					found = true
				}
				if !found {
					t.Fatal("run.json missing")
				}
			})
		}
	}
}
