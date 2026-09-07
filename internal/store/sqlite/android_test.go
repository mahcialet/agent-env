package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/migrations"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func androidLease(t *testing.T, id string) domain.Lease {
	t.Helper()
	l := lease(id)
	home := filepath.Join(t.TempDir(), id)
	l.Runtimes = append(l.Runtimes, domain.Runtime{Name: "device", Type: "android-emulator", Android: &domain.AndroidEmulator{Template: "pixel_api35", AVDName: "avd-" + id, AVDHome: home, AVDPath: filepath.Join(home, "avd-"+id+".avd")}})
	return l
}
func device(l domain.Lease) *domain.AndroidEmulator { return l.Runtimes[len(l.Runtimes)-1].Android }
func TestAndroidConcurrentIndependentReservations(t *testing.T) {
	ctx := context.Background()
	s, path := database(t)
	const n = 10
	stores := make([]*Store, n)
	leases := make([]domain.Lease, n)
	for i := range n {
		var err error
		stores[i], err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer stores[i].Close()
		leases[i] = androidLease(t, fmt.Sprint(i))
	}
	start := make(chan struct{})
	results := make(chan error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- stores[i].Reserve(ctx, leases[i], 0) }()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	ports := map[int]bool{}
	names := map[string]bool{}
	paths := map[string]bool{}
	for _, l := range leases {
		got, err := s.Get(ctx, l.ID)
		if err != nil {
			t.Fatal(err)
		}
		a := device(got)
		if ports[a.ConsolePort] || names[a.AVDName] || paths[a.AVDPath] || a.ADBPort != a.ConsolePort+1 || a.Serial != fmt.Sprintf("emulator-%d", a.ConsolePort) || a.State != "reserved" {
			t.Fatalf("collision or bad reservation: %+v", a)
		}
		ports[a.ConsolePort] = true
		names[a.AVDName] = true
		paths[a.AVDPath] = true
		if device(l).ConsolePort != 0 {
			t.Fatal("Reserve mutated input")
		}
	}
	for p := 5554; p < 5554+2*n; p += 2 {
		if !ports[p] {
			t.Fatalf("first free port skipped: %d", p)
		}
	}
}
func TestAndroidRetentionReuseAndIdentity(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	l := androidLease(t, "one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	l, _ = s.Get(ctx, l.ID)
	l.Desired = "released"
	l.Observed = "quarantined"
	if err := s.Save(ctx, l); err != nil {
		t.Fatal(err)
	}
	other := androidLease(t, "two")
	if err := s.Reserve(ctx, other, 0); err != nil {
		t.Fatal(err)
	}
	other, _ = s.Get(ctx, other.ID)
	if device(other).ConsolePort != 5556 {
		t.Fatal("quarantine released port")
	}
	changed := l
	changed.Runtimes = append([]domain.Runtime(nil), l.Runtimes...)
	a := *device(l)
	a.ConsolePort = 5560
	a.ADBPort = 5561
	a.Serial = "emulator-5560"
	changed.Runtimes[1].Android = &a
	if err := s.Save(ctx, changed); !errors.Is(err, domain.ErrResourceIdentity) {
		t.Fatalf("identity mutation accepted: %v", err)
	}
	changed = l
	changed.Runtimes = changed.Runtimes[:1]
	if err := s.Save(ctx, changed); !errors.Is(err, domain.ErrResourceIdentity) {
		t.Fatalf("removal accepted: %v", err)
	}
	l.Observed = "released"
	if err := s.Save(ctx, l); err != nil {
		t.Fatal(err)
	}
	reuse := androidLease(t, "three")
	device(reuse).AVDName = device(l).AVDName
	device(reuse).AVDHome = device(l).AVDHome
	device(reuse).AVDPath = device(l).AVDPath
	if err := s.Reserve(ctx, reuse, 0); err != nil {
		t.Fatal(err)
	}
	reuse, _ = s.Get(ctx, reuse.ID)
	if device(reuse).ConsolePort != 5554 {
		t.Fatal("released slot not reused")
	}
	l.Observed = "quarantined"
	if err := s.Save(ctx, l); err == nil {
		t.Fatal("reactivated stolen slot")
	}
}
func TestAndroidFailedReserveRollsBack(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	bad := androidLease(t, "bad")
	bad.Sources = append(bad.Sources, bad.Sources[0])
	if err := s.Reserve(ctx, bad, 0); err == nil {
		t.Fatal("bad child accepted")
	}
	var n int
	if err := s.db.QueryRow("SELECT count(*) FROM android_reservations").Scan(&n); err != nil || n != 0 {
		t.Fatalf("leaked reservation: %d %v", n, err)
	}
	if _, err := s.Get(ctx, bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("partial lease: %v", err)
	}
	good := androidLease(t, "good")
	if err := s.Reserve(ctx, good, 0); err != nil {
		t.Fatal(err)
	}
	good, _ = s.Get(ctx, good.ID)
	if device(good).ConsolePort != 5554 {
		t.Fatal("failed reserve consumed slot")
	}
	duplicate := androidLease(t, "duplicate")
	device(duplicate).AVDName = device(good).AVDName
	device(duplicate).AVDPath = filepath.Join(device(duplicate).AVDHome, device(duplicate).AVDName+".avd")
	if err := s.Reserve(ctx, duplicate, 0); err == nil {
		t.Fatal("duplicate AVD name accepted")
	}
	malformed := androidLease(t, "malformed")
	device(malformed).ADBPort = 9999
	if err := s.Reserve(ctx, malformed, 0); err == nil {
		t.Fatal("caller supplied ports accepted")
	}
}
func TestAndroidStolenOwnerCannotRelease(t *testing.T) {
	ctx := context.Background()
	s, path := database(t)
	l := androidLease(t, "one")
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	l, _ = s.Get(ctx, l.ID)
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	op, release, err := s.AcquireContext(ctx, l.ID, "first", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err = other.db.Exec("UPDATE operation_locks SET token='successor' WHERE lease_id=?", l.ID); err != nil {
		t.Fatal(err)
	}
	l.Observed = "released"
	if err = s.Save(context.WithoutCancel(op), l); !errors.Is(err, domain.ErrLockLost) {
		t.Fatalf("stolen save: %v", err)
	}
	var active int
	if err = s.db.QueryRow("SELECT active FROM android_reservations WHERE lease_id=?", l.ID).Scan(&active); err != nil || active != 1 {
		t.Fatalf("stale owner released slot: %d %v", active, err)
	}
}

func TestAndroidMigrationPreservesCompose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.sqlite")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_initial.sql", "002_command_cancellation.sql"} {
		script, err := migrations.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = raw.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
		if _, err = raw.Exec("INSERT INTO schema_migrations VALUES(?,?)", name, "before"); err != nil {
			t.Fatal(err)
		}
	}
	l := lease("compose-existing")
	if _, err = raw.Exec(`INSERT INTO leases VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, l.ID, l.Owner, l.Purpose, l.Mode, l.Repository, l.Stack, l.Desired, l.Observed, stamp(l.CreatedAt), stamp(l.HeartbeatAt), stamp(l.ExpiresAt), l.ManifestDigest, l.SourceSetDigest, encoded(l)); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO lease_runtimes VALUES(?,?,?,?,?,?,?)`, l.ID, "compose", "compose", "original-project", "", 1, encoded(l.Runtimes[0])); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Get(context.Background(), l.ID)
	if err != nil || encoded(got) != encoded(l) {
		t.Fatalf("Compose payload changed: %+v %v", got, err)
	}
	var project string
	if err = s.db.QueryRow("SELECT project FROM lease_runtimes WHERE lease_id=?", l.ID).Scan(&project); err != nil || project != "original-project" {
		t.Fatalf("Compose reservation changed: %s %v", project, err)
	}
}
func TestAndroidSlotExhaustion(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	for i := 0; i < 65; i++ {
		if err := s.Reserve(ctx, androidLease(t, fmt.Sprint(i)), 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Reserve(ctx, androidLease(t, "overflow"), 0); err == nil {
		t.Fatal("exhausted pool allocated")
	}
	if _, err := s.Get(ctx, "overflow"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("exhaustion left lease: %v", err)
	}
}

func TestAndroidOverlappingWritableHomes(t *testing.T) {
	ctx := context.Background()
	s, _ := database(t)
	l := androidLease(t, "one")
	device(l).Template = "Pixel.API35"
	if err := s.Reserve(ctx, l, 0); err != nil {
		t.Fatal(err)
	}
	for _, home := range []string{device(l).AVDHome, filepath.Join(device(l).AVDHome, "nested"), filepath.Dir(device(l).AVDHome)} {
		next := androidLease(t, "overlap")
		device(next).AVDHome = home
		device(next).AVDPath = filepath.Join(home, device(next).AVDName+".avd")
		if err := s.Reserve(ctx, next, 0); err == nil {
			t.Fatalf("overlapping home accepted: %s", home)
		}
	}
	if androidPathsOverlap(filepath.Join("root", "one"), filepath.Join("root", "one-more")) {
		t.Fatal("sibling path prefix treated as ancestor")
	}
}
