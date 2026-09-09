package execx

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestExplicitRelativeExecutableUsesCommandDirectory(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	dir := t.TempDir()
	target, err := os.OpenFile(filepath.Join(dir, "argv probe.exe"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatalf("copy executable: %v %v", copyErr, closeErr)
	}
	args := []string{"two words", "日本語", "literal&argument", "%PATH%"}
	result, err := (OSRunner{}).Run(context.Background(), Command{
		Name: "./argv probe.exe", Dir: dir,
		Args:    append([]string{"-test.run=^TestProcessHelper$", "--"}, args...),
		Env:     map[string]string{"AGENT_ENV_EXECX_HELPER": "1", "GORACE": "atexit_sleep_ms=0"},
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Args []string
		Cwd  string
	}
	if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Args, args) {
		t.Fatalf("relative executable argv: %#v", got.Args)
	}
	actual, err := os.Stat(got.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(dir)
	if err != nil || !os.SameFile(actual, want) {
		t.Fatalf("child cwd %q differs from %q: %v", got.Cwd, dir, err)
	}
}
