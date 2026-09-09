// Package worker owns outbound execution receipts, separate from local resources.
package worker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/instance"
	_ "modernc.org/sqlite"
)

type Journal struct {
	db       *sql.DB
	release  func() error
	Identity protocol.WorkerIdentity
}
type Receipt struct {
	Operation protocol.Operation
	State     string
	Result    *protocol.Result
}

// OpenJournal excludes other processes for the entire worker lifetime. A new
// incarnation does not change the persistent host-instance or controller binding.
func OpenJournal(home, host string) (*Journal, error) {
	if !validIdentity(host) {
		return nil, errors.New("invalid worker host ID")
	}
	home, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	release, err := instance.Acquire(filepath.Join(home, "worker.lock"))
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, "worker.db")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		release()
		return nil, err
	}
	f.Close()
	db, err := sql.Open("sqlite", filepath.ToSlash(path))
	if err != nil {
		release()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	j := &Journal{db: db, release: release}
	fail := func(e error) (*Journal, error) { j.Close(); return nil, e }
	for _, q := range []string{
		"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA busy_timeout=5000",
		`CREATE TABLE IF NOT EXISTS worker_identity (singleton INTEGER PRIMARY KEY CHECK(singleton=1), host TEXT NOT NULL, instance TEXT NOT NULL, controller TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS receipts (id TEXT PRIMARY KEY, lease TEXT NOT NULL, epoch INTEGER NOT NULL, digest TEXT NOT NULL, operation BLOB NOT NULL, state TEXT NOT NULL, result BLOB, no_effect INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err = db.Exec(q); err != nil {
			return fail(err)
		}
	}
	// The additive migration is conservative for existing receipts: an unknown
	// historic effect boundary is never promoted to absence evidence.
	var columns int
	if err = db.QueryRow(`SELECT count(*) FROM pragma_table_info('receipts') WHERE name='no_effect'`).Scan(&columns); err != nil {
		return fail(err)
	}
	if columns == 0 {
		if _, err = db.Exec(`ALTER TABLE receipts ADD COLUMN no_effect INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fail(err)
		}
	}
	if _, err = db.Exec(`INSERT OR IGNORE INTO worker_identity VALUES(1,?,?, '')`, host, randomID()); err != nil {
		return fail(err)
	}
	if err = db.QueryRow(`SELECT host,instance,controller FROM worker_identity WHERE singleton=1`).Scan(&j.Identity.HostID, &j.Identity.HostInstanceID, &j.Identity.ControllerID); err != nil {
		return fail(err)
	}
	if j.Identity.HostID != host {
		return fail(errors.New("worker state belongs to another host ID"))
	}
	j.Identity.Incarnation = randomID()
	return j, nil
}
func randomID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func validIdentity(s string) bool {
	if len(s) == 0 || len(s) > 128 || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func (j *Journal) Close() error { return errors.Join(j.db.Close(), j.release()) }

// Bind may only be called with identity learned over authenticated transport.
func (j *Journal) Bind(ctx context.Context, controller string) error {
	if !validIdentity(controller) {
		return errors.New("invalid controller ID")
	}
	r, e := j.db.ExecContext(ctx, `UPDATE worker_identity SET controller=? WHERE singleton=1 AND (controller='' OR controller=?)`, controller, controller)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return errors.New("worker is bound to a different controller")
	}
	j.Identity.ControllerID = controller
	return nil
}
func (j *Journal) Receive(ctx context.Context, op protocol.Operation) (Receipt, error) {
	var empty Receipt
	if op.ControllerID == "" || op.ControllerID != j.Identity.ControllerID || op.HostID != j.Identity.HostID || op.HostInstanceID != j.Identity.HostInstanceID || op.Epoch <= 0 || !validIdentity(op.ID) || !validIdentity(op.LeaseID) {
		return empty, errors.New("operation assignment identity mismatch")
	}
	// Transport state/results are not execution identity. Payload bytes are exact:
	// retries must preserve the submitted immutable envelope.
	op.State = ""
	op.Result = nil
	raw, e := json.Marshal(op)
	if e != nil {
		return empty, e
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	tx, e := j.db.BeginTx(ctx, nil)
	if e != nil {
		return empty, e
	}
	defer tx.Rollback()
	var stored string
	e = tx.QueryRowContext(ctx, `SELECT digest FROM receipts WHERE id=?`, op.ID).Scan(&stored)
	if e == nil {
		if stored != digest {
			return empty, errors.New("conflicting duplicate operation")
		}
		if e = tx.Commit(); e != nil {
			return empty, e
		}
		return j.Get(ctx, op.ID)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return empty, e
	}
	var epoch sql.NullInt64
	if e = tx.QueryRowContext(ctx, `SELECT max(epoch) FROM receipts WHERE lease=?`, op.LeaseID).Scan(&epoch); e != nil {
		return empty, e
	}
	if epoch.Valid && epoch.Int64 != op.Epoch {
		return empty, errors.New("assignment epoch cannot change for existing local lease")
	}
	var active int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM receipts WHERE lease=? AND state NOT IN ('completed','uncertain')`, op.LeaseID).Scan(&active); e != nil {
		return empty, e
	}
	if active != 0 {
		return empty, errors.New("lease has an unfinished remote operation")
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO receipts(id,lease,epoch,digest,operation,state) VALUES(?,?,?,?,?,'received')`, op.ID, op.LeaseID, op.Epoch, digest, raw); e != nil {
		return empty, e
	}
	if e = tx.Commit(); e != nil {
		return empty, e
	}
	return Receipt{Operation: op, State: "received"}, nil
}
func (j *Journal) Get(ctx context.Context, id string) (Receipt, error) {
	var r Receipt
	var op, result []byte
	e := j.db.QueryRowContext(ctx, `SELECT operation,state,result FROM receipts WHERE id=?`, id).Scan(&op, &r.State, &result)
	if e != nil {
		return r, e
	}
	if e = json.Unmarshal(op, &r.Operation); e != nil {
		return r, e
	}
	if len(result) > 0 {
		e = json.Unmarshal(result, &r.Result)
	}
	return r, e
}
func (j *Journal) Advance(ctx context.Context, id, from, to string) error {
	if !(from == "received" && to == "prepared" || from == "prepared" && to == "effect_started" || from == "result_pending" && to == "completed") {
		return errors.New("invalid journal transition")
	}
	r, e := j.db.ExecContext(ctx, `UPDATE receipts SET state=? WHERE id=? AND state=?`, to, id, from)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n != 1 {
		e = errors.New("journal transition conflict")
	}
	return e
}

