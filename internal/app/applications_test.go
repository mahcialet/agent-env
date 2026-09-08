package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
)

const mobileManifest = `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  backend: {type: compose, source: self, project_directory: ., files: [compose.yaml]}
  phone: {type: android-emulator, source: self, avd: pixel}
applications:
  client:
    type: flutter-android
    source: self
    runtime: phone
    project_directory: .
    build: {command: [flutter, build, apk, --debug], artifact: build/app.apk}
    package: com.example.app
    activity: .MainActivity
    reverse: [{device_port: 8080, endpoint: api.http}]
components:
  api:
    runtime: backend
    compose_services: [api]
    endpoints: {http: {service: api, target: 8080}}
  dashboard:
    runtime: backend
    compose_services: [dashboard]
    depends_on: [api]
  mobile: {runtime: phone, application: client, depends_on: [api]}
stacks:
  api: {roots: [api]}
  dashboard: {roots: [dashboard]}
  mobile: {roots: [mobile]}
  full: {roots: [mobile, dashboard]}
`

type flutterFake struct {
	mu           sync.Mutex
	fail         bool
	builds       int
	prerequisite error
	buildErr     error
}

func (f *flutterFake) Validate(string, string, string) (string, error) { return "", nil }
func (f *flutterFake) Doctor(context.Context, string) (string, error) {
	return "3.fixture", f.prerequisite
}
func (f *flutterFake) Build(_ context.Context, root, dir string, argv []string, artifact string, _ time.Duration) (domain.ApplicationBuildResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.builds++
	result := domain.ApplicationBuildResult{Executable: argv[0], Version: "3.fixture", Directory: filepath.Join(root, dir), ArtifactPath: filepath.Join(root, dir, artifact), Stdout: "build evidence"}
	if f.buildErr != nil {
		return result, f.buildErr
	}
	if f.fail {
		return result, errors.New("build failed")
	}
	if err := os.MkdirAll(filepath.Dir(result.ArtifactPath), 0700); err != nil {
		return result, err
	}
	if err := os.WriteFile(result.ArtifactPath, []byte(root), 0600); err != nil {
		return result, err
	}
	result.Digest, _ = evidence.FileDigest(result.ArtifactPath)
	return result, nil
}

type applicationsFake struct {
	mu       sync.Mutex
	store    Store
	packages map[string]bool
	mappings map[string]map[int]int
	fail     string
}

func (f *applicationsFake) InstallAPK(ctx context.Context, r domain.Runtime, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, e := f.store.Get(ctx, r.LeaseID)
	if e != nil {
		return e
	}
	if len(l.Applications) != 1 || l.Applications[0].State != "installing" || l.Applications[0].Build.ArtifactPath != path {
		return errors.New("install without durable APK intent")
	}
	if f.fail == "install" {
		return errors.New("install failed")
	}
	f.packages[r.Android.Serial] = true
	return nil
}
func (f *applicationsFake) PackageInstalled(_ context.Context, r domain.Runtime, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.packages[r.Android.Serial], nil
}
func (f *applicationsFake) Reverse(ctx context.Context, r domain.Runtime, d, h int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, e := f.store.Get(ctx, r.LeaseID)
	if e != nil {
		return e
	}
	if !l.Applications[0].Reverse[0].Requested || l.Applications[0].Reverse[0].HostPort != h {
		return errors.New("reverse without durable intent")
	}
	if f.mappings[r.Android.Serial] == nil {
		f.mappings[r.Android.Serial] = map[int]int{}
	}
	f.mappings[r.Android.Serial][d] = h
	if f.fail == "reverse" {
		return errors.New("reverse failed after effect")
	}
	return nil
}
func (f *applicationsFake) ReverseMappings(_ context.Context, r domain.Runtime) (map[int]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int]int{}
	for k, v := range f.mappings[r.Android.Serial] {
		out[k] = v
	}
	return out, nil
}
func (f *applicationsFake) RemoveReverse(_ context.Context, r domain.Runtime, d, h int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail == "cleanup" {
		return domain.ErrResourceIdentity
	}
	if got := f.mappings[r.Android.Serial][d]; got != 0 && got != h {
		return domain.ErrResourceIdentity
	}
	delete(f.mappings[r.Android.Serial], d)
	return nil
}
func (f *applicationsFake) LaunchActivity(ctx context.Context, r domain.Runtime, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, e := f.store.Get(ctx, r.LeaseID)
	if e != nil {
		return e
	}
	if l.Applications[0].State != "launching" || !l.Applications[0].Reverse[0].Established {
		return errors.New("launch without durable mapping")
	}
	if f.fail == "launch" {
		return errors.New("launch failed")
	}
	return nil
}

