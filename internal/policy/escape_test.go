package policy

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBindSymlinkAndVolumeDriverCannotEscapePolicy(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlink privilege unavailable")
		}
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "outside"), filepath.Join(root, "outside", "new-file")} {
		cfg := Config{Services: map[string]Service{"api": {Volumes: []Mount{{Type: "bind", Source: path, Target: "/config"}}}}}
		d, err := Defaults().Evaluate(cfg, []string{"api"}, []string{root})
		if err != nil || len(d) != 1 || d[0].Code != "POLICY_EXTERNAL_BIND" {
			t.Fatalf("symlink host mount allowed: %v %v", d, err)
		}
	}
	cfg := Config{Services: map[string]Service{"api": {Volumes: []Mount{{Type: "volume", Source: "data", Target: "/host"}}}}, Volumes: map[string]NamedResource{"data": {Driver: "local", DriverOpts: map[string]any{"type": "none", "o": "bind", "device": "/"}}}}
	d, err := Defaults().Evaluate(cfg, []string{"api"}, []string{root})
	if err != nil || len(d) != 1 || d[0].Code != "POLICY_VOLUME_DRIVER" {
		t.Fatalf("volume driver host escape allowed: %v %v", d, err)
	}
}
