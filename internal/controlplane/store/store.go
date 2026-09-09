// Package store owns controller-local durable metadata, not worker resources.
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

type Fault struct {
	Code    string
	Message string
}

func (e *Fault) Error() string         { return e.Message }
func fault(code, message string) error { return &Fault{Code: code, Message: message} }

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidID(s string) bool     { return identifier.MatchString(s) }
func ValidDigest(s string) bool { return digest.MatchString(s) }

type Store struct {
	// ProductVersion is configured once before serving controller requests.
	ProductVersion string
	db             *sql.DB
	Path           string
	ID             string
	OfflineAfter   time.Duration
}

func Open(path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	// A drive-qualified Windows path belongs in URI.Path, never URI.Host.
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	db, err := sql.Open("sqlite", u.String()+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_txlock=immediate")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, Path: abs, OfflineAfter: 30 * time.Second}
	schema := `
 PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS metadata(key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS enrollments(fingerprint TEXT PRIMARY KEY,role TEXT NOT NULL,host_id TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS hosts(id TEXT PRIMARY KEY,instance_id TEXT NOT NULL,incarnation TEXT NOT NULL,data BLOB NOT NULL,draining INTEGER NOT NULL DEFAULT 0,removed INTEGER NOT NULL DEFAULT 0,last_seen INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS leases(id TEXT PRIMARY KEY,host_id TEXT NOT NULL,instance_id TEXT NOT NULL,epoch INTEGER NOT NULL,state TEXT NOT NULL,slots INTEGER NOT NULL,data BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS operations(id TEXT PRIMARY KEY,request_hash TEXT NOT NULL,lease_id TEXT NOT NULL,epoch INTEGER NOT NULL,kind TEXT NOT NULL,state TEXT NOT NULL,payload BLOB NOT NULL,result BLOB,result_hash TEXT,created INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS blob_refs(digest TEXT NOT NULL,lease_id TEXT NOT NULL,kind TEXT NOT NULL,PRIMARY KEY(digest,lease_id,kind));
 CREATE INDEX IF NOT EXISTS operations_lease ON operations(lease_id,created);
 `
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec("INSERT OR IGNORE INTO metadata(key,value) VALUES('controller_id',?)", "controller-"+ulid.Make().String()); err != nil {
		db.Close()
		return nil, err
	}
	if err = db.QueryRow("SELECT value FROM metadata WHERE key='controller_id'").Scan(&s.ID); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Enroll(ctx context.Context, e protocol.Enrollment) error {
	if !ValidDigest(e.Fingerprint) || (e.Role != protocol.RoleClient && e.Role != protocol.RoleWorker) || (e.Role == protocol.RoleWorker && !ValidID(e.HostID)) || (e.Role == protocol.RoleClient && e.HostID != "") {
		return fault("invalid", "invalid certificate enrollment")
	}
	var role, host string
	err := s.db.QueryRowContext(ctx, "SELECT role,host_id FROM enrollments WHERE fingerprint=?", e.Fingerprint).Scan(&role, &host)
	if err == nil {
		if role != e.Role || host != e.HostID {
			return fault("conflict", "certificate already enrolled with different authority")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO enrollments(fingerprint,role,host_id) VALUES(?,?,?)", e.Fingerprint, e.Role, e.HostID)
	return err
}
func (s *Store) Authorize(ctx context.Context, fingerprint string) (protocol.Enrollment, error) {
	var e protocol.Enrollment
	e.Fingerprint = fingerprint
	if err := s.db.QueryRowContext(ctx, "SELECT role,host_id FROM enrollments WHERE fingerprint=?", fingerprint).Scan(&e.Role, &e.HostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e, fault("forbidden", "certificate is not enrolled")
		}
		return e, err
	}
	return e, nil
}
func canonical(v any) ([]byte, string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	var value any
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.UseNumber()
	if err = d.Decode(&value); err != nil {
		return nil, "", err
	}
	b, err = json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(b)
	return b, hex.EncodeToString(sum[:]), nil
}
func (s *Store) Register(ctx context.Context, r protocol.RegisterRequest) (protocol.Host, error) {
	var out protocol.Host
	if r.ControllerID != "" && r.ControllerID != s.ID {
		return out, fault("identity", "controller identity mismatch")
	}
	if !ValidID(r.HostID) || !ValidID(r.HostInstanceID) || !ValidID(r.Incarnation) || r.ProductVersion == "" || r.OS == "" || r.Arch == "" || r.Capacity.MaxLeases < 1 || r.Capacity.MaxLeases > 10000 || r.Capacity.AndroidSlots < 0 || r.Capacity.AndroidSlots > 10000 || len(r.Capabilities) > 128 {
		return out, fault("invalid", "invalid host registration")
	}
	for _, c := range r.Capabilities {
		if !ValidID(c) {
			return out, fault("invalid", "invalid host capability")
		}
	}
	sort.Strings(r.Capabilities)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var prior, inc string
	var draining, removed bool
	err = tx.QueryRowContext(ctx, "SELECT instance_id,incarnation,draining,removed FROM hosts WHERE id=?", r.HostID).Scan(&prior, &inc, &draining, &removed)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	if err == nil && (prior != r.HostInstanceID || removed) {
		return out, fault("identity", "host ID is bound to a different or removed instance")
	}
	out = protocol.Host{WorkerIdentity: r.WorkerIdentity, ProductVersion: r.ProductVersion, OS: r.OS, Arch: r.Arch, Capabilities: r.Capabilities, Capacity: r.Capacity, Draining: draining, Online: true, LastSeen: time.Now().UnixNano()}
	out.ProtocolVersion = r.ProtocolVersion
	s.compatibility(&out)
	out.ControllerID = s.ID
	data, _ := json.Marshal(out)
	_, err = tx.ExecContext(ctx, "INSERT INTO hosts(id,instance_id,incarnation,data,draining,removed,last_seen) VALUES(?,?,?,?,?,0,?) ON CONFLICT(id) DO UPDATE SET incarnation=excluded.incarnation,data=excluded.data,last_seen=excluded.last_seen", r.HostID, r.HostInstanceID, r.Incarnation, data, draining, out.LastSeen)
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) worker(ctx context.Context, q queryer, w protocol.WorkerIdentity) error {
	if w.ControllerID != s.ID {
		return fault("identity", "controller identity mismatch")
	}
	var instance, inc string
	var removed bool
	if err := q.QueryRowContext(ctx, "SELECT instance_id,incarnation,removed FROM hosts WHERE id=?", w.HostID).Scan(&instance, &inc, &removed); err != nil {
		return fault("identity", "worker is not registered")
	}
	if removed || w.HostInstanceID != instance || w.Incarnation != inc {
		return fault("identity", "worker identity or incarnation mismatch")
	}
	h, err := s.host(ctx, q, w.HostID)
	if err != nil {
		return err
	}
	if !h.Compatible {
		return fault("version", h.CompatibilityError)
	}
	return nil
}

func (s *Store) compatibility(h *protocol.Host) {
	h.Compatible = false
	switch {
	case h.ProtocolVersion != protocol.Version:
		h.CompatibilityError = "incompatible protocol version"
	case s.ProductVersion != "" && h.ProductVersion != s.ProductVersion:
		h.CompatibilityError = "incompatible product version"
	default:
		h.Compatible = true
		h.CompatibilityError = ""
	}
}
func (s *Store) Heartbeat(ctx context.Context, w protocol.WorkerIdentity) (protocol.Host, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.Host{}, err
	}
	defer tx.Rollback()
	if err = s.worker(ctx, tx, w); err != nil {
		return protocol.Host{}, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE hosts SET last_seen=? WHERE id=?", time.Now().UnixNano(), w.HostID); err != nil {
		return protocol.Host{}, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Host{}, err
	}
	return s.GetHost(ctx, w.HostID)
}
func (s *Store) GetHost(ctx context.Context, id string) (protocol.Host, error) {
	return s.host(ctx, s.db, id)
}
func (s *Store) host(ctx context.Context, q queryer, id string) (protocol.Host, error) {
	var h protocol.Host
	var data []byte
	if err := q.QueryRowContext(ctx, "SELECT data,draining,removed,last_seen FROM hosts WHERE id=?", id).Scan(&data, &h.Draining, &h.Removed, &h.LastSeen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return h, fault("not_found", "host not found")
		}
		return h, err
	}
	var stored protocol.Host
	if err := json.Unmarshal(data, &stored); err != nil {
		return h, err
	}
	stored.Draining = h.Draining
	stored.Removed = h.Removed
	stored.LastSeen = h.LastSeen
	stored.Online = !stored.Removed && time.Since(time.Unix(0, h.LastSeen)) <= s.OfflineAfter
	s.compatibility(&stored)
	return stored, nil
}
func (s *Store) ListHosts(ctx context.Context) ([]protocol.Host, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM hosts ORDER BY id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []protocol.Host{}
	for _, id := range ids {
		h, e := s.GetHost(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, h)
	}
	return out, nil
}
func (s *Store) Drain(ctx context.Context, id string) (protocol.Host, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE hosts SET draining=1 WHERE id=? AND removed=0", id)
	if err != nil {
		return protocol.Host{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return protocol.Host{}, fault("not_found", "active host not found")
	}
	return s.GetHost(ctx, id)
}
func (s *Store) Remove(ctx context.Context, id string) (protocol.Host, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.Host{}, err
	}
	defer tx.Rollback()
	h, err := s.host(ctx, tx, id)
	if err != nil {
		return h, err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM leases WHERE host_id=? AND state!='RELEASED'", id).Scan(&count); err != nil {
		return h, err
	}
	if count > 0 {
		return h, fault("conflict", "host still owns unreleased or unverified leases")
	}
	if _, err = tx.ExecContext(ctx, "UPDATE hosts SET draining=1,removed=1 WHERE id=?", id); err != nil {
		return h, err
	}
	if err = tx.Commit(); err != nil {
		return h, err
	}
	return s.GetHost(ctx, id)
}
func (s *Store) Create(ctx context.Context, r protocol.CreateRequest) (protocol.Operation, error) {
	var zero protocol.Operation
	if !ValidID(r.OperationID) || r.Stack == "" || !ValidDigest(r.ManifestDigest) || !ValidDigest(r.PlanDigest) || len(r.Package) == 0 || len(r.Manifest) == 0 || r.AndroidSlots < 0 || r.AndroidSlots > 10000 {
		return zero, fault("invalid", "invalid create request")
	}
	for _, c := range r.RequiredCapabilities {
		if !ValidID(c) {
			return zero, fault("invalid", "invalid required capability")
		}
	}
	sources, sourceErr := SourceBlobs(r)
	if sourceErr != nil {
		return zero, sourceErr
	}
	payload, hash, err := canonical(r)
	if err != nil {
		return zero, fault("invalid", "invalid create payload")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if old, found, e := s.replay(ctx, tx, r.OperationID, hash); found || e != nil {
		return old, e
	}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM hosts WHERE removed=0 AND draining=0 ORDER BY id")
	if err != nil {
		return zero, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return zero, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return zero, err
	}
	var chosen protocol.Host
	for _, id := range ids {
		if r.HostID != "" && r.HostID != id {
			continue
		}
		h, e := s.host(ctx, tx, id)
		if e != nil {
			return zero, e
		}
		if !h.Online || !h.Compatible {
			continue
		}
		caps := map[string]bool{}
		for _, c := range h.Capabilities {
			caps[c] = true
		}
		compatible := true
		for _, c := range r.RequiredCapabilities {
			if !caps[c] {
				compatible = false
			}
		}
		if !compatible {
			continue
		}
		var used, slots int
		if e = tx.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(slots),0) FROM leases WHERE host_id=? AND state!='RELEASED'", id).Scan(&used, &slots); e != nil {
			return zero, e
		}
		if used >= h.Capacity.MaxLeases || slots+r.AndroidSlots > h.Capacity.AndroidSlots {
			continue
		}
		chosen = h
		break
	}
	if chosen.HostID == "" {
		return zero, fault("capacity", "no online compatible undrained host has capacity")
	}
	var options struct{ Owner string }
	if len(r.Options) > 0 {
		if err = json.Unmarshal(r.Options, &options); err != nil {
			return zero, fault("invalid", "invalid create options")
		}
	}
	if len(options.Owner) > 256 {
		return zero, fault("invalid", "owner exceeds metadata limit")
	}
	l := protocol.Lease{Owner: options.Owner, ID: ulid.Make().String(), ControllerID: s.ID, HostID: chosen.HostID, HostInstanceID: chosen.HostInstanceID, Epoch: 1, State: "ASSIGNED", LastKnownState: "ASSIGNED", AndroidSlots: r.AndroidSlots, ManifestDigest: r.ManifestDigest, PlanDigest: r.PlanDigest, SourceSetDigest: r.SourceSetDigest, RepositoryID: r.RepositoryID}
	data, _ := json.Marshal(l)
	if _, err = tx.ExecContext(ctx, "INSERT INTO leases(id,host_id,instance_id,epoch,state,slots,data) VALUES(?,?,?,?,?,?,?)", l.ID, l.HostID, l.HostInstanceID, l.Epoch, l.State, l.AndroidSlots, data); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operations(id,request_hash,lease_id,epoch,kind,state,payload,created) VALUES(?,?,?,1,'create','queued',?,?)", r.OperationID, hash, l.ID, payload, time.Now().UnixNano()); err != nil {
		return zero, err
	}
	for _, d := range sources {
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO blob_refs(digest,lease_id,kind) VALUES(?,?,'source')", d, l.ID); err != nil {
			return zero, err
		}
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return s.GetOperation(ctx, r.OperationID)
}
func (s *Store) replay(ctx context.Context, tx *sql.Tx, id, hash string) (protocol.Operation, bool, error) {
	var prior string
	err := tx.QueryRowContext(ctx, "SELECT request_hash FROM operations WHERE id=?", id).Scan(&prior)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, nil
	}
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if prior != hash {
		return protocol.Operation{}, true, fault("conflict", "operation ID already has a different payload")
	}
	op, e := s.operation(ctx, tx, id)
	return op, true, e
}

