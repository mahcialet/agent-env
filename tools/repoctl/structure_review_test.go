package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the published Japanese section titles, independently listed so a
// validator mapping change cannot silently change the regression expectations.
var reviewJapaneseSections = [][2]string{
	{"Purpose / Big Picture", "目的 / 全体像"},
	{"Progress", "進捗"},
	{"Surprises & Discoveries", "想定外の発見"},
	{"Decision Log", "判断の記録"},
	{"Outcomes & Retrospective", "成果と振り返り"},
	{"Context and Orientation", "背景と構成"},
	{"Plan of Work", "作業計画"},
	{"Concrete Steps", "具体的な手順"},
	{"Validation and Acceptance", "検証と受け入れ"},
	{"Idempotence and Recovery", "冪等性と復旧"},
	{"Artifacts and Notes", "成果物と注記"},
	{"Interfaces and Dependencies", "インターフェースと依存"},
}

func reviewRead(t *testing.T, root, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func reviewStructureFixture(t *testing.T) string {
	t.Helper()
	root := fixture(t)
	index := "docs/design-docs/index.md"
	data := reviewRead(t, root, index)
	if !strings.Contains(data, "(design.ja.md)") {
		put(t, root, index, data+"\n[日本語](design.ja.md)\n")
		pairFixtureDocument(t, root, index)
	}
	if err := docsCheck(root); err != nil {
		t.Fatalf("valid review fixture: %v", err)
	}
	return root
}

func TestReviewJapaneseDocumentsRequireBothLocalIndexes(t *testing.T) {
	for _, language := range []string{"English", "Japanese"} {
		t.Run(language, func(t *testing.T) {
			root := reviewStructureFixture(t)
			index := "docs/design-docs/index.ja.md"
			if language == "English" {
				index = "docs/design-docs/index.md"
			}
			before := reviewRead(t, root, index)
			after := strings.ReplaceAll(before, "(design.ja.md)", "(design.md)")
			if before == after {
				t.Fatal("fixture did not contain a Japanese document link")
			}
			put(t, root, index, after)
			if language == "English" {
				// Preserve the valid Japanese index and acknowledge only the
				// changed source bytes, isolating index coverage from hash drift.
				jaIndex := "docs/design-docs/index.ja.md"
				ja := reviewRead(t, root, jaIndex)
				ja = strings.Replace(ja, translationDigest([]byte(before)), translationDigest([]byte(after)), 1)
				put(t, root, jaIndex, ja)
			}
			err := docsCheck(root)
			if err == nil || !strings.Contains(err.Error(), "DOC-001") || !strings.Contains(err.Error(), filepath.Base(index)) {
				t.Fatalf("missing %s index coverage accepted: %v", language, err)
			}
		})
	}
}

func reviewTranslatedPlan(t *testing.T, root string) string {
	t.Helper()
	data := reviewRead(t, root, "docs/exec-plans/active/plan.ja.md")
	for _, section := range reviewJapaneseSections {
		data = strings.ReplaceAll(data, "## "+section[0]+"\n", "## "+section[1]+"\n")
	}
	return data
}

func TestReviewJapanesePlanRequiresEverySection(t *testing.T) {
	for _, section := range reviewJapaneseSections {
		t.Run(section[0], func(t *testing.T) {
			root := reviewStructureFixture(t)
			data := reviewTranslatedPlan(t, root)
			data = strings.Replace(data, "## "+section[1]+"\n", "## 未定義の節\n", 1)
			put(t, root, "docs/exec-plans/active/plan.ja.md", data)
			err := docsCheck(root)
			if err == nil || !strings.Contains(err.Error(), "DOC-005") || !strings.Contains(err.Error(), "plan.ja.md") {
				t.Fatalf("missing Japanese %q accepted: %v", section[1], err)
			}
		})
	}
}

func TestReviewJapanesePlanAcceptsPublishedAndEnglishHeadings(t *testing.T) {
	for _, language := range []string{"Japanese", "English", "mixed", "CRLF"} {
		t.Run(language, func(t *testing.T) {
			root := reviewStructureFixture(t)
			data := reviewTranslatedPlan(t, root)
			for n, section := range reviewJapaneseSections {
				if language == "English" || language == "mixed" && n%2 == 0 {
					data = strings.Replace(data, "## "+section[1]+"\n", "## "+section[0]+"\n", 1)
				}
			}
			if language == "CRLF" {
				data = strings.ReplaceAll(data, "\n", "\r\n")
			}
			put(t, root, "docs/exec-plans/active/plan.ja.md", data)
			if err := docsCheck(root); err != nil {
				t.Fatalf("valid %s sections rejected: %v", language, err)
			}
		})
	}
}
