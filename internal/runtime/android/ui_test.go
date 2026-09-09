package android

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/runtime/android/uihelper"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func uiEnvelope(raw string) string {
	return "INSTRUMENTATION_RESULT: response64=" + base64.StdEncoding.EncodeToString([]byte(raw)) + "\nINSTRUMENTATION_CODE: 0\n"
}
func uiValidResponse(t *testing.T) string {
	t.Helper()
	b, e := json.Marshal(domain.UIObservation{Version: 1, Status: "ok", Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: strings.Repeat("a", 64)}}}})
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func TestUIResponseCompletionAndStructure(t *testing.T) {
	raw := uiValidResponse(t)
	if _, err := decodeUIResponse(uiEnvelope(raw)); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"missing-code": strings.ReplaceAll(uiEnvelope(raw), "INSTRUMENTATION_CODE: 0", ""), "nonzero-code": strings.ReplaceAll(uiEnvelope(raw), "INSTRUMENTATION_CODE: 0", "INSTRUMENTATION_CODE: 1"), "duplicate-result": uiEnvelope(raw) + uiEnvelope(raw), "duplicate-code": uiEnvelope(raw) + "INSTRUMENTATION_CODE: 0\n", "conflicting-code": uiEnvelope(raw) + "INSTRUMENTATION_CODE: 1\n", "bad-json": uiEnvelope("{"), "bad-base64": "INSTRUMENTATION_RESULT: response64=!\nINSTRUMENTATION_CODE: 0\n", "unknown-status": uiEnvelope(strings.Replace(raw, `"status":"ok"`, `"status":"invented"`, 1)), "wrong-version": uiEnvelope(strings.Replace(raw, `"version":1`, `"version":2`, 1)), "nonhex-fingerprint": uiEnvelope(strings.Replace(raw, strings.Repeat("a", 64), strings.Repeat("z", 64), 1)), "oversized": strings.Repeat("x", 2*uiResponseLimit+1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeUIResponse(value); err == nil {
				t.Fatal("invalid helper response accepted")
			}
		})
	}
}
func TestUIResponseRejectsSensitiveNodeFields(t *testing.T) {
	for _, flag := range []string{"editable", "password"} {
		for _, field := range []string{"text", "description", "hint"} {
			t.Run(flag+"/"+field, func(t *testing.T) {
				raw := `{"version":1,"status":"ok","snapshot":{"nodes":[{"ref":"n1","fingerprint":"` + strings.Repeat("a", 64) + `","` + flag + `":true,"` + field + `":"private payload"}]}}`
				o, err := decodeUIResponse(uiEnvelope(raw))
				if err == nil {
					t.Fatalf("sensitive field accepted: %s", field)
				}
				if strings.Contains(string(o.Raw), "private payload") {
					t.Fatal("rejected payload returned as retainable raw evidence")
				}
			})
		}
	}
}
func TestUIRejectsInvalidRequestsWithoutEffects(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	for _, q := range []domain.UIRequest{{Version: 2, Operation: "home"}, {Version: 1, Operation: "shell"}, {Version: 1, Operation: "home", Package: "com.app;id"}, {Version: 1, Operation: "tap-coordinate", X: -1}, {Version: 1, Operation: "swipe", DurationMS: 49}, {Version: 1, Operation: "logcat", Package: "com.example.app", SinceSeconds: 0}} {
		if _, err := a.ObserveUI(context.Background(), r, q); err == nil {
			t.Fatalf("invalid request accepted: %+v", q)
		}
	}
	if len(runner.commands) != 0 {
		t.Fatal("invalid request reached device")
	}
}
func TestUINavigationUsesExactOwnedSerial(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	t.Setenv("ANDROID_SERIAL", "foreign")
	t.Setenv("ADB_SERVER_SOCKET", "tcp:foreign:6000")
	for _, op := range []string{"back", "home"} {
		key := "shell input keyevent 4"
		if op == "home" {
			key = "shell input keyevent 3"
		}
		runner.output[key] = execx.Result{}
		o, err := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: op})
		if err != nil || !o.ActionPerformed || !o.Confirmed {
			t.Fatalf("navigation failed: %+v %v", o, err)
		}
	}
	for _, c := range runner.commands {
		if !reflect.DeepEqual(c.Args[:6], []string{"-H", "127.0.0.1", "-P", "5037", "-s", r.Android.Serial}) || !reflect.DeepEqual(c.UnsetEnv, adbRoutingEnvironment()) || c.CaptureLimit <= 0 {
			t.Fatalf("unsafe command %+v", c)
		}
	}
}
func TestUILogPIDAttributionAndBounds(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	runner.output["shell pidof com.example.app"] = execx.Result{Stdout: "1234\n"}
	runner.output["shell date +%s"] = execx.Result{Stdout: "1000\n"}
	runner.output["shell logcat -d -v epoch --pid 1234 -t 2001"] = execx.Result{Stdout: "900.000 1234 1234 I Tag: old\n999.000 1234 1234 I Tag: current\n"}
	o, err := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "logcat", Package: "com.example.app", SinceSeconds: 30})
	if err != nil || o.PID != 1234 || o.Since != "970.000" || strings.Contains(string(o.Binary), "old") || !strings.Contains(string(o.Binary), "current") {
		t.Fatalf("bad attribution: %+v %v", o, err)
	}
	for _, pid := range []string{"", "1234 5678", "-1", "not-pid"} {
		runner.output["shell pidof com.example.app"] = execx.Result{Stdout: pid}
		if _, err = a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "logcat", Package: "com.example.app", SinceSeconds: 30}); err == nil {
			t.Fatalf("ambiguous PID accepted %q", pid)
		}
	}
	t.Setenv("AGENT_ENV_TEST_SECRET", "private-value")
	raw, truncated := boundedUILog(strings.Repeat("999.000 1234 1234 I Tag: private-value "+strings.Repeat("x", 200)+"\n", 2001), 970)
	if len(raw) > 256<<10 || !truncated || strings.Contains(string(raw), "private-value") {
		t.Fatal("log evidence not bounded/redacted")
	}
}
func TestUISafeErrorDropsPayloadButPreservesMarkers(t *testing.T) {
	e := &execx.ExitError{Command: execx.Command{Name: "adb", Args: []string{"private payload"}}, Result: execx.Result{Stderr: "private payload"}, Err: errors.Join(context.DeadlineExceeded, execx.ErrOutputIncomplete)}
	err := uiSafeError(e)
	if strings.Contains(err.Error(), "private") || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, execx.ErrOutputIncomplete) {
		t.Fatalf("unsafe error %v", err)
	}
	var exit *execx.ExitError
	if errors.As(err, &exit) {
		t.Fatal("request-bearing ExitError retained")
	}
}

