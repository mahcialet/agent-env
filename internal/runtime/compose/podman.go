package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
)

type podmanClient struct {
	Runner execx.Runner
	Policy policy.Policy
}

func (c podmanClient) runner() execx.Runner {
	if c.Runner != nil {
		return c.Runner
	}
	return execx.OSRunner{}
}
func (c podmanClient) command(ctx context.Context, p podmanIdentity, args ...string) (execx.Result, error) {
	result, err := c.runner().Run(ctx, execx.Command{Name: p.Executable, Args: append(p.flags(), args...), UnsetEnv: podmanUnsetEnvironment(), Timeout: 2 * time.Minute})
	if err != nil {
		return result, fmt.Errorf("Podman command failed: %w", err)
	}
	if result.ExitCode != 0 {
		return result, fmt.Errorf("Podman command failed (%d): %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return result, nil
}
func (c podmanClient) fingerprint(ctx context.Context, p podmanIdentity) (string, string, error) {
	out, err := c.command(ctx, p, "info", "--format", "json")
	if err != nil {
		return "", "", err
	}
	var info struct {
		Host    struct{ Hostname, OS, Arch string }
		Store   struct{ GraphRoot, RunRoot, VolumePath string }
		Version struct{ Version string }
	}
	if err = json.Unmarshal([]byte(out.Stdout), &info); err != nil {
		return "", "", fmt.Errorf("invalid Podman info: %w", err)
	}
	if info.Host.Hostname == "" || info.Host.OS == "" || info.Host.Arch == "" || info.Store.GraphRoot == "" || info.Store.RunRoot == "" || info.Store.VolumePath == "" || !strings.HasPrefix(info.Version.Version, "5.") {
		return "", "", fmt.Errorf("Podman 5.x and complete engine/store identity are required")
	}
	b, _ := json.Marshal(struct {
		Host  any
		Store any
	}{info.Host, info.Store})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), info.Version.Version, nil
}
func (c podmanClient) InventoryDoctor(ctx context.Context) (map[string]string, error) {
	result := map[string]string{"provider": "podman-compose"}
	executable, err := exec.LookPath("podman")
	if err != nil {
		return result, fmt.Errorf("Podman prerequisite missing: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return result, err
	}
	composePath, lookupErr := exec.LookPath("podman-compose")
	if lookupErr == nil {
		composePath, err = filepath.Abs(composePath)
		if err != nil {
			return result, err
		}
	} else {
		composePath = ""
	}
	p := podmanIdentity{Version: 1, Executable: executable, ComposeExecutable: composePath}
	// Explicit ambient selection is resolved once; later commands use the snapshot.
	connection := os.Getenv("CONTAINER_CONNECTION")
	p.URL = os.Getenv("CONTAINER_HOST")
	p.Identity = os.Getenv("CONTAINER_SSHKEY")
	if connection != "" || (p.URL == "" && runtime.GOOS != "linux") {
		listing, e := c.command(ctx, p, "system", "connection", "list", "--format", "json")
		if e != nil {
			return result, e
		}
		var connections []struct {
			Name, URI, Identity string
			Default             bool
		}
		if e = json.Unmarshal([]byte(listing.Stdout), &connections); e != nil {
			return result, fmt.Errorf("invalid Podman connections")
		}
		found := false
		for _, item := range connections {
			if (connection != "" && item.Name == connection) || (connection == "" && item.Default) {
				if found {
					return result, fmt.Errorf("ambiguous Podman connection")
				}
				p.URL = item.URI
				p.Identity = item.Identity
				found = true
			}
		}
		if !found {
			return result, fmt.Errorf("selected Podman connection unavailable; start and configure a machine connection")
		}
	}
	p.Fingerprint = "pending"
	if _, err = decodePodmanIdentity(p.encode()); err != nil {
		return result, err
	}
	p.Fingerprint, result["podman_version"], err = c.fingerprint(ctx, p)
	if err != nil {
		return result, err
	}
	result["context"] = p.encode()
	result["engine_fingerprint"] = p.Fingerprint
	return result, nil
}
func (c podmanClient) Doctor(ctx context.Context) (map[string]string, error) {
	result, err := c.InventoryDoctor(ctx)
	if err != nil {
		return result, err
	}
	p, err := decodePodmanIdentity(result["context"])
	if err != nil {
		return result, err
	}
	if p.ComposeExecutable == "" {
		return result, fmt.Errorf("standalone podman-compose prerequisite missing")
	}
	bridge, err := os.Executable()
	if err != nil {
		return result, err
	}
	version, err := c.runner().Run(ctx, execx.Command{Name: p.ComposeExecutable, Args: []string{"--podman-path", bridge, "--version"}, Env: map[string]string{podmanBridgeEnv: p.encode()}, UnsetEnv: podmanUnsetEnvironment(), Timeout: 30 * time.Second})
	if err != nil || version.ExitCode != 0 {
		return result, fmt.Errorf("standalone podman-compose version probe failed (exit %d): %v: %s", version.ExitCode, err, strings.TrimSpace(version.Stderr))
	}
	var major, minor, patch int
	text := strings.TrimSpace(version.Stdout)
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "podman-compose version ") {
			text = strings.TrimPrefix(line, "podman-compose version ")
			break
		}
	}
	if _, err = fmt.Sscanf(text, "%d.%d.%d", &major, &minor, &patch); err != nil || major != 1 || minor < 6 {
		return result, fmt.Errorf("podman-compose >=1.6.0,<2.0.0 required (found %q)", text)
	}
	nativeVersion, err := c.command(ctx, p, "--version")
	if err != nil {
		return result, err
	}
	result["client_version"] = strings.TrimSpace(nativeVersion.Stdout)
	result["server_version"] = result["podman_version"]
	hostInfo, err := c.command(ctx, p, "info", "--format", "json")
	if err != nil {
		return result, err
	}
	var details struct {
		Host struct {
			ServiceIsRemote bool
			Security        struct{ Rootless bool }
		}
	}
	if err = json.Unmarshal([]byte(hostInfo.Stdout), &details); err != nil {
		return result, err
	}
	result["mode"] = "local"
	if p.URL != "" {
		result["mode"] = "remote"
	}
	result["rootless"] = fmt.Sprint(details.Host.Security.Rootless)
	result["compose_version"] = text
	result["context"] = p.encode()
	result["engine_fingerprint"] = p.Fingerprint
	return result, nil
}
func (c podmanClient) checked(ctx context.Context, r domain.Runtime) (dockerClient, error) {
	p, err := decodePodmanIdentity(r.Context)
	if err != nil {
		return dockerClient{}, err
	}
	fingerprint, _, err := c.fingerprint(ctx, p)
	if err != nil {
		return dockerClient{}, err
	}
	if fingerprint != p.Fingerprint {
		return dockerClient{}, fmt.Errorf("Podman engine fingerprint changed; refusing resource operations")
	}
	return dockerClient{Runner: podmanAdapter{client: c, identity: p, project: r.Project}, Policy: c.Policy}, nil
}
func (c podmanClient) Render(ctx context.Context, r domain.Runtime, roots []string) (Rendered, error) {
	d, err := c.checked(ctx, r)
	if err != nil {
		return Rendered{}, err
	}
	return d.Render(ctx, r, roots)
}
func (c podmanClient) Up(ctx context.Context, r domain.Runtime) error {
	d, err := c.checked(ctx, r)
	if err != nil {
		return err
	}
	err = d.Up(ctx, r)
	var nativeError *podmanCommandError
	if errors.As(err, &nativeError) {
		return nativeError
	}
	return err
}
func (c podmanClient) Down(ctx context.Context, r domain.Runtime) error {
	d, err := c.checked(ctx, r)
	if err != nil {
		return err
	}
	prepared, err := c.PrepareCleanup(ctx, r)
	if err != nil {
		return err
	}
	recorded := map[string]string{}
	for _, proof := range r.CleanupEvidence {
		recorded[proof.ExternalID] = proof.Metadata["volume_fingerprint"]
	}
	for _, proof := range prepared.CleanupEvidence {
		if recorded[proof.ExternalID] != proof.Metadata["volume_fingerprint"] {
			return fmt.Errorf("anonymous-volume cleanup evidence must be persisted before down")
		}
	}
	if err = d.Down(ctx, r); err != nil {
		return err
	}
	return c.removeAnonymousVolumes(ctx, r, r.CleanupEvidence)
}

