package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reuse real six-target candidates built by release-verify; ordinary tests never
// create public tags or silently substitute fixture executables for release bytes.
func TestReleaseCandidate(t *testing.T) {
	candidate := os.Getenv("AGENT_ENV_RELEASE_CANDIDATE")
	if candidate == "" {
		t.Skip("set AGENT_ENV_RELEASE_CANDIDATE to release-verify output or build for end-to-end negative tests")
	}
	root, e := repositoryRoot()
	if e != nil {
		t.Fatal(e)
	}
	commit, e := gitOut(root, "rev-parse", "HEAD")
	if e != nil {
		t.Fatal(e)
	}
	source, cleanup, e := privateReleaseSource(root, commit, "v0.1.0")
	if e != nil {
		t.Fatal(e)
	}
	defer cleanup()
	tag, mt, e := releaseVersion(source, "0.1.0")
	if e != nil {
		t.Fatal(e)
	}
	if candidate == "build" {
		candidate = filepath.Join(t.TempDir(), "candidate")
		if e = buildRelease(source, candidate, "0.1.0", tag, commit, mt, io.Discard, io.Discard); e != nil {
			t.Fatal(e)
		}
	}
	original, e := checkRelease(source, candidate, "0.1.0", tag, commit, mt)
	if e != nil {
		t.Fatal(e)
	}
	if e = smokeRelease(source, candidate, "0.1.0", tag, commit, mt, io.Discard); e != nil {
		t.Fatal(e)
	}
	sandbox := filepath.Join(t.TempDir(), "candidate")
	if e = copyVerifiedRelease(candidate, sandbox); e != nil {
		t.Fatal(e)
	}
	mutate := func(name string, fn func(*releaseManifest)) {
		t.Run(name, func(t *testing.T) {
			m := original
			m.Artifacts = append([]releaseArtifact(nil), original.Artifacts...)
			fn(&m)
			b, e := json.MarshalIndent(m, "", "  ")
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(sandbox, "release-manifest.json")
			saved, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			defer os.WriteFile(path, saved, 0644)
			if e = os.WriteFile(path, append(b, '\n'), 0644); e != nil {
				t.Fatal(e)
			}
			if _, e = checkRelease(source, sandbox, "0.1.0", tag, commit, mt); e == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
	mutate("version", func(m *releaseManifest) { m.Version = "0.1.1" })
	mutate("commit", func(m *releaseManifest) { m.Commit = strings.Repeat("0", 40) })
	mutate("timestamp", func(m *releaseManifest) { m.SourceTimestamp++ })
	mutate("toolchain", func(m *releaseManifest) { m.GoVersion = "go1.0" })
	mutate("target", func(m *releaseManifest) { m.Artifacts[0].GOARCH = "arm64" })
	mutate("path", func(m *releaseManifest) { m.Artifacts[0].Executable = "../agent-env.exe" })
	mutate("archive_digest", func(m *releaseManifest) { m.Artifacts[0].SHA256 = strings.Repeat("0", 64) })
	mutate("binary_digest", func(m *releaseManifest) { m.Artifacts[0].ExecutableSHA256 = strings.Repeat("0", 64) })
	mutate("build_identity", func(m *releaseManifest) { m.Artifacts[0].BuildInfo.Version = "0.1.1" })
	mutate("assets", func(m *releaseManifest) {
		m.Artifacts[0].Assets = []releaseAsset{{Name: "invented", SHA256: strings.Repeat("0", 64)}}
	})
	for _, name := range []string{"checksums.txt", original.Artifacts[0].Archive} {
		t.Run("corrupt_"+name, func(t *testing.T) {
			p := filepath.Join(sandbox, name)
			saved, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			defer os.WriteFile(p, saved, 0644)
			if e = os.WriteFile(p, []byte("corrupt"), 0644); e != nil {
				t.Fatal(e)
			}
			if _, e = checkRelease(source, sandbox, "0.1.0", tag, commit, mt); e == nil {
				t.Fatal("corruption accepted")
			}
		})
	}
	// Recompute both checksums after binary tampering: semantic checks must still fail.
	for _, which := range []string{"version_record", "local_path", "license", "readme"} {
		t.Run(which, func(t *testing.T) {
			m := original
			m.Artifacts = append([]releaseArtifact(nil), original.Artifacts...)
			a := &m.Artifacts[0]
			prefix, _, exe := archiveNames("0.1.0", a.releaseTarget)
			files, e := readReleaseArchive(filepath.Join(candidate, a.Archive), prefix, true, mt)
			if e != nil {
				t.Fatal(e)
			}
			switch which {
			case "version_record":
				files[exe] = bytes.Replace(files[exe], []byte("agent-env-release-v1|0.1.0|"), []byte("agent-env-release-v1|0.1.1|"), 1)
			case "local_path":
				files[exe] = append(files[exe], []byte(source+string(filepath.Separator)+"private.go")...)
			case "license":
				files["LICENSE"] = []byte("wrong")
			case "readme":
				files["README.txt"] = []byte("wrong")
			}
			dataDir := t.TempDir()
			for n, b := range files {
				if e = os.WriteFile(filepath.Join(dataDir, n), b, 0644); e != nil {
					t.Fatal(e)
				}
			}
			packed := filepath.Join(t.TempDir(), a.Archive)
			if e = writeArchive(packed, dataDir, prefix, true, mt); e != nil {
				t.Fatal(e)
			}
			ar, e := os.ReadFile(packed)
			if e != nil {
				t.Fatal(e)
			}
			a.SHA256 = digest(ar)
			a.ExecutableSHA256 = digest(files[exe])
			mb, e := json.MarshalIndent(m, "", "  ")
			if e != nil {
				t.Fatal(e)
			}
			for n, b := range map[string][]byte{a.Archive: ar, "release-manifest.json": append(mb, '\n'), "checksums.txt": releaseChecksums(m)} {
				p := filepath.Join(sandbox, n)
				saved, e := os.ReadFile(p)
				if e != nil {
					t.Fatal(e)
				}
				defer os.WriteFile(p, saved, 0644)
				if e = os.WriteFile(p, b, 0644); e != nil {
					t.Fatal(e)
				}
			}
			if _, e = checkRelease(source, sandbox, "0.1.0", tag, commit, mt); e == nil {
				t.Fatal("rehashed invalid artifact accepted")
			}
		})
	}
	t.Run("existing_output_preserved", func(t *testing.T) {
		before, e := os.ReadFile(filepath.Join(sandbox, "checksums.txt"))
		if e != nil {
			t.Fatal(e)
		}
		if e = buildRelease(source, sandbox, "0.1.0", tag, commit, mt, io.Discard, io.Discard); e == nil {
			t.Fatal("existing output accepted")
		}
		after, e := os.ReadFile(filepath.Join(sandbox, "checksums.txt"))
		if e != nil || !bytes.Equal(before, after) {
			t.Fatal("caller output altered")
		}
	})
	t.Run("nonignored_worktree_output", func(t *testing.T) {
		output := filepath.Join(source, "release candidates 日本語", "nested", "v0.1.0")
		if _, _, err := releaseVersion(source, "0.1.0"); err != nil {
			t.Fatal(err)
		}
		if err := buildRelease(source, output, "0.1.0", tag, commit, mt, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		if err := compareReleaseDirectories(candidate, output); err != nil {
			t.Fatal(err)
		}
		if _, err := checkRelease(source, output, "0.1.0", tag, commit, mt); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(filepath.Dir(output))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "v0.1.0" {
			t.Fatal("owned staging leaked beside output")
		}
	})

}

func TestReleaseCommandUsage(t *testing.T) {
	for _, args := range [][]string{{"release-build"}, {"release-check", "0.1.0"}, {"release-build", "--version", "0.1.0", "--tag", "v0.1.0", "--out", "ignored"}, {"release-build", "--version", "0.1.0", "--out", "ignored", "extra"}} {
		if e := executeRelease(t.TempDir(), args, io.Discard, io.Discard); e == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
