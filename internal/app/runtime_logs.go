package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

// BoundedRuntimeLogProvider is required for interactive log display; ordinary
// RuntimeProvider.Logs remains the separate durable cleanup evidence path.
type BoundedRuntimeLogProvider interface {
	LogsBounded(context.Context, domain.Runtime) (string, error)
}

// RuntimeLogEntries isolates both live and retained logs using the component's
// exact runtime/service contract. Aggregate historical logs cannot be split.
func (s *Service) RuntimeLogEntries(ctx context.Context, lease domain.Lease, component string) (map[string]string, error) {
	entries := map[string]string{}
	const logResponseLimit = 2 << 20
	remaining := logResponseLimit
	add := func(name, data string) error {
		if len(data) > remaining {
			return errors.New("runtime log response exceeds 2 MiB; select a narrower component or run")
		}
		remaining -= len(data)
		entries[name] = data
		return nil
	}
	selectedRuntime := ""
	selectedAndroid := false
	selectedProcess := false
	selectedServices := []string{}
	allowedKinds := map[string]bool{}
	if component != "" {
		for _, c := range lease.Components {
			if c.Name == component {
				selectedRuntime = c.Runtime
				if c.Application != "" {
					allowedKinds["application-build/"+c.Application] = true
				}
				selectedServices = append([]string(nil), c.Services...)
				for _, service := range c.Services {
					allowedKinds["compose-log/"+c.Runtime+"/"+service] = true
				}
				break
			}
		}
		for _, runtime := range lease.Runtimes {
			if runtime.Name == selectedRuntime {
				selectedAndroid = runtime.Type == "android-emulator"
				selectedProcess = runtime.Type == "process"
				if selectedProcess {
					allowedKinds["process-log/"+selectedRuntime] = true
					allowedKinds["process-log-before-stop/"+selectedRuntime] = true
				}
				break
			}
		}
		if selectedRuntime == "" || (!selectedAndroid && !selectedProcess && len(selectedServices) == 0) {
			return nil, errors.New("component has no allocated runtime services in this lease")
		}
	}
	artifacts, err := s.Store.Artifacts(ctx, lease.ID)
	if err != nil {
		return nil, err
	}
	legacy, matched := false, false
	for _, artifact := range artifacts {
		if artifact.Kind == "compose-log" {
			legacy = true
			if component != "" {
				continue
			}
		} else if strings.HasPrefix(artifact.Kind, "compose-log/") || strings.HasPrefix(artifact.Kind, "application-build/") || strings.HasPrefix(artifact.Kind, "process-log/") || strings.HasPrefix(artifact.Kind, "process-log-before-stop/") {
			if component != "" && !allowedKinds[artifact.Kind] {
				continue
			}
		} else {
			continue
		}
		file, err := os.Open(artifact.Path)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, int64(remaining)+1))
		closeErr := file.Close()
		if err = errors.Join(readErr, closeErr); err != nil {
			return nil, err
		}
		if err = add(artifact.ID, string(data)); err != nil {
			return nil, err
		}
		matched = true
	}
	for _, runtime := range lease.Runtimes {
		if runtime.Type != "android-emulator" || (component != "" && runtime.Name != selectedRuntime) {
			continue
		}
		logs, err := androidProcessLogEntries(s.Home, lease.ID, runtime, int64(remaining))
		if err != nil {
			return nil, err
		}
		for name, data := range logs {
			if err = add(name, data); err != nil {
				return nil, err
			}
			matched = true
		}
	}
	if lease.Observed == "released" {
		if component != "" && !selectedAndroid && legacy && !matched {
			return nil, errors.New("legacy aggregate logs cannot isolate this component; omit --component to inspect the retained aggregate")
		}
		return entries, nil
	}
	for _, runtime := range lease.Runtimes {
		if runtime.Type == "android-emulator" {
			continue
		}
		if component != "" {
			if runtime.Name != selectedRuntime {
				continue
			}
			runtime.Services = append([]string(nil), selectedServices...)
		}
		var logs string
		var err error
		if runtime.Type == "process" {
			if s.Process == nil {
				return nil, errors.New("process provider unavailable")
			}
			logs, err = s.Process.Logs(ctx, runtime)
		} else {
			bounded, ok := s.Runtime.(BoundedRuntimeLogProvider)
			if !ok {
				return nil, errors.New("bounded runtime log provider is required for log display")
			}
			logs, err = bounded.LogsBounded(ctx, runtime)
		}
		if err != nil {
			return nil, err
		}
		if err = add(runtime.Name, evidence.RedactString(logs, evidence.InheritedSecrets())); err != nil {
			return nil, fmt.Errorf("runtime %s: %w", runtime.Name, err)
		}
	}
	return entries, nil
}
