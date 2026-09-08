package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func rejectPodmanDotenv(directory string) error {
	data, err := os.ReadFile(filepath.Join(directory, ".env"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "export "))
		key, _, ok := strings.Cut(line, "=")
		key = strings.ToUpper(strings.Trim(strings.TrimSpace(key), "'\""))
		if ok && (strings.HasPrefix(key, "PODMAN_") || strings.HasPrefix(key, "CONTAINER_") || strings.HasPrefix(key, "AGENT_ENV_PODMAN_") || strings.HasPrefix(key, "COMPOSE_")) {
			return fmt.Errorf("reserved provider environment key %s in .env", key)
		}
	}
	return nil
}
func normalizePodmanInspection(text, project string) (string, error) {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		return "", fmt.Errorf("invalid Podman inspection: %w", err)
	}
	for _, row := range rows {
		labels, ok := row["Labels"].(map[string]any)
		if !ok {
			labels, _ = row["labels"].(map[string]any)
		}
		if config, ok := row["Config"].(map[string]any); ok {
			labels, _ = config["Labels"].(map[string]any)
			nativeService, _ := labels["io.podman.compose.service"].(string)
			compatService, _ := labels["com.docker.compose.service"].(string)
			if nativeService == "" || (compatService != "" && compatService != nativeService) {
				return "", fmt.Errorf("missing or conflicting native Podman service ownership")
			}
			labels["com.docker.compose.service"] = nativeService
		}
		native, _ := labels["io.podman.compose.project"].(string)
		compat, _ := labels["com.docker.compose.project"].(string)
		if compat != "" && compat != native {
			return "", fmt.Errorf("conflicting or missing native Podman project ownership")
		}
		if project != "" && native != project {
			return "", fmt.Errorf("Podman resource lacks expected native project ownership")
		}
		if labels != nil {
			labels["com.docker.compose.project"] = native
		}
		if state, ok := row["State"].(map[string]any); ok {
			if _, exists := state["Health"]; !exists {
				if h, ok := state["Healthcheck"]; ok {
					state["Health"] = h
				}
			}
		}
	}
	b, err := json.Marshal(rows)
	return string(b), err
}
func normalizePodmanConfig(text, directory, project string) (string, error) {
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
		return "", fmt.Errorf("invalid podman-compose YAML: %w", err)
	}
	if err := rejectPodmanExtensions(raw); err != nil {
		return "", err
	}
	for _, kind := range []string{"configs", "secrets"} {
		if declarations, ok := raw[kind].(map[string]any); ok {
			for name, value := range declarations {
				definition, ok := value.(map[string]any)
				if !ok {
					return "", fmt.Errorf("invalid %s declaration %s", kind, name)
				}
				file, ok := definition["file"].(string)
				if !ok || file == "" {
					return "", fmt.Errorf("Podman %s %s requires a local file", kind, name)
				}
				resolved, err := validatedPodmanFile(file, directory)
				if err != nil {
					return "", err
				}
				definition["file"] = resolved
			}
		} else if raw[kind] != nil {
			return "", fmt.Errorf("invalid %s declarations", kind)
		}
	}
	services, ok := raw["services"].(map[string]any)
	if !ok || len(services) == 0 {
		return "", fmt.Errorf("Podman Compose configuration has no services")
	}
	networks, _ := raw["networks"].(map[string]any)
	if networks == nil {
		networks = map[string]any{}
	}
	volumes, _ := raw["volumes"].(map[string]any)
	if volumes == nil {
		volumes = map[string]any{}
	}
	for name, value := range services {
		service, ok := value.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid service %s", name)
		}
		for key := range service {
			if strings.HasPrefix(key, "x-podman") {
				return "", fmt.Errorf("provider extension %s is unsupported", key)
			}
		}
		if value, exists := service["network_mode"]; exists && value != nil {
			mode, ok := value.(string)
			if !ok {
				return "", fmt.Errorf("invalid network_mode")
			}
			switch mode {
			case "", "bridge", "none", "host":
			default:
				return "", fmt.Errorf("unsupported Podman network_mode %q", mode)
			}
		}
		if value, exists := service["volumes"]; exists && value != nil {
			if _, ok := value.([]any); !ok {
				return "", fmt.Errorf("invalid service volumes: expected mount list")
			}
		}
		if value, exists := service["env_file"]; exists && value != nil {
			var files []any
			switch v := value.(type) {
			case string:
				files = []any{v}
			case []any:
				files = v
			default:
				return "", fmt.Errorf("unsupported env_file form; use paths")
			}
			for i, item := range files {
				file, ok := item.(string)
				if !ok {
					return "", fmt.Errorf("unsupported env_file form; use paths")
				}
				resolved, err := validatedPodmanFile(file, directory)
				if err != nil {
					return "", err
				}
				files[i] = resolved
			}
			service["env_file"] = files
		}
		for _, field := range []string{"environment", "labels"} {
			if list, ok := service[field].([]any); ok {
				m := map[string]any{}
				for _, entry := range list {
					s, ok := entry.(string)
					if !ok {
						return "", fmt.Errorf("invalid %s", field)
					}
					k, v, has := strings.Cut(s, "=")
					if has {
						m[k] = v
					} else {
						if field == "environment" {
							return "", fmt.Errorf("unresolved environment pass-through %s is unsupported; provide an explicit value", k)
						}
						m[k] = ""
					}
				}
				service[field] = m
			}
		}
		if environment, ok := service["environment"].(map[string]any); ok {
			for key, value := range environment {
				if value == nil {
					return "", fmt.Errorf("unresolved environment pass-through %s is unsupported; provide an explicit value", key)
				}
			}
		}
		if list, ok := service["depends_on"].([]any); ok {
			m := map[string]any{}
			for _, entry := range list {
				s, ok := entry.(string)
				if !ok {
					return "", fmt.Errorf("invalid depends_on")
				}
				m[s] = map[string]any{"condition": "service_started"}
			}
			service["depends_on"] = m
		}
		switch n := service["networks"].(type) {
		case []any:
			m := map[string]any{}
			for _, entry := range n {
				s, ok := entry.(string)
				if !ok {
					return "", fmt.Errorf("invalid network")
				}
				m[s] = map[string]any{}
			}
			service["networks"] = m
		case nil:
			if service["network_mode"] == nil {
				service["networks"] = map[string]any{"default": map[string]any{}}
				if _, ok := networks["default"]; !ok {
					networks["default"] = map[string]any{}
				}
			}
		}
		if list, ok := service["ports"].([]any); ok {
			for i, entry := range list {
				if _, ok := entry.(map[string]any); ok {
					continue
				}
				port, err := normalizePodmanPort(fmt.Sprint(entry))
				if err != nil {
					return "", err
				}
				list[i] = port
			}
		}
		if list, ok := service["volumes"].([]any); ok {
			for i, entry := range list {
				mount, ok := entry.(map[string]any)
				if !ok {
					s, ok := entry.(string)
					if !ok {
						return "", fmt.Errorf("invalid mount")
					}
					var err error
					mount, err = normalizePodmanMount(s, directory)
					if err != nil {
						return "", err
					}
				}
				kind, ok := mount["type"].(string)
				if !ok {
					return "", fmt.Errorf("invalid mount type")
				}
				switch kind {
				case "bind", "volume", "tmpfs":
				default:
					return "", fmt.Errorf("unsupported Podman mount type %q", kind)
				}
				target, ok := mount["target"].(string)
				if !ok || target == "" {
					return "", fmt.Errorf("mount target missing or invalid")
				}
				if source, exists := mount["source"]; exists && source != nil {
					if _, ok := source.(string); !ok {
						return "", fmt.Errorf("invalid mount source")
					}
				}
				if mount["type"] == "volume" && (mount["source"] == nil || mount["source"] == "") {
					target, _ := mount["target"].(string)
					sum := sha256.Sum256([]byte(name + "\x00" + target))
					key := "agent_env_anon_" + hex.EncodeToString(sum[:8])
					mount["source"] = key
					volumes[key] = map[string]any{"name": project + "_" + key}
				}
				if mount["type"] == "bind" {
					source, _ := mount["source"].(string)
					if source == "" {
						return "", fmt.Errorf("bind source missing")
					}
					if !filepath.IsAbs(source) {
						mount["source"] = filepath.Join(directory, source)
					}
				}
				list[i] = mount
			}
		}
	}
	for _, resources := range []map[string]any{networks, volumes} {
		for key, value := range resources {
			m, ok := value.(map[string]any)
			if value == nil {
				m = map[string]any{}
				ok = true
			}
			if !ok {
				return "", fmt.Errorf("invalid resource %s", key)
			}
			if _, ok := m["name"]; !ok {
				m["name"] = project + "_" + key
			}
			resources[key] = m
		}
	}
	raw["networks"] = networks
	raw["volumes"] = volumes
	b, err := json.Marshal(raw)
	return string(b), err
}
func normalizePodmanPort(value string) (map[string]any, error) {
	address, protocol, ok := strings.Cut(value, "/")
	if !ok {
		protocol = "tcp"
	}
	parts := strings.Split(address, ":")
	if len(parts) > 3 {
		return nil, fmt.Errorf("unsupported ambiguous port %q; use long syntax", value)
	}
	target, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || target < 1 || target > 65535 {
		return nil, fmt.Errorf("invalid port %q", value)
	}
	p := map[string]any{"target": target, "protocol": protocol}
	if len(parts) > 1 && parts[len(parts)-2] != "" {
		p["published"] = parts[len(parts)-2]
	}
	if len(parts) == 3 {
		p["host_ip"] = parts[0]
	}
	return p, nil
}
func normalizePodmanMount(value, directory string) (map[string]any, error) {
	parts := strings.Split(value, ":")
	if len(parts) > 1 && len(parts[0]) == 1 && strings.HasPrefix(parts[1], "\\") {
		parts = append([]string{parts[0] + ":" + parts[1]}, parts[2:]...)
	}
	if len(parts) > 3 {
		return nil, fmt.Errorf("ambiguous mount %q; use long syntax", value)
	}
	m := map[string]any{"type": "volume"}
	switch len(parts) {
	case 1:
		m["target"] = parts[0]
	case 2, 3:
		source := parts[0]
		m["source"] = source
		m["target"] = parts[1]
		if strings.HasPrefix(source, ".") || filepath.IsAbs(source) {
			m["type"] = "bind"
			if !filepath.IsAbs(source) {
				m["source"] = filepath.Join(directory, source)
			}
		}
		if len(parts) == 3 {
			for _, option := range strings.Split(parts[2], ",") {
				switch option {
				case "ro":
					m["read_only"] = true
				case "rw":
				case "z", "Z":
					m["bind"] = map[string]any{"selinux": option}
				default:
					return nil, fmt.Errorf("unsupported mount option %q", option)
				}
			}
		}
	}
	return m, nil
}

