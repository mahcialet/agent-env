//go:build !windows

package worker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mahcialet/agent-env/internal/remotesource"
)

func TestLegacyRetainedDuplicateSyncsWinningInode(t *testing.T) {
	e := &AppExecutor{Home: t.TempDir()}
	id := "01K00000000000000000000000"
	p := remotesource.Package{Stack: "local"}
	if err := e.retainPackage(id, p); err != nil {
		t.Fatal(err)
	}
	data, err := e.loadPackage(id)
	if err != nil {
		t.Fatal(err)
	}
	retained := filepath.Join(e.Home, "remote-lease-inputs", id, "package.json")
	// Match the old worker's unsynced file write, without changing its bytes.
	if err := os.WriteFile(retained, data, 0600); err != nil {
		t.Fatal(err)
	}
	injected := errors.New("retained inode Sync failed")
	var synced string
	publication := retainedUnixPublication(func(path string) error {
		synced = path
		return injected
	})
	if err := e.retainPackageWithPublication(id, p, publication); !errors.Is(err, injected) {
		t.Fatalf("duplicate crossed failed retained-file barrier: %v", err)
	}
	if synced != retained {
		t.Fatalf("synced %q instead of winning retained inode %q", synced, retained)
	}
	if err := e.retainPackage(id, p); err != nil {
		t.Fatalf("native retained-file Sync recovery failed: %v", err)
	}
}
