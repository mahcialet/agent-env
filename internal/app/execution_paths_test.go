package app

import (
	"errors"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/paths"
	"path/filepath"
	"testing"
)

func TestCreateExecutionDirectoriesChecksActualNestedPaths(t *testing.T) {
	root := t.TempDir()
	lease := domain.Lease{Sources: []domain.Source{{Alias: "source", RepositoryPath: filepath.Join(root, "repo"), WorktreePath: filepath.Join(root, "worktree")}}, Runtimes: []domain.Runtime{{Name: "process", Type: "process", Directory: filepath.Join(root, "worktree")}}, Components: []domain.Component{{Name: "component"}}, Applications: []domain.Application{{Source: "source", ProjectDirectory: "mobile/project"}}}
	m := &config.Manifest{Runtimes: map[string]config.Runtime{"process": {Source: "source", WorkingDirectory: "deep/runtime"}}, Tests: map[string]config.Test{"check": {Source: "source", WorkingDirectory: "tests/nested"}}, Components: map[string]config.Component{"component": {Readiness: []config.Probe{{Type: "command", Source: "source", WorkingDirectory: "probe/nested"}}}}}
	seen := map[string]bool{}
	if err := validateCreateExecutionDirectories(lease, m, func(path string) error { seen[path] = true; return nil }); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"repo", "worktree", "worktree/deep/runtime", "worktree/mobile/project", "worktree/tests/nested", "worktree/probe/nested"} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if !seen[path] {
			t.Fatalf("execution directory not validated: %s", relative)
		}
		err := validateCreateExecutionDirectories(lease, m, func(candidate string) error {
			if candidate == path {
				return paths.ErrExecutionDirectoryUnsupported
			}
			return nil
		})
		if !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) {
			t.Fatalf("unsupported %s accepted: %v", relative, err)
		}
	}
}
