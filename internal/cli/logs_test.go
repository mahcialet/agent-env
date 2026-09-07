package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

type logStore struct {
	app.Store
	artifacts []domain.Artifact
}

func (s logStore) Artifacts(context.Context, string) ([]domain.Artifact, error) {
	return s.artifacts, nil
}

type logRuntime struct {
	app.RuntimeProvider
	calls [][]string
}

func (r *logRuntime) Logs(_ context.Context, runtime domain.Runtime) (string, error) {
	r.calls = append(r.calls, append([]string(nil), runtime.Services...))
	return strings.Join(runtime.Services, " "), nil
}

func logLease() domain.Lease {
	return domain.Lease{ID: "lease", Desired: "active", Observed: "ready", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Components: []domain.Component{{Name: "api", Runtime: "shared", Services: []string{"db", "api"}}, {Name: "dashboard", Runtime: "shared", Services: []string{"dashboard"}}}, Runtimes: []domain.Runtime{{Name: "shared", Project: "ae_logs", Services: []string{"db", "api", "dashboard"}}}}
}

func TestLiveComponentLogsSelectExactServices(t *testing.T) {
	lease := logLease()
	runtime := &logRuntime{}
	service := &app.Service{Store: logStore{}, Runtime: runtime}
	entries, err := runtimeLogEntries(context.Background(), service, lease, "api")
	if err != nil {
		t.Fatal(err)
	}
	if entries["shared"] != "db api" || !reflect.DeepEqual(runtime.calls, [][]string{{"db", "api"}}) {
		t.Fatalf("component leaked unrelated service logs: %v calls=%v", entries, runtime.calls)
	}
	if !reflect.DeepEqual(lease.Runtimes[0].Services, []string{"db", "api", "dashboard"}) {
		t.Fatal("log filter mutated lease runtime")
	}
	entries, err = runtimeLogEntries(context.Background(), service, lease, "dashboard")
	if err != nil || entries["shared"] != "dashboard" {
		t.Fatalf("dashboard scope %v %v", entries, err)
	}
	if _, err := runtimeLogEntries(context.Background(), service, lease, "absent"); err == nil {
		t.Fatal("unknown component accepted")
	}
}

