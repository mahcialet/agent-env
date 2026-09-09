package paths

// ValidateStateFilesystem rejects Windows state addressed through WSL shares.
func ValidateStateFilesystem(home string) error {
	return validateWindowsStateFilesystem(home, CanonicalFuture)
}
