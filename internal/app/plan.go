// Package app coordinates environment use cases through injected boundaries.
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/stack"
)

type SourceProvider interface {
	Resolve(context.Context, string, string) (domain.Source, error)
	Materialize(context.Context, domain.Source) error
	Inspect(context.Context, domain.Source) (SourceObservation, error)
	Remove(context.Context, domain.Source, bool) error
}
type SourceObservation struct {
	Exists, Registered, TrackedDirty bool
	Commit                           string
}

type PlanOptions struct {
	Repository, Stack, Ref string
	SourceRefs             map[string]string
}
type Plan struct {
	Repository      string             `json:"repository"`
	Stack           string             `json:"stack"`
	ManifestDigest  string             `json:"manifest_digest"`
	SourceSetDigest string             `json:"source_set_digest"`
	Sources         []domain.Source    `json:"sources"`
	Components      []domain.Component `json:"components"`
	Runtimes        []domain.Runtime   `json:"runtimes"`
	Diagnostics     []string           `json:"diagnostics"`
	Manifest        *config.Manifest   `json:"-"`
}

func BuildPlan(ctx context.Context, o PlanOptions, source SourceProvider) (Plan, error) {
	p := Plan{Stack: o.Stack, Diagnostics: []string{}}
	if o.Repository == "" {
		o.Repository = "."
	}
	abs, err := filepath.Abs(o.Repository)
	if err != nil {
		return p, err
	}
	p.Repository = abs
	m, err := config.Load(abs)
	if err != nil {
		return p, err
	}
	p.Manifest = m
	p.ManifestDigest = config.Digest(m)
	if o.Stack == "" {
		return p, fmt.Errorf("--stack is required; choose one of %s", strings.Join(sortedKeys(m.Stacks), ", "))
	}
	components, err := stack.Resolve(m, o.Stack)
	if err != nil {
		return p, err
	}
	for alias := range o.SourceRefs {
		if _, ok := m.Sources[alias]; !ok {
			return p, fmt.Errorf("--source refers to unknown alias %q", alias)
		}
	}
	if o.Ref != "" && len(m.Sources) != 1 {
		return p, fmt.Errorf("--ref requires one source; use --source alias=ref for multi-repository manifests")
	}
	for _, alias := range sortedKeys(m.Sources) {
		spec := m.Sources[alias]
		if spec.Writable {
			return p, fmt.Errorf("source %q requests writable checkout; MVP supports review mode only", alias)
		}
		repo := spec.Repository
		if !filepath.IsAbs(repo) {
			repo = filepath.Join(abs, filepath.FromSlash(repo))
		}
		ref := spec.DefaultRef
		if o.Ref != "" {
			ref = o.Ref
		}
		if override, ok := o.SourceRefs[alias]; ok {
			ref = override
		}
		resolved, err := source.Resolve(ctx, repo, ref)
		if err != nil {
			return p, fmt.Errorf("source %s: %w", alias, err)
		}
		resolved.Alias = alias
		resolved.CheckoutMode = "detached"
		resolved.Writable = false
		p.Sources = append(p.Sources, resolved)
	}
	p.SourceSetDigest = domain.SourceDigest(p.Sources)
	byRuntime := map[string]int{}
	for _, name := range components {
		c := m.Components[name]
		p.Components = append(p.Components, domain.Component{Name: name, Runtime: c.Runtime, Services: c.ComposeServices, Capabilities: c.Provides})
		i, exists := byRuntime[c.Runtime]
		if !exists {
			r := m.Runtimes[c.Runtime]
			i = len(p.Runtimes)
			byRuntime[c.Runtime] = i
			p.Runtimes = append(p.Runtimes, domain.Runtime{Name: c.Runtime, Type: r.Type, Source: r.Source, Directory: r.ProjectDirectory, Files: r.Files, Services: []string{}})
		}
		for _, service := range c.ComposeServices {
			if !contains(p.Runtimes[i].Services, service) {
				p.Runtimes[i].Services = append(p.Runtimes[i].Services, service)
			}
		}
	}
	return p, nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
