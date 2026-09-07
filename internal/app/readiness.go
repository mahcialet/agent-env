package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
)

func (s *Service) probeReady(ctx context.Context, l *domain.Lease) error {
	var manifest config.Manifest
	if err := json.Unmarshal(l.Manifest, &manifest); err != nil {
		return err
	}
	for _, component := range l.Components {
		for index, probe := range manifest.Components[component.Name].Readiness {
			if probe.Type == "compose" {
				continue
			}
			if err := s.runProbe(ctx, *l, component.Name, index, probe); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) runProbe(ctx context.Context, l domain.Lease, component string, index int, p config.Probe) error {
	timeout := 30 * time.Second
	if p.Timeout != "" {
		timeout, _ = time.ParseDuration(p.Timeout)
	}
	interval := time.Second
	if p.Interval != "" {
		interval, _ = time.ParseDuration(p.Interval)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var last error
	for {
		switch p.Type {
		case "http":
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
			if err != nil {
				return err
			}
			client := &http.Client{Timeout: 5 * time.Second}
			response, err := client.Do(req)
			last = err
			if err == nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
				response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode >= 400 {
					last = fmt.Errorf("HTTP status %d", response.StatusCode)
				}
			}
		case "command":
			if err := s.event(ctx, l.ID, "readiness_command_requested", fmt.Sprintf("%s probe %d", component, index)); err != nil {
				return err
			}
			if s.Runner == nil {
				return fmt.Errorf("command probe requires runner")
			}
			var root string
			for _, source := range l.Sources {
				if source.Alias == p.Source {
					root = source.WorktreePath
				}
			}
			if root == "" {
				return fmt.Errorf("readiness source %q missing", p.Source)
			}
			directory, err := paths.Within(root, p.WorkingDirectory)
			if err != nil {
				return err
			}
			result, err := s.Runner.Run(ctx, execx.Command{Name: p.Command[0], Args: p.Command[1:], Dir: directory, Timeout: timeout})
			last = err
			if e := s.saveTextArtifact(context.WithoutCancel(ctx), l, "readiness", fmt.Sprintf("%s-%d-probe.log", component, index), result.Stdout+"\n"+result.Stderr); e != nil {
				return e
			}
		default:
			return fmt.Errorf("unsupported probe %q", p.Type)
		}
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("component %s readiness probe %d (%s) failed: %w: %v", component, index, p.Type, ctx.Err(), last)
		case <-time.After(interval):
		}
	}
}

func (s *Service) Endpoints(ctx context.Context, l domain.Lease) (map[string]string, error) {
	var m config.Manifest
	if err := json.Unmarshal(l.Manifest, &m); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, r := range l.Runtimes {
		o, err := s.inspectRuntime(ctx, r)
		if err != nil {
			return nil, err
		}
		for _, c := range l.Components {
			if c.Runtime != r.Name {
				continue
			}
			for name, e := range m.Components[c.Name].Endpoints {
				protocol := e.Protocol
				if protocol == "" {
					protocol = "tcp"
				}
				if address, ok := o.Endpoints[fmt.Sprintf("%s/%d/%s", e.Service, e.Target, protocol)]; ok {
					result[c.Name+"."+name] = address
				}
			}
		}
		for name, address := range o.Endpoints {
			result[r.Name+"."+strings.ReplaceAll(name, "/", ".")] = address
		}
	}
	return result, nil
}

// Reconciliation refreshes read-only HTTP health, but never reruns a repository
// command (which may have migration or other side effects) during list/show.
func (s *Service) probeHealth(ctx context.Context, l domain.Lease) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var m config.Manifest
	if err := json.Unmarshal(l.Manifest, &m); err != nil {
		return err
	}
	for _, c := range l.Components {
		for i, p := range m.Components[c.Name].Readiness {
			if p.Type != "http" {
				continue
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
			if err != nil {
				return err
			}
			resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
			if err != nil {
				return fmt.Errorf("component %s HTTP health probe %d: %w", c.Name, i, err)
			}
			resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 400 {
				return fmt.Errorf("component %s HTTP health probe %d returned %d", c.Name, i, resp.StatusCode)
			}
		}
	}
	return nil
}
