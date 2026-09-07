package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/execx"
	"gopkg.in/yaml.v3"
)

func originGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r, err := (execx.OSRunner{}).Run(context.Background(), execx.Command{Name: "git", Args: args, Dir: dir, Timeout: 20 * time.Second})
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, r.Stderr)
	}
	return strings.TrimSpace(r.Stdout)
}
func originRepository(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	dir = canonical
	originGit(t, dir, "-c", "init.templateDir=", "init")
	originGit(t, dir, "config", "user.name", "Origin Fixture")
	originGit(t, dir, "config", "user.email", "origin@example.invalid")
	originGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte(name), 0600); err != nil {
		t.Fatal(err)
	}
	originGit(t, dir, "add", ".")
	originGit(t, dir, "commit", "-m", "fixture")
	return dir
}
func originPlan(t *testing.T, repository, manifest, ref string) app.Plan {
	t.Helper()
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	args := []string{"plan", repository, "--manifest", manifest, "--stack", "api", "--output", "json"}
	if ref != "" {
		args = append(args, "--ref", ref)
	}
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("plan: %v %s", err, stderr.String())
	}
	var envelope struct {
		Data app.Plan `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func TestPlanManifestOriginIndependentOfRuntimeRef(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	source := originRepository(t, "runtime source 日本語")
	pinned := originGit(t, source, "rev-parse", "HEAD")
	control := originRepository(t, "control checkout spaces")
	fixture, err := os.ReadFile("../../testdata/manifests/api-dashboard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := config.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for name, spec := range m.Sources {
		spec.Repository = source
		m.Sources[name] = spec
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(control, "selected manifest 日本語.yaml")
	if err := os.WriteFile(manifest, b, 0600); err != nil {
		t.Fatal(err)
	}
	beforeHead := originGit(t, control, "rev-parse", "HEAD")
	p := originPlan(t, control, manifest, pinned)
	if p.ManifestCommit != beforeHead || !p.ManifestModified || p.ManifestPath != manifest || p.Sources[0].Commit != pinned {
		t.Fatalf("untracked explicit origin: %+v", p)
	}
	originGit(t, control, "add", ".")
	originGit(t, control, "commit", "-m", "trusted control manifest")
	controlHead := originGit(t, control, "rev-parse", "HEAD")
	p = originPlan(t, control, manifest, pinned)
	if p.ManifestCommit != controlHead || p.ManifestModified || p.Sources[0].Commit != pinned || p.ManifestCommit == p.Sources[0].Commit {
		t.Fatalf("control origin replaced by requested runtime ref: %+v", p)
	}
	if len(p.Sources) != 1 || p.Sources[0].RepositoryPath != source {
		t.Fatal("control checkout incorrectly materialized as a source")
	}
	originalDigest := p.ManifestDigest
	b = append(b, []byte("\n# local control edit\n")...)
	if err := os.WriteFile(manifest, b, 0600); err != nil {
		t.Fatal(err)
	}
	p = originPlan(t, control, manifest, pinned)
	if !p.ManifestModified || p.ManifestCommit != controlHead || p.ManifestDigest != originalDigest {
		t.Fatalf("modified origin/canonical digest: %+v", p)
	}
	if got := originGit(t, control, "rev-parse", "HEAD"); got != controlHead {
		t.Fatal("planning changed control HEAD")
	}
	if got := originGit(t, source, "rev-parse", "HEAD"); got != pinned {
		t.Fatal("planning changed runtime HEAD")
	}
	outside := filepath.Join(t.TempDir(), "external.yaml")
	if err := os.WriteFile(outside, b, 0600); err != nil {
		t.Fatal(err)
	}
	p = originPlan(t, control, outside, pinned)
	if p.ManifestCommit != "" || !p.ManifestModified || len(p.Diagnostics) == 0 || p.ManifestDigest == "" {
		t.Fatalf("unversioned snapshot authority not explicit: %+v", p)
	}
}
