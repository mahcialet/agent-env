package remotesource

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mahcialet/agent-env/internal/execx"
)

func TestGitRefusesWindowsInteropTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows Git is native on Windows")
	}
	dir := t.TempDir()
	pe := make([]byte, 132)
	copy(pe, "MZ")
	binary.LittleEndian.PutUint32(pe[60:64], 128)
	copy(pe[128:], "PE\x00\x00")
	if err := os.WriteFile(filepath.Join(dir, "git"), pe, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := git(context.Background(), dir, "--version"); !errors.Is(err, execx.ErrCrossOSExecution) {
		t.Fatalf("Git did not reject direct Windows interop before execution: %v", err)
	}
}
