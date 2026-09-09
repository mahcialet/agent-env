//go:build windows

package blobstore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

func nativePath(path string) (*uint16, error) {
	if !strings.HasPrefix(path, `\\?\`) {
		if strings.HasPrefix(path, `\\`) {
			path = `\\?\UNC\` + path[2:]
		} else {
			path = `\\?\` + path
		}
	}
	return windows.UTF16PtrFromString(path)
}

func nativeMove(from, to string, flags uint32) error {
	// Match evidence-file publication: extended native paths and write-through
	// rename, without shell commands or symbolic links.
	src, err := nativePath(from)
	if err != nil {
		return err
	}
	dst, err := nativePath(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(src, dst, flags|windows.MOVEFILE_WRITE_THROUGH)
}

func nativePublication() publicationHooks {
	return publicationHooks{
		// Windows has no POSIX directory-fsync contract. The synced file's
		// namespace is published through the native write-through operation.
		prepare: func(string) error { return nil },
		rename:  func(from, to string) error { return nativeMove(from, to, 0) },
		confirm: func(staged, destination, _ string) error {
			if staged == "" {
				return nil // successful native write-through publication
			}
			// A failed publisher may have left a visible winner. Republish the
			// synced, identical data through the native barrier instead of
			// acknowledging visibility alone or removing the enclosing directory.
			// Concurrent Open can reject a changed file identity conservatively;
			// it cannot return unverified bytes.
			return nativeMove(filepath.Join(staged, "data"), filepath.Join(destination, "data"), windows.MOVEFILE_REPLACE_EXISTING)
		},
	}
}

func lockPublication(ctx context.Context, root string) (func() error, error) {
	// Serialize verification/republication across handles and processes using
	// an OS-released byte-range lock, not a guessed stale PID.
	path, err := nativePath(filepath.Join(root, ".publication.lock"))
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	var info windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &info); err != nil || info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		windows.CloseHandle(handle)
		return nil, errors.New("CAS publication lock must be a regular file")
	}
	var position windows.Overlapped
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &position)
		if err == nil {
			return func() error {
				return errors.Join(windows.UnlockFileEx(handle, 0, 1, 0, &position), windows.CloseHandle(handle))
			}, nil
		}
		if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			windows.CloseHandle(handle)
			return nil, err
		}
		retry := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			retry.Stop()
			windows.CloseHandle(handle)
			return nil, ctx.Err()
		case <-deadline.C:
			retry.Stop()
			windows.CloseHandle(handle)
			return nil, fmt.Errorf("CAS publication lock unavailable: %w", err)
		case <-retry.C:
		}
	}
}