func TestUIOwnershipFailureCannotDispatchInput(t *testing.T) {
	a, r, p, runner, _ := applicationFixture(t)
	p.mu.Lock()
	p.name = "foreign-avd"
	p.mu.Unlock()
	_, err := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "home"})
	if !errors.Is(err, domain.ErrResourceIdentity) {
		t.Fatalf("missing identity rejection: %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatal("input dispatched despite mismatched ownership")
	}
}
func TestUILogRejectsNonfiniteTimestamps(t *testing.T) {
	got, _ := boundedUILog("NaN 1234 1234 I Tag: invalid\n+Inf 1234 1234 I Tag: invalid\n999.000 1234 1234 I Tag: valid\n", 970)
	if strings.Contains(string(got), "invalid") || !strings.Contains(string(got), "valid") {
		t.Fatalf("nonfinite timestamp retained: %q", got)
	}
}

func TestUIInstallSuccessMayHaveIncrementalTrailers(t *testing.T) {
	for _, output := range []string{"Success\n", "Serving...\nPerforming Incremental Install\nSuccess\nInstall command complete in 330 ms\n", "Performing Streamed Install\r\nSuccess\r\n"} {
		if !uiInstallConfirmed(output) {
			t.Fatalf("successful install rejected: %q", output)
		}
	}
	for _, output := range []string{"", "Install command complete in 330 ms", "Success\nSuccess", "Success\nFailure [INSTALL_FAILED_INVALID_APK]", "Success\nError: package unavailable"} {
		if uiInstallConfirmed(output) {
			t.Fatalf("unconfirmed install accepted: %q", output)
		}
	}
}
func TestUIAbsentHelperPathRequiresCleanExitOne(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	key := "shell pm path " + uihelper.Package
	for _, tc := range []struct {
		name   string
		result execx.Result
		cause  error
		absent bool
	}{{"absent", execx.Result{ExitCode: 1}, errors.New("exit status 1"), true}, {"diagnostic", execx.Result{ExitCode: 1, Stderr: "device unavailable"}, errors.New("exit status 1"), false}, {"incomplete", execx.Result{ExitCode: 1}, execx.ErrOutputIncomplete, false}, {"unconfirmed", execx.Result{ExitCode: 1}, execx.ErrProcessTreeUnconfirmed, false}, {"other-exit", execx.Result{ExitCode: 2}, errors.New("exit status 2"), false}} {
		t.Run(tc.name, func(t *testing.T) {
			runner.output[key] = tc.result
			runner.failure[key] = &execx.ExitError{Command: execx.Command{Name: "adb"}, Result: tc.result, Err: tc.cause}
			path, err := a.uiHelperPath(context.Background(), r)
			if tc.absent {
				if err != nil || path != "" {
					t.Fatalf("absence rejected: %q %v", path, err)
				}
			} else if err == nil {
				t.Fatal("uncertain absence accepted")
			}
		})
	}
}

