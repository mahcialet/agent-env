// Package compose implements isolated Docker Compose projects through an injected runner.
package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
)

const projectLabel = "com.docker.compose.project"
const leaseLabel = "io.agent-env.lease"
const runtimeLabel = "io.agent-env.runtime"

type dockerClient struct {
	Runner execx.Runner
	Policy policy.Policy
}
type Rendered struct {
	JSON     []byte
	Digest   string
	Services []string
}
type Observation struct {
	Ready       bool
	Exists      bool
	Resources   []domain.Resource
	Diagnostics []string
	Endpoints   map[string]string
}

var projectPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)
var versionPattern = regexp.MustCompile(`^v?([0-9]+)\.`)

func (c dockerClient) run(ctx context.Context, directory string, args ...string) (execx.Result, error) {
	return c.runBounded(ctx, directory, 0, args...)
}
func (c dockerClient) runBounded(ctx context.Context, directory string, captureLimit int, args ...string) (execx.Result, error) {
	if c.Runner == nil {
		return execx.Result{}, fmt.Errorf("Compose runner is not configured")
	}
	r, err := c.Runner.Run(ctx, execx.Command{CaptureLimit: captureLimit, Name: "docker", Args: args, Dir: directory, Timeout: 2 * time.Minute, UnsetEnv: []string{"COMPOSE_FILE", "COMPOSE_PROJECT_NAME", "COMPOSE_PROFILES"}})
	if err != nil {
		return r, fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	if r.ExitCode != 0 {
		return r, fmt.Errorf("docker %s failed with exit %d", strings.Join(args, " "), r.ExitCode)
	}
	return r, nil
}

// Doctor captures the selected context before checking the plugin and daemon.
// Persist context and use it for every later operation on the runtime.
func (c dockerClient) Doctor(ctx context.Context) (map[string]string, error) {
	result := map[string]string{}
	r, err := c.run(ctx, "", "context", "show")
	if err != nil {
		return result, fmt.Errorf("Docker context unavailable; install/configure Docker: %w", err)
	}
	contextName := strings.TrimSpace(r.Stdout)
	if contextName == "" || strings.ContainsAny(contextName, "\r\n\x00") {
		return result, fmt.Errorf("Docker returned an invalid active context")
	}
	result["context"] = contextName
	r, err = c.run(ctx, "", "--context", contextName, "compose", "version", "--short")
	if err != nil {
		return result, fmt.Errorf("Docker Compose v2 or later plugin is required: %w", err)
	}
	v := strings.TrimSpace(r.Stdout)
	match := versionPattern.FindStringSubmatch(v)
	if len(match) != 2 {
		return result, fmt.Errorf("cannot recognize Compose version %q", v)
	}
	major, _ := strconv.Atoi(match[1])
	if major < 2 {
		return result, fmt.Errorf("Compose v2 or later required, got %s", v)
	}
	result["compose_version"] = v
	r, err = c.run(ctx, "", "--context", contextName, "version", "--format", "{{json .}}")
	if err != nil {
		return result, fmt.Errorf("Docker daemon unavailable in context %s: %w", contextName, err)
	}
	var version struct{ Client, Server struct{ Version string } }
	if err := json.Unmarshal([]byte(r.Stdout), &version); err != nil || version.Server.Version == "" {
		return result, fmt.Errorf("Docker daemon version missing or invalid in context %s", contextName)
	}
	result["docker_version"], result["server_version"] = version.Client.Version, version.Server.Version
	return result, nil
}

func identity(r domain.Runtime) error {
	if !projectPattern.MatchString(r.Project) {
		return fmt.Errorf("invalid isolated Compose project %q", r.Project)
	}
	if r.Context == "" || strings.ContainsAny(r.Context, "\r\n\x00") {
		return fmt.Errorf("runtime %s has no valid recorded Docker context; run doctor before allocation", r.Name)
	}
	return nil
}
func composeArgs(r domain.Runtime) ([]string, error) {
	if err := identity(r); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(r.Directory) {
		return nil, fmt.Errorf("Compose project_directory must be absolute: %q", r.Directory)
	}
	files := r.Files
	// Once saved, the validated normalized configuration is startup/cleanup authority.
	if r.ConfigPath != "" {
		files = []string{r.ConfigPath}
		if r.ConfigDigest != "" {
			data, err := os.ReadFile(r.ConfigPath)
			if err != nil {
				return nil, fmt.Errorf("read recorded Compose configuration: %w", err)
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != r.ConfigDigest {
				return nil, fmt.Errorf("recorded Compose configuration digest mismatch; quarantine instead of using modified cleanup instructions")
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("runtime %s has no explicit Compose files", r.Name)
	}
	args := []string{"--context", r.Context, "compose", "-p", r.Project, "--project-directory", r.Directory}
	for _, p := range files {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("Compose file must be absolute: %q", p)
		}
		args = append(args, "-f", p)
	}
	return args, nil
}

func (c dockerClient) Render(ctx context.Context, r domain.Runtime, sourceRoots []string) (Rendered, error) {
	args, err := composeArgs(r)
	if err != nil {
		return Rendered{}, err
	}
	args = append(args, "config", "--format", "json")
	out, err := c.run(ctx, r.Directory, args...)
	if err != nil {
		return Rendered{}, fmt.Errorf("render Compose configuration: %w", err)
	}
	return validateRendered([]byte(out.Stdout), r, sourceRoots, c.Policy)
}

func validateRendered(data []byte, r domain.Runtime, sourceRoots []string, hostPolicy policy.Policy) (Rendered, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Rendered{}, fmt.Errorf("invalid rendered Compose JSON: %w", err)
	}
	var cfg policy.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Rendered{}, fmt.Errorf("invalid rendered Compose policy data: %w", err)
	}
	if len(r.Services) == 0 {
		return Rendered{}, fmt.Errorf("runtime %s has no selected Compose services", r.Name)
	}
	services, err := policy.Services(cfg, r.Services)
	if err != nil {
		return Rendered{}, err
	}
	diagnostics, err := hostPolicy.Evaluate(cfg, services, sourceRoots)
	if err != nil {
		return Rendered{}, err
	}
	if len(diagnostics) > 0 {
		messages := make([]string, 0, len(diagnostics))
		for _, d := range diagnostics {
			messages = append(messages, d.Code+" "+d.Service+": "+d.Message)
		}
		return Rendered{}, fmt.Errorf("Compose host policy rejected configuration: %s", strings.Join(messages, "; "))
	}
	for _, name := range services {
		s := cfg.Services[name]
		for key := range s.Networks {
			if n, ok := cfg.Networks[key]; ok && n.Name != "" && !strings.HasPrefix(n.Name, r.Project+"_") {
				return Rendered{}, fmt.Errorf("network %s has globally shared name %q; remove explicit name", key, n.Name)
			}
		}
		for _, mount := range s.Volumes {
			if mount.Type == "volume" {
				if v, ok := cfg.Volumes[mount.Source]; ok && v.Name != "" && !strings.HasPrefix(v.Name, r.Project+"_") {
					return Rendered{}, fmt.Errorf("volume %s has globally shared name %q; remove explicit name", mount.Source, v.Name)
				}
			}
		}
		// Block device passthrough independently of the base host policy flags.
		if all, ok := raw["services"].(map[string]any); ok {
			if service, ok := all[name].(map[string]any); ok {
				if devices, ok := service["devices"].([]any); ok && len(devices) > 0 {
					return Rendered{}, fmt.Errorf("service %s requests device passthrough; remove devices", name)
				}
			}
		}
	}
	// Compose down --volumes acts on every volume in its project definition.
	// Keep only the selected service closure and its reachable resource graph.
	pruneConfig(raw, services)
	data, err = json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return Rendered{}, err
	}
	data = append(data, '\n')
	sum := sha256.Sum256(data)
	return Rendered{JSON: data, Digest: hex.EncodeToString(sum[:]), Services: services}, nil
}

