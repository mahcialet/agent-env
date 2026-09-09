package worker

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/remotesource"
)

func TestRetainedPackageRequiresPublicationBarriers(t *testing.T) {
	for _, phase := range []string{"write", "prepare", "confirm"} {
		t.Run(phase, func(t *testing.T) {
			e := &AppExecutor{Home: t.TempDir()}
			p := remotesource.Package{Stack: "local"}
			id := "01K00000000000000000000000"
			injected := errors.New("injected retained package barrier failure")
			native := nativeRetainedPublication()
			faulty := native
			confirmCalls := 0
			switch phase {
			case "write":
				faulty.write = func(path string, data []byte, mode os.FileMode) error {
					if err := evidence.AtomicWrite(path, data, mode); err != nil {
						return err
					}
					return injected
				}
			case "prepare":
				faulty.prepare = func(string) error { return injected }
			case "confirm":
				faulty.confirm = func(string, string, string, []byte) error { confirmCalls++; return injected }
			}
			if err := e.retainPackageWithPublication(id, p, faulty); !errors.Is(err, injected) {
				t.Fatalf("%s failure crossed retention barrier: %v", phase, err)
			}
			if phase != "confirm" {
				if _, err := e.loadPackage(id); err == nil {
					t.Fatal("failed staging exposed final package")
				}
			} else {
				if _, err := e.loadPackage(id); err != nil {
					t.Fatalf("expected visible but unconfirmed package: %v", err)
				}
				if err := e.retainPackageWithPublication(id, p, faulty); !errors.Is(err, injected) || confirmCalls != 2 {
					t.Fatalf("duplicate bypassed retention confirmation: %v", err)
				}
			}
			if err := e.retainPackage(id, p); err != nil {
				t.Fatalf("durable retry failed: %v", err)
			}
			before, err := e.loadPackage(id)
			if err != nil {
				t.Fatal(err)
			}
			if err := e.retainPackage(id, p); err != nil {
				t.Fatalf("durable duplicate failed: %v", err)
			}
			after, err := e.loadPackage(id)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("duplicate changed immutable package bytes")
			}
			entries, err := os.ReadDir(filepath.Join(e.Home, "remote-lease-inputs"))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".incoming-") {
					t.Fatal("retention failure leaked staging directory")
				}
			}
		})
	}
}

func TestRetainedPackageRefusesConflictingOrCorruptDuplicate(t *testing.T) {
	e := &AppExecutor{Home: t.TempDir()}
	id := "01K00000000000000000000000"
	p := remotesource.Package{Stack: "local"}
	if err := e.retainPackage(id, p); err != nil {
		t.Fatal(err)
	}
	original, err := e.loadPackage(id)
	if err != nil {
		t.Fatal(err)
	}
	p.Stack = "different"
	if err := e.retainPackage(id, p); err == nil {
		t.Fatal("changed package replaced retained cleanup authority")
	}
	current, err := e.loadPackage(id)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatal("conflicting package changed retained data")
	}
	path := filepath.Join(e.Home, "remote-lease-inputs", id, "package.json")
	corrupt := []byte("corrupt retained package")
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	p.Stack = "local"
	if err := e.retainPackage(id, p); err == nil {
		t.Fatal("corrupt package silently repaired")
	}
	current, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(current, corrupt) {
		t.Fatal("corruption evidence overwritten")
	}
}
