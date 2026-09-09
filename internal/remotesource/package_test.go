package remotesource

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := git(context.Background(), dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(out)
}
func repository(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "checkout")
	gitTest(t, "", "init", repo)
	os.WriteFile(filepath.Join(repo, "data"), []byte("committed bytes"), 0600)
	gitTest(t, repo, "add", "data")
	gitTest(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "initial")
	return repo
}
func manifest(t *testing.T, repo, other string) {
	t.Helper()
	m := config.Manifest{Version: 1, Sources: map[string]config.Source{"app": {Repository: repo, DefaultRef: "HEAD"}, "other": {Repository: other, DefaultRef: "HEAD"}}, Runtimes: map[string]config.Runtime{"api": {Type: "process", Source: "app", WorkingDirectory: ".", Command: []string{"go", "version"}}}, Components: map[string]config.Component{"api": {Runtime: "api"}}, Stacks: map[string]config.Stack{"local": {Roots: []string{"api"}}}}
	canonical, err := config.CanonicalJSON(&m)
	b, err := manifestDocument(canonical)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), b, 0600)
	gitTest(t, repo, "add", ".agent-env.yaml")
	gitTest(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "manifest")
}
func TestCommittedMultiAliasRoundtripAndWorktreeLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, other := repository(t), repository(t)
	manifest(t, repo, other)
	cas, err := blobstore.New(filepath.Join(t.TempDir(), "cas"))
	if err != nil {
		t.Fatal(err)
	}
	before := gitTest(t, repo, "show-ref")
	p, err := Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sources) != 2 || !equalStrings(p.RequiredCapabilities, []string{"git", "persistent-process"}) {
		t.Fatalf("package %+v", p)
	}
	raw, _ := json.Marshal(p)
	if bytes.Contains(raw, []byte(repo)) || bytes.Contains(raw, []byte(other)) {
		t.Fatal("client absolute path leaked")
	}
	if gitTest(t, repo, "show-ref") != before {
		t.Fatal("original refs changed")
	}
	// HTTP persistence can recursively reorder object keys. Semantic digests
	// must survive that transport without weakening committed-manifest proof.
	var reordered any
	if err = json.Unmarshal(raw, &reordered); err != nil {
		t.Fatal(err)
	}
	transit, _ := json.Marshal(reordered)
	var received Package
	if err = json.Unmarshal(transit, &received); err != nil {
		t.Fatal(err)
	}
	if err = Validate(received); err != nil {
		t.Fatalf("JSON reordering changed identity: %v", err)
	}
	p = received
	os.WriteFile(filepath.Join(repo, "data"), []byte("uncommitted change"), 0600)
	home := filepath.Join(t.TempDir(), "worker")
	options, provider, err := Materialize(ctx, p, home, cas)
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := Materialize(ctx, p, home, cas)
	if err != nil || again.Repository != options.Repository {
		t.Fatalf("idempotent materialization: %v", err)
	}
	plan, err := app.BuildPlan(ctx, options, provider)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceSetDigest != p.SourceSetDigest || plan.ManifestCommit != p.ManifestCommit || plan.ManifestModified {
		t.Fatal("plan identity drift")
	}
	for i, s := range plan.Sources {
		s.WorktreePath = filepath.Join(t.TempDir(), "worktree")
		if err = provider.Materialize(ctx, s); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(s.WorktreePath, "data"))
		if err != nil || string(b) != "committed bytes" {
			t.Fatalf("source %d bytes %q: %v", i, b, err)
		}
		o, err := provider.Inspect(ctx, s)
		if err != nil || !o.Exists || !o.Registered || o.Commit != s.Commit || o.TrackedDirty {
			t.Fatalf("inspection %+v %v", o, err)
		}
		if err = provider.Remove(ctx, s, false); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRejectSelfConsistentManifestTamperAndWrongBundle(t *testing.T) {
	ctx := context.Background()
	repo, other := repository(t), repository(t)
	manifest(t, repo, other)
	cas, _ := blobstore.New(filepath.Join(t.TempDir(), "cas"))
	p, err := Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"command", "wrong-bundle", "path", "alias-path", "capabilities", "commit", "digest"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(p)
			var q Package
			json.Unmarshal(raw, &q)
			switch mode {
			case "command":
				m, _ := parseCanonical(q.Manifest)
				r := m.Runtimes["api"]
				r.Command = []string{"unapproved-command"}
				m.Runtimes["api"] = r
				q.Manifest, _ = config.CanonicalJSON(m)
				q.ManifestDigest = digest(q.Manifest)
			case "wrong-bundle":
				q.Sources[0].BlobDigest = q.Sources[1].BlobDigest
			case "path":
				q.ManifestPath = "../escape"
			case "alias-path":
				m, _ := parseCanonical(q.Manifest)
				s := m.Sources["app"]
				s.Repository = repo
				m.Sources["app"] = s
				q.Manifest, _ = config.CanonicalJSON(m)
				q.ManifestDigest = digest(q.Manifest)
			case "capabilities":
				q.RequiredCapabilities = []string{"git"}
			case "commit":
				q.Sources[0].Commit = strings.Repeat("a", 40)
				q.SourceSetDigest = sourceSet(q)
			case "digest":
				q.ManifestDigest = strings.Repeat("0", 64)
			}
			q.PlanDigest = packageDigest(q)
			if _, _, err := Materialize(ctx, q, filepath.Join(t.TempDir(), "worker"), cas); err == nil {
				t.Fatal("tampered package accepted")
			}
		})
	}
}
func TestRejectDirtyManifestAndUnsupportedGitSources(t *testing.T) {
	for _, mode := range []string{"dirty", "untracked", "shallow", "submodule", "lfs-pointer", "lfs-attributes"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			repo, other := repository(t), repository(t)
			manifest(t, repo, other)
			switch mode {
			case "dirty":
				f, _ := os.OpenFile(filepath.Join(repo, ".agent-env.yaml"), os.O_APPEND|os.O_WRONLY, 0600)
				f.WriteString("\n ")
				f.Close()
			case "untracked":
				gitTest(t, repo, "rm", "--cached", ".agent-env.yaml")
			case "shallow":
				clone := filepath.Join(t.TempDir(), "shallow")
				gitTest(t, "", "clone", "--no-local", "--depth", "1", other, clone)
				m, _ := config.Load(repo)
				s := m.Sources["other"]
				s.Repository = clone
				m.Sources["other"] = s
				canonical, _ := config.CanonicalJSON(m)
				b, _ := manifestDocument(canonical)
				os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), b, 0600)
				gitTest(t, repo, "add", ".agent-env.yaml")
				gitTest(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "shallow input")
			case "submodule":
				sha := gitTest(t, other, "rev-parse", "HEAD")
				gitTest(t, other, "update-index", "--add", "--cacheinfo", "160000,"+sha+",nested")
				gitTest(t, other, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "submodule")
			case "lfs-pointer", "lfs-attributes":
				name, content := "pointer", "version https://git-lfs.github.com/spec/v1\noid sha256:"+strings.Repeat("0", 64)+"\nsize 1\n"
				if mode == "lfs-attributes" {
					name, content = ".gitattributes", "*.bin filter=lfs diff=lfs merge=lfs -text\n"
				}
				os.WriteFile(filepath.Join(other, name), []byte(content), 0600)
				gitTest(t, other, "add", name)
				gitTest(t, other, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "unsupported LFS")
			}
			cas, _ := blobstore.New(filepath.Join(t.TempDir(), "cas"))
			if _, err := Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas); err == nil {
				t.Fatal("unsupported input accepted")
			}
		})
	}
}
