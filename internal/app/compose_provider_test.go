package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

type selectedRuntime struct {
	*lifecycleRuntime
	doctors []domain.ComposeProviderName
	fail    domain.ComposeProviderName
}

func (r *selectedRuntime) DoctorFor(_ context.Context, p domain.ComposeProviderName) (map[string]string, error) {
	r.doctors = append(r.doctors, p)
	if p == r.fail {
		return nil, errors.New("selected engine unavailable")
	}
	return map[string]string{"context": string(p) + "-pinned"}, nil
}
func setMixedProviders(t *testing.T, o PlanOptions) {
	t.Helper()
	path := filepath.Join(o.Repository, ".agent-env.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "second: {type: compose,", "second: {type: compose, provider: podman-compose,", 1))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestCreatePinsEachProviderBeforeReservation(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "mixed", true: "missing-podman"}[fail], func(t *testing.T) {
			s, o, _, runtime, operations := lifecycleFixture(t)
			setMixedProviders(t, o)
			selected := &selectedRuntime{lifecycleRuntime: runtime}
			if fail {
				selected.fail = domain.ComposeProviderPodman
			}
			s.Runtime = selected
			l, err := s.Create(context.Background(), o, CreateOptions{Owner: "tester"})
			if fail {
				if !errors.Is(err, ErrPrerequisite) {
					t.Fatalf("wrong prerequisite: %v", err)
				}
				rows, e := s.Store.List(context.Background())
				if e != nil || len(rows) != 0 || len(*operations) != 0 {
					t.Fatalf("failed doctor had effects: %v %v %v", rows, *operations, e)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(selected.doctors) != 2 {
				t.Fatalf("providers not independently checked: %v", selected.doctors)
			}
			stored, err := s.Store.Get(context.Background(), l.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range stored.Runtimes {
				if r.Context != string(r.Provider)+"-pinned" {
					t.Fatalf("wrong recorded identity: %+v", r)
				}
			}
			// Later source changes cannot change the snapshot's provider identity.
			path := filepath.Join(o.Repository, ".agent-env.yaml")
			if err := os.WriteFile(path, []byte("invalid: true"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Destroy(context.Background(), l.ID, false, false); err != nil {
				t.Fatal(err)
			}
			if len(selected.doctors) != 2 {
				t.Fatal("cleanup rediscovered mutable defaults")
			}
		})
	}
}

type selectedInventory struct {
	RuntimeProvider
	calls []domain.ComposeProviderName
}

func (r *selectedInventory) Doctor(context.Context) (map[string]string, error) {
	panic("unselected doctor")
}
func (r *selectedInventory) Inventory(context.Context, string) ([]domain.Resource, error) {
	panic("unselected inventory")
}
func (r *selectedInventory) DoctorFor(_ context.Context, p domain.ComposeProviderName) (map[string]string, error) {
	return map[string]string{"context": "same-engine-name"}, nil
}
func (r *selectedInventory) InventoryFor(_ context.Context, p domain.ComposeProviderName, id string) ([]domain.Resource, error) {
	r.calls = append(r.calls, p)
	owner := string(p) + "-lease"
	item := inventoryItem(string(p), "container", id, "same-project", owner)
	item.Metadata["provider"] = string(p)
	return []domain.Resource{item}, nil
}
func TestInventoryScopesCoincidentProjectsByProvider(t *testing.T) {
	runtime := &selectedInventory{}
	leases := []domain.Lease{}
	for _, p := range []domain.ComposeProviderName{domain.ComposeProviderDocker, domain.ComposeProviderPodman} {
		leases = append(leases, domain.Lease{ID: string(p) + "-lease", Desired: "active", Runtimes: []domain.Runtime{{Type: "compose", Provider: p, Context: "same-engine-name", Project: "same-project"}}})
	}
	s := Service{Store: inventoryStore{leases: leases}, Runtime: runtime, Home: t.TempDir()}
	items, err := s.Inventory(context.Background())
	if err != nil || len(items) != 0 || len(runtime.calls) != 2 {
		t.Fatalf("cross-provider collision: %+v %v %v", items, runtime.calls, err)
	}
}
