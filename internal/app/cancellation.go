package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

func (s *Service) runWithCancellation(ctx context.Context, runID string, spec execx.Command) (execx.Result, error) {
	requested, err := s.Store.RunCancellationRequested(ctx, runID)
	if err != nil {
		return execx.Result{ExitCode: -1}, fmt.Errorf("read command cancellation: %w", err)
	}
	if requested {
		return execx.Result{ExitCode: -1}, context.Canceled
	}
	commandCtx, cancel := context.WithCancelCause(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-commandCtx.Done():
				return
			case <-ticker.C:
				pollCtx, pollCancel := context.WithTimeout(commandCtx, time.Second)
				requested, err := s.Store.RunCancellationRequested(pollCtx, runID)
				pollCancel()
				if commandCtx.Err() != nil {
					return
				}
				if err != nil {
					cancel(fmt.Errorf("observe command cancellation: %w", err))
					return
				}
				if requested {
					cancel(context.Canceled)
					return
				}
			}
		}
	}()
	result, runErr := s.Runner.Run(commandCtx, spec)
	cause := context.Cause(commandCtx)
	cancel(nil)
	<-done
	return result, errors.Join(runErr, cause)
}

// The acquired lock is returned directly to Destroy: no new command can start
// between acknowledged cancellation, evidence finalization, and cleanup.
func (s *Service) acquireDestroyAfterCancellation(ctx context.Context, leaseID string) (context.Context, func() error, error) {
	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
	targets := map[string]bool{}
	requested := false
	for {
		if err := s.checkManagement(waitCtx, leaseID); err != nil {
			return nil, nil, err
		}
		runs, err := s.Store.Runs(waitCtx, leaseID)
		if err != nil {
			return nil, nil, err
		}
		if !requested {
			for _, run := range runs {
				if run.Status == "running" {
					targets[run.ID] = true
				}
			}
		}
		// Bind the final operation to the caller's context, not the short wait
		// deadline, which must not accidentally cancel the subsequent cleanup.
		operation, release, acquireErr := s.acquireWithinWait(ctx, waitCtx, leaseID)
		if acquireErr == nil {
			current, readErr := s.Store.Runs(operation, leaseID)
			if readErr != nil {
				_ = release()
				return nil, nil, readErr
			}
			for _, run := range current {
				if run.Status == "running" {
					_ = release()
					return nil, nil, fmt.Errorf("command run %s is still recorded as running without an active operation lock; inspect the prior process before cleanup (force cannot bypass this check)", run.ID)
				}
			}
			return operation, release, nil
		}
		if errors.Is(acquireErr, ErrManagementAuthority) {
			return nil, nil, acquireErr
		}
		if len(targets) == 0 {
			return nil, nil, acquireErr
		}
		if !requested {
			if err := s.checkManagement(waitCtx, leaseID); err != nil {
				return nil, nil, err
			}
			for runID := range targets {
				if err := s.Store.RequestRunCancel(waitCtx, runID); err != nil {
					return nil, nil, err
				}
			}
			requested = true
		} else {
			for _, run := range runs {
				if run.Status == "running" && !targets[run.ID] {
					return nil, nil, fmt.Errorf("a newer command run %s started while waiting; retry destroy without canceling that run implicitly", run.ID)
				}
			}
		}
		select {
		case <-waitCtx.Done():
			return nil, nil, fmt.Errorf("named command cancellation was not confirmed before cleanup deadline; no resources removed: %w", waitCtx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *Service) acquireWithinWait(parent, wait context.Context, leaseID string) (context.Context, func() error, error) {
	attemptCtx, cancelAttempt := context.WithCancel(parent)
	stopDeadline := context.AfterFunc(wait, cancelAttempt)
	operation, release, err := s.acquireManagedOperation(attemptCtx, leaseID)
	stopped := stopDeadline()
	if err != nil {
		cancelAttempt()
		return nil, nil, err
	}
	if !stopped || wait.Err() != nil {
		cancelAttempt()
		return nil, nil, errors.Join(wait.Err(), release())
	}
	return operation, func() error { err := release(); cancelAttempt(); return err }, nil
}
