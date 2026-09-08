//go:build flutterintegration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
	androidruntime "github.com/mahcialet/agent-env/internal/runtime/android"
	"github.com/mahcialet/agent-env/internal/runtime/compose"
	flutterruntime "github.com/mahcialet/agent-env/internal/runtime/flutter"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// Explicitly selecting this fixture requires all prerequisites. It never skips
// missing Flutter, Docker, SDK, AVD or acceleration. Normal Flutter/Gradle
// builds may download declared dependencies under already accepted licenses.
func TestRealFlutterAndroidBackendLease(t *testing.T) {
	realFlutterAndroidBackendLease(t, false)
}

func realFlutterAndroidBackendLease(t *testing.T, observe bool) {
	t.Helper()
	if observe && os.Getenv("AGENT_ENV_UI_HELPER") == "" {
		t.Fatal("set AGENT_ENV_UI_HELPER to a verified observer companion build")
	}
	template := os.Getenv("AGENT_ENV_ANDROID_TEMPLATE")
	if template == "" || strings.ContainsAny(template, "\r\n'\"\\/") {
		t.Fatal("set AGENT_ENV_ANDROID_TEMPLATE to an installed, stopped AVD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	for _, tool := range []string{"git", "flutter", "docker"} {
		if _, err := execx.LookPath(tool); err != nil {
			t.Fatalf("real Flutter integration prerequisite %s: %v", tool, err)
		}
	}
	flutter := flutterruntime.Adapter{}
	version, err := flutter.Doctor(ctx, "flutter")
	if err != nil {
		t.Fatal(err)
	}
	android := androidruntime.Adapter{}
	sdk, err := android.Validate(ctx, template)
	if err != nil {
		t.Fatalf("real Android prerequisite: %v", err)
	}
	if _, err := (compose.Client{Runner: execx.OSRunner{}, Policy: policy.Defaults()}).Doctor(ctx); err != nil {
		t.Fatalf("real Compose prerequisite: %v", err)
	}
	t.Logf("native=%s/%s Go=%s Flutter=%s template=%s", runtime.GOOS, runtime.GOARCH, runtime.Version(), version, template)
	root, err := os.MkdirTemp("", "agent-env Flutter integration 日本語 ")
	if err != nil {
		t.Fatal(err)
	}
	// No automatic TempDir removal: failed ownership/dirty-source cleanup must
	// retain resources and evidence for explicit recovery.
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retained Flutter integration evidence: %s", root)
		}
	})
	repository := filepath.Join(root, "source")
	run := func(directory, name string, args ...string) string {
		t.Helper()
		r, err := (execx.OSRunner{}).Run(ctx, execx.Command{Name: name, Args: args, Dir: directory, Timeout: 20 * time.Minute})
		if err != nil {
			t.Fatalf("fixture command %s %v: %v\n%s\n%s", name, args, err, r.Stdout, r.Stderr)
		}
		return r.Stdout
	}
	createArgs := []string{"create", "--platforms", "android", "--org", "dev.agentenv", "--project-name", "lease_fixture", repository}
	if os.Getenv("AGENT_ENV_FLUTTER_OFFLINE_FIXTURE") == "1" {
		createArgs = append(createArgs, "--offline")
	}
	run(root, "flutter", createArgs...)
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repository, filepath.FromSlash(name)), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/main.dart", `import 'dart:async';
import 'dart:io';
import 'package:flutter/material.dart';
void main() {
  runApp(const MaterialApp(home: Scaffold(body: Text('agent-env fixture'))));
  unawaited(probeBackend());
}
Future<void> probeBackend() async {
  final deadline = DateTime.now().add(const Duration(minutes: 10));
  while (DateTime.now().isBefore(deadline)) {
    final client = HttpClient()..connectionTimeout = const Duration(seconds: 2);
    try {
      final request = await client.getUrl(Uri.parse('http://127.0.0.1:8080/?agent-env-flutter-verified'));
      final response = await request.close().timeout(const Duration(seconds: 2));
      await response.drain<void>().timeout(const Duration(seconds: 2));
    } catch (_) {} finally { client.close(force: true); }
    await Future<void>.delayed(const Duration(seconds: 1));
  }
}
`)
	if observe {
		main, err := os.ReadFile(filepath.Join(repository, "lib", "main.dart"))
		if err != nil {
			t.Fatal(err)
		}
		write("lib/main.dart", strings.Replace(string(main), "runApp(const MaterialApp(home: Scaffold(body: Text('agent-env fixture'))));", "runApp(const MaterialApp(home: ObserverFixture()));", 1)+observerFlutterWidgets)
	}
	manifestPath := filepath.Join(repository, "android", "app", "src", "main", "AndroidManifest.xml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "<application") {
		t.Fatal("generated Android manifest lacks application")
	}
	write("android/app/src/main/AndroidManifest.xml", strings.Replace(string(manifest), "<application", "<application android:usesCleartextTraffic=\"true\"", 1))
	write("compose.yaml", `services:
  api:
    image: nginx:alpine
    ports:
      - target: 80
        host_ip: 127.0.0.1
        protocol: tcp
