package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIRequiredFlagsRejectBeforeOpeningRegistry(t *testing.T) {
	cases := [][]string{
		{"ui", "tap", "lease"},
		{"ui", "set-text", "lease", "--snapshot", "s", "--node", "n1"},
		{"ui", "tap-coordinate", "lease", "--x", "0"},
		{"ui", "swipe", "lease", "--x", "0", "--y", "0", "--to-x", "1"},
		{"ui", "wait", "lease"},
		{"ui", "logcat", "lease"},
		{"ui", "recover", "lease"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "registry")
			t.Setenv("AGENT_ENV_HOME", home)
			var out, errOut bytes.Buffer
			cmd := New(&out, &errOut)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "required flag") {
				t.Fatalf("expected missing flag refusal, got %v", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("opened registry before flag validation: %v", err)
			}
		})
	}
}

func TestUIHelpListsOwnedActions(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs([]string{"ui", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"snapshot", "screenshot", "set-text", "tap-coordinate", "back", "home", "swipe", "wait", "logcat", "recover"} {
		if !strings.Contains(out.String(), name) {
			t.Fatalf("missing command %s", name)
		}
	}
}

func TestUIExitClassificationPreservesCauses(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"missing-helper", fmt.Errorf("observer unavailable: %w", app.ErrPrerequisite), 3},
		{"selection", fmt.Errorf("invalid runtime: %w", domain.ErrUIInput), 2},
		{"missing-lease", fmt.Errorf("lookup: %w", sqlite.ErrNotFound), 2},
		{"stale", errors.New("AGENTENV-UI-STALE: node changed"), 2},
		{"ambiguous", errors.New("AGENTENV-UI-AMBIGUOUS: duplicate fingerprint"), 2},
		{"registry", errors.New("registry unavailable"), 7},
		{"observation", errors.New("invalid screenshot"), 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := uiExit(tc.err)
			if ExitCode(got) != tc.code || !errors.Is(got, tc.err) {
				t.Fatalf("exit=%d error=%v", ExitCode(got), got)
			}
		})
	}
}
func TestUIRegistryOpenFailureReturnsObservationExit(t *testing.T) {
	home := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(home, []byte("existing user file"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_ENV_HOME", home)
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"ui", "snapshot", "lease"})
	if err := cmd.Execute(); err == nil || ExitCode(err) != 7 {
		t.Fatalf("registry failure must exit7: %v code=%d", err, ExitCode(err))
	}
	data, err := os.ReadFile(home)
	if err != nil || string(data) != "existing user file" {
		t.Fatal("registry failure altered existing file")
	}
}
func TestUIMissingLeaseReturnsSelectionExit(t *testing.T) {
	t.Setenv("AGENT_ENV_HOME", filepath.Join(t.TempDir(), "registry"))
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"ui", "snapshot", "missing-lease"})
	if err := cmd.Execute(); err == nil || ExitCode(err) != 2 || !errors.Is(err, sqlite.ErrNotFound) {
		t.Fatalf("missing lease exit: %v code=%d", err, ExitCode(err))
	}
}

func TestUIInvalidOptionValueReturnsInputExit(t *testing.T) {
	for _, flags := range [][]string{{"--timeout", "61s"}, {"--application", "mobile", "--runtime", "phone"}} {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			t.Setenv("AGENT_ENV_HOME", filepath.Join(t.TempDir(), "registry"))
			var out, stderr bytes.Buffer
			cmd := New(&out, &stderr)
			cmd.SetArgs(append([]string{"ui", "snapshot", "missing-lease"}, flags...))
			if err := cmd.Execute(); err == nil || ExitCode(err) != 2 || !errors.Is(err, domain.ErrUIInput) {
				t.Fatalf("invalid option exit: %v code=%d", err, ExitCode(err))
			}
		})
	}
}
func TestUICorruptRegistryReturnsObservationExit(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "state.db")
	original := []byte("not a SQLite database")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_ENV_HOME", home)
	var out, stderr bytes.Buffer
	cmd := New(&out, &stderr)
	cmd.SetArgs([]string{"ui", "snapshot", "lease"})
	if err := cmd.Execute(); err == nil || ExitCode(err) != 7 {
		t.Fatalf("corrupt registry exit: %v code=%d", err, ExitCode(err))
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatal("corrupt registry overwritten")
	}
}

func TestUIExplicitZeroDurationRejectsBeforeRegistry(t *testing.T) {
	for _, args := range [][]string{{"ui", "snapshot", "lease", "--timeout", "0s"}, {"ui", "logcat", "lease", "--application", "mobile", "--since", "0s"}, {"ui", "swipe", "lease", "--x", "1", "--y", "1", "--to-x", "2", "--to-y", "2", "--duration", "0s"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "registry")
			t.Setenv("AGENT_ENV_HOME", home)
			var out, stderr bytes.Buffer
			cmd := New(&out, &stderr)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil || ExitCode(err) != 2 {
				t.Fatalf("zero duration exit %v", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatal("zero duration opened registry")
			}
		})
	}
}
func TestUILogTablePrintsInlineText(t *testing.T) {
	var out bytes.Buffer
	writeUILog(&out, "PID123 useful diagnostic")
	if out.String() != "PID123 useful diagnostic\n" {
		t.Fatalf("table log %q", out.String())
	}
	out.Reset()
	writeUILog(&out, "already newline\n")
	if out.String() != "already newline\n" {
		t.Fatal("extra newline")
	}
}
