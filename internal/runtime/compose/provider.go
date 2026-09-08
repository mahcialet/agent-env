package compose

import (
	"context"
	"fmt"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
)

// Client selects the recorded Compose provider. Empty provider is the legacy Docker identity.
// Selection never falls back after a provider failure.
type Client struct {
	Runner execx.Runner
	Policy policy.Policy
}

type provider interface {
	Doctor(context.Context) (map[string]string, error)
	Render(context.Context, domain.Runtime, []string) (Rendered, error)
	Up(context.Context, domain.Runtime) error
	Down(context.Context, domain.Runtime) error
	Logs(context.Context, domain.Runtime) (string, error)
	Inspect(context.Context, domain.Runtime) (Observation, error)
	Inventory(context.Context, string) ([]domain.Resource, error)
}

func (c Client) provider(name domain.ComposeProviderName) (provider, error) {
	switch domain.EffectiveComposeProvider(name) {
	case domain.ComposeProviderDocker:
		return dockerClient{Runner: c.Runner, Policy: c.Policy}, nil
	case domain.ComposeProviderPodman:
		return nil, fmt.Errorf("Podman provider implementation is not available yet")
	default:
		return nil, fmt.Errorf("unsupported Compose provider %q", name)
	}
}

func (c Client) Doctor(ctx context.Context) (map[string]string, error) {
	return c.DoctorFor(ctx, domain.ComposeProviderDocker)
}
func (c Client) DoctorFor(ctx context.Context, name domain.ComposeProviderName) (map[string]string, error) {
	p, err := c.provider(name)
	if err != nil {
		return nil, err
	}
	result, err := p.Doctor(ctx)
	if result == nil {
		result = map[string]string{}
	}
	result["provider"] = string(domain.EffectiveComposeProvider(name))
	return result, err
}
func (c Client) Render(ctx context.Context, r domain.Runtime, roots []string) (Rendered, error) {
	p, err := c.provider(r.Provider)
	if err != nil {
		return Rendered{}, err
	}
	return p.Render(ctx, r, roots)
}
func (c Client) Up(ctx context.Context, r domain.Runtime) error {
	p, err := c.provider(r.Provider)
	if err != nil {
		return err
	}
	return p.Up(ctx, r)
}
func (c Client) Down(ctx context.Context, r domain.Runtime) error {
	p, err := c.provider(r.Provider)
	if err != nil {
		return err
	}
	return p.Down(ctx, r)
}
func (c Client) Logs(ctx context.Context, r domain.Runtime) (string, error) {
	p, err := c.provider(r.Provider)
	if err != nil {
		return "", err
	}
	return p.Logs(ctx, r)
}
func (c Client) Inspect(ctx context.Context, r domain.Runtime) (Observation, error) {
	p, err := c.provider(r.Provider)
	if err != nil {
		return Observation{}, err
	}
	return p.Inspect(ctx, r)
}
func (c Client) Inventory(ctx context.Context, name string) ([]domain.Resource, error) {
	return c.InventoryFor(ctx, domain.ComposeProviderDocker, name)
}
func (c Client) InventoryFor(ctx context.Context, name domain.ComposeProviderName, identity string) ([]domain.Resource, error) {
	p, err := c.provider(name)
	if err != nil {
		return nil, err
	}
	items, err := p.Inventory(ctx, identity)
	for i := range items {
		if items[i].Metadata == nil {
			items[i].Metadata = map[string]string{}
		}
		items[i].Metadata["provider"] = string(domain.EffectiveComposeProvider(name))
		items[i].ID = string(domain.EffectiveComposeProvider(name)) + ":" + items[i].ID
	}
	return items, err
}
