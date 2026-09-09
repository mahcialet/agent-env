//go:build windows

package blobstore

import (
	"context"
	"errors"
	"testing"
)

func TestPublicationLockCancellationAndRelease(t *testing.T) {
	root := t.TempDir()
	release, err := lockPublication(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if unexpected, err := lockPublication(ctx, root); !errors.Is(err, context.Canceled) {
		if unexpected != nil {
			unexpected()
		}
		release()
		t.Fatalf("contended lock ignored cancellation: %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := lockPublication(context.Background(), root)
	if err != nil {
		t.Fatalf("released lock was not reusable: %v", err)
	}
	if err := again(); err != nil {
		t.Fatal(err)
	}
}
