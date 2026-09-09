package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

func TestManagedResourcesRequireBothOwnershipLabels(t *testing.T) {
	for _, kind := range []string{"container", "network", "volume"} {
		for _, missing := range []string{leaseLabel, runtimeLabel} {
			t.Run(kind+"/"+missing, func(t *testing.T) {
				r := runtimeFixture(t)
				r.LeaseID = "lease123"
				labels := map[string]string{projectLabel: r.Project, leaseLabel: r.LeaseID, runtimeLabel: r.Name, "com.docker.compose.service": "api"}
				delete(labels, missing)
				var steps []step
				if kind == "container" {
					data, _ := json.Marshal([]map[string]any{{"Id": "c123", "Config": map[string]any{"Labels": labels}, "State": map[string]any{"Running": true}}})
					steps = inspectSteps(r, "c123", string(data), "", "", "", "")[:2]
				} else {
					data, _ := json.Marshal([]map[string]any{{"Id": "n123", "Name": r.Project + "_data", "Labels": labels}})
					if kind == "network" {
						steps = inspectSteps(r, "", "", "n123", string(data), "", "")[:3]
					} else {
						steps = inspectSteps(r, "", "", "", "", r.Project+"_data", string(data))
					}
				}
				f := &fakeRunner{t: t, steps: steps}
				if _, err := testClient(f).Inspect(context.Background(), r); err == nil || !strings.Contains(err.Error(), "ownership mismatch") {
					t.Fatalf("missing ownership accepted: %v", err)
				}
				f.done()
			})
		}
	}
}

func namedRecord(r domain.Runtime, name string) string {
	data, _ := json.Marshal([]map[string]any{{"Id": "n123", "Name": name, "Labels": map[string]string{projectLabel: r.Project, leaseLabel: r.LeaseID, runtimeLabel: r.Name}}})
	return string(data)
}

func TestInspectIncludesNamedResourceOwnership(t *testing.T) {
	r := runtimeFixture(t)
	r.LeaseID = "lease123"
	f := &fakeRunner{t: t, steps: inspectSteps(r, "c123", containerJSON(r, "running"), "n123", namedRecord(r, r.Project+"_default"), r.Project+"_data", namedRecord(r, r.Project+"_data"))}
	observation, err := testClient(f).Inspect(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.Resources) != 3 {
		t.Fatalf("resources %+v", observation.Resources)
	}
	for _, resource := range observation.Resources {
		if resource.Metadata["lease"] != r.LeaseID || resource.Metadata["runtime"] != r.Name {
			t.Fatalf("missing ownership metadata: %+v", resource)
		}
	}
	f.done()
}

