package store

import (
	"context"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestIncompatibleInventoryRetainsAssignmentWithoutDispatch(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	s.ProductVersion = "test"
	w := register(t, s, "host", 2, 2)
	op, err := s.Create(ctx, request("original"))
	if err != nil {
		t.Fatal(err)
	}
	r := protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: protocol.Version, ProductVersion: "other", OS: "linux", Arch: "amd64", Capabilities: []string{"process"}, Capacity: protocol.Capacity{MaxLeases: 2, AndroidSlots: 2}}
	r.Incarnation = "incompatible-boot"
	expireWorkerHeartbeat(t, s, w.HostID)
	for _, kind := range []string{"product", "protocol"} {
		if kind == "protocol" {
			r.ProductVersion = "test"
			r.ProtocolVersion++
		}
		h, err := s.Register(ctx, r)
		if err != nil || h.Compatible || h.CompatibilityError == "" {
			t.Fatalf("%s incompatible registration: %+v %v", kind, h, err)
		}
		hosts, err := s.ListHosts(ctx)
		if err != nil || len(hosts) != 1 || hosts[0].Compatible {
			t.Fatalf("incompatible inventory: %+v %v", hosts, err)
		}
		if _, err := s.Poll(ctx, r.WorkerIdentity); !isFault(err, "version") {
			t.Fatalf("incompatible poll: %v", err)
		}
		if _, err := s.Create(ctx, request("new")); !isFault(err, "capacity") {
			t.Fatalf("incompatible scheduling: %v", err)
		}
		lease, err := s.GetLease(ctx, op.LeaseID)
		if err != nil || lease.HostInstanceID != w.HostInstanceID || lease.Epoch != op.Epoch || lease.State == "RELEASED" {
			t.Fatalf("assignment changed: %+v %v", lease, err)
		}
		replacement := r
		replacement.HostInstanceID = "foreign"
		if _, err := s.Register(ctx, replacement); !isFault(err, "identity") {
			t.Fatalf("incompatible instance adopted host: %v", err)
		}
	}
	r.ProductVersion, r.ProtocolVersion = "test", protocol.Version
	if _, err := s.Register(ctx, r); err != nil {
		t.Fatal(err)
	}
	queued, err := s.Poll(ctx, r.WorkerIdentity)
	if err != nil || queued == nil || queued.ID != op.ID {
		t.Fatalf("compatible recovery lost original assignment: %+v %v", queued, err)
	}
	// A controller upgrade invalidates persisted inventory immediately, without
	// waiting for the old worker to re-register and overwrite its old status.
	s.ProductVersion = "next"
	host, err := s.GetHost(ctx, w.HostID)
	if err != nil || host.Compatible {
		t.Fatalf("old inventory survived controller upgrade: %+v %v", host, err)
	}
	if _, err := s.Poll(ctx, r.WorkerIdentity); !isFault(err, "version") {
		t.Fatalf("old worker polled upgraded controller: %v", err)
	}
}
