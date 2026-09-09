package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/client"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/evidence"
)

type Transport interface {
	Info(context.Context) (protocol.Info, error)
	Register(context.Context, protocol.RegisterRequest) (protocol.RegisterResult, error)
	Heartbeat(context.Context, protocol.WorkerIdentity) error
	Poll(context.Context, protocol.PollRequest) (protocol.PollResult, error)
	Complete(context.Context, protocol.Result) error
}

// Executor must never perform runtime effects in Prepare. Recover must not
// blindly replay an effect whose result was not durably recorded.
type Executor interface {
	Prepare(context.Context, protocol.Operation) error
	Execute(context.Context, protocol.Operation) protocol.Result
	Recover(context.Context, protocol.Operation) protocol.Result
}

// CreateBoundaryExecutor must invoke the durable callback before any lease
// reservation or runtime effect; a failed callback prohibits all such effects.
type CreateBoundaryExecutor interface {
	ExecuteWithEffectBoundary(context.Context, protocol.Operation, func(context.Context) error) protocol.Result
}
type Runner struct {
	Journal           *Journal
	Transport         Transport
	Executor          Executor
	Registration      protocol.RegisterRequest
	Download          func(context.Context, protocol.Operation) error
	Upload            func(context.Context, protocol.Result) error
	RetryDelay        time.Duration
	HeartbeatInterval time.Duration
}

