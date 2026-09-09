package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestUncertainEvidenceReadPreservesRuntimeState(t *testing.T) {
	for _, kind := range []string{"logs", "artifact", "test"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s := testStore(t)
			w := register(t, s, "host", 1, 1)
			create, err := s.Create(ctx, request("create"))
			if err != nil {
				t.Fatal(err)
			}
			finishReviewOperation(t, s, w, create, "ready", false)
			op, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "operation", LeaseID: create.LeaseID, Kind: kind, Payload: json.RawMessage(`{}`)})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Poll(ctx, w); err != nil {
				t.Fatal(err)
			}
			result := protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "uncertain", LocalState: "ready", Payload: json.RawMessage(`{"error":"evidence unavailable"}`)}
			if _, err := s.Complete(ctx, result); err != nil {
				t.Fatal(err)
			}
			lease, err := s.GetLease(ctx, create.LeaseID)
			want := "READY"
			if kind == "test" {
				want = "QUARANTINED"
			}
			if err != nil || lease.State != want {
				t.Fatalf("%s uncertainty changed runtime state to %s; want %s: %v", kind, lease.State, want, err)
			}
			stored, err := s.GetOperation(ctx, op.ID)
			if err != nil || stored.State != "uncertain" || stored.Result == nil {
				t.Fatalf("operation failure evidence lost: %+v %v", stored, err)
			}
		})
	}
}

func TestEvidenceReadCannotReanimateReleasedOrQuarantinedLease(t *testing.T) {
	for _, phase := range []string{"RELEASED", "QUARANTINED"} {
		t.Run(phase, func(t *testing.T) {
			ctx := context.Background()
			s := testStore(t)
			w := register(t, s, "host", 1, 1)
			var leaseID string
			if phase == "RELEASED" {
				leaseID = releasedReviewLease(t, s, w)
			} else {
				create, err := s.Create(ctx, request("create"))
				if err != nil {
					t.Fatal(err)
				}
				leaseID = create.LeaseID
				if _, err := s.Poll(ctx, w); err != nil {
					t.Fatal(err)
				}
				if _, err := s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: create.ID, LeaseID: create.LeaseID, Epoch: create.Epoch, State: "uncertain"}); err != nil {
					t.Fatal(err)
				}
			}
			for _, kind := range []string{"logs", "artifact"} {
				for _, outcome := range []string{"completed", "uncertain"} {
					op, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: kind + "-" + outcome, LeaseID: leaseID, Kind: kind, Payload: json.RawMessage(`{}`)})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := s.Poll(ctx, w); err != nil {
						t.Fatal(err)
					}
					digest := strings.Repeat("d", 64)
					result := protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: outcome, LocalState: "ready", Artifacts: []protocol.Blob{{Digest: digest, Size: 1}}}
					if _, err := s.Complete(ctx, result); err != nil {
						t.Fatal(err)
					}
					lease, err := s.GetLease(ctx, leaseID)
					if err != nil || lease.State != phase {
						t.Fatalf("%s %s changed %s lease: %+v %v", kind, outcome, phase, lease, err)
					}
					var count int
					if err := s.db.QueryRow("SELECT count(*) FROM blob_refs WHERE digest=? AND lease_id=? AND kind='artifact'", digest, leaseID).Scan(&count); err != nil || count != 1 {
						t.Fatalf("read artifact association lost: %d %v", count, err)
					}
				}
			}
		})
	}
}
