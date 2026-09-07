package sqlite

import (
	"context"
	"errors"
	"time"
)

// RequestRunCancel is deliberately outside the lease mutation fence: it can
// request cooperation from the exact run holding that fence, but cannot alter
// the lease, its evidence, or the running operation's ownership token.
func (s *Store) RequestRunCancel(ctx context.Context, runID string) error {
	if runID == "" {
		return errors.New("command run ID is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, "SELECT status FROM command_runs WHERE id=?", runID).Scan(&status); err != nil {
		return err
	}
	if status == "running" {
		if _, err = tx.ExecContext(ctx, "INSERT INTO command_run_cancellations(run_id,requested_at) VALUES(?,?) ON CONFLICT(run_id) DO NOTHING", runID, stamp(time.Now())); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RunCancellationRequested(ctx context.Context, runID string) (bool, error) {
	var requested bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM command_run_cancellations WHERE run_id=?)", runID).Scan(&requested)
	return requested, err
}
