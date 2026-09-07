package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteReplacementAndDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "descriptor 日本語 with spaces.json")
	for _, data := range []string{"initial", "replacement content"} {
		if err := AtomicWrite(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != data {
			t.Fatalf("%q %v", got, err)
		}
		digest, err := FileDigest(path)
		if err != nil {
			t.Fatal(err)
		}
		want := sha256.Sum256([]byte(data))
		if digest != hex.EncodeToString(want[:]) {
			t.Fatal("digest mismatch")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files remain: %v %v", entries, err)
	}
}

func TestAtomicWriteFailurePreservesExistingTree(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "existing directory")
	if err := os.Mkdir(dest, 0700); err != nil {
		t.Fatal(err)
	}
	precious := filepath.Join(dest, "keep")
	if err := os.WriteFile(precious, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(dest, []byte("replace"), 0600); err == nil {
		t.Fatal("replaced nonempty directory")
	}
	got, err := os.ReadFile(precious)
	if err != nil || string(got) != "original" {
		t.Fatalf("original lost: %q %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files remain: %v %v", entries, err)
	}
	if err := AtomicWrite(filepath.Join(dir, "missing parent", "file"), nil, 0600); err == nil {
		t.Fatal("unexpected missing parent success")
	}
	if _, err := FileDigest(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("unexpected missing file digest")
	}
	if _, err := FileDigest(dest); err == nil {
		t.Fatal("unexpected directory digest")
	}
}
