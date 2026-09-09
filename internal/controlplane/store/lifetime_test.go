package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestLifetimeRenewalPersistenceAndExpiryRecovery(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "controller.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	w := register(t, s, "host", 1, 1)
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, create, "ready", false)
	original, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || original.ExpiresAt.IsZero() {
		t.Fatalf("missing create deadline: %+v %v", original, err)
	}
	renew, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "renew", LeaseID: create.LeaseID, Kind: "renew", Payload: json.RawMessage(`{"ttl":43200000000000}`)})
	if err != nil {
		t.Fatal(err)
	}
	// Even an old deadline cannot start cleanup while another mutation is pending.
	if err := s.QueueExpired(ctx, original.ExpiresAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, renew, "ready", false)
	renewed, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || !renewed.ExpiresAt.After(original.ExpiresAt.Add(7*time.Hour)) {
		t.Fatalf("renew did not extend controller deadline: %+v %v", renewed, err)
	}
	if err := s.QueueExpired(ctx, original.ExpiresAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if op, err := s.Poll(ctx, w); err != nil || op != nil {
		t.Fatalf("old deadline survived renewal: %+v %v", op, err)
	}
	// Simulate migration from the pre-deadline schema, preserving all original
	// operation evidence. Reopening must derive timestamps, not renew at startup.
	if _, err := s.db.Exec("DELETE FROM lease_lifetimes"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || !recovered.ExpiresAt.Equal(renewed.ExpiresAt) {
		t.Fatalf("restart changed expiry: %+v %v", recovered, err)
	}
	if err := s.QueueExpired(ctx, recovered.ExpiresAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := s.QueueExpired(ctx, recovered.ExpiresAt.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	destroy, err := s.Poll(ctx, w)
	if err != nil || destroy == nil || destroy.Kind != "destroy" {
		t.Fatalf("expiry not recovered: %+v %v", destroy, err)
	}
	var count int
	if err := s.db.QueryRow("SELECT count(*) FROM operations WHERE kind='destroy'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate cleanup attempts: %d %v", count, err)
	}
	uncertain := protocol.Result{WorkerIdentity: w, OperationID: destroy.ID, LeaseID: destroy.LeaseID, Epoch: destroy.Epoch, State: "uncertain", LocalState: "quarantined"}
	if _, err := s.Complete(ctx, uncertain); err != nil {
		t.Fatal(err)
	}
	if err := s.QueueExpired(ctx, recovered.ExpiresAt.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if op, err := s.Poll(ctx, w); err != nil || op != nil {
		t.Fatalf("uncertain expiry retried side effects: %+v %v", op, err)
	}
	lease, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || lease.State != "QUARANTINED" {
		t.Fatalf("uncertainty lost: %+v %v", lease, err)
	}
	if _, err := s.Create(ctx, request("capacity")); !isFault(err, "capacity") {
		t.Fatalf("uncertain expiry freed capacity: %v", err)
	}
}

func TestInvalidTTLAndFailedRenewNeverExtendLifetime(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	w := register(t, s, "host", 1, 1)
	for _, invalid := range []json.RawMessage{json.RawMessage(`{"ttl":-1}`), json.RawMessage(`{"ttl":86400000000001}`)} {
		r := request("invalid")
		r.Options = invalid
		if _, err := s.Create(ctx, r); !isFault(err, "invalid") {
			t.Fatalf("invalid create TTL persisted: %v", err)
		}
	}
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	finishReviewOperation(t, s, w, create, "ready", false)
	before, err := s.GetLease(ctx, create.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []json.RawMessage{json.RawMessage(`{"ttl":-1}`), json.RawMessage(`{"ttl":86400000000001}`)} {
		if _, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "invalid", LeaseID: create.LeaseID, Kind: "renew", Payload: invalid}); !isFault(err, "invalid") {
			t.Fatalf("invalid renew TTL persisted: %v", err)
		}
	}
	renew, err := s.Submit(ctx, protocol.SubmitRequest{OperationID: "renew", LeaseID: create.LeaseID, Kind: "renew", Payload: json.RawMessage(`{"ttl":43200000000000}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: renew.ID, LeaseID: renew.LeaseID, Epoch: renew.Epoch, State: "failed", LocalState: "ready"}); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetLease(ctx, create.LeaseID)
	if err != nil || !after.ExpiresAt.Equal(before.ExpiresAt) {
		t.Fatalf("failed renewal changed expiry: %+v %v", after, err)
	}
}

func TestLegacyInvalidTTLDoesNotPreventControllerRestart(t *testing.T) {
	for _, invalid := range []string{`{"ttl":-1}`, `{"ttl":86400000000001}`, `{"ttl":"bad"}`} {
		t.Run(invalid, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "controller.db")
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			register(t, s, "host", 1, 1)
			r := request("legacy-create")
			op, err := s.Create(ctx, r)
			if err != nil {
				t.Fatal(err)
			}
			before, err := s.GetLease(ctx, op.LeaseID)
			if err != nil {
				t.Fatal(err)
			}
			// Recreate a request accepted by the previous controller schema,
			// which left TTL validation entirely to the worker.
			r.Options = json.RawMessage(invalid)
			payload, hash, err := canonical(r)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec("UPDATE operations SET payload=?,request_hash=? WHERE id=?", payload, hash, op.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec("DELETE FROM lease_lifetimes"); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = Open(path)
			if err != nil {
				t.Fatalf("historical invalid TTL blocked restart: %v", err)
			}
			defer s.Close()
			after, err := s.GetLease(ctx, op.LeaseID)
			if err != nil || !after.ExpiresAt.Equal(before.ExpiresAt) || after.State != before.State {
				t.Fatalf("legacy migration extended lifetime or changed assignment: %+v %v", after, err)
			}
		})
	}
}
