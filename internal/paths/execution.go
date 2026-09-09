package paths

import (
	"errors"
	"fmt"
	"runtime"
	"unicode/utf16"
)

var ErrExecutionDirectoryUnsupported = errors.New("execution directory outside supported platform range")

// ValidateExecutionDirectory checks an existing or prospective native command
// working directory without creating it. Windows reserves headroom below legacy
// MAX_PATH for the OS and external tools; native Linux/macOS have no added cap.
func ValidateExecutionDirectory(path string) error {
	return validateExecutionDirectoryPath(runtime.GOOS, path, CanonicalFuture)
}

func validateExecutionDirectoryPath(goos, path string, canonicalize func(string) (string, error)) error {
	if err := validateWSLNamespace(goos, path); err != nil {
		return errors.Join(ErrExecutionDirectoryUnsupported, err)
	}
	resolved, err := canonicalize(path)
	if err != nil {
		return err
	}
	return validateExecutionDirectory(goos, resolved)
}

func validateExecutionDirectory(goos, resolved string) error {
	if err := validateWSLNamespace(goos, resolved); err != nil {
		return errors.Join(ErrExecutionDirectoryUnsupported, err)
	}
	if goos == "windows" {
		units := len(utf16.Encode([]rune(resolved)))
		if units > 240 {
			return fmt.Errorf("%w: Windows resolved working directory uses %d UTF-16 code units; maximum 240; choose a shorter state home or repository path", ErrExecutionDirectoryUnsupported, units)
		}
	}
	return nil
}
