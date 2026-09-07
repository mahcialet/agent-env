// repoctl is the portable, intentionally small repository verification harness.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "AGENTENV-USAGE-001: use repoctl doctor|check|test-unit|test-integration|docs-check|generated-check|generate|arch-check")
		return 2
	}
	root, err := repositoryRoot()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err = execute(root, args[0], out, errOut); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("AGENTENV-ROOT-001: go.mod not found; run inside the repository")
		}
		dir = parent
	}
}

func command(root string, out, errOut io.Writer, env []string, name string, args ...string) error {
	fmt.Fprintf(out, "> %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = root, out, errOut
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AGENTENV-CMD-001: %s failed: %w; repair the reported failure and rerun", name, err)
	}
	return nil
}

func execute(root, action string, out, errOut io.Writer) error {
	switch action {
	case "doctor":
		for _, tool := range []string{"go", "gofmt", "git"} {
			p, err := exec.LookPath(tool)
			if err != nil {
				return fmt.Errorf("AGENTENV-DOCTOR-001: missing %s; install it and add it to PATH", tool)
			}
			fmt.Fprintf(out, "%s: %s\n", tool, p)
		}
		fmt.Fprintln(out, "Docker is required only by explicit test-integration; that command verifies the daemon and Compose plugin.")
		return command(root, out, errOut, nil, "go", "version")
	case "test-unit":
		return command(root, out, errOut, nil, "go", "test", "./...")
	case "test-integration":
		if err := command(root, out, errOut, nil, "docker", "info"); err != nil {
			return err
		}
		if err := command(root, out, errOut, nil, "docker", "compose", "version"); err != nil {
			return err
		}
		return command(root, out, errOut, []string{"AGENT_ENV_INTEGRATION=1"}, "go", "test", "-tags=integration", "-count=1", "-v", "./...")
	case "docs-check":
		return docsCheck(root)
	case "generated-check":
		return generatedCheck(root, false)
	case "generate":
		return generatedCheck(root, true)
	case "arch-check":
		return archCheck(root)
	case "format-check":
		return formatCheck(root)
	case "check":
		for _, step := range []string{"format-check", "test-unit", "vet", "docs-check", "generated-check", "arch-check"} {
			fmt.Fprintln(out, "== "+step+" ==")
			var err error
			if step == "vet" {
				err = command(root, out, errOut, nil, "go", "vet", "./...")
			} else {
				err = execute(root, step, out, errOut)
			}
			if err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("AGENTENV-USAGE-001: unknown command %q; use check, doctor, or a documented check command", action)
	}
}

func files(root string) ([]string, error) {
	var result []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		result = append(result, path)
		return nil
	})
	sort.Strings(result)
	return result, err
}

func formatCheck(root string) error {
	paths, err := files(root)
	if err != nil {
		return err
	}
	for _, p := range paths {
		if filepath.Ext(p) != ".go" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		formatted, err := format.Source(b)
		if err != nil {
			return err
		}
		// Git for Windows can check out text files with CRLF. Newline encoding
		// does not change Go formatting; all other gofmt differences still fail.
		if !bytes.Equal(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), formatted) {
			return fmt.Errorf("AGENTENV-FMT-001: %s needs formatting; run gofmt -w on this file", p)
		}
	}
	return nil
}

var markdownLinks = regexp.MustCompile(`!?\[[^\]]*\]\((<[^>]+>|[^)]+)\)`)
var referenceLinks = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]:\s*(<[^>]+>|\S+)`)
var codePaths = regexp.MustCompile("`([^`\\n]+)`")
var planSections = []string{"Purpose / Big Picture", "Progress", "Surprises & Discoveries", "Decision Log", "Outcomes & Retrospective", "Context and Orientation", "Plan of Work", "Concrete Steps", "Validation and Acceptance", "Idempotence and Recovery", "Artifacts and Notes", "Interfaces and Dependencies"}

func links(data string) []string {
	var result []string
	// Examples in fenced code blocks are not Markdown links.
	var prose strings.Builder
	fenced := false
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~") {
			fenced = !fenced
			continue
		}
		if !fenced {
			prose.WriteString(line + "\n")
		}
	}
	for _, re := range []*regexp.Regexp{markdownLinks, referenceLinks} {
		for _, m := range re.FindAllStringSubmatch(prose.String(), -1) {
			target := strings.TrimSpace(m[1])
			if strings.HasPrefix(target, "<") {
				target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
			} else if i := strings.Index(target, ` "`); i >= 0 {
				target = target[:i]
			}
			result = append(result, target)
		}
	}
	return result
}

