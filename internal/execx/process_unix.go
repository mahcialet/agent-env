//go:build unix

package execx

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

func runProcessTree(_ context.Context, cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	var retryFor time.Duration
	if runtime.GOOS == "darwin" {
		retryFor = time.Second
	}
	terminate := func() error {
		return terminateProcessGroup(cmd.Process.Pid, retryFor, syscall.Kill)
	}
	cmd.Cancel = terminate
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	if cleanupErr := terminate(); cleanupErr != nil {
		err = errors.Join(err, ErrProcessTreeUnconfirmed, fmt.Errorf("terminate command process group: %w", cleanupErr))
	}
	return err
}

func terminateProcessGroup(pid int, retryFor time.Duration, kill func(int, syscall.Signal) error) error {
	if pid <= 0 || pid == syscall.Getpgrp() {
		return errors.New("refusing to terminate runner process group")
	}
	deadline := time.Now().Add(retryFor)
	for {
		err := kill(-pid, syscall.SIGKILL)
		if err == nil || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		// Darwin's killpg skips zombies but returns EPERM while their group
		// still exists. Allow the OS to reap them, then require a successful
		// signal or ESRCH; persistent permission failures remain errors.
		remaining := time.Until(deadline)
		if !errors.Is(err, syscall.EPERM) || remaining <= 0 {
			return err
		}
		time.Sleep(min(10*time.Millisecond, remaining))
	}
}
