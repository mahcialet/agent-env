package app

import (
	"fmt"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"path/filepath"
)

// Validate actual source and nested execution directories before reserving any
// lease resources. Prospective worktrees need not already exist.
func validateCreateExecutionDirectories(lease domain.Lease, m *config.Manifest, check func(string) error) error {
	roots := map[string]string{}
	validate := func(kind, path string) error {
		if path == "" {
			return nil
		}
		if err := check(path); err != nil {
			return fmt.Errorf("%s: %w", kind, err)
		}
		return nil
	}
	for _, source := range lease.Sources {
		roots[source.Alias] = source.WorktreePath
		for _, path := range []string{source.RepositoryPath, source.WorktreePath} {
			if err := validate("source execution directory", path); err != nil {
				return err
			}
		}
	}
	nested := func(kind, alias, relative string) error {
		root, ok := roots[alias]
		if !ok {
			return fmt.Errorf("%s source unavailable", kind)
		}
		return validate(kind, filepath.Join(root, filepath.FromSlash(relative)))
	}
	for _, r := range lease.Runtimes {
		if err := validate("runtime execution directory", r.Directory); err != nil {
			return err
		}
		declaration := m.Runtimes[r.Name]
		if r.Type == "process" {
			if err := nested("process execution directory", declaration.Source, declaration.WorkingDirectory); err != nil {
				return err
			}
		}
	}
	for _, a := range lease.Applications {
		if err := nested("application execution directory", a.Source, a.ProjectDirectory); err != nil {
			return err
		}
	}
	for _, spec := range m.Tests {
		if err := nested("named test execution directory", spec.Source, spec.WorkingDirectory); err != nil {
			return err
		}
	}
	for _, component := range lease.Components {
		for _, probe := range m.Components[component.Name].Readiness {
			if probe.Type == "command" {
				if err := nested("readiness execution directory", probe.Source, probe.WorkingDirectory); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
