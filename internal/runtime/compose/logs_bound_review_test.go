package compose

import (
	"bytes"
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/execx"
	"os"
	"testing"
)

func TestNoisyComposeLogHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_NOISY_LOG_HELPER") != "1" {
		return
	}
	block := bytes.Repeat([]byte{'x'}, 64<<10)
	for i := 0; i < 40; i++ {
		if _, err := os.Stdout.Write(block); err != nil {
			os.Exit(0)
		}
	}
	os.Exit(0)
}

type noisyComposeLogRunner struct{ command execx.Command }

func (r *noisyComposeLogRunner) Run(ctx context.Context, command execx.Command) (execx.Result, error) {
	r.command = command
	executable, err := os.Executable()
	if err != nil {
		return execx.Result{}, err
	}
	command.Name = executable
	command.Args = []string{"-test.run=^TestNoisyComposeLogHelper$"}
	command.Env = map[string]string{"AGENT_ENV_NOISY_LOG_HELPER": "1"}
	return (execx.OSRunner{}).Run(ctx, command)
}
func TestComposeLogsBoundCaptureBeforeReturningLargeOutput(t *testing.T) {
	runtime := runtimeFixture(t)
	if err := os.MkdirAll(runtime.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	runner := &noisyComposeLogRunner{}
	logs, err := (Client{Runner: runner}).LogsBounded(context.Background(), runtime)
	if runner.command.CaptureLimit <= 0 || runner.command.CaptureLimit > 1<<20 {
		t.Fatalf("unbounded log capture: %d", runner.command.CaptureLimit)
	}
	if !errors.Is(err, execx.ErrOutputIncomplete) {
		t.Fatalf("truncated logs reported success: %v", err)
	}
	if len(logs) > 2*(1<<20) {
		t.Fatalf("too much output retained: %d", len(logs))
	}
}

func TestComposeCleanupLogCaptureRetainsExistingCompleteEvidencePath(t *testing.T) {
	runtime := runtimeFixture(t)
	if err := os.MkdirAll(runtime.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	runner := &noisyComposeLogRunner{}
	logs, err := (Client{Runner: runner}).Logs(context.Background(), runtime)
	if err != nil || runner.command.CaptureLimit != 0 || len(logs) != 40*(64<<10) {
		t.Fatalf("cleanup log evidence changed: bytes=%d limit=%d error=%v", len(logs), runner.command.CaptureLimit, err)
	}
}
