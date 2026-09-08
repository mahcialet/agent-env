package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionReportsBundledInventoryWithoutState(t *testing.T) {
	state := filepath.Join(t.TempDir(), "unused state")
	t.Setenv("AGENT_ENV_HOME", state)
	t.Setenv("PATH", "")
	for _, format := range []string{"json", "table"} {
		var out, errOut bytes.Buffer
		cmd := New(&out, &errOut)
		cmd.SetArgs([]string{"version", "--output", format})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if format == "json" {
			var result struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if string(result.Data["assets"]) != "[]" {
				t.Fatalf("expected explicit empty production inventory: %s", out.String())
			}
		} else if !strings.Contains(out.String(), "Bundled assets: 0\n") {
			t.Fatalf("missing table inventory: %s", out.String())
		}
		if errOut.Len() != 0 {
			t.Fatal(errOut.String())
		}
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("version initialized state: %v", err)
	}
}

func TestVersionAndValidationJSON(t *testing.T) {
	for _, args := range [][]string{{"version", "--output", "json"}, {"validate", "../../testdata/manifests/api-dashboard.yaml", "--output", "json"}} {
		var out, errOut bytes.Buffer
		cmd := New(&out, &errOut)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		var envelope Envelope
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatalf("JSON polluted: %q: %v", out.String(), err)
		}
		if envelope.SchemaVersion != 1 || envelope.Data == nil || errOut.Len() != 0 {
			t.Fatalf("bad output: %s %s", out.String(), errOut.String())
		}
	}
}

func TestInvalidOutputFailsWithoutSuccess(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs([]string{"version", "--output", "xml"})
	if err := cmd.Execute(); err == nil || out.Len() != 0 {
		t.Fatalf("invalid output accepted: %v %s", err, out.String())
	}
}
