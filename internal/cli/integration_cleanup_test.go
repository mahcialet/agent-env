package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// Recovery enumerates only the existing registry under this fixture's private
// home. No create output, labels, or daemon-wide inventory authorize cleanup.
func cleanupIntegrationRegistry(home string, destroy func(string) (domain.Lease, error)) error {
	path := filepath.Join(home, "state.db")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	db, err := sqlite.Open(path)
	if err != nil {
		return err
	}
	leases, listErr := db.List(context.Background())
	if err := errors.Join(listErr, db.Close()); err != nil {
		return err
	}
	var failures []error
	for _, lease := range leases {
		released, err := destroy(lease.ID)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if released.ID != lease.ID || released.Observed != "released" {
			failures = append(failures, fmt.Errorf("fixture lease %s cleanup not confirmed: id=%s state=%s", lease.ID, released.ID, released.Observed))
		}
	}
	return errors.Join(failures...)
}

func TestIntegrationRegistryRecoveryWithoutCreateOutput(t *testing.T) {
	for _, output := range []string{"", "malformed JSON"} {
		t.Run(fmt.Sprintf("output=%q", output), func(t *testing.T) {
			home := t.TempDir()
			db, err := sqlite.Open(filepath.Join(home, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			lease := domain.Lease{ID: "persisted-before-response", Owner: "fixture", Desired: "ready", Observed: "starting", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
			if err := db.Reserve(context.Background(), lease, 0); err != nil {
				t.Fatal(err)
			}
			// Model a completed external effect followed by absent or undecodable CLI
			// output. The recovery entry point intentionally receives neither output.
			effect := filepath.Join(home, "fixture-owned-effect")
			if err := os.WriteFile(effect, []byte("live"), 0600); err != nil {
				t.Fatal(err)
			}
			unrelated := filepath.Join(t.TempDir(), "unrelated")
			if err := os.WriteFile(unrelated, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			var response domain.Lease
			_ = json.Unmarshal([]byte(output), &response)
			if response.ID != "" {
				t.Fatal("fixture unexpectedly provided a cleanup ID")
			}
			calls := 0
			if err := cleanupIntegrationRegistry(home, func(id string) (domain.Lease, error) {
				calls++
				if id != lease.ID {
					return domain.Lease{}, fmt.Errorf("foreign cleanup id %s", id)
				}
				if err := os.Remove(effect); err != nil {
					return domain.Lease{}, err
				}
				lease.Observed = "released"
				return lease, db.Save(context.Background(), lease)
			}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("persisted effect never reached cleanup: %d", calls)
			}
			if _, err := os.Stat(effect); !os.IsNotExist(err) {
				t.Fatal("missing response lost cleanup ownership")
			}
			if b, err := os.ReadFile(unrelated); err != nil || string(b) != "keep" {
				t.Fatal("unrelated data changed")
			}
		})
	}
}

func TestIntegrationRegistryRecoveryPreservesUnconfirmedState(t *testing.T) {
	for _, mode := range []string{"destroy-error", "wrong-id", "not-released", "corrupt-registry", "absent-registry"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, "state.db")
			if mode == "corrupt-registry" {
				if err := os.WriteFile(path, []byte("invalid SQLite"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if mode != "absent-registry" {
				db, err := sqlite.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Reserve(context.Background(), domain.Lease{ID: "owned", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}, 0); err != nil {
					db.Close()
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			}
			evidence := filepath.Join(home, "retain")
			if err := os.WriteFile(evidence, []byte("recovery evidence"), 0600); err != nil {
				t.Fatal(err)
			}
			called := 0
			err := cleanupIntegrationRegistry(home, func(id string) (domain.Lease, error) {
				called++
				switch mode {
				case "destroy-error":
					return domain.Lease{}, errors.New("injected cleanup failure")
				case "wrong-id":
					return domain.Lease{ID: "foreign", Observed: "released"}, nil
				default:
					return domain.Lease{ID: id, Observed: "quarantined"}, nil
				}
			})
			if mode == "absent-registry" {
				if err != nil || called != 0 {
					t.Fatalf("absent registry invoked cleanup: %v", err)
				}
			} else if err == nil {
				t.Fatal("unconfirmed cleanup accepted")
			}
			if mode == "corrupt-registry" && called != 0 {
				t.Fatal("invalid registry authorized destruction")
			}
			if b, e := os.ReadFile(evidence); e != nil || string(b) != "recovery evidence" {
				t.Fatal("recovery evidence lost")
			}
		})
	}
}
