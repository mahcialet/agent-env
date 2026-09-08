package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxReleaseMemberSize int64 = 256 << 20

func releaseArchiveNames(windows bool) []string {
	exe := "agent-env"
	if windows {
		exe += ".exe"
	}
	return []string{"LICENSE", "README.txt", exe}
}
func validArchiveTime(mt time.Time) error {
	if mt.Nanosecond() != 0 || mt.UTC().Year() < 1980 || mt.Unix() > int64(^uint32(0)) {
		return fmt.Errorf("archive timestamp must be whole UTC seconds from 1980 through 2106-02-07 06:28:15")
	}
	return nil
}
func validArchivePrefix(prefix string) bool {
	return prefix != "" && prefix != "." && prefix != ".." && !strings.ContainsAny(prefix, "/\\:\x00")
}
func releaseMemberMode(name string) os.FileMode {
	if name == "agent-env" || name == "agent-env.exe" {
		return 0755
	}
	return 0644
}
func writeArchive(dst, dir, prefix string, zipMode bool, mt time.Time) (err error) {
	if err = validArchiveTime(mt); err != nil {
		return err
	}
	if !validArchivePrefix(prefix) {
		return fmt.Errorf("invalid archive prefix")
	}
	names := releaseArchiveNames(zipMode)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != len(names) {
		return fmt.Errorf("unexpected archive staging files")
	}
	contents := map[string][]byte{}
	for _, name := range names {
		p := filepath.Join(dir, name)
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !st.Mode().IsRegular() || st.Size() > maxReleaseMemberSize {
			return fmt.Errorf("invalid archive source %s", name)
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		contents[name] = b
	}
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if zipMode {
		z := zip.NewWriter(f)
		defer func() { err = errors.Join(err, z.Close()) }()
		for _, name := range names {
			h := &zip.FileHeader{Name: prefix + "/" + name, Method: zip.Deflate, Modified: mt.UTC()}
			h.SetMode(releaseMemberMode(name))
			w, e := z.CreateHeader(h)
			if e != nil {
				return e
			}
			if _, e = w.Write(contents[name]); e != nil {
				return e
			}
		}
		return nil
	}
	g := gzip.NewWriter(f)
	defer func() { err = errors.Join(err, g.Close()) }()
	tw := tar.NewWriter(g)
	defer func() { err = errors.Join(err, tw.Close()) }()
	for _, name := range names {
		h := &tar.Header{Name: prefix + "/" + name, Mode: int64(releaseMemberMode(name)), Size: int64(len(contents[name])), ModTime: mt.UTC(), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}
		if e := tw.WriteHeader(h); e != nil {
			return e
		}
		if _, e := tw.Write(contents[name]); e != nil {
			return e
		}
	}
	return nil
}

func readReleaseArchive(path, prefix string, windows bool, mt time.Time) (result map[string][]byte, err error) {
	if err = validArchiveTime(mt); err != nil {
		return nil, err
	}
	if !validArchivePrefix(prefix) {
		return nil, fmt.Errorf("invalid archive prefix")
	}
	result = map[string][]byte{}
	expected := map[string]bool{}
	for _, name := range releaseArchiveNames(windows) {
		expected[prefix+"/"+name] = true
	}
	check := func(name string, mode os.FileMode, modified time.Time, size int64) error {
		if !expected[name] || result[strings.TrimPrefix(name, prefix+"/")] != nil {
			return fmt.Errorf("unexpected or duplicate archive member %q", name)
		}
		base := strings.TrimPrefix(name, prefix+"/")
		if !mode.IsRegular() || mode != releaseMemberMode(base) {
			return fmt.Errorf("invalid member mode: %s", name)
		}
		if !modified.Equal(mt) || size < 0 || size > maxReleaseMemberSize {
			return fmt.Errorf("invalid member timestamp or size: %s", name)
		}
		return nil
	}
	read := func(name string, r io.Reader) error {
		b, e := io.ReadAll(io.LimitReader(r, maxReleaseMemberSize+1))
		if e != nil {
			return e
		}
		if int64(len(b)) > maxReleaseMemberSize {
			return fmt.Errorf("oversized archive member")
		}
		result[strings.TrimPrefix(name, prefix+"/")] = b
		return nil
	}
	if windows {
		// Release ZIPs have no archive comment or appended bytes.
		endFile, e := os.Open(path)
		if e != nil {
			return nil, e
		}
		stat, e := endFile.Stat()
		if e != nil {
			return nil, errors.Join(e, endFile.Close())
		}
		end := make([]byte, 22)
		_, e = endFile.ReadAt(end, stat.Size()-22)
		e = errors.Join(e, endFile.Close())
		if e != nil {
			return nil, e
		}
		if !bytes.Equal(end[:4], []byte{'P', 'K', 5, 6}) || binary.LittleEndian.Uint16(end[20:22]) != 0 {
			return nil, fmt.Errorf("unexpected ZIP trailer")
		}
		z, e := zip.OpenReader(path)
		if e != nil {
			return nil, e
		}
		defer func() { err = errors.Join(err, z.Close()) }()
		if len(z.File) != 3 || z.Comment != "" {
			return nil, fmt.Errorf("invalid ZIP member count or comment")
		}
		for _, f := range z.File {
			if f.UncompressedSize64 > uint64(maxReleaseMemberSize) {
				return nil, fmt.Errorf("oversized ZIP member")
			}
			if e = check(f.Name, f.Mode(), f.Modified, int64(f.UncompressedSize64)); e != nil {
				return nil, e
			}
			if f.Comment != "" || f.Method != zip.Deflate || len(f.Extra) != 9 || binary.LittleEndian.Uint16(f.Extra[:2]) != 0x5455 || binary.LittleEndian.Uint16(f.Extra[2:4]) != 5 || f.Extra[4] != 1 || int64(binary.LittleEndian.Uint32(f.Extra[5:9])) != mt.Unix() {
				return nil, fmt.Errorf("unexpected ZIP metadata")
			}
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			e = errors.Join(read(f.Name, r), r.Close())
			if e != nil {
				return nil, e
			}
		}
	} else {
		f, e := os.Open(path)
		if e != nil {
			return nil, e
		}
		defer func() { err = errors.Join(err, f.Close()) }()
		buffered := bufio.NewReader(f)
		g, e := gzip.NewReader(buffered)
		if e != nil {
			return nil, e
		}
		defer func() { err = errors.Join(err, g.Close()) }()
		g.Multistream(false)
		if !g.ModTime.IsZero() || g.Name != "" || g.Comment != "" || len(g.Extra) != 0 || g.OS != 255 {
			return nil, fmt.Errorf("unexpected gzip metadata")
		}
		tr := tar.NewReader(g)
		for {
			h, e := tr.Next()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, e
			}
			if h.Typeflag != tar.TypeReg || h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" || h.Linkname != "" || !h.AccessTime.IsZero() || !h.ChangeTime.IsZero() || len(h.PAXRecords) != 0 || h.Format != tar.FormatUSTAR || h.Devmajor != 0 || h.Devminor != 0 {
				return nil, fmt.Errorf("unexpected tar metadata")
			}
			if e = check(h.Name, h.FileInfo().Mode(), h.ModTime, h.Size); e != nil {
				return nil, e
			}
			if e = read(h.Name, tr); e != nil {
				return nil, e
			}
		}
		// Drain through the gzip checksum and reject data after the tar end marker.
		tail, e := io.ReadAll(io.LimitReader(g, 1025))
		if e != nil {
			return nil, e
		}
		if _, trailingErr := buffered.Peek(1); trailingErr != io.EOF {
			return nil, fmt.Errorf("unexpected trailing compressed data: %v", trailingErr)
		}
		if len(tail) != 0 {
			return nil, fmt.Errorf("unexpected trailing archive data")
		}
	}
	if len(result) != 3 {
		return nil, fmt.Errorf("missing archive members")
	}
	return result, nil
}
