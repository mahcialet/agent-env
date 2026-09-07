package app

import (
	"context"
	"errors"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestMobileReconcileRejectsIncompleteApplicationSnapshot(t *testing.T) {
	for _, scenario := range []string{"missing", "duplicate", "digest", "source", "reverse", "argv"} {
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
