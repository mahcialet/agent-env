package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func processLease(id string, ports ...string) domain.Lease {
	allocated := map[string]int{}
	for _, name := range ports {
		allocated[name] = 0
	}
	return domain.Lease{ID: id, Desired: "ready", Observed: "reserved", ExpiresAt: time.Now().Add(time.Hour), Runtimes: []domain.Runtime{{Name: "server", Type: "process", Process: &domain.PersistentProcess{WorkingDirectory: ".", Command: []string{"server", "--port", "${port:http}"}, Env: map[string]string{"MODE": "test"}, Ports: allocated, SourceCommit: "immutable-source-commit", State: "planned"}}}}
}
func reservedProcess(t *testing.T, ports ...string) (*Store, domain.Lease) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	l := processLease("process-one", ports...)
	if err = s.Reserve(context.Background(), l, 10); err != nil {
		t.Fatal(err)
	}
	l, err = s.Get(context.Background(), l.ID)
	if err != nil {
		t.Fatal(err)
	}
	return s, l
}
func processReservationCount(t *testing.T, s *Store, id string) int {
	t.Helper()
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM process_port_reservations WHERE lease_id=?`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
func TestProcessConcurrentReservationPortsAreDisjoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	const count = 12
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := first
			if i%2 != 0 {
				s = second
			}
			l := processLease(fmt.Sprintf("process-%02d", i), "http", "admin")
			err := s.Reserve(context.Background(), l, count+1)
			if l.Runtimes[0].Process.Ports["http"] != 0 {
				err = fmt.Errorf("caller snapshot mutated")
			}
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	used := map[int]bool{}
	for i := 0; i < count; i++ {
		l, err := first.Get(context.Background(), fmt.Sprintf("process-%02d", i))
		if err != nil {
			t.Fatal(err)
		}
		for _, port := range l.Runtimes[0].Process.Ports {
			if port < processPortMin || port > processPortMax || used[port] {
				t.Fatalf("duplicate/invalid port %d", port)
			}
			used[port] = true
		}
	}
	if len(used) != count*2 {
		t.Fatal("missing allocations")
	}
}
func TestProcessSaveRejectsImmutableSnapshotChanges(t *testing.T) {
	cases := map[string]func(*domain.Lease){
		"remove-runtime":  func(l *domain.Lease) { l.Runtimes = nil },
		"remove-snapshot": func(l *domain.Lease) { l.Runtimes[0].Process = nil },
		"rename-runtime":  func(l *domain.Lease) { l.Runtimes[0].Name = "other" },
		"remove-port":     func(l *domain.Lease) { delete(l.Runtimes[0].Process.Ports, "http") },
		"rename-port": func(l *domain.Lease) {
			p := l.Runtimes[0].Process
			p.Ports["renamed"] = p.Ports["http"]
			delete(p.Ports, "http")
		},
		"reassign-port": func(l *domain.Lease) { l.Runtimes[0].Process.Ports["http"]++ },
		"new-port":      func(l *domain.Lease) { l.Runtimes[0].Process.Ports["new"] = 0 },
		"command":       func(l *domain.Lease) { l.Runtimes[0].Process.Command[0] = "other" },
		"env":           func(l *domain.Lease) { l.Runtimes[0].Process.Env["MODE"] = "other" },
		"cwd":           func(l *domain.Lease) { l.Runtimes[0].Process.WorkingDirectory = "other" },
		"source":        func(l *domain.Lease) { l.Runtimes[0].Process.SourceCommit = "different" },
		"runtime-type":  func(l *domain.Lease) { l.Runtimes[0].Type = "compose" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s, l := reservedProcess(t, "http")
			before, _ := json.Marshal(l)
			change(&l)
			if err := s.Save(context.Background(), l); err == nil {
				t.Fatal("unsafe mutation accepted")
			}
			got, err := s.Get(context.Background(), l.ID)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(got)
			if string(before) != string(after) {
				t.Fatal("failed Save mutated stored lease")
			}
			if processReservationCount(t, s, l.ID) != 1 {
				t.Fatal("failed Save changed reservation")
			}
		})
	}
}
func TestProcessReservationsRetainUntilBothStatesReleased(t *testing.T) {
	s, l := reservedProcess(t, "http")
	for _, states := range [][2]string{{"released", "quarantined"}, {"ready", "released"}, {"released", "releasing"}} {
		l.Desired, l.Observed = states[0], states[1]
		if err := s.Save(context.Background(), l); err != nil {
			t.Fatal(err)
		}
		if processReservationCount(t, s, l.ID) != 1 {
			t.Fatal("reservation removed before confirmed release")
		}
	}
	l.Desired, l.Observed = "released", "released"
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	if processReservationCount(t, s, l.ID) != 0 {
		t.Fatal("released reservation retained")
	}
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal("repeated released save failed", err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM persistent_processes WHERE lease_id=?`, l.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("historical snapshot lost", err)
	}
}
func TestProcessReleasedReservationsCannotResurrect(t *testing.T) {
	for _, ports := range [][]string{nil, {"http"}} {
		t.Run(fmt.Sprint(len(ports)), func(t *testing.T) {
			s, l := reservedProcess(t, ports...)
			l.Desired, l.Observed = "released", "released"
			if err := s.Save(context.Background(), l); err != nil {
				t.Fatal(err)
			}
			l.Desired, l.Observed = "ready", "ready"
			if err := s.Save(context.Background(), l); err == nil {
				t.Fatal("released process resurrected")
			}
			if processReservationCount(t, s, l.ID) != 0 {
				t.Fatal("reservation recreated")
			}
		})
	}
}
func TestProcessSaveRequiresExistingReservations(t *testing.T) {
	s, l := reservedProcess(t, "http")
	if _, err := s.db.Exec(`DELETE FROM process_port_reservations WHERE lease_id=?`, l.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(context.Background(), l); err == nil {
		t.Fatal("missing reservation silently recreated")
	}
}
func TestProcessPreparationEvidenceInitializesOnce(t *testing.T) {
	s, l := reservedProcess(t, "http")
	p := l.Runtimes[0].Process
	p.Executable = "native-server"
	p.ExecutableDigest = "digest"
	p.ResolvedDirectory = "source"
	p.State = "prepared"
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	p.ProcessID = 101
	p.ProcessStart = "first-instance"
	p.State = "running"
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	p.ProcessID = 102
	if err := s.Save(context.Background(), l); err == nil {
		t.Fatal("native identity reassigned")
	}
	p.ProcessID = 101
	p.ExecutableDigest = "other"
	if err := s.Save(context.Background(), l); err == nil {
		t.Fatal("executable identity changed")
	}
}
func TestProcessMalformedReservationRollsBack(t *testing.T) {
	for _, mutate := range []func(*domain.Lease){func(l *domain.Lease) { l.Runtimes[0].Process = nil }, func(l *domain.Lease) { l.Runtimes[0].Process.Command = nil }, func(l *domain.Lease) { l.Runtimes[0].Process.SourceCommit = "" }, func(l *domain.Lease) { l.Runtimes[0].Process.Ports["http"] = 22000 }, func(l *domain.Lease) { l.Runtimes[0].Process.State = "unknown" }, func(l *domain.Lease) { l.Runtimes[0].Process.ProcessID = 12 }} {
		s, err := Open(filepath.Join(t.TempDir(), "state.db"))
		if err != nil {
			t.Fatal(err)
		}
		l := processLease("bad", "http")
		mutate(&l)
		if err = s.Reserve(context.Background(), l, 1); err == nil {
			t.Fatal("malformed reservation accepted")
		}
		if processReservationCount(t, s, l.ID) != 0 {
			t.Fatal("failed reservation leaked port")
		}
		s.Close()
	}
}

func TestProcessTerminalReservationKeepsPostReleaseQuarantineVisible(t *testing.T) {
	s, l := reservedProcess(t, "http")
	l.Desired, l.Observed = "released", "released"
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	l.Observed = "quarantined"
	if err := s.Save(context.Background(), l); err != nil {
		t.Fatal("cannot record post-release anomaly", err)
	}
	if processReservationCount(t, s, l.ID) != 0 {
		t.Fatal("post-release observation reacquired port")
	}
	l.Desired, l.Observed = "ready", "ready"
	if err := s.Save(context.Background(), l); err == nil {
		t.Fatal("anomaly lost terminal reservation evidence")
	}
}

func TestProcessDynamicReservationAvoidsExternallyBoundPort(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	occupied := listener.Addr().(*net.TCPAddr).Port
	_, l := reservedProcess(t, "http", "admin")
	for _, port := range l.Runtimes[0].Process.Ports {
		if port == occupied {
			t.Fatal("allocator selected already bound external port")
		}
	}
}
