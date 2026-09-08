package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishAssetPreservesWindowsWinner(t *testing.T) {
	root := t.TempDir()
	source, destination := filepath.Join(root, "candidate"), filepath.Join(root, "published")
	if err := os.WriteFile(source, []byte("candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("winner"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishAsset(source, destination); err == nil {
		t.Fatal("published winner was replaced")
	}
	for path, want := range map[string]string{source: "candidate", destination: "winner"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v; want %q", path, got, err, want)
		}
	}
}