func (c podmanClient) Logs(ctx context.Context, r domain.Runtime) (string, error) {
	d, err := c.checked(ctx, r)
	if err != nil {
		return "", err
	}
	return d.Logs(ctx, r)
}
func (c podmanClient) Inspect(ctx context.Context, r domain.Runtime) (Observation, error) {
	d, err := c.checked(ctx, r)
	if err != nil {
		return Observation{}, err
	}
	observation, err := d.Inspect(ctx, r)
	if err != nil {
		return observation, err
	}
	anonymous, err := c.anonymousVolumes(ctx, r, observation)
	observation.Resources = append(observation.Resources, anonymous...)
	if err != nil {
		return observation, err
	}
	retained, err := c.retainedAnonymousVolumes(ctx, r, observation.Resources)
	observation.Resources = append(observation.Resources, retained...)
	observation.Exists = len(observation.Resources) > 0
	if err == nil {
		p, _ := decodePodmanIdentity(r.Context)
		if p.URL != "" {
			for service, endpoint := range observation.Endpoints {
				probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				connection, probeErr := (&net.Dialer{}).DialContext(probeCtx, "tcp", endpoint)
				cancel()
				if probeErr != nil {
					observation.Ready = false
					observation.Diagnostics = append(observation.Diagnostics, "remote Podman endpoint is not reachable from this host: "+service)
					delete(observation.Endpoints, service)
					continue
				}
				_ = connection.Close()
			}
		}
	}
	return observation, err
}
func (c podmanClient) Inventory(ctx context.Context, identity string) ([]domain.Resource, error) {
	d, err := c.checked(ctx, domain.Runtime{Context: identity})
	if err != nil {
		return nil, err
	}
	resources, err := d.Inventory(ctx, identity)
	for i := range resources {
		resources[i].Metadata["provider"] = "podman-compose"
	}
	return resources, err
}

