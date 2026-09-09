package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

type fakeExecutor struct {
	effects, recoveries int
	preflight           error
}

func (f *fakeExecutor) Prepare(context.Context, protocol.Operation) error { return f.preflight }
func (f *fakeExecutor) Execute(context.Context, protocol.Operation) protocol.Result {
	f.effects++
	return protocol.Result{State: "completed"}
}
func (f *fakeExecutor) Recover(context.Context, protocol.Operation) protocol.Result {
	f.recoveries++
	return protocol.Result{State: "uncertain"}
}

type fakeTransport struct {
	acks int
	fail bool
}

func (f *fakeTransport) Info(context.Context) (protocol.Info, error) { return protocol.Info{}, nil }
func (f *fakeTransport) Register(context.Context, protocol.RegisterRequest) (protocol.RegisterResult, error) {
	return protocol.RegisterResult{}, nil
}
func (f *fakeTransport) Heartbeat(context.Context, protocol.WorkerIdentity) error { return nil }
func (f *fakeTransport) Poll(context.Context, protocol.PollRequest) (protocol.PollResult, error) {
	return protocol.PollResult{}, nil
}
func (f *fakeTransport) Complete(context.Context, protocol.Result) error {
	f.acks++
	if f.fail {
		return errors.New("ack lost")
	}
	return nil
}
func runnerFixture(t *testing.T) (*Runner, Receipt, *fakeExecutor, *fakeTransport) {
	t.Helper()
	ctx := context.Background()
	j, e := OpenJournal(t.TempDir(), "host")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { j.Close() })
	if e = j.Bind(ctx, "controller"); e != nil {
		t.Fatal(e)
	}
	op := protocol.Operation{ID: "operation", LeaseID: "lease", ControllerID: j.Identity.ControllerID, HostID: j.Identity.HostID, HostInstanceID: j.Identity.HostInstanceID, Epoch: 1, Kind: "test"}
	receipt, e := j.Receive(ctx, op)
	if e != nil {
		t.Fatal(e)
	}
	executor := &fakeExecutor{}
	transport := &fakeTransport{}
	return &Runner{Journal: j, Executor: executor, Transport: transport}, receipt, executor, transport
}
func TestUploadFailureAndLostAcknowledgementNeverReplay(t *testing.T) {
	ctx := context.Background()
	r, receipt, executor, transport := runnerFixture(t)
	uploads := 0
	r.Upload = func(context.Context, protocol.Result) error {
		uploads++
		if uploads == 1 {
			return errors.New("CAS unavailable")
		}
		return nil
	}
	if e := r.handle(ctx, receipt); e == nil {
		t.Fatal("upload failure hidden")
	}
	if executor.effects != 1 || transport.acks != 0 {
		t.Fatal("invalid effect/upload ordering")
	}
	stored, e := r.Journal.Get(ctx, receipt.Operation.ID)
	if e != nil || stored.Result == nil || stored.State != "result_pending" {
		t.Fatalf("result not durable: %+v %v", stored, e)
	}
	transport.fail = true
	if e = r.handle(ctx, stored); e == nil {
		t.Fatal("ack loss hidden")
	}
	transport.fail = false
	if e = r.handle(ctx, stored); e != nil {
		t.Fatal(e)
	}
	duplicate, e := r.Journal.Receive(ctx, receipt.Operation)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.handle(ctx, duplicate); e != nil {
		t.Fatal(e)
	}
	if executor.effects != 1 || executor.recoveries != 0 || transport.acks != 3 {
		t.Fatalf("mutation replayed: %+v %+v", executor, transport)
	}
}
func TestEffectWithoutResultRemainsUncertain(t *testing.T) {
	ctx := context.Background()
	r, receipt, executor, _ := runnerFixture(t)
	if e := r.Journal.Advance(ctx, receipt.Operation.ID, "received", "prepared"); e != nil {
		t.Fatal(e)
	}
	if e := r.Journal.Advance(ctx, receipt.Operation.ID, "prepared", "effect_started"); e != nil {
		t.Fatal(e)
	}
	receipt, e := r.Journal.Get(ctx, receipt.Operation.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.handle(ctx, receipt); e != nil {
		t.Fatal(e)
	}
	stored, e := r.Journal.Get(ctx, receipt.Operation.ID)
	if e != nil {
		t.Fatal(e)
	}
	if executor.effects != 0 || executor.recoveries != 1 || stored.Result.State != "uncertain" || stored.Result.CleanupConfirmed {
		t.Fatal("lost effect replayed or absence invented")
	}
}
func TestPreflightFailureHasNoEffects(t *testing.T) {
	ctx := context.Background()
	r, receipt, executor, _ := runnerFixture(t)
	executor.preflight = errors.New("wrong source digest")
	if e := r.handle(ctx, receipt); e != nil {
		t.Fatal(e)
	}
	stored, e := r.Journal.Get(ctx, receipt.Operation.ID)
	if e != nil {
		t.Fatal(e)
	}
	if executor.effects != 0 || stored.Result.State != "failed" || stored.Result.CleanupConfirmed {
		t.Fatal("preflight failure caused effects or release")
	}
}

func TestLostHeartbeatFencesPreparedEffects(t *testing.T) {
	ctx := context.Background()
	r, receipt, executor, _ := runnerFixture(t)
	err := r.handleWithFence(ctx, receipt, func() error { return errors.New("heartbeat failed during prepare") })
	if err == nil || executor.effects != 0 {
		t.Fatal("disconnected worker began new effects")
	}
	stored, err := r.Journal.Get(ctx, receipt.Operation.ID)
	if err != nil || stored.State != "prepared" || stored.Result != nil {
		t.Fatalf("pre-effect receipt not retained: %+v %v", stored, err)
	}
	if err = r.handle(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if executor.effects != 1 {
		t.Fatal("reconnected prepared operation did not execute once")
	}
}
