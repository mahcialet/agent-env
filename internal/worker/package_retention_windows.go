//go:build windows

package worker

import (
	"path/filepath"
	"strings"

	"github.com/mahcialet/agent-env/internal/evidence"
	"golang.org/x/sys/windows"
)

func retainedNativePath(path string) (*uint16, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(path, `\\?\`) {
		if strings.HasPrefix(path, `\\`) {
			path = `\\?\UNC\` + path[2:]
		} else {
			path = `\\?\` + path
		}
	}
	return windows.UTF16PtrFromString(path)
}

func nativeRetainedPublication() retainedPublication {
	return retainedPublication{
		// Windows namespace barriers use write-through native publication, not
		// a claim of POSIX directory fsync. The staged file was already synced.
		prepare: func(string) error { return nil },
		rename: func(from, to string) error {
			src, err := retainedNativePath(from)
			if err != nil {
				return err
			}
			dst, err := retainedNativePath(to)
			if err != nil {
				return err
			}
			return windows.MoveFileEx(src, dst, windows.MOVEFILE_WRITE_THROUGH)
		},
		confirm: func(staged, destination, _ string, data []byte) error {
			if staged == "" {
				return nil // native write-through directory move succeeded
			}
			// Failed publication may leave a readable directory. Re-publish the
			// verified identical package via file Sync + write-through replacement
			// instead of treating visibility as durability. The worker instance
			// lock and serial journal processing own retained package publication.
			return evidence.AtomicWrite(filepath.Join(destination, "package.json"), data, 0600)
		},
	}
}
