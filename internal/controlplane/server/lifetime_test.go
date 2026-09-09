package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestServerQueuesExpiredLeaseWithoutWorkerPolling(t *testing.T) {
	f := serveFixture(t)
	ctx := context.Background()
	w := protocol.WorkerIdentity{ControllerID: f.store.ID, HostID: "host-a", HostInstanceID: "instance", Incarnation: "boot"}
	if _, err := f.worker.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: protocol.Version, ProductVersion: "test", OS: "linux", Arch: "amd64", Capacity: protocol.Capacity{MaxLeases: 1}}); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	op, err := f.store.Create(ctx, protocol.CreateRequest{OperationID: "expiry-create", Stack: "local", Manifest: json.RawMessage(`{}`), Package: json.RawMessage(`{"manifest_blob_digest":"` + digest + `","sources":[]}`), ControlBlobDigest: digest, ManifestDigest: digest, PlanDigest: strings.Repeat("b", 64), SourceSetDigest: strings.Repeat("c", 64), RepositoryID: "repository", Options: json.RawMessage(`{"ttl":1000000}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "ready"}); err != nil {
		t.Fatal(err)
	}
	// Inspect from an independent connection: calling Poll here would itself
	// trigger expiry and conceal a missing background controller sweep.
	db, err := sql.Open("sqlite", f.store.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	deadline := time.Now().Add(4 * time.Second)
	for {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM operations WHERE lease_id=? AND kind='destroy' AND state='queued'", op.LeaseID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("controller did not persist expiry cleanup without worker polling")
		}
		time.Sleep(10 * time.Millisecond)
	}
	lease, err := f.admin.GetLease(ctx, op.LeaseID)
	if err != nil || lease.State != "READY" || lease.ExpiresAt.IsZero() {
		t.Fatalf("expiry fabricated cleanup or omitted deadline: %+v %v", lease, err)
	}
	result, err := f.worker.Poll(ctx, protocol.PollRequest{WorkerIdentity: w})
	if err != nil || result.Operation == nil || result.Operation.Kind != "destroy" {
		t.Fatalf("worker did not receive durable expiry cleanup: %+v %v", result, err)
	}
}
