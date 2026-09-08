package paths

import (
	"os"
	"path/filepath"
)

// CanonicalFuture resolves existing path ancestors while retaining absent suffixes.
func CanonicalFuture(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	probe, suffix := path, ""
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			return filepath.Join(resolved, suffix), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		// A dangling link must not be mistaken for an uncreated directory.
		if info, statErr := os.Lstat(probe); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", err
		}
		suffix = filepath.Join(filepath.Base(probe), suffix)
		probe = parent
	}
}
