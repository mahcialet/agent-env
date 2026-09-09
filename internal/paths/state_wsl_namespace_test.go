package paths

import (
	"errors"
	"testing"
)

func TestWindowsWSLNamespacesRejectedBeforeAccess(t *testing.T) {
	for _, path := range []string{`\\wsl$\Ubuntu\state`, `\\WSL.LOCALHOST\Ubuntu\state`, `//wsl.localhost/Ubuntu/state`, `\\?\UNC\wsl$\Ubuntu\state`, `\\?\unc\WSL.LOCALHOST\Ubuntu\state`, `\\.\UNC\wsl$\Ubuntu\state`, `\??\UNC\wsl.localhost\Ubuntu\state`, `\\wsl$`} {
		t.Run(path, func(t *testing.T) {
			noAccess := func(string) (string, error) {
				t.Fatal("direct namespace reached filesystem resolution")
				return "", nil
			}
			if err := validateWindowsStateFilesystem(path, noAccess); !errors.Is(err, ErrWSLFilesystemUnsupported) {
				t.Fatalf("state: %v", err)
			}
			if err := validateExecutionDirectoryPath("windows", path, noAccess); !errors.Is(err, ErrExecutionDirectoryUnsupported) || !errors.Is(err, ErrWSLFilesystemUnsupported) {
				t.Fatalf("cwd: %v", err)
			}
			if err := validateWSLNamespace("linux", path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWindowsWSLCanonicalAliasesRejected(t *testing.T) {
	for _, path := range []string{`\\wsl$\Ubuntu\future\state`, `\\?\UNC\wsl.localhost\Ubuntu\cwd`} {
		resolve := func(string) (string, error) { return path, nil }
		if err := validateWindowsStateFilesystem(`C:\alias\future\state`, resolve); !errors.Is(err, ErrWSLFilesystemUnsupported) {
			t.Fatalf("state alias: %v", err)
		}
		if err := validateExecutionDirectoryPath("windows", `C:\alias\cwd`, resolve); !errors.Is(err, ErrExecutionDirectoryUnsupported) || !errors.Is(err, ErrWSLFilesystemUnsupported) {
			t.Fatalf("cwd alias: %v", err)
		}
	}
}

func TestWindowsNativePathsUnaffectedByWSLPolicy(t *testing.T) {
	for _, path := range []string{`C:\state`, `\\server\share\state`, `\\wsl.localhost.example\share`, `\\wsl$backup\share`, `\\?\UNC\server\share`, `C:\wsl$\state`} {
		if err := validateWindowsStateFilesystem(path, func(path string) (string, error) { return path, nil }); err != nil {
			t.Fatalf("%q: %v", path, err)
		}
	}
}
