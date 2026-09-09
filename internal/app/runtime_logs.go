package app

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

// RuntimeLogEntries isolates both live and retained logs using the component's
// exact runtime/service contract. Aggregate historical logs cannot be split.
func (s *Service) RuntimeLogEntries(ctx context.Context, lease domain.Lease, component string) (map[string]string, error) {
	entries := map[string]string{}
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
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			return nil, err
		}
		entries[artifact.ID] = string(data)
		matched = true
	}
	for _, runtime := range lease.Runtimes {
		if runtime.Type != "android-emulator" || (component != "" && runtime.Name != selectedRuntime) {
			continue
		}
		logs, err := AndroidProcessLogEntries(s.Home, lease.ID, runtime)
		if err != nil {
			return nil, err
		}
		for name, data := range logs {
			entries[name] = data
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
			logs, err = s.Runtime.Logs(ctx, runtime)
		}
		if err != nil {
			return nil, err
		}
		entries[runtime.Name] = evidence.RedactString(logs, evidence.InheritedSecrets())
	}
	return entries, nil
}
