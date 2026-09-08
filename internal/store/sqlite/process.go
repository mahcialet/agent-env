package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"
	"sort"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
)

const processPortMin = 1
const processPortMax = 65535

type processRecord struct {
	domain.PersistentProcess
	ReservationsReleased bool `json:"reservations_released,omitempty"`
}

func saveProcesses(ctx context.Context, tx *sql.Tx, l *domain.Lease, insert bool) error {
	rows, err := tx.QueryContext(ctx, `SELECT runtime_name,snapshot_json FROM persistent_processes WHERE lease_id=?`, l.ID)
	if err != nil {
		return err
	}
	previous := map[string]processRecord{}
	for rows.Next() {
		var name string
		var data []byte
		if err = rows.Scan(&name, &data); err != nil {
			rows.Close()
			return err
		}
		var record processRecord
		if err = json.Unmarshal(data, &record); err != nil {
			rows.Close()
			return err
		}
		previous[name] = record
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	names := map[string]bool{}
	released := l.Desired == "released" && l.Observed == "released"
	for i := range l.Runtimes {
		r := &l.Runtimes[i]
		if r.Process == nil {
			if r.Type == "process" {
				return errors.New("process runtime is missing its snapshot")
			}
			continue
		}
		if r.Type != "process" || r.Name == "" || names[r.Name] {
			return errors.New("invalid or duplicate process runtime")
		}
		names[r.Name] = true
		p := r.Process
		if err = validateProcessSnapshot(*p); err != nil {
			return err
		}
		prior, exists := previous[r.Name]
		if insert {
			if exists || p.ProcessID != 0 || p.ProcessStart != "" || released {
				return errors.New("cannot reserve an existing or launched process snapshot")
			}
			for _, port := range p.Ports {
				if port < processPortMin || port > processPortMax {
					return errors.New("new process ports must be allocated by the registry")
				}
			}
		} else {
			if !exists {
				return errors.New("cannot add or rename process runtime during Save")
			}
			if prior.ReservationsReleased && l.Desired != "released" {
				return errors.New("cannot resurrect released process reservations")
			}
			if !immutableProcessDeclaration(prior.PersistentProcess, *p) {
				return errors.New("process declaration or reservation changed after allocation")
			}
			if prior.ProcessID != 0 && (p.ProcessID != prior.ProcessID || p.ProcessStart != prior.ProcessStart) {
				return errors.New("process identity cannot be reassigned or removed")
			}
			if prior.ProcessStart != "" && p.ProcessStart != prior.ProcessStart {
				return errors.New("process start identity changed")
			}
			for _, pair := range [][2]string{{prior.Executable, p.Executable}, {prior.ExecutableDigest, p.ExecutableDigest}, {prior.ExecutableOrigin, p.ExecutableOrigin}, {prior.ResolvedDirectory, p.ResolvedDirectory}, {prior.StdoutPath, p.StdoutPath}, {prior.StderrPath, p.StderrPath}, {prior.StateDirectory, p.StateDirectory}} {
				if pair[0] != "" && pair[0] != pair[1] {
					return errors.New("prepared process evidence changed")
				}
			}
			if prior.Executable != "" && prior.Directory != p.Directory {
				return errors.New("prepared process directory changed")
			}
		}
		portNames := make([]string, 0, len(p.Ports))
		for name := range p.Ports {
			portNames = append(portNames, name)
		}
		sort.Strings(portNames)
		for _, name := range portNames {
			port := p.Ports[name]
			if insert {
				if _, err = tx.ExecContext(ctx, `INSERT INTO process_port_reservations(port,lease_id,runtime_name,port_name) VALUES(?,?,?,?)`, port, l.ID, r.Name, name); err != nil {
					return err
				}
			} else if !prior.ReservationsReleased {
				var actual int
				if err = tx.QueryRowContext(ctx, `SELECT port FROM process_port_reservations WHERE lease_id=? AND runtime_name=? AND port_name=?`, l.ID, r.Name, name).Scan(&actual); err != nil {
					return fmt.Errorf("process port reservation unavailable: %w", err)
				}
				if actual != port {
					return errors.New("process port reservation identity mismatch")
				}
			}
		}
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM process_port_reservations WHERE lease_id=? AND runtime_name=?`, l.ID, r.Name).Scan(&count); err != nil {
			return err
		}
		expected := len(p.Ports)
		if exists && prior.ReservationsReleased {
			expected = 0
		}
		if count != expected {
			return errors.New("unexpected process port reservation count")
		}
		data, e := json.Marshal(processRecord{PersistentProcess: *p, ReservationsReleased: released || prior.ReservationsReleased})
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO persistent_processes(lease_id,runtime_name,snapshot_json) VALUES(?,?,?) ON CONFLICT(lease_id,runtime_name) DO UPDATE SET snapshot_json=excluded.snapshot_json`, l.ID, r.Name, string(data)); e != nil {
			return e
		}
	}
	for name := range previous {
		if !names[name] {
			return errors.New("cannot remove process runtime snapshot during Save")
		}
	}
	if released {
		_, err = tx.ExecContext(ctx, `DELETE FROM process_port_reservations WHERE lease_id=?`, l.ID)
	}
	return err
}

