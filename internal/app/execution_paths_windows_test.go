package app

import (
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/paths"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsUnsupportedCreatePathRejectedBeforeReserve(t *testing.T) {
	s, options, _, _, operations := lifecycleFixture(t)
	s.Home = filepath.Join(t.TempDir(), strings.Repeat("x", 150), strings.Repeat("y", 100))
	boundary := false
	_, err := s.Create(context.Background(), options, CreateOptions{Owner: "fixture", BeforeReserve: func(context.Context) error { boundary = true; return nil }})
	if !errors.Is(err, paths.ErrExecutionDirectoryUnsupported) {
		t.Fatalf("unsupported path accepted: %v", err)
	}
	if boundary || len(*operations) != 0 {
		t.Fatalf("unsupported path crossed effect boundary: %v %v", boundary, *operations)
	}
	leases, e := s.Store.List(context.Background())
	if e != nil || len(leases) != 0 {
		t.Fatalf("unsupported path reserved lease: %+v %v", leases, e)
	}
	if _, e := os.Stat(s.Home); !errors.Is(e, os.ErrNotExist) {
		t.Fatalf("unsupported home created: %v", e)
	}
}
