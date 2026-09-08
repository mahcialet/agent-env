package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Browser explicitly adds browser behavior to an otherwise generic process.
type Browser struct {
	Type    string `yaml:"type" json:"type"`
	Runtime string `yaml:"runtime" json:"runtime"`
	CDPPort string `yaml:"cdp_port" json:"cdp_port"`
}

func validateBrowserPresence(data []byte) error {
	var fields map[string]any
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return err
	}
	if value, exists := fields["browsers"]; exists {
		browsers, ok := value.(map[string]any)
		if !ok || len(browsers) == 0 {
			return fmt.Errorf("manifest: browsers must be a nonempty mapping when declared")
		}
	}
	return nil
}

func validateBrowsers(m *Manifest) error {
	if err := names("browser", m.Browsers); err != nil {
		return err
	}
	folded, bound := map[string]string{}, map[string]string{}
	for _, name := range keys(m.Browsers) {
		if old, ok := folded[strings.ToLower(name)]; ok {
			return fmt.Errorf("manifest: browser names %q and %q collide under case folding", old, name)
		}
		folded[strings.ToLower(name)] = name
		base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
		reserved := base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
		if reserved || strings.HasSuffix(name, ".") {
			return fmt.Errorf("manifest: browser name %q is not portable", name)
		}
		b := m.Browsers[name]
		if b.Type != "chromium-cdp" {
			return fmt.Errorf("manifest: browser %s requires type chromium-cdp", name)
		}
		r, ok := m.Runtimes[b.Runtime]
		if !ok || r.Type != "process" {
			return fmt.Errorf("manifest: browser %s must reference a process runtime", name)
		}
		if old, ok := bound[b.Runtime]; ok {
			return fmt.Errorf("manifest: browsers %s and %s bind the same runtime", old, name)
		}
		bound[b.Runtime] = name
		port, ok := r.Ports[b.CDPPort]
		if !ok || port.Protocol != "tcp" {
			return fmt.Errorf("manifest: browser %s requires a declared TCP cdp_port", name)
		}
		if err := validateBrowserCommand(r.Command, b.CDPPort); err != nil {
			return fmt.Errorf("manifest: browser %s: %w", name, err)
		}
	}
	return nil
}

func validateBrowserCommand(argv []string, port string) error {
	required := map[string]string{
		"headless":                 "--headless=new",
		"enable-automation":        "--enable-automation",
		"user-data-dir":            "--user-data-dir=${runtime_dir}/profile",
		"remote-debugging-address": "--remote-debugging-address=127.0.0.1",
		"remote-debugging-port":    "--remote-debugging-port=${port:" + port + "}",
	}
	seen := map[string]bool{}
	if len(argv) == 0 {
		return fmt.Errorf("browser command is empty")
	}
	for _, arg := range argv[1:] {
		// Chromium accepts alternate switch prefixes on some hosts. Reject them for
		// protected switches, rather than letting native parsing override our proof.
		if arg == "--" {
			return fmt.Errorf("browser command cannot contain the -- switch terminator")
		}
		if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "/") {
			continue
		}
		key := strings.ToLower(strings.SplitN(strings.TrimLeft(arg, "-/"), "=", 2)[0])
		if want, protected := required[key]; protected {
			if arg != want || seen[key] {
				return fmt.Errorf("browser command requires exactly one %s", want)
			}
			seen[key] = true
		}
		switch key {
		case "remote-debugging-pipe", "profile-directory", "guest", "incognito", "disable-automation":
			return fmt.Errorf("browser command cannot override managed browser settings with %s", key)
		}
	}
	for _, key := range keys(required) {
		if !seen[key] {
			return fmt.Errorf("browser command requires %s", required[key])
		}
	}
	return nil
}