func pause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Run opens no inbound sockets. Loss of the controller causes reconnect only;
// it never invokes GC or treats unreachable workloads as absent.
func (r *Runner) Run(ctx context.Context) error {
	if r.Journal == nil || r.Transport == nil || r.Executor == nil {
		return errors.New("worker runner requires journal, transport and executor")
	}
	delay := r.RetryDelay
	if delay <= 0 {
		delay = time.Second
	}
	for ctx.Err() == nil {
		info, e := r.Transport.Info(ctx)
		if e != nil {
			var status *client.StatusError
			if errors.As(e, &status) && !status.Temporary() {
				return e
			}
			if e = pause(ctx, delay); e != nil {
				return e
			}
			continue
		}
		if e = r.Journal.Bind(ctx, info.ControllerID); e != nil {
			return e
		}
		registration := r.Registration
		registration.WorkerIdentity = r.Journal.Identity
		registration.ProtocolVersion = protocol.Version
		registered, e := r.Transport.Register(ctx, registration)
		if e != nil {
			var status *client.StatusError
			if errors.As(e, &status) && !status.Temporary() {
				return e
			}
			if e = pause(ctx, delay); e != nil {
				return e
			}
			continue
		}
		if info.ProtocolVersion != protocol.Version || info.ProductVersion != r.Registration.ProductVersion || !registered.Host.Compatible || registered.ControllerID != info.ControllerID || registered.Host.WorkerIdentity != r.Journal.Identity {
			return errors.New("incompatible controller/worker version or registration identity; worker inventory retained without dispatch")
		}
		e = r.session(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e = pause(ctx, delay); e != nil {
			return e
		}
	}
	return ctx.Err()
}
func (r *Runner) session(ctx context.Context) error {
	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	interval := r.HeartbeatInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	failed := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if e := pause(heartbeatCtx, interval); e != nil {
				return
			}
			if e := r.Transport.Heartbeat(heartbeatCtx, r.Journal.Identity); e != nil {
				failed <- e
				return
			}
		}
	}()
	defer func() { cancel(); <-done }()
	fence := func() error {
		select {
		case e := <-failed:
			return e
		default:
			return nil
		}
	}
	pending, e := r.Journal.Pending(ctx)
	if e != nil {
		return e
	}
	for _, receipt := range pending {
		if e = r.handleWithFence(ctx, receipt, fence); e != nil {
			return e
		}
	}
	for {
		select {
		case e := <-failed:
			return e
		default:
		}
		polled, e := r.Transport.Poll(ctx, protocol.PollRequest{WorkerIdentity: r.Journal.Identity, WaitSeconds: 15})
		if e != nil {
			return e
		}
		// A concurrent heartbeat failure fences new effects. Already dispatched work
		// can finish and retain its result; heartbeat errors do not cancel workloads.
		select {
		case e := <-failed:
			return e
		default:
		}
		if polled.Operation == nil {
			continue
		}
		receipt, e := r.Journal.Receive(ctx, *polled.Operation)
		if e != nil {
			return e
		}
		if e = r.handleWithFence(ctx, receipt, fence); e != nil {
			return e
		}
	}
}
func (r *Runner) handle(ctx context.Context, receipt Receipt) error {
	return r.handleWithFence(ctx, receipt, func() error { return nil })
}
func (r *Runner) handleWithFence(ctx context.Context, receipt Receipt, fence func() error) error {
	op := receipt.Operation
	if receipt.Result != nil {
		return r.deliver(ctx, receipt)
	}
	var result protocol.Result
	if receipt.State == "effect_started" {
		result = r.Executor.Recover(ctx, op)
	} else if receipt.State == "received" || receipt.State == "prepared" {
		if op.Kind == "destroy" {
			var request Request
			if err := strictJSON(op.Payload, &request); err == nil && operationValid(op) == nil && validateRequest(op, request) == nil && !request.DryRun {
				proved, err := r.Journal.ProvesNoEffects(ctx, op)
				if err != nil {
					return err
				}
				if proved {
					result = protocol.Result{WorkerIdentity: r.Journal.Identity, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "released", CleanupConfirmed: true, Payload: json.RawMessage(`{"absence_proof":"durable pre-effect receipts","endpoint_scope":"worker-local"}`)}
					if err = r.Journal.StoreResult(ctx, op.ID, result); err != nil {
						return err
					}
					receipt.Result = &result
					receipt.State = "result_pending"
					return r.deliver(ctx, receipt)
				}
			}
		}
		var err error
		if r.Download != nil {
			err = r.Download(ctx, op)
		}
		if err == nil {
			err = r.Executor.Prepare(ctx, op)
		}
		if err != nil {
			// Preflight failure has no local runtime effect. Never infer RELEASED here.
			message := evidence.RedactString(err.Error(), evidence.InheritedSecrets())
			if len(message) > 16<<10 {
				message = "worker preflight failed; diagnostic exceeded the remote metadata limit"
			}
			payload, _ := json.Marshal(map[string]any{"error": message, "effects_started": false})
			result = protocol.Result{State: "failed", Payload: payload}
		} else {
			if receipt.State == "received" {
				if err = r.Journal.Advance(ctx, op.ID, "received", "prepared"); err != nil {
					return err
				}
			}
			if err = fence(); err != nil {
				return err
			}
			if boundary, ok := r.Executor.(CreateBoundaryExecutor); ok && op.Kind == "create" {
				result = boundary.ExecuteWithEffectBoundary(ctx, op, func(effectCtx context.Context) error {
					if err := fence(); err != nil {
						return err
					}
					return r.Journal.Advance(effectCtx, op.ID, "prepared", "effect_started")
				})
			} else {
				if err = r.Journal.Advance(ctx, op.ID, "prepared", "effect_started"); err != nil {
					return err
				}
				result = r.Executor.Execute(ctx, op)
			}
		}
	} else {
		return errors.New("journal state has no recoverable result")
	}
	result.WorkerIdentity = r.Journal.Identity
	result.OperationID = op.ID
	result.LeaseID = op.LeaseID
	result.Epoch = op.Epoch
	// Persist even after command context cancellation; no effect may be repeated
	// merely because shutdown or the connection prevented result publication.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if e := r.Journal.StoreResult(persistCtx, op.ID, result); e != nil {
		return e
	}
	receipt.Result = &result
	receipt.State = "result_pending"
	return r.deliver(ctx, receipt)
}
func (r *Runner) deliver(ctx context.Context, receipt Receipt) error {
	result := *receipt.Result
	// The stored result belongs to the original effect, but the reconnecting
	// process must authenticate the current incarnation for upload/ack.
	result.WorkerIdentity = r.Journal.Identity
	if r.Upload != nil {
		if e := r.Upload(ctx, result); e != nil {
			return e
		}
	}
	if e := r.Transport.Complete(ctx, result); e != nil {
		return e
	}
	if receipt.State == "completed" {
		return nil
	}
	return r.Journal.Advance(ctx, receipt.Operation.ID, "result_pending", "completed")
}
