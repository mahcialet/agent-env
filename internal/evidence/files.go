package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// AtomicWrite publishes a fully written file using a temporary sibling and a
// same-directory rename. The parent directory must already exist. A failed
// write or rename leaves any prior destination intact and removes the temporary.
func AtomicWrite(path string, data []byte, mode os.FileMode) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".agent-env-write-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = f.Close(); _ = os.Remove(name) }()
	if err = f.Chmod(mode); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return replaceFile(name, path)
}

func FileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
