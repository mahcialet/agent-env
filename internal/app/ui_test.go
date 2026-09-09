package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

type fakeUIProvider struct {
	calls   int
	request domain.UIRequest
	runtime domain.Runtime
	observe func(context.Context, domain.UIRequest) (domain.UIObservation, error)
}

func (f *fakeUIProvider) ObserveUI(ctx context.Context, r domain.Runtime, q domain.UIRequest) (domain.UIObservation, error) {
	f.calls++
	f.request = q
	f.runtime = r
	if f.observe != nil {
		return f.observe(ctx, q)
	}
	return domain.UIObservation{Version: 1, Status: "ok", Backend: "test-v1", Confirmed: true, ActionPerformed: true, Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "fingerprint", Text: "Ready", Clickable: true, Visible: true}}}}, nil
}
func uiFixture(t *testing.T, management ...*domain.Management) (*Service, *sqlite.Store, domain.Lease, *fakeUIProvider) {
	t.Helper()
	s, db, l := commandFixture(t, management...)
	l.ID = newID()
	l.Sources = nil
	avdHome := filepath.Join(s.Home, "leases", l.ID, "avd")
	l.Runtimes = []domain.Runtime{{Name: "phone", Type: "android-emulator", LeaseID: l.ID, Android: &domain.AndroidEmulator{Template: "fixture", AVDName: "fixture", AVDHome: avdHome, AVDPath: filepath.Join(avdHome, "fixture.avd")}}}
	l.Applications = []domain.Application{{Name: "mobile", Runtime: "phone", Package: "com.example.mobile"}}
	if err := db.Reserve(context.Background(), l, 0); err != nil {
		t.Fatal(err)
	}
	var err error
	l, err = db.Get(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	l.Runtimes[0].Started = true
	if err := db.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	p := &fakeUIProvider{}
	s.AndroidUI = p
	return s, db, l, p
}

func TestUISnapshotAndSemanticScope(t *testing.T) {
	s, db, l, p := uiFixture(t)
	ctx := context.Background()
	snap, err := s.UI(ctx, l.ID, UIOptions{Operation: "snapshot"})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Snapshot == nil || snap.Snapshot.Serial != "emulator-5554" || len(snap.Artifacts) != 2 {
		t.Fatalf("snapshot %+v", snap)
	}
	result, err := s.UI(ctx, l.ID, UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "n1"})
	if err != nil {
		t.Fatal(err)
	}
	if p.request.ExpectedFingerprint != "fingerprint" || p.runtime.LeaseID != l.ID || result.Run.Status != "passed" {
		t.Fatalf("wrong action %+v %+v", p.request, result)
	}
	runs, err := db.Runs(ctx, l.ID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs %v %v", runs, err)
	}
}
func TestUIRejectsInvalidStateAndSelectionBeforeDevice(t *testing.T) {
	for _, mode := range []string{"expired", "quarantined", "degraded-mutation", "cleanup", "multiple", "foreign", "running", "application", "conflicting"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			o := UIOptions{Operation: "snapshot"}
			switch mode {
			case "expired":
				l.ExpiresAt = time.Now().Add(-time.Second)
			case "quarantined":
				l.Observed = "quarantined"
			case "degraded-mutation":
				l.Observed = "degraded"
				o.Operation = "back"
			case "cleanup":
				l.Desired = "released"
			case "multiple":
				r := l.Runtimes[0]
				r.Name = "second"
				l.Runtimes = append(l.Runtimes, r)
			case "foreign":
				l.Runtimes[0].LeaseID = "other"
			case "running":
				if err := db.SaveRun(context.Background(), domain.CommandRun{ID: newID(), LeaseID: l.ID, Status: "running", StartedAt: time.Now()}); err != nil {
					t.Fatal(err)
				}
			case "application":
				o.Application = "missing"
			case "conflicting":
				o.Application = "app"
				o.Runtime = "phone"
			}
			s.Store = &uiLeaseView{Store: db, lease: l}
			if _, err := s.UI(context.Background(), l.ID, o); err == nil {
				t.Fatal("invalid state accepted")
			}
			if p.calls != 0 {
				t.Fatal("device called on rejected request")
			}
		})
	}
}
func TestUIRejectsSnapshotTamperingAndAmbiguity(t *testing.T) {
	for _, mode := range []string{"digest", "foreign", "missing-node", "duplicate", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			if mode == "duplicate" || mode == "truncated" {
				p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
					return domain.UIObservation{Status: "ok", Confirmed: true, Snapshot: domain.UITree{Truncated: mode == "truncated", Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "same"}, {Ref: "n2", Fingerprint: "same"}}}}, nil
				}
			}
			snap, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if err != nil {
				t.Fatal(err)
			}
			id, node := snap.Snapshot.ID, "n1"
			switch mode {
			case "digest":
				for _, a := range snap.Artifacts {
					if a.Kind == "ui-snapshot" {
						if err := os.WriteFile(a.Path, []byte("{}"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
			case "foreign":
				id = newID()
			case "missing-node":
				node = "n999"
			}
			if _, err = s.UI(context.Background(), l.ID, UIOptions{Operation: "tap", Snapshot: id, Node: node}); err == nil {
				t.Fatal("unsafe snapshot accepted")
			}
			if p.calls != 1 {
				t.Fatal("unsafe input sent")
			}
			runs, _ := db.Runs(context.Background(), l.ID)
			if len(runs) != 1 {
				t.Fatal("rejected action saved intent")
			}
		})
	}
}
func TestUIEvidenceRedactsEnteredAndEditableText(t *testing.T) {
	s, db, l, p := uiFixture(t)
	secret := "日本語秘密🔑"
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Version: 1, Status: "ok", Confirmed: true, Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "f", Editable: true, Text: secret, Description: "field"}}}, Raw: []byte(`{"status":"ok","snapshot":{"nodes":[{"editable":true,"text":"日本語秘密🔑"}]},"unexpected":"日本語秘密🔑"}`)}, nil
	}
	snap, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	p.observe = func(_ context.Context, q domain.UIRequest) (domain.UIObservation, error) {
		called = true
		if q.Text != secret {
			t.Fatal("text payload altered")
		}
		return domain.UIObservation{Confirmed: true, Status: "refused", Detail: secret}, errors.New(secret)
	}
	result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "set-text", Snapshot: snap.Snapshot.ID, Node: "n1", Text: secret})
	if !called || err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked %v", err)
	}
	encoded, _ := json.Marshal(result)
	if bytes.Contains(encoded, []byte(secret)) {
		t.Fatal("result leaked")
	}
	artifacts, _ := db.Artifacts(context.Background(), l.ID)
	for _, a := range artifacts {
		data, e := os.ReadFile(a.Path)
		if e != nil {
			t.Fatal(e)
		}
		if bytes.Contains(data, []byte(secret)) {
			t.Fatalf("artifact leaked %s", a.Kind)
		}
	}
	runs, _ := db.Runs(context.Background(), l.ID)
	encoded, _ = json.Marshal(runs)
	if bytes.Contains(encoded, []byte(secret)) {
		t.Fatal("registry leaked")
	}
}
func TestUIUnconfirmedKeepsCleanupBarrier(t *testing.T) {
	s, db, l, p := uiFixture(t)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Status: "uncertain"}, errors.New("lost device")
	}
	if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "back"}); err == nil {
		t.Fatal("uncertain success")
	}
	runs, _ := db.Runs(context.Background(), l.ID)
	if len(runs) != 1 || runs[0].Status != "running" {
		t.Fatalf("barrier cleared %v", runs)
	}
	if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"}); err == nil || p.calls != 1 {
		t.Fatal("unfinished operation retried")
	}
}
func TestUIEvidenceFailureKeepsCleanupBarrier(t *testing.T) {
	s, db, l, _ := uiFixture(t)
	s.Store = &failingUIArtifactStore{Store: s.Store}
	if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"}); err == nil {
		t.Fatal("evidence failure ignored")
	}
	runs, _ := db.Runs(context.Background(), l.ID)
	if len(runs) != 1 || runs[0].Status != "running" {
		t.Fatal("evidence barrier cleared")
	}
}

