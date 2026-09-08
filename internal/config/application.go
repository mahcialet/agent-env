package config

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Application describes a build installed on a separately owned Android runtime.
type Application struct {
	Type             string           `yaml:"type" json:"type"`
	Source           string           `yaml:"source" json:"source"`
	Runtime          string           `yaml:"runtime" json:"runtime"`
	ProjectDirectory string           `yaml:"project_directory" json:"project_directory"`
	Build            ApplicationBuild `yaml:"build" json:"build"`
	Package          string           `yaml:"package" json:"package"`
	Activity         string           `yaml:"activity" json:"activity"`
	Reverse          []ReverseBinding `yaml:"reverse,omitempty" json:"reverse,omitempty"`
}
type ApplicationBuild struct {
	Command  []string `yaml:"command" json:"command"`
	Artifact string   `yaml:"artifact" json:"artifact"`
	Timeout  string   `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}
type ReverseBinding struct {
	DevicePort int    `yaml:"device_port" json:"device_port"`
	Endpoint   string `yaml:"endpoint" json:"endpoint"`
}

var packagePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
var activityPattern = regexp.MustCompile(`^(\.[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*|[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+)$`)

// ResolveEndpointReference rejects ambiguities when component/endpoint names contain dots.
func ResolveEndpointReference(m *Manifest, reference string) (string, string, error) {
	component, endpoint := "", ""
	for _, cn := range keys(m.Components) {
		for _, en := range keys(m.Components[cn].Endpoints) {
			if cn+"."+en == reference {
				if component != "" {
					return "", "", fmt.Errorf("ambiguous endpoint %q", reference)
				}
				component, endpoint = cn, en
			}
		}
	}
	if component == "" {
		return "", "", fmt.Errorf("unknown endpoint %q", reference)
	}
	return component, endpoint, nil
}

func validateApplications(m *Manifest) error {
	packages := map[string]map[string]string{}
	ports := map[string]map[int]string{}
	for _, n := range keys(m.Applications) {
		a := m.Applications[n]
		if a.Type != "flutter-android" {
			return fmt.Errorf("manifest: application %s has unsupported type %q", n, a.Type)
		}
		if _, ok := m.Sources[a.Source]; !ok {
			return fmt.Errorf("manifest: application %s references unknown source %q", n, a.Source)
		}
		if r, ok := m.Runtimes[a.Runtime]; !ok || r.Type != "android-emulator" {
			return fmt.Errorf("manifest: application %s requires an android-emulator runtime", n)
		}
		if err := RelativePath(a.ProjectDirectory); err != nil {
			return fmt.Errorf("manifest: application %s project_directory: %w", n, err)
		}
		if err := command("application "+n+" build.command", a.Build.Command); err != nil {
			return err
		}
		if err := duration("application "+n+" build.timeout", a.Build.Timeout); err != nil {
			return err
		}
		if strings.TrimSpace(a.Build.Artifact) == "" || path.Ext(a.Build.Artifact) != ".apk" {
			return fmt.Errorf("manifest: application %s build.artifact requires a nonempty .apk path", n)
		}
		if err := RelativePath(a.Build.Artifact); err != nil {
			return fmt.Errorf("manifest: application %s build.artifact: %w", n, err)
		}
		if !packagePattern.MatchString(a.Package) {
			return fmt.Errorf("manifest: application %s requires a valid Android package", n)
		}
		if !activityPattern.MatchString(a.Activity) {
			return fmt.Errorf("manifest: application %s requires a valid Android activity", n)
		}
		if packages[a.Runtime] == nil {
			packages[a.Runtime] = map[string]string{}
			ports[a.Runtime] = map[int]string{}
		}
		if prior, ok := packages[a.Runtime][a.Package]; ok {
			return fmt.Errorf("manifest: applications %s and %s duplicate package on runtime %s", prior, n, a.Runtime)
		}
		packages[a.Runtime][a.Package] = n
		for _, b := range a.Reverse {
			if b.DevicePort < 1 || b.DevicePort > 65535 {
				return fmt.Errorf("manifest: application %s device_port must be 1..65535", n)
			}
			if prior, ok := ports[a.Runtime][b.DevicePort]; ok {
				return fmt.Errorf("manifest: applications %s and %s duplicate device_port on runtime %s", prior, n, a.Runtime)
			}
			ports[a.Runtime][b.DevicePort] = n
			cn, en, err := ResolveEndpointReference(m, b.Endpoint)
			if err != nil {
				return fmt.Errorf("manifest: application %s reverse: %w", n, err)
			}
			ep := m.Components[cn].Endpoints[en]
			if m.Runtimes[m.Components[cn].Runtime].Type != "compose" || (ep.Protocol != "" && ep.Protocol != "tcp") {
				return fmt.Errorf("manifest: application %s reverse endpoint must use Compose TCP", n)
			}
		}
	}
	for _, n := range keys(m.Components) {
		c := m.Components[n]
		if c.Application == "" {
			continue
		}
		a, ok := m.Applications[c.Application]
		if !ok {
			return fmt.Errorf("manifest: component %s references unknown application %q", n, c.Application)
		}
		if c.Runtime != a.Runtime {
			return fmt.Errorf("manifest: component %s application runtime does not match component runtime", n)
		}
		dependencies := map[string]bool{}
		var visit func(string)
		visit = func(cn string) {
			for _, dep := range m.Components[cn].DependsOn {
				if !dependencies[dep] {
					dependencies[dep] = true
					visit(dep)
				}
			}
		}
		visit(n)
		for _, b := range a.Reverse {
			cn, _, _ := ResolveEndpointReference(m, b.Endpoint)
			if cn == n || !dependencies[cn] {
				return fmt.Errorf("manifest: component %s application reverse endpoint %s must belong to a dependency", n, b.Endpoint)
			}
		}
	}
	return nil
}