func TestUISemanticActionRejectsChangedBackendBeforeDeviceInput(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	dir := t.TempDir()
	apk := []byte("verified test fixture")
	if err := os.WriteFile(filepath.Join(dir, "observer.apk"), apk, 0600); err != nil {
		t.Fatal(err)
	}
	meta := uihelper.Metadata{Version: uihelper.Version, Package: uihelper.Package, SourceSHA256: uihelper.SourceDigest(), APKSHA256: fmt.Sprintf("%x", sha256.Sum256(apk)), Platform: "android-35", BuildTools: "36.0.0"}
	b, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "observer.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_ENV_UI_HELPER", dir)
	for _, op := range []string{"tap", "set-text"} {
		o, err := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: op, ExpectedFingerprint: strings.Repeat("a", 64), ExpectedBackend: "old-helper-build", Text: "private text"})
		if err != nil || o.Status != "stale" || !o.Confirmed || o.ActionPerformed {
			t.Fatalf("backend change accepted: %+v %v", o, err)
		}
	}
	if len(runner.commands) != 0 {
		t.Fatal("changed backend reached device")
	}
}

type uiStagingRunner struct {
	base      execx.Runner
	t         *testing.T
	directory string
	staged    string
	apk       []byte
	fail      bool
}

func (r *uiStagingRunner) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	if len(c.Args) == 8 && c.Args[6] == "install" {
		r.staged = c.Args[7]
		if filepath.Dir(r.staged) != r.directory {
			r.t.Errorf("helper staged outside owned runtime: %s", r.staged)
		}
		data, err := os.ReadFile(r.staged)
		if err != nil || string(data) != string(r.apk) {
			r.t.Errorf("staged helper bytes: %q %v", data, err)
		}
		if r.fail {
			return execx.Result{}, errors.New("fixture install failure")
		}
		return execx.Result{Stdout: "Success\n"}, nil
	}
	return r.base.Run(ctx, c)
}
func TestUIHelperStagingStaysInOwnedRuntimeAndCleansUp(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			dir := t.TempDir()
			apk := []byte("verified staging fixture")
			writeFixture(t, filepath.Join(dir, "observer.apk"), string(apk))
			meta := uihelper.Metadata{Version: uihelper.Version, Package: uihelper.Package, SourceSHA256: uihelper.SourceDigest(), APKSHA256: fmt.Sprintf("%x", sha256.Sum256(apk)), Platform: "android-35", BuildTools: "36.0.0"}
			data, err := json.Marshal(meta)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, filepath.Join(dir, "observer.json"), string(data))
			t.Setenv("AGENT_ENV_UI_HELPER", dir)
			runner.output["shell getprop ro.build.version.sdk"] = execx.Result{Stdout: "35"}
			runner.output["shell pm path "+uihelper.Package] = execx.Result{}
			staging := &uiStagingRunner{base: runner, t: t, directory: r.Directory, apk: apk, fail: fail}
			a.Runner = staging
			// An empty package observation after a successful install deliberately stops
			// subsequent instrumentation; staging cleanup must hold on both paths.
			_, err = a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "snapshot"})
			if staging.staged == "" {
				t.Fatalf("helper never staged: %v", err)
			}
			if _, err := os.Stat(staging.staged); !os.IsNotExist(err) {
				t.Fatalf("staged helper not removed: %v", err)
			}
		})
	}
}
