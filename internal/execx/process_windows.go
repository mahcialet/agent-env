package execx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A suspended initial thread cannot spawn children before job assignment.
// Assignment/resume failures kill that root before returning, never falling back
// to uncontained execution. Descendants inherit this non-breakaway job.
func runProcessTree(ctx context.Context, cmd *exec.Cmd) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("create command job: %w", err)
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return fmt.Errorf("configure command job: %w", err)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	cmd.Cancel = func() error {
		jobErr := windows.TerminateJobObject(job, 1)
		rootErr := cmd.Process.Kill() // Also covers cancellation before assignment.
		if errors.Is(rootErr, os.ErrProcessDone) {
			rootErr = nil
		}
		return errors.Join(jobErr, rootErr)
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	abort := func(cause error) error {
		_ = cmd.Process.Kill()
		_ = windows.TerminateJobObject(job, 1)
		_ = cmd.Wait()
		return cause
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return abort(fmt.Errorf("open suspended command: %w", err))
	}
	defer windows.CloseHandle(process)
	if err = windows.AssignProcessToJobObject(job, process); err != nil {
		return abort(fmt.Errorf("assign suspended command to job: %w", err))
	}
	if err = ctx.Err(); err != nil {
		return abort(err)
	}
	if err = resumeInitialThread(uint32(cmd.Process.Pid)); err != nil {
		return abort(fmt.Errorf("resume contained command: %w", err))
	}
	waitErr := cmd.Wait()
	if err = windows.TerminateJobObject(job, 1); err != nil {
		return errors.Join(waitErr, fmt.Errorf("terminate command job: %w", err))
	}
	return errors.Join(waitErr, waitForEmptyJob(job))
}

func resumeInitialThread(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		_, err = windows.ResumeThread(thread)
		_ = windows.CloseHandle(thread)
		return err
	}
	if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return errors.New("suspended command initial thread not found")
	}
	return err
}

// Layout is the documented JOBOBJECT_BASIC_ACCOUNTING_INFORMATION ABI.
type jobAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

func waitForEmptyJob(job windows.Handle) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		var accounting jobAccounting
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
			return fmt.Errorf("verify terminated command job: %w", err)
		}
		if accounting.ActiveProcesses == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("command job still has active descendants after termination deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
