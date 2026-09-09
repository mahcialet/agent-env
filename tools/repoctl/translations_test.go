package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pair only positive fixture setup. Negative tests mutate the completed pair
// afterward, so source and link regressions cannot be hidden by regeneration.
func pairFixtureDocuments(t *testing.T, root string) {
	t.Helper()
	paths, err := files(root)
	if err != nil {
		t.Fatal(err)
	}
	// Populate canonical indexes before hashing any source. This is positive
	// fixture setup only; negative cases mutate the resulting files afterward.
	for _, file := range paths {
		if filepath.Base(file) == "index.md" {
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			body := string(b)
			for _, target := range links(body) {
				if strings.HasSuffix(target, ".md") && !strings.HasSuffix(target, ".ja.md") {
					ja := strings.TrimSuffix(target, ".md") + ".ja.md"
					if !strings.Contains(body, "("+ja+")") {
						body += "\n[日本語](" + ja + ")\n"
					}
				}
			}
			if err := os.WriteFile(file, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, file := range paths {
		rel, _ := filepath.Rel(root, file)
		rel = filepath.ToSlash(rel)
		if translationScope(rel) && !strings.HasSuffix(rel, ".ja.md") {
			pairFixtureDocument(t, root, rel)
		}
	}
}

func pairFixtureDocument(t *testing.T, root, canonical string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonical)))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(b), "\r\n", "\n")
	fields, err := translationMetadata(b)
	if err == nil {
		_, body, _ = strings.Cut(body[4:], "\n---\n")
	}
	front := "---\n"
	for _, key := range []string{"status", "owner", "last_verified"} {
		value, ok := fields[key]
		if !ok {
			value = map[string]string{"status": "active", "owner": "maintainers", "last_verified": "2026-09-07"}[key]
		}
		front += key + ": " + value + "\n"
	}
	if strings.HasPrefix(canonical, "docs/exec-plans/") && strings.Contains(string(b), "\nplan_id:") {
		// Positive lifecycle fixtures preserve all structured metadata in their pair.
		parts := strings.SplitN(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n---\n", 2)
		front = parts[0] + "\n"
		body = parts[1]
	}
	front += "translation_of: " + canonical + "\nsource_sha256: " + translationDigest(b) + "\n---\n"
	// Fixtures preserve headings to test anchors, while all local document links
	// use the same language. Actual repository translations are human-maintained.
	body = strings.ReplaceAll(body, ".md", ".ja.md")
	body = strings.ReplaceAll(body, ".ja.ja.md", ".ja.md")
	put(t, root, strings.TrimSuffix(canonical, ".md")+".ja.md", front+"\n[English]("+filepath.Base(canonical)+")\n"+body)
}

func TestTranslationMissingPairsReportedIndividually(t *testing.T) {
	root := fixture(t)
	put(t, root, "README.md", "# Readme\n")
	put(t, root, "docs/new.md", "# New\n")
	err := docsCheck(root)
	if err == nil {
		t.Fatal("missing pairs accepted")
	}
	for _, name := range []string{"README.ja.md", "docs/new.ja.md"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("missing individual diagnostic for %s: %v", name, err)
		}
	}
}

func TestBrokenLanguageSelectorsStillReportEveryMissingPair(t *testing.T) {
	root := fixture(t)
	put(t, root, "README.md", "# Readme\n[日本語](README.ja.md)\n")
	put(t, root, "docs/navigation.md", "# Navigation\n[日本語](navigation.ja.md)\n")
	pairFixtureDocument(t, root, "README.md")
	pairFixtureDocument(t, root, "docs/navigation.md")
	if err := docsCheck(root); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"README.ja.md", "docs/navigation.ja.md"} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := os.ReadFile(filepath.Join(root, "README.md"))
	err := docsCheck(root)
	if err == nil || !strings.HasPrefix(err.Error(), "AGENTENV-DOC-004") || strings.Count(err.Error(), "AGENTENV-DOC-007") != 2 {
		t.Fatalf("selector failure masked missing pairs: %v", err)
	}
	for _, p := range []string{"README.ja.md", "docs/navigation.ja.md"} {
		if !strings.Contains(err.Error(), "missing Japanese translation "+p) {
			t.Fatalf("missing individual pair %s: %v", p, err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("check mutated missing translation %s: %v", p, err)
		}
	}
	after, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if string(before) != string(after) {
		t.Fatal("check changed canonical selector document")
	}
}

