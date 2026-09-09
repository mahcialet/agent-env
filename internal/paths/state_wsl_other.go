//go:build !linux && !windows

package paths

// ValidateStateFilesystem applies the WSL-only state storage boundary.
func ValidateStateFilesystem(string) error { return nil }