type failingUIArtifactStore struct{ Store }

func (*failingUIArtifactStore) SaveArtifact(context.Context, domain.Artifact) error {
	return errors.New("artifact registry unavailable")
}
func TestUIPNGValidation(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	w, h, err := validateUIPNG(b.Bytes())
	if err != nil || w != 2 || h != 3 {
		t.Fatalf("PNG %d %d %v", w, h, err)
	}
	for _, data := range [][]byte{b.Bytes()[:len(b.Bytes())-5], append(append([]byte{}, b.Bytes()...), 1), make([]byte, 17<<20)} {
		if _, _, err := validateUIPNG(data); err == nil {
			t.Fatal("invalid PNG accepted")
		}
	}
}
func TestUIWaitIsBoundedAndDoesNotInput(t *testing.T) {
	s, _, l, p := uiFixture(t)
	result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "wait", Application: "mobile", Contains: "never appears", Timeout: 20 * time.Millisecond})
	if err == nil || p.calls != 1 || p.request.Operation != "snapshot" || result.Run.Status != "failed" {
		t.Fatalf("wait %+v %v calls=%d", result, err, p.calls)
	}
}

type uiLeaseView struct {
	Store
	lease domain.Lease
}

func (s *uiLeaseView) Get(context.Context, string) (domain.Lease, error) { return s.lease, nil }

