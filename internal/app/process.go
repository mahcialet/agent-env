package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

// PersistentProcessProvider owns native process effects, never source or registry policy.
type PersistentProcessProvider interface {
	Prepare(context.Context, domain.Runtime) (domain.Runtime, error)
	Start(context.Context, domain.Runtime) (domain.Runtime, error)
	Inspect(context.Context, domain.Runtime) (RuntimeObservation, error)
	Logs(context.Context, domain.Runtime) (string, error)
	Destroy(context.Context, domain.Runtime) error
}

func (s *Service) startProcess(ctx context.Context, l *domain.Lease, index int) error {
	if s.Process == nil {
		return fmt.Errorf("persistent process provider unavailable")
	}
	r := l.Runtimes[index]
	prepared, err := s.Process.Prepare(ctx, r)
	l.Runtimes[index] = prepared
	if err != nil {
		return fmt.Errorf("prepare process %s: %w", r.Name, err)
	}
	if err = s.persist(ctx, *l); err != nil {
		return err
	}
	l.Runtimes[index].Process.State = "launching"
	l.Runtimes[index].Started = true
	l.Observed = "starting"
	if err = s.persist(ctx, *l); err != nil {
		return err
	}
	if err = s.event(ctx, l.ID, "runtime_up_requested", r.Name); err != nil {
		return err
	}
	started, startErr := s.Process.Start(ctx, l.Runtimes[index])
	l.Runtimes[index] = started
	// Save the adapter's prelaunch/identity result even when startup failed.
	if err = s.persist(ctx, *l); err != nil {
		return errors.Join(startErr, err)
	}
	if startErr != nil {
		return fmt.Errorf("start process %s: %w", r.Name, startErr)
	}
	return s.event(ctx, l.ID, "runtime_started", r.Name)
}

func (s *Service) processProbe(l domain.Lease, component string, probe config.Probe) (config.Probe, error) {
	var manifest config.Manifest
	if err := json.Unmarshal(l.Manifest, &manifest); err != nil {
		return probe, err
	}
	c := manifest.Components[component]
	spec := manifest.Runtimes[c.Runtime]
	if spec.Type != "process" {
		return probe, nil
	}
	var runtime domain.Runtime
	for _, r := range l.Runtimes {
		if r.Name == c.Runtime {
			runtime = r
			break
		}
	}
	if runtime.Process == nil {
		return probe, fmt.Errorf("process readiness snapshot missing")
	}
	expand := func(value string) (string, error) {
		var result strings.Builder
		for {
			start := strings.Index(value, "${")
			if start < 0 {
				result.WriteString(value)
				return result.String(), nil
			}
			result.WriteString(value[:start])
			value = value[start+2:]
			end := strings.IndexByte(value, '}')
			if end < 0 {
				return "", errors.New("unterminated process readiness reference")
			}
			key := value[:end]
			value = value[end+1:]
			replacement := ""
			switch {
			case strings.HasPrefix(key, "endpoint:"):
				endpoint, ok := c.Endpoints[strings.TrimPrefix(key, "endpoint:")]
				port := runtime.Process.Ports[endpoint.RuntimePort]
				if !ok || port <= 0 {
					return "", fmt.Errorf("unknown process endpoint %s", key)
				}
				replacement = strconv.Itoa(port)
			case key == "runtime_dir":
				replacement = runtime.Process.StateDirectory
			case key == "lease_id":
				replacement = l.ID
			case strings.HasPrefix(key, "port:"):
				port := runtime.Process.Ports[strings.TrimPrefix(key, "port:")]
				if port <= 0 {
					return "", fmt.Errorf("unknown process readiness port %s", key)
				}
				replacement = strconv.Itoa(port)
			case strings.HasPrefix(key, "env:"):
				var ok bool
				replacement, ok = os.LookupEnv(strings.TrimPrefix(key, "env:"))
				if !ok {
					return "", fmt.Errorf("missing process readiness environment %s", strings.TrimPrefix(key, "env:"))
				}
			default:
				return "", fmt.Errorf("unsupported process readiness reference %s", key)
			}
			result.WriteString(replacement)
		}
	}

	var err error
	probe.URL, err = expand(probe.URL)
	if err != nil {
		return probe, err
	}
	probe.Command = append([]string{}, probe.Command...)
	for i, arg := range probe.Command {
		probe.Command[i], err = expand(arg)
		if err != nil {
			return probe, err
		}
	}

	return probe, nil
}

