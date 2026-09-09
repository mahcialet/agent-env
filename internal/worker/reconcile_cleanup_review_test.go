package worker

import (
	"context"
	"encoding/json"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	controlstore "github.com/mahcialet/agent-env/internal/controlplane/store"
	"github.com/mahcialet/agent-env/internal/domain"
	"path/filepath"
	"testing"
)

type reconcileControllerTransport struct {
	fakeTransport
	store *controlstore.Store
}

func (t reconcileControllerTransport) Complete(ctx context.Context, result protocol.Result) error {
	_, err := t.store.Complete(ctx, result)
	return err
}
func TestCompensatedCreateReconcileCompletesRealControllerRelease(t *testing.T) {
	ctx := context.Background()
	executor, fixture, process, local := executorFixture(t)
	controller, err := controlstore.Open(filepath.Join(t.TempDir(), "controller.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	journal, err := OpenJournal(executor.Home, fixture.HostID)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err = journal.Bind(ctx, controller.ID); err != nil {
		t.Fatal(err)
	}
	var envelope protocol.CreateRequest
	if err = json.Unmarshal(fixture.Payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, err = controller.Register(ctx, protocol.RegisterRequest{WorkerIdentity: journal.Identity, ProtocolVersion: protocol.Version, ProductVersion: "test", OS: "linux", Arch: "amd64", Capacity: protocol.Capacity{MaxLeases: 1}, Capabilities: envelope.RequiredCapabilities}); err != nil {
		t.Fatal(err)
	}
	executor.Factory = func(m *domain.Management) *app.Service {
		return &app.Service{Store: local, Process: compensatedCreateProcess{process}, Management: m}
	}
	create, err := controller.Create(ctx, envelope)
	if err != nil {
		t.Fatal(err)
	}
	runner := Runner{Journal: journal, Executor: executor, Transport: &reconcileControllerTransport{store: controller}}
	perform := func(expected protocol.Operation) {
		t.Helper()
		op, err := controller.Poll(ctx, journal.Identity)
		if err != nil || op == nil || op.ID != expected.ID {
			t.Fatalf("dispatch: %+v %v", op, err)
		}
		receipt, err := journal.Receive(ctx, *op)
		if err != nil {
			t.Fatal(err)
		}
		if err = runner.handle(ctx, receipt); err != nil {
			t.Fatal(err)
		}
	}
	perform(create)
	lease, err := controller.GetLease(ctx, create.LeaseID)
	if err != nil || lease.State != "QUARANTINED" {
		t.Fatalf("compensated create state: %+v %v", lease, err)
	}

	// A recovered receipt can report uncertainty, but a stored released snapshot
	// alone must neither assert fresh proof nor make controller acknowledgement
	// impossible. A subsequent explicit reconcile observes current resources.
	interrupted, err := controller.Submit(ctx, protocol.SubmitRequest{OperationID: "interrupted-reconcile", LeaseID: create.LeaseID, Kind: "reconcile", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	dispatched, err := controller.Poll(ctx, journal.Identity)
	if err != nil || dispatched == nil || dispatched.ID != interrupted.ID {
		t.Fatalf("interrupted dispatch: %v", err)
	}
	receipt, err := journal.Receive(ctx, *dispatched)
	if err != nil {
		t.Fatal(err)
	}
	if err = journal.Advance(ctx, receipt.Operation.ID, "received", "prepared"); err != nil {
		t.Fatal(err)
	}
	if err = journal.Advance(ctx, receipt.Operation.ID, "prepared", "effect_started"); err != nil {
		t.Fatal(err)
	}
	receipt, err = journal.Get(ctx, interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = runner.handle(ctx, receipt); err != nil {
		t.Fatalf("uncertain recovered reconcile rejected: %v", err)
	}
	recovered, err := controller.GetOperation(ctx, interrupted.ID)
	if err != nil || recovered.Result == nil || recovered.Result.State != "uncertain" || recovered.Result.CleanupConfirmed {
		t.Fatalf("recovery invented fresh release proof: %+v %v", recovered, err)
	}
	reconcile, err := controller.Submit(ctx, protocol.SubmitRequest{OperationID: "prove-reconcile-release", LeaseID: create.LeaseID, Kind: "reconcile", Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	perform(reconcile)
	lease, err = controller.GetLease(ctx, create.LeaseID)
	if err != nil || lease.State != "RELEASED" {
		t.Fatalf("reconcile not accepted as release: %+v %v", lease, err)
	}
	operation, err := controller.GetOperation(ctx, reconcile.ID)
	if err != nil || operation.State != "completed" || operation.Result == nil || !operation.Result.CleanupConfirmed {
		t.Fatalf("reconcile receipt stuck: %+v %v", operation, err)
	}
	if process.starts != 1 {
		t.Fatalf("reconcile replayed allocation: %d", process.starts)
	}
}