func TestUIOperationFencePreventsDestroyRace(t *testing.T) {
	s, db, l, p := uiFixture(t)
	entered, finish := make(chan struct{}), make(chan struct{})
	p.observe = func(ctx context.Context, _ domain.UIRequest) (domain.UIObservation, error) {
		close(entered)
		select {
		case <-finish:
			return domain.UIObservation{Status: "ok", Confirmed: true, ActionPerformed: true}, nil
		case <-ctx.Done():
			return domain.UIObservation{}, ctx.Err()
		}
	}
	done := make(chan error, 1)
	go func() { _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "back"}); done <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("UI did not start")
	}
	if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"}); !errors.Is(err, sqlite.ErrBusy) {
		t.Errorf("concurrent UI not fenced: %v", err)
	}
	destroyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	_, err := s.Destroy(destroyCtx, l.ID, false, false)
	cancel()
	if err == nil {
		t.Error("destroy passed in-flight UI fence")
	}
	stored, e := db.Get(context.Background(), l.ID)
	if e != nil || stored.Desired != "active" || stored.Observed != "ready" {
		t.Errorf("destroy altered lease before acquiring fence: %+v %v", stored, e)
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 1 || runs[0].Status != "passed" {
		t.Fatalf("run %v %v", runs, e)
	}
}
func TestUIDegradedDiagnosticsAndApplicationScope(t *testing.T) {
	s, db, l, p := uiFixture(t)
	l.Observed = "degraded"
	l.Applications = []domain.Application{{Name: "mobile", Runtime: "phone", Package: "com.example.mobile"}}
	if err := db.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	snap, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot", Application: "mobile"})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Snapshot.Package != "com.example.mobile" || p.request.Package != "com.example.mobile" {
		t.Fatal("application scope lost")
	}
	snap, err = s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot", Application: "mobile", AllWindows: true})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Snapshot.Package != "" || p.request.Package != "" {
		t.Fatal("all-windows scope ignored")
	}
}

