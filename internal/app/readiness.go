package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
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
	probeSecrets := append(evidence.InheritedSecrets(), processProbeSecrets(p)...)
	var expandErr error
	p, expandErr = s.processProbe(l, component, p)
	if expandErr != nil {
		return expandErr
	}
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
			last = s.readinessCommand(ctx, l, component, index, p, directory, timeout, probeSecrets)
			if errors.Is(last, execx.ErrProcessTreeUnconfirmed) || errors.Is(last, execx.ErrOutputIncomplete) || errors.Is(last, context.Canceled) || operationLost(ctx) {
				return last
			}
		default:
			return fmt.Errorf("unsupported probe %q", p.Type)
		}
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("component %s readiness probe %d (%s) failed: %w: %v", component, index, p.Type, ctx.Err(), evidence.RedactString(fmt.Sprint(last), probeSecrets))
		case <-time.After(interval):
		}
	}
}

// Each command attempt has the same durable completion barrier as a named test.
// A retry must never conceal a prior attempt whose descendants or evidence are
// incomplete, and a later destroy must still see that attempt after Create ends.
func (s *Service) readinessCommand(ctx context.Context, l domain.Lease, component string, index int, p config.Probe, directory string, timeout time.Duration, secrets []string) error {
	run := domain.CommandRun{ID: newID(), LeaseID: l.ID, Name: fmt.Sprintf("readiness:%s:%d", component, index), Source: p.Source, Directory: directory, StartedAt: time.Now().UTC(), ExitCode: -1, Status: "running"}
	for _, arg := range p.Command {
		run.Argv = append(run.Argv, evidence.RedactString(arg, secrets))
	}
	if err := s.Store.SaveRun(ctx, run); err != nil {
		return err
	}
	result, commandErr := s.runWithCancellation(ctx, run.ID, execx.Command{Name: p.Command[0], Args: p.Command[1:], Dir: directory, Timeout: timeout})
	// Retain only known classification sentinels, never the runner's potentially
	// credential-bearing original error in the public error chain.
	var safetyErr error
	for _, sentinel := range []error{execx.ErrProcessTreeUnconfirmed, execx.ErrOutputIncomplete} {
		if errors.Is(commandErr, sentinel) {
			safetyErr = errors.Join(safetyErr, sentinel)
		}
	}
	var diagnostic error
	if commandErr != nil {
		diagnostic = errors.New(evidence.RedactString(commandErr.Error(), secrets))
		if errors.Is(commandErr, context.Canceled) {
			diagnostic = errors.Join(context.Canceled, diagnostic)
		}
	}
	finishCtx := context.WithoutCancel(ctx)
	artifactErr := s.saveTextArtifact(finishCtx, l, "readiness", fmt.Sprintf("%s-%d-%s-probe.log", component, index, run.ID), evidence.RedactString(result.Stdout+"\n"+result.Stderr, secrets))
	if artifactErr != nil {
		safetyErr = errors.Join(safetyErr, execx.ErrOutputIncomplete, errors.New(evidence.RedactString(artifactErr.Error(), secrets)))
	}
	if safetyErr != nil {
		return errors.Join(safetyErr, diagnostic)
	}
	if operationLost(ctx) {
		return errors.Join(domain.ErrLockLost, diagnostic)
	}
	run.FinishedAt = time.Now().UTC()
	run.ExitCode = result.ExitCode
	run.Status = "passed"
	if commandErr != nil || result.ExitCode != 0 {
		run.Status = "failed"
		if diagnostic == nil {
			diagnostic = fmt.Errorf("readiness command exited with status %d", result.ExitCode)
		}
	}
	if err := s.Store.SaveRun(finishCtx, run); err != nil {
		return errors.Join(execx.ErrOutputIncomplete, diagnostic, errors.New(evidence.RedactString(err.Error(), secrets)))
	}
	return diagnostic
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
				key := fmt.Sprintf("%s/%d/%s", e.Service, e.Target, protocol)
				if r.Type == "process" {
					key = e.RuntimePort
				}
				if address, ok := o.Endpoints[key]; ok {
					result[c.Name+"."+name] = address
				}
			}
		}
		for name, address := range o.Endpoints {
			key := r.Name + "." + strings.ReplaceAll(name, "/", ".")
			if _, declared := result[key]; !declared {
				result[key] = address
			}
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
			p, err := s.processProbe(l, c.Name, p)
			if err != nil {
				return err
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
