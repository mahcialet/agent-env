package paths

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWSLKernelDetection(t *testing.T) {
	for _, test := range []struct {
		release, version string
		want             bool
	}{
		{"6.6.87.2-microsoft-standard-WSL2", "", true},
		{"4.4.0", "Microsoft Linux", true},
		{"6.8.0-generic", "Linux version Ubuntu", false},
	} {
		if got := isWSLKernel(test.release, test.version); got != test.want {
			t.Fatalf("kernel detection=%v want %v", got, test.want)
		}
	}
}

func TestWSLStateFilesystemExistingAncestorAndAliases(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "custom mount 日本語")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{real, alias} {
		home := filepath.Join(base, "future", "state")
		for _, blocked := range []bool{false, true} {
			called := false
			err := validateWSLStateFilesystem(home, true, func(probe string) (bool, error) {
				called = true
				if probe != real {
					t.Fatalf("inspected %q instead of canonical ancestor %q", probe, real)
				}
				return blocked, nil
			})
			if !called || (err != nil) != blocked {
				t.Fatalf("blocked=%v err=%v called=%v", blocked, err, called)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("preflight created state: %v", err)
			}
		}
	}
	if err := validateWSLStateFilesystem(real, false, func(string) (bool, error) { t.Fatal("native Linux probed WSL policy"); return true, nil }); err != nil {
		t.Fatal(err)
	}
	want := errors.New("statfs unavailable")
	if err := validateWSLStateFilesystem(real, true, func(string) (bool, error) { return false, want }); !errors.Is(err, want) {
		t.Fatalf("inspection error lost: %v", err)
	}
}

func TestWSLWindowsMountDetection(t *testing.T) {
	mounts := "1 0 8:1 / / rw - ext4 /dev/root rw\n2 1 0:1 / /custom\\040drive rw - drvfs C: rw\n3 1 0:1 /sub /bind rw - 9p C: rw\n4 2 8:2 / /custom\\040drive/linux rw - ext4 /dev/other rw\n"
	for _, test := range []struct {
		path string
		want bool
	}{{"/custom drive/future", true}, {"/bind/state", true}, {"/custom drive/linux/state", false}, {"/custom drive-other/state", false}, {"/home/user", false}} {
		if got := windowsStateMount(test.path, mounts); got != test.want {
			t.Fatalf("%s: got %v want %v", test.path, got, test.want)
		}
	}
}