func localTarget(root, source, target string) (string, string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", "", err
	}
	if u.IsAbs() || u.Host != "" {
		return "", "", nil
	}
	p := u.Path
	if p == "" {
		return source, u.Fragment, nil
	}
	if strings.HasPrefix(p, "/") {
		return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(p, "/"))), u.Fragment, nil
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(p))), u.Fragment, nil
}

func headingAnchor(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out strings.Builder
	for _, r := range s {
		if r == ' ' {
			out.WriteByte('-')
		} else if r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func docsCheck(root string) error {
	paths, err := documentationFiles(root)
	if err != nil {
		return err
	}
	return errors.Join(documentStructureCheck(root, paths), translationCheck(root, paths))
}

func documentStructureCheck(root string, paths []string) error {
	agents := filepath.Join(root, "AGENTS.md")
	b, err := os.ReadFile(agents)
	if err != nil {
		return fmt.Errorf("AGENTENV-DOC-002: AGENTS.md missing; restore the repository entry point: %w", err)
	}
	if len(strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")) > 150 {
		return fmt.Errorf("AGENTENV-DOC-002: AGENTS.md exceeds 150 lines; move detail into indexed documents")
	}
	for _, match := range codePaths.FindAllStringSubmatch(string(b), -1) {
		p := match[1]
		if strings.ContainsAny(p, " \t*<>|") || strings.HasPrefix(p, "--") || p == ".agent-env.yaml" {
			continue
		}
		first, _, _ := strings.Cut(strings.TrimPrefix(p, "./"), "/")
		_, topErr := os.Stat(filepath.Join(root, first))
		if strings.HasSuffix(p, ".md") || p == "go.mod" || strings.HasSuffix(p, "/") || strings.Contains(p, "/") && topErr == nil {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
				return fmt.Errorf("AGENTENV-DOC-003: AGENTS.md references missing path %s; fix the path or restore it", p)
			}
		}
	}
	for _, p := range paths {
		if filepath.Ext(p) != ".md" {
			continue
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		japanese := strings.HasSuffix(rel, ".ja.md")
		if strings.HasPrefix(rel, "docs/references/handoffs/") && !japanese {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		data := string(b)
		for _, target := range links(data) {
			resolved, fragment, err := localTarget(root, p, target)
			if err != nil {
				return fmt.Errorf("AGENTENV-DOC-004: %s invalid link %q: %w", rel, target, err)
			}
			if resolved == "" {
				continue
			}
			if _, err := os.Stat(resolved); err != nil {
				return fmt.Errorf("AGENTENV-DOC-004: %s links to missing %s; repair the link or restore its target", rel, target)
			}
			if fragment != "" && strings.HasSuffix(resolved, ".md") {
				content, err := os.ReadFile(resolved)
				if err != nil {
					return err
				}
				found := false
				counts := map[string]int{}
				for _, line := range strings.Split(string(content), "\n") {
					if !strings.HasPrefix(line, "#") {
						continue
					}
					a := headingAnchor(strings.TrimLeft(line, "#"))
					base := a
					if counts[base] > 0 {
						a += "-" + strconv.Itoa(counts[base])
					}
					counts[base]++
					if a == fragment {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("AGENTENV-DOC-004: %s links to missing heading %s; update the fragment", rel, target)
				}
			}
		}
		indexed := strings.HasPrefix(rel, "docs/design-docs/") || strings.HasPrefix(rel, "docs/product-specs/") || strings.HasPrefix(rel, "docs/adr/")
		plan := strings.HasPrefix(rel, "docs/exec-plans/active/") || strings.HasPrefix(rel, "docs/exec-plans/completed/")
		indexName := "index.md"
		if japanese {
			indexName = "index.ja.md"
		}
		if (indexed || plan) && filepath.Base(p) != indexName {
			if err := metadataCheck(rel, data); err != nil {
				return err
			}
		}
		if indexed && filepath.Base(p) != indexName {
			index := filepath.Join(filepath.Dir(p), indexName)
			ib, err := os.ReadFile(index)
			if err != nil {
				return fmt.Errorf("AGENTENV-DOC-001: %s has no local %s; add an index with a link to this document", rel, indexName)
			}
			found := false
			for _, target := range links(string(ib)) {
				resolved, _, _ := localTarget(root, index, target)
				if resolved == p {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("AGENTENV-DOC-001: %s is not linked from its local %s; add an indexed description or move it to an archival directory", rel, indexName)
			}
		}
		if strings.HasPrefix(rel, "docs/exec-plans/active/") && !japanese {
			for _, section := range planSections {
				found := false
				for _, line := range strings.Split(data, "\n") {
					if strings.HasPrefix(line, "## ") && strings.TrimSpace(strings.TrimPrefix(line, "## ")) == section {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("AGENTENV-DOC-005: %s is missing section %q; restore the mandatory living-plan section", rel, section)
				}
			}
		}
	}
	return nil
}

func metadataCheck(path, data string) error {
	lines := strings.Split(data, "\n")
	fields := map[string]string{}
	closed := false
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) == "---" {
				closed = true
				break
			}
			key, value, ok := strings.Cut(line, ":")
			if ok {
				if _, exists := fields[key]; exists {
					return fmt.Errorf("AGENTENV-DOC-006: %s repeats metadata %s; keep one authoritative value", path, key)
				}
				fields[key] = strings.Trim(strings.TrimSpace(value), "\"'")
			}
		}
	}
	validStatus := map[string]bool{"draft": true, "active": true, "accepted": true, "superseded": true, "completed": true}
	_, dateErr := time.Parse("2006-01-02", fields["last_verified"])
	if !closed || !validStatus[fields["status"]] || fields["owner"] == "" || dateErr != nil {
		return fmt.Errorf("AGENTENV-DOC-006: %s needs front matter with valid status, nonempty owner, and last_verified: YYYY-MM-DD", path)
	}
	return nil
}

func generatedSchema(root string) ([]byte, bool, error) {
	paths, err := files(root)
	if err != nil {
		return nil, false, err
	}
	var out strings.Builder
	out.WriteString("# Database schema\n\nGenerated from SQL migrations. Do not edit; run `go run ./tools/repoctl generate`.\n")
	found := false
	for _, p := range paths {
		if filepath.Ext(p) != ".sql" || filepath.Base(filepath.Dir(p)) != "migrations" {
			continue
		}
		found = true
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, false, err
		}
		rel, _ := filepath.Rel(root, p)
		fmt.Fprintf(&out, "\n## `%s`\n\n```sql\n%s\n```\n", filepath.ToSlash(rel), strings.TrimSpace(strings.ReplaceAll(string(b), "\r\n", "\n")))
	}
	return []byte(out.String()), found, nil
}

func generatedCheck(root string, write bool) error {
	expected, exists, err := generatedSchema(root)
	if err != nil || !exists {
		return err
	}
	p := filepath.Join(root, "docs", "generated", "db-schema.md")
	if write {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		return os.WriteFile(p, expected, 0644)
	}
	actual, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n")), expected) {
		return fmt.Errorf("AGENTENV-GEN-001: docs/generated/db-schema.md differs from SQL migrations; run go run ./tools/repoctl generate and review the diff")
	}
	return nil
}

func archCheck(root string) error {
	paths, err := files(root)
	if err != nil {
		return err
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	module := ""
	for _, line := range strings.Split(string(mod), "\n") {
		if strings.HasPrefix(line, "module ") {
			module = strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	for _, p := range paths {
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			continue
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		rel = filepath.ToSlash(rel)
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			dependency, _ := strconv.Unquote(imp.Path.Value)
			local := strings.TrimPrefix(dependency, module+"/")
			bad := false
			switch {
			case rel == "internal/domain" || strings.HasPrefix(rel, "internal/domain/"):
				inDomain := local == "internal/domain" || strings.HasPrefix(local, "internal/domain/")
				first, _, _ := strings.Cut(dependency, "/")
				external := strings.Contains(first, ".") && !strings.HasPrefix(dependency, module+"/")
				bad = strings.HasPrefix(local, "internal/") && !inDomain || dependency == "database/sql" || external
			case rel == "internal/app" || strings.HasPrefix(rel, "internal/app/"):
				bad = local == "internal/cli" || strings.HasPrefix(local, "internal/cli/")
			case strings.HasPrefix(rel, "internal/runtime/"):
				bad = local == "internal/cli" || strings.HasPrefix(local, "internal/cli/")
				if strings.HasPrefix(local, "internal/runtime/") {
					own := strings.Split(rel, "/")[2]
					other := strings.Split(local, "/")[2]
					bad = bad || own != other
				}
			case strings.HasPrefix(rel, "internal/store/"):
				bad = local == "internal/app" || strings.HasPrefix(local, "internal/app/") || local == "internal/cli" || strings.HasPrefix(local, "internal/cli/") || strings.HasPrefix(local, "internal/runtime/")
			}
			if bad {
				return fmt.Errorf("AGENTENV-ARCH-002: %s imports %s; move orchestration to app and external dependencies behind domain/app interfaces", rel, dependency)
			}
		}
	}
	return nil
}
