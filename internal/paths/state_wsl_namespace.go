package paths

import (
	"errors"
	"fmt"
	"strings"
)

var ErrWSLFilesystemUnsupported = errors.New("Windows execution requires native Windows paths; WSL UNC paths are unsupported")

// validateWSLNamespace checks only known interop namespaces, without accessing
// the filesystem. Arbitrary UNC shares are outside this policy.
func validateWSLNamespace(goos, path string) error {
	if goos != "windows" {
		return nil
	}
	p := strings.ToLower(strings.ReplaceAll(path, "/", "\\"))
	for _, prefix := range []string{`\\?\unc\`, `\\.\unc\`, `\??\unc\`} {
		if strings.HasPrefix(p, prefix) {
			p = `\\` + strings.TrimPrefix(p, prefix)
			break
		}
	}
	if !strings.HasPrefix(p, `\\`) {
		return nil
	}
	server := strings.SplitN(strings.TrimPrefix(p, `\\`), `\`, 2)[0]
	if server == "wsl$" || server == "wsl.localhost" {
		return fmt.Errorf("%w: %q", ErrWSLFilesystemUnsupported, path)
	}
	return nil
}

func validateWindowsStateFilesystem(home string, canonicalize func(string) (string, error)) error {
	if err := validateWSLNamespace("windows", home); err != nil {
		return err
	}
	resolved, err := canonicalize(home)
	if err != nil {
		return err
	}
	return validateWSLNamespace("windows", resolved)
}