`)
	adbName := "adb"
	if runtime.GOOS == "windows" {
		adbName += ".exe"
	}
	adbArg, err := json.Marshal(filepath.Join(sdk.SDKPath, "platform-tools", adbName))
	if err != nil {
		t.Fatal(err)
	}
	write(".agent-env.yaml", fmt.Sprintf(`version: 1
sources:
  self: {repository: ., default_ref: HEAD}
runtimes:
  backend: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
  phone: {type: android-emulator, source: self, avd: '%s'}
applications:
  mobile-app:
    type: flutter-android
    source: self
    runtime: phone
    project_directory: .
    build:
      command: [flutter, build, apk, --debug]
      artifact: build/app/outputs/flutter-apk/app-debug.apk
      timeout: 25m
    package: dev.agentenv.lease_fixture
    activity: .MainActivity
    reverse: [{device_port: 8080, endpoint: api.http}]
components:
  api:
    runtime: backend
    compose_services: [api]
    endpoints:
      http: {service: api, target: 80, protocol: tcp}
  mobile: {runtime: phone, application: mobile-app, depends_on: [api]}
stacks:
  mobile: {roots: [mobile]}
tests:
  device-state:
    stack: mobile
    source: self
    command: [%s, -H, 127.0.0.1, -P, '5037', -s, '${android:phone:serial}', get-state]
    timeout: 30s
`, template, adbArg))
	if observe {
		manifest, err := os.ReadFile(filepath.Join(repository, ".agent-env.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		write(".agent-env.yaml", string(manifest)+fmt.Sprintf(`  resume-app:
    stack: mobile
    source: self
    command: [%s, -H, 127.0.0.1, -P, '5037', -s, '${android:phone:serial}', shell, am, start, -W, -n, dev.agentenv.lease_fixture/.MainActivity]
    timeout: 30s