func TestUIBackendRefusalsRetainStableDiagnostics(t *testing.T) {
	for _, status := range []string{"stale", "ambiguous", "unavailable"} {
		t.Run(status, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				return domain.UIObservation{Status: status, Confirmed: true}, nil
			}
			_, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if err == nil || !strings.Contains(err.Error(), "AGENTENV-UI-"+strings.ToUpper(status)) {
				t.Fatalf("diagnostic %v", err)
			}
			runs, _ := db.Runs(context.Background(), l.ID)
			if len(runs) != 1 || runs[0].Status != "failed" {
				t.Fatal("known refusal did not finalize")
			}
		})
	}
}
func TestUISnapshotSymlinkRefused(t *testing.T) {
	s, _, l, p := uiFixture(t)
	snap, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if err != nil {
		t.Fatal(err)
	}
	var path string
	for _, a := range snap.Artifacts {
		if a.Kind == "ui-snapshot" {
			path = a.Path
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "snapshot.json")
	if err = os.WriteFile(outside, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err = s.UI(context.Background(), l.ID, UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "n1"}); err == nil || p.calls != 1 {
		t.Fatal("symlink became input")
	}
}

func TestUIContainsRequiresVisibleNode(t *testing.T) {
	tree := domain.UITree{Nodes: []domain.UINode{{Text: "Ready", Visible: false}, {Description: "Ready", Visible: true, Editable: true}, {Hint: "Ready", Visible: true, Password: true}}}
	if uiContains(tree, "Ready") {
		t.Fatal("invisible or sensitive node satisfied wait")
	}
	tree.Nodes = append(tree.Nodes, domain.UINode{Text: "Ready", Visible: true})
	if !uiContains(tree, "Ready") {
		t.Fatal("visible label did not satisfy wait")
	}
}

func TestUIHelperRecoveryAfterInternalTimeout(t *testing.T) {
	for _, mode := range []string{"confirmed", "failed", "parent-canceled", "native"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p.observe = func(callCtx context.Context, q domain.UIRequest) (domain.UIObservation, error) {
				if q.Operation == "quiesce" {
					if mode != "confirmed" && mode != "failed" {
						t.Fatal("unexpected helper cleanup")
					}
					if callCtx.Err() != nil {
						t.Fatal("recovery inherited request deadline")
					}
					if mode == "confirmed" {
						return domain.UIObservation{Status: "ok", Confirmed: true}, nil
					}
					return domain.UIObservation{}, errors.New("helper remains live")
				}
				if mode == "parent-canceled" {
					cancel()
				}
				<-callCtx.Done()
				return domain.UIObservation{}, callCtx.Err()
			}
			operation := "wait"
			if mode == "native" {
				operation = "back"
			}
			result, err := s.UI(ctx, l.ID, UIOptions{Operation: operation, Application: "mobile", Contains: "Ready", Timeout: time.Millisecond})
			if err == nil {
				t.Fatal("interrupted operation incorrectly succeeded")
			}
			runs, e := db.Runs(context.Background(), l.ID)
			if e != nil || len(runs) != 1 {
				t.Fatalf("runs %v %v", runs, e)
			}
			if mode == "confirmed" {
				if result.Run.Status != "failed" || runs[0].Status != "failed" || p.calls != 2 {
					t.Fatalf("recovery %+v %v calls=%d", result, runs, p.calls)
				}
			} else {
				if runs[0].Status != "running" {
					t.Fatal("unconfirmed barrier cleared")
				}
				if (mode == "parent-canceled" || mode == "native") && p.calls != 1 {
					t.Fatal("unauthorized recovery")
				}
			}
		})
	}
}

func TestUIRecoveryRefusesLostOrRejectedFence(t *testing.T) {
	for _, mode := range []string{"lost", "rejected"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			var lose context.CancelCauseFunc
			if mode == "lost" {
				s.Store = &uiLostFenceStore{Store: s.Store, lose: &lose}
			} else {
				s.Store = &failingFinalRunStore{Store: s.Store}
			}
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				if mode == "lost" {
					lose(domain.ErrLockLost)
				}
				return domain.UIObservation{}, errors.New("interrupted helper")
			}
			if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"}); err == nil {
				t.Fatal("missing failure")
			}
			if p.calls != 1 {
				t.Fatal("recovery sent without confirmed operation fence")
			}
			runs, e := db.Runs(context.Background(), l.ID)
			if e != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("barrier %v %v", runs, e)
			}
		})
	}
}

type uiLostFenceStore struct {
	Store
	lose *context.CancelCauseFunc
}

func (s *uiLostFenceStore) AcquireContext(ctx context.Context, id, token string, ttl time.Duration) (context.Context, func() error, error) {
	op, release, err := s.Store.AcquireContext(ctx, id, token, ttl)
	if err != nil {
		return nil, nil, err
	}
	op, cancel := context.WithCancelCause(op)
	*s.lose = cancel
	return op, func() error { cancel(nil); return release() }, nil
}

