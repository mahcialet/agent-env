package android

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// testTCPServer owns both accept and handler completion. Closing a listener
// alone leaves accepted connections (including partial requests) running.
type testTCPServer struct {
	listener    net.Listener
	accepted    chan struct{}
	mu          sync.Mutex
	connections map[net.Conn]struct{}
	handlers    sync.WaitGroup
	once        sync.Once
}

func ownTestTCP(listener net.Listener, handle func(net.Conn)) *testTCPServer {
	s := &testTCPServer{listener: listener, accepted: make(chan struct{}), connections: make(map[net.Conn]struct{})}
	go func() {
		defer close(s.accepted)
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.connections[c] = struct{}{}
			s.mu.Unlock()
			s.handlers.Add(1)
			go func() {
				defer s.handlers.Done()
				defer func() {
					c.Close()
					s.mu.Lock()
					delete(s.connections, c)
					s.mu.Unlock()
				}()
				handle(c)
			}()
		}
	}()
	return s
}

func (s *testTCPServer) close() {
	s.once.Do(func() {
		s.listener.Close()
		<-s.accepted // All Add calls precede Wait, even with an in-flight Accept.
		s.mu.Lock()
		for c := range s.connections {
			c.Close()
		}
		s.mu.Unlock()
		s.handlers.Wait()
	})
}

func TestTCPFixtureCleanupJoinsPartialRequest(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, exited := make(chan struct{}), make(chan struct{})
	s := ownTestTCP(l, func(c net.Conn) {
		defer close(exited)
		close(entered)
		_, _ = io.ReadFull(c, make([]byte, 14))
	})
	t.Cleanup(s.close)
	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// Rescue even a deliberately broken close implementation during fail-before.
	defer func() { c.Close(); <-exited }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never entered")
	}
	if _, err := c.Write([]byte("0")); err != nil {
		t.Fatal(err)
	}
	s.close()
	select {
	case <-exited:
	default:
		t.Fatal("cleanup returned with a partial-request handler still active")
	}
	s.close() // Idempotent teardown owns completion on every call.
}
