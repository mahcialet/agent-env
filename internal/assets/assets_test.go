package assets

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/fixture.bin
var embeddedFixture []byte

func TestEmbeddedFixture(t *testing.T) {
	info, err := Describe("fixture.bin", "1", embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 17 || info.SHA256 != "f16f1ff497622c100ccc1cbce16afd88c63fe5338ab571a7e43caad90b655751" {
		t.Fatalf("unexpected embedded provenance: %+v", info)
	}
	path, err := Materialize(t.TempDir(), info, embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, embeddedFixture) {
		t.Fatalf("embedded materialization: %q, %v", got, err)
	}
}

func TestMaterializeAcrossProcesses(t *testing.T) {
	const rounds = 30
	if root := os.Getenv("AGENT_ENV_ASSET_TEST_ROOT"); root != "" {
		info, err := Describe("fixture.bin", "1", embeddedFixture)
		if err != nil {
			t.Fatal(err)
		}
		for round := range rounds {
			if _, err := io.ReadFull(os.Stdin, make([]byte, 1)); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				if _, err := Materialize(filepath.Join(root, fmt.Sprint(round)), info, embeddedFixture); err != nil {
					t.Fatal(err)
				}
			}
		}
		return
	}
	root := filepath.Join(t.TempDir(), "asset state 日本語")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	type worker struct {
		cmd    *exec.Cmd
		stdin  io.WriteCloser
		output bytes.Buffer
	}
	workers := make([]worker, 12)
	for i := range workers {
		w := &workers[i]
		w.cmd = exec.CommandContext(ctx, executable, "-test.run=^TestMaterializeAcrossProcesses$")
		w.cmd.Env = append(os.Environ(), "AGENT_ENV_ASSET_TEST_ROOT="+root)
		w.cmd.Stdout, w.cmd.Stderr = &w.output, &w.output
		w.stdin, err = w.cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := w.cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if w.cmd.ProcessState == nil {
				_ = w.cmd.Process.Kill()
				_ = w.cmd.Wait()
			}
		})
	}
	for range rounds {
		for i := range workers {
			if _, err := workers[i].stdin.Write([]byte{1}); err != nil {
				t.Error(err)
			}
		}
	}
	for i := range workers {
		w := &workers[i]
		w.stdin.Close()
		if err := w.cmd.Wait(); err != nil {
			t.Errorf("worker %d: %v\n%s", i, err, &w.output)
		}
	}
	info, _ := Describe("fixture.bin", "1", embeddedFixture)
	for round := range rounds {
		dir := filepath.Join(root, fmt.Sprint(round), "assets", info.Name, info.SHA256)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 1 || entries[0].Name() != info.Name {
			t.Errorf("round %d: unexpected output or leaked temporary files: %v, %v", round, entries, err)
			continue
		}
		got, err := os.ReadFile(filepath.Join(dir, info.Name))
		if err != nil || !bytes.Equal(got, embeddedFixture) {
			t.Errorf("round %d: incomplete publication %q: %v", round, got, err)
		}
	}
}

