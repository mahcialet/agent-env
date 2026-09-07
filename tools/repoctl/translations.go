package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Keep existing link checks outside docs, but never inherit source-tree scan
// exclusions such as vendor, node_modules or hidden folders inside docs.
func documentationFiles(root string) ([]string, error) {
	paths, err := files(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, p := range paths {
		seen[p] = true
	}
	docs := filepath.Join(root, "docs")
	err = filepath.WalkDir(docs, func(p string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) && p == docs {
			return nil
		}
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(p, ".md") && !seen[p] {
			paths = append(paths, p)
			seen[p] = true
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func translationScope(rel string) bool {
	return strings.HasSuffix(rel, ".md") && (!strings.Contains(rel, "/") || strings.HasPrefix(rel, "docs/"))
}

func translationDigest(source []byte) string {
	sum := sha256.Sum256(bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n")))
	return hex.EncodeToString(sum[:])
}

// Translation checks never update documents: a source hash records a human
// reviewed translation state, not merely the last time the checker ran.
func translationCheck(root string, paths []string) error {
	documents := map[string][]byte{}
	var problems []error
	for _, file := range paths {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !translationScope(rel) {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		documents[rel] = data
	}
	exceptions, err := translationExceptions(root, documents)
	if err != nil {
		problems = append(problems, err)
	}
	names := make([]string, 0, len(documents))
	for name := range documents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasSuffix(name, ".ja.md") {
			if !exceptions[name] {
				sibling := strings.TrimSuffix(name, ".md") + ".ja.md"
				if _, ok := documents[sibling]; !ok {
					problems = append(problems, fmt.Errorf("AGENTENV-DOC-007: %s is missing Japanese translation %s; add the maintained sibling", name, sibling))
				}
			}
			continue
		}
		canonical := strings.TrimSuffix(name, ".ja.md") + ".md"
		source, exists := documents[canonical]
		if !exists || strings.HasSuffix(canonical, ".ja.md") {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-008: %s has no canonical English sibling %s; restore the source or remove the orphan translation", name, canonical))
			continue
		}
		fields, err := translationMetadata(documents[name])
		if err != nil {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-008: %s translation metadata: %w", name, err))
			continue
		}
		if fields["translation_of"] != canonical {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-008: %s translation_of must be exactly %s", name, canonical))
		}
		digest := fields["source_sha256"]
		decoded, hashErr := hex.DecodeString(digest)
		if hashErr != nil || len(decoded) != sha256.Size || digest != strings.ToLower(digest) {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-008: %s source_sha256 must contain 64 lowercase hexadecimal digits", name))
		} else if digest != translationDigest(source) {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-009: %s is stale relative to %s; update the translation and source_sha256 together", name, canonical))
		}
		canonicalFields, err := translationMetadata(source)
		if err != nil {
			// Documents without canonical front matter (for example AGENTS.md)
			// need only translation provenance on their Japanese sibling.
			if bytes.HasPrefix(source, []byte("---\n")) || bytes.HasPrefix(source, []byte("---\r\n")) {
				problems = append(problems, fmt.Errorf("AGENTENV-DOC-008: %s canonical metadata: %w", canonical, err))
			}
			continue
		}
		for _, key := range []string{"status", "owner", "last_verified"} {
			if value, ok := canonicalFields[key]; ok && fields[key] != value {
				problems = append(problems, fmt.Errorf("AGENTENV-DOC-011: %s metadata %s must match canonical %s", name, key, canonical))
			}
		}
	}
	return errors.Join(problems...)
}

func translationMetadata(data []byte) (map[string]string, error) {
	lines := strings.Split(string(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, errors.New("front matter is required")
	}
	fields := map[string]string{}
	for _, line := range lines[1:] {
		if line == "---" {
			return fields, nil
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid front matter line %q", line)
		}
		key = strings.TrimSpace(key)
		if _, ok := fields[key]; ok {
			return nil, fmt.Errorf("duplicate field %s", key)
		}
		scalar, err := translationScalar(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid scalar for %s: %w", key, err)
		}
		fields[key] = scalar
	}
	return nil, errors.New("front matter closing delimiter is required")
}

// Metadata deliberately supports single-line plain or quoted scalar strings,
// rather than silently accepting malformed YAML by trimming arbitrary quotes.
func translationScalar(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		if len(value) < 2 || value[len(value)-1] != quote {
			return "", errors.New("unterminated quoted value")
		}
		if quote == '"' {
			// JSON string syntax is a deliberately strict subset of YAML's quoted
			// scalar syntax and rejects trailing text or malformed escapes.
			var decoded string
			if err := json.Unmarshal([]byte(value), &decoded); err != nil {
				return "", err
			}
			return decoded, nil
		}
		inner := value[1 : len(value)-1]
		if strings.Contains(strings.ReplaceAll(inner, "''", ""), "'") {
			return "", errors.New("unescaped quote in single-quoted value")
		}
		return strings.ReplaceAll(inner, "''", "'"), nil
	}
	if strings.ContainsAny(value, "\"'\r\n\t") || strings.ContainsAny(value[:1], "!&*{[|>@`#") || strings.Contains(value, " #") || strings.Contains(value, ": ") {
		return "", errors.New("use a plain single-line value or a balanced quoted string")
	}
	return value, nil
}

func translationExceptions(root string, documents map[string][]byte) (map[string]bool, error) {
	allowed := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(root, "docs", "translation-exceptions.json"))
	if errors.Is(err, os.ErrNotExist) {
		return allowed, nil
	}
	if err != nil {
		return allowed, err
	}
	type exemption struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	}
	var document struct {
		Exceptions *[]exemption `json:"exceptions"`
	}
	if err := uniqueJSONFields(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return allowed, fmt.Errorf("AGENTENV-DOC-010: invalid docs/translation-exceptions.json: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return allowed, fmt.Errorf("AGENTENV-DOC-010: invalid docs/translation-exceptions.json: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return allowed, errors.New("AGENTENV-DOC-010: docs/translation-exceptions.json must contain exactly one JSON object")
	}
	if document.Exceptions == nil {
		return allowed, errors.New("AGENTENV-DOC-010: docs/translation-exceptions.json requires an exceptions array")
	}
	var problems []error
	seen := map[string]bool{}
	for _, entry := range *document.Exceptions {
		p := entry.Path
		category := strings.HasPrefix(p, "docs/generated/") || strings.HasPrefix(p, "docs/references/handoffs/") || strings.HasPrefix(p, "docs/exec-plans/completed/")
		_, exists := documents[p]
		if !category || path.Clean(p) != p || strings.ContainsAny(p, "\\*?[]") || !strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".ja.md") || !exists || strings.TrimSpace(entry.Reason) == "" || seen[p] {
			problems = append(problems, fmt.Errorf("AGENTENV-DOC-010: invalid translation exception %q; use a unique existing exact English Markdown path in generated, archived handoff, or completed-plan categories with a nonempty reason", p))
			continue
		}
		seen[p] = true
		allowed[p] = true
	}
	return allowed, errors.Join(problems...)
}

// encoding/json otherwise accepts duplicate policy keys using the last value.
// Reject ambiguity both in the top-level list and individual exception records.
func uniqueJSONFields(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	switch token {
	case json.Delim('{'):
		seen := map[string]bool{}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := token.(string)
			if !ok {
				return errors.New("JSON object field name must be a string")
			}
			if key != "exceptions" && key != "path" && key != "reason" {
				return fmt.Errorf("unknown exception schema field %s; keys are case-sensitive", strconv.Quote(key))
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = true
			if err := uniqueJSONFields(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case json.Delim('['):
		for decoder.More() {
			if err := uniqueJSONFields(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	}
	return nil
}
