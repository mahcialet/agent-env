package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerRejectsTransientInputBeforeJournalOrPreparation(t *testing.T) {
	for _, kind := range []string{"ui", "browser"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			home := t.TempDir()
			j, err := OpenJournal(home, "host")
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			if err = j.Bind(ctx, "controller"); err != nil {
				t.Fatal(err)
			}
			secret := "transient-worker-input-792b94"
			payloads := []string{`{"` + kind + `":{"Operation":"set-text","Text":"` + secret + `"}}`, `{"` + kind + `":{"Operation":"snapshot","Text":"` + secret + `","Text":""}}`}
			for _, payload := range payloads {
				op := protocol.Operation{ID: "secret-operation", LeaseID: "lease", ControllerID: j.Identity.ControllerID, HostID: j.Identity.HostID, HostInstanceID: j.Identity.HostInstanceID, Epoch: 1, Kind: kind, Payload: json.RawMessage(payload)}
				if _, err = j.Receive(ctx, op); !errors.Is(err, protocol.ErrTransientInput) || strings.Contains(err.Error(), secret) {
					t.Fatalf("journal guard error: %v", err)
				}
				if err = (&AppExecutor{}).Prepare(ctx, op); !errors.Is(err, protocol.ErrTransientInput) || strings.Contains(err.Error(), secret) {
					t.Fatalf("direct executor guard: %v", err)
				}
			}
			var count int
			if err = j.db.QueryRowContext(ctx, "SELECT count(*) FROM receipts").Scan(&count); err != nil || count != 0 {
				t.Fatalf("unsafe receipt persisted: %d %v", count, err)
			}
			files, err := os.ReadDir(home)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "worker.db") {
					b, e := os.ReadFile(filepath.Join(home, file.Name()))
					if e != nil {
						t.Fatal(e)
					}
					if bytes.Contains(b, []byte(secret)) {
						t.Fatalf("transient input in %s", file.Name())
					}
				}
			}
			req := Request{UI: app.UIOptions{Text: secret}, Browser: app.BrowserOptions{}}
			if kind == "browser" {
				req = Request{Browser: app.BrowserOptions{}}
				req.Browser.Text = secret
			}
			if err = validateRequest(protocol.Operation{Kind: kind}, req); !errors.Is(err, protocol.ErrTransientInput) {
				t.Fatalf("typed request guard: %v", err)
			}
		})
	}
}
