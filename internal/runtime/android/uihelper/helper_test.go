package uihelper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyRejectsMismatchedProvenance(t *testing.T) {
	dir := t.TempDir()
	apk := filepath.Join(dir, "observer.apk")
	meta := filepath.Join(dir, "observer.json")
	b := []byte("test artifact")
	sum := sha256.Sum256(b)
	m := Metadata{Version: Version, Package: Package, SourceSHA256: SourceDigest(), APKSHA256: hex.EncodeToString(sum[:])}
	write := func() {
		t.Helper()
		encoded, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(meta, encoded, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.WriteFile(apk, b, 0600); e != nil {
		t.Fatal(e)
	}
	write()
	if _, e := Load(dir); e != nil {
		t.Fatal(e)
	}
	m.SourceSHA256 = "foreign"
	write()
	if _, e := Load(dir); e == nil {
		t.Fatal("foreign source accepted")
	}
	m.SourceSHA256 = SourceDigest()
	m.Version++
	write()
	if _, e := Load(dir); e == nil {
		t.Fatal("foreign version accepted")
	}
	m.Version = Version
	m.Package = "foreign"
	write()
	if _, e := Load(dir); e == nil {
		t.Fatal("foreign package accepted")
	}
	m.Package = Package
	write()
	if e := os.WriteFile(apk, []byte("tampered"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(dir); e == nil {
		t.Fatal("tampered APK accepted")
	}
	if _, e := Load(apk); e == nil {
		t.Fatal("directory accepted as APK")
	}
}
func TestBuildRejectsMissingAndEscapingInputs(t *testing.T) {
	for _, o := range []BuildOptions{{}, {SDK: t.TempDir(), JDK: t.TempDir(), Output: filepath.Join(t.TempDir(), "out"), Platform: "../android-35", BuildTools: "36.0.0"}, {SDK: t.TempDir(), JDK: t.TempDir(), Output: filepath.Join(t.TempDir(), "out"), Platform: "android-35", BuildTools: "../36.0.0"}} {
		if _, e := Build(context.Background(), o); e == nil {
			t.Fatal("invalid options accepted")
		}
	}
}
func TestEmbeddedContract(t *testing.T) {
	if len(SourceDigest()) != 64 {
		t.Fatal("invalid source digest")
	}
	for _, p := range []string{"source/AndroidManifest.xml", "source/Observer.java"} {
		if b, e := source.ReadFile(p); e != nil || len(b) == 0 {
			t.Fatalf("missing embedded source %s: %v", p, e)
		}
	}
}

func TestLoadRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink privilege unavailable: %v", err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlink directory accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "metadata"), filepath.Join(target, "observer.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(target); err == nil {
		t.Fatal("symlink metadata accepted")
	}
}
