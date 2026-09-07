package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

func TestMobileReconcileRejectsIncompleteApplicationSnapshot(t *testing.T) {
	for _, scenario := range []string{"missing", "duplicate", "digest", "source", "reverse", "argv", "launch", "executable", "directory"} {
		t.Run(scenario, func(t *testing.T) {
			s, o, _, _, _ := mobileFixture(t)
			ctx := context.Background()
			l, e := s.Create(ctx, o, CreateOptions{Owner: "test"})
			if e != nil {
				t.Fatal(e)
			}
			original, e := s.Store.Get(ctx, l.ID)
			if e != nil {
				t.Fatal(e)
			}
			switch scenario {
			case "missing":
				l.Applications = nil
			case "duplicate":
				l.Applications = append(l.Applications, l.Applications[0])
			case "digest":
				l.Applications[0].InstalledDigest = "wrong"
			case "source":
				l.Applications[0].SourceCommit = "wrong"
			case "reverse":
				l.Applications[0].Reverse[0].Endpoint = "other.http"
			case "launch":
				l.Applications[0].LaunchConfirmed = false
				l.Applications[0].State = "launching"
			case "executable":
				l.Applications[0].Build.Executable = "other"
			case "directory":
				l.Applications[0].Build.Directory = "other"
			case "argv":
				l.Applications[0].Command = []string{"other", "build"}
			}
			if e = s.Store.Save(ctx, l); e != nil {
				t.Fatal(e)
			}
			observed, e := s.Reconcile(ctx, l.ID)
			if e != nil || observed.Observed != "degraded" {
				t.Fatalf("%s: %v %+v", scenario, e, observed)
			}
			if e = s.Store.Save(ctx, original); e != nil {
				t.Fatal(e)
			}
			if _, e = s.Destroy(ctx, l.ID, false, false); e != nil {
				t.Fatal(e)
			}
		})
	}
}

type existingPackageProvider struct{ *applicationsFake }

func (existingPackageProvider) PackageInstalled(context.Context, domain.Runtime, string) (bool, error) {
	return true, nil
}
func TestMobilePreexistingPackageCannotClaimNewAPK(t *testing.T) {
	s, o, _, apps, _ := mobileFixture(t)
	s.AndroidApplications = existingPackageProvider{apps}
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if e == nil || l.Observed != "released" || l.Applications[0].InstalledDigest != "" {
		t.Fatalf("%v %+v", e, l)
	}
	if len(apps.packages) != 0 {
		t.Fatal("installed over preexisting package")
	}
}

// Ordinary ownership ambiguity preserves the uncertain device but still cleans
// independently-owned Compose resources; DB errors instead fence all effects.
func TestMobileMappingAmbiguityCleansIndependentCompose(t *testing.T) {
	s, o, _, apps, android := mobileFixture(t)
	apps.fail = "reverse"
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if !errors.Is(e, domain.ErrResourceIdentity) || l.Observed != "quarantined" {
		t.Fatalf("%v %+v", e, l)
	}
	if android.destroys != 0 {
		t.Fatal("destroyed ambiguous device")
	}
	for _, r := range l.Runtimes {
		if r.Type == "compose" {
			obs, e := s.Runtime.Inspect(context.Background(), r)
			if e != nil || obs.Exists {
				t.Fatalf("independent Compose retained: %v %+v", e, obs)
			}
		}
	}
}

func TestMobileUnconfirmedBuildRemainsQuarantinedOnObservation(t *testing.T) {
	s, o, f, _, _ := mobileFixture(t)
	f.buildErr = execx.ErrProcessTreeUnconfirmed
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if !errors.Is(e, execx.ErrProcessTreeUnconfirmed) {
		t.Fatal(e)
	}
	observed, e := s.Reconcile(context.Background(), l.ID)
	if e != nil || observed.Observed != "quarantined" || !observed.Applications[0].BuildUnconfirmed {
		t.Fatalf("%v %+v", e, observed)
	}
}

type failedBuildArtifactStore struct{ Store }

func (s failedBuildArtifactStore) SaveArtifact(ctx context.Context, a domain.Artifact) error {
	if strings.HasPrefix(a.Kind, "application-build/") {
		return errors.New("injected build artifact persistence failure")
	}
	return s.Store.SaveArtifact(ctx, a)
}
func TestMobileIncompleteBuildEvidenceBlocksCleanup(t *testing.T) {
	s, o, _, _, android := mobileFixture(t)
	base := s.Store
	s.Store = failedBuildArtifactStore{base}
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "test"})
	if e == nil || l.Observed != "quarantined" || android.creates != 0 {
		t.Fatalf("%v %+v", e, l)
	}
	saved, e := base.Get(context.Background(), l.ID)
	if e != nil || !saved.Applications[0].BuildEvidenceIncomplete || saved.Applications[0].BuildUnconfirmed {
		t.Fatalf("%v %+v", e, saved)
	}
	// Storage recovers, but no command or observation may silently claim that
	// missing evidence was finalized or remove its remaining source/APK.
	s.Store = base
	observed, e := s.Reconcile(context.Background(), l.ID)
	if e != nil || observed.Observed != "quarantined" {
		t.Fatalf("%v %+v", e, observed)
	}
	for _, force := range []bool{false, true} {
		if _, e = s.Destroy(context.Background(), l.ID, force, false); e == nil {
			t.Fatalf("force=%v removed incomplete evidence", force)
		}
		if _, e = os.Stat(saved.Applications[0].Build.ArtifactPath); e != nil {
			t.Fatalf("lost APK: %v", e)
		}
	}
}
