package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
)

type reviewUIFunc func(context.Context, domain.Runtime, domain.UIRequest) (domain.UIObservation, error)

func (f reviewUIFunc) ObserveUI(ctx context.Context, r domain.Runtime, q domain.UIRequest) (domain.UIObservation, error) {
	return f(ctx, r, q)
}

func TestWorkerCanceledUIRecoveryRetainsRunningUncertainty(t *testing.T) {
	e, created, db, _ := routingUIFixture(t)
	prior := e.Factory
	var cancelRecovery context.CancelFunc
	snapshots, quiesces := 0, 0
	e.Factory = func(m *domain.Management) *app.Service {
		s := prior(m)
		s.AndroidUI = reviewUIFunc(func(ctx context.Context, _ domain.Runtime, q domain.UIRequest) (domain.UIObservation, error) {
			if q.Operation == "snapshot" {
				snapshots++
			} else if q.Operation == "quiesce" {
				quiesces++
				if cancelRecovery != nil {
					cancelRecovery()
					<-ctx.Done()
					return domain.UIObservation{}, ctx.Err()
				}
			} else {
				t.Fatalf("unexpected operation %s", q.Operation)
			}
			return domain.UIObservation{}, errors.New("fixture helper termination unconfirmed")
		})
		return s
	}
	op := nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "snapshot", Application: "mobile"}})
	if err := e.Prepare(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	initial := e.Execute(context.Background(), op)
	var response Response
	if err := json.Unmarshal(initial.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if initial.State != "uncertain" || response.Run == nil || response.Run.Status != "running" {
		t.Fatalf("initial barrier: %s", initial.Payload)
	}
	runID := response.Run.ID
	op = nextOperation(created, "ui-recover", Request{Name: runID})
	if err := e.Prepare(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelRecovery = cancel
	before := quiesces
	result := e.Execute(ctx, op)
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if result.State != "uncertain" || response.Run == nil || response.Run.ID != runID || response.Run.Status != "running" || response.Error == "" {
		t.Fatalf("canceled recovery lost barrier: state=%s payload=%s", result.State, result.Payload)
	}
	if snapshots != 1 || quiesces != before+1 {
		t.Fatalf("replayed input: snapshots=%d quiesces=%d", snapshots, quiesces)
	}
	runs, err := db.Runs(context.Background(), created.LeaseID)
	if err != nil || len(runs) != 1 || runs[0].Status != "running" {
		t.Fatalf("durable barrier: %+v %v", runs, err)
	}
}

func TestWorkerSuccessfulInputSurvivesArtifactStagingFailure(t *testing.T) {
	e, created, db, p := routingUIFixture(t)
	snap := routingUISnapshot(t, e, created)
	op := nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "n1"}})
	if err := e.Prepare(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	// Move only this test's CAS after preparation. Local action/evidence writes
	// remain healthy, but publication must fail even when tests run as root.
	casPath := filepath.Join(e.Home, "blobs")
	backup := casPath + "-offline"
	if err := os.Rename(casPath, backup); err != nil {
		t.Fatal(err)
	}
	result := e.Execute(context.Background(), op)
	if err := os.Rename(backup, casPath); err != nil {
		t.Fatal(err)
	}
	var response Response
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if result.State != "completed" || response.Run == nil || response.Run.Status != "passed" || response.Error != "" || response.EvidenceStatus != "unavailable" || !strings.Contains(response.EvidenceError, "do not retry the action") || len(result.Artifacts) != 0 || len(response.Artifacts) != 0 {
		t.Fatalf("known success changed by publication: state=%s payload=%s", result.State, result.Payload)
	}
	if len(response.EvidenceError) > 256 {
		t.Fatal("unbounded evidence diagnostic")
	}
	runID := response.Run.ID
	runs, err := db.Runs(context.Background(), created.LeaseID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs: %+v %v", runs, err)
	}
	for _, run := range runs {
		if run.Status != "passed" {
			t.Fatalf("action outcome lost: %+v", run)
		}
	}
	result = executePrepared(t, e, nextOperation(created, "artifact", Request{Run: runID}))
	if len(result.Artifacts) == 0 || p.calls != 2 {
		t.Fatalf("evidence retry lost artifacts or replayed input: blobs=%d calls=%d", len(result.Artifacts), p.calls)
	}
}
