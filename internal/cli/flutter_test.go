package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	flutterruntime "github.com/mahcialet/agent-env/internal/runtime/flutter"
)

func TestFlutterBuildLogsRetainedAndComponentScoped(t *testing.T) {
	home := t.TempDir()
	lease := domain.Lease{ID: "flutter-logs", Observed: "released", Components: []domain.Component{{Name: "phone", Runtime: "device", Application: "first"}, {Name: "tablet", Runtime: "device", Application: "second"}}, Runtimes: []domain.Runtime{{Name: "device", Type: "android-emulator", Directory: filepath.Join(home, "leases", "flutter-logs", "android", "device")}}}
	var artifacts []domain.Artifact
	for _, name := range []string{"first", "second"} {
		path := filepath.Join(home, name+".log")
		if err := os.WriteFile(path, []byte(name+" build output"), 0600); err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, domain.Artifact{ID: name, Kind: "application-build/" + name, Path: path})
	}
	s := &app.Service{Home: home, Store: logStore{artifacts: artifacts}}
	for _, component := range []string{"", "phone", "tablet"} {
		entries, err := runtimeLogEntries(context.Background(), s, lease, component)
		if err != nil {
			t.Fatal(err)
		}
		if component == "" {
			if len(entries) != 2 {
				t.Fatalf("missing aggregate builds: %v", entries)
			}
			continue
		}
		name := "first"
		if component == "tablet" {
			name = "second"
		}
		if len(entries) != 1 || entries[name] != name+" build output" {
			t.Fatalf("component leaked sibling build logs: %v", entries)
		}
	}
}

const flutterCLIManifest = `version: 1
sources:
  self: {repository: ., default_ref: HEAD}
runtimes:
  device: {type: android-emulator, source: self, avd: Pixel.API35}
applications:
  app:
    type: flutter-android
    source: self
    runtime: device
    project_directory: .
    build: {command: [flutter-custom, build, apk, --debug], artifact: build/app.apk}
    package: com.example.app
    activity: .MainActivity
components:
  mobile: {runtime: device, application: app}
stacks:
  mobile: {roots: [mobile]}
`

type flutterDoctorFixture struct {
	flutterruntime.Adapter
	calls []string
	err   error
}

func (f *flutterDoctorFixture) Doctor(_ context.Context, executable string) (string, error) {
	f.calls = append(f.calls, executable)
	return "fixture-version", f.err
}

func TestFlutterRepositoryDoctorReportsConfiguredToolsAndProject(t *testing.T) {
	for _, failure := range []string{"", "project", "flutter", "avd"} {
		t.Run(failure, func(t *testing.T) {
			repo := t.TempDir()
			if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(flutterCLIManifest), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(repo, "android"), 0700); err != nil {
				t.Fatal(err)
			}
			if failure != "project" {
				if err := os.WriteFile(filepath.Join(repo, "pubspec.yaml"), []byte("name: fixture\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			flutter := &flutterDoctorFixture{}
			android := &manifestAndroidValidator{invalid: map[string]error{}}
			if failure == "flutter" {
				flutter.err = errors.New("Flutter unavailable")
			}
			if failure == "avd" {
				android.invalid["Pixel.API35"] = errors.New("AVD unavailable")
			}
			digest, report, err := doctorFlutterManifest(context.Background(), repo, flutter, android)
			if failure == "" && err != nil || failure != "" && err == nil {
				t.Fatalf("prerequisite %q: %v", failure, err)
			}
			if digest == "" || !reflect.DeepEqual(flutter.calls, []string{"flutter-custom"}) || !reflect.DeepEqual(android.calls, []string{"Pixel.API35"}) {
				t.Fatalf("missing selected prerequisites %s %v %v", digest, flutter.calls, android.calls)
			}
			entry := report["app"].(map[string]string)
			if entry["executable"] != "flutter-custom" || entry["flutter_version"] != "fixture-version" || entry["runtime"] != "device" {
				t.Fatalf("missing application evidence: %v", entry)
			}
		})
	}
}

func TestFlutterDoctorMissingExecutableDoesNotRequireDockerOrAllocateState(t *testing.T) {
	bin := t.TempDir()
	git := "git"
	if runtime.GOOS == "windows" {
		git += ".exe"
	}
	if err := os.WriteFile(filepath.Join(bin, git), nil, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	state := filepath.Join(t.TempDir(), "unused")
	t.Setenv("AGENT_ENV_HOME", state)
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"doctor", "--runtime", "flutter-android", "--output", "json"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("missing tool exit: %v", err)
	}
	var got struct {
		Data struct {
			OK          bool     `json:"ok"`
			Diagnostics []string `json:"diagnostics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	diagnostics := strings.Join(got.Data.Diagnostics, " ")
	if got.Data.OK || !strings.Contains(diagnostics, "Flutter version prerequisite") || strings.Contains(strings.ToLower(diagnostics), "docker") {
		t.Fatalf("wrong diagnostics: %s", out.String())
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("doctor allocated state: %v", err)
	}
}

func TestFlutterPlanJSONIsPureAndIncludesApplication(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := originRepository(t, "Flutter source spaces 日本語")
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(flutterCLIManifest), 0600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "unused")
	t.Setenv("AGENT_ENV_HOME", state)
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "missing-sdk"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"plan", repo, "--stack", "mobile", "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pure Flutter plan: %v %s", err, stderr.String())
	}
	var got struct {
		Data app.Plan `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data.Applications) != 1 || got.Data.Applications[0].Name != "app" || got.Data.Applications[0].Runtime != "device" || got.Data.Applications[0].Artifact != "build/app.apk" {
		t.Fatalf("missing application plan: %s", out.String())
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("plan allocated state: %v", err)
	}
}
