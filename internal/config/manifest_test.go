package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../../testdata/manifests/api-dashboard.yaml")
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestStrictManifest(t *testing.T) {
	base := strings.ReplaceAll(string(fixture(t)), "\r\n", "\n")
	for _, tc := range []struct{ name, body, want string }{
		{"unknown", base + "typo: true\n", "field typo not found"},
		{"nested unknown", strings.Replace(base, "default_ref: HEAD", "default_ref: HEAD\n    typo: bad", 1), "field typo not found"},
		{"duplicate", strings.Replace(base, "default_ref: HEAD", "default_ref: HEAD\n    default_ref: main", 1), "already defined"},
		{"version", strings.Replace(base, "version: 1", "version: 2", 1), "unsupported version"},
		{"documents", base + "---\nversion: 1\n", "exactly one"},
		{"empty document", base + "---\n", "exactly one"},
		{"runtime", strings.Replace(base, "type: compose", "type: flutter-android", 1), "unsupported"},
		{"runtime source", strings.Replace(base, "source: backend", "source: missing", 1), "unknown source"},
		{"source remote", strings.Replace(base, "repository: .", "repository: https://example.com/repo", 1), "local repository"},
		{"file escape", strings.Replace(base, "- compose.yaml", "- ../compose.yaml", 1), "escapes"},
		{"portable escape", strings.Replace(base, "- compose.yaml", "- '..\\compose.yaml'", 1), "nonportable"},
		{"drive escape", strings.Replace(base, "- compose.yaml", "- C:/compose.yaml", 1), "source-relative"},
		{"unknown dependency", strings.Replace(base, "depends_on:\n      - api", "depends_on:\n      - absent", 1), "unknown dependency"},
		{"cycle", strings.Replace(base, "depends_on:\n      - api", "depends_on:\n      - dashboard", 1), "cycle"},
		{"empty roots", strings.Replace(base, "roots:\n      - api", "roots: []", 1), "must not be empty"},
		{"test stack", strings.Replace(base, "stack: api", "stack: absent", 1), "unknown stack"},
		{"test source", strings.Replace(base, "working_directory: .", "source: missing\n    working_directory: .", 1), "already defined"},
		{"string command", strings.Replace(base, "command:\n      - go\n      - test\n      - ./...", "command: go test ./...", 1), "cannot unmarshal"},
		{"invalid timeout", strings.Replace(base, "working_directory: .", "timeout: 0s\n    working_directory: .", 1), "positive duration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.body == base {
				t.Fatal("negative fixture did not modify the manifest")
			}
			_, e := Parse([]byte(tc.body))
			if e == nil || !strings.Contains(e.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v", tc.want, e)
			}
		})
	}
}

func TestCRLFManifest(t *testing.T) {
	b := strings.ReplaceAll(string(fixture(t)), "\r\n", "\n")
	m, err := Parse([]byte(strings.ReplaceAll(b, "\n", "\r\n")))
	if err != nil {
		t.Fatal(err)
	}
	lf, err := Parse([]byte(b))
	if err != nil {
		t.Fatal(err)
	}
	if Digest(m) != Digest(lf) {
		t.Fatal("line endings changed manifest identity")
	}
}

func TestManifestCanonicalDigestAndLoad(t *testing.T) {
	b := fixture(t)
	m, e := Parse(b)
	if e != nil {
		t.Fatal(e)
	}
	changed, e := Parse(append([]byte("# formatting does not change identity\n"), b...))
	if e != nil {
		t.Fatal(e)
	}
	if Digest(m) != Digest(changed) || len(Digest(m)) != 64 {
		t.Fatal("unstable digest")
	}
	s := changed.Sources["backend"]
	s.DefaultRef = "main"
	changed.Sources["backend"] = s
	if Digest(m) == Digest(changed) {
		t.Fatal("source changed without digest change")
	}
	root := filepath.Join(t.TempDir(), "source with spaces 日本語")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(root, ".agent-env.yaml")
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{root, p} {
		loaded, e := Load(p)
		if e != nil || Digest(loaded) != Digest(m) {
			t.Fatalf("load %s: %v", p, e)
		}
	}
}

func TestReadinessEndpointsAndMultipleSources(t *testing.T) {
	m, e := Parse(fixture(t))
	if e != nil {
		t.Fatal(e)
	}
	m.Sources["schema"] = Source{Repository: "../shared schema 日本語", DefaultRef: "main"}
	c := m.Components["api"]
	c.Readiness = []Probe{{Type: "compose", Timeout: "1m"}, {Type: "http", URL: "http://127.0.0.1:1234/health", Timeout: "3s", Interval: "10ms"}, {Type: "command", Source: "schema", Command: []string{"tool", "", `space and "quotes"`}}}
	c.Endpoints = map[string]Endpoint{"api": {Service: "api", Target: 8080}}
	m.Components["api"] = c
	if e := Validate(m); e != nil {
		t.Fatal(e)
	}
	c.Readiness[1].URL = "http://secret:credential@localhost/"
	m.Components["api"] = c
	if e := Validate(m); e == nil {
		t.Fatal("credential URL accepted")
	}
	c.Readiness = nil
	c.Endpoints["api"] = Endpoint{Service: "unselected", Target: 8080}
	m.Components["api"] = c
	if e := Validate(m); e == nil {
		t.Fatal("unselected endpoint accepted")
	}
}

func TestRelativePath(t *testing.T) {
	for _, p := range []string{".", "", "dir with spaces/日本語", "a/../b"} {
		if e := RelativePath(p); e != nil {
			t.Errorf("%q: %v", p, e)
		}
	}
	for _, p := range []string{"../escape", "a/../../escape", "/root", "C:/root", `C:\root`, `\\server\share`, `..\escape`, "a\x00b"} {
		if e := RelativePath(p); e == nil {
			t.Errorf("accepted %q", p)
		}
	}
}
