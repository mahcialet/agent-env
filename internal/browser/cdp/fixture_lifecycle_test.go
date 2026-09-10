package cdp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// Cancellation races with c.call's socket-close callback and write deadline.
// These exact transport errors are valid only with the expected expired context;
// context state alone must never excuse a different operation failure.
func fixtureCancellationError(ctx context.Context, err, cause error) bool {
	if errors.Is(err, cause) {
		return true
	}
	if err == nil || !errors.Is(ctx.Err(), cause) {
		return false
	}
	switch err.Error() {
	case "CDP disconnected", "CDP disconnected; effect may be uncertain", "CDP write failed; effect may be uncertain":
		return true
	}
	return false
}

func TestFixtureCancellationErrorDiscriminatesReturnedCause(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			var ctx context.Context
			if cause == context.Canceled {
				c, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = c
			} else {
				c, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
				defer cancel()
				ctx = c
			}
			for _, message := range []string{"CDP disconnected", "CDP disconnected; effect may be uncertain", "CDP write failed; effect may be uncertain"} {
				err := errors.New(message)
				if !fixtureCancellationError(ctx, err, cause) {
					t.Fatalf("valid cancellation transport outcome rejected: %v", err)
				}
				if fixtureCancellationError(context.Background(), err, cause) {
					t.Fatalf("transport error without cancellation accepted: %v", err)
				}
			}
			if !fixtureCancellationError(ctx, cause, cause) {
				t.Fatal("context cause rejected")
			}
			for _, err := range []error{nil, errors.New("unrelated fixture failure"), errors.New("CDP Runtime.evaluate failed (-1)"), errors.New("prefix CDP disconnected"), errors.New("CDP disconnected suffix")} {
				if fixtureCancellationError(ctx, err, cause) {
					t.Fatalf("unrelated result accepted with canceled context: %v", err)
				}
			}
		})
	}
}

// Admission closes before waiting, so late HTTP handlers cannot add work after
// cleanup has observed an empty group. Hijacking does not transfer ownership.
type fixtureWork struct {
	mu      sync.Mutex
	stopped bool
	wg      sync.WaitGroup
}

func (w *fixtureWork) enter() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return false
	}
	w.wg.Add(1)
	return true
}
func (w *fixtureWork) stop() { w.mu.Lock(); w.stopped = true; w.mu.Unlock() }
func (w *fixtureWork) join() { w.wg.Wait() }

type fixtureServer struct {
	*httptest.Server
	work fixtureWork
	once sync.Once
}

func newFixtureServer(handler http.Handler) *fixtureServer {
	s := &fixtureServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.work.enter() {
			http.Error(w, "fixture closed", http.StatusServiceUnavailable)
			return
		}
		defer s.work.wg.Done()
		handler.ServeHTTP(w, r)
	}))
	return s
}

// Callers close clients and release deliberately blocked callbacks first.
func (s *fixtureServer) Close() {
	s.once.Do(func() { s.work.stop(); s.Server.Close(); s.work.join() })
}

func closeFixtureConnection(c *connection) {
	c.close()
	<-c.readExited
}

func TestFixtureWorkJoinWaitsForCallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var work fixtureWork
		if !work.enter() {
			t.Fatal("initial admission rejected")
		}
		release := make(chan struct{})
		go func() { <-release; work.wg.Done() }()
		work.stop()
		if work.enter() {
			t.Fatal("admitted work after shutdown")
		}
		joined := make(chan struct{})
		go func() { work.join(); close(joined) }()
		synctest.Wait()
		select {
		case <-joined:
			t.Error("join returned before callback completion")
		default:
		}
		close(release)
		synctest.Wait()
		select {
		case <-joined:
		default:
			t.Fatal("join failed after callback completion")
		}
		work.join()
	})
}
