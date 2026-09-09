package paths

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestNativeWindowsWSLStateAndCWDPreflight(t *testing.T) {
	for _, path := range []string{`\\wsl$\nonexistent-distro\state`, `\\?\UNC\wsl.localhost\nonexistent-distro\state`} {
		if err := ValidateStateFilesystem(path); !errors.Is(err, ErrWSLFilesystemUnsupported) {
			t.Fatalf("state: %v", err)
		}
		if err := ValidateExecutionDirectory(path); !errors.Is(err, ErrExecutionDirectoryUnsupported) || !errors.Is(err, ErrWSLFilesystemUnsupported) {
			t.Fatalf("cwd: %v", err)
		}
	}
	if err := ValidateStateFilesystem(filepath.Join(t.TempDir(), "future", "state")); err != nil {
		t.Fatal(err)
	}
}
