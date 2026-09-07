//go:build unix

package execx

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

func runProcessTree(_ context.Context, cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	terminate := func() error {
		pid := cmd.Process.Pid
		if pid <= 0 || pid == syscall.Getpgrp() {
			return errors.New("refusing to terminate runner process group")
		}
		err := syscall.Kill(-pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	cmd.Cancel = terminate
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	if cleanupErr := terminate(); cleanupErr != nil {
		err = errors.Join(err, fmt.Errorf("terminate command process group: %w", cleanupErr))
	}
	return err
}
