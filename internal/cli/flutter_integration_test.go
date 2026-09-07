//go:build flutterintegration

package cli

import (
	"context"
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
	if _, err := android.Validate(ctx, template); err != nil {
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
	run(root, "flutter", "create", "--platforms", "android", "--org", "dev.agentenv", "--project-name", "lease_fixture", repository)
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
  for (var attempt = 0; attempt < 120; attempt++) {
    final client = HttpClient()..connectionTimeout = const Duration(seconds: 2);
    try {
      final request = await client.getUrl(Uri.parse('http://127.0.0.1:8080/?agent-env-flutter-verified'));
      final response = await request.close();
      await response.drain<void>();
      if (response.statusCode == 200) return;
    } catch (_) {} finally { client.close(force: true); }
    await Future<void>.delayed(const Duration(seconds: 1));
  }
}
`)
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
`, template))
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
		for _, lease := range leases {
			released, err := s.Destroy(cleanupCtx, lease.ID, false, false)
			if err != nil || released.Observed != "released" {
				safe = false
				t.Errorf("cleanup lease %s requires recovery: %v state=%s", lease.ID, err, released.Observed)
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
	t.Log("creating pinned Flutter + Compose + Emulator lease")
	lease, err := s.Create(ctx, app.PlanOptions{Repository: repository, Stack: "mobile"}, app.CreateOptions{Owner: "flutter-integration", TTL: time.Hour})
	if err != nil {
		t.Fatalf("real lease create %s: %v", lease.ID, err)
	}
	if lease.Observed != "ready" || len(lease.Applications) != 1 {
		t.Fatalf("application not ready: %+v", lease)
	}
	a := lease.Applications[0]
	if a.SourceCommit != commit || a.Build.Version != version || len(a.InstalledDigest) != 64 || a.InstalledDigest != a.Build.Digest || len(a.Reverse) != 1 {
		t.Fatalf("build/install identity incomplete: %+v", a)
	}
	serial := ""
	for _, r := range lease.Runtimes {
		if r.Name == a.Runtime && r.Android != nil {
			serial = r.Android.Serial
		}
	}
	if serial == "" {
		t.Fatal("application target serial missing")
	}
	t.Logf("ready lease=%s source=%s APK=%s serial=%s reverse=%+v", lease.ID, commit, a.InstalledDigest, serial, a.Reverse)
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/", a.Reverse[0].HostPort)
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
	deadline := time.Now().Add(2 * time.Minute)
	for {
		seen := false
		for _, r := range lease.Runtimes {
			if r.Type != "compose" {
				continue
			}
			logs, err := s.Runtime.Logs(ctx, r)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(logs, "GET /?agent-env-flutter-verified") {
				seen = true
			}
		}
		if seen {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("installed Flutter application did not reach backend through adb reverse")
		}
		time.Sleep(time.Second)
	}
	shown, err := s.Show(ctx, lease.ID)
	if err != nil || shown.Observed != "ready" {
		t.Fatalf("real package/reverse reconcile: %v state=%s", err, shown.Observed)
	}
	if released, err := s.Destroy(ctx, lease.ID, false, false); err != nil || released.Observed != "released" {
		t.Fatalf("real cleanup: %v state=%s", err, released.Observed)
	}
	t.Log("real APK install, package observation, reverse mapping, activity launch, Flutter HTTP backend request, show and cleanup passed")
}
