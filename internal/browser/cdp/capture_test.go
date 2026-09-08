package cdp

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestConsoleIgnoresHistoryAndOmitsOversizeValues(t *testing.T) {
	c := eventBrowser(t, func(string) []envelope {
		var events []envelope
		for _, x := range []struct {
			stamp int64
			text  string
		}{{time.Now().Add(-time.Hour).UnixMilli(), "old-secret"}, {time.Now().Add(time.Minute).UnixMilli(), strings.Repeat("secret-prefix", 400)}} {
			raw, _ := json.Marshal(map[string]any{"type": "log", "timestamp": x.stamp, "args": []any{map[string]any{"type": "string", "value": x.text}}})
			events = append(events, envelope{Session: "s", Method: "Runtime.consoleAPICalled", Params: raw})
		}
		return events
	})
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

func TestNetworkCaptureBoundsEveryPersistedString(t *testing.T) {
	huge := strings.Repeat("é", 5000)
	c := eventBrowser(t, func(string) []envelope {
		raw, _ := json.Marshal(map[string]any{"requestId": huge, "type": huge, "request": map[string]any{"url": "http://localhost/", "method": huge}})
		failure, _ := json.Marshal(map[string]any{"requestId": huge, "errorText": huge})
		return []envelope{{Session: "s", Method: "Network.requestWillBeSent", Params: raw}, {Session: "s", Method: "Network.loadingFailed", Params: failure}}
	})
	var obs domain.BrowserObservation
	err := capture(context.Background(), c, "s", domain.BrowserRequest{Operation: "network", Duration: 20 * time.Millisecond}, &obs)
	if err != nil {
		t.Fatal(err)
	}
	if !obs.Truncated {
		t.Fatal("oversized network fields not marked truncated")
	}
	for _, e := range obs.Network {
		for _, v := range []string{e.ID, e.URL, e.Method, e.Type, e.Failure} {
			if len(v) > 4096 || !utf8.ValidString(v) {
				t.Fatalf("unbounded or invalid UTF-8 network field: %d", len(v))
			}
		}
	}
}
func TestNetworkCaptureCountsAllStringsAgainstTotalBudget(t *testing.T) {
	large := strings.Repeat("m", 4096)
	c := eventBrowser(t, func(string) []envelope {
		var events []envelope
		for range 100 {
			raw, _ := json.Marshal(map[string]any{"requestId": large, "type": large, "request": map[string]any{"url": "http://localhost/", "method": large}})
			events = append(events, envelope{Session: "s", Method: "Network.requestWillBeSent", Params: raw})
		}
		return events
	})
	var obs domain.BrowserObservation
	err := capture(context.Background(), c, "s", domain.BrowserRequest{Operation: "network", Duration: 20 * time.Millisecond}, &obs)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, e := range obs.Network {
		total += len(e.ID) + len(e.URL) + len(e.Method) + len(e.Type) + len(e.Failure)
	}
	if total > 65536 || !obs.Truncated {
		t.Fatalf("network budget: bytes=%d truncated=%v", total, obs.Truncated)
	}
}

// Domain-enable responses arrive only after these events are queued. An expired
// capture may omit the remaining queue, but must report that evidence loss.
func TestCaptureDeadlineMarksPendingEventsTruncated(t *testing.T) {
	for _, operation := range []string{"console", "network"} {
		t.Run(operation, func(t *testing.T) {
			const count = 128
			c := eventBrowser(t, func(string) []envelope {
				events := make([]envelope, count)
				method := "Network.requestWillBeSent"
				raw := json.RawMessage(`{"requestId":"wanted","request":{"url":"http://localhost/","method":"GET"}}`)
				if operation == "console" {
					method = "Runtime.consoleAPICalled"
					raw, _ = json.Marshal(map[string]any{"type": "log", "timestamp": time.Now().Add(time.Minute).UnixMilli(), "args": []any{map[string]any{"type": "string", "value": "wanted"}}})
				}
				for i := range events {
					events[i] = envelope{Session: "s", Method: method, Params: raw}
				}
				return events
			})
			var obs domain.BrowserObservation
			if err := capture(context.Background(), c, "s", domain.BrowserRequest{Operation: operation, Duration: time.Nanosecond}, &obs); err != nil {
				t.Fatal(err)
			}
			retained := len(obs.Console) + len(obs.Network)
			if retained < count && !obs.Truncated {
				t.Fatalf("deadline silently omitted queued events: retained=%d of %d", retained, count)
			}
			c.mu.Lock()
			subscribed := c.capture != nil
			c.mu.Unlock()
			if subscribed {
				t.Fatal("capture deadline left subscription active")
			}
		})
	}
}