func TestTranslationMetadataFailures(t *testing.T) {
	for _, tc := range []struct{ name, old, new, code string }{
		{"wrong-source", "translation_of: ARCHITECTURE.md", "translation_of: AGENTS.md", "DOC-008"},
		{"relative-source", "translation_of: ARCHITECTURE.md", "translation_of: ./ARCHITECTURE.md", "DOC-008"},
		{"missing-source", "translation_of: ARCHITECTURE.md\n", "", "DOC-008"},
		{"duplicate-source", "translation_of: ARCHITECTURE.md", "translation_of: ARCHITECTURE.md\ntranslation_of: ARCHITECTURE.md", "DOC-008"},
		{"unterminated-source-quote", "translation_of: ARCHITECTURE.md", "translation_of: 'ARCHITECTURE.md", "DOC-008"},
		{"unterminated-source-double-quote", "translation_of: ARCHITECTURE.md", "translation_of: \"ARCHITECTURE.md", "DOC-008"},
		{"unterminated-hash-quote", "source_sha256: ", "source_sha256: '", "DOC-008"},
		{"trailing-source-quote", "translation_of: ARCHITECTURE.md", "translation_of: ARCHITECTURE.md'", "DOC-008"},
		{"nested-source", "translation_of: ARCHITECTURE.md", "nested:\n  translation_of: ARCHITECTURE.md", "DOC-008"},
		{"multiline-source", "translation_of: ARCHITECTURE.md", "translation_of: |\n  ARCHITECTURE.md", "DOC-008"},
		{"bad-hash", "source_sha256:", "source_sha256: invalid", "DOC-008"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t)
			p := "ARCHITECTURE.ja.md"
			b, _ := os.ReadFile(filepath.Join(root, p))
			put(t, root, p, strings.Replace(string(b), tc.old, tc.new, 1))
			err := docsCheck(root)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("expected %s: %v", tc.code, err)
			}
		})
	}
	root := fixture(t)
	put(t, root, "orphan.ja.md", "---\ntranslation_of: absent.md\nsource_sha256: "+strings.Repeat("0", 64)+"\n---\n# 孤立\n")
	if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "orphan.ja.md has no canonical") {
		t.Fatalf("orphan accepted: %v", err)
	}
}

func TestTranslationStalePairsAndSynchronizedUpdate(t *testing.T) {
	root := fixture(t)
	for _, name := range []string{"ARCHITECTURE.md", "docs/design-docs/design.md"} {
		b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		put(t, root, name, string(b)+"\nUpdated canonical behavior.\n")
	}
	err := docsCheck(root)
	if err == nil || strings.Count(err.Error(), "AGENTENV-DOC-009") != 2 {
		t.Fatalf("stale pairs not individually reported: %v", err)
	}
	before, _ := os.ReadFile(filepath.Join(root, "ARCHITECTURE.ja.md"))
	_ = docsCheck(root)
	after, _ := os.ReadFile(filepath.Join(root, "ARCHITECTURE.ja.md"))
	if string(before) != string(after) {
		t.Fatal("docs-check mutated translation")
	}
	pairFixtureDocument(t, root, "ARCHITECTURE.md")
	pairFixtureDocument(t, root, "docs/design-docs/design.md")
	if err := docsCheck(root); err != nil {
		t.Fatalf("synchronized pair rejected: %v", err)
	}
}

