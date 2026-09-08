package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahcialet/agent-env/internal/domain"
)

// A real WebSocket peer emits events before each matching method response.
// This exercises reader queueing and dispatch rather than calling internals.
func eventBrowser(t *testing.T, events func(string) []envelope) *connection {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			_, raw, err := ws.ReadMessage()
			if err != nil {
				return
			}
			var req envelope
			if json.Unmarshal(raw, &req) != nil {
				return
			}
			for _, event := range events(req.Method) {
				raw, _ := json.Marshal(event)
				if ws.WriteMessage(websocket.TextMessage, raw) != nil {
					return
				}
			}
			reply, _ := json.Marshal(map[string]any{"id": req.ID, "result": map[string]any{}})
			if ws.WriteMessage(websocket.TextMessage, reply) != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	c, err := dial(context.Background(), "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.close)
	return c
}

func TestTransportIgnoresUnsubscribedEvents(t *testing.T) {
	c := eventBrowser(t, func(string) []envelope {
		events := make([]envelope, 1200)
		for i := range events {
			events[i] = envelope{Session: "s", Method: "Page.lifecycleEvent", Params: json.RawMessage(`{}`)}
		}
		return events
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.call(ctx, "s", "Runtime.evaluate", nil, nil); err != nil {
		t.Fatalf("irrelevant notifications disconnected ordinary command: %v", err)
	}
}

func TestTransportCaptureIgnoresOtherSessionsAndMethods(t *testing.T) {
	c := eventBrowser(t, func(string) []envelope {
		events := make([]envelope, 1200)
		for i := range events {
			events[i] = envelope{Session: "other", Method: "Network.requestWillBeSent", Params: json.RawMessage(`{}`)}
			if i%2 == 0 {
				events[i].Session = "s"
				events[i].Method = "Page.lifecycleEvent"
			}
		}
		events = append(events, envelope{Session: "s", Method: "Network.requestWillBeSent", Params: json.RawMessage(`{"requestId":"wanted","request":{"url":"http://localhost/","method":"GET"},"type":"Fetch"}`)})
		return events
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var obs domain.BrowserObservation
	err := capture(ctx, c, "s", domain.BrowserRequest{Operation: "network", Duration: time.Second}, &obs)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs.Network) != 1 || obs.Network[0].ID != "wanted" {
		t.Fatalf("capture: %+v", obs.Network)
	}
}

func TestTransportSubscribedOverflowFailsClosed(t *testing.T) {
	c := eventBrowser(t, func(string) []envelope {
		events := make([]envelope, 600)
		for i := range events {
			events[i] = envelope{Session: "s", Method: "Network.requestWillBeSent", Params: json.RawMessage(`{"requestId":"wanted","request":{"url":"http://localhost/","method":"GET"}}`)}
		}
		return events
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := capture(ctx, c, "s", domain.BrowserRequest{Operation: "network", Duration: time.Second}, &domain.BrowserObservation{}); err == nil {
		t.Fatal("subscribed overflow silently succeeded")
	}
}

func TestTransportCaptureSubscriptionEndsWithOperation(t *testing.T) {
	c := eventBrowser(t, func(method string) []envelope {
		count := 1
		if method != "Network.enable" {
			count = 1200
		}
		events := make([]envelope, count)
		for i := range events {
			events[i] = envelope{Session: "s", Method: "Network.requestWillBeSent", Params: json.RawMessage(`{"requestId":"wanted","request":{"url":"http://localhost/","method":"GET"}}`)}
		}
		return events
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := capture(ctx, c, "s", domain.BrowserRequest{Operation: "network", Duration: time.Second}, &domain.BrowserObservation{}); err != nil {
		t.Fatal(err)
	}
	if err := c.call(ctx, "s", "Runtime.evaluate", nil, nil); err != nil {
		t.Fatalf("completed capture kept queueing events: %v", err)
	}
}
