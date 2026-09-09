package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestPageCreatePopupRaceRollsBackOnlyCreatedTarget(t *testing.T) {
	for _, mode := range []string{"normal", "popup", "close-false", "close-error", "close-missing-success", "close-still-present", "absence-error", "absence-missing-census", "ownership-lost", "census-error", "empty-id", "old-id", "old-worker-id", "post-nonpage", "post-missing-type", "rollback-nonpage", "rollback-missing-type"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if e := os.Mkdir(filepath.Join(root, "profile"), 0700); e != nil {
				t.Fatal(e)
			}
			var createCount, closeCount atomic.Int32
			var created atomic.Bool
			var mu sync.Mutex
			ids := map[string]bool{}
			for i := 0; i < 127; i++ {
				ids[fmt.Sprint(i)] = true
			}
			if mode == "old-worker-id" {
				ids["existing-worker"] = true
			}
			var closedID string
			var postCensus atomic.Int32
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
						if created.Load() {
							postCensus.Add(1)
						}
						if mode == "absence-missing-census" && postCensus.Load() == 2 {
							ws.WriteJSON(envelope{ID: q.ID, Result: json.RawMessage("{}")})
							continue
						}
						if (mode == "census-error" && postCensus.Load() == 1) || (mode == "absence-error" && postCensus.Load() == 2) {
							ws.WriteJSON(envelope{ID: q.ID, Error: &struct {
								Code    int
								Message string
							}{Code: -1, Message: "census failed"}})
							continue
						}
						mu.Lock()
						targets := []any{}
						for id := range ids {
							kind := "page"
							if id == "existing-worker" || (id == "created-by-call" && (mode == "post-nonpage" || (mode == "rollback-nonpage" && postCensus.Load() >= 2))) {
								kind = "worker"
							}
							if id == "created-by-call" && ((mode == "post-missing-type" && postCensus.Load() == 1) || (mode == "rollback-missing-type" && postCensus.Load() >= 2)) {
								kind = ""
							}
							targets = append(targets, map[string]any{"targetId": id, "type": kind, "url": "about:blank"})
						}
						mu.Unlock()
						result = map[string]any{"targetInfos": targets}
					case "Target.createTarget":
						createCount.Add(1)
						created.Store(true)
						mu.Lock()
						if mode != "normal" {
							ids["independent-popup"] = true
						}
						ids["created-by-call"] = true
						mu.Unlock()
						id := "created-by-call"
						if mode == "empty-id" {
							id = ""
						}
						if mode == "old-id" {
							id = "0"
						}
						if mode == "old-worker-id" {
							id = "existing-worker"
						}
						result = map[string]any{"targetId": id}
					case "Target.closeTarget":
						var a struct {
							TargetID string `json:"targetId"`
						}
						json.Unmarshal(q.Params, &a)
						closeCount.Add(1)
						mu.Lock()
						closedID = a.TargetID
						if mode != "close-false" && mode != "close-still-present" && mode != "rollback-nonpage" && mode != "rollback-missing-type" {
							delete(ids, a.TargetID)
						}
						mu.Unlock()
						result = map[string]any{"success": mode != "close-false"}
						if mode == "close-error" {
							ws.WriteJSON(envelope{ID: q.ID, Error: &struct {
								Code    int
								Message string
							}{Code: -1, Message: "close failed"}})
							continue
						}
						if mode == "close-missing-success" {
							result = map[string]any{}
						}
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
			o, e := (Client{}).Observe(context.Background(), r, domain.BrowserBinding{Runtime: "browser", CDPPort: "cdp"}, domain.BrowserRequest{Operation: "page-create", URL: "about:blank"}, func(context.Context) error {
				if mode == "ownership-lost" && created.Load() {
					return errors.New("ownership lost")
				}
				return nil
			})
			mu.Lock()
			defer mu.Unlock()
			if createCount.Load() != 1 || !o.ActionPerformed {
				t.Fatal("create effect not reached")
			}
			for i := 0; i < 127; i++ {
				if !ids[fmt.Sprint(i)] {
					t.Fatalf("sibling %d removed", i)
				}
			}
			if mode == "normal" {
				if e != nil || !o.Confirmed || closeCount.Load() != 0 || !ids["created-by-call"] || postCensus.Load() == 0 {
					t.Fatalf("normal create not confirmed: %+v %v", o, e)
				}
				return
			}
			if e == nil {
				t.Fatal("raced creation falsely succeeded")
			}
			if !ids["independent-popup"] {
				t.Fatal("independent popup removed")
			}
			if mode == "empty-id" || mode == "old-id" || mode == "ownership-lost" || mode == "old-worker-id" || mode == "post-nonpage" {
				if closeCount.Load() != 0 || o.Confirmed {
					t.Fatalf("unowned rollback attempted/confirmed: %+v %v", o, e)
				}
				return
			}
			if closedID != "created-by-call" || closeCount.Load() != 1 {
				t.Fatalf("wrong rollback: %q %d", closedID, closeCount.Load())
			}
			wantConfirmed := mode == "popup" || mode == "census-error" || mode == "post-missing-type"
			if o.Confirmed != wantConfirmed {
				t.Fatalf("rollback confirmation=%v want=%v: %v", o.Confirmed, wantConfirmed, e)
			}
			if wantConfirmed && (ids["created-by-call"] || len(ids) != 128 || postCensus.Load() < 2) {
				t.Fatal("rollback absence/census unproven")
			}
		})
	}
}
