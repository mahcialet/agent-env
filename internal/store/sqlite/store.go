// Package sqlite persists registry intent and evidence in an embedded SQLite database.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/migrations"
	sqliteDriver "modernc.org/sqlite"
	sqliteCodes "modernc.org/sqlite/lib"
)

var ErrNotFound = sql.ErrNoRows
var ErrBusy = errors.New("lease operation is already in progress")
var ErrCapacity = errors.New("maximum active lease count reached")

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: uriPath(filepath.ToSlash(abs))}
	q := url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(10000)"}, "_txlock": {"immediate"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = s.initializeWithRetry(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// A drive letter belongs in the URI path, never the URI authority.
func uriPath(path string) string {
	if len(path) >= 3 && path[1] == ':' && path[2] == '/' && !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

// WAL promotion can return SQLITE_BUSY immediately when concurrent readers
// also need its exclusive lock, bypassing busy_timeout. Retry only initialization;
// each failed migration transaction is rolled back before the next attempt.
func (s *Store) initializeWithRetry(ctx context.Context) error {
	for {
		err := s.initialize(ctx)
		if err == nil {
			return nil
		}
		var sqliteErr *sqliteDriver.Error
		if !errors.As(err, &sqliteErr) || sqliteErr.Code()&0xff != sqliteCodes.SQLITE_BUSY {
			return err
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ctx.Err(), err)
		case <-timer.C:
		}
	}
}

func (s *Store) initialize(ctx context.Context) error {
	var foreign, busy int
	var journal string
	if err := s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreign); err != nil {
		return err
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		return err
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journal); err != nil {
		return err
	}
	if foreign != 1 || busy < 1000 || journal != "wal" {
		return fmt.Errorf("SQLite invariants unavailable: foreign_keys=%d busy_timeout=%d journal_mode=%s", foreign, busy, journal)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		return err
	}
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations WHERE name=?", entry.Name()).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		data, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(name,applied_at) VALUES(?,?)", entry.Name(), stamp(time.Now())); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Close() error  { return s.db.Close() }
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func encoded(v any) string     { b, _ := json.Marshal(v); return string(b) }

