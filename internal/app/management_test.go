package app

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func managementFixture() *domain.Management {
	return &domain.Management{ControllerID: "controller-01", HostID: "linux-01", HostInstanceID: "instance-01", AssignmentEpoch: 7}
}

type managedSourceProof struct {
	SourceProvider
	store      Store
	id         string
	management domain.Management
	called     bool
}

func (s *managedSourceProof) Materialize(ctx context.Context, source domain.Source) error {
	l, e := s.store.Get(ctx, s.id)
	if e != nil {
		return e
	}
	if l.Management == nil || *l.Management != s.management {
		return errors.New("management not durable before source materialization")
	}
	s.called = true
	return s.SourceProvider.Materialize(ctx, source)
}

func TestManagedCreatePersistsGlobalIdentityBeforeEffects(t *testing.T) {
	s, o, _, runtime, _ := lifecycleFixture(t)
	m := managementFixture()
	s.Management = copyManagement(m)
	id := newID()
	proof := &managedSourceProof{SourceProvider: s.Source, store: s.Store, id: id, management: *m}
	s.Source = proof
	l, e := s.Create(context.Background(), o, CreateOptions{Owner: "remote", LeaseID: id, Management: m})
	if e != nil {
		t.Fatal(e)
	}
	if !proof.called || l.ID != id || l.Management == nil || *l.Management != *m || l.Observed != "ready" {
		t.Fatalf("global identity missing: %+v", l)
	}
	for _, source := range l.Sources {
		if !strings.Contains(source.WorktreePath, id) {
			t.Fatal("worktree uses a different lease ID")
		}
	}
	m.AssignmentEpoch++
	if l.Management.AssignmentEpoch != 7 {
		t.Fatal("caller mutated lease assignment through pointer")
	}
	stored, e := s.Store.Get(context.Background(), id)
	if e != nil || stored.Management == nil || *stored.Management != *s.Management {
		t.Fatalf("assignment persistence: %+v %v", stored, e)
	}
	before := runtime.upCount
	if _, e = s.Create(context.Background(), o, CreateOptions{Owner: "remote", LeaseID: id, Management: s.Management}); e == nil || runtime.upCount != before {
		t.Fatalf("duplicate ID started effects: %v", e)
	}
	if _, e = s.Renew(context.Background(), id, time.Hour); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Reconcile(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Destroy(context.Background(), id, false, false); e != nil {
		t.Fatal(e)
	}
}

func TestManagedCreateRejectsInvalidAuthorityBeforeSourceEffects(t *testing.T) {
	for _, name := range []string{"local-forged", "local-id", "missing-id", "traversal-id", "lowercase-id", "invalid-controller", "empty-host", "invalid-instance", "zero-epoch", "mismatch", "missing-metadata"} {
		t.Run(name, func(t *testing.T) {
			s, o, _, _, effects := lifecycleFixture(t)
			m := managementFixture()
			s.Management = copyManagement(m)
			opt := CreateOptions{Owner: "remote", LeaseID: newID(), Management: m}
			switch name {
			case "local-forged":
				s.Management = nil
			case "local-id":
				s.Management = nil
				opt.Management = nil
			case "missing-id":
				opt.LeaseID = ""
			case "traversal-id":
				opt.LeaseID = "../escape"
			case "lowercase-id":
				opt.LeaseID = strings.ToLower("01ARZ3NDEKTSV4RRFFQ69G5FAV")
			case "invalid-controller":
				m.ControllerID = "../controller"
				s.Management = copyManagement(m)
			case "empty-host":
				m.HostID = ""
				s.Management = copyManagement(m)
			case "invalid-instance":
				m.HostInstanceID = " instance "
				s.Management = copyManagement(m)
			case "zero-epoch":
				m.AssignmentEpoch = 0
				s.Management = copyManagement(m)
			case "mismatch":
				m.AssignmentEpoch++
			case "missing-metadata":
				opt.Management = nil
			}
			if _, e := s.Create(context.Background(), o, opt); e == nil {
				t.Fatal("invalid create accepted")
			}
			leases, e := s.Store.List(context.Background())
			if e != nil || len(leases) != 0 || len(*effects) != 0 {
				t.Fatalf("rejected request left effects: %+v %v %v", leases, *effects, e)
			}
		})
	}
}

func TestManagedLeaseLocalMutationsRefusedAndReadsPreserved(t *testing.T) {
	s, o, _, _, effects := lifecycleFixture(t)
	ctx := context.Background()
	s.Management = managementFixture()
	l, e := s.Create(ctx, o, CreateOptions{Owner: "remote", LeaseID: newID(), Management: s.Management})
	if e != nil {
		t.Fatal(e)
	}
	s.Management = nil
	before, _ := json.Marshal(l)
	effectCount := len(*effects)
	calls := map[string]func() error{
		"renew":     func() error { _, e := s.Renew(ctx, l.ID, time.Hour); return e },
		"reconcile": func() error { _, e := s.Reconcile(ctx, l.ID); return e },
		"destroy":   func() error { _, e := s.Destroy(ctx, l.ID, false, false); return e },
		"force":     func() error { _, e := s.Destroy(ctx, l.ID, true, false); return e },
		"preview":   func() error { _, e := s.Destroy(ctx, l.ID, true, true); return e },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if e := call(); !errors.Is(e, ErrManagementAuthority) {
				t.Fatalf("authority not refused: %v", e)
			}
		})
	}
	got, e := s.Show(ctx, l.ID)
	if e != nil || !reflect.DeepEqual(got.Management, l.Management) {
		t.Fatalf("managed show: %+v %v", got, e)
	}
	listed, e := s.List(ctx, false)
	if e != nil || len(listed) != 1 || !reflect.DeepEqual(listed[0].Management, l.Management) {
		t.Fatalf("managed list: %+v %v", listed, e)
	}
	stored, e := s.Get(ctx, l.ID)
	after, _ := json.Marshal(stored)
	if e != nil || string(before) != string(after) || len(*effects) != effectCount {
		t.Fatalf("local refusal/read modified lease: %v", e)
	}
	for _, field := range []string{"controller", "host", "instance", "epoch"} {
		wrong := *l.Management
		switch field {
		case "controller":
			wrong.ControllerID += "-other"
		case "host":
			wrong.HostID += "-other"
		case "instance":
			wrong.HostInstanceID += "-other"
		case "epoch":
			wrong.AssignmentEpoch++
		}
		s.Management = &wrong
		if _, e := s.Renew(ctx, l.ID, time.Hour); !errors.Is(e, ErrManagementAuthority) {
			t.Fatalf("%s mismatch accepted: %v", field, e)
		}
	}
}

