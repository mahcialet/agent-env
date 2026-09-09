package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
)

const (
	androidConsolePortFirst = 5554
	androidConsolePortLast  = 5682
	// AndroidSlotCapacity is the fixed pool of console/ADB port pairs allocated
	// by this registry, shared with worker capacity advertisement.
	AndroidSlotCapacity = (androidConsolePortLast-androidConsolePortFirst)/2 + 1
)

var androidName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var androidTemplate = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func androidPath(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func validateAndroid(r domain.Runtime) error {
	a := r.Android
	if r.Type != "android-emulator" {
		if a != nil {
			return errors.New("Android reservation requires android-emulator runtime")
		}
		return nil
	}
	if a == nil || r.Name == "" {
		return errors.New("Android runtime requires reservation metadata")
	}
	if !androidName.MatchString(a.AVDName) || !androidTemplate.MatchString(a.Template) || a.Template == "." || a.Template == ".." || !filepath.IsAbs(a.AVDHome) || !filepath.IsAbs(a.AVDPath) || androidPath(a.AVDPath) != androidPath(filepath.Join(a.AVDHome, a.AVDName+".avd")) {
		return errors.New("Android reservation requires valid template, AVD name and private absolute AVD paths")
	}
	return nil
}

// Allocation is part of Reserve's BEGIN IMMEDIATE transaction. The registry,
// rather than a host port scan, is the authority for this finite slot pool.
func allocateAndroid(ctx context.Context, tx *sql.Tx, l domain.Lease) (domain.Lease, error) {
	l.Runtimes = append([]domain.Runtime(nil), l.Runtimes...)
	used := map[int]bool{}
	var homes []string
	rows, err := tx.QueryContext(ctx, "SELECT console_port,avd_home FROM android_reservations WHERE active=1")
	if err != nil {
		return l, err
	}
	for rows.Next() {
		var p int
		var home string
		if err = rows.Scan(&p, &home); err != nil {
			rows.Close()
			return l, err
		}
		used[p] = true
		homes = append(homes, home)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return l, err
	}
	for i, r := range l.Runtimes {
		if err = validateAndroid(r); err != nil {
			return l, err
		}
		if r.Android == nil {
			continue
		}
		if l.Observed == "released" {
			return l, errors.New("cannot reserve Android resources for released lease")
		}
		a := *r.Android
		home := androidPath(a.AVDHome)
		for _, existing := range homes {
			if androidPathsOverlap(home, existing) {
				return l, errors.New("Android writable home overlaps an existing reservation")
			}
		}
		homes = append(homes, home)
		if a.ConsolePort != 0 || a.ADBPort != 0 || a.Serial != "" || a.ProcessID != 0 || a.ProcessStart != "" || (a.State != "" && a.State != "reserved") {
			return l, errors.New("Android reservation must not supply allocated ports or process identity")
		}
		for p := androidConsolePortFirst; p <= androidConsolePortLast; p += 2 {
			if !used[p] {
				a.ConsolePort = p
				used[p] = true
				break
			}
		}
		if a.ConsolePort == 0 {
			return l, errors.New("Android emulator slots exhausted")
		}
		a.ADBPort = a.ConsolePort + 1
		a.Serial = fmt.Sprintf("emulator-%d", a.ConsolePort)
		a.State = "reserved"
		l.Runtimes[i].Android = &a
	}
	return l, nil
}

func androidPathsOverlap(a, b string) bool {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// Reservations survive quarantine and cannot silently disappear or change
// identity during a snapshot save. Released rows retain historical identity.
func saveAndroid(ctx context.Context, tx *sql.Tx, l domain.Lease, insert bool) error {
	count := 0
	for _, r := range l.Runtimes {
		if err := validateAndroid(r); err != nil {
			return err
		}
		a := r.Android
		if a == nil {
			continue
		}
		count++
		if a.ConsolePort < androidConsolePortFirst || a.ConsolePort > androidConsolePortLast || a.ConsolePort%2 != 0 || a.ADBPort != a.ConsolePort+1 || a.Serial != fmt.Sprintf("emulator-%d", a.ConsolePort) {
			return errors.New("invalid Android port reservation")
		}
		active := l.Observed != "released"
		if active {
			rows, err := tx.QueryContext(ctx, "SELECT avd_home FROM android_reservations WHERE active=1 AND NOT (lease_id=? AND runtime_name=?)", l.ID, r.Name)
			if err != nil {
				return err
			}
			for rows.Next() {
				var home string
				if err := rows.Scan(&home); err != nil {
					rows.Close()
					return err
				}
				if androidPathsOverlap(androidPath(a.AVDHome), home) {
					rows.Close()
					return errors.New("Android writable home overlaps an existing reservation")
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
		}
		if insert {
			_, err := tx.ExecContext(ctx, `INSERT INTO android_reservations(lease_id,runtime_name,avd_name,avd_home,avd_path,console_port,adb_port,serial,active) VALUES(?,?,?,?,?,?,?,?,?)`, l.ID, r.Name, a.AVDName, androidPath(a.AVDHome), androidPath(a.AVDPath), a.ConsolePort, a.ADBPort, a.Serial, active)
			if err != nil {
				return err
			}
		} else {
			res, err := tx.ExecContext(ctx, `UPDATE android_reservations SET active=? WHERE lease_id=? AND runtime_name=? AND avd_name=? AND avd_home=? AND avd_path=? AND console_port=? AND adb_port=? AND serial=?`, active, l.ID, r.Name, a.AVDName, androidPath(a.AVDHome), androidPath(a.AVDPath), a.ConsolePort, a.ADBPort, a.Serial)
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return fmt.Errorf("%w: Android reservation identity changed", domain.ErrResourceIdentity)
			}
		}
	}
	var recorded int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM android_reservations WHERE lease_id=?", l.ID).Scan(&recorded); err != nil {
		return err
	}
	if recorded != count {
		return fmt.Errorf("%w: Android reservation removed", domain.ErrResourceIdentity)
	}
	return nil
}
