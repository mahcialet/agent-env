package cli

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// The owner must call stop before closing the transport or controller. stop
// cancels and joins; signaling cancellation alone never means a callback exited.
func startManifestFixtureHeartbeat(parent context.Context, ticks <-chan time.Time, beat func(context.Context) error) (<-chan error, func() error) {
	ctx, cancel := context.WithCancel(parent)
	ready := make(chan error, 1)
	done := make(chan struct{})
	var result error
	go func() {
		defer close(done)
		first := true
		for {
			err := beat(ctx)
			if first {
				ready <- err
				first = false
			}
			if err != nil {
				if ctx.Err() == nil || !errors.Is(err, ctx.Err()) {
					result = err
				}
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticks:
			}
		}
	}()
	return ready, func() error { cancel(); <-done; return result }
}

func TestManifestHeartbeatStopJoinsActiveCallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time)
		entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
		calls := 0
		ready, stop := startManifestFixtureHeartbeat(context.Background(), ticks, func(ctx context.Context) error {
			calls++
			if calls == 1 {
				return nil
			}
			close(entered)
			<-ctx.Done()
			close(canceled)
			<-release
			return ctx.Err()
		})
		defer stop()
		if err := <-ready; err != nil {
			t.Fatal(err)
		}
		ticks <- time.Time{}
		<-entered
		stopped := make(chan error, 1)
		go func() { stopped <- stop() }()
		<-canceled
		// Quiescence lets a broken cancel-only stop finish sending its result;
		// merely seeing cancellation does not establish that scheduling step.
		synctest.Wait()
		// The callback acknowledged cancellation but is still deliberately held.
		select {
		case err := <-stopped:
			close(release)
			t.Fatalf("stop returned before callback completion: %v", err)
		default:
		}
		close(release)
		if err := <-stopped; err != nil {
			t.Fatal(err)
		}
		if calls != 2 {
			t.Fatalf("unexpected heartbeat count %d", calls)
		}
	})
}

func TestManifestHeartbeatReportsStartupAndLaterFailure(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			ticks := make(chan time.Time)
			failed := make(chan struct{})
			injected := errors.New("injected heartbeat failure")
			calls := 0
			ready, stop := startManifestFixtureHeartbeat(context.Background(), ticks, func(context.Context) error {
				calls++
				if calls == failAt {
					close(failed)
					return injected
				}
				return nil
			})
			defer stop()
			startup := <-ready
			if failAt == 1 && !errors.Is(startup, injected) {
				t.Fatalf("startup error lost: %v", startup)
			}
			if failAt == 2 {
				if startup != nil {
					t.Fatal(startup)
				}
				ticks <- time.Time{}
			}
			<-failed
			if err := stop(); !errors.Is(err, injected) {
				t.Fatalf("heartbeat error lost at cleanup: %v", err)
			}
		})
	}
}
