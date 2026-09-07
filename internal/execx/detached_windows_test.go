package execx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestDetachedObservationRejectsDifferentWindowsSession(t *testing.T) {
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	// Neither this root nor job exists locally. Returning false,nil would confuse
	// an invisible job in another session with a confirmed empty process tree.
	id := ProcessIdentity{PID: 2147483647, StartID: fmt.Sprintf("job|%d|Local\\agent-env-nonexistent|1:1|%s", session^1, filepath.Join(t.TempDir(), ".detached-nonexistent-empty"))}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err == nil || !strings.Contains(err.Error(), "different Windows session") {
		t.Fatalf("cross-session observation authorized absence: alive=%v err=%v", alive, err)
	}
}

func TestDetachedObservationRejectsMissingWindowsSessionIdentity(t *testing.T) {
	for _, start := range []string{"job|Local\\agent-env-nonexistent|1:1", "job|invalid|Local\\agent-env-nonexistent|1:1"} {
		if alive, err := (NativeDetached{}).Alive(context.Background(), ProcessIdentity{PID: 2147483647, StartID: start}); alive || err == nil {
			t.Fatalf("incomplete session identity authorized absence: %q alive=%v err=%v", start, alive, err)
		}
	}
}

func TestDetachedMissingGuardianRequiresDurableEmptyEvidence(t *testing.T) {
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	name := "Local\\agent-env-never-created"
	proof := filepath.Join(t.TempDir(), ".detached-never-created-empty")
	id := ProcessIdentity{PID: 2147483647, StartID: fmt.Sprintf("job|%d|%s|1:1|%s", session, name, proof)}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err == nil {
		t.Fatalf("guardian loss authorized cleanup: %v %v", alive, err)
	}
	if err := os.WriteFile(proof, []byte("different-job"), 0600); err != nil {
		t.Fatal(err)
	}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err == nil {
		t.Fatalf("foreign evidence authorized cleanup: %v %v", alive, err)
	}
	if err := os.WriteFile(proof, []byte(name), 0600); err != nil {
		t.Fatal(err)
	}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err != nil {
		t.Fatalf("confirmed empty job: %v %v", alive, err)
	}
}

func TestDetachedGuardianRecordsConfirmedEmptyJob(t *testing.T) {
	name := fmt.Sprintf("Local\\agent-env-guardian-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	wide, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	job, err := windows.CreateJobObject(nil, wide)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(job)
	dir := t.TempDir()
	proof := filepath.Join(dir, ".detached-"+strings.TrimPrefix(name, "Local\\agent-env-")+"-empty")
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	id := ProcessIdentity{PID: 2147483647, StartID: fmt.Sprintf("job|%d|%s|1:1|%s", session, name, proof)}
	if err := startDetachedGuardian(job, name, proof); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		alive, err := (NativeDetached{}).Alive(context.Background(), id)
		if err == nil && !alive {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("guardian did not complete empty job: %v %v", alive, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("guardian completion left a late write or open evidence handle: %v", err)
	}
}

func TestDetachedEmptyJobWaitsForPublishedCompletion(t *testing.T) {
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("completion-%d-%d", os.Getpid(), time.Now().UnixNano())
	name := "Local\\agent-env-" + suffix
	wide, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	job, err := windows.CreateJobObject(nil, wide)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(job)
	dir := t.TempDir()
	proof := filepath.Join(dir, ".detached-"+suffix+"-empty")
	id := ProcessIdentity{PID: 2147483647, StartID: fmt.Sprintf("job|%d|%s|1:1|%s", session, name, proof)}
	// A real empty kernel Job models the exact interval before its guardian has
	// finished. No timing or helper scheduling can make this assertion vacuous.
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); !alive || err != nil {
		t.Fatalf("empty job authorized deletion before proof: %v %v", alive, err)
	}
	// Fully written but unpublished temporary output still is not completion.
	pending := proof + ".pending"
	f, err := os.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(name); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); !alive || err != nil {
		f.Close()
		t.Fatalf("pending write authorized deletion: %v %v", alive, err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(pending, proof); err != nil {
		t.Fatal(err)
	}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err != nil {
		t.Fatalf("published completion blocked absence: %v %v", alive, err)
	}
	if err = os.RemoveAll(dir); err != nil {
		t.Fatalf("completion left writable/open evidence: %v", err)
	}
}

func TestDetachedEmptyProofPublicationFailurePreservesBarrier(t *testing.T) {
	dir := t.TempDir()
	proof := filepath.Join(dir, "empty")
	if err := os.WriteFile(proof+".pending", []byte("previous interrupted write"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishDetachedEmptyProof(proof, "owned-job"); err == nil {
		t.Fatal("overwrote interrupted publication")
	}
	if _, err := os.Stat(proof); !os.IsNotExist(err) {
		t.Fatalf("failed publication exposed completion: %v", err)
	}
}
