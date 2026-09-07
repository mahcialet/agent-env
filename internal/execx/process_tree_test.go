package execx

import (
	"context"
	"errors"
	"fmt"
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
		if err := os.WriteFile(marker+".ready", nil, 0600); err != nil {
			os.Exit(8)
		}
		time.Sleep(800 * time.Millisecond)
		_ = os.WriteFile(marker, []byte("descendant survived"), 0600)
		os.Exit(0)
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
			marker := filepath.Join(t.TempDir(), "late write 日本語")
			timeout := 5 * time.Second
			if mode == "timeout" {
				timeout = 300 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			spec := Command{Name: exe, Args: []string{"-test.run=^TestProcessTreeHelper$"}, Timeout: timeout, Env: map[string]string{"AGENT_ENV_TREE_HELPER": mode, "AGENT_ENV_TREE_MARKER": marker, "GORACE": "atexit_sleep_ms=0"}}
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
			deadline := time.NewTimer(time.Second)
			defer deadline.Stop()
			<-deadline.C
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("descendant wrote after runner returned: %v", err)
			}
		})
	}
}

type cancelOnWrite struct{ cancel context.CancelFunc }

func (w cancelOnWrite) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }
