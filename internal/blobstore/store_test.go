package blobstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func TestBoundsDigestAndDedup(t *testing.T) {
	ctx := context.Background()
	s, err := New(filepath.Join(t.TempDir(), "cas"))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 31, 32, 33} {
		data := bytes.Repeat([]byte("x"), n)
		b, err := s.Put(ctx, bytes.NewReader(data), sum(data), 32)
		if n > 32 {
			if err == nil {
				t.Fatal("oversized blob accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		f, got, err := s.Open(ctx, b.Digest, 32)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(f)
		f.Close()
		if got.Size != int64(n) || !bytes.Equal(raw, data) {
			t.Fatalf("roundtrip %d", n)
		}
	}
	other, err := New(s.root)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("concurrent immutable object")
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		store := s
		if i%2 == 0 {
			store = other
		}
		go func() {
			defer wg.Done()
			if _, err := store.Put(ctx, bytes.NewReader(data), sum(data), 32); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if _, err := s.Put(ctx, strings.NewReader("other"), sum(data), 32); err == nil {
		t.Fatal("digest mismatch accepted")
	}
	if _, _, err := s.Open(ctx, sum(data), 1); err == nil {
		t.Fatal("existing oversized blob accepted")
	}
	if err = os.WriteFile(filepath.Join(s.root, sum(data), "data"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.Open(ctx, sum(data), 32); err == nil {
		t.Fatal("corrupt stored blob accepted")
	}
	if _, err = s.Put(ctx, bytes.NewReader(data), sum(data), 32); err == nil {
		t.Fatal("corrupt dedup accepted")
	}
	entries, _ := os.ReadDir(s.root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".incoming") {
			t.Fatal("partial file retained")
		}
	}
}
func TestRejectUnsafePathsAndCancellation(t *testing.T) {
	ctx := context.Background()
	s, err := New(filepath.Join(t.TempDir(), "cas"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"../escape", strings.Repeat("A", 64), strings.Repeat("g", 64), ""} {
		if _, _, err := s.Open(ctx, d, 32); err == nil {
			t.Fatalf("unsafe digest %q", d)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = s.Put(canceled, strings.NewReader("x"), "", 32); err == nil {
		t.Fatal("cancellation ignored")
	}
	target := filepath.Join(t.TempDir(), "target")
	os.WriteFile(target, []byte("x"), 0600)
	if err = os.Symlink(target, filepath.Join(s.root, sum([]byte("x")))); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err = s.Open(ctx, sum([]byte("x")), 32); err == nil {
		t.Fatal("symlink blob accepted")
	}
	if _, err = s.Put(ctx, strings.NewReader("x"), "", 32); err == nil {
		t.Fatal("symlink dedup accepted")
	}
	rootLink := filepath.Join(t.TempDir(), "root-link")
	if err = os.Symlink(s.root, rootLink); err != nil {
		t.Fatal(err)
	}
	if _, err = New(rootLink); err == nil {
		t.Fatal("symlink root accepted")
	}
}
