// Package stack resolves explicit roots into deterministic dependency closure.
package stack

import (
	"fmt"
	"sort"

	"github.com/mahcialet/agent-env/internal/config"
)

func Resolve(m *config.Manifest, name string) ([]string, error) {
	if err := config.Validate(m); err != nil {
		return nil, err
	}
	s, ok := m.Stacks[name]
	if !ok {
		return nil, fmt.Errorf("unknown stack %q; choose a declared stack", name)
	}
	result := []string{}
	seen := map[string]bool{}
	var visit func(string)
	visit = func(n string) {
		if seen[n] {
			return
		}
		seen[n] = true
		deps := append([]string(nil), m.Components[n].DependsOn...)
		sort.Strings(deps)
		for _, dep := range deps {
			visit(dep)
		}
		result = append(result, n)
	}
	roots := append([]string(nil), s.Roots...)
	sort.Strings(roots)
	for _, root := range roots {
		visit(root)
	}
	return result, nil
}
