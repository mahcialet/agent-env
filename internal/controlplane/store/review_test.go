package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func finishReviewOperation(t *testing.T, s *Store, w protocol.WorkerIdentity, op protocol.Operation, state string, proof bool) {
	t.Helper()
	p, err := s.Poll(context.Background(), w)
	if err != nil || p == nil || p.ID != op.ID {
		t.Fatalf("dispatch: %+v %v", p, err)
	}
	_, err = s.Complete(context.Background(), protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: state, CleanupConfirmed: proof})
	if err != nil {
		t.Fatal(err)
	}
}

func releasedReviewLease(t *testing.T, s *Store, w protocol.WorkerIdentity) string {
	t.Helper()
	ctx := context.Background()
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, create, "ready", false)
	destroy, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "destroy", LeaseID: create.LeaseID, Kind: "destroy", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, destroy, "released", true)
	return create.LeaseID
}

func TestRemovedHostCannotStrandReleasedLeaseOperations(t *testing.T) {
	for _, state := range []string{"queued", "dispatched", "removed"} {
		t.Run(state, func(t *testing.T) {
			ctx := context.Background()
			s := testStore(t)
			w := register(t, s, "host", 1, 1)
			id := releasedReviewLease(t, s, w)
			r := protocol.SubmitRequest{OperationID: "show", LeaseID: id, Kind: "show", Payload: json.RawMessage(`{}`)}
			if state == "removed" {
				if _, err := s.Remove(ctx, w.HostID); err != nil {
					t.Fatal(err)
				}
				if _, err := s.Submit(ctx, r); !isFault(err, "conflict") {
					t.Fatalf("removed host accepted new operation: %v", err)
				}
				return
			}
			if _, err := s.Submit(ctx, r); err != nil {
				t.Fatal(err)
			}
			if state == "dispatched" {
				if _, err := s.Poll(ctx, w); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Remove(ctx, w.HostID); !isFault(err, "conflict") {
				t.Fatalf("removed host with %s operation: %v", state, err)
			}
			pending, err := s.GetOperation(ctx, r.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			finishReviewOperation(t, s, w, pending, "released", false)
			if _, err := s.Remove(ctx, w.HostID); err != nil {
				t.Fatalf("completed operation blocked removal: %v", err)
			}
		})
	}
}

func TestControllerExpiryQueuesProofRequiringDestroy(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	w := register(t, s, "host", 1, 1)
	r := request("create")
	r.Options = json.RawMessage(`{"ttl":1000000}`)
	op, err := s.Create(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, op, "ready", false)
	time.Sleep(3 * time.Millisecond)
	destroy, err := s.Poll(ctx, w)
	if err != nil || destroy == nil || destroy.Kind != "destroy" {
		t.Fatalf("expired lease did not receive controller destroy: %+v %v", destroy, err)
	}
	if destroy.HostID != w.HostID || destroy.HostInstanceID != w.HostInstanceID || destroy.Epoch != op.Epoch {
		t.Fatal("expiry changed assignment")
	}
	if _, err := s.Create(ctx, request("blocked")); !isFault(err, "capacity") {
		t.Fatalf("expiry freed unproven capacity: %v", err)
	}
	if _, err := s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: destroy.ID, LeaseID: destroy.LeaseID, Epoch: destroy.Epoch, State: "completed", LocalState: "released"}); !isFault(err, "invalid") {
		t.Fatalf("expiry bypassed proof: %v", err)
	}
	if _, err := s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: destroy.ID, LeaseID: destroy.LeaseID, Epoch: destroy.Epoch, State: "completed", LocalState: "released", CleanupConfirmed: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, request("reused")); err != nil {
		t.Fatal(err)
	}
}

func TestCompensatedCreateUncertaintyRequiresExplicitDestroyProof(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	w := register(t, s, "host", 1, 1)
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	result := protocol.Result{WorkerIdentity: w, OperationID: create.ID, LeaseID: create.LeaseID, Epoch: create.Epoch, State: "uncertain", Payload: json.RawMessage(`{"lease":{"observed_state":"released"}}`)}
	if _, err := s.Complete(ctx, result); err != nil {
		t.Fatalf("compensated create could not report uncertainty: %v", err)
	}
	lease, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || lease.State != "QUARANTINED" {
		t.Fatalf("create compensation fabricated controller release: %+v %v", lease, err)
	}
	if _, err := s.Create(ctx, request("blocked")); !isFault(err, "capacity") {
		t.Fatalf("compensation freed unproven capacity: %v", err)
	}
	destroy, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "destroy", LeaseID: create.LeaseID, Kind: "destroy", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, destroy, "released", true)
}
