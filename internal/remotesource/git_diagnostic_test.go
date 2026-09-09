package remotesource

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitFailureDiagnosticRedactsSecretsAndPreservesExit(t *testing.T) {
	const secret = "diagnostic-fixture-private-value"
	t.Setenv("AGENT_ENV_DIAGNOSTIC_TOKEN", secret)
	_, err := git(context.Background(), "", "clone", "--bare", filepath.Join(t.TempDir(), secret), filepath.Join(t.TempDir(), "target"))
	var exit *exec.ExitError
	if !errors.As(err, &exit) || !strings.Contains(err.Error(), "local Git clone failed") || !strings.Contains(err.Error(), "[REDACTED]") || strings.Contains(err.Error(), secret) {
		t.Fatalf("unexpected Git diagnostic: %v", err)
	}
	_, err = git(context.Background(), "", "clone", "--bare", filepath.Join(t.TempDir(), strings.Repeat("long", 3000)), filepath.Join(t.TempDir(), "target"))
	if err == nil || len(err.Error()) > 4352 {
		t.Fatal("Git failure diagnostic was missing or unbounded")
	}
}
