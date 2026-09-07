package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

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
