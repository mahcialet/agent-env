package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	server := newFixtureServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(func() { closeFixtureConnection(c) })
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
	entered := make(chan *eventSubscription, 1)
	var c *connection
	c = eventBrowser(t, func(string) []envelope {
		c.mu.Lock()
		entered <- c.capture
		c.mu.Unlock()
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
	closeFixtureConnection(c)
	select {
	case sub := <-entered:
		if sub == nil || !sub.overflow {
			t.Fatal("capture failed without exercising queue overflow")
		}
	default:
		t.Fatal("capture failed before enabling event delivery")
	}
}

func TestTransportSubscriptionQueueBoundary(t *testing.T) {
	for _, count := range []int{512, 513} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			c := eventBrowser(t, func(string) []envelope {
				events := make([]envelope, count)
				for i := range events {
					events[i] = envelope{Session: "s", Method: "Network.requestWillBeSent", Params: json.RawMessage(`{}`)}
				}
				return events
			})
			queue, stop, err := c.subscribe("s", "Network.requestWillBeSent")
			if err != nil {
				t.Fatal(err)
			}
			defer stop()
			err = c.call(context.Background(), "s", "Network.enable", nil, nil)
			// No consumer drains the queue before the response: the peer sends
			// every event first, so crossing its capacity is a forced schedule.
			if len(queue) != 512 {
				t.Fatalf("boundary not reached: queued %d", len(queue))
			}
			_, stoppedErr := stop()
			if count == 512 {
				if err != nil || stoppedErr != nil {
					t.Fatalf("exact capacity rejected: %v / %v", err, stoppedErr)
				}
			} else if err == nil || stoppedErr == nil || !strings.Contains(stoppedErr.Error(), "queue overflow") {
				t.Fatalf("overflow cause missing: %v / %v", err, stoppedErr)
			}
		})
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