func TestMaterializeRejectsNonDirectoryAndSymlinkEntries(t *testing.T) {
	info, err := Describe("fixture.bin", "1", embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, position := range []string{"root", "assets", "name", "digest", "file"} {
		for _, kind := range []string{"regular", "symlink"} {
			if position == "file" && kind == "regular" {
				continue // Corrupted regular files are covered by the tamper test.
			}
			t.Run(position+"/"+kind, func(t *testing.T) {
				root := filepath.Join(t.TempDir(), "state")
				locations := map[string]string{
					"root":   root,
					"assets": filepath.Join(root, "assets"),
					"name":   filepath.Join(root, "assets", info.Name),
					"digest": filepath.Join(root, "assets", info.Name, info.SHA256),
					"file":   filepath.Join(root, "assets", info.Name, info.SHA256, info.Name),
				}
				path := locations[position]
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				outside := t.TempDir()
				if kind == "symlink" {
					if err := os.Symlink(outside, path); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				} else if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := Materialize(root, info, embeddedFixture); err == nil {
					t.Fatal("unsafe path accepted")
				}
				entries, err := os.ReadDir(outside)
				if err != nil || len(entries) != 0 {
					t.Fatalf("asset escaped state root: %v, %v", entries, err)
				}
			})
		}
	}
}

func TestMaterializeIsContentAddressedAndIdempotent(t *testing.T) {
	data := []byte("standalone asset")
	info, err := Describe("fixture.bin", "1", data)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := Materialize(root, info, data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Materialize(root, info, data)
	if err != nil || first != second {
		t.Fatalf("%s %s %v", first, second, err)
	}
	if filepath.Base(filepath.Dir(first)) != info.SHA256 {
		t.Fatal("asset is not content addressed")
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(root, info, data); err == nil {
		t.Fatal("corrupted materialized asset accepted")
	}
}

func TestMaterializeRejectsSymlinkedAncestorAndPortableNames(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "assets", "escape")); err != nil {
		t.Skip("symlink unavailable")
	}
	info, err := Describe("asset", "1", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	info.Name = "escape"
	if _, err := Materialize(root, info, []byte("x")); err == nil {
		t.Fatal("symlink ancestor accepted")
	}
	for _, name := range []string{`a\b`, `C:asset`} {
		if _, err := Describe(name, "1", []byte("x")); err == nil {
			t.Fatalf("portable name accepted: %q", name)
		}
	}
}

func TestDescribeRejectsPathTraversal(t *testing.T) {
	if _, err := Describe("../asset", "1", []byte("x")); err == nil {
		t.Fatal("traversal accepted")
	}
}

func TestDescribePortableLogicalNames(t *testing.T) {
	for _, name := range []string{"NUL", "nul.bin", "CON", "PRN.txt", "AUX", "COM1", "com9.apk", "LPT1", "LPT9.bin", "COM¹.txt", "LPT²", "COM³", "CONIN$", "CONOUT$", "NUL .bin", "trailing.", "trailing ", "a<b", "a>b", "a:b", "a\"b", "a|b", "a?b", "a*b", "a\x00b", "a\x1fb"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Describe(name, "1", []byte("x")); err == nil {
				t.Fatalf("nonportable asset name accepted: %q", name)
			}
		})
	}
	for _, name := range []string{"fixture.bin", "helper 日本語.apk", "com10.bin", "null", "console", "a.b"} {
		if _, err := Describe(name, "1", []byte("x")); err != nil {
			t.Errorf("portable asset name %q rejected: %v", name, err)
		}
	}
}

func TestMaterializeRejectsOversizedCacheBeforeReading(t *testing.T) {
	info, err := Describe("fixture.bin", "1", embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path, err := Materialize(root, info, embeddedFixture)
	if err != nil {
		t.Fatal(err)
	}
	const oversized = 32 << 20
	if err := os.Truncate(path, oversized); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(root, info, embeddedFixture); err == nil || !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("oversized cache must fail its size check before content reading: %v", err)
	}
	if st, err := os.Stat(path); err != nil || st.Size() != oversized {
		t.Fatalf("corrupt cache changed: %v, %v", st, err)
	}
}

// Both ordinary cache reuse and losing publication use this reader. Its size
// gate must reject corruption before either path allocates from the file size.
func TestReadAssetRejectsSizeMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "winner.bin")
	if err := os.WriteFile(path, embeddedFixture, 0600); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []int64{0, int64(len(embeddedFixture) - 1), int64(len(embeddedFixture) + 1)} {
		got, err := readAsset(path, expected)
		if err == nil || !strings.Contains(err.Error(), "size mismatch") || got != nil {
			t.Errorf("expected size %d returned %q, %v", expected, got, err)
		}
	}
	got, err := readAsset(path, int64(len(embeddedFixture)))
	if err != nil || !bytes.Equal(got, embeddedFixture) {
		t.Fatalf("valid publication winner: %q, %v", got, err)
	}
}
