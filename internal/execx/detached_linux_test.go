package execx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDetachedProcStatReadAfterExitReportsVanishedEntry(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetachedProcessHelper$")
	cmd.Env = mergeEnv(os.Environ(), map[string]string{"AGENT_ENV_DETACHED_HELPER": "leaf", "AGENT_ENV_DETACHED_DIR": dir, "GORACE": "atexit_sleep_ms=0"})
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	// Open while the task is alive, then read only after Wait has reaped it.
	// This deterministically reaches the kernel boundary behind the CI race,
	// rather than depending on a lucky exit during os.ReadFile.
	stat, err := os.Open(fmt.Sprintf("/proc/%d/stat", cmd.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer stat.Close()
	if err := os.WriteFile(filepath.Join(dir, "stop"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(stat)
	if !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("expected task disappearance at proc read boundary, got %v", err)
	}
	if !detachedProcEntryGone(err) {
		t.Fatalf("proc read disappearance treated as an inspection failure: %v", err)
	}
	if _, alive, err := detachedIdentity(cmd.Process.Pid); err != nil || alive {
		t.Fatalf("reaped leader identity: alive=%v err=%v", alive, err)
	}
}

func TestDetachedProcDisappearanceDoesNotHideInspectionFailures(t *testing.T) {
	for _, cause := range []error{syscall.EACCES, syscall.EPERM, syscall.EIO, io.ErrUnexpectedEOF} {
		err := &os.PathError{Op: "read", Path: "/proc/123/stat", Err: cause}
		if detachedProcEntryGone(err) {
			t.Fatalf("ambiguous inspection failure treated as absence: %v", err)
		}
	}
}
