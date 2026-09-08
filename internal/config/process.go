package config

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProcessPort struct {
	Protocol string `yaml:"protocol" json:"protocol"`
}

var processEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Check field presence separately: null and empty incompatible fields are not
// defaults. yaml.Unmarshal resolves aliases and merge keys before this check.
func validateProcessPresence(data []byte, m *Manifest) error {
	var raw struct {
		Runtimes   map[string]map[string]any `yaml:"runtimes"`
		Components map[string]struct {
			Fields    map[string]any            `yaml:",inline"`
			Endpoints map[string]map[string]any `yaml:"endpoints"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	for name, fields := range raw.Runtimes {
		forbidden := []string{"working_directory", "command", "env", "ports"}
		if m.Runtimes[name].Type == "process" {
			forbidden = []string{"provider", "project_directory", "files", "avd"}
		}
		for _, key := range forbidden {
			if _, ok := fields[key]; ok {
				return fmt.Errorf("manifest: runtime %s type %s cannot declare %s", name, m.Runtimes[name].Type, key)
			}
		}
	}
	for name, c := range raw.Components {
		process := m.Runtimes[m.Components[name].Runtime].Type == "process"
		if process {
			if _, ok := c.Fields["compose_services"]; ok {
				return fmt.Errorf("manifest: process component %s cannot declare compose_services", name)
			}
		}
		for endpoint, fields := range c.Endpoints {
			forbidden := []string{"runtime_port"}
			if process {
				forbidden = []string{"service", "target", "protocol"}
			}
			for _, key := range forbidden {
				if _, ok := fields[key]; ok {
					return fmt.Errorf("manifest: component %s endpoint %s cannot declare %s", name, endpoint, key)
				}
			}
		}
	}
	return nil
}

func validateProcess(name string, r Runtime) error {
	baseName := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	reserved := baseName == "CON" || baseName == "PRN" || baseName == "AUX" || baseName == "NUL" || len(baseName) == 4 && (strings.HasPrefix(baseName, "COM") || strings.HasPrefix(baseName, "LPT")) && baseName[3] >= '1' && baseName[3] <= '9'
	if reserved || strings.HasSuffix(name, ".") {
		return fmt.Errorf("manifest: process runtime name %q is not portable as a directory", name)
	}

	if r.Provider != "" || r.ProjectDirectory != "" || len(r.Files) > 0 || r.AVD != "" {
		return fmt.Errorf("manifest: process runtime %s cannot use Compose or Android fields", name)
	}
	if strings.TrimSpace(r.WorkingDirectory) == "" || strings.Contains(r.WorkingDirectory, "${") {
		return fmt.Errorf("manifest: process runtime %s requires a literal working_directory", name)
	}
	if err := RelativePath(r.WorkingDirectory); err != nil {
		return fmt.Errorf("manifest: process runtime %s working_directory: %w", name, err)
	}
	if err := command("process runtime "+name, r.Command); err != nil {
		return err
	}
	if strings.Contains(r.Command[0], "${") {
		return fmt.Errorf("manifest: process executable cannot use interpolation")
	}
	if err := RelativePath(r.Command[0]); err != nil {
		return fmt.Errorf("manifest: process executable must be source-relative or a bare PATH name: %w", err)
	}
	base := strings.ToLower(path.Base(r.Command[0]))
	base = strings.TrimSuffix(base, ".exe")
	if strings.HasSuffix(base, ".bat") || strings.HasSuffix(base, ".cmd") {
		return fmt.Errorf("manifest: process Windows shell wrappers are unsupported")
	}
	if err := names("process port", r.Ports); err != nil {
		return err
	}
	for name, port := range r.Ports {
		if port.Protocol != "tcp" {
			return fmt.Errorf("manifest: process port %s requires protocol tcp", name)
		}
	}
	for _, arg := range r.Command[1:] {
		if err := processInterpolation(arg, r.Ports); err != nil {
			return err
		}
	}
	folded := map[string]bool{}
	for key, value := range r.Env {
		if !processEnvName.MatchString(key) || strings.ContainsRune(value, 0) || folded[strings.ToUpper(key)] {
			return fmt.Errorf("manifest: process runtime %s has invalid or case-colliding environment keys", name)
		}
		folded[strings.ToUpper(key)] = true
		if err := processInterpolation(value, r.Ports); err != nil {
			return err
		}
	}
	return nil
}

func processInterpolation(value string, ports map[string]ProcessPort) error {
	for {
		start := strings.Index(value, "${")
		if start < 0 {
			return nil
		}
		value = value[start+2:]
		end := strings.IndexByte(value, '}')
		if end < 0 {
			return fmt.Errorf("manifest: process has unterminated interpolation")
		}
		expr := value[:end]
		value = value[end+1:]
		if expr == "runtime_dir" || expr == "lease_id" {
			continue
		}
		if strings.HasPrefix(expr, "port:") {
			if _, ok := ports[strings.TrimPrefix(expr, "port:")]; ok {
				continue
			}
		}
		if strings.HasPrefix(expr, "env:") && processEnvName.MatchString(strings.TrimPrefix(expr, "env:")) {
			continue
		}
		return fmt.Errorf("manifest: unsupported process interpolation %q", expr)
	}
}

var processEndpointReference = regexp.MustCompile(`\$\{endpoint:([^}]+)\}`)

func validateProcessProbeReferences(p Probe, c Component, runtime Runtime, process bool) error {
	values := append([]string{p.URL}, p.Command...)
	for _, value := range values {
		refs := processEndpointReference.FindAllStringSubmatch(value, -1)
		for _, ref := range refs {
			if !process {
				return fmt.Errorf("endpoint interpolation requires a process component")
			}
			if _, ok := c.Endpoints[ref[1]]; !ok {
				return fmt.Errorf("unknown local endpoint %q", ref[1])
			}
		}
		if process {
			value = processEndpointReference.ReplaceAllString(value, "1")
			if p.Type == "http" {
				if strings.Contains(value, "${") {
					return fmt.Errorf("HTTP readiness accepts only local endpoint interpolation")
				}
			} else if err := processInterpolation(value, runtime.Ports); err != nil {
				return err
			}
		}
	}
	if process && len(p.Command) > 0 && strings.Contains(p.Command[0], "${") {
		return fmt.Errorf("readiness executable cannot use interpolation")
	}
	return nil
}
