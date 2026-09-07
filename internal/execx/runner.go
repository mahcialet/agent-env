// Package execx provides injectable native process execution.
package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ErrProcessTreeUnconfirmed prevents callers from assuming cleanup is safe.
var ErrProcessTreeUnconfirmed = errors.New("command process tree termination unconfirmed")

// ErrOutputIncomplete identifies failed or truncated command evidence.
var ErrOutputIncomplete = errors.New("command output capture incomplete")

type Command struct {
	Name           string
	Args           []string
	Dir            string
	Env            map[string]string
	UnsetEnv       []string
	Timeout        time.Duration
	Stdout, Stderr io.Writer
}

type Result struct {
	Stdout, Stderr        string
	ExitCode              int
	StartedAt, FinishedAt time.Time
}

type Runner interface {
	Run(context.Context, Command) (Result, error)
}
type OSRunner struct{}

type ExitError struct {
	Command Command
	Result  Result
	Err     error
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s failed (exit %d): %v", e.Command.Name, e.Result.ExitCode, e.Err)
}
func (e *ExitError) Unwrap() error { return e.Err }

func (OSRunner) Run(ctx context.Context, spec Command) (Result, error) {
	result := Result{ExitCode: -1, StartedAt: time.Now().UTC()}
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	cmd, err := platformCommand(ctx, spec)
	if err == nil {
		cmd.Dir = spec.Dir
		base := cmd.Environ()
		for _, key := range spec.UnsetEnv {
			filtered := base[:0]
			for _, pair := range base {
				k, _, _ := strings.Cut(pair, "=")
				if k == key || runtime.GOOS == "windows" && strings.EqualFold(k, key) {
					continue
				}
				filtered = append(filtered, pair)
			}
			base = filtered
		}
		cmd.Env = mergeEnv(base, spec.Env)
		// A killed child can leave inherited pipes open. Bound the drain wait too.
		cmd.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if spec.Stdout != nil {
			cmd.Stdout = io.MultiWriter(&stdout, spec.Stdout)
		}
		if spec.Stderr != nil {
			cmd.Stderr = io.MultiWriter(&stderr, spec.Stderr)
		}
		err = runCaptured(ctx, cmd)
		result.Stdout, result.Stderr = stdout.String(), stderr.String()
		if cmd.ProcessState != nil {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
		if ctx.Err() != nil {
			err = errors.Join(ctx.Err(), err)
		}
	}
	result.FinishedAt = time.Now().UTC()
	if err != nil {
		return result, &ExitError{Command: spec, Result: result, Err: err}
	}
	return result, nil
}

// Own the stream pumps so Cmd.Wait observes the root's exit immediately even
// when a descendant inherits stdout/stderr. Tree cleanup precedes final draining.
func runCaptured(ctx context.Context, cmd *exec.Cmd) error {
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer outRead.Close()
	defer outWrite.Close()
	errRead, errWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer errRead.Close()
	defer errWrite.Close()
	outDst, errDst := cmd.Stdout, cmd.Stderr
	cmd.Stdout, cmd.Stderr = outWrite, errWrite
	done := make(chan error, 2)
	go func() { _, err := io.Copy(outDst, outRead); _ = outRead.Close(); done <- err }()
	go func() { _, err := io.Copy(errDst, errRead); _ = errRead.Close(); done <- err }()
	runErr := runProcessTree(ctx, cmd)
	_ = outWrite.Close()
	_ = errWrite.Close()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	var drainErrors []error
	for remaining := 2; remaining > 0; remaining-- {
		select {
		case err := <-done:
			drainErrors = append(drainErrors, err)
		case <-deadline.C:
			_ = outRead.Close()
			_ = errRead.Close()
			drainErrors = append(drainErrors, exec.ErrWaitDelay)
			// Closing the read descriptors releases pipe readers. A destination
			// writer itself must obey its own bounded-write contract.
			for ; remaining > 0; remaining-- {
				drainErrors = append(drainErrors, <-done)
			}
		}
	}
	if drainErr := errors.Join(drainErrors...); drainErr != nil {
		return errors.Join(runErr, ErrOutputIncomplete, drainErr)
	}
	return runErr
}

func mergeEnv(base []string, overrides map[string]string) []string {
	values := make(map[string]string)
	keyOf := func(k string) string {
		if runtime.GOOS == "windows" {
			return strings.ToUpper(k)
		}
		return k
	}
	for _, pair := range base {
		if k, _, ok := strings.Cut(pair, "="); ok {
			values[keyOf(k)] = pair
		}
	}
	for k, v := range overrides {
		values[keyOf(k)] = k + "=" + v
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		result = append(result, values[k])
	}
	return result
}

// LookPath is exposed for prerequisite diagnostics, not command-string execution.
func LookPath(name string) (string, error) { return exec.LookPath(name) }
