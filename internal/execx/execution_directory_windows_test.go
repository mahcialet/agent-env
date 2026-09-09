package execx

import (
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/paths"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnsupportedExecutionDirectoryDoesNotStartOrCreateOutput(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, strings.Repeat("x", 150), strings.Repeat("y", 100))
	result, err := (OSRunner{}).Run(context.Background(), Command{Name: "unused.exe", Dir: directory})
	if !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) || result.ExitCode != -1 {
		t.Fatalf("runner accepted unsupported cwd: %+v %v", result, err)
	}
	out := filepath.Join(root, "stdout.log")
	stderr := filepath.Join(root, "stderr.log")
	id, err := (NativeDetached{}).Start(context.Background(), Command{Name: "unused.exe", Dir: directory}, out, stderr)
	if !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) || !errors.Is(err, ErrProcessNotStarted) || id.PID != 0 {
		t.Fatalf("detached unsupported cwd: %+v %v", id, err)
	}
	for _, p := range []string{out, stderr} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unsupported cwd produced output: %v", err)
		}
	}
}
