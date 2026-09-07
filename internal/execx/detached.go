package execx

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ProcessIdentity binds a PID to a persistent tree observation scope. StartID
// encodes its native birth identity and process group or named Windows job;
// observing only the root PID is not sufficient evidence for safe cleanup.
type ProcessIdentity struct {
	PID     int    `json:"pid"`
	StartID string `json:"start_id"`
}

// DetachedProcess manages observation of persistent native processes. Unlike
// Runner, its processes outlive the launching CLI and are not context-owned.
// Callers must durably record launch intent before Start and retain any nonzero
// identity returned with an error for conservative recovery.
type DetachedProcess interface {
	Start(context.Context, Command, string, string) (ProcessIdentity, error)
	Alive(context.Context, ProcessIdentity) (bool, error)
}

type NativeDetached struct{}

// Start connects output directly to private files, never CLI-owned pipes. The
// context gates launch only; cancellation after launch does not stop the process.
// Command.Timeout and writer destinations are intentionally unsupported.
func (NativeDetached) Start(ctx context.Context, spec Command, stdoutPath, stderrPath string) (ProcessIdentity, error) {
	if err := ctx.Err(); err != nil {
		return ProcessIdentity{}, err
	}
	if spec.Timeout != 0 || spec.Stdout != nil || spec.Stderr != nil {
		return ProcessIdentity{}, errors.New("detached processes require file output and no command timeout")
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(spec.Name))
		if ext == ".bat" || ext == ".cmd" {
			return ProcessIdentity{}, errors.New("detached processes require a native executable")
		}
	}
	out, err := os.OpenFile(stdoutPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return ProcessIdentity{}, err
	}
	defer out.Close()
	errout, err := os.OpenFile(stderrPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return ProcessIdentity{}, err
	}
	defer errout.Close()
	cmd := exec.Command(spec.Name, spec.Args...)
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
	cmd.Stdout, cmd.Stderr = out, errout
	identity, err := startDetached(ctx, cmd)
	if err != nil {
		return identity, err
	}
	// Reap during a long-lived CLI; OS adoption handles a CLI that exits first.
	go func() { _ = cmd.Wait() }()
	return identity, nil
}

func (NativeDetached) Alive(ctx context.Context, identity ProcessIdentity) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if identity.PID <= 0 || identity.StartID == "" {
		return false, errors.New("incomplete detached process identity")
	}
	return detachedTreeAlive(identity)
}
