package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseArchiveRoundTripDeterministic(t *testing.T) {
	for _, windows := range []bool{false, true} {
		t.Run(map[bool]string{false: "tar", true: "zip"}[windows], func(t *testing.T) {
			dir := t.TempDir()
			stage := filepath.Join(dir, "stage")
			if err := os.Mkdir(stage, 0755); err != nil {
				t.Fatal(err)
			}
			for _, name := range releaseArchiveNames(windows) {
				if err := os.WriteFile(filepath.Join(stage, name), []byte("contents:"+name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			mt := time.Date(2026, 9, 8, 1, 2, 3, 0, time.UTC)
			var previous []byte
			for _, output := range []string{"first", "second"} {
				p := filepath.Join(dir, output)
				if err := writeArchive(p, stage, "agent-env_v0.1.0_test", windows, mt); err != nil {
					t.Fatal(err)
				}
				got, err := readReleaseArchive(p, "agent-env_v0.1.0_test", windows, mt)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range releaseArchiveNames(windows) {
					if string(got[name]) != "contents:"+name {
						t.Fatalf("wrong content %s", name)
					}
				}
				current, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if previous != nil && !bytes.Equal(previous, current) {
					t.Fatal("archive bytes differ")
				}
				previous = current
			}
		})
	}
}

func TestReleaseArchiveRejectsUnsafeMembers(t *testing.T) {
	mt := time.Date(2026, 9, 8, 1, 2, 3, 0, time.UTC)
	for _, windows := range []bool{false, true} {
		for _, mutation := range []string{"traversal", "absolute", "symlink", "duplicate", "unexpected", "missing", "mode", "timestamp", "oversize", "corrupt", "owner", "trailing", "empty-gzip"} {
			t.Run(map[bool]string{false: "tar", true: "zip"}[windows]+"/"+mutation, func(t *testing.T) {
				var raw bytes.Buffer
				names := releaseArchiveNames(windows)
				if mutation == "missing" {
					names = names[:2]
				}
				var zw *zip.Writer
				var tw *tar.Writer
				var gz *gzip.Writer
				if windows {
					zw = zip.NewWriter(&raw)
				} else {
					gz = gzip.NewWriter(&raw)
					tw = tar.NewWriter(gz)
				}
				for i, base := range names {
					name := "prefix/" + base
					mode := releaseMemberMode(base)
					stamp := mt
					typ := byte(tar.TypeReg)
					owner := 0
					if i == 0 {
						switch mutation {
						case "traversal":
							name = "prefix/../LICENSE"
						case "absolute":
							name = "/prefix/LICENSE"
						case "symlink":
							mode |= os.ModeSymlink
							typ = tar.TypeSymlink
						case "duplicate":
							name = "prefix/" + names[1]
						case "unexpected":
							name = "prefix/extra"
						case "mode":
							mode = 0777
						case "timestamp":
							stamp = mt.Add(time.Second)
						case "owner":
							owner = 12
						}
					}
					if windows {
						h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: stamp}
						h.SetMode(mode)
						if mutation == "owner" && i == 0 {
							h.Comment = "host information"
						}
						w, err := zw.CreateHeader(h)
						if err != nil {
							t.Fatal(err)
						}
						if _, err = w.Write([]byte("x")); err != nil {
							t.Fatal(err)
						}
					} else {
						h := &tar.Header{Name: name, Mode: int64(mode.Perm()), Size: 1, ModTime: stamp, Typeflag: typ, Uid: owner}
						if typ == tar.TypeSymlink {
							h.Size = 0
							h.Linkname = "/tmp/elsewhere"
						}
						if mutation == "oversize" && i == 0 {
							h.Size = maxReleaseMemberSize + 1
						}
						if err := tw.WriteHeader(h); err != nil {
							t.Fatal(err)
						}
						if h.Size != 0 {
							if _, err := tw.Write([]byte("x")); err != nil {
								t.Fatal(err)
							}
						}
						if mutation == "oversize" {
							break
						}
					}
				}
				if windows {
					if err := zw.Close(); err != nil {
						t.Fatal(err)
					}
				} else {
					err := tw.Close()
					if err != nil && mutation != "oversize" {
						t.Fatal(err)
					}
					if err := gz.Close(); err != nil {
						t.Fatal(err)
					}
				}
				if mutation == "empty-gzip" && !windows {
					second := gzip.NewWriter(&raw)
					if err := second.Close(); err != nil {
						t.Fatal(err)
					}
				}
				data := raw.Bytes()
				if mutation == "trailing" {
					data = append(data, []byte("unexpected")...)
				}
				if windows && mutation == "empty-gzip" {
					return
				} // A gzip-specific fixture.

				if mutation == "corrupt" {
					data = data[:len(data)-8]
				}
				if windows && mutation == "oversize" {
					// Modify the first central directory entry's uncompressed size.
					at := bytes.Index(data, []byte{'P', 'K', 1, 2})
					if at < 0 {
						t.Fatal("no ZIP directory")
					}
					data[at+24] = 1
					data[at+25] = 0
					data[at+26] = 0
					data[at+27] = 0x10
				}
				p := filepath.Join(t.TempDir(), "archive")
				if err := os.WriteFile(p, data, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := readReleaseArchive(p, "prefix", windows, mt); err == nil {
					t.Fatal("invalid archive accepted")
				}
			})
		}
	}
}

func TestArchiveTimestampRange(t *testing.T) {
	for _, mt := range []time.Time{time.Date(1979, 12, 31, 0, 0, 0, 0, time.UTC), time.Unix(int64(^uint32(0))+1, 0), time.Unix(1800000000, 1)} {
		if err := validArchiveTime(mt); err == nil {
			t.Fatalf("accepted %v", mt)
		}
	}
	for _, mt := range []time.Time{time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), time.Unix(int64(^uint32(0)), 0)} {
		if err := validArchiveTime(mt); err != nil {
			t.Fatal(err)
		}
	}
}
