package execx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Model PID reuse deterministically with another real live native process. The
// old birth token and exact Job proof stay unchanged; only its historical PID
// now resolves to the unrelated process, as Windows may do between CLI calls.
func TestManagedWindowsCompletedJobIgnoresReusedHistoricalPID(t *testing.T) {
	p := NativeDetached{}
	ctx := context.Background()
	launch := func() ProcessIdentity {
		t.Helper()
		dir := t.TempDir()
		id, err := p.Start(ctx, Command{Name: os.Args[0], Args: []string{"-test.run=^TestManagedHelper$"}, Env: map[string]string{"AGENT_ENV_MANAGED_HELPER": "leaf", "AGENT_ENV_MANAGED_DIR": dir, "GORACE": "atexit_sleep_ms=0"}}, filepath.Join(dir, "out"), filepath.Join(dir, "err"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := p.Terminate(ctx, id, time.Second); err != nil {
				t.Error(err)
			}
		})
		return id
	}
	old, other := launch(), launch()
	fields := strings.Split(old.StartID, "|")
	wide, err := windows.UTF16PtrFromString(fields[2])
	if err != nil {
		t.Fatal(err)
	}
	raw, _, callErr := openDetachedJob.Call(0x0004, 0, uintptr(unsafe.Pointer(wide)))
	if raw == 0 {
		t.Fatal(callErr)
	}
	retained := windows.Handle(raw)
	defer func() {
		if retained != 0 {
			_ = windows.CloseHandle(retained)
		}
	}()
	recycled := old
	recycled.PID = other.PID
	if _, err := p.Observe(ctx, recycled); err == nil {
		t.Fatal("active Job accepted mismatched root birth")
	}
	if err := p.Terminate(ctx, recycled, 0); err == nil {
		t.Fatal("active Job allowed termination with mismatched root birth")
	}
	assertOther := func() {
		t.Helper()
		o, err := p.Observe(ctx, other)
		if err != nil || !o.Alive || !o.RootAlive {
			t.Fatalf("unrelated native process affected: %+v %v", o, err)
		}
	}
	assertOther()
	if err := p.Terminate(ctx, old, time.Second); err != nil {
		t.Fatal(err)
	}
	assertAbsent := func() {
		t.Helper()
		o, err := p.Observe(ctx, recycled)
		if err != nil || o.Alive || o.RootAlive {
			t.Fatalf("completed Job blocked by recycled PID: %+v %v", o, err)
		}
		if alive, err := p.Alive(ctx, recycled); err != nil || alive {
			t.Fatalf("detached completed Job=%v %v", alive, err)
		}
		if err := p.Terminate(ctx, recycled, 0); err != nil {
			t.Fatal(err)
		}
		assertOther()
	}
	// First keep the exact named Job open: empty count plus its synced guardian
	// proof authorizes absence even while an unrelated process uses the old PID.
	assertAbsent()
	if err := windows.CloseHandle(retained); err != nil {
		t.Fatal(err)
	}
	retained = 0
	deadline := time.Now().Add(5 * time.Second)
	for {
		raw, _, err := openDetachedJob.Call(0x0004, 0, uintptr(unsafe.Pointer(wide)))
		if raw == 0 {
			if err != windows.ERROR_FILE_NOT_FOUND {
				t.Fatal(err)
			}
			break
		}
		_ = windows.CloseHandle(windows.Handle(raw))
		if time.Now().After(deadline) {
			t.Fatal("completed guardian did not close the exact Job")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Then exercise the later-CLI missing-Job path, where the exact guardian
	// tombstone is the durable authority and no historical PID is a signal target.
	assertAbsent()
	proof, err := os.ReadFile(fields[4])
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(fields[4], proof, 0600) })
	backup := fields[4] + ".saved"
	if err := os.Rename(fields[4], backup); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Observe(ctx, recycled); err == nil {
		t.Fatal("missing Job without completion proof accepted")
	}
	if err := p.Terminate(ctx, recycled, 0); err == nil {
		t.Fatal("missing completion proof allowed cleanup")
	}
	assertOther()
	if err := os.WriteFile(fields[4], []byte("different-job-proof"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Observe(ctx, recycled); err == nil {
		t.Fatal("wrong completion proof accepted")
	}
	if err := p.Terminate(ctx, recycled, 0); err == nil {
		t.Fatal("wrong completion proof allowed cleanup")
	}
	assertOther()
	if err := os.WriteFile(fields[4], proof, 0600); err != nil {
		t.Fatal(err)
	}
	assertAbsent()
}