// Independent endpoint ports derive from the real SQLite-reserved Android slot,
// while Compose project isolation and runtime intent use the existing fixture.
type mobileRuntime struct {
	*lifecycleRuntime
	mu    sync.Mutex
	ports map[string]int
}

func (f *mobileRuntime) Up(ctx context.Context, r domain.Runtime) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, e := f.store.Get(ctx, r.LeaseID)
	if e != nil {
		return e
	}
	if l.Applications[0].Build.Digest == "" {
		return errors.New("runtime started before APK built")
	}
	for _, other := range l.Runtimes {
		if other.Android != nil {
			f.ports[r.Project] = 30000 + other.Android.ConsolePort
		}
	}
	return f.lifecycleRuntime.Up(ctx, r)
}
func (f *mobileRuntime) Render(ctx context.Context, r domain.Runtime, roots []string) (Rendered, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lifecycleRuntime.Render(ctx, r, roots)
}
func (f *mobileRuntime) Inspect(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, e := f.lifecycleRuntime.Inspect(ctx, r)
	o.Endpoints = map[string]string{"api/8080/tcp": fmt.Sprintf("127.0.0.1:%d", f.ports[r.Project])}
	return o, e
}
func (f *mobileRuntime) Down(ctx context.Context, r domain.Runtime) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lifecycleRuntime.Down(ctx, r)
}
func mobileFixture(t *testing.T) (*Service, PlanOptions, *flutterFake, *applicationsFake, *androidLifecycleFake) {
	t.Helper()
	s, o, source, r, _ := lifecycleFixture(t)
	if e := os.WriteFile(filepath.Join(o.Repository, ".agent-env.yaml"), []byte(mobileManifest), 0600); e != nil {
		t.Fatal(e)
	}
	// Source operations have their own lock; avoid sharing the runtime log slice.
	source.operations = &[]string{}
	s.Source = &androidLifecycleSource{lifecycleSource: source}
	android := &androidLifecycleFake{store: s.Store, live: map[string]domain.Runtime{}}
	apps := &applicationsFake{store: s.Store, packages: map[string]bool{}, mappings: map[string]map[int]int{}}
	flutter := &flutterFake{}
	s.Android, s.AndroidApplications, s.Flutter = android, apps, flutter
	s.Runtime = &mobileRuntime{lifecycleRuntime: r, ports: map[string]int{}}
	s.ReadinessTimeout = 5 * time.Second
	o.Stack = "mobile"
	return s, o, flutter, apps, android
}
func TestMobilePlanStackClosure(t *testing.T) {
	s, o, f, _, a := mobileFixture(t)
	for name, want := range map[string]int{"api": 1, "dashboard": 2, "mobile": 2, "full": 3} {
		o.Stack = name
		p, e := BuildPlan(context.Background(), o, s.Source)
		if e != nil {
			t.Fatal(e)
		}
		if len(p.Components) != want {
			t.Fatalf("%s components=%+v", name, p.Components)
		}
		has := name == "mobile" || name == "full"
		if (len(p.Applications) > 0) != has {
			t.Fatalf("%s applications=%+v", name, p.Applications)
		}
	}
	if f.builds != 0 || a.creates != 0 {
		t.Fatal("plan performed effects")
	}
}
func TestMobileConcurrentLeasesAndObservation(t *testing.T) {
	s, o, _, apps, _ := mobileFixture(t)
	ctx := context.Background()
	type result struct {
		l domain.Lease
		e error
	}
	ch := make(chan result, 2)
	for range 2 {
		go func() { l, e := s.Create(ctx, o, CreateOptions{Owner: "test"}); ch <- result{l, e} }()
	}
	one, two := <-ch, <-ch
	if one.e != nil || two.e != nil {
		t.Fatalf("create: %v / %v", one.e, two.e)
	}
	a, b := one.l.Applications[0], two.l.Applications[0]
	if a.State != "ready" || b.State != "ready" || a.Build.Digest == b.Build.Digest || a.Build.ArtifactPath == b.Build.ArtifactPath || a.Reverse[0].HostPort == b.Reverse[0].HostPort || one.l.Runtimes[0].Project == two.l.Runtimes[0].Project {
		t.Fatalf("collision: %+v %+v", a, b)
	}
	r1, _ := applicationRuntime(one.l, "phone")
	r2, _ := applicationRuntime(two.l, "phone")
	if r1.Android.Serial == r2.Android.Serial || r1.Android.AVDPath == r2.Android.AVDPath {
		t.Fatal("Android collision")
	}
	// Save/get roundtrip retained all build identity through real SQLite.
	saved, e := s.Store.Get(ctx, one.l.ID)
	if e != nil || saved.Applications[0].InstalledDigest != a.Build.Digest {
		t.Fatalf("snapshot: %v %+v", e, saved)
	}
	apps.mu.Lock()
	apps.packages[r1.Android.Serial] = false
	apps.mu.Unlock()
	l, e := s.Reconcile(ctx, one.l.ID)
	if e != nil || l.Observed != "degraded" {
		t.Fatalf("uninstall: %v %+v", e, l)
	}
	apps.mu.Lock()
	apps.packages[r1.Android.Serial] = true
	delete(apps.mappings[r1.Android.Serial], 8080)
	apps.mu.Unlock()
	l, e = s.Reconcile(ctx, one.l.ID)
	if e != nil || l.Observed != "degraded" {
		t.Fatalf("reverse loss: %v %+v", e, l)
	}
	l, e = s.Destroy(ctx, one.l.ID, false, false)
	if e != nil || l.Observed != "released" {
		t.Fatalf("destroy: %v %+v", e, l)
	}
	l, e = s.Reconcile(ctx, two.l.ID)
	if e != nil || l.Observed != "ready" {
		t.Fatalf("sibling: %v %+v", e, l)
	}
	if _, e = s.Destroy(ctx, two.l.ID, false, false); e != nil {
		t.Fatal(e)
	}
}
func TestMobileFailureCompensation(t *testing.T) {
	for _, stage := range []string{"build", "install", "launch"} {
		t.Run(stage, func(t *testing.T) {
			s, o, f, apps, a := mobileFixture(t)
			f.fail = stage == "build"
			apps.fail = stage
			l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
			if e == nil || l.Observed != "released" {
				t.Fatalf("%s: %v %+v", stage, e, l)
			}
			if stage == "build" && a.creates != 0 {
				t.Fatal("build failure started emulator")
			}
			artifacts, e := s.Store.Artifacts(context.Background(), l.ID)
			if e != nil || len(artifacts) < 2 {
				t.Fatalf("lost evidence: %v %+v", e, artifacts)
			}
			for _, maps := range apps.mappings {
				if len(maps) != 0 {
					t.Fatal("reverse leaked")
				}
			}
		})
	}
}
func TestMobilePrerequisiteNoReservation(t *testing.T) {
	s, o, f, _, a := mobileFixture(t)
	f.prerequisite = errors.New("flutter missing")
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if !errors.Is(e, ErrPrerequisite) || l.ID != "" || a.creates != 0 {
		t.Fatalf("%v %+v", e, l)
	}
}
func TestMobileUnknownIdentityAndCleanupPreservation(t *testing.T) {
	s, o, _, apps, a := mobileFixture(t)
	ctx := context.Background()
	l, e := s.Create(ctx, o, CreateOptions{Owner: "test"})
	if e != nil {
		t.Fatal(e)
	}
	a.identityErr = true
	l, e = s.Reconcile(ctx, l.ID)
	if e != nil || l.Observed != "quarantined" {
		t.Fatalf("%v %+v", e, l)
	}
	if _, e = s.Destroy(ctx, l.ID, true, false); e == nil {
		t.Fatal("force bypassed identity")
	}
	if a.destroys != 0 {
		t.Fatal("destroyed ambiguous emulator")
	}
	a.identityErr = false
	apps.fail = "cleanup"
	l, e = s.Destroy(ctx, l.ID, false, false)
	if e == nil || l.Observed != "quarantined" || a.destroys != 0 {
		t.Fatalf("cleanup ambiguity: %v %+v", e, l)
	}
	apps.fail = ""
	if _, e = s.Destroy(ctx, l.ID, false, false); e != nil {
		t.Fatal(e)
	}
}
func TestAndroidSerialInterpolation(t *testing.T) {
	s, o, _, _, a := mobileFixture(t)
	ctx := context.Background()
	l, e := s.Create(ctx, o, CreateOptions{Owner: "test"})
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("ANDROID_SERIAL", "foreign-device")
	resolve := func(name string) (string, error) { return s.androidTestSerial(ctx, l, name) }
	got, e := expandTestValue("${lease_id}:${android:phone:serial}", l.ID, resolve)
	r, _ := applicationRuntime(l, "phone")
	if e != nil || got != l.ID+":"+r.Android.Serial {
		t.Fatalf("%q %v", got, e)
	}
	for _, bad := range []string{"${android:missing:serial}", "${android:backend:serial}", "${android:phone:bad}", "${android::serial}"} {
		if _, e = expandTestValue(bad, l.ID, resolve); e == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	a.identityErr = true
	if _, e = expandTestValue("${android:phone:serial}", l.ID, resolve); !errors.Is(e, domain.ErrResourceIdentity) {
		t.Fatalf("identity: %v", e)
	}
	a.identityErr = false
	if _, e = s.Destroy(ctx, l.ID, false, false); e != nil {
		t.Fatal(e)
	}
}
func TestReverseEndpointRejectsRemoteAndMalformed(t *testing.T) {
	for _, v := range []string{"192.0.2.1:80", "localhost:80", "127.0.0.1:0", "127.0.0.1:99999", "http://127.0.0.1:80", ""} {
		if _, e := loopbackPort(v); e == nil {
			t.Fatalf("accepted %s", v)
		}
	}
	if n, e := loopbackPort("127.0.0.1:8080"); e != nil || n != 8080 {
		t.Fatal(n, e)
	}
}
func TestMobileEvidenceDoesNotKeepRawBuildOutputInLease(t *testing.T) {
	s, o, _, _, _ := mobileFixture(t)
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(l.Applications[0].Build.Stdout, "build evidence") {
		t.Fatal("raw output persisted")
	}
	if _, e = s.Destroy(context.Background(), l.ID, false, false); e != nil {
		t.Fatal(e)
	}
}

func TestMobileUnconfirmedBuildBlocksLaterForcedCleanup(t *testing.T) {
	s, o, f, _, android := mobileFixture(t)
	f.buildErr = execx.ErrProcessTreeUnconfirmed
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if !errors.Is(e, execx.ErrProcessTreeUnconfirmed) || l.Observed != "quarantined" {
		t.Fatalf("%v %+v", e, l)
	}
	saved, e := s.Store.Get(context.Background(), l.ID)
	if e != nil || !saved.Applications[0].BuildUnconfirmed {
		t.Fatalf("%v %+v", e, saved)
	}
	// Use a fresh service value to prove the guard is durable across restart.
	fresh := *s
	for _, force := range []bool{false, true} {
		kept, e := fresh.Destroy(context.Background(), l.ID, force, false)
		if e == nil || kept.Observed != "quarantined" || android.destroys != 0 {
			t.Fatalf("force=%v %v %+v", force, e, kept)
		}
		if _, e = os.Stat(saved.Sources[0].WorktreePath); e != nil {
			t.Fatalf("lost source: %v", e)
		}
	}
}

func TestMobileUnconfirmedReversePreservesMapping(t *testing.T) {
	s, o, _, apps, a := mobileFixture(t)
	apps.fail = "reverse"
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if e == nil || l.Observed != "quarantined" || a.destroys != 0 {
		t.Fatalf("%v %+v", e, l)
	}
	r, _ := applicationRuntime(l, "phone")
	if apps.mappings[r.Android.Serial][8080] == 0 {
		t.Fatal("unconfirmed mapping deleted")
	}
	// A later forced cleanup must preserve the same ownership ambiguity.
	if _, e = s.Destroy(context.Background(), l.ID, true, false); e == nil {
		t.Fatal("force bypassed unconfirmed mapping")
	}
	delete(apps.mappings[r.Android.Serial], 8080)
	apps.fail = ""
	if _, e = s.Destroy(context.Background(), l.ID, false, false); e != nil {
		t.Fatal(e)
	}
}

// These tests retain real app orchestration and SQLite evidence while injecting
// only the external named command. The shared runner records argv per worktree.
type mobileNamedRunner struct {
	mu       sync.Mutex
	commands []execx.Command
	entered  chan struct{}
	resume   <-chan struct{}
	fail     bool
}

func (r *mobileNamedRunner) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	r.mu.Lock()
	r.commands = append(r.commands, c)
	r.mu.Unlock()
	if r.entered != nil {
		r.entered <- struct{}{}
	}
	if r.resume != nil {
		select {
		case <-r.resume:
		case <-ctx.Done():
			return execx.Result{}, ctx.Err()
		}
	}
	if _, err := fmt.Fprintln(c.Stdout, "mobile command output"); err != nil {
		return execx.Result{}, err
	}
	if _, err := fmt.Fprintln(c.Stderr, "mobile command diagnostics"); err != nil {
		return execx.Result{}, err
	}
	report := filepath.Join(c.Dir, "reports", "mobile-result.txt")
	if err := os.MkdirAll(filepath.Dir(report), 0700); err != nil {
		return execx.Result{}, err
	}
	if err := os.WriteFile(report, []byte("mobile report"), 0600); err != nil {
		return execx.Result{}, err
	}
	if r.fail {
		return execx.Result{ExitCode: 9}, errors.New("mobile test failed")
	}
	return execx.Result{}, nil
}

