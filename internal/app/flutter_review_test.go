package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestSelectedApplicationAPKOutputsCannotCollide(t *testing.T) {
	for _, tc := range []struct {
		name, project, artifact string
		collision               bool
	}{
		{"same", ".", "build/app.apk", true},
		{"normalized", "./", "./build/app.apk", true},
		{"project-relative", "build", "app.apk", true},
		{"case-insensitive-host", ".", "BUILD/App.apk", true},
		{"distinct", ".", "build/second.apk", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, o, flutter, _, android := mobileFixture(t)
			manifest := fmt.Sprintf(`version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  first: {type: android-emulator, source: self, avd: pixel}
  second: {type: android-emulator, source: self, avd: pixel}
applications:
  one: {type: flutter-android, source: self, runtime: first, project_directory: ., build: {command: [flutter, build, apk, --debug], artifact: build/app.apk}, package: com.example.one, activity: .MainActivity}
  two: {type: flutter-android, source: self, runtime: second, project_directory: %s, build: {command: [flutter, build, apk, --release], artifact: %s}, package: com.example.two, activity: .MainActivity}
components:
  one: {runtime: first, application: one}
  two: {runtime: second, application: two}
stacks:
  both: {roots: [one, two]}
  first: {roots: [one]}
  second: {roots: [two]}
`, tc.project, tc.artifact)
			if err := os.WriteFile(filepath.Join(o.Repository, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
				t.Fatal(err)
			}
			for _, stack := range []string{"first", "second"} {
				o.Stack = stack
				if _, err := BuildPlan(context.Background(), o, s.Source); err != nil {
					t.Fatalf("unselected output affected %s: %v", stack, err)
				}
			}
			o.Stack = "both"
			_, err := BuildPlan(context.Background(), o, s.Source)
			if !tc.collision {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "same source-relative APK output") {
				t.Fatalf("collision not rejected: %v", err)
			}
			_, err = s.Create(context.Background(), o, CreateOptions{Owner: "test"})
			if err == nil || !strings.Contains(err.Error(), "same source-relative APK output") {
				t.Fatalf("create collision: %v", err)
			}
			leases, err := s.Store.List(context.Background())
			if err != nil || len(leases) != 0 || flutter.builds != 0 || android.creates != 0 {
				t.Fatalf("collision performed effects: leases=%d build=%d Android=%d err=%v", len(leases), flutter.builds, android.creates, err)
			}
		})
	}
}

type noReverseReviewProvider struct {
	AndroidApplicationProvider
	queries int
}

func (p *noReverseReviewProvider) ReverseMappings(context.Context, domain.Runtime) (map[int]int, error) {
	p.queries++
	return nil, errors.New("reverse unsupported on this device")
}
func TestApplicationWithoutReverseSkipsNetworkObservation(t *testing.T) {
	s, o, _, _, _ := mobileFixture(t)
	l, err := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var manifest config.Manifest
	if err := json.Unmarshal(l.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	spec := manifest.Applications["client"]
	spec.Reverse = nil
	manifest.Applications["client"] = spec
	l.Manifest, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	l.ManifestDigest = config.Digest(&manifest)
	a := l.Applications[0]
	a.Reverse = nil
	noReverse := &noReverseReviewProvider{AndroidApplicationProvider: s.AndroidApplications}
	s.AndroidApplications = noReverse
	// A network-free application observation must not consult the Compose provider.
	s.Runtime = nil
	if err := s.observeApplication(context.Background(), l, &a); err != nil || a.State != "ready" || noReverse.queries != 0 {
		t.Fatalf("network-free observation: state=%s queries=%d err=%v", a.State, noReverse.queries, err)
	}
}

func TestDestroyPreviewHonorsApplicationBuildBarriers(t *testing.T) {
	for _, guard := range []string{"evidence", "process"} {
		t.Run(guard, func(t *testing.T) {
			s, o, _, _, android := mobileFixture(t)
			l, err := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
			if err != nil {
				t.Fatal(err)
			}
			l.Applications[0].BuildEvidenceIncomplete = guard == "evidence"
			l.Applications[0].BuildUnconfirmed = guard == "process"
			if err := s.Store.Save(context.Background(), l); err != nil {
				t.Fatal(err)
			}
			before, err := s.Store.Get(context.Background(), l.ID)
			if err != nil {
				t.Fatal(err)
			}
			events, err := s.Store.Events(context.Background(), l.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, force := range []bool{false, true} {
				preview, err := s.Destroy(context.Background(), l.ID, force, true)
				text := strings.Join(preview.Diagnostics, "\n")
				if err != nil || !strings.Contains(text, "would quarantine application client") || strings.Contains(text, "would remove") || strings.Contains(text, "would verify ownership, stop") {
					t.Fatalf("force=%t unsafe preview: %s err=%v", force, text, err)
				}
			}
			after, err := s.Store.Get(context.Background(), l.ID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("preview changed lease: %v", err)
			}
			laterEvents, err := s.Store.Events(context.Background(), l.ID)
			if err != nil || !reflect.DeepEqual(events, laterEvents) || android.destroys != 0 {
				t.Fatalf("preview changed evidence/resources: %v", err)
			}
		})
	}
}