func TestManagedGCNeverUsesLocalExpiryAsAuthority(t *testing.T) {
	s, o, source, runtime, _ := lifecycleFixture(t)
	ctx := context.Background()
	s.Management = managementFixture()
	l, e := s.Create(ctx, o, CreateOptions{Owner: "remote", LeaseID: newID(), Management: s.Management})
	if e != nil {
		t.Fatal(e)
	}
	l.ExpiresAt = time.Now().Add(-time.Hour)
	l.HeartbeatAt = l.ExpiresAt
	if e = s.Store.Save(ctx, l); e != nil {
		t.Fatal(e)
	}
	for _, authority := range []*domain.Management{nil, {ControllerID: "other", HostID: "linux-01", HostInstanceID: "instance-01", AssignmentEpoch: 7}} {
		s.Management = authority
		for _, apply := range []bool{false, true} {
			got, e := s.GC(ctx, apply)
			if e != nil || len(got) != 0 {
				t.Fatalf("unauthorized GC: %+v %v", got, e)
			}
		}
	}
	if len(source.states) != 1 || len(runtime.projects) != 2 {
		t.Fatal("unauthorized GC removed resources")
	}
	s.Management = copyManagement(l.Management)
	got, e := s.GC(ctx, true)
	if e != nil || len(got) != 1 || got[0].Observed != "released" {
		t.Fatalf("authorized GC failed: %+v %v", got, e)
	}
}

