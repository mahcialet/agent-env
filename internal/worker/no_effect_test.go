package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestFailedCreateAbsenceProofSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	j, e := OpenJournal(home, "host")
	if e != nil {
		t.Fatal(e)
	}
	if e = j.Bind(ctx, "controller"); e != nil {
		t.Fatal(e)
	}
	op := protocol.Operation{ID: "create", Kind: "create", LeaseID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", ControllerID: j.Identity.ControllerID, HostID: j.Identity.HostID, HostInstanceID: j.Identity.HostInstanceID, Epoch: 1}
	receipt, e := j.Receive(ctx, op)
	if e != nil {
		t.Fatal(e)
	}
	executor := &fakeExecutor{preflight: errors.New("source preflight failed")}
	r := &Runner{Journal: j, Executor: executor, Transport: &fakeTransport{}}
	if e = r.handle(ctx, receipt); e != nil {
		t.Fatal(e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e = OpenJournal(home, "host")
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	r.Journal = j
	op.ID = "dry-destroy"
	op.Kind = "destroy"
	op.Payload = json.RawMessage(`{"dry_run":true}`)
	receipt, e = j.Receive(ctx, op)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.handle(ctx, receipt); e != nil {
		t.Fatal(e)
	}
	stored, e := j.Get(ctx, op.ID)
	if e != nil {
		t.Fatal(e)
	}
	if stored.Result.CleanupConfirmed {
		t.Fatal("dry run released capacity")
	}
	for i, payload := range []string{`{"component":"x"}`, `{"run":"x"}`, `{"ttl":-1}`, `{"unknown":true}`} {
		op.ID = fmt.Sprintf("invalid-destroy-%d", i)
		op.Payload = json.RawMessage(payload)
		receipt, e = j.Receive(ctx, op)
		if e != nil {
			t.Fatal(e)
		}
		if e = r.handle(ctx, receipt); e != nil {
			t.Fatal(e)
		}
		stored, e = j.Get(ctx, op.ID)
		if e != nil {
			t.Fatal(e)
		}
		if stored.Result.CleanupConfirmed {
			t.Fatal("invalid destroy request released capacity")
		}
	}
	op.ID = "destroy"
	op.Payload = json.RawMessage(`{}`)
	receipt, e = j.Receive(ctx, op)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.handle(ctx, receipt); e != nil {
		t.Fatal(e)
	}
	stored, e = j.Get(ctx, op.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !stored.Result.CleanupConfirmed || stored.Result.LocalState != "released" || executor.effects != 0 {
		t.Fatalf("durable absence proof lost: %+v", stored)
	}
}

func TestEffectPayloadCannotInventAbsenceProof(t *testing.T) {
	ctx := context.Background()
	r, receipt, _, _ := runnerFixture(t)
	// Even a lying result payload cannot override the durable effect boundary.
	if e := r.Journal.Advance(ctx, receipt.Operation.ID, "received", "prepared"); e != nil {
		t.Fatal(e)
	}
	if e := r.Journal.Advance(ctx, receipt.Operation.ID, "prepared", "effect_started"); e != nil {
		t.Fatal(e)
	}
	result := protocol.Result{WorkerIdentity: r.Journal.Identity, OperationID: receipt.Operation.ID, LeaseID: receipt.Operation.LeaseID, Epoch: 1, State: "failed", Payload: json.RawMessage(`{"effects_started":false}`)}
	if e := r.Journal.StoreResult(ctx, receipt.Operation.ID, result); e != nil {
		t.Fatal(e)
	}
	if e := r.Journal.Advance(ctx, receipt.Operation.ID, "result_pending", "completed"); e != nil {
		t.Fatal(e)
	}
	destroy := receipt.Operation
	destroy.ID = "destroy"
	destroy.Kind = "destroy"
	destroy.Payload = json.RawMessage(`{}`)
	proved, e := r.Journal.ProvesNoEffects(ctx, destroy)
	if e != nil {
		t.Fatal(e)
	}
	if proved {
		t.Fatal("result payload invented absence")
	}
	destroy.LeaseID = "other"
	proved, e = r.Journal.ProvesNoEffects(ctx, destroy)
	if e != nil || proved {
		t.Fatalf("missing journal proves absence: %v %v", proved, e)
	}
}
