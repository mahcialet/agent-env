package worker

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestReceiptRestartNeverReauthorizesEffect(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	j, e := OpenJournal(home, "host-a")
	if e != nil {
		t.Fatal(e)
	}
	if e = j.Bind(ctx, "controller-a"); e != nil {
		t.Fatal(e)
	}
	identity := j.Identity
	op := protocol.Operation{ID: "operation-a", ControllerID: identity.ControllerID, HostID: identity.HostID, HostInstanceID: identity.HostInstanceID, LeaseID: "lease-a", Epoch: 1, Kind: "test", Payload: json.RawMessage(`{"name":"e2e"}`)}
	r, e := j.Receive(ctx, op)
	if e != nil || r.State != "received" {
		t.Fatalf("receive: %+v %v", r, e)
	}
	if e = j.Advance(ctx, op.ID, "received", "prepared"); e != nil {
		t.Fatal(e)
	}
	if e = j.Advance(ctx, op.ID, "prepared", "effect_started"); e != nil {
		t.Fatal(e)
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	j, e = OpenJournal(home, "host-a")
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if j.Identity.HostInstanceID != identity.HostInstanceID || j.Identity.ControllerID != identity.ControllerID || j.Identity.Incarnation == identity.Incarnation {
		t.Fatal("restart identity not preserved")
	}
	if e = j.Bind(ctx, "another-controller"); e == nil {
		t.Fatal("controller adoption allowed")
	}
	r, e = j.Receive(ctx, op)
	if e != nil || r.State != "effect_started" {
		t.Fatalf("duplicate reset lost-effect receipt: %+v %v", r, e)
	}
	if e = j.Advance(ctx, op.ID, "effect_started", "received"); e == nil {
		t.Fatal("mutation replay enabled")
	}
	different := op
	different.Payload = json.RawMessage(`{"name":"different"}`)
	if _, e = j.Receive(ctx, different); e == nil {
		t.Fatal("conflicting duplicate accepted")
	}
	different = op
	different.ID = "operation-b"
	if _, e = j.Receive(ctx, different); e == nil {
		t.Fatal("unfinished operation bypassed")
	}
	result := protocol.Result{WorkerIdentity: j.Identity, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: 1, State: "uncertain", Payload: json.RawMessage(`{"error":"effect result unavailable"}`)}
	if e = j.StoreResult(ctx, op.ID, result); e != nil {
		t.Fatal(e)
	}
	pending, e := j.Pending(ctx)
	if e != nil || len(pending) != 1 || pending[0].Result == nil {
		t.Fatalf("durable upload pending: %+v %v", pending, e)
	}
	if e = j.StoreResult(ctx, op.ID, result); e == nil {
		t.Fatal("durable result overwritten")
	}
	if e = j.Advance(ctx, op.ID, "result_pending", "completed"); e != nil {
		t.Fatal(e)
	}
	different.Kind = "reconcile"
	if _, e = j.Receive(ctx, different); e != nil {
		t.Fatal(e)
	}
}

func TestJournalIdentityAndSingleOwner(t *testing.T) {
	ctx := context.Background()
	home := filepath.Join(t.TempDir(), "state")
	j, e := OpenJournal(home, "host-a")
	if e != nil {
		t.Fatal(e)
	}
	if other, e := OpenJournal(home, "host-a"); e == nil {
		other.Close()
		t.Fatal("duplicate worker")
	}
	if e = j.Bind(ctx, "controller-a"); e != nil {
		t.Fatal(e)
	}
	op := protocol.Operation{ID: "op", LeaseID: "lease", ControllerID: "controller-a", HostID: "host-a", HostInstanceID: j.Identity.HostInstanceID, Epoch: 1, Kind: "show", Payload: json.RawMessage(`{}`)}
	for _, change := range []func(*protocol.Operation){func(o *protocol.Operation) { o.ControllerID = "wrong" }, func(o *protocol.Operation) { o.HostID = "wrong" }, func(o *protocol.Operation) { o.HostInstanceID = "wrong" }, func(o *protocol.Operation) { o.Epoch = 0 }} {
		bad := op
		change(&bad)
		if _, e = j.Receive(ctx, bad); e == nil {
			t.Fatal("wrong assignment accepted")
		}
	}
	if _, e = j.Receive(ctx, op); e != nil {
		t.Fatal(e)
	}
	result := protocol.Result{WorkerIdentity: j.Identity, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: 1, State: "completed"}
	bad := result
	bad.Epoch = 2
	if e = j.StoreResult(ctx, op.ID, bad); e == nil {
		t.Fatal("wrong result epoch accepted")
	}
	if e = j.StoreResult(ctx, op.ID, result); e != nil {
		t.Fatal(e)
	}
	if e = j.Advance(ctx, op.ID, "result_pending", "completed"); e != nil {
		t.Fatal(e)
	}
	op.ID = "next"
	op.Epoch = 2
	if _, e = j.Receive(ctx, op); e == nil {
		t.Fatal("assignment epoch changed")
	}
	if e = j.Close(); e != nil {
		t.Fatal(e)
	}
	if other, e := OpenJournal(home, "host-b"); e == nil {
		other.Close()
		t.Fatal("host root adopted")
	}
}
