package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
)

// Exercise the public command boundary with actual executable discovery, rather
// than a provider fake: optional tools must be requested only by relevant work.
func TestStandaloneCommandsRequestGitOnlyWhenSourcesAreNeeded(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(flutterCLIManifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "absent-sdk"))
	t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
	for _, command := range []string{"version", "help", "validate", "plan", "create"} {
		t.Run(command, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "unused state 日本語")
			t.Setenv("AGENT_ENV_HOME", home)
			args := []string{command, "--output", "json"}
			switch command {
			case "help":
				args = []string{"--help"}
			case "validate":
				args = append(args, repo)
			case "plan", "create":
				args = append(args, repo, "--stack", "mobile")
			}
			var out, diagnostics bytes.Buffer
			cmd := New(&out, &diagnostics)
			cmd.SetArgs(args)
			err := cmd.Execute()
			if command == "plan" || command == "create" {
				if !errors.Is(err, app.ErrPrerequisite) || ExitCode(err) != 3 || !strings.Contains(strings.ToLower(err.Error()), "git") {
					t.Fatalf("missing Git was not a prerequisite failure: %v", err)
				}
				if out.Len() != 0 {
					t.Fatalf("failed command emitted success or allocated a lease: %s", out.String())
				}
			} else if err != nil || out.Len() == 0 {
				t.Fatalf("core command requires optional prerequisites: %v %s", err, out.String())
			}
			if diagnostics.Len() != 0 {
				t.Fatalf("unexpected provider output: %s", diagnostics.String())
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("command created state before it was needed: %v", err)
			}
		})
	}
}
