package execx

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writePEFixture(t *testing.T, path string) {
	t.Helper()
	header := make([]byte, 132)
	copy(header, "MZ")
	binary.LittleEndian.PutUint32(header[60:64], 128)
	copy(header[128:], "PE\x00\x00")
	if err := os.WriteFile(path, header, 0700); err != nil {
		t.Fatal(err)
	}
}

func TestNativeExecutableBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "renamed-tool")
	writePEFixture(t, path)
	for _, goos := range []string{"linux", "darwin"} {
		if err := validateNativeExecutable("renamed-tool", dir, goos); !errors.Is(err, ErrCrossOSExecution) {
			t.Fatalf("%s accepted PE executable: %v", goos, err)
		}
	}
	if err := validateNativeExecutable(path, dir, "windows"); err != nil {
		t.Fatalf("native Windows executable rejected: %v", err)
	}
	if err := validateNativeExecutable("wsl.exe", dir, "windows"); !errors.Is(err, ErrCrossOSExecution) {
		t.Fatalf("Windows-to-WSL bridge accepted: %v", err)
	}
	// A suffix is not an executable format. Native binaries and ordinary missing
	// commands must keep their existing semantics rather than being mislabeled.
	if err := os.WriteFile(filepath.Join(dir, "linux.exe"), []byte("\x7fELF"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"linux.exe", "missing.exe"} {
		if err := validateNativeExecutable(name, dir, "linux"); err != nil {
			t.Fatalf("%s incorrectly rejected: %v", name, err)
		}
	}
}

func TestNonWindowsRunnerRefusesDirectPEBeforeEffects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PE is the native executable format on Windows")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "renamed-tool")
	writePEFixture(t, tool)
	alias := filepath.Join(dir, "tool-link")
	if err := os.Symlink(tool, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, name := range []string{tool, "./renamed-tool", "renamed-tool", alias} {
		t.Run(filepath.Base(name)+name[:1], func(t *testing.T) {
			spec := Command{Name: name, Dir: dir}
			result, err := (OSRunner{}).Run(context.Background(), spec)
			if !errors.Is(err, ErrCrossOSExecution) || result.ExitCode != -1 {
				t.Fatalf("runner did not refuse before start: %+v %v", result, err)
			}
			out, stderr := filepath.Join(dir, "stdout"), filepath.Join(dir, "stderr")
			id, err := (NativeDetached{}).Start(context.Background(), spec, out, stderr)
			if !errors.Is(err, ErrCrossOSExecution) || !errors.Is(err, ErrProcessNotStarted) || id.PID != 0 {
				t.Fatalf("detached did not refuse before start: %+v %v", id, err)
			}
			for _, path := range []string{out, stderr} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("refused executable created evidence file: %s: %v", path, err)
				}
			}
		})
	}
}
