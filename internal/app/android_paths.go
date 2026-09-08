package app

import (
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/paths"
	"path/filepath"
	"runtime"
	"strings"
)

// Check every generated runtime against every immutable input, not just its own
// template. Resolve existing ancestors without creating future lease paths.
func validateAndroidInputPaths(lease domain.Lease) error {
	for _, writable := range lease.Runtimes {
		if writable.Android == nil {
			continue
		}
		for _, input := range lease.Runtimes {
			if input.Android == nil {
				continue
			}
			for _, path := range []string{input.Android.TemplatePath, input.Android.SystemImage} {
				overlap, err := androidInputOverlap(writable.Directory, path)
				if err != nil {
					return fmt.Errorf("%w: Android paths: %v", ErrPrerequisite, err)
				}
				if overlap {
					return fmt.Errorf("%w: Android runtime %s directory overlaps immutable input %s", ErrPrerequisite, writable.Name, path)
				}
			}
		}
	}
	return nil
}

func androidInputOverlap(a, b string) (bool, error) {
	a, err := resolveFutureAndroidPath(a)
	if err != nil {
		return false, err
	}
	b, err = resolveFutureAndroidPath(b)
	if err != nil {
		return false, err
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		a, b = strings.ToLower(a), strings.ToLower(b)
	}
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true, nil
		}
	}
	return false, nil
}

func resolveFutureAndroidPath(path string) (string, error) {
	return paths.CanonicalFuture(path)
}
