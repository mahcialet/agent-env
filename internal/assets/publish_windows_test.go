package assets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestMaterializeWindowsLongPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "asset state 日本語")
	for len(root) < 280 {
		root = filepath.Join(root, strings.Repeat("long", 12))
	}
	info, err := Describe("fixture.bin", "1", embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Materialize(root, info, embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Materialize(root, info, embeddedFixture)
	if err != nil || first != second {
		t.Fatalf("long-path reuse: %q, %q, %v", first, second, err)
	}
}

func TestAssetMoveWindowsPathNamespaces(t *testing.T) {
	for input, want := range map[string]string{
		`C:\asset state 日本語\file`:   `\\?\C:\asset state 日本語\file`,
		`\\server\share\file`:       `\\?\UNC\server\share\file`,
		`\\?\C:\asset\file`:         `\\?\C:\asset\file`,
		`\\?\UNC\server\share\file`: `\\?\UNC\server\share\file`,
	} {
		got, err := assetMovePath(input)
		if err != nil {
			t.Fatal(err)
		}
		if actual := windows.UTF16PtrToString(got); actual != want {
			t.Errorf("%q: got %q, want %q", input, actual, want)
		}
	}
}

func TestReadAssetWindowsSharingConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte("winner"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			windows.CloseHandle(lock)
		}
	})
	if _, err := readAssetOnce(path); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("fixture does not block reads: %v", err)
	}
	start := time.Now()
	if _, err := readAssetUntil(path, start.Add(30*time.Millisecond)); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("persistent conflict must propagate: %v", err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatal("sharing conflict was not retried until deadline")
	}
	result := make(chan error, 1)
	go func() {
		_, err := readAsset(path)
		result <- err
	}()
	// Keep the conflicting handle alive long enough to exercise retries before
	// releasing it; the separate deadline case above proves retries occurred.
	time.Sleep(30 * time.Millisecond)
	if err := windows.CloseHandle(lock); err != nil {
		t.Fatal(err)
	}
	closed = true
	if err := <-result; err != nil {
		t.Fatalf("released sharing conflict did not recover: %v", err)
	}
	got, err := readAsset(path)
	if err != nil || string(got) != "winner" {
		t.Fatalf("read changed immutable content: %q, %v", got, err)
	}
}

func TestReadAssetWindowsDoesNotRetryMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	start := time.Now()
	_, err := readAssetUntil(path, start.Add(2*time.Second))
	if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		t.Fatalf("missing file error not propagated: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("non-sharing error was retried")
	}
}

func TestPublishAssetPreservesWindowsWinner(t *testing.T) {
	root := t.TempDir()
	source, destination := filepath.Join(root, "candidate"), filepath.Join(root, "published")
	if err := os.WriteFile(source, []byte("candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("winner"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishAsset(source, destination); err == nil {
		t.Fatal("published winner was replaced")
	}
	for path, want := range map[string]string{source: "candidate", destination: "winner"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v; want %q", path, got, err, want)
		}
	}
}
