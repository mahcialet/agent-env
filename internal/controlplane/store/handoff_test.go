package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func expireWorkerHeartbeat(t *testing.T, s *Store, host string) {
	t.Helper()
	if _, err := s.db.Exec("UPDATE hosts SET last_seen=? WHERE id=?", time.Now().Add(-s.OfflineAfter-time.Second).UnixNano(), host); err != nil {
		t.Fatal(err)
	}
}

func replacementRequest(w protocol.WorkerIdentity) protocol.RegisterRequest {
	w.Incarnation = "replacement"
	return protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: protocol.Version, ProductVersion: "test", OS: "linux", Arch: "amd64", Capabilities: []string{"process"}, Capacity: protocol.Capacity{MaxLeases: 1, AndroidSlots: 1}}
}

func TestLiveWorkerCannotLoseDispatchToNewIncarnation(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	w := register(t, s, "host", 1, 1)
	op, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	replacement := replacementRequest(w)
	if _, err := s.Register(ctx, replacement); !isFault(err, "unavailable") {
		t.Fatalf("live incarnation could be replaced: %v", err)
	}
	if _, err := s.Heartbeat(ctx, w); err != nil {
		t.Fatalf("rejected clone fenced original: %v", err)
	}
	expireWorkerHeartbeat(t, s, w.HostID)
	if _, err := s.Register(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	if dispatched, err := s.Poll(ctx, replacement.WorkerIdentity); err != nil || dispatched != nil {
		t.Fatalf("replacement obtained old dispatched payload: %+v %v", dispatched, err)
	}
	lease, err := s.GetLease(ctx, op.LeaseID)
	if err != nil || lease.State != "QUARANTINED" || lease.Epoch != op.Epoch || lease.HostInstanceID != w.HostInstanceID {
		t.Fatalf("handoff lost uncertainty or placement: %+v %v", lease, err)
	}
	if _, err := s.Create(ctx, request("blocked")); !isFault(err, "capacity") {
		t.Fatalf("handoff released capacity: %v", err)
	}
	if _, err := s.Heartbeat(ctx, w); !isFault(err, "identity") {
		t.Fatalf("old incarnation not fenced: %v", err)
	}
	// A real restart recovers its pre-existing receipt, without another Poll.
	if _, err := s.Complete(ctx, protocol.Result{WorkerIdentity: replacement.WorkerIdentity, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "ready"}); err != nil {
		t.Fatalf("same-journal recovery blocked: %v", err)
	}
	lease, err = s.GetLease(ctx, op.LeaseID)
	if err != nil || lease.State != "READY" {
		t.Fatalf("recovery not accepted: %+v %v", lease, err)
	}
}

func TestOfflineHandoffCanReceivePreviouslyUndispatchedWork(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	w := register(t, s, "host", 1, 1)
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	expireWorkerHeartbeat(t, s, w.HostID)
	r := replacementRequest(w)
	if _, err := s.Register(ctx, r); err != nil {
		t.Fatal(err)
	}
	op, err := s.Poll(ctx, r.WorkerIdentity)
	if err != nil || op == nil || op.ID != create.ID {
		t.Fatalf("queued work blocked: %+v %v", op, err)
	}
}

func TestLegacyDispatchedOwnerSurvivesControllerRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "controller.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	w := register(t, s, "host", 1, 1)
	create, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("DROP TABLE operation_dispatches"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var owner string
	if err := s.db.QueryRow("SELECT incarnation FROM operation_dispatches WHERE operation_id=?", create.ID).Scan(&owner); err != nil || owner != "" {
		t.Fatalf("legacy dispatch owner was guessed: %q %v", owner, err)
	}
	if op, err := s.Poll(ctx, w); err != nil || op != nil {
		t.Fatalf("unverified legacy dispatch was replayed: %+v %v", op, err)
	}
	expireWorkerHeartbeat(t, s, w.HostID)
	r := replacementRequest(w)
	if _, err := s.Register(ctx, r); err != nil {
		t.Fatal(err)
	}
	if op, err := s.Poll(ctx, r.WorkerIdentity); err != nil || op != nil {
		t.Fatalf("migration replayed dispatch: %+v %v", op, err)
	}
}
