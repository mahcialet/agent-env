package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslationRequiresVisibleSourceLink(t *testing.T) {
	for _, body := range []string{"# Translation\n", "[](ARCHITECTURE.md)\n", "[   ](ARCHITECTURE.md)\n", "[English](AGENTS.md)\n", "```md\n[English](ARCHITECTURE.md)\n```\n", "`[English](ARCHITECTURE.md)`\n", "<!-- [English](ARCHITECTURE.md) -->\n", "![English](ARCHITECTURE.md)\n", "[unused]: ARCHITECTURE.md\n",
		"    [English](ARCHITECTURE.md)\n", "\t[English](ARCHITECTURE.md)\n",
		`\[English](ARCHITECTURE.md)`, "<!-- [English](ARCHITECTURE.md)\n",
		"````md\n```\n[English](ARCHITECTURE.md)\n````\n",
		"```md\n~~~\n[English](ARCHITECTURE.md)\n```\n",
		"`` example ` [English](ARCHITECTURE.md) ``\n",
		"[English][src]\n\n[src]: AGENTS.md\n[src]: ARCHITECTURE.md\n"} {
		t.Run(body, func(t *testing.T) {
			root := fixture(t)
			p := "ARCHITECTURE.ja.md"
			b, _ := os.ReadFile(filepath.Join(root, p))
			_, rest, _ := strings.Cut(string(b), "\n---\n")
			put(t, root, p, strings.TrimSuffix(string(b), rest)+body)
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-012") {
				t.Fatalf("missing visible source link accepted: %v", err)
			}
		})
	}
	for _, body := range []string{"[English](./ARCHITECTURE.md)\n", "<!--\n```md\n-->\n\n[English](ARCHITECTURE.md)\n", "`<!--` denotes an HTML comment.\n\n[English](ARCHITECTURE.md)\n", "~~~md\n<!--\n~~~\n[English](ARCHITECTURE.md)\n", "[`English`](ARCHITECTURE.md)\n", "[English](ARCHITECTURE.md#architecture)\n", "[English][source]\n\n[source]: ARCHITECTURE.md\n", "[source][]\n\n[source]: ARCHITECTURE.md\n", "[source]\n\n[source]: ARCHITECTURE.md\n"} {
		t.Run(body, func(t *testing.T) {
			root := fixture(t)
			p := "ARCHITECTURE.ja.md"
			b, _ := os.ReadFile(filepath.Join(root, p))
			_, rest, _ := strings.Cut(string(b), "\n---\n")
			put(t, root, p, strings.TrimSuffix(string(b), rest)+body)
			if err := docsCheck(root); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRootTranslationsRequireCoreMetadata(t *testing.T) {
	for _, canonical := range []string{"AGENTS.md", "README.md"} {
		for _, field := range []string{"status", "owner", "last_verified"} {
			for _, value := range []string{"missing", "", "invalid", `" "`} {
				if field == "owner" && value == "invalid" {
					continue
				}
				t.Run(canonical+"/"+field+"/"+value, func(t *testing.T) {
					root := fixture(t)
					if canonical == "README.md" {
						put(t, root, canonical, "# Readme\n")
						pairFixtureDocument(t, root, canonical)
					}
					p := strings.TrimSuffix(canonical, ".md") + ".ja.md"
					b, _ := os.ReadFile(filepath.Join(root, p))
					lines := strings.Split(string(b), "\n")
					for n, line := range lines {
						if strings.HasPrefix(line, field+":") {
							if value == "missing" {
								lines = append(lines[:n], lines[n+1:]...)
							} else {
								lines[n] = field + ": " + value
							}
							break
						}
					}
					put(t, root, p, strings.Join(lines, "\n"))
					if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-006") {
						t.Fatalf("invalid root metadata accepted: %v", err)
					}
				})
			}
		}
	}
}

func TestCompletedPlanExceptionCannotExpandMigrationSet(t *testing.T) {
	root := fixture(t)
	p := "docs/exec-plans/completed/new-plan.md"
	put(t, root, p, metadata+"# New completed plan\n")
	put(t, root, "docs/translation-exceptions.json", fmt.Sprintf(`{"exceptions":[{"path":%q,"reason":"New completion"}]}`, p))
	err := docsCheck(root)
	if err == nil || !strings.Contains(err.Error(), "DOC-010") || !strings.Contains(err.Error(), "DOC-007") {
		t.Fatalf("new completed plan bypassed translation: %v", err)
	}
}

func TestRequiredPlanSectionsMustBeProse(t *testing.T) {
	for _, path := range []string{"docs/exec-plans/active/plan.md", "docs/exec-plans/active/plan.ja.md"} {
		t.Run(path, func(t *testing.T) {
			root := fixture(t)
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			start := strings.Index(text, "## Purpose / Big Picture")
			if start < 0 {
				t.Fatal("fixture lacks headings")
			}
			put(t, root, path, text[:start]+"```markdown\n"+text[start:]+"\n```\n")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-005") {
				t.Fatalf("example headings satisfy mandatory sections: %v", err)
			}
		})
	}
}

func TestIndexRequiresNavigableJapaneseLinks(t *testing.T) {
	for _, replacement := range []string{"[unused-ja]: design.ja.md\n", "<!-- [Japanese](design.ja.md) -->\n"} {
		t.Run(replacement, func(t *testing.T) {
			root := fixture(t)
			p := "docs/design-docs/index.md"
			data, _ := os.ReadFile(filepath.Join(root, p))
			text := strings.ReplaceAll(string(data), "[日本語](design.ja.md)", "")
			put(t, root, p, text+"\n"+replacement)
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-001") {
				t.Fatalf("hidden index link accepted: %v", err)
			}
		})
	}
}
