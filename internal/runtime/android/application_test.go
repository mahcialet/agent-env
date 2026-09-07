package android

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type applicationRunner struct {
	base     *testRunner
	commands []execx.Command
	output   map[string]execx.Result
	failure  map[string]error
}

func (r *applicationRunner) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	if len(c.Args) > 6 {
		operation := strings.Join(c.Args[6:], " ")
		if result, ok := r.output[operation]; ok {
			r.commands = append(r.commands, c)
			return result, r.failure[operation]
		}
	}
	return r.base.Run(ctx, c)
}
func applicationFixture(t *testing.T) (Adapter, domain.Runtime, *testProcess, *applicationRunner, string) {
	t.Helper()
	a, r, p := fixture(t)
	var err error
	r, err = a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	runner := &applicationRunner{base: a.Runner.(*testRunner), output: map[string]execx.Result{}, failure: map[string]error{}}
	a.Runner = runner
	apk := filepath.Join(t.TempDir(), "日本語 app with spaces.apk")
	writeFixture(t, apk, "APK fixture")
	runner.output["install -r "+apk] = execx.Result{Stdout: "Performing Streamed Install\nSuccess\n"}
	runner.output["shell pm list packages --user 0 com.example.app"] = execx.Result{Stdout: "package:com.example.app.other\npackage:com.example.app\n"}
	runner.output["reverse --no-rebind tcp:8080 tcp:49173"] = execx.Result{}
	runner.output["reverse --list"] = execx.Result{Stdout: "host-17 tcp:8080 tcp:49173\n"}
	runner.output["reverse --remove tcp:8080"] = execx.Result{}
	runner.output["shell am start -W --user 0 -n com.example.app/.MainActivity"] = execx.Result{Stdout: "Starting: Intent { cmp=com.example.app/.MainActivity }\nStatus: ok\nComplete\n"}
	t.Cleanup(func() {
		p.mu.Lock()
		p.name = r.Android.AVDName
		p.mu.Unlock()
		a.ProbeADB = func(context.Context) (int, bool, error) { return 41, true, nil }
		_ = a.Destroy(context.Background(), r)
	})
	return a, r, p, runner, apk
}

func TestApplicationOperationsUseOwnedSerialAndLocalServer(t *testing.T) {
	a, r, _, runner, apk := applicationFixture(t)
	ctx := context.Background()
	t.Setenv("ANDROID_SERIAL", "foreign-device")
	t.Setenv("ADB_SERVER_SOCKET", "tcp:remote.example:6000")
	if err := a.InstallAPK(ctx, r, apk); err != nil {
		t.Fatal(err)
	}
	if ok, err := a.PackageInstalled(ctx, r, "com.example.app"); err != nil || !ok {
		t.Fatalf("installed=%t %v", ok, err)
	}
	if err := a.Reverse(ctx, r, 8080, 49173); err != nil {
		t.Fatal(err)
	}
	if got, err := a.ReverseMappings(ctx, r); err != nil || !reflect.DeepEqual(got, map[int]int{8080: 49173}) {
		t.Fatalf("reverse=%v %v", got, err)
	}
	if err := a.RemoveReverse(ctx, r, 8080, 49173); err != nil {
		t.Fatal(err)
	}
	if err := a.LaunchActivity(ctx, r, "com.example.app", ".MainActivity"); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 7 {
		t.Fatalf("operations=%d", len(runner.commands))
	}
	for _, c := range runner.commands {
		if c.Name != executable(r.Android.SDKPath, "platform-tools", "adb") || !reflect.DeepEqual(c.Args[:6], []string{"-H", "127.0.0.1", "-P", "5037", "-s", r.Android.Serial}) {
			t.Fatalf("unscoped command: %+v", c)
		}
		if !reflect.DeepEqual(c.UnsetEnv, adbRoutingEnvironment()) || c.Timeout <= 0 {
			t.Fatalf("missing execution constraints: %+v", c)
		}
	}
	if got := runner.commands[0].Args[len(runner.commands[0].Args)-1]; got != apk {
		t.Fatalf("APK argv changed: %q", got)
	}
}

