package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseVerifyRejectsHiddenIndexEntries(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		for _, changed := range []bool{false, true} {
			t.Run(flag+map[bool]string{false: "/unchanged", true: "/modified"}[changed], func(t *testing.T) {
				root := sourceTestRepo(t) // Preview callers do not need a release tag.
				sourceTestGit(t, root, "update-index", flag, "tracked.txt")
				want := []byte("initial\n")
				if changed {
					want = []byte("hidden change\n")
					if err := os.WriteFile(filepath.Join(root, "tracked.txt"), want, 0644); err != nil {
						t.Fatal(err)
					}
				}
				if status := sourceTestGit(t, root, "status", "--porcelain"); status != "" {
					t.Fatalf("fixture is visibly dirty: %s", status)
				}
				indexPath := filepath.Join(root, ".git", "index")
				index, err := os.ReadFile(indexPath)
				if err != nil {
					t.Fatal(err)
				}
				refs := sourceTestGit(t, root, "show-ref")
				dest := filepath.Join(t.TempDir(), "candidate")
				var out, errOut bytes.Buffer
				err = executeArgs(root, []string{"release-verify", "--out", dest}, &out, &errOut)
				if err == nil || !strings.Contains(err.Error(), "index flags") {
					t.Errorf("expected unsupported index flags error, got %v", err)
				}
				if out.Len() != 0 || errOut.Len() != 0 {
					t.Errorf("verification reached build effects: stdout=%q stderr=%q", out.String(), errOut.String())
				}
				if _, err := os.Lstat(dest); !os.IsNotExist(err) {
					t.Errorf("output created or unreadable: %v", err)
				}
				after, err := os.ReadFile(indexPath)
				if err != nil || !bytes.Equal(index, after) {
					t.Errorf("caller index changed: %v", err)
				}
				data, err := os.ReadFile(filepath.Join(root, "tracked.txt"))
				if err != nil || !bytes.Equal(data, want) {
					t.Errorf("caller bytes changed: %v", err)
				}
				if after := sourceTestGit(t, root, "show-ref"); after != refs {
					t.Error("caller refs changed")
				}
			})
		}
	}
}
