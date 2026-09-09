//go:build !windows

package worker

import (
	"errors"
	"os"
	"path/filepath"
)

func syncRetainedDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func nativeRetainedPublication() retainedPublication {
	return retainedUnixPublication(syncRetainedFile)
}

func syncRetainedFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	opened, statErr := f.Stat()
	current, pathErr := os.Lstat(path)
	if statErr != nil || pathErr != nil || !opened.Mode().IsRegular() || !current.Mode().IsRegular() || !os.SameFile(opened, current) {
		return errors.Join(errors.New("unsafe retained package file"), statErr, pathErr, f.Close())
	}
	return errors.Join(f.Sync(), f.Close())
}

func retainedUnixPublication(syncExisting func(string) error) retainedPublication {
	return retainedPublication{
		prepare: syncRetainedDirectory,
		rename:  os.Rename,
		confirm: func(staged, destination, root string, _ []byte) error {
			// A duplicate can be visible after a prior failed directory barrier.
			// Do not cross the effect boundary until its namespace is synced too.
			if staged != "" {
				// Older workers wrote the winning file without Sync. Syncing our
				// discarded staging inode does not flush that retained inode.
				if err := syncExisting(filepath.Join(destination, "package.json")); err != nil {
					return err
				}
			}
			for _, dir := range []string{destination, root, filepath.Dir(root)} {
				if err := syncRetainedDirectory(dir); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
