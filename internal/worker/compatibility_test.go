package worker

import (
	"context"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

type incompatibleTransport struct {
	fakeTransport
	registered bool
	polled     bool
}

func (t *incompatibleTransport) Info(context.Context) (protocol.Info, error) {
	return protocol.Info{ControllerID: "controller", ProtocolVersion: protocol.Version, ProductVersion: "new-version"}, nil
}
func (t *incompatibleTransport) Register(_ context.Context, r protocol.RegisterRequest) (protocol.RegisterResult, error) {
	t.registered = true
	return protocol.RegisterResult{Info: protocol.Info{ControllerID: "controller", ProtocolVersion: protocol.Version, ProductVersion: "new-version"}, Host: protocol.Host{WorkerIdentity: r.WorkerIdentity, Compatible: false}}, nil
}
func (t *incompatibleTransport) Poll(context.Context, protocol.PollRequest) (protocol.PollResult, error) {
	t.polled = true
	return protocol.PollResult{}, nil
}
func TestIncompatibleWorkerRegistersInventoryBeforeRefusingEffects(t *testing.T) {
	j, e := OpenJournal(t.TempDir(), "host")
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	transport := &incompatibleTransport{}
	executor := &fakeExecutor{}
	r := &Runner{Journal: j, Transport: transport, Executor: executor, Registration: protocol.RegisterRequest{ProductVersion: "old-version"}}
	e = r.Run(context.Background())
	if e == nil || !strings.Contains(e.Error(), "incompatible") || !transport.registered || transport.polled || executor.effects != 0 || executor.recoveries != 0 {
		t.Fatalf("incompatible worker did not remain visible without effects: %v %+v %+v", e, transport, executor)
	}
}
