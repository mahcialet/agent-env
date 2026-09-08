package cli

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

type composeDoctor interface {
	DoctorFor(context.Context, domain.ComposeProviderName) (map[string]string, error)
}

func doctorCompose(ctx context.Context, path, explicit string, client composeDoctor) (map[string]any, error) {
	selected := map[domain.ComposeProviderName]bool{}
	if explicit != "" {
		name := domain.ComposeProviderName(explicit)
		if name != domain.ComposeProviderDocker && name != domain.ComposeProviderPodman {
			return nil, fmt.Errorf("unknown Compose provider %q", explicit)
		}
		selected[name] = true
	}
	if path != "" {
		m, err := config.Load(path)
		if err != nil {
			return nil, err
		}
		if explicit == "" {
			for _, r := range m.Runtimes {
				if r.Type == "compose" {
					selected[domain.EffectiveComposeProvider(domain.ComposeProviderName(r.Provider))] = true
				}
			}
		}
	} else if explicit == "" {
		selected[domain.ComposeProviderDocker] = true
	}
	names := []string{}
	for n := range selected {
		names = append(names, string(n))
	}
	sort.Strings(names)
	report := map[string]any{}
	var failures []error
	for _, name := range names {
		d, err := client.DoctorFor(ctx, domain.ComposeProviderName(name))
		report[name] = d
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
		}
	}
	return report, errors.Join(failures...)
}
