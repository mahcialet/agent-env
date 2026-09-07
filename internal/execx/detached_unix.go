//go:build linux || darwin

package execx

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func startDetached(ctx context.Context, cmd *exec.Cmd) (ProcessIdentity, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := ctx.Err(); err != nil {
		return ProcessIdentity{}, err
	}
	if err := cmd.Start(); err != nil {
		return ProcessIdentity{}, err
	}
	id := ProcessIdentity{PID: cmd.Process.Pid}
	start, alive, err := detachedIdentity(id.PID)
	if err != nil || !alive {
		if err == nil {
			err = errors.New("detached process exited before identity capture")
		}
		killErr := syscall.Kill(-id.PID, syscall.SIGKILL)
		if errors.Is(killErr, syscall.ESRCH) || errors.Is(killErr, os.ErrProcessDone) {
			killErr = nil
		}
		go func() { _ = cmd.Wait() }()
		return id, errors.Join(err, killErr, ErrProcessTreeUnconfirmed)
	}
	id.StartID = "group|" + start
	return id, nil
}

func detachedTreeAlive(id ProcessIdentity) (bool, error) {
	want, ok := strings.CutPrefix(id.StartID, "group|")
	if !ok || want == "" {
		return false, errors.New("detached process group identity missing")
	}
	actual, rootAlive, err := detachedIdentity(id.PID)
	if err != nil {
		return false, err
	}
	if rootAlive && actual == want {
		return true, nil
	}
	groupAlive, err := detachedGroupAlive(id.PID)
	if err != nil {
		return false, err
	}
	if rootAlive && actual != want {
		if groupAlive {
			return false, errors.New("detached process group identity reused or ambiguous")
		}
		return false, nil
	}
	// Occupancy cannot distinguish descendants from a recycled PGID whose new
	// leader also exited. Retain the barrier without authorizing unsafe signals.
	if groupAlive {
		return true, errors.Join(ErrProcessTreeUnconfirmed, errors.New("detached leader absent; live group ownership is uncertain"))
	}
	return false, nil
}