`, adbArg))
	}
	run(repository, "flutter", "pub", "get")
	run(repository, "git", "init")
	run(repository, "git", "add", ".")
	run(repository, "git", "-c", "user.name=agent-env integration", "-c", "user.email=integration@example.invalid", "commit", "-m", "Flutter integration fixture")
	commit := strings.TrimSpace(run(repository, "git", "rev-parse", "HEAD"))
	home := filepath.Join(root, "state")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := serviceForStore(home, db, io.Discard, io.Discard)
	s.ReadinessTimeout = 5 * time.Minute
	s.ReadinessInterval = time.Second
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cleanupCancel()
		safe := true
		leases, err := db.List(cleanupCtx)
		if err != nil {
			safe = false
			t.Errorf("cleanup list: %v", err)
		}
		var retry []string
		for _, lease := range leases {
			released, err := s.Destroy(cleanupCtx, lease.ID, false, false)
			if err != nil || released.Observed != "released" {
				safe = false
				t.Errorf("cleanup lease %s requires recovery: %v state=%s", lease.ID, err, released.Observed)
				retry = append(retry, lease.ID)
			}
		}
		// Retain the failure above, but retry ordinary conservative cleanup after
		// every sibling has stopped. Never force-delete an ambiguous live group.
		for _, id := range retry {
			released, err := s.Destroy(cleanupCtx, id, false, false)
			if err != nil || released.Observed != "released" {
				t.Errorf("final cleanup retry lease %s: %v state=%s", id, err, released.Observed)
			} else {
				t.Logf("final conservative cleanup released lease %s; failure evidence retained", id)
			}
		}
		if err := db.Close(); err != nil {
			safe = false
			t.Errorf("close registry: %v", err)
		}
		if safe && !t.Failed() {
			if err := os.RemoveAll(root); err != nil {
				t.Errorf("remove released fixture: %v", err)
			}
		}
	})
	t.Log("creating two pinned Flutter + Compose + Emulator leases concurrently")
	type created struct {
		lease domain.Lease
		err   error
	}
	results := make(chan created, 2)
	for range 2 {
		go func() {
			lease, err := s.Create(ctx, app.PlanOptions{Repository: repository, Stack: "mobile"}, app.CreateOptions{Owner: "flutter-integration", TTL: time.Hour})
			results <- created{lease, err}
		}()
	}
	var leases []domain.Lease
	var failures []created
	for range 2 {
		result := <-results
		leases = append(leases, result.lease)
		if result.err != nil {
			failures = append(failures, result)
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			t.Errorf("real create lease=%s observed=%s: %v", failure.lease.ID, failure.lease.Observed, failure.err)
		}
		t.FailNow()
	}
	backendHTTP := func(lease domain.Lease) {
		t.Helper()
		endpoint := fmt.Sprintf("http://127.0.0.1:%d/", lease.Applications[0].Reverse[0].HostPort)
		request, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("backend status %d", response.StatusCode)
		}
	}
	guestRequests := func(lease domain.Lease) int {
		t.Helper()
		count := 0
		for _, r := range lease.Runtimes {
			if r.Type != "compose" {
				continue
			}
			logs, err := s.Runtime.Logs(ctx, r)
			if err != nil {
				t.Fatal(err)
			}
			count += strings.Count(logs, `GET /?agent-env-flutter-verified HTTP/1.1" 200`)
		}
		return count
	}
	devices := make([]domain.AndroidEmulator, 2)
	projects := make([]string, 2)
	for i, lease := range leases {
		if lease.Observed != "ready" || len(lease.Applications) != 1 {
			t.Fatalf("application not ready: %+v", lease)
		}
		a := lease.Applications[0]
		if a.SourceCommit != commit || a.Build.Version != version || len(a.InstalledDigest) != 64 || a.InstalledDigest != a.Build.Digest || len(a.Reverse) != 1 {
			t.Fatalf("build/install identity incomplete: %+v", a)
		}
		for _, r := range lease.Runtimes {
			if r.Type == "compose" {
				projects[i] = r.Project
			}
			if r.Name == a.Runtime && r.Android != nil {
				devices[i] = *r.Android
			}
		}
		if devices[i].Serial == "" {
			t.Fatal("application target serial missing")
		}
		t.Logf("ready lease=%s source=%s APK=%s serial=%s reverse=%+v", lease.ID, commit, a.InstalledDigest, devices[i].Serial, a.Reverse)
		backendHTTP(lease)
		deadline := time.Now().Add(2 * time.Minute)
		for {
			if guestRequests(lease) > 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("installed Flutter application did not reach its backend through adb reverse")
			}
			time.Sleep(time.Second)
		}
		shown, err := s.Show(ctx, lease.ID)
		if err != nil || shown.Observed != "ready" {
			t.Fatalf("real package/reverse reconcile: %v state=%s", err, shown.Observed)
		}
		command, err := s.Test(ctx, lease.ID, "device-state")
		if err != nil || command.Status != "passed" {
			t.Fatalf("named owned-device test: %+v %v", command, err)
		}
		if len(command.Notes) == 0 {
			t.Fatal("named test lacks current-worktree build semantics warning")
		}
	}
	if projects[0] == "" || projects[0] == projects[1] || leases[0].ID == leases[1].ID || leases[0].Sources[0].WorktreePath == leases[1].Sources[0].WorktreePath || leases[0].Applications[0].Build.ArtifactPath == leases[1].Applications[0].Build.ArtifactPath || devices[0].Serial == devices[1].Serial || devices[0].AVDName == devices[1].AVDName || devices[0].AVDPath == devices[1].AVDPath || devices[0].ConsolePort == devices[1].ConsolePort || devices[0].ADBPort == devices[1].ADBPort || leases[0].Applications[0].Reverse[0].HostPort == leases[1].Applications[0].Reverse[0].HostPort {
		t.Fatal("two real Flutter leases share source, APK, device, port or reverse resources")
	}
	if observe {
		t.Setenv("AGENT_ENV_HOME", home)
		exerciseRealAndroidUI(t, ctx, s, leases)
	}
	if released, err := s.Destroy(ctx, leases[0].ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("first lease cleanup: %v state=%s", err, released.Observed)
	}
	sibling, err := s.Show(ctx, leases[1].ID)
	if err != nil || sibling.Observed != "ready" {
		t.Fatalf("sibling changed after first cleanup: %v state=%s", err, sibling.Observed)
	}
	backendHTTP(sibling)
	if observe {
		shot := realUICommand(t, ctx, "snapshot", sibling.ID, "--application", "mobile-app")
		if shot.Snapshot == nil || shot.Snapshot.Serial != devices[1].Serial {
			t.Fatal("sibling UI identity changed after first destroy")
		}
		realUINode(t, shot, func(n domain.UINode) bool { return strings.Contains(n.Text+" "+n.Description, "Count0") }, "unmodified sibling counter")
	}
	// Establish the baseline only after the first Destroy completes; requests
	// received during shutdown do not prove the remaining guest still works.
	baseline := guestRequests(sibling)
	deadline := time.Now().Add(2 * time.Minute)
	for guestRequests(sibling) <= baseline {
		if time.Now().After(deadline) {
			t.Fatal("remaining Flutter guest stopped reaching its backend after sibling destroy")
		}
		time.Sleep(time.Second)
	}
	if released, err := s.Destroy(ctx, leases[1].ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("second lease cleanup: %v state=%s", err, released.Observed)
	}
	for _, lease := range leases {
		for _, r := range lease.Runtimes {
			if r.Type != "android-emulator" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(r.Directory, "emulator.stdout.log"))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "Netsim Wifi") {
					t.Logf("lease=%s emulator network observation: %s", lease.ID, line)
				}
			}
		}
	}
	t.Log("two concurrent real APK installs, package/reverse observations, activity launches, Flutter HTTP backend requests, named device tests, sibling isolation and cleanup passed")
}
