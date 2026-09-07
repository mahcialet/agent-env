package execx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDetachedIdentityRejectsReusedPID(t *testing.T) {
	start, alive, err := detachedIdentity(os.Getpid())
	if err != nil || !alive || start == "" {
		t.Fatalf("self identity: %q %v %v", start, alive, err)
	}
	process := NativeDetached{}
	if alive, err := process.Alive(context.Background(), ProcessIdentity{PID: os.Getpid(), StartID: start}); err == nil || alive {
		t.Fatalf("bare PID birth is not tree identity: alive=%v err=%v", alive, err)
	}
	if _, err := process.Alive(context.Background(), ProcessIdentity{PID: os.Getpid()}); err == nil {
		t.Fatal("incomplete identity accepted")
	}
}

func TestDetachedCanceledStartDoesNotLaunch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	id, err := (NativeDetached{}).Start(ctx, Command{Name: os.Args[0]}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
	if !errors.Is(err, context.Canceled) || id.PID != 0 {
		t.Fatalf("id=%+v err=%v", id, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("canceled start had filesystem effects: %v %v", entries, err)
	}
}

func TestDetachedSurvivesLaunchingCLI(t *testing.T) {
	for _, tree := range []bool{false, true} {
		t.Run(fmt.Sprintf("root_exits_%v", tree), func(t *testing.T) { testDetachedSurvivesLaunchingCLI(t, tree) })
	}
}

func testDetachedSurvivesLaunchingCLI(t *testing.T, tree bool) {
	dir := filepath.Join(t.TempDir(), "emulator space 日本語")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	launcher := exec.Command(os.Args[0], "-test.run=^TestDetachedProcessHelper$")
	launcher.Env = mergeEnv(os.Environ(), map[string]string{"AGENT_ENV_DETACHED_HELPER": "launcher", "AGENT_ENV_DETACHED_DIR": dir, "GORACE": "atexit_sleep_ms=0"})
	if tree {
		launcher.Env = append(launcher.Env, "AGENT_ENV_DETACHED_TREE=1")
	}
	output, err := launcher.Output()
	if err != nil {
		t.Fatalf("launcher: %v output=%s", err, output)
	}
	var identity ProcessIdentity
	if err := json.Unmarshal(output, &identity); err != nil {
		t.Fatalf("launcher identity %q: %v", output, err)
	}
	process := NativeDetached{}
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(dir, "stop"), nil, 0600)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			alive, err := process.Alive(context.Background(), identity)
			if err == nil && !alive {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Error("detached helper did not stop")
		if alive, err := process.Alive(context.Background(), identity); err == nil && alive {
			if p, err := os.FindProcess(identity.PID); err == nil {
				_ = p.Kill()
			}
		}
	})
	if alive, err := process.Alive(context.Background(), identity); err != nil || !alive {
		t.Fatalf("child did not survive CLI: %v %v", alive, err)
	}
	wrong := identity
	wrong.StartID += "-reused"
	if alive, err := process.Alive(context.Background(), wrong); alive || err == nil {
		t.Fatalf("reused group/job identity accepted: %v %v", alive, err)
	}
	if tree {
		if err := os.WriteFile(filepath.Join(dir, "root-exit"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			_, alive, err := detachedIdentity(identity.PID)
			if err != nil {
				t.Fatal(err)
			}
			if !alive {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("tree root did not exit")
			}
			time.Sleep(10 * time.Millisecond)
		}
		alive, err := process.Alive(context.Background(), identity)
		if !alive || (runtime.GOOS == "windows" && err != nil) || (runtime.GOOS != "windows" && !errors.Is(err, ErrProcessTreeUnconfirmed)) {
			t.Fatalf("descendant ownership after root exit: %v %v", alive, err)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		out, _ := os.ReadFile(filepath.Join(dir, "stdout.log"))
		errout, _ := os.ReadFile(filepath.Join(dir, "stderr.log"))
		if strings.Contains(string(out), "argument space 日本語|override|unset") && strings.Contains(string(errout), "persistent diagnostic") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("persistent output/argv/env mismatch: stdout=%q stderr=%q", out, errout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "stop"), nil, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDetachedProcessHelper(t *testing.T) {
	mode := os.Getenv("AGENT_ENV_DETACHED_HELPER")
	if mode == "" {
		return
	}
	dir := os.Getenv("AGENT_ENV_DETACHED_DIR")
	if mode == "launcher" {
		_ = os.Setenv("AGENT_ENV_DETACHED_UNSET", "must be removed")
		ctx, cancel := context.WithCancel(context.Background())
		childMode := "leaf"
		if os.Getenv("AGENT_ENV_DETACHED_TREE") == "1" {
			childMode = "root"
		}
		identity, err := (NativeDetached{}).Start(ctx, Command{
			Name: os.Args[0], Args: []string{"-test.run=^TestDetachedProcessHelper$", "--", "argument space 日本語"}, Dir: dir,
			Env:      map[string]string{"AGENT_ENV_DETACHED_HELPER": childMode, "AGENT_ENV_DETACHED_OVERRIDE": "override"},
			UnsetEnv: []string{"AGENT_ENV_DETACHED_UNSET"},
		}, filepath.Join(dir, "stdout.log"), filepath.Join(dir, "stderr.log"))
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode(identity)
		os.Exit(0)
	}
	if mode == "root" {
		child := exec.Command(os.Args[0], "-test.run=^TestDetachedProcessHelper$", "--", "argument space 日本語")
		child.Env = mergeEnv(os.Environ(), map[string]string{"AGENT_ENV_DETACHED_HELPER": "leaf"})
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(5)
		}
		_ = child.Process.Release()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(filepath.Join(dir, "root-exit")); err == nil {
				os.Exit(0)
			}
			time.Sleep(10 * time.Millisecond)
		}
		os.Exit(6)
	}
	if mode != "leaf" {
		os.Exit(3)
	}
	unset := "not unset"
	if _, exists := os.LookupEnv("AGENT_ENV_DETACHED_UNSET"); !exists {
		unset = "unset"
	}
	fmt.Fprintf(os.Stdout, "%s|%s|%s\n", os.Args[len(os.Args)-1], os.Getenv("AGENT_ENV_DETACHED_OVERRIDE"), unset)
	fmt.Fprintln(os.Stderr, "persistent diagnostic")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "stop")); err == nil {
			os.Exit(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
	os.Exit(4)
}
