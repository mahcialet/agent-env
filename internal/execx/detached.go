package execx

import (
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/paths"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	if err := paths.ValidateExecutionDirectory(spec.Dir); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	if spec.Timeout != 0 || spec.Stdout != nil || spec.Stderr != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, errors.New("detached processes require file output and no command timeout"))
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(spec.Name))
		if ext == ".bat" || ext == ".cmd" {
			return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, errors.New("detached processes require a native executable"))
		}
	}
	cmd := exec.Command(spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	if err := ValidateNativeExecutable(cmd.Path, spec.Dir); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	out, err := os.OpenFile(stdoutPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	defer out.Close()
	errout, err := os.OpenFile(stderrPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	defer errout.Close()
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

// ErrProcessNotStarted proves the target was not spawned, permitting cleanup of prelaunch state.
var ErrProcessNotStarted = errors.New("detached target process was not started")

// ManagedProcess adds identity-gated lifecycle operations without changing the
// detached observation contract used by Android.
type ManagedProcess interface {
	DetachedProcess
	Observe(context.Context, ProcessIdentity) (ProcessObservation, error)
	Terminate(context.Context, ProcessIdentity, time.Duration) error
}

// ProcessObservation distinguishes the root's health from remaining tree effects.
type ProcessObservation struct {
	Alive     bool
	RootAlive bool
}

func (NativeDetached) Observe(ctx context.Context, id ProcessIdentity) (ProcessObservation, error) {
	if err := ctx.Err(); err != nil {
		return ProcessObservation{}, err
	}
	if id.PID <= 0 || id.StartID == "" {
		return ProcessObservation{}, errors.New("incomplete managed process identity")
	}
	for attempt := 0; ; attempt++ {
		observed, err := observeManaged(id)
		if !errors.Is(err, errProcessCensusUnstable) || attempt == 4 {
			return observed, err
		}
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return observed, ctx.Err()
		case <-timer.C:
		}
	}
}

var errProcessCensusUnstable = errors.New("process census changed during termination observation")

func (NativeDetached) Terminate(ctx context.Context, id ProcessIdentity, grace time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id.PID <= 0 || id.StartID == "" {
		return errors.New("incomplete managed process identity")
	}
	if grace < 0 {
		return errors.New("negative process termination grace")
	}
	return terminateManaged(ctx, id, grace)
}

// Observation failure never authorizes a signal. Retry only observation within
// this bound so transient native census races need not become permanent failures.
func waitManagedGone(ctx context.Context, id ProcessIdentity, duration time.Duration) error {
	deadline := time.Now().Add(duration)
	var last error
	for {
		observed, err := (NativeDetached{}).Observe(ctx, id)
		if err == nil && !observed.Alive {
			return nil
		}
		last = err
		if err := ctx.Err(); err != nil {
			return err
		}
		if !time.Now().Before(deadline) {
			return errors.Join(ErrProcessTreeUnconfirmed, last)
		}
		timer := time.NewTimer(min(20*time.Millisecond, time.Until(deadline)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