func declaredFixture(t *testing.T, kind string) domain.Runtime {
	t.Helper()
	r := runtimeFixture(t)
	r.LeaseID = "lease123"
	data, _ := json.Marshal(map[string]any{"services": map[string]any{"api": map[string]any{}}, kind + "s": map[string]any{"data": map[string]any{"name": r.Project + "_data"}}})
	r.ConfigPath = filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(r.ConfigPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	r.ConfigDigest = hex.EncodeToString(sum[:])
	return r
}

func declaredListing(r domain.Runtime, kind string) []string {
	return []string{"--context", r.Context, kind, "ls", "--filter", "name=" + r.Project + "_data", "--format", "{{json .}}"}
}
func pinnedBase(r domain.Runtime) []string {
	return []string{"--context", r.Context, "compose", "-p", r.Project, "--project-directory", r.Directory, "-f", r.ConfigPath}
}

func TestDeclaredForeignNamesCannotBeMountedOrDeleted(t *testing.T) {
	for _, kind := range []string{"network", "volume"} {
		for _, operation := range []string{"up", "down"} {
			t.Run(kind+"/"+operation, func(t *testing.T) {
				r := declaredFixture(t, kind)
				name := r.Project + "_data"
				var steps []step
				if operation == "down" {
					steps = inspectSteps(r, "", "", "", "", "", "")
				}
				foreign := r
				foreign.Project = "unrelated-project"
				foreign.LeaseID = "another-lease"
				steps = append(steps, step{args: declaredListing(r, kind), output: `{"Name":"` + name + `"}`}, step{args: []string{"--context", r.Context, kind, "inspect", name}, output: namedRecord(foreign, name)})
				f := &fakeRunner{t: t, steps: steps}
				client := testClient(f)
				var err error
				if operation == "up" {
					err = client.Up(context.Background(), r)
				} else {
					err = client.Down(context.Background(), r)
				}
				if err == nil || !strings.Contains(err.Error(), "refuse to touch declared") {
					t.Fatalf("foreign name %s accepted: %v", name, err)
				}
				f.done()
			})
		}
	}
}

func TestDeclaredOwnedOrAbsentNamesPermitEffects(t *testing.T) {
	for _, kind := range []string{"network", "volume"} {
		for _, exists := range []bool{false, true} {
			t.Run(kind+"/"+map[bool]string{false: "absent", true: "owned"}[exists], func(t *testing.T) {
				r := declaredFixture(t, kind)
				name := r.Project + "_data"
				listing := `{"Name":"` + name + `-other"}`
				if exists {
					listing = `{"Name":"` + name + `"}`
				}
				steps := []step{{args: declaredListing(r, kind), output: listing}}
				if exists {
					steps = append(steps, step{args: []string{"--context", r.Context, kind, "inspect", name}, output: namedRecord(r, name)})
				}
				steps = append(steps, step{args: with(pinnedBase(r), "up", "-d", "api")})
				f := &fakeRunner{t: t, steps: steps}
				if err := testClient(f).Up(context.Background(), r); err != nil {
					t.Fatal(err)
				}
				f.done()
			})
		}
	}
}

func TestDeclaredOwnershipObservationFailuresAreClosed(t *testing.T) {
	for _, mode := range []string{"listing-error", "malformed-list", "inspect-error", "empty-inspect"} {
		t.Run(mode, func(t *testing.T) {
			r := declaredFixture(t, "volume")
			name := r.Project + "_data"
			first := step{args: declaredListing(r, "volume"), output: `{"Name":"` + name + `"}`}
			if mode == "listing-error" {
				first.err = errors.New("daemon unavailable")
			}
			if mode == "malformed-list" {
				first.output = "not json"
			}
			steps := []step{first}
			if mode == "inspect-error" || mode == "empty-inspect" {
				second := step{args: []string{"--context", r.Context, "volume", "inspect", name}, output: "[]"}
				if mode == "inspect-error" {
					second.err = errors.New("inspection unavailable")
				}
				steps = append(steps, second)
			}
			f := &fakeRunner{t: t, steps: steps}
			if err := testClient(f).Up(context.Background(), r); err == nil {
				t.Fatal("unknown ownership allowed effects")
			}
			f.done()
		})
	}
}

func TestMissingContainerIdentityRefusesInspectionAndDown(t *testing.T) {
	for _, operation := range []string{"inspect", "down"} {
		for _, explicitEmpty := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/missing", true: "/empty"}[explicitEmpty], func(t *testing.T) {
				r := runtimeFixture(t)
				r.LeaseID = "owned-lease"
				r.ConfigPath = filepath.Join(t.TempDir(), "snapshot.json")
				if err := os.WriteFile(r.ConfigPath, []byte(`{"services":{"api":{"image":"fixture"}}}`), 0600); err != nil {
					t.Fatal(err)
				}
				down, inspected := false, false
				c := Client{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
					args := strings.Join(q.Args, " ")
					if strings.Contains(args, " compose ") {
						down = true
						return execx.Result{}, nil
					}
					if down {
						return execx.Result{}, nil
					}
					if strings.Contains(args, " ps ") {
						return execx.Result{Stdout: "listed-container"}, nil
					}
					if strings.Contains(args, " inspect --type container ") {
						inspected = true
						row := map[string]any{"Config": map[string]any{"Labels": map[string]string{projectLabel: r.Project, leaseLabel: r.LeaseID, runtimeLabel: r.Name, "com.docker.compose.service": "api"}}, "State": map[string]any{"Running": true, "Status": "running"}}
						if explicitEmpty {
							row["Id"] = ""
						}
						data, err := json.Marshal([]any{row})
						if err != nil {
							t.Fatal(err)
						}
						return execx.Result{Stdout: string(data)}, nil
					}
					return execx.Result{}, nil
				})}
				var err error
				if operation == "down" {
					err = c.Down(context.Background(), r)
				} else {
					_, err = c.Inspect(context.Background(), r)
				}
				if !inspected || err == nil || !strings.Contains(err.Error(), "empty identity") || down {
					t.Fatalf("missing identity did not fail closed: inspected=%v down=%v err=%v", inspected, down, err)
				}
			})
		}
	}
}
