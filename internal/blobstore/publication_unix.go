//go:build !windows

package blobstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func nativePublication() publicationHooks {
	return publicationHooks{
		prepare: syncDirectory,
		rename:  os.Rename,
		confirm: func(_, destination, root string) error {
			// A visible verified duplicate may have come from a publisher that
			// failed after rename, so re-establish its durability barriers too.
			if err := syncDirectory(destination); err != nil {
				return err
			}
			if err := syncDirectory(root); err != nil {
				return err
			}
			return syncDirectory(filepath.Dir(root))
		},
	}
}

func lockPublication(context.Context, string) (func() error, error) {
	return func() error { return nil }, nil
}