var AllowedKinds = map[string]bool{"show": true, "renew": true, "reconcile": true, "destroy": true, "test": true, "logs": true, "artifact": true, "ui": true, "ui-recover": true, "browser": true}

func (s *Store) Submit(ctx context.Context, r protocol.SubmitRequest) (protocol.Operation, error) {
	var zero protocol.Operation
	if !ValidID(r.OperationID) || !ValidID(r.LeaseID) || !AllowedKinds[r.Kind] || len(r.Payload) == 0 {
		return zero, fault("invalid", "invalid lease operation")
	}
	payload, hash, err := canonical(r)
	if err != nil {
		return zero, fault("invalid", "invalid operation payload")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if op, found, e := s.replay(ctx, tx, r.OperationID, hash); found || e != nil {
		return op, e
	}
	l, err := s.lease(ctx, tx, r.LeaseID, false)
	if err != nil {
		return zero, err
	}
	var active int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE lease_id=? AND state IN ('queued','dispatched')", r.LeaseID).Scan(&active); err != nil {
		return zero, err
	}
	if active > 0 {
		return zero, fault("conflict", "lease already has an active operation")
	}
	if l.State == "RELEASED" && r.Kind != "show" && r.Kind != "artifact" && r.Kind != "logs" && r.Kind != "destroy" && r.Kind != "reconcile" {
		return zero, fault("conflict", "released lease does not accept runtime effects")
	}
	// Only the typed operation payload is delivered; envelope identity remains separate.
	var envelope protocol.SubmitRequest
	if err = json.Unmarshal(payload, &envelope); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operations(id,request_hash,lease_id,epoch,kind,state,payload,created) VALUES(?,?,?,?,?,'queued',?,?)", r.OperationID, hash, l.ID, l.Epoch, r.Kind, envelope.Payload, time.Now().UnixNano()); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return s.GetOperation(ctx, r.OperationID)
}
func (s *Store) operation(ctx context.Context, q queryer, id string) (protocol.Operation, error) {
	var o protocol.Operation
	var result []byte
	err := q.QueryRowContext(ctx, "SELECT o.id,o.lease_id,l.host_id,l.instance_id,o.epoch,o.kind,o.state,o.payload,o.result FROM operations o JOIN leases l ON l.id=o.lease_id WHERE o.id=?", id).Scan(&o.ID, &o.LeaseID, &o.HostID, &o.HostInstanceID, &o.Epoch, &o.Kind, &o.State, &o.Payload, &result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return o, fault("not_found", "operation not found")
		}
		return o, err
	}
	o.ControllerID = s.ID
	if len(result) > 0 {
		var r protocol.Result
		if err = json.Unmarshal(result, &r); err != nil {
			return o, err
		}
		o.Result = &r
	}
	return o, nil
}
func (s *Store) GetOperation(ctx context.Context, id string) (protocol.Operation, error) {
	return s.operation(ctx, s.db, id)
}
func (s *Store) lease(ctx context.Context, q queryer, id string, observe bool) (protocol.Lease, error) {
	var l protocol.Lease
	var data []byte
	if err := q.QueryRowContext(ctx, "SELECT data,epoch,state FROM leases WHERE id=?", id).Scan(&data, &l.Epoch, &l.State); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return l, fault("not_found", "lease not found")
		}
		return l, err
	}
	epoch, state := l.Epoch, l.State
	if err := json.Unmarshal(data, &l); err != nil {
		return l, err
	}
	l.Epoch = epoch
	l.State = state
	l.LastKnownState = state
	if observe && state != "RELEASED" {
		h, err := s.host(ctx, q, l.HostID)
		if err != nil {
			return l, err
		}
		if !h.Online {
			l.State = "UNKNOWN"
		}
	}
	return l, nil
}
func (s *Store) GetLease(ctx context.Context, id string) (protocol.Lease, error) {
	return s.lease(ctx, s.db, id, true)
}
func (s *Store) ListLeases(ctx context.Context) ([]protocol.Lease, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM leases ORDER BY id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []protocol.Lease{}
	for _, id := range ids {
		l, e := s.GetLease(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, l)
	}
	return out, nil
}
func (s *Store) Poll(ctx context.Context, w protocol.WorkerIdentity) (*protocol.Operation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = s.worker(ctx, tx, w); err != nil {
		return nil, err
	}
	var id string
	err = tx.QueryRowContext(ctx, "SELECT o.id FROM operations o JOIN leases l ON l.id=o.lease_id WHERE l.host_id=? AND l.instance_id=? AND o.epoch=l.epoch AND o.state IN ('queued','dispatched') ORDER BY o.created LIMIT 1", w.HostID, w.HostInstanceID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE operations SET state='dispatched' WHERE id=? AND state='queued'", id); err != nil {
		return nil, err
	}
	op, err := s.operation(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &op, nil
}
func (s *Store) Complete(ctx context.Context, r protocol.Result) (protocol.Operation, error) {
	var zero protocol.Operation
	if r.State != "completed" && r.State != "failed" && r.State != "uncertain" {
		return zero, fault("invalid", "invalid operation result state")
	}
	if len(r.Payload) > 4<<20 || len(r.Artifacts) > 1024 {
		return zero, fault("invalid", "operation result exceeds metadata limit")
	}
	for _, a := range r.Artifacts {
		if !ValidDigest(a.Digest) || a.Size < 0 {
			return zero, fault("invalid", "invalid artifact descriptor")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if err = s.worker(ctx, tx, r.WorkerIdentity); err != nil {
		return zero, err
	}
	op, err := s.operation(ctx, tx, r.OperationID)
	if err != nil {
		return zero, err
	}
	l, err := s.lease(ctx, tx, op.LeaseID, false)
	if err != nil {
		return zero, err
	}
	if op.LeaseID != r.LeaseID || op.HostID != r.HostID || op.HostInstanceID != r.HostInstanceID || op.Epoch != r.Epoch || l.Epoch != r.Epoch {
		return zero, fault("identity", "result assignment or epoch mismatch")
	}
	compare := r
	compare.Incarnation = ""
	_, hash, err := canonical(compare)
	if err != nil {
		return zero, err
	}
	if op.Result != nil {
		old := *op.Result
		old.Incarnation = ""
		_, prior, e := canonical(old)
		if e != nil {
			return zero, e
		}
		if hash != prior {
			return zero, fault("conflict", "operation result conflicts with durable result")
		}
		return op, nil
	}
	if op.State != "dispatched" {
		return zero, fault("conflict", "operation was not dispatched")
	}
	state := l.State
	switch r.LocalState {
	case "":
	case "requested", "allocating", "starting", "ready", "degraded", "failed", "releasing", "released", "quarantined", "unknown":
		state = strings.ToUpper(r.LocalState)
	default:
		return zero, fault("invalid", "invalid local observed lease state")
	}
	if state == "RELEASED" && l.State != "RELEASED" {
		if !r.CleanupConfirmed || r.State == "uncertain" || (op.Kind != "destroy" && op.Kind != "reconcile") {
			return zero, fault("invalid", "release requires worker destroy/reconcile cleanup proof")
		}
	}
	if r.State == "uncertain" {
		state = "QUARANTINED"
	}
	data, err := json.Marshal(r)
	if err != nil {
		return zero, err
	}
	for _, a := range r.Artifacts {
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO blob_refs(digest,lease_id,kind) VALUES(?,?,'artifact')", a.Digest, r.LeaseID); err != nil {
			return zero, err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE operations SET state=?,result=?,result_hash=? WHERE id=?", r.State, data, hash, r.OperationID); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE leases SET state=? WHERE id=?", state, r.LeaseID); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return s.GetOperation(ctx, r.OperationID)
}
func (s *Store) String() string { return fmt.Sprintf("controller %s", s.ID) }

func SourceBlobs(r protocol.CreateRequest) ([]string, error) {
	var p struct {
		ManifestBlobDigest string `json:"manifest_blob_digest"`
		Sources            []struct {
			BlobDigest string `json:"blob_digest"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(r.Package, &p); err != nil {
		return nil, fault("invalid", "invalid source package")
	}
	if !ValidDigest(p.ManifestBlobDigest) || (r.ControlBlobDigest != "" && r.ControlBlobDigest != p.ManifestBlobDigest) {
		return nil, fault("invalid", "invalid control source digest")
	}
	out := []string{p.ManifestBlobDigest}
	for _, source := range p.Sources {
		if !ValidDigest(source.BlobDigest) {
			return nil, fault("invalid", "invalid source bundle digest")
		}
		out = append(out, source.BlobDigest)
	}
	return out, nil
}
func (s *Store) AuthorizeBlob(ctx context.Context, e protocol.Enrollment, digest, kind string) error {
	if e.Role == protocol.RoleClient {
		return nil
	}
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM blob_refs b JOIN leases l ON l.id=b.lease_id JOIN hosts h ON h.id=l.host_id WHERE b.digest=? AND b.kind=? AND l.host_id=? AND l.instance_id=h.instance_id AND h.removed=0", digest, kind, e.HostID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fault("forbidden", "blob is not assigned to this worker")
	}
	return nil
}

func (s *Store) Undrain(ctx context.Context, id string) (protocol.Host, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE hosts SET draining=0 WHERE id=? AND removed=0", id)
	if err != nil {
		return protocol.Host{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return protocol.Host{}, fault("not_found", "active host not found")
	}
	return s.GetHost(ctx, id)
}