func (c dockerClient) Up(ctx context.Context, r domain.Runtime) error {
	args, err := composeArgs(r)
	if err != nil {
		return err
	}
	if len(r.Services) == 0 {
		return fmt.Errorf("cannot start runtime without selected services")
	}
	if err := c.verifyDeclaredResourceOwnership(ctx, r); err != nil {
		return err
	}
	args = append(args, "up", "-d")
	args = append(args, r.Services...)
	_, err = c.run(ctx, r.Directory, args...)
	return err
}

func (c dockerClient) Logs(ctx context.Context, r domain.Runtime) (string, error) {
	return c.logs(ctx, r, 0)
}
func (c dockerClient) LogsBounded(ctx context.Context, r domain.Runtime) (string, error) {
	return c.logs(ctx, r, 1<<20)
}
func (c dockerClient) logs(ctx context.Context, r domain.Runtime, limit int) (string, error) {
	args, err := composeArgs(r)
	if err != nil {
		return "", err
	}
	args = append(args, "logs", "--no-color", "--timestamps")
	args = append(args, r.Services...)
	// Display callers bound both streams before OSRunner accumulates them.
	// Cleanup retains its existing zero-limit evidence path. Display overflow
	// is explicit, never a successful truncated log result.
	out, err := c.runBounded(ctx, r.Directory, limit, args...)
	if limit > 0 && errors.Is(err, execx.ErrOutputIncomplete) {
		err = fmt.Errorf("Compose logs exceed the 1 MiB per-stream capture limit; query a narrower log selection: %w", err)
	}
	return out.Stdout + out.Stderr, err
}

