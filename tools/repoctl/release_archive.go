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
		b, e := regularRead(p, maxReleaseMemberSize)
		if e != nil {
			return fmt.Errorf("invalid archive source %s: %w", name, e)
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
		if e = validateReleaseZIPRecords(path, z.File, end, stat.Size()); e != nil {
			return nil, e
		}
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

// Go's ZIP reader trusts the central directory for names and metadata. Validate
// local records too: streaming extractors may instead trust those records.
// Release archives use three ordinary (non-ZIP64) deflated files with signed
// data descriptors. Compare structure, not recompressed bytes, so validation
// remains independent of the toolchain's compression implementation.
func validateReleaseZIPRecords(path string, files []*zip.File, end []byte, size int64) (err error) {
	if len(files) != 3 || binary.LittleEndian.Uint16(end[4:6]) != 0 || binary.LittleEndian.Uint16(end[6:8]) != 0 || binary.LittleEndian.Uint16(end[8:10]) != 3 || binary.LittleEndian.Uint16(end[10:12]) != 3 {
		return fmt.Errorf("unexpected ZIP directory structure")
	}
	centralStart := int64(binary.LittleEndian.Uint32(end[16:20]))
	centralSize := int64(binary.LittleEndian.Uint32(end[12:16]))
	if centralStart+centralSize != size-22 {
		return fmt.Errorf("unexpected ZIP directory extent")
	}
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, source.Close()) }()
	readAt := func(offset int64, n int) ([]byte, error) {
		if offset < 0 || offset+int64(n) > size {
			return nil, fmt.Errorf("ZIP record outside archive")
		}
		b := make([]byte, n)
		_, e := source.ReadAt(b, offset)
		return b, e
	}
	centralOffset, localOffset := centralStart, int64(0)
	for _, file := range files {
		central, e := readAt(centralOffset, 46)
		if e != nil {
			return e
		}
		if !bytes.Equal(central[:4], []byte{'P', 'K', 1, 2}) || int64(binary.LittleEndian.Uint32(central[42:46])) != localOffset {
			return fmt.Errorf("unexpected ZIP local record offset")
		}
		nameLen := int(binary.LittleEndian.Uint16(central[28:30]))
		extraLen := int(binary.LittleEndian.Uint16(central[30:32]))
		commentLen := int(binary.LittleEndian.Uint16(central[32:34]))
		centralVariable, e := readAt(centralOffset+46, nameLen+extraLen+commentLen)
		if e != nil {
			return e
		}
		if nameLen != len(file.Name) || extraLen != len(file.Extra) || commentLen != 0 || string(centralVariable[:nameLen]) != file.Name || !bytes.Equal(centralVariable[nameLen:], file.Extra) {
			return fmt.Errorf("inconsistent ZIP central metadata")
		}
		local, e := readAt(localOffset, 30)
		if e != nil {
			return e
		}
		if !bytes.Equal(local[:4], []byte{'P', 'K', 3, 4}) || !bytes.Equal(local[4:14], central[6:16]) || binary.LittleEndian.Uint16(local[26:28]) != uint16(nameLen) || binary.LittleEndian.Uint16(local[28:30]) != uint16(extraLen) {
			return fmt.Errorf("ZIP local and central headers differ")
		}
		// Streaming headers carry sizes/CRC in the trailing descriptor, never here.
		if file.Flags & ^uint16(0x808) != 0 || file.Flags&8 == 0 || !bytes.Equal(local[14:26], make([]byte, 12)) {
			return fmt.Errorf("unexpected ZIP streaming metadata")
		}
		localVariable, e := readAt(localOffset+30, nameLen+extraLen)
		if e != nil {
			return e
		}
		if !bytes.Equal(localVariable, centralVariable) {
			return fmt.Errorf("ZIP local and central name or metadata differ")
		}
		if file.CompressedSize64 > uint64(maxReleaseMemberSize) {
			return fmt.Errorf("oversized ZIP compressed member")
		}
		descriptorOffset := localOffset + 30 + int64(nameLen+extraLen) + int64(file.CompressedSize64)
		descriptor, e := readAt(descriptorOffset, 16)
		if e != nil {
			return e
		}
		if !bytes.Equal(descriptor[:4], []byte{'P', 'K', 7, 8}) || binary.LittleEndian.Uint32(descriptor[4:8]) != file.CRC32 || uint64(binary.LittleEndian.Uint32(descriptor[8:12])) != file.CompressedSize64 || uint64(binary.LittleEndian.Uint32(descriptor[12:16])) != file.UncompressedSize64 {
			return fmt.Errorf("inconsistent ZIP data descriptor")
		}
		localOffset = descriptorOffset + 16
		centralOffset += 46 + int64(nameLen+extraLen+commentLen)
	}
	if localOffset != centralStart || centralOffset != size-22 {
		return fmt.Errorf("unexpected data between ZIP records")
	}
	return nil
}
