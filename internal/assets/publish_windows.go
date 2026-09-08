package assets

import "golang.org/x/sys/windows"

// A published asset is immutable. Replacing it on Windows creates a sharing
// violation window for concurrent readers, even when all writers have the same
// bytes. Move without REPLACE_EXISTING and let Materialize verify the winner.
func publishAsset(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, 0)
}
