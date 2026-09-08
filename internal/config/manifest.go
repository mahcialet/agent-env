// Package config decodes the explicit repository environment contract.
package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Applications map[string]Application `yaml:"applications,omitempty" json:"applications,omitempty"`
	Version      int                    `yaml:"version" json:"version"`
	Sources      map[string]Source      `yaml:"sources" json:"sources"`
	Runtimes     map[string]Runtime     `yaml:"runtimes" json:"runtimes"`
	Components   map[string]Component   `yaml:"components" json:"components"`
	Stacks       map[string]Stack       `yaml:"stacks" json:"stacks"`
	Tests        map[string]Test        `yaml:"tests,omitempty" json:"tests,omitempty"`
}
type Source struct {
	Repository string `yaml:"repository" json:"repository"`
	DefaultRef string `yaml:"default_ref" json:"default_ref"`
	Writable   bool   `yaml:"writable,omitempty" json:"writable,omitempty"`
}
type Runtime struct {
	Type             string   `yaml:"type" json:"type"`
	Provider         string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Source           string   `yaml:"source" json:"source"`
	ProjectDirectory string   `yaml:"project_directory" json:"project_directory"`
	Files            []string `yaml:"files" json:"files"`
	AVD              string   `yaml:"avd,omitempty" json:"avd,omitempty"`
}
type Component struct {
	Application     string              `yaml:"application,omitempty" json:"application,omitempty"`
	Runtime         string              `yaml:"runtime" json:"runtime"`
	ComposeServices []string            `yaml:"compose_services" json:"compose_services"`
	DependsOn       []string            `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Provides        []string            `yaml:"provides,omitempty" json:"provides,omitempty"`
	Readiness       []Probe             `yaml:"readiness,omitempty" json:"readiness,omitempty"`
	Endpoints       map[string]Endpoint `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`
}
type Stack struct {
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Roots       []string `yaml:"roots" json:"roots"`
}
type Test struct {
	Stack            string            `yaml:"stack" json:"stack"`
	Source           string            `yaml:"source" json:"source"`
	WorkingDirectory string            `yaml:"working_directory" json:"working_directory"`
	Command          []string          `yaml:"command" json:"command"`
	Env              map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Timeout          string            `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Artifacts        []string          `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
}
type Probe struct {
	Type             string   `yaml:"type" json:"type"`
	URL              string   `yaml:"url,omitempty" json:"url,omitempty"`
	Source           string   `yaml:"source,omitempty" json:"source,omitempty"`
	WorkingDirectory string   `yaml:"working_directory,omitempty" json:"working_directory,omitempty"`
	Timeout          string   `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Interval         string   `yaml:"interval,omitempty" json:"interval,omitempty"`
	Command          []string `yaml:"command,omitempty" json:"command,omitempty"`
}
type Endpoint struct {
	Service  string `yaml:"service" json:"service"`
	Target   int    `yaml:"target" json:"target"`
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// Load reads either a repository directory or an explicit manifest file.
func Load(repository string) (*Manifest, error) {
	if repository == "" {
		repository = "."
	}
	p := repository
	info, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("manifest: locate %s: %w", p, err)
	}
	if info.IsDir() {
		p = filepath.Join(p, ".agent-env.yaml")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %s: %w", p, err)
	}
	return Parse(b)
}

func Parse(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("manifest: invalid YAML: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("manifest: use exactly one YAML document")
	}
	// A string's zero value preserves old canonical snapshots, but explicit
	// empty/null providers are configuration errors rather than defaults.
	var fields struct {
		Runtimes map[string]map[string]any `yaml:"runtimes"`
	}
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("manifest: invalid YAML: %w", err)
	}
	for _, name := range keys(fields.Runtimes) {
		if value, present := fields.Runtimes[name]["provider"]; present {
			provider, ok := value.(string)
			if !ok || provider == "" {
				return nil, fmt.Errorf("manifest: runtime %s provider must be docker-compose or podman-compose", name)
			}
		}
	}
	if err := Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func CanonicalJSON(m *Manifest) ([]byte, error) { return json.Marshal(m) }