// GenerateOverride writes service ownership labels; Compose's own project labels
// additionally identify networks and volumes. The caller records the override digest.
func GenerateOverride(r domain.Runtime, leaseID, directory string) (string, error) {
	if err := identity(r); err != nil {
		return "", err
	}
	if leaseID == "" {
		return "", fmt.Errorf("lease ID is required for ownership labels")
	}
	if !filepath.IsAbs(directory) {
		return "", fmt.Errorf("generated directory must be absolute")
	}
	services := map[string]any{}
	for _, s := range r.Services {
		services[s] = map[string]any{"labels": map[string]string{leaseLabel: leaseID, runtimeLabel: r.Name}}
	}
	b, err := json.MarshalIndent(map[string]any{"services": services}, "", "  ")
	if err != nil {
		return "", err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	p := filepath.Join(directory, r.Project+"-labels.json")
	if err := os.WriteFile(p, b, 0600); err != nil {
		return "", err
	}
	return p, nil
}

type containerData struct {
	ID     string `json:"Id"`
	Name   string
	Image  string
	Config struct {
		Image  string
		Labels map[string]string
	}
	State struct {
		Status   string
		Running  bool
		ExitCode int
		Health   *struct{ Status string }
	}
	NetworkSettings struct {
		Ports map[string][]struct{ HostIP, HostPort string }
	}
}
type namedData struct {
	ID     string `json:"Id"`
	Name   string
	Labels map[string]string
}

func ids(s string) []string { values := strings.Fields(s); sort.Strings(values); return values }
func resource(r domain.Runtime, kind, id string, metadata map[string]string) domain.Resource {
	return domain.Resource{ID: r.Name + ":" + kind + ":" + id, Runtime: r.Name, Kind: kind, ExternalID: id, Metadata: metadata}
}
func verifyLabels(r domain.Runtime, labels map[string]string) error {
	if labels[projectLabel] != r.Project {
		return fmt.Errorf("resource identity mismatch: expected project %s, observed %q; quarantine without deleting", r.Project, labels[projectLabel])
	}
	if r.LeaseID != "" {
		if labels[leaseLabel] != r.LeaseID {
			return fmt.Errorf("resource lease ownership mismatch: expected %s, observed %q", r.LeaseID, labels[leaseLabel])
		}
		if labels[runtimeLabel] != r.Name {
			return fmt.Errorf("resource runtime ownership mismatch: expected %s, observed %q", r.Name, labels[runtimeLabel])
		}
	}
	if owner := labels[runtimeLabel]; owner != "" && owner != r.Name {
		return fmt.Errorf("resource runtime ownership mismatch: expected %s, observed %s", r.Name, owner)
	}
	return nil
}

func (c dockerClient) Inspect(ctx context.Context, r domain.Runtime) (Observation, error) {
	o := Observation{Ready: true, Endpoints: map[string]string{}, Resources: []domain.Resource{}, Diagnostics: []string{}}
	if err := identity(r); err != nil {
		return o, err
	}
	base := []string{"--context", r.Context}
	filter := "label=" + projectLabel + "=" + r.Project
	out, err := c.run(ctx, "", append(append([]string{}, base...), "ps", "--all", "--quiet", "--filter", filter)...)
	if err != nil {
		return o, err
	}
	containerIDs := ids(out.Stdout)
	found := map[string]bool{}
	if len(containerIDs) > 0 {
		args := append(append([]string{}, base...), "inspect", "--type", "container")
		args = append(args, containerIDs...)
		out, err = c.run(ctx, "", args...)
		if err != nil {
			return o, err
		}
		var data []containerData
		if err := json.Unmarshal([]byte(out.Stdout), &data); err != nil {
			return o, fmt.Errorf("invalid container inspection JSON: %w", err)
		}
		if len(data) != len(containerIDs) {
			return o, fmt.Errorf("container inspection returned an incomplete resource set")
		}
		for _, v := range data {
			if v.ID == "" {
				return o, fmt.Errorf("container inspection returned an empty identity; quarantine without deleting")
			}
			if err := verifyLabels(r, v.Config.Labels); err != nil {
				return o, err
			}
			service := v.Config.Labels["com.docker.compose.service"]
			if service == "" {
				return o, fmt.Errorf("container %s is missing its Compose service identity", v.ID)
			}
			found[service] = true
			status := v.State.Status
			health := ""
			if v.State.Health != nil {
				health = v.State.Health.Status
			}
			if !v.State.Running || health != "" && health != "healthy" {
				o.Ready = false
				o.Diagnostics = append(o.Diagnostics, fmt.Sprintf("service %s container %s state=%s health=%s", service, v.ID, status, health))
			}
			o.Resources = append(o.Resources, resource(r, "container", v.ID, map[string]string{"project": r.Project, "context": r.Context, "service": service, "image": v.Config.Image, "image_id": v.Image, "status": status, "health": health, "lease": v.Config.Labels[leaseLabel], "runtime": v.Config.Labels[runtimeLabel]}))
			for port, bindings := range v.NetworkSettings.Ports {
				for _, b := range bindings {
					host := b.HostIP
					if host == "" || host == "0.0.0.0" || host == "::" {
						host = "127.0.0.1"
					}
					if strings.Contains(host, ":") {
						host = "[" + host + "]"
					}
					o.Endpoints[service+"/"+port] = host + ":" + b.HostPort
					o.Resources[len(o.Resources)-1].Metadata["endpoint."+port] = host + ":" + b.HostPort
				}
			}
		}
	}
	for _, kind := range []string{"network", "volume"} {
		args := append(append([]string{}, base...), kind, "ls", "--quiet", "--filter", filter)
		out, err = c.run(ctx, "", args...)
		if err != nil {
			return o, err
		}
		resourceIDs := ids(out.Stdout)
		if len(resourceIDs) == 0 {
			continue
		}
		args = append(append([]string{}, base...), kind, "inspect")
		args = append(args, resourceIDs...)
		out, err = c.run(ctx, "", args...)
		if err != nil {
			return o, err
		}
		var data []namedData
		if err := json.Unmarshal([]byte(out.Stdout), &data); err != nil {
			return o, fmt.Errorf("invalid %s inspection JSON: %w", kind, err)
		}
		if len(data) != len(resourceIDs) {
			return o, fmt.Errorf("%s inspection returned an incomplete resource set", kind)
		}
		for _, v := range data {
			if err := verifyLabels(r, v.Labels); err != nil {
				return o, err
			}
			id := v.ID
			if kind == "volume" {
				id = v.Name
			}
			if id == "" {
				return o, fmt.Errorf("%s inspection returned an empty identity", kind)
			}
			o.Resources = append(o.Resources, resource(r, kind, id, map[string]string{"project": r.Project, "context": r.Context, "name": v.Name, "lease": v.Labels[leaseLabel], "runtime": v.Labels[runtimeLabel]}))
		}
	}
	o.Exists = len(o.Resources) > 0
	for _, s := range r.Services {
		if !found[s] {
			o.Ready = false
			o.Diagnostics = append(o.Diagnostics, "selected service "+s+" is missing")
		}
	}
	if len(r.Services) == 0 || len(containerIDs) == 0 {
		o.Ready = false
	}
	sort.Strings(o.Diagnostics)
	return o, nil
}

func (c dockerClient) Down(ctx context.Context, r domain.Runtime) error {
	if _, err := c.Inspect(ctx, r); err != nil {
		return fmt.Errorf("cannot verify project resource identity before cleanup: %w", err)
	}
	args, err := composeArgs(r)
	if err != nil {
		return err
	}
	if err := c.verifyDeclaredResourceOwnership(ctx, r); err != nil {
		return err
	}
	args = append(args, "down", "--volumes", "--remove-orphans")
	if _, err := c.run(ctx, r.Directory, args...); err != nil {
		return err
	}
	o, err := c.Inspect(ctx, r)
	if err != nil {
		return fmt.Errorf("cannot verify cleanup: %w", err)
	}
	if o.Exists {
		return fmt.Errorf("Compose project %s still has resources after down; quarantine remaining resources", r.Project)
	}
	return nil
}

// Project-label discovery cannot see a same-name resource created outside this
// lease. Compose can nevertheless reuse or remove it by its configured name.
// Resolve each name independently and reject unknown ownership before any effect.
func (c dockerClient) verifyDeclaredResourceOwnership(ctx context.Context, r domain.Runtime) error {
	if r.ConfigPath == "" {
		if r.LeaseID != "" {
			return fmt.Errorf("managed runtime requires an immutable configuration snapshot")
		}
		return nil
	}
	data, err := os.ReadFile(r.ConfigPath)
	if err != nil {
		return err
	}
	var config struct {
		Networks map[string]struct {
			Name string `json:"name"`
		} `json:"networks"`
		Volumes map[string]struct {
			Name string `json:"name"`
		} `json:"volumes"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid immutable resource declarations: %w", err)
	}
	for _, kind := range []string{"network", "volume"} {
		resources := config.Networks
		if kind == "volume" {
			resources = config.Volumes
		}
		names := make([]string, 0, len(resources))
		for key, resource := range resources {
			if resource.Name == "" {
				return fmt.Errorf("declared %s %s lacks its resolved name", kind, key)
			}
			names = append(names, resource.Name)
		}
		sort.Strings(names)
		for _, name := range names {
			listing, err := c.run(ctx, "", "--context", r.Context, kind, "ls", "--filter", "name="+name, "--format", "{{json .}}")
			if err != nil {
				return fmt.Errorf("cannot determine declared %s %s ownership: %w", kind, name, err)
			}
			decoder := json.NewDecoder(strings.NewReader(listing.Stdout))
			found := false
			for {
				var listed struct{ Name string }
				if err := decoder.Decode(&listed); err == io.EOF {
					break
				} else if err != nil {
					return fmt.Errorf("invalid %s name listing: %w", kind, err)
				}
				if listed.Name == "" {
					return fmt.Errorf("%s name listing contains an empty identity", kind)
				}
				if listed.Name == name {
					found = true
				}
			}
			if !found {
				continue
			}
			inspection, err := c.run(ctx, "", "--context", r.Context, kind, "inspect", name)
			if err != nil {
				return fmt.Errorf("cannot verify declared %s %s ownership: %w", kind, name, err)
			}
			var records []namedData
			if err := json.Unmarshal([]byte(inspection.Stdout), &records); err != nil {
				return fmt.Errorf("invalid declared %s inspection: %w", kind, err)
			}
			if len(records) != 1 || records[0].Name != name {
				return fmt.Errorf("declared %s %s inspection identity mismatch", kind, name)
			}
			if err := verifyLabels(r, records[0].Labels); err != nil {
				return fmt.Errorf("refuse to touch declared %s %s: %w", kind, name, err)
			}
		}
	}
	return nil
}