func mobileNamedManifest(t *testing.T, o PlanOptions) {
	t.Helper()
	text := mobileManifest + `tests:
  mobile-e2e:
    stack: mobile
    source: self
    working_directory: .
    command: [fixture, --serial, '${android:phone:serial}', --lease, '${lease_id}']
    artifacts: [reports/mobile-result.txt]
    timeout: 5s
`
	if err := os.WriteFile(filepath.Join(o.Repository, ".agent-env.yaml"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertMobileNamedEvidence(t *testing.T, s *Service, l domain.Lease, run domain.CommandRun, wantStatus string) {
	t.Helper()
	ctx := context.Background()
	runs, err := s.Store.Runs(ctx, l.ID)
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID || runs[0].Status != wantStatus || runs[0].FinishedAt.IsZero() {
		t.Fatalf("durable named run missing: %+v %v", runs, err)
	}
	if len(runs[0].Notes) != 1 || !strings.Contains(runs[0].Notes[0], "does not prove execution of that exact APK") {
		t.Fatalf("APK provenance warning not persisted: %+v", runs[0].Notes)
	}
	stdout, err := os.ReadFile(run.StdoutPath)
	if err != nil || !strings.Contains(string(stdout), runs[0].Notes[0]) || !strings.Contains(string(stdout), "mobile command output") {
		t.Fatalf("stdout warning/output missing: %q %v", stdout, err)
	}
	stderr, err := os.ReadFile(run.StderrPath)
	if err != nil || !strings.Contains(string(stderr), "mobile command diagnostics") {
		t.Fatalf("stderr evidence missing: %q %v", stderr, err)
	}
	descriptor, err := os.ReadFile(filepath.Join(filepath.Dir(run.StdoutPath), "run.json"))
	if err != nil || !strings.Contains(string(descriptor), "does not prove execution of that exact APK") {
		t.Fatalf("descriptor warning missing: %s %v", descriptor, err)
	}
	artifacts, err := s.Store.Artifacts(ctx, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	reportFound := false
	for _, artifact := range artifacts {
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == "mobile report" {
			digest, err := evidence.FileDigest(artifact.Path)
			if err != nil || digest != artifact.Digest {
				t.Fatalf("report digest lost: %v", err)
			}
			reportFound = true
		}
	}
	if !reportFound {
		t.Fatal("declared mobile report not retained")
	}
	observed, err := s.Reconcile(ctx, l.ID)
	if err != nil || observed.Observed != "ready" || observed.Applications[0].State != "ready" || observed.Applications[0].InstalledDigest != l.Applications[0].InstalledDigest {
		t.Fatalf("named test changed lifecycle build/ready state: %+v %v", observed, err)
	}
}

func TestMobileConcurrentNamedTestsUseOwnedSerialAndPersistWarning(t *testing.T) {
	s, o, _, _, _ := mobileFixture(t)
	mobileNamedManifest(t, o)
	t.Setenv("ANDROID_SERIAL", "unrelated-device")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	leases := make([]domain.Lease, 2)
	for i := range leases {
		var err error
		leases[i], err = s.Create(ctx, o, CreateOptions{Owner: "mobile-named"})
		if err != nil {
			t.Fatal(err)
		}
	}
	resume := make(chan struct{})
	runner := &mobileNamedRunner{entered: make(chan struct{}, 2), resume: resume}
	type namedResult struct {
		index  int
		run    domain.CommandRun
		err    error
		stdout string
	}
	results := make(chan namedResult, 2)
	for i, l := range leases {
		go func(i int, l domain.Lease) {
			service := *s
			service.Runner = runner
			var stdout strings.Builder
			service.Stdout = &stdout
			run, err := service.Test(ctx, l.ID, "mobile-e2e")
			results <- namedResult{i, run, err, stdout.String()}
		}(i, l)
	}
	entered := 0
	for entered < 2 {
		select {
		case <-runner.entered:
			entered++
		case <-ctx.Done():
			goto release
		}
	}
release:
	close(resume)
	got := make([]namedResult, 2)
	for range 2 {
		result := <-results
		got[result.index] = result
	}
	if entered != 2 {
		t.Fatalf("separate leases did not execute named tests concurrently: %d", entered)
	}
	for i, result := range got {
		if result.err != nil || result.run.Status != "passed" {
			t.Fatalf("run %d: %+v %v", i, result.run, result.err)
		}
		if len(result.run.Notes) != 1 || !strings.Contains(result.stdout, result.run.Notes[0]) {
			t.Fatalf("stream warning missing: %q", result.stdout)
		}
		assertMobileNamedEvidence(t, s, leases[i], result.run, "passed")
	}
	runner.mu.Lock()
	commands := append([]execx.Command(nil), runner.commands...)
	runner.mu.Unlock()
	if len(commands) != 2 {
		t.Fatalf("expected exactly two commands: %+v", commands)
	}
	serials := map[string]bool{}
	for _, l := range leases {
		device, err := applicationRuntime(l, "phone")
		if err != nil {
			t.Fatal(err)
		}
		expected := strings.Join([]string{"--serial", device.Android.Serial, "--lease", l.ID}, "|")
		found := false
		for _, c := range commands {
			if c.Dir == l.Sources[0].WorktreePath {
				if c.Name != "fixture" || strings.Join(c.Args, "|") != expected {
					t.Fatalf("wrong owned serial/source argv: %+v", c)
				}
				found = true
			}
		}
		if !found || serials[device.Android.Serial] {
			t.Fatal("missing command or colliding device serial")
		}
		serials[device.Android.Serial] = true
	}
	for _, l := range leases {
		if _, err := s.Destroy(context.Background(), l.ID, false, false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMobileFailedNamedTestRetainsEvidenceAndReadyLease(t *testing.T) {
	s, o, _, _, _ := mobileFixture(t)
	mobileNamedManifest(t, o)
	ctx := context.Background()
	l, err := s.Create(ctx, o, CreateOptions{Owner: "mobile-named-failure"})
	if err != nil {
		t.Fatal(err)
	}
	s.Runner = &mobileNamedRunner{fail: true}
	run, err := s.Test(ctx, l.ID, "mobile-e2e")
	if !errors.Is(err, ErrTestFailed) || run.Status != "failed" || run.ExitCode != 9 {
		t.Fatalf("wrong failed test result: %+v %v", run, err)
	}
	assertMobileNamedEvidence(t, s, l, run, "failed")
	if _, err := s.Destroy(ctx, l.ID, false, false); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{run.StdoutPath, run.StderrPath, filepath.Join(filepath.Dir(run.StdoutPath), "run.json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("cleanup removed failed test evidence: %v", err)
		}
	}
}
