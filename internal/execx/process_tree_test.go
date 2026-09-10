package execx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessTreeHelper(t *testing.T) {
	mode := os.Getenv("AGENT_ENV_TREE_HELPER")
	if mode == "" {
		return
	}
	marker := os.Getenv("AGENT_ENV_TREE_MARKER")
	if mode == "leaf" {
		conn, err := net.DialTimeout("tcp", os.Getenv("AGENT_ENV_TREE_OBSERVER"), 5*time.Second)
		if err != nil {
			os.Exit(7)
		}
		if _, err := conn.Write([]byte{'R'}); err != nil {
			os.Exit(8)
		}
		if err := os.WriteFile(marker+".ready", nil, 0600); err != nil {
			os.Exit(8)
		}
		// The leaf owns this socket for its entire remaining lifetime. Echoes
		// support an independent live control; EOF follows termination, not a
		// delayed marker write or production ownership bookkeeping.
		var b [1]byte
		for {
			_, err := io.ReadFull(conn, b[:])
			if err == io.EOF {
				os.Exit(0)
			}
			if err != nil {
				os.Exit(8)
			}
			if _, err := conn.Write(b[:]); err != nil {
				os.Exit(8)
			}
		}
	}
	exe, _ := os.Executable()
	child := exec.Command(exe, "-test.run=^TestProcessTreeHelper$")
	child.Env = mergeEnv(os.Environ(), map[string]string{"AGENT_ENV_TREE_HELPER": "leaf"})
	if mode == "normal-inherit" {
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
	}
	if err := child.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
	_ = child.Process.Release()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker + ".ready"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			os.Exit(10)
		}
		time.Sleep(5 * time.Millisecond)
	}
	fmt.Fprint(os.Stdout, "child started")
	if mode == "timeout" || mode == "cancel" {
		time.Sleep(10 * time.Second)
	}
	os.Exit(0)
}

func TestRunnerReapsOrdinaryDescendants(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"timeout", "cancel", "normal", "normal-inherit"} {
		t.Run(mode, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "startup 日本語")
			listener := descendantObserver(t)
			timeout := 5 * time.Second
			if mode == "timeout" {
				timeout = 3 * time.Second
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			spec := Command{Name: exe, Args: []string{"-test.run=^TestProcessTreeHelper$"}, Timeout: timeout, Env: map[string]string{"AGENT_ENV_TREE_HELPER": mode, "AGENT_ENV_TREE_MARKER": marker, "AGENT_ENV_TREE_OBSERVER": listener.Addr().String(), "GORACE": "atexit_sleep_ms=0"}}
			if mode == "cancel" {
				spec.Stdout = cancelOnWrite{cancel}
			}
			result, err := (OSRunner{}).Run(ctx, spec)
			if mode == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timeout result %+v: %v", result, err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel result %+v: %v", result, err)
			}
			if (mode == "normal" || mode == "normal-inherit") && err != nil {
				t.Fatal(err)
			}
			if result.Stdout != "child started" {
				t.Fatalf("descendant never started: %+v", result)
			}
			conn := observeDescendant(t, listener)
			requireDescendantExit(t, conn)
		})
	}
}

type cancelOnWrite struct{ cancel context.CancelFunc }

func (w cancelOnWrite) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }

func descendantObserver(t *testing.T) *net.TCPListener {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close observer: %v", err)
		}
	})
	return listener
}

func observeDescendant(t *testing.T, listener *net.TCPListener) net.Conn {
	t.Helper()
	if err := listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal("descendant connection missing:", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close descendant observer: %v", err)
		}
	})
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil || b[0] != 'R' {
		t.Fatalf("descendant startup evidence %q: %v", b, err)
	}
	return conn
}

func requireDescendantExit(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if n, err := conn.Read(b[:]); n != 0 || err != io.EOF {
		t.Fatalf("descendant lifetime connection did not end: n=%d err=%v", n, err)
	}
}

func TestDescendantObserverPositiveControl(t *testing.T) {
	listener := descendantObserver(t)
	marker := filepath.Join(t.TempDir(), "observer 日本語")
	ctx, cancel := context.WithCancel(context.Background())
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessTreeHelper$")
	child.Env = append(os.Environ(), "AGENT_ENV_TREE_HELPER=leaf", "AGENT_ENV_TREE_MARKER="+marker, "AGENT_ENV_TREE_OBSERVER="+listener.Addr().String(), "GORACE=atexit_sleep_ms=0")
	if err := child.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		cancel()
		if !waited {
			_ = child.Wait()
		}
	})
	conn := observeDescendant(t, listener)
	if _, err := conn.Write([]byte{'L'}); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err := io.ReadFull(conn, b[:]); err != nil || b[0] != 'L' {
		t.Fatalf("live descendant did not answer: %q %v", b, err)
	}
	// Closing only our write side explicitly asks the positive-control leaf to
	// exit. Its read side must then report EOF and native Wait must succeed.
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	requireDescendantExit(t, conn)
	err := child.Wait()
	waited = true
	if err != nil {
		t.Fatal("positive-control leaf exit:", err)
	}
}
