// Package assets materializes agent-env-owned immutable bytes under state.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Describe(name, version string, data []byte) (AssetInfo, error) {
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." || version == "" || strings.ContainsAny(name, `/\\:`) {
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
	if err := safeMkdirAll(root, dir); err != nil {
		return "", err
	}
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return "", fmt.Errorf("asset path is not regular")
		}
		got, err := readAsset(path)
		if err != nil {
			return "", err
		}
		if string(got) != string(data) {
			return "", fmt.Errorf("materialized asset digest mismatch")
		}
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
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
	if err = publishAsset(tmpPath, path); err != nil {
		if st, statErr := os.Lstat(path); statErr == nil && st.Mode().IsRegular() {
			got, readErr := readAsset(path)
			if readErr != nil {
				return "", readErr
			}
			if string(got) == string(data) {
				return path, nil
			}
		}
		return "", err
	}
	return path, nil
}

func safeMkdirAll(root, target string) error {
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("asset path escapes root")
	}
	cur := root
	if err := os.MkdirAll(cur, 0700); err != nil {
		return err
	}
	if st, err := os.Lstat(cur); err != nil {
		return err
	} else if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return fmt.Errorf("asset root is not a directory")
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		cur = filepath.Join(cur, part)
		st, statErr := os.Lstat(cur)
		if os.IsNotExist(statErr) {
			if err := os.Mkdir(cur, 0700); err != nil && !os.IsExist(err) {
				return err
			}
			// A concurrent materializer can create this component after Lstat.
			// Check the winner's entry rather than trusting EEXIST alone.
			st, statErr = os.Lstat(cur)
		}
		if statErr != nil {
			return statErr
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
			return fmt.Errorf("asset ancestor is not a directory")
		}
	}
	return nil
}