// StoreResult commits output before any upload. An effect-started record without
// output is never reset to received: recovery must inspect it, not replay it.
func (j *Journal) StoreResult(ctx context.Context, id string, result protocol.Result) error {
	receipt, e := j.Get(ctx, id)
	if e != nil {
		return e
	}
	op := receipt.Operation
	if result.OperationID != id || result.LeaseID != op.LeaseID || result.Epoch != op.Epoch || result.ControllerID != op.ControllerID || result.HostID != op.HostID || result.HostInstanceID != op.HostInstanceID {
		return errors.New("result identity mismatch")
	}
	raw, e := json.Marshal(result)
	if e != nil {
		return e
	}
	r, e := j.db.ExecContext(ctx, `UPDATE receipts SET no_effect=CASE WHEN state IN ('received','prepared') THEN 1 ELSE 0 END,result=?,state='result_pending' WHERE id=? AND result IS NULL AND state IN ('received','prepared','effect_started')`, raw, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n != 1 {
		e = errors.New("result already persisted or operation closed")
	}
	return e
}

// ProvesNoEffects uses the receipt's durable pre-effect boundary, not payload
// claims or an absent local lease row. Every prior command must be closed without
// effects, and the original create must carry this exact assignment identity.
func (j *Journal) ProvesNoEffects(ctx context.Context, current protocol.Operation) (bool, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT operation,state,no_effect FROM receipts WHERE lease=? AND id!=?`, current.LeaseID, current.ID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	created := false
	for rows.Next() {
		var raw []byte
		var state string
		var noEffect int
		if err = rows.Scan(&raw, &state, &noEffect); err != nil {
			return false, err
		}
		var op protocol.Operation
		if err = json.Unmarshal(raw, &op); err != nil {
			return false, err
		}
		if op.ControllerID != current.ControllerID || op.HostID != current.HostID || op.HostInstanceID != current.HostInstanceID || op.Epoch != current.Epoch || noEffect != 1 || (state != "completed" && state != "result_pending") {
			return false, nil
		}
		if op.Kind == "create" {
			created = true
		}
	}
	return created, rows.Err()
}
func (j *Journal) Pending(ctx context.Context) ([]Receipt, error) {
	rows, e := j.db.QueryContext(ctx, `SELECT operation,state,result FROM receipts WHERE state!='completed' ORDER BY rowid`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Receipt
	for rows.Next() {
		var r Receipt
		var op, result []byte
		if e = rows.Scan(&op, &r.State, &result); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(op, &r.Operation); e != nil {
			return nil, e
		}
		if len(result) > 0 {
			if e = json.Unmarshal(result, &r.Result); e != nil {
				return nil, e
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