func TestUIInvalidCompletedCaptureFinalizesFailed(t *testing.T) {
	s, db, l, p := uiFixture(t)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Status: "ok", Confirmed: true, Binary: []byte("not a PNG")}, nil
	}
	result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "screenshot"})
	if err == nil || result.Run.Status != "failed" {
		t.Fatalf("invalid capture %+v %v", result, err)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatalf("invalid response kept termination barrier: %v %v", runs, e)
	}
	artifacts, e := db.Artifacts(context.Background(), l.ID)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range artifacts {
		if a.Kind == "ui-screenshot" {
			t.Fatal("invalid PNG persisted")
		}
	}
}
func TestUIEditableDescriptionAndHintRedacted(t *testing.T) {
	observation := domain.UIObservation{Snapshot: domain.UITree{Nodes: []domain.UINode{{Editable: true, Text: "private", Description: "private", Hint: "private"}, {Password: true, Description: "private", Hint: "private"}}}, Raw: []byte(`{"snapshot":{"nodes":[{"editable":true,"description":"private","hint":"private"}]}}`)}
	sanitizeUIObservation(&observation, nil)
	data, _ := json.Marshal(observation)
	if bytes.Contains(data, []byte("private")) || bytes.Contains(observation.Raw, []byte("private")) {
		t.Fatal("editable/password labels leaked")
	}
}

func TestUIHostUnconfirmedCannotBeQuiescedAway(t *testing.T) {
	for _, marker := range []error{execx.ErrProcessTreeUnconfirmed, execx.ErrOutputIncomplete} {
		t.Run(marker.Error(), func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				return domain.UIObservation{Status: "uncertain", Confirmed: true}, marker
			}
			result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if err == nil || p.calls != 1 {
				t.Fatal("host uncertainty recovered via helper")
			}
			runs, e := db.Runs(context.Background(), l.ID)
			if e != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("host barrier cleared %v %v", runs, e)
			}
			if _, err = s.RecoverUI(context.Background(), l.ID, result.Run.ID); err == nil || p.calls != 1 {
				t.Fatal("explicit helper recovery waived host process barrier")
			}
		})
	}
}

func TestUIPrerequisiteErrorSurvivesPrivacySanitization(t *testing.T) {
	s, db, l, p := uiFixture(t)
	secret := "private-observer-location"
	t.Setenv("AGENT_ENV_OBSERVER_TOKEN", secret)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Status: "unavailable", Confirmed: true}, errors.Join(ErrPrerequisite, errors.New("helper missing: "+secret))
	}
	_, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if !errors.Is(err, ErrPrerequisite) || strings.Contains(err.Error(), secret) || p.calls != 1 {
		t.Fatalf("prerequisite classification/privacy lost: %v calls=%d", err, p.calls)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatalf("known prerequisite failure retained barrier: %v %v", runs, e)
	}
}

func TestUILogcatReturnsBoundedRedactedInlineEvidence(t *testing.T) {
	s, db, l, p := uiFixture(t)
	l.Applications = []domain.Application{{Name: "mobile", Runtime: "phone", Package: "com.example.mobile"}}
	if err := db.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	secret := "log-private-value"
	t.Setenv("AGENT_ENV_LOG_TOKEN", secret)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Status: "ok", Confirmed: true, Binary: []byte(secret + "\n" + strings.Repeat("line\n", 60000)), Log: secret}, nil
	}
	result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "logcat", Application: "mobile"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observation.Log) != 256<<10 || strings.Contains(result.Observation.Log, secret) || !strings.Contains(result.Observation.Log, "[REDACTED]") || !result.Observation.Truncated {
		t.Fatal("inline log bound/redaction lost")
	}
	for _, a := range result.Artifacts {
		if a.Kind == "ui-logcat" {
			data, e := os.ReadFile(a.Path)
			if e != nil || string(data) != result.Observation.Log {
				t.Fatal("inline and artifact logs differ")
			}
		}
	}
	observation := domain.UIObservation{Log: secret + strings.Repeat("x", 300000)}
	sanitizeUIObservation(&observation, []string{secret})
	if len(observation.Log) > 256<<10 || strings.Contains(observation.Log, secret) {
		t.Fatal("provider supplied Log bypassed sanitation")
	}
}
func TestUICoordinateRangeRejectsBeforeProvider(t *testing.T) {
	s, _, l, p := uiFixture(t)
	if _, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "tap-coordinate", X: 32768}); !errors.Is(err, domain.ErrUIInput) || p.calls != 0 {
		t.Fatalf("range failure %v calls=%d", err, p.calls)
	}
}