func TestTranslationDigestNormalizesOnlyCRLF(t *testing.T) {
	canonical := "---\nstatus: active\n---\n# Title\nbody\n"
	sum := sha256.Sum256([]byte(canonical))
	want := hex.EncodeToString(sum[:])
	if got := translationDigest([]byte(strings.ReplaceAll(canonical, "\n", "\r\n"))); got != want {
		t.Fatalf("CRLF source hash=%s want=%s", got, want)
	}
	if translationDigest([]byte(canonical+"\n")) == want {
		t.Fatal("hash omitted trailing source bytes")
	}
	root := fixture(t)
	paths, _ := files(root)
	for _, p := range paths {
		if strings.HasSuffix(p, ".md") {
			b, _ := os.ReadFile(p)
			if err := os.WriteFile(p, []byte(strings.ReplaceAll(string(b), "\n", "\r\n")), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := docsCheck(root); err != nil {
		t.Fatalf("Windows checkout rejected: %v", err)
	}
}

func TestTranslationMetadataTracksCanonicalStatusOwnerAndDate(t *testing.T) {
	for _, field := range []string{"status: active", "owner: maintainers", "last_verified: 2026-09-07"} {
		t.Run(field, func(t *testing.T) {
			root := fixture(t)
			p := "docs/design-docs/design.ja.md"
			b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
			replacement := map[string]string{"status: active": "status: draft", "owner: maintainers": "owner: translators", "last_verified: 2026-09-07": "last_verified: 2026-09-08"}[field]
			put(t, root, p, strings.Replace(string(b), field, replacement, 1))
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-011") {
				t.Fatalf("metadata mismatch accepted: %v", err)
			}
		})
	}
}

func TestJapaneseLinksAndLanguageSpecificIndex(t *testing.T) {
	for _, tc := range []struct{ name, path, data, code string }{
		{"broken-ja-link", "ARCHITECTURE.ja.md", "\n[壊れたリンク](absent.md)\n", "DOC-004"},
		{"broken-ja-anchor", "ARCHITECTURE.ja.md", "\n[壊れた見出し](#存在しない)\n", "DOC-004"},
		{"ja-index-links-only-english", "docs/design-docs/index.ja.md", "", "DOC-001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t)
			b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.path)))
			data := string(b) + tc.data
			if tc.data == "" {
				data = strings.ReplaceAll(data, "(design.ja.md)", "(design.md)")
			}
			put(t, root, tc.path, data)
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("%s accepted: %v", tc.name, err)
			}
		})
	}
}

func TestJapaneseActivePlanMayTranslateSectionHeadings(t *testing.T) {
	root := fixture(t)
	p := "docs/exec-plans/active/plan.ja.md"
	b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	data := string(b)
	for _, section := range planSections {
		data = strings.ReplaceAll(data, "## "+section, "## "+japanesePlanSections[section])
	}
	put(t, root, p, data)
	if err := docsCheck(root); err != nil {
		t.Fatalf("translated section headings rejected: %v", err)
	}
}

func TestTranslationExceptionsAreExactAndRestricted(t *testing.T) {
	for _, name := range []string{"docs/generated/schema.md", "docs/references/handoffs/history.md", "docs/exec-plans/completed/agent-env-mvp.md"} {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			put(t, root, name, metadata+"# Archive\n")
			put(t, root, "docs/translation-exceptions.json", fmt.Sprintf(`{"exceptions":[{"path":%q,"reason":"Historical or generated provenance"}]}`, name))
			if err := docsCheck(root); err != nil {
				t.Fatal(err)
			}
			put(t, root, strings.TrimSuffix(name, ".md")+"-new.md", metadata+"# New\n")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-007") {
				t.Fatalf("exception expanded to unlisted file: %v", err)
			}
		})
	}
	for _, exception := range []string{
		`{"exceptions":[{"path":"ARCHITECTURE.md","reason":"too long"}]}`,
		`{"exceptions":[{"path":"docs/design-docs/design.md","reason":"too long"}]}`,
		`{"exceptions":[{"path":"docs/exec-plans/active/plan.md","reason":"active"}]}`,
		`{"exceptions":[{"path":"docs/generated/*","reason":"all generated"}]}`,
		`{"exceptions":[{"path":"docs/generated/missing.md","reason":"missing"}]}`,
		`{"exceptions":[{"path":"docs/generated/schema.md","reason":""}]}`,
		`{"exceptions":[{"path":"docs/generated/schema.md","reason":"generated"},{"path":"docs/generated/schema.md","reason":"duplicate"}]}`,
		`{"exceptions":[{"path":"docs/generated/schema.md","reason":"generated","extra":true}]}`,
		`{"exceptions":[{"path":"ARCHITECTURE.md","reason":"not archival"}],"exceptions":[]}`,
		`{"exceptions":[{"path":"ARCHITECTURE.md","path":"docs/generated/schema.md","reason":"generated"}]}`,
		`{"exceptions":[{"path":"docs/generated/schema.md","reason":"","reason":"generated"}]}`,
		`{"Exceptions":[{"path":"docs/generated/schema.md","reason":"generated"}]}`,
		`{"Exceptions":[{"path":"ARCHITECTURE.md","reason":"not archival"}],"exceptions":[]}`,
		`{"exceptions":[{"Path":"ARCHITECTURE.md","path":"docs/generated/schema.md","reason":"generated"}]}`,
		`{"exceptions":[{"path":"docs/generated/schema.md","Reason":"generated"}]}`,
		`{"exceptions":null}`, `{}`, `{"exceptions":[]} {"exceptions":[]}`,
	} {
		t.Run(exception, func(t *testing.T) {
			root := fixture(t)
			put(t, root, "docs/generated/schema.md", "# Generated\n")
			put(t, root, "docs/translation-exceptions.json", exception)
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-010") {
				t.Fatalf("unsafe exception accepted: %v", err)
			}
		})
	}
}

