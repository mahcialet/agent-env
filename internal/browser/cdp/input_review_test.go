package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/mahcialet/agent-env/internal/domain"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func reviewActionFixture(t *testing.T, focus bool, readbackFailure string) (*connection, func(), *atomic.Int32) {
	t.Helper()
	var inputs atomic.Int32
	var activated atomic.Bool
	up := websocket.Upgrader{}
	server := newFixtureServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, e := up.Upgrade(w, r, nil)
		if e != nil {
			return
		}
		defer ws.Close()
		for {
			var q envelope
			if ws.ReadJSON(&q) != nil {
				return
			}
			var result any = map[string]any{}
			if strings.HasPrefix(q.Method, "Input.") {
				inputs.Add(1)
			}
			switch q.Method {
			case "Page.bringToFront":
				activated.Store(true)
			case "DOM.focus":
				if !activated.Load() {
					t.Error("control focused before page activation")
				}
			case "Page.getFrameTree":
				result = map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": "http://localhost/"}}}
			case "Accessibility.getFullAXTree":
				backend := 4
				if readbackFailure == "activation-replaced" && activated.Load() {
					backend = 5
				}
				result = map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": backend, "role": map[string]any{"value": "textbox"}, "name": map[string]any{"value": "Field"}, "properties": []any{map[string]any{"name": "editable", "value": map[string]any{"value": "plaintext"}}}}}}
			case "DOM.describeNode":
				result = map[string]any{"node": map[string]any{"attributes": []string{"type", "text"}}}
			case "Page.createIsolatedWorld":
				result = map[string]any{"executionContextId": 12}
			case "DOM.resolveNode":
				result = map[string]any{"object": map[string]any{"objectId": "object"}}
			case "DOM.getBoxModel":
				result = map[string]any{"model": map[string]any{"content": []int{0, 0, 100, 0, 100, 100, 0, 100}}}
			case "Runtime.callFunctionOn":
				var params struct{ FunctionDeclaration string }
				_ = json.Unmarshal(q.Params, &params)
				value := true
				if strings.Contains(params.FunctionDeclaration, "activeElement") {
					value = focus
					if readbackFailure == "inactive-document" {
						if !strings.Contains(params.FunctionDeclaration, "hasFocus()") {
							t.Error("keyboard verification ignores document focus")
						} else {
							value = false
						}
					}
					if readbackFailure == "selection-focus" && inputs.Load() >= 2 {
						value = false
					}
				}
				if strings.Contains(params.FunctionDeclaration, "function(expected)") && readbackFailure == "protocol" {
					if ws.WriteJSON(map[string]any{"id": q.ID, "error": map[string]any{"code": -32000, "message": "node gone"}}) != nil {
						return
					}
					continue
				}
				result = map[string]any{"result": map[string]any{"value": value}}
				if strings.Contains(params.FunctionDeclaration, "function(expected)") && readbackFailure == "exception" {
					result = map[string]any{"exceptionDetails": map[string]any{"text": "getter threw"}, "result": map[string]any{}}
				}
			}
			raw, _ := json.Marshal(result)
			if ws.WriteJSON(envelope{ID: q.ID, Result: raw}) != nil {
				return
			}
		}
	}))
	c, e := dial(context.Background(), "ws"+strings.TrimPrefix(server.URL, "http"))
	if e != nil {
		server.Close()
		t.Fatal(e)
	}
	return c, func() { closeFixtureConnection(c); server.Close() }, &inputs
}

func TestInputFocusRedirectRefusesKeyboardDispatch(t *testing.T) {
	for _, op := range []string{"key", "set-text"} {
		t.Run(op, func(t *testing.T) {
			c, done, inputs := reviewActionFixture(t, false, "")
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
			if e != nil {
				t.Fatal(e)
			}
			performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: op, Prior: sn, Node: sn.Nodes[0].Ref, Key: "Enter", Text: "secret"}, func() error { return nil })
			if e == nil || !performed || inputs.Load() != 0 {
				t.Fatalf("focus redirect dispatched input: performed=%v inputs=%d err=%v", performed, inputs.Load(), e)
			}
		})
	}
}

func TestPostInsertErrorRemainsUncertain(t *testing.T) {
	for _, mode := range []string{"protocol", "exception"} {
		t.Run(mode, func(t *testing.T) {
			c, done, inputs := reviewActionFixture(t, true, mode)
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
			if e != nil {
				t.Fatal(e)
			}
			performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "set-text", Prior: sn, Node: sn.Nodes[0].Ref, Text: "secret"}, func() error { return nil })
			var confirmed confirmedError
			if e == nil || !performed || inputs.Load() != 3 || errors.As(e, &confirmed) {
				t.Fatalf("post-effect error misclassified: performed=%v inputs=%d err=%v", performed, inputs.Load(), e)
			}
		})
	}
}

func TestURLWaitUsesRawURLButPersistsScrubbedEvidence(t *testing.T) {
	for _, suffix := range []string{"?status=ready", "#complete"} {
		t.Run(suffix, func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					frameURL, fragment := "http://localhost/"+suffix, ""
					if strings.HasPrefix(suffix, "#") {
						frameURL, fragment = "http://localhost/", suffix
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": frameURL, "urlFragment": fragment}}}
				case "Accessibility.getFullAXTree":
					return map[string]any{"nodes": []any{}}
				}
				return map[string]any{}
			})
			defer done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			sn, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"}, domain.BrowserRequest{WaitFor: "url", Contains: suffix})
			if e != nil {
				t.Fatal(e)
			}
			if sn.Page.URL != "http://localhost/" {
				t.Fatalf("raw URL published: %s", sn.Page.URL)
			}
		})
	}
}

func TestSelectionFocusChangeRefusesTextInsertion(t *testing.T) {
	c, done, inputs := reviewActionFixture(t, true, "selection-focus")
	defer done()
	sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
	if e != nil {
		t.Fatal(e)
	}
	performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "set-text", Prior: sn, Node: sn.Nodes[0].Ref, Text: "secret"}, func() error { return nil })
	if e == nil || !performed || inputs.Load() != 2 {
		t.Fatalf("text inserted after selection moved focus: performed=%v inputs=%d err=%v", performed, inputs.Load(), e)
	}
}

func TestInactiveDocumentRefusesKeyboardDispatch(t *testing.T) {
	c, done, inputs := reviewActionFixture(t, true, "inactive-document")
	defer done()
	sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
	if e != nil {
		t.Fatal(e)
	}
	performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "key", Prior: sn, Node: sn.Nodes[0].Ref, Key: "Enter"}, func() error { return nil })
	if e == nil || !performed || inputs.Load() != 0 {
		t.Fatalf("inactive document accepted keyboard input: performed=%v inputs=%d err=%v", performed, inputs.Load(), e)
	}
}

func TestActivationMutationRefusesKeyboardDispatch(t *testing.T) {
	c, done, inputs := reviewActionFixture(t, true, "activation-replaced")
	defer done()
	sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
	if e != nil {
		t.Fatal(e)
	}
	performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "key", Prior: sn, Node: sn.Nodes[0].Ref, Key: "Enter"}, func() error { return nil })
	if e == nil || !strings.Contains(e.Error(), "page activation") || !performed || inputs.Load() != 0 {
		t.Fatalf("activation mutation accepted keyboard input: performed=%v inputs=%d err=%v", performed, inputs.Load(), e)
	}
}
