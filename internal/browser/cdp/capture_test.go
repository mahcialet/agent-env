package cdp

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestConsoleIgnoresHistoryAndOmitsOversizeValues(t *testing.T) {
	c, done := mockBrowser(t, func(envelope) any { return map[string]any{} })
	defer done()
	for _, x := range []struct {
		stamp int64
		text  string
	}{{time.Now().Add(-time.Hour).UnixMilli(), "old-secret"}, {time.Now().Add(time.Minute).UnixMilli(), strings.Repeat("secret-prefix", 400)}} {
		raw, _ := json.Marshal(map[string]any{"type": "log", "timestamp": x.stamp, "args": []any{map[string]any{"type": "string", "value": x.text}}})
		c.events <- envelope{Session: "s", Method: "Runtime.consoleAPICalled", Params: raw}
	}
	var o domain.BrowserObservation
	e := capture(context.Background(), c, "s", domain.BrowserRequest{Operation: "console", Duration: 20 * time.Millisecond}, &o)
	if e != nil {
		t.Fatal(e)
	}
	if len(o.Console) != 1 || o.Console[0].Text != "[TRUNCATED]" || !o.Truncated {
		t.Fatalf("unsafe console evidence: %+v", o)
	}
}
func TestSnapshotOmitsOversizeSensitiveValue(t *testing.T) {
	c, done := mockBrowser(t, func(q envelope) any {
		switch q.Method {
		case "Page.getFrameTree":
			return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": "http://localhost/"}}}
		case "Accessibility.getFullAXTree":
			return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 4, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": strings.Repeat("secret", 1000)}}}}
		default:
			return map[string]any{}
		}
	})
	defer done()
	sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{})
	if e != nil {
		t.Fatal(e)
	}
	if !sn.Truncated || len(sn.Nodes) != 1 || sn.Nodes[0].Name != "[TRUNCATED]" {
		t.Fatalf("unsafe snapshot: %+v", sn)
	}
}
func TestWaitUsesOperationDeadlineRatherThanCaptureDuration(t *testing.T) {
	var calls atomic.Int32
	c, done := mockBrowser(t, func(q envelope) any {
		switch q.Method {
		case "Page.getFrameTree":
			return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": "http://localhost/"}}}
		case "Accessibility.getFullAXTree":
			name := "pending"
			if calls.Add(1) > 1 {
				name = "ready"
			}
			return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 4, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": name}}}}
		default:
			return map[string]any{}
		}
	})
	defer done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sn, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{}, domain.BrowserRequest{WaitFor: "text", Contains: "ready", Duration: time.Millisecond})
	if e != nil || sn == nil || calls.Load() < 2 {
		t.Fatalf("wait prematurely ended: %v", e)
	}
}
