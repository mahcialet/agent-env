package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "controller.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func register(t *testing.T, s *Store, id string, leases, slots int) protocol.WorkerIdentity {
	t.Helper()
	w := protocol.WorkerIdentity{ControllerID: s.ID, HostID: id, HostInstanceID: id + "-instance", Incarnation: "boot-1"}
	_, e := s.Register(context.Background(), protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: protocol.Version, ProductVersion: "test", OS: "linux", Arch: "amd64", Capabilities: []string{"process", "android"}, Capacity: protocol.Capacity{MaxLeases: leases, AndroidSlots: slots}})
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func request(id string) protocol.CreateRequest {
	d := strings.Repeat("a", 64)
	return protocol.CreateRequest{OperationID: id, Stack: "app", Manifest: json.RawMessage("{}"), ManifestDigest: d, PlanDigest: d, ControlBlobDigest: d, Package: json.RawMessage(`{"manifest_blob_digest":"` + d + `","sources":[]}`), RequiredCapabilities: []string{"process"}, AndroidSlots: 1}
}
func isFault(err error, code string) bool { var f *Fault; return errors.As(err, &f) && f.Code == code }
func TestControllerSchedulerAtomicCapacityAndAndroidSlots(t *testing.T) {
	s := testStore(t)
	register(t, s, "a", 1, 2)
	register(t, s, "b", 2, 1)
	// Two handles model independent controller DB clients; SQLite owns serialization.
	peer, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	defer peer.Close()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var got []protocol.Operation
	var unexpected []error
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			db := s
			if i%2 == 1 {
				db = peer
			}
			o, e := db.Create(context.Background(), request(fmt.Sprint("op-", i)))
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				got = append(got, o)
			} else if !isFault(e, "capacity") {
				unexpected = append(unexpected, e)
			}
		}(i)
	}
	wg.Wait()
	if len(unexpected) > 0 || len(got) != 2 {
		t.Fatalf("assignments=%d errors=%v", len(got), unexpected)
	}
	if got[0].HostID == got[1].HostID {
		t.Fatal("Android slots overcommitted")
	}
}
func TestControllerOfflineNeverReassignsOrReleases(t *testing.T) {
	s := testStore(t)
	w := register(t, s, "a", 1, 1)
	register(t, s, "b", 1, 1)
	q := request("create")
	q.HostID = "a"
	op, e := s.Create(context.Background(), q)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE hosts SET last_seen=? WHERE id='a'", time.Now().Add(-time.Minute).UnixNano()); e != nil {
		t.Fatal(e)
	}
	l, e := s.GetLease(context.Background(), op.LeaseID)
	if e != nil || l.State != "UNKNOWN" || l.HostID != "a" || l.LastKnownState != "ASSIGNED" {
		t.Fatalf("offline lease %+v %v", l, e)
	}
	q.OperationID = "another"
	if _, e = s.Create(context.Background(), q); !isFault(e, "capacity") {
		t.Fatalf("offline selected host accepted: %v", e)
	}
	q.HostID = ""
	other, e := s.Create(context.Background(), q)
	if e != nil || other.HostID != "b" {
		t.Fatalf("fallback %+v %v", other, e)
	}
	if _, e = s.Heartbeat(context.Background(), w); e != nil {
		t.Fatal(e)
	}
	q.OperationID = "after-reconnect"
	q.HostID = "a"
	if _, e = s.Create(context.Background(), q); !isFault(e, "capacity") {
		t.Fatal("offline period freed owned capacity")
	}
	if _, e = s.Remove(context.Background(), "a"); !isFault(e, "conflict") {
		t.Fatal("removed host with owned assignment")
	}
}
func TestControllerIdempotencyEpochAndDurableIdentity(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := register(t, s, "a", 1, 1)
	q := request("create")
	o, e := s.Create(ctx, q)
	if e != nil {
		t.Fatal(e)
	}
	duplicate, e := s.Create(ctx, q)
	if e != nil || duplicate.LeaseID != o.LeaseID {
		t.Fatal("create replay changed placement")
	}
	q.Stack = "other"
	if _, e = s.Create(ctx, q); !isFault(e, "conflict") {
		t.Fatal("operation payload conflict not rejected")
	}
	polled, e := s.Poll(ctx, w)
	if e != nil || polled == nil || polled.ID != o.ID {
		t.Fatalf("poll %+v %v", polled, e)
	}
	id := s.ID
	peer, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	defer peer.Close()
	if peer.ID != id {
		t.Fatal("controller restart changed durable ID")
	}
	replay, e := peer.Poll(ctx, w)
	if e != nil || replay.ID != o.ID || replay.Epoch != o.Epoch {
		t.Fatal("restart replaced dispatched work")
	}
	result := protocol.Result{WorkerIdentity: w, OperationID: o.ID, LeaseID: o.LeaseID, Epoch: o.Epoch, State: "completed", LocalState: "ready", Payload: json.RawMessage(`{"ok":true}`)}
	for _, bad := range []string{"controller", "instance", "host", "incarnation", "epoch"} {
		r := result
		switch bad {
		case "controller":
			r.ControllerID = "wrong"
		case "instance":
			r.HostInstanceID = "wrong"
		case "host":
			r.HostID = "b"
		case "incarnation":
			r.Incarnation = "wrong"
		case "epoch":
			r.Epoch++
		}
		if _, e = s.Complete(ctx, r); !isFault(e, "identity") {
			t.Fatalf("%s identity accepted: %v", bad, e)
		}
	}
	if _, e = s.Complete(ctx, result); e != nil {
		t.Fatal(e)
	}
	w.Incarnation = "boot-2"
	if _, e = s.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: 1, ProductVersion: "test", OS: "linux", Arch: "amd64", Capabilities: []string{"process"}, Capacity: protocol.Capacity{MaxLeases: 1, AndroidSlots: 1}}); e != nil {
		t.Fatal(e)
	}
	result.WorkerIdentity = w
	if _, e = s.Complete(ctx, result); e != nil {
		t.Fatalf("durable result replay across restart: %v", e)
	}
	result.Payload = json.RawMessage(`{"ok":false}`)
	if _, e = s.Complete(ctx, result); !isFault(e, "conflict") {
		t.Fatal("conflicting result accepted")
	}
	next, e := s.Submit(ctx, protocol.SubmitRequest{OperationID: "inspect", LeaseID: o.LeaseID, Kind: "show", Payload: json.RawMessage("{}")})
	if e != nil || next.Epoch != o.Epoch {
		t.Fatalf("assignment epoch changed: %+v %v", next, e)
	}
}
func TestControllerReleaseRequiresWorkerProofAndRemovalIsConservative(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := register(t, s, "a", 1, 1)
	o, e := s.Create(ctx, request("create"))
	if e != nil {
		t.Fatal(e)
	}
	s.Poll(ctx, w)
	r := protocol.Result{WorkerIdentity: w, OperationID: o.ID, LeaseID: o.LeaseID, Epoch: o.Epoch, State: "completed", LocalState: "released"}
	if _, e = s.Complete(ctx, r); !isFault(e, "invalid") {
		t.Fatal("unverified release accepted")
	}
	r.CleanupConfirmed = true
	if _, e = s.Complete(ctx, r); !isFault(e, "invalid") {
		t.Fatal("create result released capacity without destroy/reconcile operation")
	}
	r.CleanupConfirmed = false
	r.State = "uncertain"
	r.LocalState = "failed"
	if _, e = s.Complete(ctx, r); e != nil {
		t.Fatal(e)
	}
	l, e := s.GetLease(ctx, o.LeaseID)
	if e != nil || l.State != "QUARANTINED" {
		t.Fatal("uncertain result lost quarantine")
	}
	if _, e = s.Remove(ctx, "a"); !isFault(e, "conflict") {
		t.Fatal("uncertain host removed")
	}
	d, e := s.Submit(ctx, protocol.SubmitRequest{OperationID: "destroy", LeaseID: o.LeaseID, Kind: "destroy", Payload: json.RawMessage("{}")})
	if e != nil {
		t.Fatal(e)
	}
	s.Poll(ctx, w)
	r.OperationID = d.ID
	r.State = "completed"
	r.LocalState = "released"
	r.CleanupConfirmed = false
	if _, e = s.Complete(ctx, r); !isFault(e, "invalid") {
		t.Fatal("lowercase released accepted without cleanup proof")
	}
	if _, e = s.Create(ctx, request("still-reserved")); !isFault(e, "capacity") {
		t.Fatal("unconfirmed release freed capacity")
	}
	r.LocalState = "destroyed"
	if _, e = s.Complete(ctx, r); !isFault(e, "invalid") {
		t.Fatal("invented local state accepted")
	}
	r.LocalState = "released"
	r.CleanupConfirmed = true
	if _, e = s.Complete(ctx, r); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Remove(ctx, "a"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: 1, ProductVersion: "test", OS: "linux", Arch: "amd64", Capacity: protocol.Capacity{MaxLeases: 1}}); !isFault(e, "identity") {
		t.Fatal("removed host reregistered")
	}
}
func TestControllerDrainIdentityAndBlobAssignment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := register(t, s, "a", 1, 1)
	register(t, s, "b", 1, 1)
	_, e := s.Drain(ctx, "a")
	if e != nil {
		t.Fatal(e)
	}
	q := request("create")
	q.HostID = "a"
	if _, e = s.Create(ctx, q); !isFault(e, "capacity") {
		t.Fatal("draining host accepted new work")
	}
	if _, e = s.Undrain(ctx, "a"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Create(ctx, q); e != nil {
		t.Fatal(e)
	}
	good := protocol.Enrollment{Role: "worker", HostID: "a"}
	if e = s.AuthorizeBlob(ctx, good, q.ControlBlobDigest, "source"); e != nil {
		t.Fatal(e)
	}
	good.HostID = "b"
	if e = s.AuthorizeBlob(ctx, good, q.ControlBlobDigest, "source"); !isFault(e, "forbidden") {
		t.Fatal("other host source authorized")
	}
	w.HostInstanceID = "replacement"
	if _, e = s.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: 1, ProductVersion: "test", OS: "linux", Arch: "amd64", Capacity: protocol.Capacity{MaxLeases: 1}}); !isFault(e, "identity") {
		t.Fatal("host identity silently replaced")
	}
}

func TestControllerUnknownOperationNeverQueues(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := register(t, s, "a", 1, 1)
	o, e := s.Create(ctx, request("create"))
	if e != nil {
		t.Fatal(e)
	}
	s.Poll(ctx, w)
	if _, e = s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: o.ID, LeaseID: o.LeaseID, Epoch: o.Epoch, State: "completed", LocalState: "ready"}); e != nil {
		t.Fatal(e)
	}
	_, e = s.Submit(ctx, protocol.SubmitRequest{OperationID: "invalid-kind", LeaseID: o.LeaseID, Kind: "shell", Payload: json.RawMessage("{}")})
	if !isFault(e, "invalid") {
		t.Fatalf("unknown kind accepted: %v", e)
	}
	if _, e = s.GetOperation(ctx, "invalid-kind"); !isFault(e, "not_found") {
		t.Fatal("unknown operation queued")
	}
}
