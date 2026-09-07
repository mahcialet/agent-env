package execx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const detachedGuardianArg = "--agent-env-internal-detached-job-guardian"

// This private self-exec entry runs before CLI or test dispatch. It receives
// only explicitly inherited kernel handles and literal argv (never a shell).
// One helper exists per detached resource, and exits when its Job is empty.
func init() {
	if len(os.Args) != 6 || os.Args[1] != detachedGuardianArg {
		return
	}
	job, e1 := strconv.ParseUint(os.Args[2], 10, 64)
	ready, e2 := strconv.ParseUint(os.Args[3], 10, 64)
	if e1 != nil || e2 != nil || job == 0 || ready == 0 {
		os.Exit(125)
	}
	if err := runDetachedGuardian(windows.Handle(job), windows.Handle(ready), os.Args[4], os.Args[5]); err != nil {
		os.Exit(125)
	}
	os.Exit(0)
}

func startDetachedGuardian(job windows.Handle, name, proof string) error {
	if err := windows.SetHandleInformation(job, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
		return err
	}
	defer windows.SetHandleInformation(job, windows.HANDLE_FLAG_INHERIT, 0)
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), InheritHandle: 1}
	ready, err := windows.CreateEvent(&sa, 1, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(ready)
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, detachedGuardianArg, strconv.FormatUint(uint64(job), 10), strconv.FormatUint(uint64(ready), 10), name, proof)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS, HideWindow: true, AdditionalInheritedHandles: []syscall.Handle{syscall.Handle(job), syscall.Handle(ready)}}
	if err = cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	status, err := windows.WaitForSingleObject(ready, 5000)
	if err != nil || status != windows.WAIT_OBJECT_0 {
		_ = cmd.Process.Kill()
		return fmt.Errorf("detached job guardian did not become ready: wait=%d: %w", status, errors.Join(err, errors.New("guardian startup unconfirmed")))
	}
	return nil
}

func runDetachedGuardian(job, ready windows.Handle, name, proof string) error {
	defer windows.CloseHandle(job)
	defer windows.CloseHandle(ready)
	signaled := false
	for {
		var a jobAccounting
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil); err != nil {
			return err
		}
		if !signaled {
			if err := windows.SetEvent(ready); err != nil {
				return err
			}
			signaled = true
		}
		if a.ActiveProcesses == 0 {
			f, err := os.OpenFile(proof, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			_, writeErr := f.WriteString(name)
			syncErr := f.Sync()
			closeErr := f.Close()
			return errors.Join(writeErr, syncErr, closeErr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
