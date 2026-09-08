package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	sqliteDriver "modernc.org/sqlite"
	sqliteCodes "modernc.org/sqlite/lib"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConcurrentColdOpen(t *testing.T) {
	for round := 0; round < 8; round++ {
		path := filepath.Join(t.TempDir(), "cold.sqlite")
		start := make(chan struct{})
		results := make(chan error, 16)
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				s, err := Open(path)
				if err == nil {
					defer s.Close()
					err = s.Reserve(context.Background(), lease(fmt.Sprintf("lease-%d", i)), 0)
				}
				results <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestInitializationRetriesOnlyBoundedBusyContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.sqlite")
	u := url.URL{Scheme: "file", Path: uriPath(filepath.ToSlash(path))}
	u.RawQuery = url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(1)"}, "_txlock": {"immediate"}}.Encode()
	raw, err := sql.Open("sqlite", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	blocker, err := sql.Open("sqlite", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	if _, err := blocker.Exec("CREATE TABLE cold_start_probe(id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	tx, err := blocker.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	s := &Store{db: raw}
	// A held write transaction deterministically prevents promotion from DELETE
	// journal mode to WAL, before any migration can be applied.
	err = s.initialize(context.Background())
	var busy *sqliteDriver.Error
	if !errors.As(err, &busy) || busy.Code()&0xff != sqliteCodes.SQLITE_BUSY {
		t.Fatalf("expected real SQLite BUSY, got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if err := s.initializeWithRetry(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("retry did not respect deadline: %v", err)
	}
	// Use the production busy timeout before checking invariants on success.
	if _, err := raw.Exec("PRAGMA busy_timeout=10000"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		done <- s.initializeWithRetry(ctx)
	}()
	select {
	case err := <-done:
		t.Fatalf("initialization returned before lock release: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var count int
	if err := raw.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&count); err != nil || count != 4 {
		t.Fatalf("migrations %d: %v", count, err)
	}
}

func TestColdOpenProcessHelper(t *testing.T) {
	path := os.Getenv("AGENT_ENV_SQLITE_COLD_PATH")
	if path == "" {
		return
	}
	s, err := Open(path)
	if err == nil {
		err = s.Reserve(context.Background(), lease(os.Getenv("AGENT_ENV_SQLITE_COLD_ID")), 0)
		_ = s.Close()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestConcurrentColdOpenProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "process cold 日本語.sqlite")
	start := make(chan struct{})
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestColdOpenProcessHelper$")
			cmd.Env = append(os.Environ(), "AGENT_ENV_SQLITE_COLD_PATH="+path, fmt.Sprintf("AGENT_ENV_SQLITE_COLD_ID=process-%d", i), "GORACE=atexit_sleep_ms=0")
			<-start
			out, err := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("process %d: %w: %s", i, err, out)
			}
			results <- err
		}(i)
	}
	close(start)
	for i := 0; i < 8; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.List(context.Background())
	if err != nil || len(rows) != 8 {
		t.Fatalf("concurrent process records %d: %v", len(rows), err)
	}
}

func TestInitializationDoesNotRetryInvariantFailure(t *testing.T) {
	s, _ := database(t)
	if _, err := s.db.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := s.initializeWithRetry(ctx)
	if err == nil || ctx.Err() != nil {
		t.Fatalf("invariant failure was retried or ignored: %v", err)
	}
}
