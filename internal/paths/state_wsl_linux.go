package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// ValidateStateFilesystem rejects WSL state on Windows-backed filesystems.
// This is a supported-storage boundary, not a filesystem durability probe.
func ValidateStateFilesystem(home string) error {
	osrelease, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return fmt.Errorf("inspect kernel for state filesystem policy: %w", err)
	}
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return fmt.Errorf("inspect kernel version for state filesystem policy: %w", err)
	}
	return validateWSLStateFilesystem(home, isWSLKernel(string(osrelease), string(version)), func(path string) (bool, error) {
		var stat unix.Statfs_t
		if err := unix.Statfs(path, &stat); err != nil {
			return false, err
		}
		// WSL2 exposes Windows drives through 9p. Conservatively exclude all
		// 9p state in WSL, including custom mountpoints and bind aliases.
		if uint64(stat.Type) == 0x01021997 {
			return true, nil
		}
		mounts, err := os.ReadFile("/proc/self/mountinfo")
		if err != nil {
			return false, err
		}
		return windowsStateMount(path, string(mounts)), nil
	})
}

func isWSLKernel(release, version string) bool {
	return strings.Contains(strings.ToLower(release+"\n"+version), "microsoft")
}

func validateWSLStateFilesystem(home string, wsl bool, inspect func(string) (bool, error)) error {
	if !wsl {
		return nil
	}
	canonical, err := CanonicalFuture(home)
	if err != nil {
		return err
	}
	probe := canonical
	for {
		_, err = os.Stat(probe)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) || filepath.Dir(probe) == probe {
			return err
		}
		probe = filepath.Dir(probe)
	}
	unsupported, err := inspect(probe)
	if err != nil {
		return fmt.Errorf("inspect WSL state filesystem: %w", err)
	}
	if unsupported {
		return fmt.Errorf("WSL state directory must use the Linux filesystem; Windows-backed and 9p state are unsupported: %q", home)
	}
	return nil
}

func windowsStateMount(path, mountinfo string) bool {
	best, windows := -1, false
	unescape := strings.NewReplacer("\\040", " ", "\\011", "\t", "\\012", "\n", "\\134", "\\")
	for _, line := range strings.Split(mountinfo, "\n") {
		left, right, ok := strings.Cut(line, " - ")
		fields, fs := strings.Fields(left), strings.Fields(right)
		if !ok || len(fields) < 5 || len(fs) < 1 {
			continue
		}
		mount := filepath.Clean(unescape.Replace(fields[4]))
		if len(mount) >= best && contained(mount, path) {
			best = len(mount)
			windows = fs[0] == "drvfs" || fs[0] == "9p"
		}
	}
	return windows
}
