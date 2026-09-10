package cdp

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
)

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
