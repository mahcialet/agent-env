package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
)

const androidCLIManifest = `version: 1
sources:
  self: {repository: ., default_ref: HEAD}
runtimes:
  device: {type: android-emulator, source: self, avd: Pixel.API35}
components:
  device: {runtime: device, provides: [android-emulator]}
stacks:
  android: {roots: [device]}
`

func TestAndroidPlanJSONNeedsNeitherSDKNorDockerDaemon(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := originRepository(t, "Android source spaces 日本語")
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(androidCLIManifest), 0600); err != nil {
		t.Fatal(err)
	}
	head := originGit(t, repo, "rev-parse", "HEAD")
	state := filepath.Join(t.TempDir(), "unused-state")
	t.Setenv("AGENT_ENV_HOME", state)
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "missing-sdk"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	t.Setenv("DOCKER_CONTEXT", "")
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"plan", repo, "--stack", "android", "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pure Android plan: %v %s", err, stderr.String())
	}
	var got struct {
		SchemaVersion int      `json:"schema_version"`
		Data          app.Plan `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 1 || len(got.Data.Runtimes) != 1 || len(got.Data.Sources) != 1 || got.Data.Sources[0].Commit != head || stderr.Len() != 0 {
		t.Fatalf("bad plan envelope: %s %s", out.String(), stderr.String())
	}
	r := got.Data.Runtimes[0]
	if r.Type != "android-emulator" || r.Android == nil || r.Android.Template != "Pixel.API35" || r.Android.State != "planned" || r.Android.ConsolePort != 0 || r.Android.ADBPort != 0 || r.Android.Serial != "" || r.Android.AVDPath != "" || r.Android.ProcessID != 0 || r.Started {
		t.Fatalf("plan allocated or omitted Android requirement: %+v", r)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("pure plan created durable state: %v", err)
	}
	if got.Data.Sources[0].WorktreePath != "" {
		t.Fatal("pure plan allocated worktree")
	}
}

func TestAndroidDoctorMissingSDKStructuredWithoutDocker(t *testing.T) {
	// Doctor only needs to discover Git here; an inert native executable proves
	// it neither runs Git nor requires a Docker executable before SDK discovery.
	bin := t.TempDir()
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, name), nil, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "missing-sdk"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"doctor", "--runtime", "android-emulator", "--output", "json"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("missing SDK exit: %v", err)
	}
	var got struct {
		SchemaVersion int `json:"schema_version"`
		Data          struct {
			OK          bool              `json:"ok"`
			Diagnostics []string          `json:"diagnostics"`
			Runtime     map[string]string `json:"runtime"`
			Docker      string            `json:"docker"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out.String(), err)
	}
	if got.SchemaVersion != 1 || got.Data.OK || len(got.Data.Diagnostics) != 1 || !strings.Contains(got.Data.Diagnostics[0], "Android SDK prerequisite") || got.Data.Runtime["android"] != "unavailable" || got.Data.Docker != "" || stderr.Len() != 0 {
		t.Fatalf("bad Android diagnosis: %s %s", out.String(), stderr.String())
	}
	if strings.Contains(strings.ToLower(strings.Join(got.Data.Diagnostics, " ")), "missing docker") {
		t.Fatal("Android doctor depended on Docker")
	}
}

func TestAndroidDoctorRejectsUnsupportedRuntime(t *testing.T) {
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"doctor", "--runtime", "flutter", "--output", "json"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != 2 || out.Len() != 0 {
		t.Fatalf("invalid runtime accepted: %v %s", err, out.String())
	}
}

func TestAndroidDoctorLeaseIDInspectsRecordedLeaseWithoutPrerequisites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AGENT_ENV_HOME", home)
	s, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	l := domain.Lease{ID: "01K4H40HNM2M04XNNRTPYH33RR", Owner: "test", Desired: "released", Observed: "released", CreatedAt: now, HeartbeatAt: now, ExpiresAt: now.Add(time.Hour)}
	if err = s.Reserve(context.Background(), l, 0); err != nil {
		s.Close()
		t.Fatal(err)
	}
	s.Close()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "absent"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"doctor", l.ID, "--output", "json"})
	err = cmd.Execute()
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("released lease diagnosis: %v %s", err, out.String())
	}
	var got struct {
		SchemaVersion int `json:"schema_version"`
		Data          struct {
			Lease domain.Lease `json:"lease"`
			OK    bool         `json:"ok"`
		} `json:"data"`
	}
	if err = json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 1 || got.Data.Lease.ID != l.ID || got.Data.Lease.Observed != "released" || got.Data.OK {
		t.Fatalf("lease ID interpreted as repository/prerequisite request: %s", out.String())
	}
}

type manifestAndroidValidator struct {
	app.AndroidProvider
	calls   []string
	invalid map[string]error
}

func (a *manifestAndroidValidator) Validate(_ context.Context, template string) (domain.AndroidEmulator, error) {
	a.calls = append(a.calls, template)
	return domain.AndroidEmulator{Template: template}, a.invalid[template]
}

func TestAndroidRepositoryDoctorValidatesEverySelectedTemplate(t *testing.T) {
	repo := t.TempDir()
	manifest := strings.Replace(androidCLIManifest, "components:", "  second: {type: android-emulator, source: self, avd: Other_API35}\ncomponents:", 1)
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	for _, problem := range []string{"", "missing template", "locked template", "corrupt config", "unsupported image architecture"} {
		t.Run(problem, func(t *testing.T) {
			validator := &manifestAndroidValidator{invalid: map[string]error{}}
			if problem != "" {
				validator.invalid["Pixel.API35"] = errors.New(problem)
			}
			digest, err := doctorManifest(context.Background(), repo, "android-emulator", validator)
			if !reflect.DeepEqual(validator.calls, []string{"Pixel.API35", "Other_API35"}) {
				t.Fatalf("selected AVD validation missing: %v", validator.calls)
			}
			if problem == "" {
				if err != nil || digest == "" {
					t.Fatalf("valid templates: %q %v", digest, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "device (AVD Pixel.API35): "+problem) {
				t.Fatalf("invalid selected AVD reported healthy: %q %v", digest, err)
			}
		})
	}
	validator := &manifestAndroidValidator{}
	if _, err := doctorManifest(context.Background(), repo, "compose", validator); err != nil || len(validator.calls) != 0 {
		t.Fatalf("Compose doctor unexpectedly checked Android: %v %v", validator.calls, err)
	}
}

func TestAndroidRepositoryDoctorReportsSelectedAVDDiagnostics(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(androidCLIManifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "absent-sdk"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	state := filepath.Join(t.TempDir(), "unused-state")
	t.Setenv("AGENT_ENV_HOME", state)
	var out, diagnostics bytes.Buffer
	cmd := New(&out, &diagnostics)
	cmd.SetArgs([]string{"doctor", repo, "--runtime", "android-emulator", "--output", "json"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("unusable selected AVD accepted: %v %s", err, out.String())
	}
	var response struct {
		Data struct {
			OK          bool     `json:"ok"`
			Diagnostics []string `json:"diagnostics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.OK || !strings.Contains(strings.Join(response.Data.Diagnostics, "; "), "Android runtime device (AVD Pixel.API35)") {
		t.Fatalf("missing selected-runtime diagnostic: %s", out.String())
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("doctor allocated state: %v", err)
	}
}
