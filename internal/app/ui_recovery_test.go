package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

func TestUIRecoverInterruptedHelperPreservesFailedOutcome(t *testing.T) {
	s, db, l, p := uiFixture(t)
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{}, errors.New("remote completion lost")
	}
	initial, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if err == nil {
		t.Fatal("expected interrupted snapshot")
	}
	before, err := db.Artifacts(context.Background(), l.ID)
	if err != nil || len(before) != 1 {
		t.Fatalf("original evidence %v %v", before, err)
	}
	original, err := os.ReadFile(before[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	p.observe = func(_ context.Context, q domain.UIRequest) (domain.UIObservation, error) {
		if q.Operation != "quiesce" {
			t.Fatal("recovery retried original input")
		}
		return domain.UIObservation{Status: "ok", Confirmed: true, Backend: "helper-digest"}, nil
	}
	l.Diagnostics = []string{"stale command run " + initial.Run.ID + " remains running in registry; process completion is unverified and GC is blocked", "unrelated diagnostic"}
	if err = db.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.RecoverUI(context.Background(), l.ID, initial.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Run.Status != "failed" || recovered.Run.FinishedAt.IsZero() || len(recovered.Artifacts) != 1 || recovered.Artifacts[0].Kind != "ui-recovery" {
		t.Fatalf("recovery %+v", recovered)
	}
	unchanged, err := os.ReadFile(before[0].Path)
	if err != nil || string(unchanged) != string(original) {
		t.Fatal("original evidence changed")
	}
	runs, err := db.Runs(context.Background(), l.ID)
	if err != nil || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatalf("barrier not finalized %v %v", runs, err)
	}
	stored, e := db.Get(context.Background(), l.ID)
	if e != nil || len(stored.Diagnostics) != 1 || stored.Diagnostics[0] != "unrelated diagnostic" {
		t.Fatalf("stale diagnostic not cleared narrowly: %v %v", stored.Diagnostics, e)
	}
	calls := p.calls
	if _, err = s.RecoverUI(context.Background(), l.ID, initial.Run.ID); err == nil || p.calls != calls {
		t.Fatal("terminal run recovered again")
	}
}
func TestUIRecoverRefusesUnsafeIntentAndEvidence(t *testing.T) {
	for _, mode := range []string{"native", "wrong-serial", "missing-evidence", "digest", "evidence-incomplete", "other-run"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				return domain.UIObservation{}, errors.New("interrupted")
			}
			initial, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if err == nil {
				t.Fatal("expected interruption")
			}
			run := initial.Run
			switch mode {
			case "native":
				run.Name = "ui-back"
				run.Argv[1] = "back"
			case "wrong-serial":
				for i, n := range run.Notes {
					if strings.HasPrefix(n, "Android serial: ") {
						run.Notes[i] = "Android serial: emulator-5556"
					}
				}
			case "missing-evidence":
				s.Store = &uiWithoutArtifacts{Store: s.Store}
			case "digest":
				artifacts, _ := db.Artifacts(context.Background(), l.ID)
				if err = os.WriteFile(artifacts[0].Path, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "evidence-incomplete":
				run.Notes = append(run.Notes, "UI recovery classification: evidence-incomplete")
			case "other-run":
				other := run
				other.ID = newID()
				if err = db.SaveRun(context.Background(), other); err != nil {
					t.Fatal(err)
				}
			}
			if err = db.SaveRun(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			calls := p.calls
			if _, err = s.RecoverUI(context.Background(), l.ID, run.ID); err == nil || p.calls != calls {
				t.Fatalf("unsafe recovery %v calls=%d", err, p.calls)
			}
		})
	}
}

type uiWithoutArtifacts struct{ Store }

func (*uiWithoutArtifacts) Artifacts(context.Context, string) ([]domain.Artifact, error) {
	return nil, nil
}
func TestUIRecoverFailureRetainsBarrier(t *testing.T) {
	for _, mode := range []string{"quiesce", "evidence"} {
		t.Run(mode, func(t *testing.T) {
			s, db, l, p := uiFixture(t)
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				return domain.UIObservation{}, errors.New("interrupted")
			}
			initial, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if err == nil {
				t.Fatal("expected interruption")
			}
			if mode == "evidence" {
				s.Store = &failingUIArtifactStore{Store: s.Store}
				p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
					return domain.UIObservation{Status: "ok", Confirmed: true}, nil
				}
			}
			if _, err = s.RecoverUI(context.Background(), l.ID, initial.Run.ID); err == nil {
				t.Fatal("failed recovery succeeded")
			}
			runs, err := db.Runs(context.Background(), l.ID)
			if err != nil || len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("barrier cleared %v %v", runs, err)
			}
		})
	}
}

func TestUIRecoverRefusesUnpersistedFailureClassification(t *testing.T) {
	s, db, l, p := uiFixture(t)
	s.Store = &failingFinalRunStore{Store: s.Store}
	p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
		return domain.UIObservation{Status: "uncertain", Confirmed: true}, execx.ErrProcessTreeUnconfirmed
	}
	result, err := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
	if err == nil || p.calls != 1 {
		t.Fatalf("expected unconfirmed host failure, err=%v calls=%d", err, p.calls)
	}
	runs, err := db.Runs(context.Background(), l.ID)
	if err != nil || len(runs) != 1 || runs[0].Status != "running" {
		t.Fatalf("runs %v %v", runs, err)
	}
	for _, note := range runs[0].Notes {
		if strings.HasPrefix(note, "UI recovery classification:") {
			t.Fatal("fixture did not fail classification persistence")
		}
	}
	artifacts, err := db.Artifacts(context.Background(), l.ID)
	if err != nil || len(artifacts) != 1 || artifacts[0].Kind != "ui-result" {
		t.Fatalf("result must precede failed classification: %v %v", artifacts, err)
	}
	s.Store = db
	if _, err = s.RecoverUI(context.Background(), l.ID, result.Run.ID); err == nil || !strings.Contains(err.Error(), "lacks durable termination-unconfirmed eligibility") || p.calls != 1 {
		t.Fatalf("unclassified host uncertainty recovered: %v calls=%d", err, p.calls)
	}
}
