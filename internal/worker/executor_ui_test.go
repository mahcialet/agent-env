package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/remotesource"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"github.com/oklog/ulid/v2"
)

type routingUIProvider struct {
	calls        int
	runtime      domain.Runtime
	request      domain.UIRequest
	rejectDevice bool
}

func (p *routingUIProvider) ObserveUI(_ context.Context, r domain.Runtime, q domain.UIRequest) (domain.UIObservation, error) {
	p.calls++
	p.runtime = r
	p.request = q
	if p.rejectDevice {
		return domain.UIObservation{Version: 1, Status: "invalid", Confirmed: true}, errors.New("owned device console identity mismatch")
	}
	return domain.UIObservation{Version: 1, Status: "ok", Confirmed: true, Backend: "fixture-verified-helper", ActionPerformed: q.Operation == "tap", Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "safe-node-fingerprint", Text: "Ready", Clickable: true, Visible: true}}}}, nil
}

// The source/app/registry/evidence paths are real. Only the device adapter is
// fake: this tests worker routing and both authority layers without claiming
// a remote physical-device/TLS acceptance run.
func routingUIFixture(t *testing.T) (*AppExecutor, protocol.Operation, *sqlite.Store, *routingUIProvider) {
	t.Helper()
	ctx := context.Background()
	e, created, _, db := executorFixture(t)
	executePrepared(t, e, created)
	lease, err := db.Get(ctx, created.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := e.loadPackage(created.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	var pkg remotesource.Package
	if err = strictJSON(data, &pkg); err != nil {
		t.Fatal(err)
	}
	created.LeaseID = ulid.Make().String()
	lease.ID = created.LeaseID
	lease.Sources = nil
	lease.Resources = nil
	lease.Components = nil
	avd := filepath.Join(e.Home, "leases", lease.ID, "fixture-avd")
	lease.Runtimes = []domain.Runtime{{Name: "phone", LeaseID: lease.ID, Type: "android-emulator", Android: &domain.AndroidEmulator{Template: "fixture", AVDName: "fixture", AVDHome: avd, AVDPath: filepath.Join(avd, "fixture.avd")}}}
	lease.Applications = []domain.Application{{Name: "mobile", Runtime: "phone", Package: "com.example.fixture"}}
	if err = db.Reserve(ctx, lease, 0); err != nil {
		t.Fatal(err)
	}
	lease, err = db.Get(ctx, lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	lease.Runtimes[0].Started = true
	if err = db.Save(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if err = e.retainPackage(lease.ID, pkg); err != nil {
		t.Fatal(err)
	}
	provider := &routingUIProvider{}
	prior := e.Factory
	e.Factory = func(m *domain.Management) *app.Service { s := prior(m); s.AndroidUI = provider; return s }
	return e, created, db, provider
}

func routingUISnapshot(t *testing.T, e *AppExecutor, created protocol.Operation) app.UIResult {
	t.Helper()
	op := nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "snapshot", Application: "mobile"}})
	result := executePrepared(t, e, op)
	var response struct {
		Value         app.UIResult `json:"value"`
		EndpointScope string       `json:"endpoint_scope"`
	}
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if response.Value.Snapshot == nil || response.Value.Snapshot.Serial != "emulator-5554" || response.Value.Run.Status != "passed" || response.EndpointScope != "worker-local" {
		t.Fatalf("worker snapshot response: %s", result.Payload)
	}
	return response.Value
}

func TestWorkerTypedUIRoutesSnapshotAndInputThroughManagedApp(t *testing.T) {
	e, created, db, p := routingUIFixture(t)
	snap := routingUISnapshot(t, e, created)
	if p.calls != 1 || p.runtime.LeaseID != created.LeaseID || p.runtime.Name != "phone" || p.request.Package != "com.example.fixture" {
		t.Fatalf("snapshot lost local device selection: %+v", p)
	}
	op := nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "n1"}})
	result := executePrepared(t, e, op)
	var response struct {
		Value app.UIResult `json:"value"`
	}
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if p.calls != 2 || p.request.Operation != "tap" || p.request.ExpectedFingerprint != "safe-node-fingerprint" || p.request.ExpectedBackend != "fixture-verified-helper" || p.request.Package != "com.example.fixture" || !response.Value.Observation.ActionPerformed || response.Value.Run.Status != "passed" {
		t.Fatalf("semantic request bypassed snapshot authority: %+v %s", p.request, result.Payload)
	}
	runs, err := db.Runs(context.Background(), created.LeaseID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("UI runs not durable: %+v %v", runs, err)
	}
	artifacts, err := db.Artifacts(context.Background(), created.LeaseID)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, results := 0, 0
	for _, a := range artifacts {
		if a.Kind == "ui-snapshot" {
			snapshots++
		}
		if a.Kind == "ui-result" {
			results++
		}
	}
	if snapshots != 1 || results != 2 {
		t.Fatalf("UI evidence route omitted artifacts: %+v", artifacts)
	}
	// Uncertain recovery must not repeat the input even with valid assignment.
	recovered := e.Recover(context.Background(), op)
	if recovered.State != "uncertain" || p.calls != 2 {
		t.Fatalf("uncertain UI input replayed: %+v calls%d", recovered, p.calls)
	}
}

