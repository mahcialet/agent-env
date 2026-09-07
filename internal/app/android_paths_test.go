package app

import (
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type androidInputPaths struct {
	*androidLifecycleFake
	template, image string
}

func (f androidInputPaths) Validate(ctx context.Context, name string) (domain.AndroidEmulator, error) {
	a, err := f.androidLifecycleFake.Validate(ctx, name)
	a.TemplatePath, a.SystemImage = f.template, f.image
	return a, err
}
func TestAndroidInputOverlapRejectedBeforeAllocation(t *testing.T) {
	for _, input := range []string{"template", "image"} {
		for _, nested := range []bool{false, true} {
			t.Run(input+"/"+map[bool]string{false: "equal", true: "nested"}[nested], func(t *testing.T) {
				s, options, source, android := androidAppFixture(t)
				base := t.TempDir()
				template, image := filepath.Join(base, "template"), filepath.Join(base, "image")
				for _, p := range []string{template, image} {
					if err := os.MkdirAll(p, 0700); err != nil {
						t.Fatal(err)
					}
				}
				s.Home = template
				if input == "image" {
					s.Home = image
				}
				if nested {
					s.Home = filepath.Join(s.Home, "state")
				}
				s.Android = androidInputPaths{android, template, image}
				_, err := s.Create(context.Background(), options, CreateOptions{Owner: "test"})
				if !errors.Is(err, ErrPrerequisite) || !strings.Contains(err.Error(), "overlap") {
					t.Fatalf("expected overlap prerequisite: %v", err)
				}
				leases, err := s.Store.List(context.Background())
				if err != nil || len(leases) != 0 || len(source.states) != 0 || android.creates != 0 {
					t.Fatalf("overlap allocated: leases=%v err=%v", leases, err)
				}
				for _, p := range []string{template, image} {
					entries, err := os.ReadDir(p)
					if err != nil || len(entries) != 0 {
						t.Fatalf("input mutated %s: %v %v", p, entries, err)
					}
				}
			})
		}
	}
}
func TestAndroidInputOverlapResolvesAliasesAndAncestors(t *testing.T) {
	base := t.TempDir()
	input := filepath.Join(base, "input")
	if err := os.Mkdir(input, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(input, alias); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	for _, pair := range [][2]string{{filepath.Join(alias, "future", "runtime"), input}, {input, filepath.Join(alias, "future", "image")}} {
		overlap, err := androidInputOverlap(pair[0], pair[1])
		if err != nil || !overlap {
			t.Fatalf("missed aliased overlap %v: %t %v", pair, overlap, err)
		}
	}
	overlap, err := androidInputOverlap(filepath.Join(base, "input-sibling"), input)
	if err != nil || overlap {
		t.Fatalf("sibling incorrectly overlaps: %t %v", overlap, err)
	}
}

func TestAndroidCaseCollisionFailsBeforeAllocation(t *testing.T) {
	s, options, source, android := androidAppFixture(t)
	path := filepath.Join(options.Repository, ".agent-env.yaml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(strings.Replace(string(body), "runtimes:\n", "runtimes:\n  Phone: {type: android-emulator, source: self, avd: pixel}\n", 1))
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Create(context.Background(), options, CreateOptions{Owner: "test"})
	if err == nil || !strings.Contains(err.Error(), "case") {
		t.Fatalf("case collision accepted: %v", err)
	}
	leases, err := s.Store.List(context.Background())
	if err != nil || len(leases) != 0 || len(source.states) != 0 || android.creates != 0 {
		t.Fatal("case collision allocated resources")
	}
}

func TestAndroidPathsCompareOtherRuntimeInputs(t *testing.T) {
	base := t.TempDir()
	lease := domain.Lease{Runtimes: []domain.Runtime{
		{Name: "one", Directory: filepath.Join(base, "runtime-one"), Android: &domain.AndroidEmulator{TemplatePath: filepath.Join(base, "template-one"), SystemImage: filepath.Join(base, "image")}},
		{Name: "two", Directory: filepath.Join(base, "runtime-two"), Android: &domain.AndroidEmulator{TemplatePath: filepath.Join(base, "runtime-one", "template-two"), SystemImage: filepath.Join(base, "image")}},
	}}
	if err := validateAndroidInputPaths(lease); !errors.Is(err, ErrPrerequisite) {
		t.Fatalf("other runtime input overlap accepted: %v", err)
	}
}
