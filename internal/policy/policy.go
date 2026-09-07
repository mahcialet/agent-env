// Package policy evaluates host permissions separately from parsing and execution.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Policy struct {
	DefaultTTL           time.Duration
	MaxTTL               time.Duration
	MaxActive            int
	GCGrace              time.Duration
	HeartbeatGrace       time.Duration
	ForbidPrivileged     bool
	ForbidHostNetwork    bool
	ForbidDockerSocket   bool
	ForbidContainerName  bool
	AllowedExternalRoots []string
}

func Defaults() Policy {
	return Policy{DefaultTTL: 4 * time.Hour, MaxTTL: 24 * time.Hour, MaxActive: 8, GCGrace: 5 * time.Minute, HeartbeatGrace: time.Minute, ForbidPrivileged: true, ForbidHostNetwork: true, ForbidDockerSocket: true, ForbidContainerName: true}
}

func (p Policy) TTL(ttl time.Duration) (time.Duration, error) {
	if ttl == 0 {
		ttl = p.DefaultTTL
	}
	if ttl <= 0 || ttl > p.MaxTTL {
		return 0, fmt.Errorf("TTL must be positive and at most %s", p.MaxTTL)
	}
	return ttl, nil
}

type Diagnostic struct {
	Code    string `json:"code"`
	Service string `json:"service"`
	Message string `json:"message"`
}
type Config struct {
	Services map[string]Service       `json:"services"`
	Networks map[string]NamedResource `json:"networks"`
	Volumes  map[string]NamedResource `json:"volumes"`
}
type NamedResource struct {
	External   bool           `json:"external"`
	Name       string         `json:"name"`
	Driver     string         `json:"driver"`
	DriverOpts map[string]any `json:"driver_opts"`
}
type Service struct {
	ContainerName string                     `json:"container_name"`
	Privileged    bool                       `json:"privileged"`
	NetworkMode   string                     `json:"network_mode"`
	DependsOn     map[string]json.RawMessage `json:"depends_on"`
	Ports         []Port                     `json:"ports"`
	Volumes       []Mount                    `json:"volumes"`
	Networks      map[string]json.RawMessage `json:"networks"`
}
type Port struct {
	Target    uint16 `json:"target"`
	Published any    `json:"published"`
	HostIP    string `json:"host_ip"`
	Protocol  string `json:"protocol"`
}
type Mount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// Services includes Compose's implicit dependency closure in deterministic order.
func Services(c Config, selected []string) ([]string, error) {
	states := map[string]int{}
	var result []string
	var visit func(string) error
	visit = func(name string) error {
		if states[name] == 2 {
			return nil
		}
		if states[name] == 1 {
			return fmt.Errorf("Compose dependency cycle at %s", name)
		}
		s, ok := c.Services[name]
		if !ok {
			return fmt.Errorf("selected Compose service %q is missing", name)
		}
		states[name] = 1
		keys := make([]string, 0, len(s.DependsOn))
		for d := range s.DependsOn {
			keys = append(keys, d)
		}
		sort.Strings(keys)
		for _, d := range keys {
			if err := visit(d); err != nil {
				return err
			}
		}
		states[name] = 2
		result = append(result, name)
		return nil
	}
	for _, name := range selected {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p Policy) Evaluate(c Config, selected []string, sourceRoots []string) ([]Diagnostic, error) {
	services, err := Services(c, selected)
	if err != nil {
		return nil, err
	}
	var result []Diagnostic
	for _, name := range services {
		s := c.Services[name]
		add := func(code, msg string) { result = append(result, Diagnostic{code, name, msg}) }
		if s.ContainerName != "" && p.ForbidContainerName {
			add("POLICY_CONTAINER_NAME", "remove explicit container_name so concurrent leases have isolated names")
		}
		if s.Privileged && p.ForbidPrivileged {
			add("POLICY_PRIVILEGED", "privileged containers are forbidden by host policy")
		}
		if s.NetworkMode == "host" && p.ForbidHostNetwork {
			add("POLICY_HOST_NETWORK", "host networking bypasses project isolation")
		}
		for _, port := range s.Ports {
			published := fmt.Sprint(port.Published)
			if port.Published != nil && published != "" && published != "0" {
				add("POLICY_FIXED_PORT", "replace fixed published host port "+published+" with dynamic publishing")
			}
		}
		for _, v := range s.Volumes {
			if p.ForbidDockerSocket && (strings.Contains(strings.ToLower(v.Source), "docker.sock") || strings.Contains(strings.ToLower(v.Target), "docker.sock") || strings.Contains(strings.ToLower(v.Source), "docker_engine")) {
				add("POLICY_DOCKER_SOCKET", "Docker socket mounts expose the host daemon")
			}
			if v.Type == "bind" && !underAny(v.Source, append(append([]string{}, sourceRoots...), p.AllowedExternalRoots...)) {
				add("POLICY_EXTERNAL_BIND", "bind source outside allocated worktrees/allowed roots: "+v.Source)
			}
			if v.Type == "volume" {
				if n, ok := c.Volumes[v.Source]; ok {
					if n.External {
						add("POLICY_EXTERNAL_VOLUME", "external volume "+v.Source+" is shared across leases")
					}
					if len(n.DriverOpts) > 0 || n.Driver != "" && n.Driver != "local" {
						add("POLICY_VOLUME_DRIVER", "custom volume drivers/options can mount shared host resources; use a project-scoped local volume without driver_opts")
					}
				}
			}
		}
		for network := range s.Networks {
			if n, ok := c.Networks[network]; ok && n.External {
				add("POLICY_EXTERNAL_NETWORK", "external network "+network+" is shared across leases")
			}
		}
	}
	return result, nil
}

func underAny(path string, roots []string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	for _, root := range roots {
		canonicalRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		canonicalPath := path
		for {
			resolved, e := filepath.EvalSymlinks(canonicalPath)
			if e == nil {
				suffix, relErr := filepath.Rel(canonicalPath, path)
				if relErr != nil {
					return false
				}
				canonicalPath = filepath.Join(resolved, suffix)
				break
			}
			if !os.IsNotExist(e) {
				return false
			}
			parent := filepath.Dir(canonicalPath)
			if parent == canonicalPath {
				return false
			}
			canonicalPath = parent
		}
		rel, err := filepath.Rel(canonicalRoot, canonicalPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}
