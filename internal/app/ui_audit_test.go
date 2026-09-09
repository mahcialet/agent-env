package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestUIAuditEditableSnapshotRemainsActionable(t *testing.T) {
	s, db, l, p := uiFixture(t)
	actionCalls := 0
	p.observe = func(_ context.Context, q domain.UIRequest) (domain.UIObservation, error) {
		if q.Operation == "set-text" {
			actionCalls++
			if q.ExpectedFingerprint != "safe-fingerprint" || q.Text != "safe" {
				t.Fatal("action provenance changed")
			}
			return domain.UIObservation{Status: "ok", Confirmed: true, ActionPerformed: true, ReadbackEqual: true, Backend: "test-v1"}, nil
		}
		return domain.UIObservation{Status: "ok", Confirmed: true, Backend: "test-v1", Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "safe-fingerprint", Editable: true, Enabled: true, Visible: true}}}}, nil
	}
	o, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if e != nil {
		t.Fatal(e)
	}
	if o.Snapshot.Tree.Nodes[0].Fingerprint == "" {
		t.Error("safe editable fingerprint cleared")
	}
	_, e = s.UI(context.Background(), l.ID, UIOptions{Operation: "set-text", Snapshot: o.Snapshot.ID, Node: "n1", Text: "safe"})
	if e != nil {
		t.Errorf("editable input refused before provider: %v", e)
	}
	if actionCalls != 1 {
		t.Fatalf("action callback calls=%d", actionCalls)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 2 {
		t.Fatalf("runs=%v %v", runs, e)
	}
	for _, r := range runs {
		if r.Status != "passed" {
			t.Fatalf("durable run not passed: %+v", r)
		}
	}
}
func TestUIAuditWindowSecretClearsDerivedNodeHashes(t *testing.T) {
	t.Setenv("AUDIT_PRIVATE_SECRET", "private-title")
	s, _, l, p := uiFixture(t)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		o := domain.UIObservation{Status: "ok", Confirmed: true, Snapshot: domain.UITree{Windows: []domain.UIWindow{{Title: "private-title", Key: strings.Repeat("a", 64)}}, Nodes: []domain.UINode{{Ref: "n1", Fingerprint: strings.Repeat("b", 64)}}}}
		o.Raw, _ = json.Marshal(o)
		return o, nil
	}
	o, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if e != nil {
		t.Fatal(e)
	}
	if o.Snapshot.Tree.Nodes[0].Fingerprint != "" {
		t.Error("secret-derived node hash retained while window key cleared")
	}
	if o.Snapshot.Tree.Windows[0].Key != "" {
		t.Fatal("secret-derived window key retained")
	}
	for _, a := range o.Artifacts {
		raw, e := os.ReadFile(a.Path)
		if e != nil {
			t.Fatal(e)
		}
		for _, secret := range []string{"private-title", strings.Repeat("a", 64), strings.Repeat("b", 64)} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("derived secret identifier in %s", a.Kind)
			}
		}
	}
	beforeCalls := p.calls
	_, e = s.UI(context.Background(), l.ID, UIOptions{Operation: "tap", Snapshot: o.Snapshot.ID, Node: "n1"})
	if e == nil || p.calls != beforeCalls {
		t.Fatalf("secret-bearing snapshot authorized input: %v", e)
	}

}
func TestUIAuditFractionalLogLookback(t *testing.T) {
	s, db, l, p := uiFixture(t)
	_, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "logcat", Application: "mobile", Since: 1900 * time.Millisecond})
	if !errors.Is(e, domain.ErrUIInput) || p.calls != 0 {
		t.Fatalf("fractional lookback must fail before provider: calls=%d err=%v", p.calls, e)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 0 {
		t.Fatalf("invalid lookback created durable intent: %v %v", runs, e)
	}
}

func TestUIOpaqueAfterFingerprintIsNotPublishedWithConfiguredSecrets(t *testing.T) {
	t.Setenv("AUDIT_PRIVATE_SECRET", "after-only-secret")
	s, _, l, p := uiFixture(t)
	before, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if e != nil {
		t.Fatal(e)
	}
	hiddenHash := fmt.Sprintf("%x", sha256.Sum256([]byte("after window: after-only-secret")))
	called := false
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		called = true
		o := domain.UIObservation{Status: "ok", Confirmed: true, ActionPerformed: true, AfterFingerprint: hiddenHash}
		o.Raw, _ = json.Marshal(o)
		return o, nil
	}
	result, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "tap", Snapshot: before.Snapshot.ID, Node: "n1"})
	if e != nil || !called {
		t.Fatalf("action callback not completed: %v", e)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), hiddenHash) {
		t.Fatal("opaque after hash published in result")
	}
	for _, a := range result.Artifacts {
		raw, e := os.ReadFile(a.Path)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(raw), hiddenHash) {
			t.Fatalf("opaque after hash in %s", a.Kind)
		}
	}
	safe := domain.UIObservation{AfterFingerprint: "verified-without-secret-context"}
	sanitizeUIObservation(&safe, nil)
	if safe.AfterFingerprint == "" {
		t.Fatal("no-secret after fingerprint unnecessarily omitted")
	}
}
