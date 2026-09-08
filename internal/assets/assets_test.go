package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeIsContentAddressedAndIdempotent(t *testing.T) {
	data := []byte("standalone asset")
	info, err := Describe("fixture.bin", "1", data)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := Materialize(root, info, data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Materialize(root, info, data)
	if err != nil || first != second {
		t.Fatalf("%s %s %v", first, second, err)
	}
	if filepath.Base(filepath.Dir(first)) != info.SHA256 {
		t.Fatal("asset is not content addressed")
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(root, info, data); err == nil {
		t.Fatal("corrupted materialized asset accepted")
	}
}

func TestMaterializeRejectsSymlinkedAncestorAndPortableNames(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "assets", "escape")); err != nil {
		t.Skip("symlink unavailable")
	}
	info, err := Describe("asset", "1", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	info.Name = "escape"
	if _, err := Materialize(root, info, []byte("x")); err == nil {
		t.Fatal("symlink ancestor accepted")
	}
	for _, name := range []string{`a\b`, `C:asset`} {
		if _, err := Describe(name, "1", []byte("x")); err == nil {
			t.Fatalf("portable name accepted: %q", name)
		}
	}
}

func TestDescribeRejectsPathTraversal(t *testing.T) {
	if _, err := Describe("../asset", "1", []byte("x")); err == nil {
		t.Fatal("traversal accepted")
	}
}
