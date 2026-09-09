// Package uihelper owns the optional Android observer companion and its provenance.
package uihelper

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Version = 1
const Package = "dev.agentenv.observer"
const Runner = Package + "/" + Package + ".Observer"
const MaxAPKBytes = 16 << 20

//go:embed source/Observer.java source/AndroidManifest.xml
var source embed.FS

type Metadata struct {
	APKPath      string `json:"-"`
	Version      int    `json:"version"`
	Package      string `json:"package"`
	SourceSHA256 string `json:"source_sha256"`
	APKSHA256    string `json:"apk_sha256"`
	Platform     string `json:"platform"`
	BuildTools   string `json:"build_tools"`
}

func SourceDigest() string {
	h := sha256.New()
	for _, name := range []string{"source/AndroidManifest.xml", "source/Observer.java"} {
		b, _ := source.ReadFile(name)
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func regular(path string, limit int64) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > limit || st.Size() == 0 {
		return nil, fmt.Errorf("invalid companion file")
	}
	return os.ReadFile(path)
}

// Load confines companion reads to a verified explicit directory and refuses symlinks.
func Load(dir string) (Metadata, error) {
	var m Metadata
	if strings.TrimSpace(dir) == "" {
		return m, fmt.Errorf("explicit companion directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return m, err
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return m, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return m, fmt.Errorf("invalid companion directory")
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return m, err
	}
	defer root.Close()
	openedDir, err := root.Stat(".")
	if err != nil {
		return m, err
	}
	if !os.SameFile(st, openedDir) {
		return m, fmt.Errorf("companion directory changed")
	}
	read := func(name string, limit int64) ([]byte, error) {
		before, e := root.Lstat(name)
		if e != nil {
			return nil, e
		}
		if !before.Mode().IsRegular() || before.Size() == 0 || before.Size() > limit {
			return nil, fmt.Errorf("invalid companion file")
		}
		f, e := root.Open(name)
		if e != nil {
			return nil, e
		}
		defer f.Close()
		opened, e := f.Stat()
		if e != nil {
			return nil, e
		}
		if !os.SameFile(before, opened) {
			return nil, fmt.Errorf("companion file changed")
		}
		b, e := io.ReadAll(io.LimitReader(f, limit+1))
		if e != nil {
			return nil, e
		}
		if int64(len(b)) > limit {
			return nil, fmt.Errorf("companion file oversized")
		}
		return b, nil
	}
	b, err := read("observer.json", 16384)
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	if m.Version != Version || m.Package != Package || m.SourceSHA256 != SourceDigest() {
		return m, fmt.Errorf("companion source/version mismatch")
	}
	b, err = read("observer.apk", MaxAPKBytes)
	if err != nil {
		return m, err
	}
	sum := sha256.Sum256(b)
	if m.APKSHA256 != hex.EncodeToString(sum[:]) {
		return m, fmt.Errorf("companion APK digest mismatch")
	}
	m.APKPath = filepath.Join(abs, "observer.apk")
	return m, nil
}
