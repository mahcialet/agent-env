package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
)

type createPreflightAndroid struct {
	app.AndroidProvider
	metadata     domain.AndroidEmulator
	prerequisite error
	live         bool
}

func (a *createPreflightAndroid) Validate(context.Context, string) (domain.AndroidEmulator, error) {
	return a.metadata, a.prerequisite
}
func (a *createPreflightAndroid) Create(_ context.Context, r domain.Runtime) (domain.Runtime, error) {
	a.live = true
	return r, nil
}
func (a *createPreflightAndroid) Inspect(context.Context, domain.Runtime) (app.RuntimeObservation, error) {
	return app.RuntimeObservation{Exists: a.live, Ready: a.live}, nil
}
func (a *createPreflightAndroid) Destroy(context.Context, domain.Runtime) error {
	a.live = false
	return nil
}

func createPreflightFixture(t *testing.T) (app.PlanOptions, *createPreflightAndroid) {
	t.Helper()
	repo := originRepository(t, "create preflight source")
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(androidCLIManifest), 0600); err != nil {
		t.Fatal(err)
	}
	sdk, template := filepath.Join(t.TempDir(), "sdk"), filepath.Join(t.TempDir(), "template.avd")
	image := filepath.Join(sdk, "system-images", "test")
	for _, dir := range []string{image, template} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "sentinel"), []byte("immutable input"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return app.PlanOptions{Repository: repo, Stack: "android"}, &createPreflightAndroid{metadata: domain.AndroidEmulator{Template: "Pixel.API35", SDKPath: sdk, TemplatePath: template, SystemImage: image}}
}

func TestCreateServiceRejectsInputOverlapBeforeOpeningRegistry(t *testing.T) {
	for _, input := range []string{"template", "image"} {
		t.Run(input, func(t *testing.T) {
			options, android := createPreflightFixture(t)
			roots := map[string]string{"template": android.metadata.TemplatePath, "image": android.metadata.SystemImage}
			home := filepath.Join(roots[input], "state")
			t.Setenv("AGENT_ENV_HOME", home)
			s, close, err := openCreateService(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			defer close()
			s.Android = android
			if _, err := s.Create(context.Background(), options, app.CreateOptions{Owner: "fixture"}); err == nil || !strings.Contains(err.Error(), "overlap") {
				t.Fatalf("immutable input overlap accepted: %v", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("preflight mutated immutable input with state files: %v", err)
			}
			for _, dir := range []string{android.metadata.SystemImage, android.metadata.TemplatePath} {
				data, err := os.ReadFile(filepath.Join(dir, "sentinel"))
				if err != nil || string(data) != "immutable input" {
					t.Fatalf("immutable input modified: %q %v", data, err)
				}
			}
			if err := close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateServicePrerequisiteFailureLeavesStateHomeAbsent(t *testing.T) {
	options, android := createPreflightFixture(t)
	home := filepath.Join(t.TempDir(), "state")
	t.Setenv("AGENT_ENV_HOME", home)
	android.prerequisite = errors.New("unusable selected SDK")
	s, close, err := openCreateService(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	s.Android = android
	if _, err := s.Create(context.Background(), options, app.CreateOptions{Owner: "fixture"}); !errors.Is(err, app.ErrPrerequisite) {
		t.Fatalf("prerequisite error: %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("prerequisite failure created state: %v", err)
	}
}

func TestCreateServiceOpensRegistryAtReservationAndClosesIt(t *testing.T) {
	options, android := createPreflightFixture(t)
	home := filepath.Join(t.TempDir(), "state")
	t.Setenv("AGENT_ENV_HOME", home)
	s, close, err := openCreateService(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	s.Android = android
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("factory eagerly opened state: %v", err)
	}
	lease, err := s.Create(context.Background(), options, app.CreateOptions{Owner: "fixture"})
	if err != nil || lease.Observed != "ready" {
		t.Fatalf("create: %+v %v", lease, err)
	}
	if _, err := os.Stat(filepath.Join(home, "state.db")); err != nil {
		t.Fatalf("reservation did not initialize registry: %v", err)
	}
	opened := s.Store.(*createStore).Store
	if err := s.Store.Reserve(context.Background(), lease, 0); err == nil {
		t.Fatal("duplicate reservation accepted")
	}
	if s.Store.(*createStore).Store != opened {
		t.Fatal("repeated reservation replaced the open registry")
	}
	stored, err := s.Store.Get(context.Background(), lease.ID)
	if err != nil || stored.ID != lease.ID {
		t.Fatalf("delegated store: %+v %v", stored, err)
	}
	if _, err := s.Destroy(context.Background(), lease.ID, false, false); err != nil {
		t.Fatal(err)
	}
	if err := close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Get(context.Background(), lease.ID); err == nil {
		t.Fatal("factory close left registry open")
	}
}

func TestCreateStorePropagatesOpeningAndClosingFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("existing file"), 0600); err != nil {
		t.Fatal(err)
	}
	store := &createStore{home: path}
	if err := store.Reserve(context.Background(), domain.Lease{ID: "lease"}, 0); err == nil || store.Store != nil {
		t.Fatalf("failed opening published a store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close unopened store: %v", err)
	}
	failure := errors.New("close failed")
	store.close = func() error { return failure }
	if err := store.Close(); !errors.Is(err, failure) {
		t.Fatalf("lost close error: %v", err)
	}
}
