// Package paths defines native state paths without creating them during plans.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func Resolve() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return For(runtime.GOOS, home, os.Getenv)
}

func For(goos, home string, env func(string) string) (string, error) {
	var root string
	if override := env("AGENT_ENV_HOME"); override != "" {
		root = override
	} else {
		switch goos {
		case "windows":
			root = env("LOCALAPPDATA")
			if root == "" {
				root = filepath.Join(home, "AppData", "Local")
			}
			root = filepath.Join(root, "agent-env")
		case "darwin":
			root = filepath.Join(home, "Library", "Application Support", "agent-env")
		default:
			root = env("XDG_STATE_HOME")
			if root == "" {
				root = filepath.Join(home, ".local", "state")
			}
			root = filepath.Join(root, "agent-env")
		}
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("state directory must be absolute: %q", root)
	}
	return filepath.Clean(root), nil
}

// Within also rejects a resolved symlink escaping the allocated source root.
func Within(root, relative string) (string, error) {
	if relative == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) || strings.Contains(relative, "\\") || strings.Contains(relative, ":") {
		return "", fmt.Errorf("expected a portable relative path: %q", relative)
	}
	p := filepath.Join(root, filepath.FromSlash(relative))
	if !contained(root, p) {
		return "", fmt.Errorf("path escapes source: %q", relative)
	}
	probe := p
	for {
		resolved, err := filepath.EvalSymlinks(probe)
		if err == nil {
			canonical, e := filepath.EvalSymlinks(root)
			if e != nil {
				return "", e
			}
			if !contained(canonical, resolved) {
				return "", fmt.Errorf("symlink escapes source: %q", relative)
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if probe == filepath.Clean(root) {
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	return p, nil
}

func contained(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
