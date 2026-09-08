package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, path, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo spaces 日本語")
	put(t, root, "go.mod", "module example.org/repo\n\ngo 1.26.0\n")
	put(t, root, "AGENTS.md", "# Agent map\n[Architecture](ARCHITECTURE.md)\n")
	put(t, root, "ARCHITECTURE.md", "# Architecture\n")
	put(t, root, "docs/design-docs/index.md", "[Design](design.md)\n")
	put(t, root, "docs/design-docs/design.md", metadata+"# Design\n")
	plan := metadata
	for _, section := range planSections {
		plan += "\n## " + section + "\nEvidence goes here.\n"
	}
	put(t, root, "docs/exec-plans/active/plan.md", plan)
	pairFixtureDocuments(t, root)
	return root
}

const metadata = "---\nstatus: active\nowner: maintainers\nlast_verified: 2026-09-07\n---\n"

func TestFormattingCRLFAndActualDrift(t *testing.T) {
	root := t.TempDir()
	put(t, root, "main.go", "package main\r\n\r\nfunc main() {}\r\n")
	if err := formatCheck(root); err != nil {
		t.Fatalf("Windows checkout rejected: %v", err)
	}
	put(t, root, "main.go", "package main\r\nfunc main(){ }\r\n")
	if err := formatCheck(root); err == nil || !strings.Contains(err.Error(), "FMT-001") {
		t.Fatalf("real formatting drift missed: %v", err)
	}
}