func TestArchivedComponentLogsThroughCLI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AGENT_ENV_HOME", home)
	db, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	lease := logLease()
	lease.Desired, lease.Observed = "released", "released"
	if err := db.Reserve(context.Background(), lease, 0); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"db", "api", "dashboard"} {
		path := filepath.Join(home, name+".log")
		if err := os.WriteFile(path, []byte("only-"+name), 0600); err != nil {
			t.Fatal(err)
		}
		if err := db.SaveArtifact(context.Background(), domain.Artifact{ID: name, LeaseID: lease.ID, Kind: "compose-log/shared/" + name, Path: path, CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	for _, component := range []string{"api", "dashboard", ""} {
		t.Run(component, func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			cmd := New(&out, &diagnostics)
			args := []string{"logs", lease.ID, "--output", "json"}
			if component != "" {
				args = append(args, "--component", component)
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var response struct {
				SchemaVersion int `json:"schema_version"`
				Data          struct {
					LeaseID string            `json:"lease_id"`
					Logs    map[string]string `json:"logs"`
				} `json:"data"`
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("invalid JSON %s: %v", out.String(), err)
			}
			want := map[string]string{"db": "only-db", "api": "only-api", "dashboard": "only-dashboard"}
			if component == "api" {
				delete(want, "dashboard")
			}
			if component == "dashboard" {
				delete(want, "api")
				delete(want, "db")
			}
			if response.SchemaVersion != 1 || response.Data.LeaseID != lease.ID || !reflect.DeepEqual(response.Data.Logs, want) {
				t.Fatalf("incorrect archive filter: %s", out.String())
			}
		})
	}
}

func TestLegacyAggregateCannotPretendComponentIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.log")
	if err := os.WriteFile(path, []byte("api and dashboard"), 0600); err != nil {
		t.Fatal(err)
	}
	lease := logLease()
	lease.Observed = "released"
	service := &app.Service{Store: logStore{artifacts: []domain.Artifact{{ID: "legacy", Kind: "compose-log", Path: path}}}}
	if _, err := runtimeLogEntries(context.Background(), service, lease, "api"); err == nil {
		t.Fatal("legacy aggregate falsely isolated")
	}
	all, err := runtimeLogEntries(context.Background(), service, lease, "")
	if err != nil || all["legacy"] != "api and dashboard" {
		t.Fatalf("legacy logs unavailable: %v %v", all, err)
	}
}

func TestAndroidProcessLogsActiveReleasedAndComponentIsolation(t *testing.T) {
	t.Setenv("ANDROID_LOG_TEST_TOKEN", "private-android-test-secret")
	for _, state := range []string{"ready", "released"} {
		t.Run(state, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("AGENT_ENV_HOME", home)
			db, err := sqlite.Open(filepath.Join(home, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			lease := logLease()
			lease.Observed = "ready"
			lease.Components = []domain.Component{{Name: "phone", Runtime: "device"}, {Name: "tablet", Runtime: "second"}}
			lease.Runtimes = nil
			for _, name := range []string{"device", "second"} {
				dir := filepath.Join(home, "leases", lease.ID, "android", name)
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				lease.Runtimes = append(lease.Runtimes, domain.Runtime{Name: name, Type: "android-emulator", Directory: dir, Android: &domain.AndroidEmulator{Template: "Pixel_API35", AVDName: "ae_" + name, AVDHome: filepath.Join(dir, "avd"), AVDPath: filepath.Join(dir, "avd", "ae_"+name+".avd")}})
				for _, filename := range []string{"emulator.stdout.log", "emulator.stderr.log", "adb-server.stdout.log", "adb-server.stderr.log"} {
					if err := os.WriteFile(filepath.Join(dir, filename), []byte(name+" "+filename+" private-android-test-secret"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := db.Reserve(context.Background(), lease, 0); err != nil {
				t.Fatal(err)
			}
			if state == "released" {
				saved, err := db.Get(context.Background(), lease.ID)
				if err != nil {
					t.Fatal(err)
				}
				saved.Observed, saved.Desired = "released", "released"
				if err := db.Save(context.Background(), saved); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			for _, component := range []string{"", "phone", "tablet"} {
				var out, diagnostic bytes.Buffer
				cmd := New(&out, &diagnostic)
				args := []string{"logs", lease.ID, "--output", "json"}
				if component != "" {
					args = append(args, "--component", component)
				}
				cmd.SetArgs(args)
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				var response struct {
					Data struct {
						Logs map[string]string `json:"logs"`
					} `json:"data"`
				}
				if err := json.Unmarshal(out.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				wantCount := 8
				if component != "" {
					wantCount = 4
				}
				if len(response.Data.Logs) != wantCount {
					t.Fatalf("missing retained process logs: %s", out.String())
				}
				for name, data := range response.Data.Logs {
					if strings.Contains(data, "private-android-test-secret") || !strings.Contains(data, "[REDACTED]") {
						t.Fatalf("unredacted process output: %q", data)
					}
					if component == "phone" && !strings.HasPrefix(name, "device/") || component == "tablet" && !strings.HasPrefix(name, "second/") {
						t.Fatalf("component leaked sibling logs: %s", out.String())
					}
				}
			}
		})
	}
}

func TestAndroidProcessLogsRejectUnsafeFilesAndIgnoreAbsentLogs(t *testing.T) {
	home := t.TempDir()
	runtime := domain.Runtime{Name: "phone", Directory: filepath.Join(home, "leases", "lease", "android", "phone")}
	if err := os.MkdirAll(runtime.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	if logs, err := androidProcessLogEntries(home, "lease", runtime); err != nil || len(logs) != 0 {
		t.Fatalf("absent diagnostics: %v %v", logs, err)
	}
	outside := filepath.Join(t.TempDir(), "private.log")
	if err := os.WriteFile(outside, []byte("must not read"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Run("outside-directory", func(t *testing.T) {
		altered := runtime
		altered.Directory = filepath.Dir(outside)
		if _, err := androidProcessLogEntries(home, "lease", altered); err == nil {
			t.Fatal("read unrelated directory")
		}
	})
	t.Run("symlink-log", func(t *testing.T) {
		if err := os.Symlink(outside, filepath.Join(runtime.Directory, "emulator.stdout.log")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := androidProcessLogEntries(home, "lease", runtime); err == nil {
			t.Fatal("read symlink log")
		}
	})
	t.Run("symlink-runtime", func(t *testing.T) {
		altered := runtime
		altered.Name = "sibling"
		altered.Directory = filepath.Join(filepath.Dir(runtime.Directory), altered.Name)
		if err := os.Symlink(filepath.Dir(outside), altered.Directory); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := androidProcessLogEntries(home, "lease", altered); err == nil {
			t.Fatal("read redirected runtime")
		}
	})
}