func TestInvalidTranslationExceptionsReportedIndividually(t *testing.T) {
	root := fixture(t)
	put(t, root, "docs/translation-exceptions.json", `{"exceptions":[{"path":"ARCHITECTURE.md","reason":"not archival"},{"path":"docs/exec-plans/active/plan.md","reason":"not completed"}]}`)
	err := docsCheck(root)
	if err == nil || strings.Count(err.Error(), "AGENTENV-DOC-010") != 2 {
		t.Fatalf("invalid exceptions not individually reported: %v", err)
	}
}

func TestExemptCanonicalDoesNotBypassJapaneseChecks(t *testing.T) {
	root := fixture(t)
	canonical := "docs/references/handoffs/history.md"
	put(t, root, canonical, "# Historical record\n")
	put(t, root, "docs/translation-exceptions.json", fmt.Sprintf(`{"exceptions":[{"path":%q,"reason":"Historical source"}]}`, canonical))
	pairFixtureDocument(t, root, canonical)
	ja := "docs/references/handoffs/history.ja.md"
	b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(ja)))
	put(t, root, ja, string(b)+"\n[壊れたリンク](missing.md)\n")
	if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004") {
		t.Fatalf("exception suppressed Japanese link validation: %v", err)
	}
}

func TestDocumentationScopeIncludesEveryDocsSubdirectory(t *testing.T) {
	for _, directory := range []string{"docs/vendor", "docs/.notes", "docs/node_modules", "docs/nested/.hidden/vendor"} {
		t.Run(directory, func(t *testing.T) {
			root := fixture(t)
			canonical := directory + "/note.md"
			put(t, root, canonical, "# Note\n")
			err := docsCheck(root)
			if err == nil || !strings.Contains(err.Error(), "AGENTENV-DOC-007: "+canonical) {
				t.Fatalf("directory silently excluded from translation coverage: %v", err)
			}
			pairFixtureDocument(t, root, canonical)
			if err := docsCheck(root); err != nil {
				t.Fatalf("paired scoped document rejected: %v", err)
			}
			ja := strings.TrimSuffix(canonical, ".md") + ".ja.md"
			b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(ja)))
			put(t, root, ja, string(b)+"\n[bad](missing.md)\n")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004") {
				t.Fatalf("scoped Japanese links not checked: %v", err)
			}
		})
	}
}

func TestDocumentationEnumerationPreservesOtherMarkdownLinkChecks(t *testing.T) {
	root := fixture(t)
	put(t, root, "examples/README.md", "# Example\n[bad](missing.md)\n")
	put(t, root, "vendor/ignored.go", "intentionally not Go source")
	if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004: examples/README.md") {
		t.Fatalf("legacy external-doc links no longer checked: %v", err)
	}
	if err := formatCheck(root); err != nil {
		t.Fatalf("common code scan exclusions changed: %v", err)
	}
}

func TestTranslationMetadataBalancedQuotedScalars(t *testing.T) {
	root := fixture(t)
	p := "ARCHITECTURE.ja.md"
	b, _ := os.ReadFile(filepath.Join(root, p))
	data := string(b)
	data = strings.Replace(data, "translation_of: ARCHITECTURE.md", "translation_of: 'ARCHITECTURE.md'", 1)
	data = strings.Replace(data, "source_sha256: "+translationDigest([]byte("# Architecture\n")), "source_sha256: \""+translationDigest([]byte("# Architecture\n"))+"\"", 1)
	put(t, root, p, data)
	if err := docsCheck(root); err != nil {
		t.Fatalf("balanced quoted metadata rejected: %v", err)
	}
}
