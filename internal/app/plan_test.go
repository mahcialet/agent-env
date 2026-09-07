package app

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

type planningSource struct{ calls []string }

func (s *planningSource) Resolve(_ context.Context, repo, ref string) (domain.Source, error) {
	s.calls = append(s.calls, repo+":"+ref)
	return domain.Source{RepositoryID: repo, RepositoryPath: repo, RequestedRef: ref, Commit: "0123456789012345678901234567890123456789"}, nil
}
func (*planningSource) Materialize(context.Context, domain.Source) error { panic("plan mutated Git") }
func (*planningSource) Inspect(context.Context, domain.Source) (SourceObservation, error) {
	panic("plan inspected lease")
}
func (*planningSource) Remove(context.Context, domain.Source, bool) error { panic("plan removed Git") }

func TestPlanClosureAndNoAllocation(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "spaces 日本語")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../../testdata/manifests/api-dashboard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), fixture, 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		stack string
		want  []string
	}{{"api", []string{"api"}}, {"dashboard", []string{"api", "dashboard"}}} {
		s := &planningSource{}
		p, err := BuildPlan(context.Background(), PlanOptions{Repository: repo, Stack: tc.stack}, s)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, c := range p.Components {
			names = append(names, c.Name)
		}
		if !reflect.DeepEqual(names, tc.want) {
			t.Fatalf("%s: %v", tc.stack, names)
		}
		if len(s.calls) != 1 || p.SourceSetDigest == "" || p.ManifestDigest == "" {
			t.Fatalf("missing immutable source data: %+v", p)
		}
		p2, err := BuildPlan(context.Background(), PlanOptions{Repository: repo, Stack: tc.stack}, s)
		if err != nil || !reflect.DeepEqual(p, p2) {
			t.Fatalf("nondeterministic plan: %v", err)
		}
	}
	entries, err := os.ReadDir(repo)
	if err != nil || len(entries) != 1 {
		t.Fatalf("plan modified repository: %v %v", entries, err)
	}
}

func TestExplicitManifestUsesControlRepositoryForSources(t *testing.T) {
	repo := t.TempDir()
	fixture, err := os.ReadFile("../../testdata/manifests/api-dashboard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "custom.yaml")
	if err = os.WriteFile(path, fixture, 0644); err != nil {
		t.Fatal(err)
	}
	for _, options := range []PlanOptions{{Repository: repo, ManifestPath: "custom.yaml", Stack: "api"}, {Repository: path, Stack: "api"}} {
		p, err := BuildPlan(context.Background(), options, &planningSource{})
		if err != nil {
			t.Fatal(err)
		}
		if p.Repository != repo || p.Sources[0].RepositoryPath != repo {
			t.Fatalf("manifest filename used as source base: %+v", p)
		}
	}
}
