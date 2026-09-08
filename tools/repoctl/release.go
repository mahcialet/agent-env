package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var releaseTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

type releaseTarget struct{ GOOS, GOARCH string }

var releaseTargets = []releaseTarget{{"windows", "amd64"}, {"windows", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}

func gitOut(root string, args ...string) (string, error) {
	b, e := exec.Command("git", args...).Output()
	if e != nil {
		return "", e
	}
	return strings.TrimSpace(string(b)), nil
}
func releaseVersion(root, requested string) (string, time.Time, error) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(requested) {
		return "", time.Time{}, fmt.Errorf("invalid semver")
	}
	st, e := gitOut(root, "status", "--porcelain")
	if e != nil {
		return "", time.Time{}, e
	}
	if st != "" {
		return "", time.Time{}, fmt.Errorf("working tree is not clean")
	}
	head, e := gitOut(root, "rev-parse", "HEAD")
	if e != nil {
		return "", time.Time{}, e
	}
	tags, e := gitOut(root, "tag", "--points-at", head)
	if e != nil {
		return "", time.Time{}, e
	}
	var match string
	for _, t := range strings.Fields(tags) {
		if releaseTag.MatchString(t) && strings.TrimPrefix(t, "v") == requested {
			if match != "" {
				return "", time.Time{}, fmt.Errorf("multiple matching tags")
			}
			match = t
		}
	}
	if match == "" {
		return "", time.Time{}, fmt.Errorf("no matching release tag")
	}
	ts, e := gitOut(root, "show", "-s", "--format=%ct", match)
	if e != nil {
		return "", time.Time{}, e
	}
	var sec int64
	fmt.Sscan(ts, &sec)
	return match, time.Unix(sec, 0).UTC(), nil
}
func executeRelease(root string, args []string, out, errOut io.Writer) error {
	if args[0] == "release-check" {
		if len(args) != 2 {
			return fmt.Errorf("usage: release-check VERSION")
		}
		_, _, e := releaseVersion(root, args[1])
		return e
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: release-build VERSION")
	}
	ver := args[1]
	tag, mt, e := releaseVersion(root, ver)
	if e != nil {
		return e
	}
	outdir := filepath.Join(root, "dist", ver)
	os.RemoveAll(outdir)
	if e = os.MkdirAll(outdir, 0755); e != nil {
		return e
	}
	_ = tag
	for _, t := range releaseTargets {
		ext := ".tar.gz"
		if t.GOOS == "windows" {
			ext = ".zip"
		}
		name := fmt.Sprintf("agent-env_v%s_%s_%s", ver, t.GOOS, t.GOARCH)
		bin := filepath.Join(outdir, name, "agent-env")
		if t.GOOS == "windows" {
			bin += ".exe"
		}
		if e = os.MkdirAll(filepath.Dir(bin), 0755); e != nil {
			return e
		}
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", fmt.Sprintf("-s -w -X github.com/mahcialet/agent-env/internal/buildinfo.Version=%s", ver), "-o", bin, "./cmd/agent-env")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.GOOS, "GOARCH="+t.GOARCH)
		cmd.Stdout, cmd.Stderr = errOut, errOut
		if e = cmd.Run(); e != nil {
			return e
		}
		for _, f := range []string{"LICENSE", "README.md"} {
			src := filepath.Join(root, f)
			if b, x := os.ReadFile(src); x == nil {
				os.WriteFile(filepath.Join(filepath.Dir(bin), f), b, 0644)
			}
		}
		if e = writeArchive(filepath.Join(outdir, name+ext), filepath.Dir(bin), name, t.GOOS == "windows", mt); e != nil {
			return e
		}
	}
	return nil
}
func writeArchive(dst, dir, prefix string, zipMode bool, mt time.Time) error {
	files := []string{"agent-env", "agent-env.exe", "LICENSE", "README.md"}
	if zipMode {
		f, e := os.Create(dst)
		if e != nil {
			return e
		}
		defer f.Close()
		z := zip.NewWriter(f)
		defer z.Close()
		for _, n := range files {
			p := filepath.Join(dir, n)
			if b, e := os.ReadFile(p); e == nil {
				h := &zip.FileHeader{Name: prefix + "/" + n, Method: zip.Deflate, Modified: mt}
				w, e := z.CreateHeader(h)
				if e != nil {
					return e
				}
				if _, e = w.Write(b); e != nil {
					return e
				}
			}
		}
		return nil
	}
	f, e := os.Create(dst)
	if e != nil {
		return e
	}
	defer f.Close()
	g := gzip.NewWriter(f)
	defer g.Close()
	tw := tar.NewWriter(g)
	defer tw.Close()
	for _, n := range files {
		p := filepath.Join(dir, n)
		b, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		h := &tar.Header{Name: prefix + "/" + n, Mode: 0755, Size: int64(len(b)), ModTime: mt}
		if n == "LICENSE" || n == "README.md" {
			h.Mode = 0644
		}
		if e = tw.WriteHeader(h); e != nil {
			return e
		}
		if _, e = tw.Write(b); e != nil {
			return e
		}
	}
	return nil
}

var _ = sha256.Sum256
var _ = hex.EncodeToString
var _ = json.Marshal
var _ io.Reader