func TestApplicationOperationsRefuseOwnershipOrServerMismatch(t *testing.T) {
	operations := []string{"install", "package", "reverse", "list", "remove", "launch"}
	for _, mode := range []string{"console", "server", "absent-server"} {
		for _, operation := range operations {
			t.Run(mode+"/"+operation, func(t *testing.T) {
				a, r, p, runner, apk := applicationFixture(t)
				switch mode {
				case "console":
					p.mu.Lock()
					p.name = "unrelated-avd"
					p.mu.Unlock()
				case "server":
					a.ProbeADB = func(context.Context) (int, bool, error) { return 40, true, nil }
				case "absent-server":
					a.ProbeADB = func(context.Context) (int, bool, error) { return 0, false, nil }
				}
				ctx := context.Background()
				var err error
				switch operation {
				case "install":
					err = a.InstallAPK(ctx, r, apk)
				case "package":
					_, err = a.PackageInstalled(ctx, r, "com.example.app")
				case "reverse":
					err = a.Reverse(ctx, r, 8080, 49173)
				case "list":
					_, err = a.ReverseMappings(ctx, r)
				case "remove":
					err = a.RemoveReverse(ctx, r, 8080, 49173)
				case "launch":
					err = a.LaunchActivity(ctx, r, "com.example.app", ".MainActivity")
				}
				if err == nil {
					t.Fatal("unsafe device operation succeeded")
				}
				if mode == "console" && !errors.Is(err, domain.ErrResourceIdentity) {
					t.Fatalf("lost ownership diagnostic: %v", err)
				}
				if len(runner.commands) != 0 {
					t.Fatalf("device effects after failed preflight: %+v", runner.commands)
				}
			})
		}
	}
}

func TestApplicationChecksServerAgainAfterObservation(t *testing.T) {
	a, r, _, runner, apk := applicationFixture(t)
	count := 0
	a.ProbeADB = func(context.Context) (int, bool, error) {
		count++
		if count == 1 {
			return 41, true, nil
		}
		return 40, true, nil
	}
	if err := a.InstallAPK(context.Background(), r, apk); err == nil {
		t.Fatal("changed server accepted")
	}
	if len(runner.commands) != 0 || count != 2 {
		t.Fatalf("operation/probes=%d/%d", len(runner.commands), count)
	}
}

func TestReverseCleanupRequiresExactMapping(t *testing.T) {
	for _, mode := range []string{"changed", "absent", "matching"} {
		t.Run(mode, func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			if mode == "changed" {
				runner.output["reverse --list"] = execx.Result{Stdout: "host-17 tcp:8080 tcp:49999\n"}
			}
			if mode == "absent" {
				runner.output["reverse --list"] = execx.Result{}
			}
			err := a.RemoveReverse(context.Background(), r, 8080, 49173)
			if mode == "changed" {
				if !errors.Is(err, domain.ErrResourceIdentity) {
					t.Fatalf("mismatched reverse cleanup: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			removed := false
			for _, c := range runner.commands {
				if c.Args[7] == "--remove" {
					removed = true
				}
			}
			if removed != (mode == "matching") {
				t.Fatalf("unexpected removal %t", removed)
			}
		})
	}
	a, r, _, runner, _ := applicationFixture(t)
	runner.failure["reverse --no-rebind tcp:8080 tcp:49173"] = errors.New("cannot rebind existing socket")
	if err := a.Reverse(context.Background(), r, 8080, 49173); err == nil {
		t.Fatal("existing reverse overwrite accepted")
	}
}

func TestParseReverseMappings(t *testing.T) {
	for _, test := range []struct {
		input string
		want  map[int]int
	}{
		{"", map[int]int{}}, {"host-7 tcp:8080 tcp:49173\r\nhost-7 tcp:8081 tcp:49174\r\n", map[int]int{8080: 49173, 8081: 49174}},
	} {
		got, err := parseReverseMappings(test.input)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("%q: %v %v", test.input, got, err)
		}
	}
	for _, input := range []string{"tcp:8080 tcp:49173", "host tcp:8080 tcp:49173 extra", "host tcp:0 tcp:49173", "host tcp:65536 tcp:49173", "host tcp:abc tcp:1", "host tcp:08080 tcp:1", "host tcp:8080 localabstract:test", "host tcp:8080 tcp:49173\nhost tcp:8080 tcp:49173"} {
		if got, err := parseReverseMappings(input); err == nil {
			t.Fatalf("malformed %q accepted: %v", input, got)
		}
	}
}

func TestPackageListingRequiresExactIdentity(t *testing.T) {
	for _, test := range []struct {
		output  string
		want    bool
		invalid bool
	}{
		{"", false, false}, {"package:com.example.app.other\n", false, false}, {"package:com.example.app\r\n", true, false}, {"Error: package manager unavailable", false, true},
	} {
		t.Run(fmt.Sprintf("%q", test.output), func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			runner.output["shell pm list packages --user 0 com.example.app"] = execx.Result{Stdout: test.output}
			got, err := a.PackageInstalled(context.Background(), r, "com.example.app")
			if got != test.want || (err != nil) != test.invalid {
				t.Fatalf("installed=%t err=%v", got, err)
			}
		})
	}
}

