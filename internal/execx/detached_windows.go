package execx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func startDetached(ctx context.Context, cmd *exec.Cmd) (ProcessIdentity, error) {
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	name := "Local\\agent-env-" + hex.EncodeToString(nonce[:])
	wide, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	job, err := windows.CreateJobObject(nil, wide)
	if err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	defer windows.CloseHandle(job)
	// No KILL_ON_JOB_CLOSE: a per-resource guardian retains a handle while
	// members live, preserving named observation after this CLI exits.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS | windows.CREATE_SUSPENDED}
	if err := ctx.Err(); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	if err := cmd.Start(); err != nil {
		return ProcessIdentity{}, errors.Join(ErrProcessNotStarted, err)
	}
	id := ProcessIdentity{PID: cmd.Process.Pid}
	abort := func(cause error) (ProcessIdentity, error) {
		killErr := cmd.Process.Kill()
		jobErr := windows.TerminateJobObject(job, 1)
		go func() { _ = cmd.Wait() }()
		return id, errors.Join(cause, killErr, jobErr, ErrProcessTreeUnconfirmed)
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(id.PID))
	if err != nil {
		return abort(err)
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		return abort(err)
	}
	birth, alive, err := detachedIdentity(id.PID)
	if err != nil || !alive {
		return abort(errors.Join(err, errors.New("suspended process identity unavailable")))
	}
	id.StartID = "job|" + strconv.FormatUint(uint64(session), 10) + "|" + name + "|" + birth
	output, ok := cmd.Stdout.(*os.File)
	if !ok {
		return abort(errors.New("detached output file unavailable"))
	}
	proof := filepath.Join(filepath.Dir(output.Name()), ".detached-"+hex.EncodeToString(nonce[:])+"-empty")
	proof, err = filepath.Abs(proof)
	if err != nil {
		return abort(err)
	}
	if err := startDetachedGuardian(job, name, proof); err != nil {
		return abort(err)
	}
	id.StartID += "|" + proof
	if err := ctx.Err(); err != nil {
		return abort(err)
	}
	if err := resumeInitialThread(uint32(id.PID)); err != nil {
		return abort(err)
	}
	return id, nil
}

var openDetachedJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("OpenJobObjectW")

func detachedTreeAlive(id ProcessIdentity) (bool, error) {
	observed, err := observeDetachedJob(id)
	return observed.Alive, err
}

func observeDetachedJob(id ProcessIdentity) (ProcessObservation, error) {
	fields := strings.Split(id.StartID, "|")
	if len(fields) != 5 || fields[0] != "job" || !strings.HasPrefix(fields[2], "Local\\agent-env-") || fields[3] == "" || !filepath.IsAbs(fields[4]) {
		return ProcessObservation{}, errors.New("detached job identity missing")
	}
	if filepath.Base(fields[4]) != ".detached-"+strings.TrimPrefix(fields[2], "Local\\agent-env-")+"-empty" {
		return ProcessObservation{}, errors.New("detached guardian proof identity mismatch")
	}
	wantedSession, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return ProcessObservation{}, errors.New("invalid detached job session identity")
	}
	var currentSession uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &currentSession); err != nil {
		return ProcessObservation{}, err
	}
	// Local job names resolve in the observing CLI's Terminal Services session.
	// A missing name in another session says nothing about the original job,
	// especially after its root exits while descendants remain alive.
	if uint64(currentSession) != wantedSession {
		return ProcessObservation{}, errors.New("detached job belongs to a different Windows session; observation is uncertain")
	}
	wide, err := windows.UTF16PtrFromString(fields[2])
	if err != nil {
		return ProcessObservation{}, err
	}
	// JOB_OBJECT_QUERY = 0x0004. x/sys does not expose OpenJobObjectW.
	raw, _, callErr := openDetachedJob.Call(0x0004, 0, uintptr(unsafe.Pointer(wide)))
	if raw == 0 {
		if errors.Is(callErr, windows.ERROR_FILE_NOT_FOUND) {
			// The exact guardian tombstone proves the recorded Job completed.
			// A later process using its historical PID is unrelated and must
			// neither prevent absence nor become a signal target.
			proof, proofErr := os.ReadFile(fields[4])
			if proofErr != nil || string(proof) != fields[2] {
				return ProcessObservation{}, errors.New("detached job missing without confirmed empty-process evidence")
			}
			return ProcessObservation{}, nil
		}
		return ProcessObservation{}, callErr
	}
	job := windows.Handle(raw)
	defer windows.CloseHandle(job)
	var accounting jobAccounting
	if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
		return ProcessObservation{}, err
	}
	if accounting.ActiveProcesses > 0 {
		actual, rootAlive, identityErr := detachedIdentity(id.PID)
		mismatch := rootAlive && actual != fields[3]
		if identityErr == nil && !mismatch {
			return ProcessObservation{Alive: true, RootAlive: rootAlive}, nil
		}
		// The last member may exit between accounting and PID observation.
		// Recheck this already-open exact Job before diagnosing a reused PID.
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
			return ProcessObservation{}, err
		}
		if accounting.ActiveProcesses > 0 {
			return ProcessObservation{Alive: true}, errors.Join(ErrProcessTreeUnconfirmed, identityErr, errors.New("detached job root identity reused or ambiguous"))
		}
	}
	// An empty Job precedes the guardian's last write. Keep the cleanup barrier
	// until it publishes its closed, synced proof; otherwise it could create a
	// file while a caller removes the evidence directory after observed absence.
	proof, proofErr := os.ReadFile(fields[4])
	// Publication may briefly hold DELETE access while renaming the proof.
	// A Windows sharing conflict is not corrupted evidence: preserve the exact
	// empty Job barrier until its immutable proof can actually be read.
	if errors.Is(proofErr, os.ErrNotExist) || errors.Is(proofErr, windows.ERROR_SHARING_VIOLATION) {
		return ProcessObservation{Alive: true}, nil
	}
	if proofErr != nil || string(proof) != fields[2] {
		return ProcessObservation{Alive: true}, errors.Join(ErrProcessTreeUnconfirmed, proofErr, errors.New("detached guardian completion evidence invalid"))
	}
	return ProcessObservation{}, nil
}

func detachedIdentity(pid int) (string, bool, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer windows.CloseHandle(handle)
	status, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		return "", false, err
	}
	if status == windows.WAIT_OBJECT_0 {
		return "", false, nil
	}
	if status != uint32(windows.WAIT_TIMEOUT) {
		return "", false, fmt.Errorf("unexpected process wait status %d", status)
	}
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return "", false, err
	}
	return fmt.Sprintf("%d:%d", created.HighDateTime, created.LowDateTime), true, nil
}
