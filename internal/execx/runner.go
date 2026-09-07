// Package execx provides injectable native process execution.
package execx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

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
		err = cmd.Run()
		result.Stdout, result.Stderr = stdout.String(), stderr.String()
		if cmd.ProcessState != nil {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}
	result.FinishedAt = time.Now().UTC()
	if err != nil {
		return result, &ExitError{Command: spec, Result: result, Err: err}
	}
	return result, nil
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