func TestActivityLaunchReportsExitZeroFailures(t *testing.T) {
	for _, output := range []string{"Error type 3\nError: Activity class does not exist.", "Starting: Intent {}\njava.lang.SecurityException: denied", "Status: timeout\nComplete", "Status: ok\nError: denied", "Starting: Intent {}\n"} {
		if err := activityStartResult(output); err == nil {
			t.Fatalf("failed activity accepted: %q", output)
		}
	}
	if err := activityStartResult("Warning: Activity not started, intent delivered\nStatus: ok\r\nComplete"); err != nil {
		t.Fatal(err)
	}
	a, r, _, runner, _ := applicationFixture(t)
	runner.output["shell am start -W --user 0 -n com.example.app/.MainActivity"] = execx.Result{Stdout: "Error type 3"}
	if err := a.LaunchActivity(context.Background(), r, "com.example.app", ".MainActivity"); err == nil {
		t.Fatal("exit-zero am failure accepted")
	}
}

func TestApplicationInputsRejectImplicitOrShellDeviceCommands(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	ctx := context.Background()
	for _, pkg := range []string{"", "-x", "com.example.app;id", "com.example.$USER"} {
		if _, err := a.PackageInstalled(ctx, r, pkg); err == nil {
			t.Fatalf("unsafe package accepted %q", pkg)
		}
	}
	for _, activity := range []string{"", "MainActivity", ".MainActivity;id", ".Main$Activity", ".Main Activity", "../MainActivity"} {
		if err := a.LaunchActivity(ctx, r, "com.example.app", activity); err == nil {
			t.Fatalf("unsafe activity accepted %q", activity)
		}
	}
	if err := a.InstallAPK(ctx, r, "relative.apk"); err == nil {
		t.Fatal("relative APK accepted")
	}
	if err := a.InstallAPK(ctx, r, t.TempDir()); err == nil {
		t.Fatal("directory APK accepted")
	}
	if err := a.Reverse(ctx, r, 0, 49173); err == nil {
		t.Fatal("invalid port accepted")
	}
	if err := a.RemoveReverse(ctx, r, 8080, 65536); err == nil {
		t.Fatal("invalid expected port accepted")
	}
	if _, err := a.ReverseMappings(ctx, domain.Runtime{}); !errors.Is(err, domain.ErrResourceIdentity) {
		t.Fatalf("invalid runtime accepted %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatal("invalid input reached device operation")
	}
}

func TestApplicationFailureDiagnosticsAreRedactedAndBounded(t *testing.T) {
	t.Setenv("AGENT_ENV_TEST_SECRET", "private diagnostic value")
	a, r, _, runner, apk := applicationFixture(t)
	runner.output["shell am start -W --user 0 -n com.example.app/.MainActivity"] = execx.Result{
		Stdout: "Starting: Intent {}\nStatus: timeout\nprivate diagnostic value\n" + strings.Repeat("x", 4096),
		Stderr: "activity manager diagnostic\nprivate diagnostic value\x1b[31m",
	}
	err := a.LaunchActivity(context.Background(), r, "com.example.app", ".MainActivity")
	if err == nil {
		t.Fatal("timeout accepted")
	}
	for _, want := range []string{"did not report OK status", "Status: timeout", "activity manager diagnostic", "[REDACTED]", "[truncated]"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in diagnostic: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "private") || strings.ContainsAny(err.Error(), "\n\r\x1b") || len(err.Error()) > 2300 {
		t.Fatalf("unsafe or unbounded diagnostic: %v", err)
	}
	runner.output["install -r "+apk] = execx.Result{Stderr: "Failure [INSTALL_FAILED_TEST]: private diagnostic value"}
	err = a.InstallAPK(context.Background(), r, apk)
	if err == nil || !strings.Contains(err.Error(), "INSTALL_FAILED_TEST") || !strings.Contains(err.Error(), "[REDACTED]") || strings.Contains(err.Error(), "private") {
		t.Fatalf("install failure diagnostic: %v", err)
	}
}

func TestAPKInstallRequiresExplicitSuccess(t *testing.T) {
	a, r, _, runner, apk := applicationFixture(t)
	runner.output["install -r "+apk] = execx.Result{Stdout: "Failure [INSTALL_FAILED_INVALID_APK]"}
	if err := a.InstallAPK(context.Background(), r, apk); err == nil {
		t.Fatal("exit-zero install failure accepted")
	}
}
