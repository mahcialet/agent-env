package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The real file grows immediately after its opened-file Stat result is captured.
// This controls the filesystem interleaving without sleeps or scheduler luck.
type growingReleaseFile struct {
	*os.File
	afterStat func()
	readBytes int
}

func (f *growingReleaseFile) Stat() (os.FileInfo, error) {
	st, err := f.File.Stat()
	if err == nil {
		f.afterStat()
	}
	return st, err
}
func (f *growingReleaseFile) Read(p []byte) (int, error) {
	n, err := f.File.Read(p)
	f.readBytes += n
	return n, err
}
func TestReleaseReadBoundsGrowthAfterOpenedStat(t *testing.T) {
	for _, limit := range []int64{0, 1, 16} {
		t.Run(strconv.FormatInt(limit, 10), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "candidate")
			if err := os.WriteFile(path, make([]byte, limit), 0600); err != nil {
				t.Fatal(err)
			}
			f, err := os.OpenFile(path, os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			grew := false
			reader := &growingReleaseFile{File: f, afterStat: func() {
				grew = true
				if _, err := f.WriteAt(bytes.Repeat([]byte("x"), 65536), 0); err != nil {
					t.Fatal(err)
				}
			}}
			data, err := readOpenedReleaseFile(reader, limit)
			if !grew || err == nil || data != nil {
				t.Fatalf("growth not rejected: grew=%v bytes=%d err=%v", grew, len(data), err)
			}
			if int64(reader.readBytes) > limit+1 {
				t.Fatalf("reader consumed %d bytes past cap %d", reader.readBytes, limit)
			}
		})
	}
}
func TestReleaseRegularReadExactBounds(t *testing.T) {
	for _, limit := range []int64{0, 1, 16} {
		for _, size := range []int64{0, limit, limit + 1} {
			path := filepath.Join(t.TempDir(), "candidate")
			want := bytes.Repeat([]byte("x"), int(size))
			if err := os.WriteFile(path, want, 0600); err != nil {
				t.Fatal(err)
			}
			got, err := regularRead(path, limit)
			if size > limit {
				if err == nil {
					t.Fatalf("size %d exceeds limit %d", size, limit)
				}
			} else if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("size=%d limit=%d err=%v", size, limit, err)
			}
		}
	}
}
