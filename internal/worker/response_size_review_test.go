package worker

import (
	"bytes"
	"encoding/json"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"testing"
)

func TestRemoteLeaseResponseCompactsLargeDeclarationsWithoutChangingStoredState(t *testing.T) {
	padding := string(bytes.Repeat([]byte{'a'}, (4<<20)-150))
	manifest := json.RawMessage(`{"padding":"` + padding + `"}`)
	lease := domain.Lease{ID: "fixture-lease", Observed: "ready", Manifest: manifest, ManifestDigest: "retained-manifest-digest", Runtimes: []domain.Runtime{{Name: "process", Type: "process", LeaseID: "fixture-lease", Process: &domain.PersistentProcess{Command: []string{"fixture", padding}, Env: map[string]string{"FILL": padding}, ProcessID: 42, ProcessStart: "birth-proof", State: "running", StateDirectory: "retained-state", Ports: map[string]int{"http": 12345}}}}}
	result := responseResult("completed", Response{Lease: &lease}, nil)
	if result.State != "completed" {
		t.Fatalf("large successful create changed state: %s %s", result.State, result.Payload)
	}
	var out Response
	if err := json.Unmarshal(result.Payload, &out); err != nil {
		t.Fatal(err)
	}
	if out.Lease == nil || string(out.Lease.Manifest) != "null" || out.Lease.ManifestDigest != lease.ManifestDigest {
		t.Fatal("remote manifest summary incorrect")
	}
	process := out.Lease.Runtimes[0].Process
	if process == nil || len(process.Command) != 0 || len(process.Env) != 0 || process.ProcessID != 42 || process.ProcessStart != "birth-proof" || process.StateDirectory != "retained-state" || process.Ports["http"] != 12345 {
		t.Fatalf("remote runtime observation changed: %+v", process)
	}
	if !bytes.Equal(lease.Manifest, manifest) || lease.Runtimes[0].Process.Env["FILL"] != padding || len(lease.Runtimes[0].Process.Command) != 2 {
		t.Fatal("response compaction modified stored declarations")
	}
	// Completed operation retrieval/poll responses contain request and result.
	// The retained canonical manifest must fit once within the unchanged8MiB cap.
	payload, _ := json.Marshal(protocol.CreateRequest{Manifest: manifest})
	wire, err := json.Marshal(protocol.Operation{ID: "operation", LeaseID: lease.ID, Payload: payload, Result: &result})
	if err != nil || len(wire) > 8<<20 {
		t.Fatalf("completed operation exceeds metadata envelope: %d %v", len(wire), err)
	}
}