func TestWorkerTypedUIRetainsAssignmentStaleDeviceAndFenceChecks(t *testing.T) {
	for _, mode := range []string{"assignment", "missing-node", "wrong-runtime", "snapshot-tamper", "device-refusal", "operation-fence"} {
		t.Run(mode, func(t *testing.T) {
			e, created, db, p := routingUIFixture(t)
			ctx := context.Background()
			snap := routingUISnapshot(t, e, created)
			op := nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "n1"}})
			switch mode {
			case "assignment":
				op.Epoch++
			case "missing-node":
				op = nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "tap", Snapshot: snap.Snapshot.ID, Node: "missing"}})
			case "wrong-runtime":
				op = nextOperation(created, "ui", Request{UI: app.UIOptions{Operation: "tap", Runtime: "different-device", Snapshot: snap.Snapshot.ID, Node: "n1"}})
			case "snapshot-tamper":
				path := filepath.Join(e.Home, "leases", created.LeaseID, "artifacts", snap.Snapshot.ID, "snapshot.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, append(data, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "device-refusal":
				p.rejectDevice = true
			}
			err := e.Prepare(ctx, op)
			if mode == "assignment" {
				if err == nil || p.calls != 1 {
					t.Fatalf("wrong assignment reached UI: %v calls%d", err, p.calls)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			executeCtx := ctx
			if mode == "operation-fence" {
				_, release, err := db.AcquireContext(ctx, created.LeaseID, "other-local-operation", time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := release(); err != nil {
						t.Error(err)
					}
				}()
				bounded, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
				defer cancel()
				executeCtx = bounded
			}
			result := e.Execute(executeCtx, op)
			if result.State == "completed" {
				t.Fatalf("%s accepted: %+v", mode, result)
			}
			wantCalls := 1
			if mode == "device-refusal" {
				wantCalls = 2
				if !strings.Contains(string(result.Payload), "owned device console identity mismatch") {
					t.Fatalf("device refusal lost: %s", result.Payload)
				}
			}
			if p.calls != wantCalls {
				t.Fatalf("%s crossed device boundary: calls%d want%d", mode, p.calls, wantCalls)
			}
			runs, err := db.Runs(ctx, created.LeaseID)
			if err != nil {
				t.Fatal(err)
			}
			wantRuns := 1
			if mode == "device-refusal" {
				wantRuns = 2
			}
			if len(runs) != wantRuns {
				t.Fatalf("%s bypassed local preflight: %+v", mode, runs)
			}
			for _, run := range runs {
				if run.Status == "running" {
					t.Fatalf("proven refusal left live-input barrier: %+v", run)
				}
			}
		})
	}
}
