// Package assets materializes agent-env-owned immutable bytes under state.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type AssetInfo struct {
	Name, Version, SHA256 string
	Size                  int64
}

func Describe(name, version string, data []byte) (AssetInfo, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." || version == "" {
		return AssetInfo{}, fmt.Errorf("invalid asset identity")
	}
	sum := sha256.Sum256(data)
	return AssetInfo{Name: name, Version: version, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))}, nil
}

// Materialize verifies or atomically creates a content-addressed asset. The
// returned path is never derived from caller-controlled path separators.
func Materialize(root string, info AssetInfo, data []byte) (string, error) {
	actual, err := Describe(info.Name, info.Version, data)
	if err != nil || actual.SHA256 != info.SHA256 || actual.Size != info.Size {
		return "", fmt.Errorf("asset provenance mismatch")
	}
	dir := filepath.Join(root, "assets", info.Name, info.SHA256)
	path := filepath.Join(dir, info.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return "", fmt.Errorf("asset path is not regular")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if string(got) != string(data) {
			return "", fmt.Errorf("materialized asset digest mismatch")
		}
		return path, nil
	}
	tmp, err := os.CreateTemp(dir, ".asset-")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		if st, statErr := os.Lstat(path); statErr == nil && st.Mode().IsRegular() {
			got, readErr := os.ReadFile(path)
			if readErr == nil && string(got) == string(data) {
				return path, nil
			}
		}
		return "", err
	}
	return path, nil
}
