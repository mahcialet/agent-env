package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

// Unimplemented embedded methods panic if inventory accidentally mutates state.
type inventoryStore struct {
	Store
	leases []domain.Lease
}

func (s inventoryStore) List(context.Context) ([]domain.Lease, error) { return s.leases, nil }

type inventoryRuntime struct {
	RuntimeProvider
	items       map[string][]domain.Resource
	contexts    []string
	doctorError error
}

func (r *inventoryRuntime) Doctor(context.Context) (map[string]string, error) {
	return map[string]string{"context": "active"}, r.doctorError
}
func (r *inventoryRuntime) Inventory(_ context.Context, c string) ([]domain.Resource, error) {
	r.contexts = append(r.contexts, c)
	return r.items[c], nil
}

type inventorySource struct {
	SourceProvider
	resolved, inspected []string
}

func (s *inventorySource) Resolve(_ context.Context, path, ref string) (domain.Source, error) {
	s.resolved = append(s.resolved, path)
	return domain.Source{RepositoryID: "canonical-repo", RepositoryPath: path, Commit: "pinned"}, nil
}
func (s *inventorySource) Inspect(_ context.Context, source domain.Source) (SourceObservation, error) {
	s.inspected = append(s.inspected, source.WorktreePath)
	return SourceObservation{Exists: true, Registered: true, TrackedDirty: true, Commit: "pinned"}, nil
}

func inventoryItem(id, kind, contextName, project, lease string) domain.Resource {
	return domain.Resource{ID: id, Kind: kind, LeaseID: lease, ExternalID: id, Metadata: map[string]string{"context": contextName, "project": project}}
}

func TestInventorySeparatesKnownOrphanForeignAndMismatch(t *testing.T) {
	home := t.TempDir()
	knownPath := filepath.Join(home, "worktrees", "known", "api")
	orphanPath := filepath.Join(home, "worktrees", "missing", "api 日本語")
	for _, p := range []string{knownPath, orphanPath, filepath.Join(home, "leases", "missing"), filepath.Join(home, "leases", "known")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store := inventoryStore{leases: []domain.Lease{{ID: "known", Desired: "active", Sources: []domain.Source{{WorktreePath: knownPath}}, Runtimes: []domain.Runtime{{Project: "registered", Context: "recorded"}}}}}
	runtime := &inventoryRuntime{items: map[string][]domain.Resource{
		"recorded": {inventoryItem("known-container", "container", "recorded", "registered", "known")},
		"active":   {inventoryItem("orphan-container", "container", "active", "unregistered", "gone"), inventoryItem("orphan-project", "compose-project", "active", "unregistered", ""), inventoryItem("foreign", "compose-project", "active", "foreign", ""), inventoryItem("mismatch", "container", "active", "wrong-project", "known")},
	}}
	source := &inventorySource{}
	service := Service{Store: store, Runtime: runtime, Source: source, Home: home}
	items, err := service.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runtime.contexts, []string{"active", "recorded"}) {
		t.Fatalf("context inventory %v", runtime.contexts)
	}
	statuses := map[string]string{}
	for _, item := range items {
		statuses[item.ID] = item.Metadata["status"]
	}
	for id, want := range map[string]string{"orphan-container": "orphaned", "orphan-project": "orphaned", "foreign": "foreign", "mismatch": "identity_mismatch", "worktree:" + orphanPath: "orphaned", "lease-directory:" + filepath.Join(home, "leases", "missing"): "orphaned"} {
		if statuses[id] != want {
			t.Errorf("%s wanted %s got %s", id, want, statuses[id])
		}
	}
	if _, ok := statuses["known-container"]; ok {
		t.Fatal("registered resource reported as orphan")
	}
	if !reflect.DeepEqual(source.resolved, []string{orphanPath}) || !reflect.DeepEqual(source.inspected, []string{orphanPath}) {
		t.Fatalf("unexpected source inspection: %+v", source)
	}
	if _, err := os.Stat(orphanPath); err != nil {
		t.Fatal("orphan path was removed")
	}
}

func TestInventoryContinuesFilesystemObservationAfterDoctorFailure(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "leases", "missing")
	if err := os.MkdirAll(p, 0700); err != nil {
		t.Fatal(err)
	}
	service := Service{Store: inventoryStore{}, Runtime: &inventoryRuntime{doctorError: errors.New("daemon unavailable")}, Home: home}
	items, err := service.Inventory(context.Background())
	if err == nil || len(items) != 1 || items[0].ExternalID != p {
		t.Fatalf("lost independent filesystem evidence: %+v %v", items, err)
	}
}

func TestInventoryReportsConflictingOwnership(t *testing.T) {
	runtime := &inventoryRuntime{items: map[string][]domain.Resource{"active": {inventoryItem("a", "container", "active", "same-project", "lease-a"), inventoryItem("b", "container", "active", "same-project", "lease-b")}}}
	service := Service{Store: inventoryStore{}, Runtime: runtime, Home: t.TempDir()}
	items, err := service.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Metadata["status"] != "identity_mismatch" {
			t.Fatalf("ambiguous project adopted: %+v", item)
		}
	}
}
