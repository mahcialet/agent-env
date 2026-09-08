package execx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestManagedTermination(t *testing.T) {
	for _, tree := range []bool{false, true} {
		t.Run(fmt.Sprint(tree), func(t *testing.T) {
			dir := t.TempDir()
			mode := "stubborn"
			if tree {
				mode = "root"
			}
			p := NativeDetached{}
			id, err := p.Start(context.Background(), Command{Name: os.Args[0], Args: []string{"-test.run=^TestManagedHelper$"}, Env: map[string]string{"AGENT_ENV_MANAGED_HELPER": mode, "AGENT_ENV_MANAGED_DIR": dir, "GORACE": "atexit_sleep_ms=0"}}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = os.WriteFile(filepath.Join(dir, "stop"), nil, 0600)
				_ = p.Terminate(context.Background(), id, time.Second)
			})
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("helper not ready")
				}
				time.Sleep(10 * time.Millisecond)
			}
			observed, err := p.Observe(context.Background(), id)
			if err != nil || !observed.Alive || !observed.RootAlive {
				t.Fatalf("observe=%+v err=%v", observed, err)
			}
			wrong := id
			wrong.StartID += "-mismatch"
			if err := p.Terminate(context.Background(), wrong, 0); err == nil {
				t.Fatal("mismatched identity accepted")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := p.Terminate(ctx, id, 0); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel=%v", err)
			}
			observed, err = p.Observe(context.Background(), id)
			if err != nil || !observed.RootAlive {
				t.Fatalf("negative calls affected live root: %+v %v", observed, err)
			}
			if err := p.Terminate(context.Background(), id, time.Second); err != nil {
				t.Fatal(err)
			}
			observed, err = p.Observe(context.Background(), id)
			if err != nil || observed.Alive || observed.RootAlive {
				t.Fatalf("terminated=%+v %v", observed, err)
			}
			if err := p.Terminate(context.Background(), id, 0); err != nil {
				t.Fatalf("repeated=%v", err)
			}
		})
	}
}

func TestManagedRootGoneDescendant(t *testing.T) {
	dir := t.TempDir()
	p := NativeDetached{}
	id, err := p.Start(context.Background(), Command{Name: os.Args[0], Args: []string{"-test.run=^TestManagedHelper$"}, Env: map[string]string{"AGENT_ENV_MANAGED_HELPER": "root", "AGENT_ENV_MANAGED_DIR": dir, "GORACE": "atexit_sleep_ms=0"}}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(dir, "stop"), nil, 0600)
		_ = waitManagedGone(context.Background(), id, 5*time.Second)
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = os.WriteFile(filepath.Join(dir, "root-exit"), nil, 0600)
	for {
		_, alive, err := detachedIdentity(id.PID)
		if err != nil {
			t.Fatal(err)
		}
		if !alive {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("root did not exit")
		}
		time.Sleep(10 * time.Millisecond)
	}
	observed, err := p.Observe(context.Background(), id)
	if !observed.Alive || observed.RootAlive {
		t.Fatalf("observation=%+v %v", observed, err)
	}
	if runtime.GOOS == "windows" {
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Terminate(context.Background(), id, time.Second); err != nil {
			t.Fatal(err)
		}
	} else {
		if !errors.Is(err, ErrProcessTreeUnconfirmed) {
			t.Fatalf("ambiguous tree accepted: %v", err)
		}
		if err := p.Terminate(context.Background(), id, 0); !errors.Is(err, ErrProcessTreeUnconfirmed) {
			t.Fatalf("ambiguous termination=%v", err)
		}
		if observed, err := p.Observe(context.Background(), id); !observed.Alive || !errors.Is(err, ErrProcessTreeUnconfirmed) {
			t.Fatalf("refused termination affected descendant: %+v %v", observed, err)
		}
	}
}

func TestManagedHelper(t *testing.T) {
	mode := os.Getenv("AGENT_ENV_MANAGED_HELPER")
	if mode == "" {
		return
	}
	dir := os.Getenv("AGENT_ENV_MANAGED_DIR")
	if mode == "stubborn" {
		signal.Ignore(syscall.SIGTERM)
	}
	if mode == "root" {
		child := exec.Command(os.Args[0], "-test.run=^TestManagedHelper$")
		child.Env = mergeEnv(os.Environ(), map[string]string{"AGENT_ENV_MANAGED_HELPER": "leaf"})
		if err := child.Start(); err != nil {
			os.Exit(5)
		}
		go func() { _ = child.Wait() }()
	} else {
		_ = os.WriteFile(filepath.Join(dir, "ready"), nil, 0600)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "stop")); err == nil {
			os.Exit(0)
		}
		if mode == "root" {
			if _, err := os.Stat(filepath.Join(dir, "root-exit")); err == nil {
				os.Exit(0)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	os.Exit(4)
}

func TestManagedInvalidIdentityAndGrace(t *testing.T) {
	p := NativeDetached{}
	for _, id := range []ProcessIdentity{{}, {PID: os.Getpid()}, {PID: -1, StartID: "group|birth"}} {
		if _, err := p.Observe(context.Background(), id); err == nil {
			t.Fatalf("invalid identity observed: %+v", id)
		}
		if err := p.Terminate(context.Background(), id, 0); err == nil {
			t.Fatalf("invalid identity terminated: %+v", id)
		}
	}
	if err := p.Terminate(context.Background(), ProcessIdentity{PID: os.Getpid(), StartID: "invalid"}, -time.Second); err == nil {
		t.Fatal("negative grace accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Observe(ctx, ProcessIdentity{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled observation: %v", err)
	}
}

func TestManagedNoSpawnFailureIsExplicit(t *testing.T) {
	dir := t.TempDir()
	p := NativeDetached{}
	id, err := p.Start(context.Background(), Command{Name: os.Args[0], Timeout: time.Second}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
	if !errors.Is(err, ErrProcessNotStarted) || id.PID != 0 {
		t.Fatalf("prelaunch argument failure=%+v %v", id, err)
	}
	{
		id, err = p.Start(context.Background(), Command{Name: filepath.Join(dir, "does-not-exist")}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
		if !errors.Is(err, ErrProcessNotStarted) || id.PID != 0 {
			t.Fatalf("native exec failure=%+v %v", id, err)
		}
	}
}
