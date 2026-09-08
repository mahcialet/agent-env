package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateRemovalRetriesTransientErrorAfterPathValidation(t *testing.T) {
	_, r, _ := prepared(t)
	transient := errors.New("transient sharing violation")
	calls := 0
	remove := func(path string) error {
		calls++
		if calls == 1 {
			return transient
		}
		return os.RemoveAll(path)
	}
	if err := removeStateDirectory(context.Background(), r, remove, func(err error) bool { return errors.Is(err, transient) }, time.Second); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("attempts=%d", calls)
	}
	if _, err := os.Stat(r.Process.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("state removal incomplete: %v", err)
	}
}

func TestStateRemovalPreservesNonSharingFailure(t *testing.T) {
	_, r, _ := prepared(t)
	denied := os.ErrPermission
	calls := 0
	err := removeStateDirectory(context.Background(), r, func(string) error { calls++; return denied }, func(error) bool { return false }, time.Second)
	if !errors.Is(err, denied) || calls != 1 {
		t.Fatalf("error=%v attempts=%d", err, calls)
	}
}

func TestStateRemovalCancellationStopsRetries(t *testing.T) {
	_, r, _ := prepared(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	err := removeStateDirectory(ctx, r, func(string) error { calls++; cancel(); return errors.New("sharing") }, func(error) bool { return true }, time.Second)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("error=%v attempts=%d", err, calls)
	}
}

func TestStateRemovalRetainsFailureAtRetryDeadline(t *testing.T) {
	_, r, _ := prepared(t)
	transient := errors.New("sharing")
	calls := 0
	err := removeStateDirectory(context.Background(), r, func(string) error { calls++; return transient }, func(error) bool { return true }, 75*time.Millisecond)
	if !errors.Is(err, transient) || calls == 0 {
		t.Fatalf("error=%v attempts=%d", err, calls)
	}
	if _, err := os.Stat(r.Process.StateDirectory); err != nil {
		t.Fatalf("failed removal discarded state: %v", err)
	}
}

func TestStateRemovalRevalidatesOwnerBeforeRetry(t *testing.T) {
	_, r, _ := prepared(t)
	calls := 0
	err := removeStateDirectory(context.Background(), r, func(string) error {
		calls++
		if err := os.WriteFile(filepath.Join(r.Process.Directory, "owner.json"), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
		return errors.New("sharing")
	}, func(error) bool { return true }, time.Second)
	if err == nil || calls != 1 {
		t.Fatalf("changed ownership retried removal: error=%v attempts=%d", err, calls)
	}
}
