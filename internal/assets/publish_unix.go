//go:build !windows

package assets

import "os"

// Rename publishes the complete file atomically while existing Unix readers
// retain their open inode. Windows requires a non-replacing native move.
func publishAsset(source, destination string) error {
	return os.Rename(source, destination)
}
