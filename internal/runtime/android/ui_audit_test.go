package android

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/runtime/android/uihelper"
)

func TestUIAuditNativePreflightRemainsConfirmed(t *testing.T) {
	a, r, p, runner, _ := applicationFixture(t)
	p.mu.Lock()
	p.name = "foreign-avd"
	p.mu.Unlock()
	o, e := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "home"})
	if e == nil {
		t.Fatal("no error")
	}
	if len(runner.commands) != 0 {
		t.Fatal("unexpected input")
	}
	if !o.Confirmed {
		t.Error("no input dispatched but confirmed false")
	}
}
func TestUIAuditHelperBackendCarriesVerifiedDigest(t *testing.T) {
	a, r, _, runner, _ := applicationFixture(t)
	d := t.TempDir()
	apk := []byte("fixture")
	m := uihelper.Metadata{Version: uihelper.Version, Package: uihelper.Package, SourceSHA256: uihelper.SourceDigest(), APKSHA256: fmt.Sprintf("%x", sha256.Sum256(apk))}
	b, _ := json.Marshal(m)
	if e := os.WriteFile(filepath.Join(d, "observer.apk"), apk, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "observer.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("AGENT_ENV_UI_HELPER", d)
	want := fmt.Sprintf("uiautomation-v%d:source=%s:apk=%s", m.Version, m.SourceSHA256, m.APKSHA256)
	runner.output["shell getprop ro.build.version.sdk"] = execx.Result{Stdout: "35\n"}
	runner.output["shell pm path "+uihelper.Package] = execx.Result{Stdout: "package:/data/app/observer/base.apk\n"}
	runner.output["shell sha256sum /data/app/observer/base.apk"] = execx.Result{Stdout: m.APKSHA256 + "  /data/app/observer/base.apk\n"}
	q := domain.UIRequest{Version: 1, Operation: "tap", ExpectedBackend: want}
	request, _ := json.Marshal(q)
	response, _ := json.Marshal(domain.UIObservation{Version: 1, Status: "ok", ActionPerformed: true})
	command := "shell am instrument -w -r -e request " + base64.StdEncoding.EncodeToString(request) + " " + uihelper.Runner
	runner.output[command] = execx.Result{Stdout: "INSTRUMENTATION_RESULT: response64=" + base64.StdEncoding.EncodeToString(response) + "\nINSTRUMENTATION_CODE: 0\n"}
	o, e := a.ObserveUI(context.Background(), r, q)
	if e != nil || o.Status != "ok" || !o.ActionPerformed || !o.Confirmed {
		t.Fatalf("matching build action failed: %+v %v", o, e)
	}
	if o.Backend != want {
		t.Errorf("verified helper identity lost: %q", o.Backend)
	}
}

func TestUILogExactTailUsesOverflowProof(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  bool
	}{{1999, false}, {2000, false}, {2001, true}} {
		t.Run(fmt.Sprint(tc.count), func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			runner.output["shell pidof com.example.app"] = execx.Result{Stdout: "1234\n"}
			runner.output["shell date +%s"] = execx.Result{Stdout: "1000\n"}
			var text strings.Builder
			for i := 0; i < tc.count; i++ {
				fmt.Fprintf(&text, "999.000 1234 1234 I Tag: line-%04d\n", i)
			}
			runner.output["shell logcat -d -v epoch --pid 1234 -t 2001"] = execx.Result{Stdout: text.String()}
			o, e := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "logcat", Package: "com.example.app", SinceSeconds: 30})
			want := tc.count
			if want > 2000 {
				want = 2000
			}
			if e != nil || o.Truncated != tc.want || strings.Count(string(o.Binary), "\n") != want {
				t.Fatalf("count=%d truncated=%v err=%v", strings.Count(string(o.Binary), "\n"), o.Truncated, e)
			}
			if tc.count > 2000 && strings.Contains(string(o.Binary), "line-0000") {
				t.Fatal("retained old prefix instead of newest tail")
			}
		})
	}
	raw := "960.000 1234 1234 I Tag: outside\n" + strings.Repeat("999.000 1234 1234 I Tag: current\n", 2000)
	_, truncated := boundedUILog(raw, 970)
	if truncated {
		t.Fatal("out-of-window probe falsely truncates complete requested window")
	}
}

