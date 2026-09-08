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
	if _, err := Materialize(root, info, []byte("tampered")); err == nil {
		t.Fatal("tampered bytes accepted")
	}
}

func TestDescribeRejectsPathTraversal(t *testing.T) {
	if _, err := Describe("../asset", "1", []byte("x")); err == nil {
		t.Fatal("traversal accepted")
	}
}
