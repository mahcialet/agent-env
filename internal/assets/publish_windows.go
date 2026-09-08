package assets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// A published asset is immutable. Replacing it on Windows creates a sharing
// violation window for concurrent readers, even when all writers have the same
// bytes. Move without REPLACE_EXISTING and let Materialize verify the winner.
func publishAsset(source, destination string) error {
	from, err := assetMovePath(source)
	if err != nil {
		return err
	}
	to, err := assetMovePath(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, 0)
}

// MoveFileEx needs extended absolute names for paths beyond MAX_PATH; unlike
// os file operations it does not receive Go's automatic long-path conversion.
func assetMovePath(path string) (*uint16, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(abs, `\\?\`) && !strings.HasPrefix(abs, `\\.\`) {
		if strings.HasPrefix(abs, `\\`) {
			abs = `\\?\UNC\` + abs[2:]
		} else {
			abs = `\\?\` + abs
		}
	}
	return windows.UTF16PtrFromString(abs)
}

func readAsset(path string) ([]byte, error) {
	return readAssetUntil(path, time.Now().Add(2*time.Second))
}

// Even a losing non-replacing move can briefly hold a conflicting Windows
// handle. Retry only native sharing or byte-range locking conflicts. Keep Go's
// file opening so extended Windows paths retain their standard support.
// Persistent locks remain errors.
func readAssetUntil(path string, deadline time.Time) ([]byte, error) {
	for {
		data, err := readAssetOnce(path)
		if err == nil || (!errors.Is(err, windows.ERROR_SHARING_VIOLATION) && !errors.Is(err, windows.ERROR_LOCK_VIOLATION)) || !time.Now().Before(deadline) {
			return data, err
		}
		time.Sleep(min(10*time.Millisecond, time.Until(deadline)))
	}
}

func readAssetOnce(path string) ([]byte, error) {
	return os.ReadFile(path)
}
