package blobstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicationBarriersRejectUnconfirmedSuccess(t *testing.T) {
	for _, phase := range []string{"prepare", "confirm", "rename-race"} {
		t.Run(phase, func(t *testing.T) {
			s, err := New(filepath.Join(t.TempDir(), "cas"))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			data := []byte("durable publication fixture")
			digest := sum(data)
			injected := errors.New("injected publication barrier failure")
			native := s.publication
			prepareCalls, confirmCalls := 0, 0
			s.publication.prepare = func(dir string) error {
				prepareCalls++
				if phase == "prepare" {
					return injected
				}
				return native.prepare(dir)
			}
			s.publication.confirm = func(staged, destination, root string) error {
				confirmCalls++
				return injected
			}
			if phase == "rename-race" {
				s.publication.rename = func(staged, destination string) error {
					// Model a winner whose data was synced but whose namespace
					// publication is not yet confirmed; keep our staged file intact.
					if err := os.Mkdir(destination, 0700); err != nil {
						return err
					}
					f, err := os.OpenFile(filepath.Join(destination, "data"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if err != nil {
						return err
					}
					_, writeErr := f.Write(data)
					err = errors.Join(writeErr, f.Sync(), f.Close())
					if err != nil {
						return err
					}
					return injected
				}
			}
			if _, err := s.Put(ctx, bytes.NewReader(data), digest, 100); !errors.Is(err, injected) {
				t.Fatalf("barrier failure acknowledged: %v", err)
			}
			if prepareCalls != 1 {
				t.Fatal("staging directory barrier was skipped")
			}
			if phase == "prepare" {
				if _, err := os.Stat(filepath.Join(s.root, digest)); !os.IsNotExist(err) || confirmCalls != 0 {
					t.Fatal("publication proceeded past failed staging barrier")
				}
			} else {
				if confirmCalls != 1 {
					t.Fatal("visible object skipped publication confirmation")
				}
				// Retrying an already-visible object must run the barrier again.
				if _, err := s.Put(ctx, bytes.NewReader(data), digest, 100); !errors.Is(err, injected) || confirmCalls != 2 {
					t.Fatalf("dedup bypassed failed publication barrier: %v", err)
				}
			}
			s.publication = native
			if _, err := s.Put(ctx, bytes.NewReader(data), digest, 100); err != nil {
				t.Fatalf("native publication recovery failed: %v", err)
			}
			f, _, err := s.Open(ctx, digest, 100)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := io.ReadAll(f)
			f.Close()
			if err != nil || !bytes.Equal(actual, data) {
				t.Fatal("publication changed immutable bytes")
			}
			entries, err := os.ReadDir(s.root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".incoming-") {
					t.Fatal("failed publication retained staging directory")
				}
			}
		})
	}
}
