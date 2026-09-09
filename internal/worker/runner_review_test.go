package worker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"strings"
	"testing"
)

type rejectedPreflight struct{ err error }

func (p rejectedPreflight) Prepare(context.Context, protocol.Operation) error { return p.err }
func (rejectedPreflight) Execute(context.Context, protocol.Operation) protocol.Result {
	panic("rejected preflight executed")
}
func (rejectedPreflight) Recover(context.Context, protocol.Operation) protocol.Result {
	panic("fresh rejected preflight recovered")
}
func TestPreflightResultRedactsAndBoundsBeforeJournalAndUpload(t *testing.T) {
	secret := "worker-only-secret-value-75291"
	t.Setenv("WORKER_API_TOKEN", secret)
	for _, stage := range []string{"prepare", "download"} {
		for _, oversize := range []bool{false, true} {
			name := stage
			if oversize {
				name += "-oversize"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				r, receipt, _, base := runnerFixture(t)
				j := r.Journal
				op := receipt.Operation
				message := "preflight failed: " + secret
				if oversize {
					message += strings.Repeat("界", 20000)
				}
				executor := rejectedPreflight{errors.New(message)}
				client := &preflightTransport{fakeTransport: base}
				r.Executor = executor
				r.Transport = client
				if stage == "download" {
					r.Download = func(context.Context, protocol.Operation) error { return errors.New(message) }
				}
				if err := r.handle(ctx, receipt); err != nil {
					t.Fatal(err)
				}
				stored, err := j.Get(ctx, op.ID)
				if err != nil || stored.Result == nil {
					t.Fatalf("missing durable result %v", err)
				}
				raw, _ := json.Marshal(stored)
				if strings.Contains(string(raw), secret) {
					t.Fatal("worker secret persisted in receipt and controller result")
				}
				if len(stored.Result.Payload) > 20<<10 {
					t.Fatalf("unbounded preflight result: %d bytes", len(stored.Result.Payload))
				}
				if stored.Result.State != "failed" || !strings.Contains(string(stored.Result.Payload), `"effects_started":false`) {
					t.Fatal("preflight failure crossed effect boundary")
				}
				if len(client.results) != 1 || strings.Contains(string(client.results[0].Payload), secret) {
					t.Fatal("secret reached controller")
				}
			})
		}
	}
}

type preflightTransport struct {
	*fakeTransport
	results []protocol.Result
}

func (c *preflightTransport) Complete(ctx context.Context, r protocol.Result) error {
	c.results = append(c.results, r)
	return c.fakeTransport.Complete(ctx, r)
}
