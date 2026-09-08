package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleasePublicationPreservesOutputAndCleansTransfer(t *testing.T) {
	stage := t.TempDir()
	parent := t.TempDir()
	dest := filepath.Join(parent, "candidate")
	for i := 0; i < 8; i++ {
		if err := os.WriteFile(filepath.Join(stage, string(rune('a'+i))), []byte("validated bytes"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := publishReleaseDirectory(stage, dest); err != nil {
		t.Fatal(err)
	}
	if err := compareReleaseDirectories(stage, dest); err != nil {
		t.Fatal(err)
	}
	if err := publishReleaseDirectory(stage, dest); err == nil {
		t.Fatal("existing output accepted")
	}
	if err := compareReleaseDirectories(stage, dest); err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	if err := os.Mkdir(filepath.Join(bad, "invalid-member"), 0755); err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join(parent, "failed")
	if err := publishReleaseDirectory(bad, absent); err == nil {
		t.Fatal("invalid copy succeeded")
	}
	if _, err := os.Lstat(absent); !os.IsNotExist(err) {
		t.Fatalf("partial output appeared: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "candidate" {
		t.Fatal("temporary publication directory leaked")
	}
}
