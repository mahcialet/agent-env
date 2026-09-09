// Package instance protects one native service process per private state root.
package instance

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// Acquire holds a native SQLite exclusive file lock in a separate lock database.
// It needs no wall-clock lease, stale PID guess or lock-file deletion; the OS
// releases the underlying lock if the owner exits. Copied roots are not fenced.
func Acquire(path string) (func() error, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("instance lock requires an absolute path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("instance lock must be a regular file")
	}
	db, err := sql.Open("sqlite", filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(context.Background())
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = conn.ExecContext(context.Background(), "PRAGMA busy_timeout=0"); err == nil {
		_, err = conn.ExecContext(context.Background(), "BEGIN EXCLUSIVE")
	}
	if err != nil {
		conn.Close()
		db.Close()
		return nil, errors.New("another service owns this state root")
	}
	var once sync.Once
	var closeErr error
	return func() error {
		once.Do(func() {
			_, closeErr = conn.ExecContext(context.Background(), "ROLLBACK")
			closeErr = errors.Join(closeErr, conn.Close(), db.Close())
		})
		return closeErr
	}, nil
}