func TestDocsBrokenFixtures(t *testing.T) {
	for _, tc := range []struct{ name, path, data, code string }{
		{"missing-link", "ARCHITECTURE.md", "[missing](nowhere.md)", "DOC-004"},
		{"missing-fragment", "ARCHITECTURE.md", "[missing](#nowhere)", "DOC-004"},
		{"reference-link", "ARCHITECTURE.md", "[text][reference]\n[reference]: missing.md", "DOC-004"},
		{"agent-length", "AGENTS.md", strings.Repeat("line\n", 151), "DOC-002"},
		{"agent-path", "AGENTS.md", "Read `docs/missing.md`.", "DOC-003"},
		{"unindexed", "docs/design-docs/extra.md", metadata + "# Extra", "DOC-001"},
		{"missing-metadata", "docs/design-docs/design.md", "# Design", "DOC-006"},
		{"bad-date", "docs/design-docs/design.md", strings.ReplaceAll(metadata, "2026-09-07", "2026-02-30"), "DOC-006"},
		{"bad-status", "docs/design-docs/design.md", strings.ReplaceAll(metadata, "active", "anything"), "DOC-006"},
		{"missing-plan-section", "docs/exec-plans/active/plan.md", metadata + "## Progress", "DOC-005"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t)
			if err := docsCheck(root); err != nil {
				t.Fatal(err)
			}
			put(t, root, tc.path, tc.data)
			err := docsCheck(root)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
}

func TestDocumentationLinks(t *testing.T) {
	root := fixture(t)
	put(t, root, "ARCHITECTURE.md", "# Architecture\n[space](<docs/a b 日本語.md#the-heading>)\n[external](https://example.org)\n```md\n[example](missing.md)\n```\n")
	put(t, root, "docs/a b 日本語.md", "# The heading\n")
	pairFixtureDocuments(t, root)
	if err := docsCheck(root); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedDrift(t *testing.T) {
	root := fixture(t)
	if err := generatedCheck(root, false); err != nil {
		t.Fatal(err)
	}
	put(t, root, "internal/store/sqlite/migrations/001.sql", "CREATE TABLE lease (id TEXT PRIMARY KEY);\n")
	if err := generatedCheck(root, false); err == nil {
		t.Fatal("missing schema accepted")
	}
	if err := generatedCheck(root, true); err != nil {
		t.Fatal(err)
	}
	if err := generatedCheck(root, false); err != nil {
		t.Fatal(err)
	}
	put(t, root, "docs/generated/db-schema.md", "drift")
	if err := generatedCheck(root, false); err == nil || !strings.Contains(err.Error(), "GEN-001") {
		t.Fatalf("drift: %v", err)
	}
	if err := generatedCheck(root, true); err != nil {
		t.Fatal(err)
	}
	put(t, root, "internal/store/sqlite/migrations/002.sql", "ALTER TABLE lease ADD COLUMN owner TEXT;\n")
	if err := generatedCheck(root, false); err == nil {
		t.Fatal("migration addition drift accepted")
	}
}

func TestArchitectureBoundaries(t *testing.T) {
	for _, tc := range []struct {
		pkg, dependency string
		bad             bool
	}{
		{"domain", "internal/runtime/compose", true},
		{"domain", "database/sql", true},
		{"domain", "github.com/go-git/go-git/v5", true},
		{"domain", "internal/domainadapter", true},
		{"domain", "fmt", false},
		{"app", "internal/cli", true},
		{"app", "internal/browser/cdp", true},
		{"store/sqlite", "internal/browser/cdp", true},
		{"app", "internal/domain", false},
		{"browser/cdp", "internal/app", true},
		{"browser/cdp", "internal/runtime/process", true},
		{"browser/cdp", "internal/runtime/android", true},
		{"browser/cdp", "internal/execx", true},
		{"browser/cdp", "internal/store/sqlite", true},
		{"browser/cdp", "internal/domain", false},
		{"browser/cdp", "internal/evidence", false},
		{"browser/cdp", "os/exec", true},
		{"browser/cdp", "github.com/gorilla/websocket", false},
		{"runtime/process", "internal/browser/cdp", true},
		{"runtime/android", "internal/browser/cdp", true},
		{"runtime/flutter", "internal/browser/cdp", true},

		{"runtime/compose", "internal/cli", true},
		{"runtime/process", "internal/runtime/compose", true},
		{"runtime/process", "internal/runtime/android", true},
		{"runtime/android", "internal/runtime/process", true},
		{"runtime/process", "internal/cli", true},
		{"domain", "internal/runtime/process", true},
		{"store/sqlite", "internal/runtime/process", true},
		{"runtime/process", "internal/domain", false},
		{"runtime/process", "internal/execx", false},
		{"cli", "internal/runtime/process", false},
		{"runtime/android", "internal/runtime/compose", true},
		{"runtime/compose", "internal/runtime/android", true},
		{"runtime/android", "internal/app", false},
		{"runtime/android", "internal/execx", false},
		{"runtime/flutter", "internal/runtime/android", true},
		{"runtime/flutter/build", "internal/runtime/android/device", true},
		{"runtime/android", "internal/runtime/flutter", true},
		{"runtime/flutter", "internal/runtime/compose", true},
		{"runtime/compose", "internal/runtime/flutter", true},
		{"runtime/flutter", "internal/cli", true},
		{"domain", "internal/runtime/flutter", true},
		{"store/sqlite", "internal/runtime/flutter", true},
		{"runtime/flutter", "internal/app", false},
		{"runtime/flutter", "internal/domain", false},
		{"runtime/flutter", "internal/execx", false},
		{"runtime/flutter/build", "internal/runtime/flutter/helpers", false},
		{"cli", "internal/runtime/flutter", false},
		{"store/sqlite", "internal/app", true},
		{"store/sqlite", "internal/domain", false},
		{"cli", "internal/runtime/compose", false},
		{"cli", "internal/app", false},
	} {
		t.Run(tc.pkg+"/"+tc.dependency, func(t *testing.T) {
			root := fixture(t)
			dep := tc.dependency
			if strings.HasPrefix(dep, "internal/") {
				dep = "example.org/repo/" + dep
			}
			put(t, root, "internal/"+tc.pkg+"/bad.go", "package fixture\nimport _ "+strconvQuote(dep)+"\n")
			err := archCheck(root)
			if (err != nil) != tc.bad {
				t.Fatalf("bad=%v got %v", tc.bad, err)
			}
			if tc.bad && !strings.Contains(err.Error(), "AGENTENV-ARCH-002") {
				t.Fatal(err)
			}
		})
	}
}

func strconvQuote(s string) string { return `"` + s + `"` }

func TestCommandArgvRoundTrip(t *testing.T) {
	if os.Getenv("REPOCTL_HELPER") == "1" {
		for _, a := range os.Args[3:] {
			os.Stdout.WriteString(a + "\n")
		}
		os.Exit(0)
	}
	root := fixture(t)
	args := []string{"-test.run=TestCommandArgvRoundTrip", "--", "space value", `quote"value`, "日本語", `trailing\`}
	var out, errOut bytes.Buffer
	if err := command(root, &out, &errOut, []string{"REPOCTL_HELPER=1"}, os.Args[0], args...); err != nil {
		t.Fatalf("%v: %s", err, errOut.String())
	}
	if !strings.HasSuffix(out.String(), strings.Join(args[2:], "\n")+"\n") {
		t.Fatalf("argument corruption: %q", out.String())
	}
}

func TestFailureExitStatus(t *testing.T) {
	var out bytes.Buffer
	if status := run(nil, &out, &out); status != 2 {
		t.Fatalf("usage status %d", status)
	}
	root := fixture(t)
	t.Chdir(root)
	put(t, root, "AGENTS.md", "[broken](no-file.md)")
	if status := run([]string{"docs-check"}, &out, &out); status != 1 {
		t.Fatalf("broken docs status %d", status)
	}
	if !strings.Contains(out.String(), "AGENTENV-DOC-004") {
		t.Fatal(out.String())
	}
}
