package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsStateRemovalRetriesHeldFile(t *testing.T) {
	_, r, _ := prepared(t)
	path := filepath.Join(r.Process.StateDirectory, "held-cache")
	if err := os.WriteFile(path, []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	nativePath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(nativePath, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	release := sync.OnceFunc(func() {
		if err := syscall.CloseHandle(handle); err != nil {
			t.Error(err)
		}
	})
	defer release()
	// Reproduce the exact native deletion failure before testing recovery.
	if err := os.RemoveAll(r.Process.StateDirectory); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("expected sharing violation: %v", err)
	}
	released := make(chan struct{})
	go func() { defer close(released); time.Sleep(50 * time.Millisecond); release() }()
	err = removeStateDirectory(context.Background(), r, os.RemoveAll, retryableStateRemoval, 2*time.Second)
	<-released
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.Process.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("state remains: %v", err)
	}
}

func TestWindowsStateRemovalOnlyRetriesSharingViolation(t *testing.T) {
	if !retryableStateRemoval(&os.PathError{Op: "remove", Path: "fixture", Err: windows.ERROR_SHARING_VIOLATION}) {
		t.Fatal("wrapped sharing violation not recognized")
	}
	if retryableStateRemoval(syscall.ERROR_ACCESS_DENIED) || retryableStateRemoval(syscall.ERROR_FILE_NOT_FOUND) {
		t.Fatal("non-sharing error considered retryable")
	}
}
