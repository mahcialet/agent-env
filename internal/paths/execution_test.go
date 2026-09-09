package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionDirectoryPlatformBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		bad        bool
	}{
		{"ascii-limit", "C:/" + strings.Repeat("a", 237), false},
		{"ascii-over", "C:/" + strings.Repeat("a", 238), true},
		{"bmp-limit", "C:/" + strings.Repeat("日", 237), false},
		{"supplementary-limit", "C:/" + strings.Repeat("😀", 118) + "a", false},
		{"supplementary-over", "C:/" + strings.Repeat("😀", 119), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExecutionDirectory("windows", tc.path)
			if errors.Is(err, ErrExecutionDirectoryUnsupported) != tc.bad {
				t.Fatalf("boundary: %v", err)
			}
			for _, platform := range []string{"linux", "darwin"} {
				if err := validateExecutionDirectory(platform, tc.path+strings.Repeat("b", 500)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestExecutionDirectoryProspectivePathDoesNotCreate(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "not-created", "runtime")
	if err := ValidateExecutionDirectory(target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "not-created")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("validation created directory: %v", err)
	}
}
