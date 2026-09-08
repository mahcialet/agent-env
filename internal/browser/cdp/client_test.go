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
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestEndpointBoundary(t *testing.T) {
	for _, s := range []string{"ws://127.0.0.1:123/devtools/browser/..", "ws://127.0.0.1:123/devtools/browser/%2e%2e", "ws://localhost:123/devtools/browser/a", "ws://127.0.0.1:124/devtools/browser/a", "wss://127.0.0.1:123/devtools/browser/a", "ws://u:p@127.0.0.1:123/devtools/browser/a", "ws://127.0.0.1:123/devtools/page/a", "ws://127.0.0.1:123/devtools/browser/a?q=x", "ws://127.0.0.1:123/devtools/browser/a/next"} {
		if validEndpoint(s, 123) {
			t.Errorf("accepted %q", s)
		}
	}
	if !validEndpoint("ws://127.0.0.1:123/devtools/browser/a", 123) {
		t.Fatal("rejected exact endpoint")
	}
}
func TestPrivateProfileFlags(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "profile"), 0700); e != nil {
		t.Fatal(e)
	}
	args := []string{"chrome", "--enable-automation", "--headless=new", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=123", "--user-data-dir=" + filepath.Join(root, "profile")}
	if e := validateFlags(args, root, 123); e != nil {
		t.Fatal(e)
	}
	mixed := append([]string(nil), args...)
	mixed[len(mixed)-1] = "--user-data-dir=" + root + "/profile"
	if e := validateFlags(mixed, root, 123); e != nil {
		t.Fatalf("native mixed separators: %v", e)
	}

	if validateFlags(append(args, "--remote-debugging-port=124"), root, 123) == nil {
		t.Fatal("duplicate flag accepted")
	}
	if validateFlags(args, root, 124) == nil {
		t.Fatal("foreign port accepted")
	}
	if validateFlags(args[:len(args)-1], root, 123) == nil {
		t.Fatal("missing profile accepted")
	}
}

func mockBrowser(t *testing.T, respond func(envelope) any) (*connection, func()) {
	t.Helper()
	up := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			a := respond(q)
			if a == nil {
				return
			}
			raw, _ := json.Marshal(a)
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
	return c, func() { c.close(); server.Close() }
}
func TestTransportCancellationAndDisconnect(t *testing.T) {
	c, done := mockBrowser(t, func(envelope) any { time.Sleep(100 * time.Millisecond); return map[string]any{} })
	defer done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if c.call(ctx, "", "Test.wait", nil, nil) == nil {
		t.Fatal("deadline ignored")
	}
	c.close()
	if c.call(context.Background(), "", "Test.closed", nil, nil) == nil {
		t.Fatal("disconnect ignored")
	}
}
func TestDOMSnapshotSuppressesSensitiveStrings(t *testing.T) {
	c, done := mockBrowser(t, func(envelope) any {
		return map[string]any{"strings": []string{"INPUT", "password-secret", "value", "cookie-secret"}, "documents": []any{map[string]any{"nodes": map[string]any{"nodeType": []int{1}, "nodeName": []int{0}, "parentIndex": []int{-1}, "backendNodeId": []int{42}, "nodeValue": []int{1}, "attributes": [][]int{{2, 3}}, "inputValue": map[string]any{"index": []int{0}, "value": []int{1}}}, "layout": map[string]any{"nodeIndex": []int{0}, "bounds": [][]int{{1, 2, 3, 4}}}}}}
	})
	defer done()
	raw, tr, e := domSnapshot(context.Background(), c, "s")
	if e != nil || tr {
		t.Fatalf("%v truncated=%v", e, tr)
	}
	if strings.Contains(string(raw), "secret") || !strings.Contains(string(raw), "INPUT") || !strings.Contains(string(raw), "42") {
		t.Fatalf("unsafe or missing DOM: %s", raw)
	}
}
func TestBrowserPIDAndDiscoveryProof(t *testing.T) {
	for _, bad := range []string{"", "pid", "version", "endpoint", "oversize"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			os.Mkdir(filepath.Join(root, "profile"), 0700)
			port := 0
			up := websocket.Upgrader{}
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/json/version" {
					if bad == "oversize" {
						fmt.Fprint(w, strings.Repeat("x", 65537))
						return
					}
					endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/browser/exact"
					if bad == "endpoint" {
						endpoint = "ws://127.0.0.1:1/devtools/browser/foreign"
					}
					json.NewEncoder(w).Encode(map[string]any{"Browser": "Chrome/123", "Protocol-Version": "1.3", "webSocketDebuggerUrl": endpoint})
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
						p := "Chrome/123"
						if bad == "version" {
							p = "Chrome/foreign"
						}
						result = map[string]any{"product": p, "protocolVersion": "1.3"}
					case "SystemInfo.getProcessInfo":
						pid := 42
						if bad == "pid" {
							pid = 43
						}
						result = map[string]any{"processInfo": []any{map[string]any{"type": "browser", "id": pid}}}
					case "Browser.getBrowserCommandLine":
						result = map[string]any{"arguments": []string{"chrome", "--enable-automation", "--headless=new", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=" + strconv.Itoa(port), "--user-data-dir=" + filepath.Join(root, "profile")}}
					case "Target.getTargets":
						result = map[string]any{"targetInfos": []any{map[string]any{"targetId": "page", "type": "page", "url": "http://example.invalid/?token=secret", "title": "test"}}}
					default:
						t.Errorf("unexpected method %s", q.Method)
						return
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
			checks := 0
			r := domain.Runtime{Process: &domain.PersistentProcess{ProcessID: 42, ProcessStart: "birth", Ports: map[string]int{"cdp": port}, StateDirectory: root}}
			o, e := (Client{}).Observe(context.Background(), r, domain.BrowserBinding{Runtime: "browser", CDPPort: "cdp"}, domain.BrowserRequest{Operation: "pages"}, func(context.Context) error { checks++; return nil })
			if bad != "" {
				if e == nil {
					t.Fatal("unsafe identity accepted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if checks != 2 || len(o.Pages) != 1 || strings.Contains(o.Pages[0].URL, "secret") {
				t.Fatalf("bad proof/output: %d %+v", checks, o)
			}
		})
	}
}
