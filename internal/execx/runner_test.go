package execx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

// The test binary doubles as a native argv/env/cwd probe on every supported OS.
func TestProcessHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_EXECX_HELPER") != "1" {
		return
	}
	switch os.Getenv("AGENT_ENV_EXECX_MODE") {
	case "sleep":
		time.Sleep(10 * time.Second)
	case "exit":
		fmt.Fprint(os.Stderr, "diagnostic")
		os.Exit(7)
	default:
		cwd, _ := os.Getwd()
		args := os.Args
		for i, arg := range args {
			if arg == "--" {
				args = args[i+1:]
				break
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Args       []string
			Cwd, Value string
		}{args, cwd, os.Getenv("AGENT_ENV_EXECX_VALUE")})
		fmt.Fprint(os.Stderr, "stderr stream")
	}
	os.Exit(0)
}

func TestNativeRoundTrip(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	args := []string{"", "spaces in argument", `quote "inside"`, "日本語", `C:\directory with space\`, `a&b|c<d>e`, `%PATH%`, "!literal!"}
	var out, stderr bytes.Buffer
	r, err := (OSRunner{}).Run(context.Background(), Command{Name: exe, Args: append([]string{"-test.run=^TestProcessHelper$", "--"}, args...), Dir: dir, Env: map[string]string{"AGENT_ENV_EXECX_HELPER": "1", "AGENT_ENV_EXECX_VALUE": "value with 日本語"}, Timeout: 10 * time.Second, Stdout: &out, Stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Args       []string
		Cwd, Value string
	}
	if err = json.Unmarshal([]byte(r.Stdout), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Args, args) || got.Value != "value with 日本語" {
		t.Fatalf("round trip: %+v", got)
	}
	a, _ := os.Stat(got.Cwd)
	b, _ := os.Stat(dir)
	if a == nil || b == nil || !os.SameFile(a, b) {
		t.Fatalf("cwd %q != %q", got.Cwd, dir)
	}
	if out.String() != r.Stdout || stderr.String() != r.Stderr || r.Stderr != "stderr stream" || r.ExitCode != 0 || r.FinishedAt.Before(r.StartedAt) {
		t.Fatalf("bad result: %+v", r)
	}
}

func TestExitAndTimeout(t *testing.T) {
	exe, _ := os.Executable()
	for _, mode := range []string{"exit", "sleep"} {
		t.Run(mode, func(t *testing.T) {
			r, err := (OSRunner{}).Run(context.Background(), Command{Name: exe, Args: []string{"-test.run=^TestProcessHelper$"}, Env: map[string]string{"AGENT_ENV_EXECX_HELPER": "1", "AGENT_ENV_EXECX_MODE": mode}, Timeout: 100 * time.Millisecond})
			var ee *ExitError
			if !errors.As(err, &ee) {
				t.Fatalf("expected typed error, got %v", err)
			}
			if mode == "exit" && (r.ExitCode != 7 || r.Stderr != "diagnostic") {
				t.Fatalf("exit result: %+v", r)
			}
			if mode == "sleep" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timeout: %v", err)
			}
		})
	}
}

func TestMissingCommand(t *testing.T) {
	r, err := (OSRunner{}).Run(context.Background(), Command{Name: "agent-env-this-program-does-not-exist"})
	if err == nil || r.ExitCode != -1 {
		t.Fatalf("%+v %v", r, err)
	}
}

type failingOutputWriter struct{}

func (failingOutputWriter) Write([]byte) (int, error) {
	return 0, errors.New("injected output write failure")
}

func TestOutputWriteFailureIsIncompleteEvidence(t *testing.T) {
	_, err := (OSRunner{}).Run(context.Background(), Command{Name: os.Args[0], Args: []string{"-test.run=^TestProcessHelper$"}, Env: map[string]string{"AGENT_ENV_EXECX_HELPER": "1", "GORACE": "atexit_sleep_ms=0"}, Stdout: failingOutputWriter{}, Timeout: 5 * time.Second})
	if !errors.Is(err, ErrOutputIncomplete) {
		t.Fatalf("output failure marker missing: %v", err)
	}
}