func TestManagedCommandCancellationAndUIBrowserGuarded(t *testing.T) {
	ctx := context.Background()
	t.Run("test-and-cancellation", func(t *testing.T) {
		s, db, l := commandFixture(t, managementFixture())
		if _, e := s.Test(ctx, l.ID, "check"); !errors.Is(e, ErrManagementAuthority) {
			t.Fatalf("test accepted: %v", e)
		}
		if e := db.SaveRun(ctx, domain.CommandRun{ID: "active-run", LeaseID: l.ID, Status: "running", StartedAt: time.Now()}); e != nil {
			t.Fatal(e)
		}
		_, release, e := db.AcquireContext(ctx, l.ID, newID(), time.Minute)
		if e != nil {
			t.Fatal(e)
		}
		defer release()
		if _, e = s.Destroy(ctx, l.ID, true, false); !errors.Is(e, ErrManagementAuthority) {
			t.Fatalf("destroy requested cancellation: %v", e)
		}
		cancel, e := db.RunCancellationRequested(ctx, "active-run")
		if e != nil || cancel {
			t.Fatalf("local caller canceled remote run: %v %v", cancel, e)
		}
	})
	t.Run("ui", func(t *testing.T) {
		s, db, l, p := uiFixture(t, managementFixture())
		for _, operation := range []string{"snapshot", "home"} {
			if _, e := s.UI(ctx, l.ID, UIOptions{Operation: operation}); !errors.Is(e, ErrManagementAuthority) {
				t.Fatalf("%s accepted: %v", operation, e)
			}
		}
		if _, e := s.RecoverUI(ctx, l.ID, "running"); !errors.Is(e, ErrManagementAuthority) {
			t.Fatalf("recovery accepted: %v", e)
		}
		runs, e := db.Runs(ctx, l.ID)
		if e != nil || len(runs) != 0 || p.calls != 0 {
			t.Fatalf("UI effect before refusal: %d %+v %v", p.calls, runs, e)
		}
		s.Management = copyManagement(l.Management)
		if _, e = s.UI(ctx, l.ID, UIOptions{Operation: "snapshot"}); e != nil || p.calls != 1 {
			t.Fatalf("authorized UI failed: %v", e)
		}
	})
	t.Run("browser", func(t *testing.T) {
		s, l, p, _ := browserFixture(t, managementFixture())
		for _, operation := range []string{"snapshot", "page-create"} {
			if _, e := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: operation}}); !errors.Is(e, ErrManagementAuthority) {
				t.Fatalf("%s accepted: %v", operation, e)
			}
		}
		runs, e := s.Store.Runs(ctx, l.ID)
		if e != nil || len(runs) != 0 || p.calls != 0 {
			t.Fatalf("browser effect before refusal: %d %+v %v", p.calls, runs, e)
		}
		s.Management = copyManagement(l.Management)
		if _, e = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}}); e != nil || p.calls != 1 {
			t.Fatalf("authorized browser failed: %v", e)
		}
	})
}

type managementChangesAtFence struct {
	Store
	acquired bool
}

func (s *managementChangesAtFence) Get(ctx context.Context, id string) (domain.Lease, error) {
	l, e := s.Store.Get(ctx, id)
	if s.acquired {
		l.Management = managementFixture()
	}
	return l, e
}
func (s *managementChangesAtFence) AcquireContext(ctx context.Context, id, owner string, ttl time.Duration) (context.Context, func() error, error) {
	op, release, e := s.Store.AcquireContext(ctx, id, owner, ttl)
	if e == nil {
		s.acquired = true
	}
	return op, release, e
}

func TestManagedAuthorityRecheckedUnderFence(t *testing.T) {
	for _, operation := range []string{"renew", "destroy", "reconcile", "gc"} {
		t.Run(operation, func(t *testing.T) {
			s, o, _, _, effects := lifecycleFixture(t)
			ctx := context.Background()
			l, e := s.Create(ctx, o, CreateOptions{Owner: "local"})
			if e != nil {
				t.Fatal(e)
			}
			l.ExpiresAt = time.Now().Add(-time.Hour)
			l.HeartbeatAt = l.ExpiresAt
			if e = s.Store.Save(ctx, l); e != nil {
				t.Fatal(e)
			}
			before := len(*effects)
			s.Store = &managementChangesAtFence{Store: s.Store}
			switch operation {
			case "renew":
				_, e = s.Renew(ctx, l.ID, time.Hour)
			case "destroy":
				_, e = s.Destroy(ctx, l.ID, true, false)
			case "reconcile":
				_, e = s.Reconcile(ctx, l.ID)
			case "gc":
				_, e = s.GC(ctx, true)
			}
			if !errors.Is(e, ErrManagementAuthority) || len(*effects) != before {
				t.Fatalf("assignment changed before fence but mutation continued: %v", e)
			}
			stored, e := s.Store.Get(ctx, l.ID)
			if e != nil || stored.Observed == "released" || !stored.ExpiresAt.Equal(l.ExpiresAt) {
				t.Fatalf("unauthorized state write: %+v %v", stored, e)
			}
		})
	}
}