func Digest(m *Manifest) string {
	b, _ := CanonicalJSON(m)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
var drivePattern = regexp.MustCompile(`^[A-Za-z]:`)

func keys[T any](m map[string]T) []string {
	r := make([]string, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}
func names[T any](kind string, m map[string]T) error {
	for _, n := range keys(m) {
		if !namePattern.MatchString(n) {
			return fmt.Errorf("manifest: invalid %s name %q; use letters, digits, dots, underscores or hyphens", kind, n)
		}
	}
	return nil
}
func distinct(field string, values []string, required bool) error {
	if required && len(values) == 0 {
		return fmt.Errorf("manifest: %s must not be empty", field)
	}
	seen := map[string]bool{}
	for _, v := range values {
		if strings.TrimSpace(v) == "" || strings.ContainsRune(v, 0) {
			return fmt.Errorf("manifest: %s contains an empty or invalid value", field)
		}
		if seen[v] {
			return fmt.Errorf("manifest: %s duplicates %q", field, v)
		}
		seen[v] = true
	}
	return nil
}

// RelativePath validates paths that must remain within a materialized source.
// Both separator styles are checked, even when parsing on the other OS.
func RelativePath(p string) error {
	if strings.ContainsRune(p, 0) || filepath.IsAbs(p) || drivePattern.MatchString(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return fmt.Errorf("path %q must be source-relative", p)
	}
	if strings.Contains(p, `\`) {
		return fmt.Errorf("path %q uses nonportable separators; use forward slashes in manifests", p)
	}
	depth := 0
	for _, part := range strings.Split(p, "/") {
		switch part {
		case "", ".":
		case "..":
			depth--
		default:
			depth++
		}
		if depth < 0 {
			return fmt.Errorf("path %q escapes its declared source root", p)
		}
	}
	return nil
}
func duration(field, value string) error {
	if value == "" {
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return fmt.Errorf("manifest: %s must be a positive duration", field)
	}
	return nil
}
func command(field string, argv []string) error {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return fmt.Errorf("manifest: %s requires a nonempty argv command", field)
	}
	for _, v := range argv {
		if strings.ContainsRune(v, 0) {
			return fmt.Errorf("manifest: %s contains a NUL argument", field)
		}
	}
	return nil
}

func Validate(m *Manifest) error {
	if m == nil || m.Version != 1 {
		return fmt.Errorf("manifest: unsupported version; set version: 1")
	}
	if len(m.Sources) == 0 || len(m.Runtimes) == 0 || len(m.Components) == 0 || len(m.Stacks) == 0 {
		return fmt.Errorf("manifest: sources, runtimes, components and stacks must be nonempty")
	}
	for _, err := range []error{names("source", m.Sources), names("runtime", m.Runtimes), names("component", m.Components), names("stack", m.Stacks), names("test", m.Tests), names("application", m.Applications)} {
		if err != nil {
			return err
		}
	}
	for _, n := range keys(m.Sources) {
		s := m.Sources[n]
		if strings.TrimSpace(s.Repository) == "" || strings.ContainsRune(s.Repository, 0) {
			return fmt.Errorf("manifest: source %s requires a local repository path", n)
		}
		if strings.Contains(s.Repository, "://") || strings.HasPrefix(s.Repository, "git@") {
			return fmt.Errorf("manifest: source %s must use a local repository, remote URLs are unsupported", n)
		}
		if strings.Contains(s.Repository, `\`) && strings.Contains(s.Repository, "/") {
			return fmt.Errorf("manifest: source %s mixes path separator styles", n)
		}
		if drivePattern.MatchString(s.Repository) && filepath.VolumeName(s.Repository) == "" {
			return fmt.Errorf("manifest: source %s uses a Windows drive path on a non-Windows host", n)
		}
		if strings.HasPrefix(s.DefaultRef, "-") || strings.ContainsRune(s.DefaultRef, 0) {
			return fmt.Errorf("manifest: source %s has an invalid default_ref", n)
		}
	}
	androidNames := map[string]string{}
	for _, n := range keys(m.Runtimes) {
		r := m.Runtimes[n]
		if r.Type != "compose" && r.Type != "android-emulator" {
			return fmt.Errorf("manifest: runtime %s type %q is unsupported; use compose or android-emulator", n, r.Type)
		}
		if _, ok := m.Sources[r.Source]; !ok {
			return fmt.Errorf("manifest: runtime %s references unknown source %q", n, r.Source)
		}
		if r.Provider != "" && r.Type != "compose" {
			return fmt.Errorf("manifest: runtime %s provider is only supported for compose", n)
		}
		if r.Type == "compose" && r.Provider != "" && r.Provider != "docker-compose" && r.Provider != "podman-compose" {
			return fmt.Errorf("manifest: runtime %s provider %q is unsupported; use docker-compose or podman-compose", n, r.Provider)
		}
		if r.Type == "android-emulator" {
			folded := strings.ToLower(n)
			if previous, exists := androidNames[folded]; exists {
				return fmt.Errorf("manifest: Android runtime names %q and %q collide under case folding", previous, n)
			}
			androidNames[folded] = n
			if !namePattern.MatchString(r.AVD) {
				return fmt.Errorf("manifest: Android runtime %s requires a valid avd template name", n)
			}
			if r.ProjectDirectory != "" || len(r.Files) != 0 {
				return fmt.Errorf("manifest: Android runtime %s cannot use Compose files or project_directory", n)
			}
			continue
		}
		if r.AVD != "" {
			return fmt.Errorf("manifest: Compose runtime %s cannot use avd", n)
		}
		if err := RelativePath(r.ProjectDirectory); err != nil {
			return fmt.Errorf("manifest: runtime %s project_directory: %w", n, err)
		}
		if err := distinct("runtime "+n+" files", r.Files, true); err != nil {
			return err
		}
		for _, p := range r.Files {
			if err := RelativePath(p); err != nil {
				return fmt.Errorf("manifest: runtime %s file: %w", n, err)
			}
		}
	}
	for _, n := range keys(m.Components) {
		c := m.Components[n]
		if _, ok := m.Runtimes[c.Runtime]; !ok {
			return fmt.Errorf("manifest: component %s references unknown runtime %q", n, c.Runtime)
		}
		android := m.Runtimes[c.Runtime].Type == "android-emulator"
		if android && (len(c.ComposeServices) != 0 || len(c.Endpoints) != 0) {
			return fmt.Errorf("manifest: Android component %s cannot use Compose services or endpoints", n)
		}
		if err := distinct("component "+n+" compose_services", c.ComposeServices, !android); err != nil {
			return err
		}
		for _, service := range c.ComposeServices {
			if !namePattern.MatchString(service) {
				return fmt.Errorf("manifest: component %s has invalid Compose service %q", n, service)
			}
		}
		if err := distinct("component "+n+" depends_on", c.DependsOn, false); err != nil {
			return err
		}
		for _, dep := range c.DependsOn {
			if _, ok := m.Components[dep]; !ok {
				return fmt.Errorf("manifest: component %s references unknown dependency %q", n, dep)
			}
		}
		if err := distinct("component "+n+" provides", c.Provides, false); err != nil {
			return err
		}
		for i, p := range c.Readiness {
			if android && p.Type == "compose" {
				return fmt.Errorf("manifest: Android component %s cannot use Compose readiness", n)
			}
			if err := probe(m, p); err != nil {
				return fmt.Errorf("manifest: component %s readiness[%d]: %w", n, i, err)
			}
		}
		if err := names("endpoint", c.Endpoints); err != nil {
			return err
		}
		for _, en := range keys(c.Endpoints) {
			e := c.Endpoints[en]
			found := false
			for _, s := range c.ComposeServices {
				found = found || s == e.Service
			}
			if !found || e.Target < 1 || e.Target > 65535 || (e.Protocol != "" && e.Protocol != "tcp" && e.Protocol != "udp") {
				return fmt.Errorf("manifest: component %s endpoint %s requires a selected service, target port 1..65535, and tcp/udp protocol", n, en)
			}
		}
	}
	if err := validateApplications(m); err != nil {
		return err
	}
	colors := map[string]int{}
	var visit func(string, []string) error
	visit = func(n string, path []string) error {
		if colors[n] == 1 {
			return fmt.Errorf("manifest: dependency cycle %s; remove a depends_on edge", strings.Join(append(path, n), " -> "))
		}
		if colors[n] == 2 {
			return nil
		}
		colors[n] = 1
		deps := append([]string(nil), m.Components[n].DependsOn...)
		sort.Strings(deps)
		for _, dep := range deps {
			if err := visit(dep, append(path, n)); err != nil {
				return err
			}
		}
		colors[n] = 2
		return nil
	}
	for _, n := range keys(m.Components) {
		if err := visit(n, nil); err != nil {
			return err
		}
	}
	for _, n := range keys(m.Stacks) {
		s := m.Stacks[n]
		if err := distinct("stack "+n+" roots", s.Roots, true); err != nil {
			return err
		}
		for _, v := range s.Roots {
			if _, ok := m.Components[v]; !ok {
				return fmt.Errorf("manifest: stack %s references unknown root %q", n, v)
			}
		}
	}
	for _, n := range keys(m.Tests) {
		t := m.Tests[n]
		if _, ok := m.Stacks[t.Stack]; !ok {
			return fmt.Errorf("manifest: test %s references unknown stack %q", n, t.Stack)
		}
		if _, ok := m.Sources[t.Source]; !ok {
			return fmt.Errorf("manifest: test %s references unknown source %q", n, t.Source)
		}
		if err := RelativePath(t.WorkingDirectory); err != nil {
			return fmt.Errorf("manifest: test %s working_directory: %w", n, err)
		}
		if err := command("test "+n, t.Command); err != nil {
			return err
		}
		if err := duration("test "+n+" timeout", t.Timeout); err != nil {
			return err
		}
		for k, v := range t.Env {
			if k == "" || strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, 0) {
				return fmt.Errorf("manifest: test %s has invalid environment key/value", n)
			}
		}
		for _, p := range t.Artifacts {
			if err := RelativePath(p); err != nil {
				return fmt.Errorf("manifest: test %s artifact: %w", n, err)
			}
		}
	}
	return nil
}

func probe(m *Manifest, p Probe) error {
	if err := duration("probe timeout", p.Timeout); err != nil {
		return err
	}
	if err := duration("probe interval", p.Interval); err != nil {
		return err
	}
	switch p.Type {
	case "http":
		u, err := url.Parse(p.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return fmt.Errorf("HTTP probe requires an http(s) URL without credentials")
		}
		if len(p.Command) > 0 || p.Source != "" || p.WorkingDirectory != "" {
			return fmt.Errorf("HTTP probe cannot contain command/source/working_directory")
		}
	case "command":
		if p.URL != "" {
			return fmt.Errorf("command probe cannot contain URL")
		}
		if _, ok := m.Sources[p.Source]; !ok {
			return fmt.Errorf("command probe references unknown source %q", p.Source)
		}
		if err := RelativePath(p.WorkingDirectory); err != nil {
			return err
		}
		return command("probe", p.Command)
	case "compose":
		if p.URL != "" || len(p.Command) > 0 || p.Source != "" || p.WorkingDirectory != "" {
			return fmt.Errorf("compose probe uses the component services and accepts only type/timeout/interval")
		}
	default:
		return fmt.Errorf("unsupported readiness type %q; use compose, http, or command", p.Type)
	}
	return nil
}
