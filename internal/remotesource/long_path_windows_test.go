//go:build windows

package remotesource

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLongPathProcessHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_LONG_PATH_HELPER") != "1" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(cwd)
	os.Exit(0)
}

func TestWindowsLongPathProcessDiagnostics(t *testing.T) {
	dir := t.TempDir()
	for len(dir) < 300 {
		dir = filepath.Join(dir, strings.Repeat("nested", 8))
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	prefixed := `\\?\` + dir
	if strings.HasPrefix(dir, `\\`) {
		prefixed = `\\?\UNC\` + strings.TrimPrefix(dir, `\\`)
	}
	for _, strategy := range []struct{ name, cwd string }{{"ordinary", dir}, {"extended", prefixed}} {
		for _, probe := range []struct {
			name string
			exe  string
			args []string
		}{
			{"go-helper", exe, []string{"-test.run=^TestLongPathProcessHelper$"}},
			{"git-version", "git", []string{"--version"}},
			{"git-config", "git", []string{"-c", "core.longpaths=true", "config", "--bool", "--get", "core.longpaths"}},
		} {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, probe.exe, probe.args...)
			cmd.Dir = strategy.cwd
			cmd.Env = append(os.Environ(), "AGENT_ENV_LONG_PATH_HELPER=1", "GORACE=atexit_sleep_ms=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
			output, err := cmd.CombinedOutput()
			cancel()
			fmt.Printf("cwd_strategy=%s probe=%s cwd_length=%d error=%v output=%q\n", strategy.name, probe.name, len(strategy.cwd), err, output)
		}
	}
	// This diagnostic does not replace the required native source/worktree
	// lifecycle test; that test continues to fail until the real defect is fixed.
}
