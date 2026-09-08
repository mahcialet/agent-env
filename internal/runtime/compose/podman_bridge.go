package compose

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const podmanBridgeEnv = "AGENT_ENV_PODMAN_BRIDGE_V1"

type podmanIdentity struct {
	Version           int    `json:"version"`
	Executable        string `json:"executable"`
	ComposeExecutable string `json:"compose_executable"`
	URL               string `json:"url,omitempty"`
	Identity          string `json:"identity,omitempty"`
	Fingerprint       string `json:"fingerprint"`
}

func (p podmanIdentity) encode() string {
	b, _ := json.Marshal(p)
	return "podman-v1:" + base64.RawURLEncoding.EncodeToString(b)
}
func decodePodmanIdentity(value string) (podmanIdentity, error) {
	var p podmanIdentity
	if !strings.HasPrefix(value, "podman-v1:") {
		return p, fmt.Errorf("missing recorded Podman connection identity; run doctor")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "podman-v1:"))
	if err != nil {
		return p, fmt.Errorf("invalid Podman identity")
	}
	if err = json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("invalid Podman identity")
	}
	if p.Version != 1 || !filepath.IsAbs(p.Executable) || (p.ComposeExecutable != "" && !filepath.IsAbs(p.ComposeExecutable)) || p.Fingerprint == "" || strings.ContainsAny(p.Executable+p.URL+p.Identity, "\x00\r\n") {
		return p, fmt.Errorf("invalid recorded Podman identity")
	}
	if p.URL != "" && !strings.HasPrefix(p.URL, "ssh://") && !strings.HasPrefix(p.URL, "unix://") && !strings.HasPrefix(p.URL, "tcp://") {
		return p, fmt.Errorf("unsupported Podman endpoint")
	}
	if p.URL != "" {
		u, err := url.Parse(p.URL)
		if err != nil {
			return p, fmt.Errorf("invalid Podman endpoint")
		}
		if u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
			return p, fmt.Errorf("Podman endpoint must not contain query, fragment or opaque data")
		}
		if u.Scheme != "unix" && u.Host == "" {
			return p, fmt.Errorf("Podman endpoint host missing")
		}
		if u.User != nil {
			if _, present := u.User.Password(); present {
				return p, fmt.Errorf("Podman endpoint must not embed credentials")
			}
		}
	}
	if p.Identity != "" && (p.URL == "" || !filepath.IsAbs(p.Identity)) {
		return p, fmt.Errorf("invalid Podman SSH identity")
	}
	return p, nil
}
func (p podmanIdentity) flags() []string {
	if p.URL == "" {
		return []string{"--remote=false"}
	}
	a := []string{"--remote=true", "--url", p.URL}
	if p.Identity != "" {
		a = append(a, "--identity", p.Identity)
	}
	return a
}
func podmanEnvironment() []string {
	var result []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		key = strings.ToUpper(key)
		if key == podmanBridgeEnv || strings.HasPrefix(key, "CONTAINER_") || strings.HasPrefix(key, "PODMAN_") || strings.HasPrefix(key, "DOCKER_") || strings.HasPrefix(key, "COMPOSE_") {
			continue
		}
		result = append(result, e)
	}
	return result
}

// RunPodmanBridge provides a native executable trampoline for podman-compose.
// The provider is selected before the subcommand, including on Windows.
func RunPodmanBridge(args []string) (bool, int) {
	value := os.Getenv(podmanBridgeEnv)
	if value == "" {
		return false, 0
	}
	p, err := decodePodmanIdentity(value)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 125
	}
	if len(args) > 0 && args[0] == "--version" {
		onlyVersion := true
		for _, arg := range args[1:] {
			if arg != "" {
				onlyVersion = false
			}
		}
		if onlyVersion {
			args = []string{"--version"}
		}
	}
	if len(args) == 0 || (strings.HasPrefix(args[0], "-") && !(len(args) == 1 && args[0] == "--version")) {
		fmt.Fprintln(os.Stderr, "Podman bridge requires a subcommand")
		return true, 125
	}
	command := exec.Command(p.Executable, append(p.flags(), args...)...)
	command.Env = podmanEnvironment()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err = command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return true, exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return true, 125
	}
	return true, 0
}

func podmanUnsetEnvironment() []string {
	keep := map[string]bool{}
	for _, e := range podmanEnvironment() {
		k, _, _ := strings.Cut(e, "=")
		keep[k] = true
	}
	var unset []string
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		if !keep[k] {
			unset = append(unset, k)
		}
	}
	return unset
}