// The adapter shares resource traversal and ownership checks, never executables.
// Every outgoing request is translated into native pinned Podman arguments.
type podmanAdapter struct {
	client   podmanClient
	identity podmanIdentity
	project  string
}

func (a podmanAdapter) Run(ctx context.Context, q execx.Command) (execx.Result, error) {
	args := q.Args
	if len(args) < 3 || args[0] != "--context" || args[1] != a.identity.encode() {
		return execx.Result{}, fmt.Errorf("invalid Podman runtime request")
	}
	args = append([]string(nil), args[2:]...)
	if args[0] == "compose" {
		return a.compose(ctx, q, args[1:])
	}
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "com.docker.compose.project=", "io.podman.compose.project=")
	}
	q.Name = a.identity.Executable
	q.Args = append(a.identity.flags(), args...)
	q.Env = nil
	q.UnsetEnv = podmanUnsetEnvironment()
	out, err := a.client.runner().Run(ctx, q)
	if err != nil || out.ExitCode != 0 {
		return out, err
	}
	if args[0] == "inspect" || len(args) > 1 && args[1] == "inspect" {
		out.Stdout, err = normalizePodmanInspection(out.Stdout, a.project)
	}
	return out, err
}
func (a podmanAdapter) compose(ctx context.Context, q execx.Command, args []string) (execx.Result, error) {
	if a.identity.ComposeExecutable == "" {
		return execx.Result{}, fmt.Errorf("recorded identity lacks standalone podman-compose executable")
	}
	executable, err := os.Executable()
	if err != nil {
		return execx.Result{}, err
	}
	directory := q.Dir
	// podman-compose uses the first file directory, not --project-directory.
	var translated []string
	config := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--project-directory" {
			i++
			directory = args[i]
			continue
		}
		if arg == "--format" {
			i++
			continue
		}
		if arg == "config" {
			config = true
		}
		translated = append(translated, arg)
	}
	if config {
		for i := 0; i < len(translated)-1; i++ {
			if translated[i] == "-f" {
				if filepath.Clean(filepath.Dir(translated[i+1])) != filepath.Clean(directory) {
					return execx.Result{}, fmt.Errorf("Podman initial Compose file must share project_directory; differing first-file directory is unsupported")
				}
				break
			}
		}
	}
	// A validated JSON snapshot contains already-interpolated literal strings.
	// Escape dollars before the provider reparses it; never reread mutable sources.
	if !config {
		for i := 0; i < len(translated)-1; i++ {
			if translated[i] != "-f" {
				continue
			}
			data, readErr := os.ReadFile(translated[i+1])
			if readErr != nil {
				return execx.Result{}, readErr
			}
			if !json.Valid(data) {
				return execx.Result{}, fmt.Errorf("Podman mutations require a validated canonical JSON snapshot")
			}
			escaped, escapeErr := escapePodmanSnapshot(data)
			if escapeErr != nil {
				return execx.Result{}, escapeErr
			}
			tempDir, tempErr := os.MkdirTemp("", "agent-env-podman-config-")
			if tempErr != nil {
				return execx.Result{}, tempErr
			}
			defer os.RemoveAll(tempDir)
			path := filepath.Join(tempDir, "compose.json")
			if writeErr := os.WriteFile(path, escaped, 0600); writeErr != nil {
				return execx.Result{}, writeErr
			}
			translated[i+1] = path
		}
	}
	if err = rejectPodmanDotenv(directory); err != nil {
		return execx.Result{}, err
	}
	q.Name = a.identity.ComposeExecutable
	envFile := filepath.Join(directory, ".env")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		envFile = os.DevNull
	} else if err != nil {
		return execx.Result{}, err
	}
	q.Args = append([]string{"--podman-path", executable, "--in-pod=false", "--env-file", envFile}, translated...)
	q.Dir = directory
	q.UnsetEnv = podmanUnsetEnvironment()
	q.Env = map[string]string{podmanBridgeEnv: a.identity.encode()}
	if !config {
		fingerprint, _, e := a.client.fingerprint(ctx, a.identity)
		if e != nil {
			return execx.Result{}, e
		}
		if fingerprint != a.identity.Fingerprint {
			return execx.Result{}, fmt.Errorf("Podman engine fingerprint changed before Compose operation")
		}
	}
	out, err := a.client.runner().Run(ctx, q)
	if err != nil || out.ExitCode != 0 {
		secrets := evidence.InheritedSecrets()
		for i := 0; i < len(translated)-1; i++ {
			if translated[i] == "-f" {
				if data, readErr := os.ReadFile(translated[i+1]); readErr == nil {
					var config map[string]any
					if json.Unmarshal(data, &config) == nil {
						if services, ok := config["services"].(map[string]any); ok {
							for _, value := range services {
								if service, ok := value.(map[string]any); ok {
									if environment, ok := service["environment"].(map[string]any); ok {
										env := map[string]string{}
										for key, value := range environment {
											if text, ok := value.(string); ok {
												env[key] = text
												env[key+"_UNESCAPED"] = strings.ReplaceAll(text, "$$", "$")
											}
										}
										secrets = append(secrets, evidence.Secrets(env)...)
									}
								}
							}
						}
					}
				}
			}
		}
		detail := evidence.RedactString(strings.TrimSpace(out.Stderr), secrets)
		if len(detail) > 8192 {
			detail = detail[len(detail)-8192:]
		}
		return out, &podmanCommandError{exitCode: out.ExitCode, cause: err, detail: detail}
	}
	if config {
		out.Stdout, err = normalizePodmanConfig(out.Stdout, directory, a.project)
	}
	return out, err
}

type podmanCommandError struct {
	exitCode int
	cause    error
	detail   string
}

func (e *podmanCommandError) Error() string {
	message := fmt.Sprintf("podman-compose failed (exit %d)", e.exitCode)
	if e.cause != nil {
		message += ": " + e.cause.Error()
	}
	if e.detail != "" {
		message += ": " + e.detail
	}
	return message
}
func (e *podmanCommandError) Unwrap() error { return e.cause }
