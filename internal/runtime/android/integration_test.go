//go:build androidintegration

package android

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

// Opt in with -tags=androidintegration and AGENT_ENV_ANDROID_TEMPLATE=<AVD>.
// Missing SDK/image/acceleration is a failure, never a passing skipped test.
func TestRealAndroidEmulatorLeases(t *testing.T) {
	template := os.Getenv("AGENT_ENV_ANDROID_TEMPLATE")
	if !safeName(template) {
		t.Fatal("real Android integration requires AGENT_ENV_ANDROID_TEMPLATE naming an installed, stopped AVD")
	}
	a := Adapter{}
	preflight, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	diagnostics, err := a.Doctor(preflight)
	if err != nil {
		t.Fatalf("Android integration prerequisite: %v", err)
	}
	t.Logf("native host=%s/%s Go=%s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				t.Logf("revision=%s", setting.Value)
			}
		}
	}
	keys := make([]string, 0, len(diagnostics))
	for key := range diagnostics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		t.Logf("%s=%s", key, diagnostics[key])
	}
	device, err := a.Validate(preflight, template)
	if err != nil {
		t.Fatalf("Android integration template: %v", err)
	}
	image, _ := filepath.Rel(device.SDKPath, device.SystemImage)
	t.Logf("system image=%s", filepath.ToSlash(image))
	root, err := os.MkdirTemp("", "agent-env Android integration 日本語 ")
	if err != nil {
		t.Fatal(err)
	}
	s, options, db := serviceFixture(t, root, template, a)
	s.ReadinessTimeout = 4 * time.Minute
	s.ReadinessInterval = time.Second
	t.Cleanup(func() {
		// Preserve files whenever resource termination cannot be established.
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		safe := true
		leases, err := db.List(ctx)
		if err != nil {
			safe = false
			t.Errorf("list integration cleanup: %v", err)
		}
		for _, lease := range leases {
			if released, err := s.Destroy(ctx, lease.ID, false, false); err != nil || released.Observed != "released" {
				safe = false
				t.Errorf("integration cleanup requires recovery: lease=%s error=%v", lease.ID, err)
			}
		}
		if err := db.Close(); err != nil {
			safe = false
			t.Errorf("close integration registry: %v", err)
		}
		if safe {
			if err := os.RemoveAll(root); err != nil {
				t.Errorf("remove released integration state: %v", err)
			}
		} else {
			t.Logf("retained Android recovery evidence at %s", root)
		}
	})
	type result struct {
		lease domain.Lease
		err   error
	}
	done := make(chan result, 2)
	ctx, cancelCreate := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancelCreate()
	for range 2 {
		go func() {
			lease, err := s.Create(ctx, options, app.CreateOptions{Owner: "android-integration"})
			done <- result{lease, err}
		}()
	}
	one, two := <-done, <-done
	if one.err != nil || two.err != nil {
		t.Fatalf("concurrent emulator create: %v; %v", one.err, two.err)
	}
	x, y := one.lease.Runtimes[0].Android, two.lease.Runtimes[0].Android
	if one.lease.Observed != "ready" || two.lease.Observed != "ready" || x.AVDName == y.AVDName || x.AVDHome == y.AVDHome || x.AVDPath == y.AVDPath || x.ConsolePort == y.ConsolePort || x.Serial == y.Serial {
		t.Fatalf("real emulator lease identities collide: %+v %+v", x, y)
	}
	t.Logf("ready leases: %s serial=%s AVD=%s; %s serial=%s AVD=%s", one.lease.ID, x.Serial, x.AVDName, two.lease.ID, y.Serial, y.AVDName)
	if released, err := s.Destroy(ctx, one.lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("destroy first emulator: %+v %v", released, err)
	}
	if sibling, err := s.Reconcile(ctx, two.lease.ID); err != nil || sibling.Observed != "ready" {
		t.Fatalf("sibling emulator disrupted: %+v %v", sibling, err)
	}
	// Simulate an operator stopping the second owned emulator outside app state.
	// Use the same verified console session rather than an unscoped adb target.
	c, err := connectConsole(ctx, y.ConsolePort, y.AVDName)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.command("kill")
	c.close()
	if err != nil {
		t.Fatalf("manual owned emulator stop: %v", err)
	}
	stopCtx, cancelStop := context.WithTimeout(ctx, 30*time.Second)
	defer cancelStop()
	for {
		alive, err := a.processes().Alive(stopCtx, execx.ProcessIdentity{PID: y.ProcessID, StartID: y.ProcessStart})
		if err == nil && !alive && portAvailable(y.ConsolePort) && portAvailable(y.ADBPort) {
			break
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-stopCtx.Done():
			timer.Stop()
			t.Fatalf("manual emulator termination incomplete: %v; observation=%v", stopCtx.Err(), err)
		case <-timer.C:
		}
	}
	if stale, err := s.Reconcile(ctx, two.lease.ID); err != nil || stale.Observed != "degraded" {
		t.Fatalf("manual emulator stop not reconciled degraded: %+v %v", stale, err)
	}
	t.Logf("manual termination reconciled lease %s to degraded", two.lease.ID)
	if released, err := s.Destroy(ctx, two.lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("destroy second emulator: %+v %v", released, err)
	}
}
