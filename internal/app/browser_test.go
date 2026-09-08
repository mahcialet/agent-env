package app

import (
	"context"
	"encoding/json"
	"errors"
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
func browserFixture(t *testing.T) (*Service, domain.Lease, *fixtureBrowser, *lifecycleProcess) {
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
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
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
	done := make(chan error, 1)
	go func() {
		_, err := s.Browser(context.Background(), l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "navigate", URL: "https://example.invalid/"}})
		done <- err
	}()
	<-entered
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
