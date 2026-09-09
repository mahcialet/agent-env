package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExplicitControllerNeverFallsBackToLocal(t *testing.T) {
	for _, args := range [][]string{{"gc", "--apply"}, {"doctor"}, {"validate", "missing-repository"}, {"plan", "missing-repository"}, {"version"}} {
		t.Run(args[0], func(t *testing.T) {
			var out, errOut bytes.Buffer
			cmd := New(&out, &errOut)
			cmd.SetArgs(append([]string{"--controller", "https://127.0.0.1:1"}, args...))
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "unavailable with --controller") {
				t.Fatalf("unexpected fallback: %v %s", err, out.String())
			}
		})
	}
}
func TestRemoteRequiresExplicitTLSBeforeSubmission(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs([]string{"--controller", "https://127.0.0.1:1", "create", "missing-repository"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("missing explicit TLS accepted: %v", err)
	}
	if strings.Contains(errOut.String(), "Remote operation:") {
		t.Fatal("submission began before validating identity")
	}
}
func TestInvalidRemoteWaitRejectedBeforeSubmission(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs([]string{"--controller", "https://127.0.0.1:1", "--wait=-1s", "destroy", "lease"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--wait") {
		t.Fatalf("invalid wait not rejected: %v", err)
	}
	if strings.Contains(errOut.String(), "Remote operation:") {
		t.Fatal("invalid wait submitted mutation")
	}
}
