package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorMissingPrerequisitesReturnsStructuredFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs([]string{"doctor", "--output", "json"})
	err := cmd.Execute()
	if err == nil || ExitCode(err) != 3 {
		t.Fatalf("missing prerequisites not distinguished: %v", err)
	}
	var result struct {
		SchemaVersion int `json:"schema_version"`
		Data          struct {
			OK          bool     `json:"ok"`
			Diagnostics []string `json:"diagnostics"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != 1 || result.Data.OK || len(result.Data.Diagnostics) != 2 {
		t.Fatalf("misleading doctor: %s", out.String())
	}
}

func TestDefaultMineGroupsInvocationOwnersOnlyForSameUserHost(t *testing.T) {
	a := "user@host/01M1XW44BF319A898QJAXZRYNW"
	b := "user@host/01M1XW44C92YKVWEZRSMZJ83FD"
	if !ownerMatches(a, b, false) || ownerMatches(a, b, true) || ownerMatches("other@host/01M1XW44BF319A898QJAXZRYNW", b, false) {
		t.Fatal("advisory owner matching incorrect")
	}
}

func TestInitDoesNotOverwriteManifest(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "compose.yaml"), []byte("services:\n  api:\n    image: busybox:1.37\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		var out, errOut bytes.Buffer
		cmd := New(&out, &errOut)
		cmd.SetArgs([]string{"init", repo})
		err := cmd.Execute()
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 && (err == nil || !strings.Contains(err.Error(), "already exists")) {
			t.Fatalf("overwrote manifest: %v", err)
		}
	}
}
