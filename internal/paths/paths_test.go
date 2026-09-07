package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeLayoutsAndOverride(t *testing.T) {
	home := t.TempDir()
	for _, tc := range []struct{ os, suffix string }{{"linux", ".local/state/agent-env"}, {"darwin", "Library/Application Support/agent-env"}, {"windows", "AppData/Local/agent-env"}} {
		got, err := For(tc.os, home, func(string) string { return "" })
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, filepath.FromSlash(tc.suffix)); got != want {
			t.Fatalf("%s: %s != %s", tc.os, got, want)
		}
	}
	override := filepath.Join(home, "spaces 日本語")
	got, err := For(runtime.GOOS, home, func(k string) string {
		if k == "AGENT_ENV_HOME" {
			return override
		}
		return ""
	})
	if err != nil || got != override {
		t.Fatalf("override %q %v", got, err)
	}
	if _, err := os.Stat(override); !os.IsNotExist(err) {
		t.Fatal("resolving a path must not create it")
	}
	if _, err := For("linux", home, func(string) string { return "relative" }); err == nil {
		t.Fatal("relative state root accepted")
	}
}

func TestSourceConfinement(t *testing.T) {
	root := t.TempDir()
	for _, bad := range []string{"../escape", "a/../../escape", "/tmp/escape", `C:\\escape`, `a\b`} {
		if _, err := Within(root, bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if _, err := Within(root, "space 日本語/fixture.yaml"); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("host disallows symlink creation")
		}
		t.Fatal(err)
	}
	for _, p := range []string{"link", "link/not-created/yet"} {
		if _, err := Within(root, p); err == nil {
			t.Fatalf("symlink escape accepted %s", p)
		}
	}
}
