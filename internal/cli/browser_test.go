package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserRejectsMissingOrUnsafeCLIInputBeforeStore(t *testing.T) {
	for _, args := range [][]string{{"browser", "set-text", "lease", "--snapshot", "snap", "--node", "n1"}, {"browser", "navigate", "lease"}, {"browser", "page-close", "lease"}, {"browser", "pages", "lease", "--timeout", "0s"}, {"browser", "network", "lease", "--duration", "0s"}, {"browser", "evaluate", "lease", "--expression", "document.cookie"}, {"browser", "pages", "lease", "--cdp-url", "http://127.0.0.1:9222"}} {
		t.Run(args[1], func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "state")
			t.Setenv("AGENT_ENV_HOME", home)
			var out bytes.Buffer
			c := New(&out, &out)
			c.SetArgs(args)
			if err := c.Execute(); err == nil {
				t.Fatal("invalid browser args accepted")
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatal("invalid input opened store")
			}
		})
	}
}
