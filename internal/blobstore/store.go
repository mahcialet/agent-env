// Package blobstore retains immutable, digest-addressed transfer objects.
package blobstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const SourceLimit int64 = 1 << 30
const ArtifactLimit int64 = 64 << 20

type Blob struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type publicationHooks struct {
	prepare func(string) error
	rename  func(string, string) error
	confirm func(string, string, string) error
}
type Store struct {
	root        string
	publication publicationHooks
}

func ValidDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func privateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("blob directory is not a real directory")
	}
	return os.Chmod(path, 0700)
}
func New(root string) (*Store, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = privateDir(root); err != nil {
		return nil, err
	}
	return &Store{root: root, publication: nativePublication()}, nil
}
func (s *Store) path(digest string) (string, error) {
	if !ValidDigest(digest) {
		return "", errors.New("invalid SHA256 digest")
	}
	st, err := os.Lstat(s.root)
	if err != nil {
		return "", err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("unsafe blob directory")
	}
	return filepath.Join(s.root, digest), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
func checkLimit(limit int64) error {
	if limit < 0 || limit > SourceLimit {
		return errors.New("invalid blob size limit")
	}
	return nil
}
func (s *Store) Put(ctx context.Context, r io.Reader, expected string, limit int64) (result Blob, resultErr error) {
	var b Blob
	if err := checkLimit(limit); err != nil {
		return b, err
	}
	if expected != "" && !ValidDigest(expected) {
		return b, errors.New("invalid SHA256 digest")
	}
	if _, err := s.path(fmt.Sprintf("%064d", 0)); err != nil {
		return b, err
	}
	temp, err := os.MkdirTemp(s.root, ".incoming-")
	if err != nil {
		return b, err
	}
	defer os.RemoveAll(temp)
	f, err := os.OpenFile(filepath.Join(temp, "data"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return b, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(contextReader{ctx, r}, limit+1))
	if err != nil {
		return b, err
	}
	if n > limit {
		return b, errors.New("blob size limit exceeded")
	}
	if err = ctx.Err(); err != nil {
		return b, err
	}
	b = Blob{hex.EncodeToString(h.Sum(nil)), n}
	if expected != "" && b.Digest != expected {
		return Blob{}, errors.New("blob digest mismatch")
	}
	if err = f.Sync(); err != nil {
		return Blob{}, err
	}
	if err = f.Close(); err != nil {
		return Blob{}, err
	}
	if err = s.publication.prepare(temp); err != nil {
		return Blob{}, err
	}
	release, err := lockPublication(ctx, s.root)
	if err != nil {
		return Blob{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, release()) }()
	if err = ctx.Err(); err != nil {
		return Blob{}, err
	}
	dest, err := s.path(b.Digest)
	if err != nil {
		return Blob{}, err
	}
	if st, e := os.Lstat(dest); e == nil {
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return Blob{}, errors.New("unsafe existing blob")
		}
		old, _, e := s.Open(ctx, b.Digest, limit)
		if e != nil {
			return Blob{}, e
		}
		old.Close()
		return b, s.publication.confirm(temp, dest, s.root)
	} else if !os.IsNotExist(e) {
		return Blob{}, e
	}
	if err = s.publication.rename(temp, dest); err != nil {
		old, _, e := s.Open(ctx, b.Digest, limit)
		if e != nil {
			return Blob{}, err
		}
		old.Close()
		return b, s.publication.confirm(temp, dest, s.root)
	}
	return b, s.publication.confirm("", dest, s.root)
}

// Open verifies bytes before exposing an object, including already-present objects.
// The caller owns the returned file and must close it.
func (s *Store) Open(ctx context.Context, digest string, limit int64) (*os.File, Blob, error) {
	var b Blob
	if err := checkLimit(limit); err != nil {
		return nil, b, err
	}
	p, err := s.path(digest)
	if err != nil {
		return nil, b, err
	}
	directory, err := os.Lstat(p)
	if err != nil {
		return nil, b, err
	}
	if !directory.IsDir() || directory.Mode()&os.ModeSymlink != 0 {
		return nil, b, errors.New("blob object is not a real directory")
	}
	p = filepath.Join(p, "data")
	before, err := os.Lstat(p)
	if err != nil {
		return nil, b, err
	}
	if !before.Mode().IsRegular() {
		return nil, b, errors.New("blob is not a regular file")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, b, err
	}
	fail := func(e error) (*os.File, Blob, error) { f.Close(); return nil, Blob{}, e }
	st, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if !os.SameFile(before, st) {
		return fail(errors.New("blob identity changed"))
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(contextReader{ctx, f}, limit+1))
	if err != nil {
		return fail(err)
	}
	if n > limit {
		return fail(errors.New("blob size limit exceeded"))
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return fail(errors.New("blob digest mismatch"))
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return fail(err)
	}
	return f, Blob{digest, n}, nil
}
