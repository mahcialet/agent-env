// Package remotesource transports committed Git inputs without client paths.
package remotesource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/source/gitcli"
	"github.com/mahcialet/agent-env/internal/stack"
)

type Source struct {
	Alias        string `json:"alias"`
	Commit       string `json:"commit"`
	RepositoryID string `json:"repository_id"`
	BlobDigest   string `json:"blob_digest"`
}
type Package struct {
	AndroidSlots         int             `json:"android_slots"`
	RepositoryID         string          `json:"repository_id"`
	Manifest             json.RawMessage `json:"manifest"`
	ManifestDigest       string          `json:"manifest_digest"`
	ManifestCommit       string          `json:"manifest_commit"`
	ManifestPath         string          `json:"manifest_path"`
	ManifestBlobDigest   string          `json:"manifest_blob_digest"`
	SourceSetDigest      string          `json:"source_set_digest"`
	Stack                string          `json:"stack"`
	Sources              []Source        `json:"sources"`
	RequiredCapabilities []string        `json:"required_capabilities"`
	PlanDigest           string          `json:"plan_digest"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func packageDigest(p Package) string {
	p.PlanDigest = ""
	m, err := parseCanonical(p.Manifest)
	if err != nil {
		return ""
	}
	p.Manifest, err = config.CanonicalJSON(m)
	if err != nil {
		return ""
	}
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return digest(b)
}
func commitOK(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func sourceSet(p Package) string {
	var s []domain.Source
	for _, v := range p.Sources {
		s = append(s, domain.Source{Alias: v.Alias, Commit: v.Commit, RepositoryID: v.RepositoryID})
	}
	return domain.SourceDigest(s)
}
func required(m *config.Manifest, components []string) []string {
	set := map[string]bool{"git": true}
	for _, name := range components {
		c := m.Components[name]
		rt := m.Runtimes[c.Runtime]
		switch rt.Type {
		case "compose":
			if rt.Provider == "podman-compose" {
				set["compose.podman"] = true
			} else {
				set["compose.docker"] = true
			}
		case "process":
			set["persistent-process"] = true
		case "android-emulator":
			set["android-emulator"] = true
		}
		if c.Application != "" && m.Applications[c.Application].Type == "flutter-android" {
			set["flutter-android"] = true
		}
	}
	selected := map[string]bool{}
	for _, name := range components {
		selected[m.Components[name].Runtime] = true
	}
	for _, b := range m.Browsers {
		if selected[b.Runtime] {
			set["browser-cdp"] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Validate checks transport metadata without extracting or starting resources.
func Validate(p Package) error {
	if !blobstore.ValidDigest(p.PlanDigest) || packageDigest(p) != p.PlanDigest {
		return errors.New("remote plan digest mismatch")
	}
	if len(p.Manifest) == 0 || len(p.Manifest) > 4<<20 {
		return errors.New("remote manifest size limit exceeded")
	}
	m, err := parseCanonical(p.Manifest)
	if err != nil {
		return err
	}
	canon, err := config.CanonicalJSON(m)
	if err != nil {
		return err
	}
	if digest(canon) != p.ManifestDigest {
		return errors.New("remote manifest is not canonical or digest mismatched")
	}
	if !strings.HasPrefix(p.RepositoryID, "git-sha256:") || !blobstore.ValidDigest(strings.TrimPrefix(p.RepositoryID, "git-sha256:")) {
		return errors.New("invalid control repository identity")
	}
	if !commitOK(p.ManifestCommit) || !blobstore.ValidDigest(p.ManifestBlobDigest) || !safeRelative(p.ManifestPath) {
		return errors.New("invalid manifest commit")
	}
	components, err := stack.Resolve(m, p.Stack)
	if err != nil {
		return err
	}
	used := map[string]bool{}
	for alias := range m.Sources {
		used[alias] = true
	}
	portableAliases := map[string]bool{}
	for alias, s := range m.Sources {
		key := strings.ToLower(alias)
		if !portableAlias(alias) || portableAliases[key] {
			return errors.New("nonportable source aliases")
		}
		portableAliases[key] = true
		if s.Repository != "sources/"+alias {
			return errors.New("remote manifest contains nonportable repository path")
		}
	}
	seen := map[string]bool{}
	last := ""
	for _, s := range p.Sources {
		if !used[s.Alias] || seen[s.Alias] || s.Alias <= last {
			return errors.New("remote source closure or order mismatch")
		}
		seen[s.Alias] = true
		last = s.Alias
		if !commitOK(s.Commit) || m.Sources[s.Alias].DefaultRef != s.Commit || !blobstore.ValidDigest(s.BlobDigest) || !strings.HasPrefix(s.RepositoryID, "git-sha256:") || !blobstore.ValidDigest(strings.TrimPrefix(s.RepositoryID, "git-sha256:")) {
			return errors.New("invalid remote source identity")
		}
	}
	if len(seen) != len(used) || sourceSet(p) != p.SourceSetDigest {
		return errors.New("remote source set mismatch")
	}
	if p.AndroidSlots != androidSlots(m, components) {
		return errors.New("remote Android slot count mismatch")
	}
	if !equalStrings(required(m, components), p.RequiredCapabilities) {
		return errors.New("remote capabilities mismatch")
	}
	return nil
}
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// git runs only local operations, disabling ambient Git transport/config overrides.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	if err := paths.ValidateExecutionDirectory(dir); err != nil {
		return "", err
	}
	base := []string{"-c", "protocol.allow=never", "-c", "protocol.file.allow=always", "-c", "core.hooksPath=" + os.DevNull, "-c", "core.longpaths=true"}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.Dir = dir
	if err := execx.ValidateNativeExecutable(cmd.Path, dir); err != nil {
		return "", err
	}
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		if !strings.HasPrefix(strings.ToUpper(k), "GIT_") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "GIT_NO_REPLACE_OBJECTS=1", "GIT_LFS_SKIP_SMUDGE=1", "GIT_NO_LAZY_FETCH=1")
	var out, stderr capped
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if out.overflow || stderr.overflow {
		return "", errors.New("Git metadata output limit exceeded")
	}
	if err != nil {
		phase := "command"
		if len(args) > 0 {
			switch args[0] {
			case "init", "fetch", "clone", "bundle", "rev-parse", "fsck", "show", "update-ref", "worktree", "status", "config", "grep", "ls-tree":
				phase = args[0]
			}
		}
		// Keep the failure actionable without logging argv or the environment.
		// Redact before bounding the text so a truncated credential cannot leak.
		diagnostic := evidence.RedactString(stderr.String(), evidence.InheritedSecrets())
		diagnostic = strings.Map(func(r rune) rune {
			if r < 32 && r != '\n' && r != '\t' || r == 127 {
				return -1
			}
			return r
		}, diagnostic)
		if len(diagnostic) > 4096 {
			diagnostic = diagnostic[:4096] + " [truncated]"
		}
		return out.String(), fmt.Errorf("local Git %s failed: %w: %s", phase, err, strings.TrimSpace(diagnostic))
	}
	return out.String(), nil
}

type capped struct {
	// Do not embed Buffer: its promoted ReadFrom lets os/exec's io.Copy
	// bypass Write and therefore bypass the output limit.
	buffer   bytes.Buffer
	overflow bool
}

func (c *capped) String() string { return c.buffer.String() }

func (c *capped) Write(p []byte) (int, error) {
	n := len(p)
	left := (16 << 20) - c.buffer.Len()
	if len(p) > left {
		p = p[:left]
		c.overflow = true
	}
	_, _ = c.buffer.Write(p)
	return n, nil
}
func supported(ctx context.Context, repo, commit string) error {
	shallow, err := git(ctx, repo, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return err
	}
	if strings.TrimSpace(shallow) != "false" {
		return errors.New("shallow repositories are unsupported")
	}
	promisor, err := git(ctx, repo, "config", "--local", "--get-regexp", `^remote\..*\.promisor$`)
	if err == nil && strings.Contains(strings.ToLower(promisor), "true") {
		return errors.New("partial repositories are unsupported")
	}
	tree, err := git(ctx, repo, "ls-tree", "-r", "-z", commit)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(tree, "\x00") {
		if strings.HasPrefix(line, "160000 ") {
			return errors.New("Git submodules are unsupported")
		}
	}
	for _, args := range [][]string{
		{"grep", "-I", "-l", "-e", `filter[[:space:]]*=[[:space:]]*lfs`, commit, "--", ".gitattributes", ":(glob)**/.gitattributes"},
		{"grep", "-I", "-l", "-e", "^version https://git-lfs.github.com/spec/v1$", commit, "--"},
	} {
		_, err = git(ctx, repo, args...)
		if err == nil {
			return errors.New("Git LFS sources are unsupported")
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return err
		}
	}

	return nil
}
func Build(ctx context.Context, o app.PlanOptions, cas *blobstore.Store) (Package, error) {
	var p Package
	if err := paths.ValidateExecutionDirectory(o.Repository); err != nil {
		return p, err
	}
	plan, err := app.BuildPlan(ctx, o, &Provider{git: gitcli.Client{Runner: localRunner{}}})
	if err != nil {
		return p, err
	}
	if plan.ManifestModified || !commitOK(plan.ManifestCommit) {
		return p, errors.New("remote manifest must be committed and unmodified")
	}
	m, err := config.Load(plan.ManifestPath)
	if err != nil {
		return p, err
	}
	p.Stack = plan.Stack
	p.ManifestCommit = plan.ManifestCommit
	for alias, s := range m.Sources {
		s.Repository = "sources/" + alias
		m.Sources[alias] = s
	}
	if err = paths.ValidateExecutionDirectory(filepath.Join(os.TempDir(), "agent-env-bundles-4294967295")); err != nil {
		return p, err
	}
	temp, err := os.MkdirTemp("", "agent-env-bundles-")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(temp)
	if err = paths.ValidateExecutionDirectory(temp); err != nil {
		return p, err
	}
	for i, s := range plan.Sources {
		if err = supported(ctx, s.RepositoryPath, s.Commit); err != nil {
			return p, err
		}
		bd, e := makeBundle(ctx, s.RepositoryPath, s.Commit, temp, fmt.Sprintf("source-%d", i), cas)
		if e != nil {
			return p, e
		}

		p.Sources = append(p.Sources, Source{s.Alias, s.Commit, "git-sha256:" + digest([]byte(s.RepositoryID)), bd})
		v := m.Sources[s.Alias]
		v.DefaultRef = s.Commit
		m.Sources[s.Alias] = v
	}
	sort.Slice(p.Sources, func(i, j int) bool { return p.Sources[i].Alias < p.Sources[j].Alias })
	p.Manifest, err = config.CanonicalJSON(m)
	if err != nil {
		return p, err
	}
	control, e := git(ctx, filepath.Dir(plan.ManifestPath), "rev-parse", "--show-toplevel")
	if e != nil {
		return p, e
	}
	control = strings.TrimSpace(control)
	identity, e := (&Provider{git: gitcli.Client{Runner: localRunner{}}}).Resolve(ctx, control, p.ManifestCommit)
	if e != nil {
		return p, e
	}
	p.RepositoryID = "git-sha256:" + digest([]byte(identity.RepositoryID))
	relative, e := filepath.Rel(control, plan.ManifestPath)
	if e != nil {
		return p, e
	}
	p.ManifestPath = filepath.ToSlash(relative)
	p.ManifestBlobDigest, err = makeBundle(ctx, control, p.ManifestCommit, temp, "control", cas)
	if err != nil {
		return p, err
	}
	original, e := git(ctx, control, "show", p.ManifestCommit+":"+p.ManifestPath)
	if e != nil {
		return p, e
	}
	normalized, e := normalizeCommitted([]byte(original), p)
	if e != nil || !bytes.Equal(normalized, p.Manifest) {
		return p, errors.New("manifest changed during packaging or differs from committed authority")
	}
	p.ManifestDigest = digest(p.Manifest)
	p.SourceSetDigest = sourceSet(p)
	names, _ := stack.Resolve(m, p.Stack)
	p.RequiredCapabilities = required(m, names)
	p.AndroidSlots = androidSlots(m, names)
	p.PlanDigest = packageDigest(p)
	return p, Validate(p)
}

func privateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe source materialization directory")
	}
	return os.Chmod(path, 0700)
}
func Materialize(ctx context.Context, p Package, home string, cas *blobstore.Store) (app.PlanOptions, app.SourceProvider, error) {
	var options app.PlanOptions
	if err := Validate(p); err != nil {
		return options, nil, err
	}
	m, _ := parseCanonical(p.Manifest)
	p.Manifest, _ = config.CanonicalJSON(m)
	home, err := filepath.Abs(home)
	if err != nil {
		return options, nil, err
	}
	// Validate all future Git working directories before creating any source
	// state. MkdirTemp currently uses a uint32 suffix; the final digest directory
	// is longer, but both layouts are checked explicitly and the actual temp
	// directory is checked again below.
	root := filepath.Join(home, "remote-sources")
	for _, layout := range []string{filepath.Join(root, ".incoming-4294967295"), filepath.Join(root, p.PlanDigest)} {
		candidates := []string{layout, filepath.Join(layout, "control")}
		for _, source := range p.Sources {
			candidates = append(candidates, filepath.Join(layout, "sources", source.Alias))
		}
		for _, candidate := range candidates {
			if err = paths.ValidateExecutionDirectory(candidate); err != nil {
				return options, nil, err
			}
		}
	}
	if err = privateDirectory(home); err != nil {
		return options, nil, err
	}
	// Host-provided ancestors may be aliases (for example macOS /var).
	// Resolve this trusted root once so provider identities match Git's
	// canonical paths; managed descendants still reject symlinks below.
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return options, nil, err
	}
	root = filepath.Join(home, "remote-sources")
	if err = privateDirectory(root); err != nil {
		return options, nil, err
	}
	dest := filepath.Join(root, p.PlanDigest)
	if _, err = os.Lstat(dest); err == nil {
		return validateMaterialized(ctx, p, dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return options, nil, err
	}
	temp, err := os.MkdirTemp(root, ".incoming-")
	if err != nil {
		return options, nil, err
	}
	defer os.RemoveAll(temp)
	if err = paths.ValidateExecutionDirectory(temp); err != nil {
		return options, nil, err
	}
	control := filepath.Join(temp, "control")
	if err = cloneBundle(ctx, p.ManifestBlobDigest, p.ManifestCommit, temp, "control", control, cas); err != nil {
		return options, nil, err
	}
	original, err := git(ctx, control, "show", p.ManifestCommit+":"+p.ManifestPath)
	if err != nil {
		return options, nil, err
	}
	normalized, err := normalizeCommitted([]byte(original), p)
	if err != nil || !bytes.Equal(normalized, p.Manifest) {
		return options, nil, errors.New("normalized manifest differs from committed authority")
	}

	if err = privateDirectory(filepath.Join(temp, "sources")); err != nil {
		return options, nil, err
	}
	for _, s := range p.Sources {
		if err = cloneBundle(ctx, s.BlobDigest, s.Commit, temp, s.Alias, filepath.Join(temp, "sources", s.Alias), cas); err != nil {
			return options, nil, err
		}
	}

	document, err := manifestDocument(p.Manifest)
	if err != nil {
		return options, nil, err
	}
	if err = os.WriteFile(filepath.Join(temp, "manifest.json"), document, 0600); err != nil {
		return options, nil, err
	}
	if err = os.Rename(temp, dest); err != nil {
		if _, e := os.Lstat(dest); e != nil {
			return options, nil, err
		}
	}
	return validateMaterialized(ctx, p, dest)
}

// validateMaterialized rechecks the owned digest directory before using its
// existing repositories. It neither repairs unexpected paths nor opens bundles.
func validateMaterialized(ctx context.Context, p Package, dest string) (app.PlanOptions, app.SourceProvider, error) {
	var options app.PlanOptions
	if err := materializedDirectory(dest); err != nil {
		return options, nil, err
	}
	if err := materializedDirectory(filepath.Join(dest, "sources")); err != nil {
		return options, nil, err
	}
	control := filepath.Join(dest, "control")
	if err := materializedRepository(ctx, control, p.ManifestCommit); err != nil {
		return options, nil, err
	}
	original, err := git(ctx, control, "show", p.ManifestCommit+":"+p.ManifestPath)
	if err != nil {
		return options, nil, err
	}
	normalized, err := normalizeCommitted([]byte(original), p)
	if err != nil || !bytes.Equal(normalized, p.Manifest) {
		return options, nil, errors.New("cached manifest differs from committed authority")
	}
	document, err := manifestDocument(p.Manifest)
	if err != nil {
		return options, nil, err
	}
	manifestInfo, e := os.Lstat(filepath.Join(dest, "manifest.json"))
	if e != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Size() > 4<<20 {
		return options, nil, errors.New("unsafe materialized manifest")
	}
	raw, err := os.ReadFile(filepath.Join(dest, "manifest.json"))
	if err != nil || !bytes.Equal(raw, document) {
		return options, nil, errors.New("existing materialized manifest mismatch")
	}
	provider := &Provider{git: gitcli.Client{Runner: localRunner{}}, pkg: p, repositories: map[string]string{}}
	for _, s := range p.Sources {
		repo := filepath.Join(dest, "sources", s.Alias)
		if err := materializedRepository(ctx, repo, s.Commit); err != nil {
			return options, nil, err
		}
		provider.repositories[repo] = s.RepositoryID
	}
	options = app.PlanOptions{Repository: dest, ManifestPath: "manifest.json", Stack: p.Stack}
	plan, err := app.BuildPlan(ctx, options, provider)
	if err != nil {
		return options, nil, err
	}
	if plan.ManifestDigest != p.ManifestDigest || plan.SourceSetDigest != p.SourceSetDigest || plan.Stack != p.Stack {
		return options, nil, errors.New("materialized plan identity mismatch")
	}
	return options, provider, nil
}

func materializedDirectory(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe materialized directory")
	}
	return nil
}

func materializedRepository(ctx context.Context, repo, commit string) error {
	if err := materializedDirectory(repo); err != nil {
		return err
	}
	bare, err := git(ctx, repo, "rev-parse", "--is-bare-repository")
	if err != nil {
		return err
	}
	if strings.TrimSpace(bare) != "true" {
		return errors.New("materialized repository must be bare")
	}
	common, err := git(ctx, repo, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	if filepath.Clean(strings.TrimSpace(common)) != filepath.Clean(repo) {
		return errors.New("materialized repository identity mismatch")
	}
	resolved, err := git(ctx, repo, "rev-parse", "--verify", "--end-of-options", commit+"^{commit}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(resolved) != commit {
		return errors.New("materialized commit identity mismatch")
	}
	return nil
}

// Provider converts worker-local Git identities to the immutable client identities.
type Provider struct {
	git          gitcli.Client
	pkg          Package
	repositories map[string]string
}

func (p *Provider) Origin(ctx context.Context, path string) (string, bool, error) {
	if p.repositories == nil {
		return p.git.Origin(ctx, path)
	}
	return p.pkg.ManifestCommit, false, nil
}
func (p *Provider) Resolve(ctx context.Context, repo, ref string) (domain.Source, error) {
	r, err := p.git.Resolve(ctx, repo, ref)
	if err != nil {
		return domain.Source{}, err
	}
	if p.repositories != nil {
		id, ok := p.repositories[filepath.Clean(repo)]
		if !ok {
			return domain.Source{}, errors.New("repository is outside materialized package")
		}
		r.RepositoryID = id
	}
	return domain.Source{RepositoryID: r.RepositoryID, RepositoryPath: r.RepositoryPath, RequestedRef: r.RequestedRef, Commit: r.Commit}, nil
}
func (p *Provider) native(ctx context.Context, s domain.Source) (gitcli.Worktree, error) {
	if p.repositories != nil {
		id, ok := p.repositories[filepath.Clean(s.RepositoryPath)]
		if !ok || id != s.RepositoryID {
			return gitcli.Worktree{}, errors.New("remote repository identity mismatch")
		}
	}
	r, err := p.git.Resolve(ctx, s.RepositoryPath, s.Commit)
	if err != nil {
		return gitcli.Worktree{}, err
	}
	return gitcli.Worktree{Resolved: r, Path: s.WorktreePath}, nil
}
func (p *Provider) Materialize(ctx context.Context, s domain.Source) error {
	if err := paths.ValidateExecutionDirectory(s.WorktreePath); err != nil {
		return err
	}
	w, err := p.native(ctx, s)
	if err != nil {
		return err
	}
	_, err = p.git.Materialize(ctx, w.Resolved, w.Path)
	return err
}
func (p *Provider) Inspect(ctx context.Context, s domain.Source) (app.SourceObservation, error) {
	w, err := p.native(ctx, s)
	if err != nil {
		return app.SourceObservation{}, err
	}
	o, err := p.git.Inspect(ctx, w)
	return app.SourceObservation{Exists: o.Exists, Registered: o.Registered, TrackedDirty: o.TrackedDirty, Commit: o.Commit}, err
}
func (p *Provider) Remove(ctx context.Context, s domain.Source, force bool) error {
	w, err := p.native(ctx, s)
	if err != nil {
		return err
	}
	return p.git.Remove(ctx, w, force)
}

func (p *Provider) Diff(ctx context.Context, s domain.Source) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	w, err := p.native(ctx, s)
	if err != nil {
		return "", err
	}
	observed, err := p.git.Inspect(ctx, w)
	if err != nil {
		return "", err
	}
	if !observed.Exists || !observed.Registered || observed.Commit != s.Commit {
		return "", errors.New("tracked diff requires the owned pinned worktree")
	}
	// git bounds stdout/stderr and rejects overflow instead of returning a
	// truncated patch that could incorrectly authorize destructive cleanup.
	return git(ctx, s.WorktreePath, "diff", "HEAD", "--binary", "--no-ext-diff", "--no-textconv", "--")
}
func safeRelative(s string) bool {
	if s == "" || strings.ContainsAny(s, "\\:\x00") || strings.HasPrefix(s, "/") {
		return false
	}
	for _, v := range strings.Split(s, "/") {
		if v == "" || v == "." || v == ".." {
			return false
		}
	}
	return true
}

func makeBundle(ctx context.Context, source, commit, temp, name string, cas *blobstore.Store) (string, error) {
	if err := supported(ctx, source, commit); err != nil {
		return "", err
	}
	repo := filepath.Join(temp, name+"-git")
	if err := paths.ValidateExecutionDirectory(repo); err != nil {
		return "", err
	}
	if _, err := git(ctx, "", "init", "--bare", repo); err != nil {
		return "", err
	}
	if _, err := git(ctx, repo, "fetch", "--no-tags", source, commit); err != nil {
		return "", err
	}
	if _, err := git(ctx, repo, "update-ref", "refs/heads/agent-env", commit); err != nil {
		return "", err
	}
	bundle := filepath.Join(temp, name+".bundle")
	if _, err := git(ctx, repo, "bundle", "create", bundle, "refs/heads/agent-env"); err != nil {
		return "", err
	}
	f, err := os.Open(bundle)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := cas.Put(ctx, f, "", blobstore.SourceLimit)
	return b.Digest, err
}
func cloneBundle(ctx context.Context, digest, commit, temp, name, repo string, cas *blobstore.Store) error {
	for _, directory := range []string{temp, repo} {
		if err := paths.ValidateExecutionDirectory(directory); err != nil {
			return err
		}
	}
	f, _, err := cas.Open(ctx, digest, blobstore.SourceLimit)
	if err != nil {
		return err
	}
	defer f.Close()
	bundle := filepath.Join(temp, name+"-input.bundle")
	dst, err := os.OpenFile(bundle, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, f)
	closeErr := dst.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	// Git for Windows forwards the clone destination to index-pack as
	// GIT_DIR, whose parser still has a legacy path limit even with longpaths
	// enabled. Use paths relative to the already-private extraction directory,
	// in addition to validating the supported native execution-directory scope.
	relativeRepo, err := filepath.Rel(temp, repo)
	if err != nil || !safeRelative(filepath.ToSlash(relativeRepo)) {
		return errors.New("bundle repository is outside extraction directory")
	}
	if _, err = git(ctx, temp, "clone", "--bare", filepath.Base(bundle), relativeRepo); err != nil {
		return err
	}
	if err = supported(ctx, repo, commit); err != nil {
		return err
	}
	resolved, err := git(ctx, repo, "rev-parse", "--verify", commit+"^{commit}")
	if err != nil || strings.TrimSpace(resolved) != commit {
		return errors.New("bundle exact commit mismatch")
	}
	if _, err = git(ctx, repo, "fsck", "--strict"); err != nil {
		return err
	}
	return os.Remove(bundle)
}

// The local adapter also protects app.BuildPlan and later worktree operations
// from ambient Git transport configuration and lazy object fetches.
type localRunner struct{}

func (localRunner) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	if c.Name != "git" {
		return execx.Result{}, errors.New("only local Git is allowed")
	}
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	out, err := git(ctx, c.Dir, c.Args...)
	r := execx.Result{Stdout: out}
	if err != nil {
		r.ExitCode = -1
	}
	return r, err
}

func portableAlias(s string) bool {
	if !safeRelative(s) || strings.Contains(s, "/") || strings.HasSuffix(s, ".") {
		return false
	}
	base := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return false
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return false
	}
	return true
}

func parseCanonical(data []byte) (*config.Manifest, error) {
	var m config.Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, errors.New("multiple manifest values")
	}
	if err := config.Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Canonical snapshots include zero fields for historical digest stability;
// authored manifests omit incompatible fields whose presence is invalid.
func manifestDocument(data []byte) ([]byte, error) {
	m, err := parseCanonical(data)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	runtimes := raw["runtimes"].(map[string]any)
	for name, r := range m.Runtimes {
		if r.Type == "process" {
			fields := runtimes[name].(map[string]any)
			for _, key := range []string{"provider", "project_directory", "files", "avd"} {
				delete(fields, key)
			}
		}
	}
	components := raw["components"].(map[string]any)
	for name, c := range m.Components {
		if m.Runtimes[c.Runtime].Type == "process" {
			fields := components[name].(map[string]any)
			delete(fields, "compose_services")
			if endpoints, ok := fields["endpoints"].(map[string]any); ok {
				for _, value := range endpoints {
					endpoint := value.(map[string]any)
					for _, key := range []string{"service", "target", "protocol"} {
						delete(endpoint, key)
					}
				}
			}
		}
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if _, err = config.Parse(out); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeCommitted(data []byte, p Package) ([]byte, error) {
	m, err := config.Parse(data)
	if err != nil {
		return nil, err
	}
	if len(m.Sources) != len(p.Sources) {
		return nil, errors.New("committed source alias set mismatch")
	}
	for _, source := range p.Sources {
		v, ok := m.Sources[source.Alias]
		if !ok {
			return nil, errors.New("committed source alias mismatch")
		}
		v.Repository = "sources/" + source.Alias
		v.DefaultRef = source.Commit
		m.Sources[source.Alias] = v
	}
	return config.CanonicalJSON(m)
}

func androidSlots(m *config.Manifest, components []string) int {
	runtimes := map[string]bool{}
	for _, name := range components {
		c := m.Components[name]
		if m.Runtimes[c.Runtime].Type == "android-emulator" {
			runtimes[c.Runtime] = true
		}
	}
	return len(runtimes)
}