func TestUINativePreflightThroughAppDoesNotBlockCleanup(t *testing.T) {
	a, _, p := fixture(t)
	s, options, db := serviceFixture(t, t.TempDir(), "Pixel", a)
	defer db.Close()
	// Reserve externally occupied pairs only in this isolated registry.
	for port := 5554; port <= 5682; port += 2 {
		if portAvailable(port) && portAvailable(port+1) {
			break
		}
		id := fmt.Sprintf("fixture-slot-%d", port)
		home := filepath.Join(s.Home, "fixture-reservations", id, "avd")
		l := domain.Lease{ID: id, Observed: "quarantined", Desired: "ready", CreatedAt: time.Now(), HeartbeatAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Runtimes: []domain.Runtime{{LeaseID: id, Name: "fixture", Type: "android-emulator", Android: &domain.AndroidEmulator{Template: "Pixel", AVDName: id, AVDHome: home, AVDPath: filepath.Join(home, id+".avd")}}}}
		if e := db.Reserve(context.Background(), l, 0); e != nil {
			t.Fatal(e)
		}
	}
	s.Policy.MaxActive = 65
	l, e := s.Create(context.Background(), options, app.CreateOptions{Owner: "ui-preflight-test"})
	if e != nil {
		t.Fatal(e)
	}
	s.AndroidUI = a
	p.mu.Lock()
	p.name = "foreign-avd"
	p.mu.Unlock()
	_, e = s.UI(context.Background(), l.ID, app.UIOptions{Operation: "home"})
	p.mu.Lock()
	p.name = l.Runtimes[0].Android.AVDName
	p.mu.Unlock()
	if e == nil || !strings.Contains(e.Error(), "resource ownership") {
		t.Fatalf("preflight error=%v", e)
	}
	runs, e := db.Runs(context.Background(), l.ID)
	if e != nil || len(runs) != 1 || runs[0].Status != "failed" {
		t.Fatalf("preflight left false running barrier: %+v %v", runs, e)
	}
	if released, e := s.Destroy(context.Background(), l.ID, false, false); e != nil || released.Observed != "released" {
		t.Fatalf("preflight blocked subsequent safe cleanup: %+v %v", released, e)
	}
}

func TestUIRecoveryUsesRecordedHelperWithAbsentOrReplacedHostFiles(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		t.Run(fmt.Sprint(replaced), func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			oldHash := fmt.Sprintf("%x", sha256.Sum256([]byte("old verified helper")))
			expected := fmt.Sprintf("uiautomation-v1:source=%s:apk=%s", uihelper.SourceDigest(), oldHash)
			d := filepath.Join(t.TempDir(), "missing")
			if replaced {
				if e := os.Mkdir(d, 0700); e != nil {
					t.Fatal(e)
				}
				apk := []byte("different new helper")
				m := uihelper.Metadata{Version: uihelper.Version, Package: uihelper.Package, SourceSHA256: uihelper.SourceDigest(), APKSHA256: fmt.Sprintf("%x", sha256.Sum256(apk))}
				data, _ := json.Marshal(m)
				if e := os.WriteFile(filepath.Join(d, "observer.apk"), apk, 0600); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(d, "observer.json"), data, 0600); e != nil {
					t.Fatal(e)
				}
			}
			t.Setenv("AGENT_ENV_UI_HELPER", d)
			runner.output["shell getprop ro.build.version.sdk"] = execx.Result{Stdout: "35\n"}
			runner.output["shell am force-stop "+uihelper.Package] = execx.Result{}
			runner.output["shell pidof "+uihelper.Package] = execx.Result{ExitCode: 1}
			runner.output["shell pm path "+uihelper.Package] = execx.Result{Stdout: "package:/data/app/observer/base.apk\n"}
			runner.output["shell sha256sum /data/app/observer/base.apk"] = execx.Result{Stdout: oldHash + "  /data/app/observer/base.apk\n"}
			runner.failure["shell pidof "+uihelper.Package] = &execx.ExitError{Result: execx.Result{ExitCode: 1}, Err: fmt.Errorf("exit status 1")}
			o, e := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: "quiesce", ExpectedBackend: expected})
			if e != nil || !o.Confirmed || o.Status != "ok" || o.Backend != expected {
				t.Fatalf("recorded recovery failed: %+v %v", o, e)
			}
			var commands []string
			for _, c := range runner.commands {
				commands = append(commands, strings.Join(c.Args, " "))
			}
			joined := strings.Join(commands, "\n")
			if !strings.Contains(joined, "shell am force-stop "+uihelper.Package) || strings.Contains(joined, "install ") {
				t.Fatalf("wrong recovery effects: %s", joined)
			}
		})
	}
}

func TestUINativeDispatchedFailureRemainsUnconfirmed(t *testing.T) {
	for _, tc := range []struct{ operation, command string }{{"home", "shell input keyevent 3"}, {"back", "shell input keyevent 4"}, {"tap-coordinate", "shell input tap 0 0"}, {"swipe", "shell input swipe 0 0 0 0 100"}} {
		t.Run(tc.operation, func(t *testing.T) {
			a, r, _, runner, _ := applicationFixture(t)
			runner.output[tc.command] = execx.Result{}
			runner.failure[tc.command] = fmt.Errorf("command failed after dispatch")
			o, e := a.ObserveUI(context.Background(), r, domain.UIRequest{Version: 1, Operation: tc.operation, DurationMS: 100})
			if e == nil || o.Confirmed || len(runner.commands) != 1 {
				t.Fatalf("dispatched failure lost uncertainty: %+v %v commands%d", o, e, len(runner.commands))
			}
		})
	}
}
