package execx

import (
	"context"
	"errors"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func observeManaged(id ProcessIdentity) (ProcessObservation, error) { return observeDetachedJob(id) }

var isManagedProcessInJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")

func terminateManaged(ctx context.Context, id ProcessIdentity, grace time.Duration) error {
	observed, err := observeManaged(id)
	if err != nil {
		return err
	}
	if !observed.Alive {
		return nil
	}
	fields := strings.Split(id.StartID, "|")
	wide, err := windows.UTF16PtrFromString(fields[2])
	if err != nil {
		return err
	}
	// Keep this exact kernel Job handle throughout revalidation and termination.
	raw, _, callErr := openDetachedJob.Call(0x0004|0x0008, 0, uintptr(unsafe.Pointer(wide)))
	if raw == 0 {
		if errors.Is(callErr, windows.ERROR_FILE_NOT_FOUND) {
			current, err := observeManaged(id)
			if err == nil && !current.Alive {
				return nil
			}
			return errors.Join(ErrProcessTreeUnconfirmed, callErr, err)
		}
		return errors.Join(ErrProcessTreeUnconfirmed, callErr)
	}
	job := windows.Handle(raw)
	defer windows.CloseHandle(job)
	var accounting jobAccounting
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
		return err
	}
	if accounting.ActiveProcesses == 0 {
		return waitManagedGone(ctx, id, max(grace, 2*time.Second))
	}
	actual, root, err := detachedIdentity(id.PID)
	if err != nil || root && actual != fields[3] {
		identityErr := err
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
			return err
		}
		if accounting.ActiveProcesses == 0 {
			return waitManagedGone(ctx, id, max(grace, 2*time.Second))
		}
		return errors.Join(ErrProcessTreeUnconfirmed, identityErr, errors.New("managed job root identity mismatch"))
	}
	if root {
		process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(id.PID))
		if err != nil {
			return err
		}
		defer windows.CloseHandle(process)
		var member uint32
		ok, _, callErr := isManagedProcessInJob.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&member)))
		if ok == 0 {
			return callErr
		}
		if member == 0 {
			return errors.Join(ErrProcessTreeUnconfirmed, errors.New("managed root is not a member of recorded Job"))
		}
		// Recheck birth after acquiring the process handle and checking membership.
		actual, root, err = detachedIdentity(id.PID)
		if err != nil || !root || actual != fields[3] {
			return errors.Join(ErrProcessTreeUnconfirmed, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Detached native processes have no shared console to receive CTRL_BREAK.
	// Job termination is the explicit Windows contract; grace bounds observation.
	if err := windows.TerminateJobObject(job, 1); err != nil {
		return err
	}
	return waitManagedGone(ctx, id, max(grace, 2*time.Second))
}
