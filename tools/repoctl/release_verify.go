package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// executeReleaseVerify exercises the real strict-tag path using a private clone.
// It creates no tags in the user's repository and never publishes a release.
func executeReleaseVerify(root string, args []string, out, errOut io.Writer) error {
	if len(args) == 0 || (args[0] != "release-verify" && args[0] != "release-preview-smoke") {
		return fmt.Errorf("unknown release verification command")
	}
	preview := args[0] == "release-preview-smoke"
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(errOut)
	flagName := "out"
	if preview {
		flagName = "dir"
	}
	dir := fs.String(flagName, "", "release artifact directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 || *dir == "" {
		return fmt.Errorf("--%s directory required", flagName)
	}
	dest, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	if !preview {
		if _, err := os.Lstat(dest); !os.IsNotExist(err) {
			return fmt.Errorf("output must not exist: %s", dest)
		}
		status, err := gitOut(root, "status", "--porcelain=v1", "--untracked-files=all", "--ignore-submodules=none")
		if err != nil {
			return err
		}
		if status != "" {
			return fmt.Errorf("release verification requires a clean source working tree and index, including untracked files")
		}
	}
	commit, err := gitOut(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "agent-env-release-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	clone, cleanup, err := privateReleaseSource(root, commit, "v0.1.0")
	if err != nil {
		return err
	}
	defer cleanup()
	tag, mt, err := releaseVersion(clone, "0.1.0")
	if err != nil {
		return err
	}
	if preview {
		if err := smokeRelease(clone, dest, "0.1.0", tag, commit, mt, out); err != nil {
			return err
		}
		fmt.Fprintln(out, "release-preview-smoke passed at", commit)
		return nil
	}
	first, second := filepath.Join(workspace, "first"), filepath.Join(workspace, "second")
	for _, candidate := range []string{first, second} {
		if err := buildRelease(clone, candidate, "0.1.0", tag, commit, mt, out, errOut); err != nil {
			return err
		}
	}
	if err := compareReleaseDirectories(first, second); err != nil {
		return err
	}
	if err := smokeRelease(clone, first, "0.1.0", tag, commit, mt, out); err != nil {
		return err
	}
	current, err := gitOut(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	if current != commit {
		return fmt.Errorf("source HEAD changed during release verification")
	}
	if err := copyVerifiedRelease(first, dest); err != nil {
		return err
	}
	fmt.Fprintln(out, "release-verify passed: eight byte-identical files and native smoke at", commit)
	return nil
}

func compareReleaseDirectories(first, second string) error {
	a, err := os.ReadDir(first)
	if err != nil {
		return err
	}
	b, err := os.ReadDir(second)
	if err != nil {
		return err
	}
	if len(a) != 8 || len(b) != 8 {
		return fmt.Errorf("reproducibility comparison requires exactly eight files per build")
	}
	for i, entry := range a {
		if entry.Name() != b[i].Name() {
			return fmt.Errorf("reproducibility file set differs")
		}
		left, err := regularRead(filepath.Join(first, entry.Name()), 256<<20)
		if err != nil {
			return err
		}
		right, err := regularRead(filepath.Join(second, entry.Name()), 256<<20)
		if err != nil {
			return err
		}
		if !bytes.Equal(left, right) {
			return fmt.Errorf("reproducibility mismatch: %s", entry.Name())
		}
	}
	return nil
}

func copyVerifiedRelease(source, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	// Claim the destination with exclusive directory creation. Failed copies are
	// retained as visible evidence; never remove a caller-selected path.
	if err := os.Mkdir(dest, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		data, err := regularRead(filepath.Join(source, entry.Name()), 256<<20)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(filepath.Join(dest, entry.Name()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