func immutableProcessDeclaration(a, b domain.PersistentProcess) bool {
	return a.WorkingDirectory == b.WorkingDirectory && a.SourceCommit == b.SourceCommit && slices.Equal(a.Command, b.Command) && maps.Equal(a.Env, b.Env) && maps.Equal(a.Ports, b.Ports)
}
func validateProcessSnapshot(p domain.PersistentProcess) error {
	if strings.TrimSpace(p.WorkingDirectory) == "" || len(p.Command) == 0 || p.Command[0] == "" || p.SourceCommit == "" || p.ProcessID < 0 || p.ProcessID == 0 && p.ProcessStart != "" {
		return errors.New("malformed process snapshot")
	}
	switch p.State {
	case "planned", "reserved", "prepared", "launching", "running", "degraded", "stopped", "quarantined", "released":
	default:
		return errors.New("unknown process snapshot state")
	}
	used := map[int]bool{}
	for name, port := range p.Ports {
		if name == "" || port < 0 || port != 0 && (port < processPortMin || port > processPortMax) || port != 0 && used[port] {
			return errors.New("invalid process port snapshot")
		}
		if port != 0 {
			used[port] = true
		}
	}
	return nil
}

// Allocation happens before serializing the lease payload, in the same locked
// transaction as normalized reservation insertion. Clone caller-owned snapshots.
func allocateProcesses(ctx context.Context, tx *sql.Tx, l *domain.Lease) (func(), error) {
	listeners := []net.Listener{}
	release := func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}
	fail := func(err error) (func(), error) { release(); return nil, err }
	rows, err := tx.QueryContext(ctx, `SELECT port FROM process_port_reservations`)
	if err != nil {
		return fail(err)
	}
	used := map[int]bool{}
	for rows.Next() {
		var port int
		if err = rows.Scan(&port); err != nil {
			rows.Close()
			return fail(err)
		}
		used[port] = true
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return fail(err)
	}
	l.Runtimes = append([]domain.Runtime(nil), l.Runtimes...)
	for i := range l.Runtimes {
		r := &l.Runtimes[i]
		if r.Process == nil {
			continue
		}
		p := *r.Process
		p.Ports = maps.Clone(p.Ports)
		r.Process = &p
		names := make([]string, 0, len(p.Ports))
		for name, port := range p.Ports {
			if port != 0 {
				return fail(errors.New("new process ports must be unassigned"))
			}
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			found := false
			for attempt := 0; attempt < 64; attempt++ {
				if err = ctx.Err(); err != nil {
					return fail(err)
				}
				listener, e := net.Listen("tcp4", "127.0.0.1:0")
				if e != nil {
					return fail(e)
				}
				port := listener.Addr().(*net.TCPAddr).Port
				if used[port] {
					_ = listener.Close()
					continue
				}
				listeners = append(listeners, listener)
				p.Ports[name] = port
				used[port] = true
				found = true
				break
			}
			if !found {
				return fail(errors.New("cannot allocate unreserved process TCP port"))
			}
		}
	}
	return release, nil
}