func observeProcess(r *domain.Runtime, o RuntimeObservation, err error) {
	if r.Process == nil {
		return
	}
	switch {
	case errors.Is(err, domain.ErrResourceIdentity):
		r.Process.State = "quarantined"
	case err != nil: // Keep launch evidence state; uncertain observation is not absence.
	case !r.Started && r.Process.State == "released":
	case o.Ready:
		r.Process.State = "running"
	case o.Exists:
		r.Process.State = "degraded"
	default:
		// Preparation/launch states distinguish a never-started allocation from a
		// crash before identity persistence. Do not erase this recovery authority.
		if r.Process.ProcessID > 0 {
			r.Process.State = "stopped"
		}
	}
}

func (s *Service) cleanupProcess(ctx context.Context, l *domain.Lease, r *domain.Runtime) (bool, error) {
	if s.Process == nil || r.Process == nil {
		return false, errors.New("persistent process provider or snapshot unavailable")
	}
	observed, err := s.Process.Inspect(ctx, *r)
	if err != nil {
		observeProcess(r, observed, err)
		return false, err
	}
	if err := ownedResources(l.ID, observed.Resources); err != nil {
		return false, err
	}
	// Prepared evidence remains useful even if the root exited before readiness.
	if r.Process.Executable != "" {
		if err := s.event(ctx, l.ID, "runtime_logs_requested", r.Name); err != nil {
			return true, err
		}
		logs, err := s.Process.Logs(ctx, *r)
		if err != nil {
			return false, fmt.Errorf("collect process %s logs: %w", r.Name, err)
		}
		if err := s.saveTextArtifact(ctx, *l, "process-log-before-stop/"+r.Name, r.Name+"-process-before-stop.log", logs); err != nil {
			return true, err
		}
	}
	if err := s.event(ctx, l.ID, "runtime_down_requested", r.Name); err != nil {
		return true, err
	}
	if err := s.Process.Destroy(ctx, *r); err != nil {
		observeProcess(r, RuntimeObservation{}, err)
		return operationLost(ctx), err
	}
	remaining, err := s.Process.Inspect(ctx, *r)
	if err != nil || remaining.Exists {
		if err == nil {
			err = errors.New("persistent process remains after cleanup")
		}
		return false, err
	}
	if r.Process.Executable != "" {
		logs, err := s.Process.Logs(ctx, *r)
		if err != nil {
			return false, fmt.Errorf("collect final process %s logs: %w", r.Name, err)
		}
		if err := s.saveTextArtifact(ctx, *l, "process-log/"+r.Name, r.Name+"-process.log", logs); err != nil {
			return true, err
		}
	}
	r.Started = false
	r.Process.State = "released"
	if err := s.persist(ctx, *l); err != nil {
		return true, err
	}
	return false, nil
}

func processProbeSecrets(probe config.Probe) []string {
	var secrets []string
	for _, value := range probe.Command {
		for {
			start := strings.Index(value, "${env:")
			if start < 0 {
				break
			}
			value = value[start+6:]
			end := strings.IndexByte(value, '}')
			if end < 0 {
				break
			}
			if secret, ok := os.LookupEnv(value[:end]); ok && secret != "" {
				secrets = append(secrets, secret)
			}
			value = value[end+1:]
		}
	}
	return secrets
}

// Probes may stop their runtime. A successful probe is not native liveness proof.
func (s *Service) confirmProcessesReady(ctx context.Context, l *domain.Lease) error {
	for i := range l.Runtimes {
		r := &l.Runtimes[i]
		if r.Type != "process" {
			continue
		}
		observed, err := s.inspectRuntime(ctx, *r)
		observeProcess(r, observed, err)
		if err != nil {
			return err
		}
		if err := ownedResources(l.ID, observed.Resources); err != nil {
			return err
		}
		if !observed.Exists || !observed.Ready {
			return fmt.Errorf("persistent process %s stopped before readiness completed", r.Name)
		}
	}
	return nil
}
