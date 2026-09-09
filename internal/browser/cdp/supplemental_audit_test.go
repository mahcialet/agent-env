package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestSupplementPageLimitPreventsIrrecoverableGrowth(t *testing.T) {
	for _, initial := range []int32{127, 128, 129} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			root := t.TempDir()
			if e := os.Mkdir(filepath.Join(root, "profile"), 0700); e != nil {
				t.Fatal(e)
			}
			var count atomic.Int32
			count.Store(initial)
			var created, closed atomic.Int32
			port := 0
			up := websocket.Upgrader{}
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/json/version" {
					json.NewEncoder(w).Encode(map[string]any{"Browser": "Chrome/123", "Protocol-Version": "1.3", "webSocketDebuggerUrl": "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/browser/exact"})
					return
				}
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
					var result any
					switch q.Method {
					case "Browser.getVersion":
						result = map[string]any{"product": "Chrome/123", "protocolVersion": "1.3"}
					case "SystemInfo.getProcessInfo":
						result = map[string]any{"processInfo": []any{map[string]any{"type": "browser", "id": 42}}}
					case "Browser.getBrowserCommandLine":
						result = map[string]any{"arguments": []string{"chrome", "--enable-automation", "--headless=new", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=" + strconv.Itoa(port), "--user-data-dir=" + filepath.Join(root, "profile")}}
					case "Target.getTargets":
						pages := []any{}
						for i := int32(0); i < count.Load(); i++ {
							pages = append(pages, map[string]any{"targetId": fmt.Sprint(i), "type": "page", "url": "about:blank"})
						}
						result = map[string]any{"targetInfos": pages}
					case "Target.createTarget":
						created.Add(1)
						next := count.Add(1) - 1
						result = map[string]any{"targetId": fmt.Sprint(next)}
					case "Target.closeTarget":
						closed.Add(1)
						count.Add(-1)
						result = map[string]any{"success": true}
					default:
						result = map[string]any{}
					}
					raw, _ := json.Marshal(result)
					if ws.WriteJSON(envelope{ID: q.ID, Result: raw}) != nil {
						return
					}
				}
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			port, _ = strconv.Atoi(u.Port())
			r := domain.Runtime{Process: &domain.PersistentProcess{ProcessID: 42, ProcessStart: "birth", Ports: map[string]int{"cdp": port}, StateDirectory: root}}
			observe := func(q domain.BrowserRequest) (domain.BrowserObservation, error) {
				return (Client{}).Observe(context.Background(), r, domain.BrowserBinding{Runtime: "browser", CDPPort: "cdp"}, q, func(context.Context) error { return nil })
			}
			_, createErr := observe(domain.BrowserRequest{Operation: "page-create", URL: "about:blank"})
			_, closeErr := observe(domain.BrowserRequest{Operation: "page-close", Page: "0"})
			wantCreates, wantCloses := int32(0), int32(0)
			if initial == 127 {
				wantCreates = 1
			}
			if initial <= 128 {
				wantCloses = 1
			}
			if (createErr == nil) != (initial == 127) || created.Load() != wantCreates || (closeErr == nil) != (initial <= 128) || closed.Load() != wantCloses || count.Load() != initial+wantCreates-wantCloses {
				t.Fatalf("creation/close boundary broken: initial=%d create=%v close=%v createCalls=%d closeCalls=%d pages=%d", initial, createErr, closeErr, created.Load(), closed.Load(), count.Load())
			}
		})
	}
}

func TestSupplementIgnoredAXCannotSatisfyWait(t *testing.T) {
	for _, kind := range []string{"text", "gone"} {
		t.Run(kind, func(t *testing.T) {
			var ax atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": "http://localhost/"}}}
				case "Accessibility.getFullAXTree":
					ax.Add(1)
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 1, "ignored": true, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": "Needle"}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			sn, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"}, domain.BrowserRequest{WaitFor: kind, Contains: "Needle", Role: "button"})
			if ax.Load() == 0 {
				t.Fatal("AX predicate not reached")
			}
			if kind == "text" && (e == nil || sn != nil) {
				t.Fatal("ignored AX node falsely satisfied accessible text")
			}
			if kind == "gone" && (e != nil || sn == nil) {
				t.Fatalf("ignored AX node prevented gone success: %v", e)
			}
		})
	}
}
func TestSupplementPressedStateRefusesStaleInput(t *testing.T) {
	for _, mode := range []string{"pressed", "pressed-mixed", "checked", "selected", "expanded", "readonly", "required", "focusable", "focused", "multiselectable"} {
		t.Run(mode, func(t *testing.T) {
			property := strings.TrimSuffix(mode, "-mixed")
			var changed atomic.Bool
			var effects atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				if strings.HasPrefix(q.Method, "Input.") || q.Method == "DOM.focus" {
					effects.Add(1)
				}
				switch q.Method {
				case "Page.createIsolatedWorld":
					return map[string]any{"executionContextId": 42}
				case "Page.getFrameTree":
					pageURL := "http://localhost/"
					if changed.Load() && mode == "same-document-url" {
						pageURL += "#secret-fragment"
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": pageURL}}}
				case "Accessibility.getFullAXTree":
					backend := 4
					if changed.Load() && mode == "replaced" {
						backend = 5
					}
					role := "button"
					if mode == "password" {
						role = "textbox"
					}
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": backend, "role": map[string]any{"value": role}, "name": map[string]any{"value": "Control"}, "properties": []any{map[string]any{"name": property, "value": map[string]any{"value": func() any {
						if changed.Load() && mode == "pressed-mixed" {
							return "mixed"
						}
						return changed.Load()
					}()}}, map[string]any{"name": "disabled", "value": map[string]any{"value": changed.Load() && mode == "disabled"}}}}}}
				case "DOM.describeNode":
					typ := "text"
					if changed.Load() {
						typ = "password"
					}
					return map[string]any{"node": map[string]any{"attributes": []string{"type", typ}}}
				case "DOM.resolveNode":
					return map[string]any{"object": map[string]any{"objectId": "owned-object"}}
				case "DOM.getBoxModel":
					return map[string]any{"model": map[string]any{"content": []int{0, 0, 100, 0, 100, 100, 0, 100}}}
				case "Runtime.callFunctionOn":
					return map[string]any{"result": map[string]any{"value": !(changed.Load() && mode == "hidden")}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
			if e != nil {
				t.Fatal(e)
			}
			performed, _, e := act(context.Background(), c, "s", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "click", Prior: sn, Node: sn.Nodes[0].Ref}, func() error { changed.Store(true); return nil })
			if !changed.Load() {
				t.Fatal("action failed before ownership verification mutation")
			}
			if e == nil || performed || effects.Load() != 0 {
				t.Fatalf("mutation during verification reached input: %v %v", performed, e)
			}
		})
	}
}
