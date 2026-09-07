package execx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestBatchWrapperRoundTrip(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".cmd", ".bat"} {
		t.Run(ext, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "tools 日本語 with spaces")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			wrapper := filepath.Join(dir, "argument probe"+ext)
			if err := os.WriteFile(wrapper, []byte("@echo off\r\n\""+exe+"\" -test.run=^TestProcessHelper$ -- %*\r\n"), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"", "two words", `quote "inside"`, "日本語", `C:\directory with space\`, "x&y", `%PATH%`, "!literal!", "a^b"}
			r, err := (OSRunner{}).Run(context.Background(), Command{Name: wrapper, Args: args, Env: map[string]string{"AGENT_ENV_EXECX_HELPER": "1"}, Timeout: 10 * time.Second})
			if err != nil {
				t.Fatalf("%v: %s", err, r.Stderr)
			}
			var got struct{ Args []string }
			if err := json.Unmarshal([]byte(r.Stdout), &got); err != nil {
				t.Fatalf("%v: %q", err, r.Stdout)
			}
			if !reflect.DeepEqual(got.Args, args) {
				t.Fatalf("got %#v; want %#v", got.Args, args)
			}
		})
	}
}
