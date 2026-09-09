package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFragmentsRequireRenderedTargetHeadings(t *testing.T) {
	for _, fake := range []string{"```md\n## Phantom\n```", "<!--\n## Phantom\n-->", "    ## Phantom"} {
		t.Run(fake, func(t *testing.T) {
			root := fixture(t)
			appendPaired := func(name, extra string) {
				b, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				put(t, root, name, string(b)+"\n"+extra+"\n")
				pairFixtureDocument(t, root, name)
			}
			appendPaired("ARCHITECTURE.md", fake)
			appendPaired("AGENTS.md", "[phantom](ARCHITECTURE.md#phantom)")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004") {
				t.Fatalf("invalid fragment accepted: %v", err)
			}
			// A real heading after the same hidden example restores navigation.
			appendPaired("ARCHITECTURE.md", "## Phantom")
			if err := docsCheck(root); err != nil {
				t.Fatal(err)
			}
			appendPaired("AGENTS.md", "[duplicate](ARCHITECTURE.md#phantom-1)")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004") {
				t.Fatalf("invalid fragment accepted: %v", err)
			}
			appendPaired("ARCHITECTURE.md", "## Phantom")
			if err := docsCheck(root); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFragmentsPreserveInlineCodeHeadingText(t *testing.T) {
	for _, tc := range []struct{ heading, valid, invalid string }{
		{"## Configure `foo`", "configure-foo", "configure-code"},
		{"## Configure `foo`", "configure-foo", "configure"},
		{"## Use ``foo`bar`` now", "use-foobar-now", "use-code-now"},
		{"## Configure `foo` <!-- hidden -->", "configure-foo", "configure-code"},
	} {
		t.Run(tc.valid+"/"+tc.invalid, func(t *testing.T) {
			root := fixture(t)
			appendPaired := func(name, extra string) {
				b, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				put(t, root, name, string(b)+"\n"+extra+"\n")
				pairFixtureDocument(t, root, name)
			}
			appendPaired("ARCHITECTURE.md", tc.heading+"\n"+tc.heading)
			appendPaired("AGENTS.md", "[valid](ARCHITECTURE.md#"+tc.valid+")\n[duplicate](ARCHITECTURE.md#"+tc.valid+"-1)")
			if err := docsCheck(root); err != nil {
				t.Fatalf("rendered inline-code anchor rejected: %v", err)
			}
			appendPaired("AGENTS.md", "[invalid](ARCHITECTURE.md#"+tc.invalid+")")
			if err := docsCheck(root); err == nil || !strings.Contains(err.Error(), "DOC-004") {
				t.Fatalf("nonexistent substituted anchor accepted: %v", err)
			}
		})
	}
}