func (s *Store) Reserve(ctx context.Context, lease domain.Lease, maxActive int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if maxActive > 0 {
		var count int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM leases WHERE observed_state <> 'released'").Scan(&count); err != nil {
			return err
		}
		if count >= maxActive {
			return ErrCapacity
		}
	}
	lease, err = allocateAndroid(ctx, tx, lease)
	if err != nil {
		return err
	}
	releaseProcessPorts, err := allocateProcesses(ctx, tx, &lease)
	if err != nil {
		return err
	}
	defer releaseProcessPorts()
	if err = writeLease(ctx, tx, lease, true); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Save(ctx context.Context, lease domain.Lease) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fence(ctx, tx, lease.ID); err != nil {
		return err
	}
	var previousJSON string
	if err = tx.QueryRowContext(ctx, "SELECT payload FROM leases WHERE id=?", lease.ID).Scan(&previousJSON); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	var previous domain.Lease
	if err = json.Unmarshal([]byte(previousJSON), &previous); err != nil {
		return err
	}
	if (previous.Management == nil) != (lease.Management == nil) || previous.Management != nil && *previous.Management != *lease.Management {
		return errors.New("lease controller assignment is immutable")
	}
	if err = writeLease(ctx, tx, lease, false); err != nil {
		return err
	}
	if err = fence(ctx, tx, lease.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func writeLease(ctx context.Context, tx *sql.Tx, l domain.Lease, insert bool) error {
	if l.ID == "" {
		return errors.New("lease id is required")
	}
	payload, err := json.Marshal(l)
	if err != nil {
		return err
	}
	args := []any{l.Owner, l.Purpose, l.Mode, l.Repository, l.Stack, l.Desired, l.Observed, stamp(l.CreatedAt), stamp(l.HeartbeatAt), stamp(l.ExpiresAt), l.ManifestDigest, l.SourceSetDigest, string(payload), l.ID}
	query := "UPDATE leases SET owner=?,purpose=?,mode=?,repository=?,stack=?,desired_state=?,observed_state=?,created_at=?,heartbeat_at=?,expires_at=?,manifest_digest=?,source_set_digest=?,payload=? WHERE id=?"
	if insert {
		query = "INSERT INTO leases(owner,purpose,mode,repository,stack,desired_state,observed_state,created_at,heartbeat_at,expires_at,manifest_digest,source_set_digest,payload,id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound
	}
	if err := saveAndroid(ctx, tx, l, insert); err != nil {
		return err
	}
	if err := saveProcesses(ctx, tx, &l, insert); err != nil {
		return err
	}
	for _, table := range []string{"lease_sources", "lease_components", "lease_runtimes", "runtime_resources"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE lease_id=?", l.ID); err != nil {
			return err
		}
	}
	for _, v := range l.Sources {
		if v.RepositoryID == "" || v.Alias == "" {
			return errors.New("source repository id and alias are required")
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO repositories(id,canonical_path) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET canonical_path=excluded.canonical_path", v.RepositoryID, v.RepositoryPath); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO lease_sources(lease_id,alias,repository_id,requested_ref,resolved_commit,worktree_path,checkout_mode,writable,resolved_at) VALUES(?,?,?,?,?,?,?,?,?)", l.ID, v.Alias, v.RepositoryID, v.RequestedRef, v.Commit, v.WorktreePath, v.CheckoutMode, v.Writable, stamp(v.ResolvedAt)); err != nil {
			return err
		}
	}
	for order, v := range l.Components {
		if _, err := tx.ExecContext(ctx, "INSERT INTO lease_components(lease_id,name,runtime,services,capabilities,resolution_order) VALUES(?,?,?,?,?,?)", l.ID, v.Name, v.Runtime, encoded(v.Services), encoded(v.Capabilities), order); err != nil {
			return err
		}
	}
	for _, v := range l.Runtimes {
		if _, err := tx.ExecContext(ctx, "INSERT INTO lease_runtimes(lease_id,name,type,project,context,active,payload) VALUES(?,?,?,?,?,?,?)", l.ID, v.Name, v.Type, v.Project, v.Context, l.Observed != "released", encoded(v)); err != nil {
			return err
		}
	}
	for _, v := range l.Resources {
		if v.LeaseID != "" && v.LeaseID != l.ID {
			return errors.New("resource belongs to another lease")
		}
		if v.ID == "" {
			return errors.New("resource id is required")
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO runtime_resources(id,lease_id,runtime,kind,external_id,metadata) VALUES(?,?,?,?,?,?)", v.ID, l.ID, v.Runtime, v.Kind, v.ExternalID, encoded(v.Metadata)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (domain.Lease, error) {
	var l domain.Lease
	var b string
	err := s.db.QueryRowContext(ctx, "SELECT payload FROM leases WHERE id=?", id).Scan(&b)
	if err == nil {
		err = json.Unmarshal([]byte(b), &l)
	}
	return l, err
}
func (s *Store) List(ctx context.Context) ([]domain.Lease, error) {
	return readRows[domain.Lease](ctx, s.db, "SELECT payload FROM leases ORDER BY created_at,id")
}

func readRows[T any](ctx context.Context, db *sql.DB, query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]T, 0)
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal([]byte(b), &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (s *Store) Event(ctx context.Context, v domain.Event) error {
	_, err := s.execBound(ctx, v.LeaseID, "INSERT INTO events(id,lease_id,time,type,message,payload) VALUES(?,?,?,?,?,?)", v.ID, v.LeaseID, stamp(v.Time), v.Type, v.Message, encoded(v))
	return err
}
func (s *Store) Events(ctx context.Context, id string) ([]domain.Event, error) {
	return readRows[domain.Event](ctx, s.db, "SELECT payload FROM events WHERE lease_id=? ORDER BY time,id", id)
}
func (s *Store) SaveRun(ctx context.Context, v domain.CommandRun) error {
	res, err := s.execBound(ctx, v.LeaseID, "INSERT INTO command_runs(id,lease_id,name,source_alias,working_directory,argv_json,started_at,finished_at,exit_code,stdout_path,stderr_path,status,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,source_alias=excluded.source_alias,working_directory=excluded.working_directory,argv_json=excluded.argv_json,started_at=excluded.started_at,finished_at=excluded.finished_at,exit_code=excluded.exit_code,stdout_path=excluded.stdout_path,stderr_path=excluded.stderr_path,status=excluded.status,payload=excluded.payload WHERE command_runs.lease_id=excluded.lease_id", v.ID, v.LeaseID, v.Name, v.Source, v.Directory, encoded(v.Argv), stamp(v.StartedAt), stamp(v.FinishedAt), v.ExitCode, v.StdoutPath, v.StderrPath, v.Status, encoded(v))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n != 1 {
		return errors.New("command run belongs to another lease")
	}
	return err
}
func (s *Store) Runs(ctx context.Context, id string) ([]domain.CommandRun, error) {
	return readRows[domain.CommandRun](ctx, s.db, "SELECT payload FROM command_runs WHERE lease_id=? ORDER BY started_at,id", id)
}
func (s *Store) SaveArtifact(ctx context.Context, v domain.Artifact) error {
	var run any
	if v.RunID != "" {
		run = v.RunID
	}
	_, err := s.execBound(ctx, v.LeaseID, "INSERT INTO artifacts(id,lease_id,run_id,kind,path,digest,created_at,payload) VALUES(?,?,?,?,?,?,?,?)", v.ID, v.LeaseID, run, v.Kind, v.Path, v.Digest, stamp(v.CreatedAt), encoded(v))
	return err
}
func (s *Store) Artifacts(ctx context.Context, id string) ([]domain.Artifact, error) {
	return readRows[domain.Artifact](ctx, s.db, "SELECT payload FROM artifacts WHERE lease_id=? ORDER BY created_at,id", id)
}

// Lock bindings are private capabilities. WithoutCancel retains them but cannot
// bypass token/expiry checks or the independent lock-loss flag.
type lockKey struct{}
type lockBinding struct {
	leaseID, token string
	lost           atomic.Bool
	lose           func(error)
}
type operationContext struct {
	context.Context
	binding *lockBinding
}

func (c *operationContext) LockLost() bool { return c.binding.lost.Load() }

func fence(ctx context.Context, tx *sql.Tx, id string) error {
	binding, _ := ctx.Value(lockKey{}).(*lockBinding)
	if binding != nil && (binding.leaseID != id || binding.lost.Load()) {
		return domain.ErrLockLost
	}
	var token string
	var expires int64
	err := tx.QueryRowContext(ctx, "SELECT token,expires_at FROM operation_locks WHERE lease_id=?", id).Scan(&token, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		if binding != nil {
			binding.lose(errors.New("operation lock row disappeared"))
			return domain.ErrLockLost
		}
		return nil
	}
	if err != nil {
		return err
	}
	if binding == nil {
		return ErrBusy
	}
	if binding.token != token || expires <= time.Now().UnixNano() {
		binding.lose(errors.New("operation token or expiration no longer matches"))
		return domain.ErrLockLost
	}
	return nil
}

func (s *Store) execBound(ctx context.Context, id, query string, args ...any) (sql.Result, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := fence(ctx, tx, id); err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := fence(ctx, tx, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// AcquireContext binds every lease mutation to a renewable operation capability.
// The returned context cancels on lock loss. It additionally exposes LockLost()
// so a prior caller cancellation cannot mask a later ownership failure.
func (s *Store) AcquireContext(ctx context.Context, id, token string, ttl time.Duration) (context.Context, func() error, error) {
	if token == "" || ttl < time.Second {
		return nil, nil, errors.New("lock requires a token and ttl of at least one second")
	}
	now := time.Now()
	result, err := s.db.ExecContext(ctx, "INSERT INTO operation_locks(lease_id,token,expires_at) VALUES(?,?,?) ON CONFLICT(lease_id) DO UPDATE SET token=excluded.token,expires_at=excluded.expires_at WHERE operation_locks.expires_at<=?", id, token, now.Add(ttl).UnixNano(), now.UnixNano())
	if err != nil {
		return nil, nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, nil, err
	}
	if affected != 1 {
		return nil, nil, ErrBusy
	}
	binding := &lockBinding{leaseID: id, token: token}
	base, cancel := context.WithCancelCause(ctx)
	operation := &operationContext{Context: context.WithValue(base, lockKey{}, binding), binding: binding}
	stop := make(chan struct{})
	renewed := make(chan time.Time, 1)
	done := make(chan struct{})
	workerCtx, stopWorker := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	workers.Add(2)
	var failureMu sync.Mutex
	var renewalErr error
	lose := func(err error) {
		binding.lost.Store(true)
		failureMu.Lock()
		if renewalErr == nil {
			renewalErr = fmt.Errorf("%w: %v", domain.ErrLockLost, err)
		}
		cause := renewalErr
		failureMu.Unlock()
		cancel(cause)
		stopWorker()
	}
	binding.lose = lose
	// Independent watchdog fires even while a database renewal is blocked. The
	// safety margin ensures cancellation starts before the old token may be stolen.
	go func() {
		defer workers.Done()
		timer := time.NewTimer(time.Until(now.Add(ttl - ttl/6)))
		defer timer.Stop()
		for {
			select {
			case <-stop:
				return
			case expires := <-renewed:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(time.Until(expires.Add(-ttl / 6)))
			case <-timer.C:
				lose(errors.New("operation renewal deadline exceeded"))
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		ticker := time.NewTicker(ttl / 3)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if binding.lost.Load() {
					return
				}
				refreshCtx, refreshCancel := context.WithTimeout(workerCtx, ttl/3)
				refreshed := time.Now()
				expires := refreshed.Add(ttl)
				result, err := s.db.ExecContext(refreshCtx, "UPDATE operation_locks SET expires_at=? WHERE lease_id=? AND token=? AND expires_at>?", expires.UnixNano(), id, token, refreshed.UnixNano())
				refreshCancel()
				if err == nil {
					var n int64
					n, err = result.RowsAffected()
					if err == nil && n != 1 {
						err = ErrBusy
					}
				}
				if err != nil {
					select {
					case <-stop:
						return
					default:
					}
					lose(err)
					return
				}
				if binding.lost.Load() {
					return
				}
				select {
				case renewed <- expires:
				case <-stop:
					return
				}
			}
		}
	}()
	go func() { workers.Wait(); close(done) }()
	var once sync.Once
	var releaseErr error
	release := func() error {
		once.Do(func() {
			close(stop)
			stopWorker()
			<-done
			releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer releaseCancel()
			_, releaseErr = s.db.ExecContext(releaseCtx, "DELETE FROM operation_locks WHERE lease_id=? AND token=?", id, token)
			failureMu.Lock()
			releaseErr = errors.Join(renewalErr, releaseErr)
			failureMu.Unlock()
			cancel(context.Canceled)
		})
		return releaseErr
	}
	return operation, release, nil
}

// Acquire is compatibility-only. New operation callers must propagate the
// context returned by AcquireContext to both external commands and mutations.
func (s *Store) Acquire(ctx context.Context, id, token string, ttl time.Duration) (func() error, error) {
	_, release, err := s.AcquireContext(ctx, id, token, ttl)
	return release, err
}
