package execx

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func detachedIdentity(pid int) (string, bool, error) {
	// The slice API distinguishes a disappeared process (zero bytes) from an
	// inaccessible one; SysctlKinfoProc reports EIO for that valid empty result.
	infos, err := unix.SysctlKinfoProcSlice("kern.proc.pid", pid)
	if errors.Is(err, unix.ESRCH) || errors.Is(err, unix.ENOENT) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if len(infos) == 0 {
		return "", false, nil
	}
	if len(infos) != 1 || infos[0].Proc.P_pid != int32(pid) {
		return "", false, errors.New("ambiguous process identity")
	}
	info := infos[0]
	// BSD SZOMB (5) is no longer an executing process.
	if info.Proc.P_stat == 5 {
		return "", false, nil
	}
	start := info.Proc.P_starttime
	if start.Sec == 0 {
		return "", false, errors.New("process creation identity unavailable")
	}
	return fmt.Sprintf("%d:%d", start.Sec, start.Usec), true, nil
}

func detachedGroupAlive(pgid int) (bool, error) {
	infos, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pgid)
	if err != nil {
		return false, err
	}
	for _, info := range infos {
		if info.Proc.P_stat != 5 {
			return true, nil
		}
	}
	return false, nil
}