func escapePodmanSnapshot(data []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	// The common snapshot represents OS-assigned ports as published zero.
	// Podman requires an omitted host port; preserve host_ip in the private copy.
	if root, ok := value.(map[string]any); ok {
		if services, ok := root["services"].(map[string]any); ok {
			for _, entry := range services {
				if service, ok := entry.(map[string]any); ok {
					if ports, ok := service["ports"].([]any); ok {
						for _, entry := range ports {
							if port, ok := entry.(map[string]any); ok {
								switch published := port["published"].(type) {
								case string:
									if published == "0" {
										delete(port, "published")
									}
								case float64:
									if published == 0 {
										delete(port, "published")
									}
								}
							}
						}
					}
				}
			}
		}
	}
	var escape func(any) any
	escape = func(value any) any {
		switch v := value.(type) {
		case string:
			return strings.ReplaceAll(v, "$", "$$")
		case []any:
			for i := range v {
				v[i] = escape(v[i])
			}
			return v
		case map[string]any:
			for k, entry := range v {
				v[k] = escape(entry)
			}
			return v
		default:
			return value
		}
	}
	return json.Marshal(escape(value))
}

func rejectPodmanExtensions(value any) error {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if strings.HasPrefix(strings.ToLower(key), "x-podman") {
				return fmt.Errorf("provider execution extension %s is unsupported", key)
			}
			if err := rejectPodmanExtensions(child); err != nil {
				return err
			}
		}
	case map[any]any:
		for key, child := range v {
			if text, ok := key.(string); ok && strings.HasPrefix(strings.ToLower(text), "x-podman") {
				return fmt.Errorf("provider execution extension %s is unsupported", text)
			}
			if err := rejectPodmanExtensions(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if err := rejectPodmanExtensions(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatedPodmanFile(path, directory string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty Compose file reference")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(directory, path)
	}
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", fmt.Errorf("resolve Compose directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve Compose file reference: %w", err)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("Compose file reference must remain inside project_directory")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Compose file reference must be a regular file")
	}
	return resolved, nil
}
