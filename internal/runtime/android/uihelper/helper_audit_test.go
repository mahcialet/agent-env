package uihelper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestUIAuditUnsetHelperRefusesCurrentDirectory(t *testing.T) {
	d := t.TempDir()
	apk := []byte("fixture")
	m := Metadata{Version: Version, Package: Package, SourceSHA256: SourceDigest(), APKSHA256: fmt.Sprintf("%x", sha256.Sum256(apk))}
	b, _ := json.Marshal(m)
	if e := os.WriteFile(filepath.Join(d, "observer.apk"), apk, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(d, "observer.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	t.Chdir(d)
	if _, e := Load(""); e == nil {
		t.Error("unset explicit companion silently accepted cwd")
	}
}
