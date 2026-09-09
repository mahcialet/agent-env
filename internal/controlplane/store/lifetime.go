package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/oklog/ulid/v2"
)

func requestTTL(payload json.RawMessage) (time.Duration, error) {
	var request struct {
		TTL time.Duration `json:"ttl"`
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &request); err != nil {
			return 0, fault("invalid", "invalid lifetime request")
		}
	}
	if request.TTL < 0 {
		return 0, fault("invalid", "ttl must not be negative")
	}
	ttl, err := policy.Defaults().TTL(request.TTL)
	if err != nil {
		return 0, fault("invalid", err.Error())
	}
	return ttl, nil
}

// Backfill only from durable operation timestamps, never from restart time.
func (s *Store) backfillLifetimes(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT o.lease_id,o.kind,o.payload,o.created FROM operations o
 JOIN leases l ON l.id=o.lease_id LEFT JOIN lease_lifetimes f ON f.lease_id=l.id
 WHERE f.lease_id IS NULL AND (o.kind='create' OR (o.kind='renew' AND o.state='completed'))
 ORDER BY o.created,o.rowid`)
	if err != nil {
		return err
	}
	deadlines := map[string]int64{}
	for rows.Next() {
		var id, kind string
		var payload []byte
		var created int64
		if err = rows.Scan(&id, &kind, &payload, &created); err != nil {
			break
		}
		if kind == "create" {
			var request protocol.CreateRequest
			if err = json.Unmarshal(payload, &request); err != nil {
				break
			}
			payload = request.Options
		}
		var ttl time.Duration
		ttl, err = requestTTL(payload)
		if err != nil {
			// Older controllers persisted requests before worker TTL validation.
			// Do not make such a failed request prevent controller startup. An
			// invalid create receives the normal retention window from its
			// original timestamp; an invalid renewal cannot extend that window.
			err = nil
			if kind == "renew" {
				continue
			}
			ttl, _ = policy.Defaults().TTL(0)
		}
		deadlines[id] = time.Unix(0, created).Add(ttl).UnixNano()
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if rowErr != nil {
		return rowErr
	}
	for id, expiry := range deadlines {
		if _, err = tx.ExecContext(ctx, "INSERT INTO lease_lifetimes(lease_id,expires_at) VALUES(?,?)", id, expiry); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// QueueExpired records one ordinary, non-force destroy per expired lifetime.
// It never releases capacity or treats unavailable/uncertain resources as absent.
func (s *Store) QueueExpired(ctx context.Context, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.queueExpired(ctx, tx, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) queueExpired(ctx context.Context, tx *sql.Tx, now time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT l.id,l.epoch FROM lease_lifetimes f JOIN leases l ON l.id=f.lease_id
 WHERE f.expires_at<=? AND f.cleanup_operation_id='' AND l.state!='RELEASED'
 AND NOT EXISTS(SELECT 1 FROM operations o WHERE o.lease_id=l.id AND o.state IN ('queued','dispatched'))`, now.UnixNano())
	if err != nil {
		return err
	}
	type assignment struct {
		id    string
		epoch int64
	}
	var assignments []assignment
	for rows.Next() {
		var a assignment
		if err = rows.Scan(&a.id, &a.epoch); err != nil {
			rows.Close()
			return err
		}
		assignments = append(assignments, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range assignments {
		request := protocol.SubmitRequest{OperationID: "expiry-" + ulid.Make().String(), LeaseID: a.id, Kind: "destroy", Payload: json.RawMessage(`{}`)}
		_, hash, err := canonical(request)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO operations(id,request_hash,lease_id,epoch,kind,state,payload,created) VALUES(?,?,?,?,?,'queued',?,?)", request.OperationID, hash, a.id, a.epoch, request.Kind, []byte(request.Payload), now.UnixNano()); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE lease_lifetimes SET cleanup_operation_id=? WHERE lease_id=?", request.OperationID, a.id); err != nil {
			return err
		}
	}
	return nil
}
