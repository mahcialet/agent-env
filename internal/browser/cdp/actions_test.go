package cdp

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestNodeChangesDuringOwnershipVerificationNeverInputs(t *testing.T) {
	for _, mode := range []string{"disabled", "hidden", "replaced", "password", "checked", "same-document-url"} {
		t.Run(mode, func(t *testing.T) {
			var changed atomic.Bool
			var effects atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				if strings.HasPrefix(q.Method, "Input.") || q.Method == "DOM.focus" {
					effects.Add(1)
				}
				switch q.Method {
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
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": backend, "role": map[string]any{"value": role}, "name": map[string]any{"value": "Control"}, "properties": []any{map[string]any{"name": "checked", "value": map[string]any{"value": changed.Load() && mode == "checked"}}, map[string]any{"name": "disabled", "value": map[string]any{"value": changed.Load() && mode == "disabled"}}}}}}
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
			if e == nil || performed || effects.Load() != 0 {
				t.Fatalf("mutation during verification reached input: %v %v", performed, e)
			}
		})
	}
}

func TestScreenshotMessageBoundAndDecodeFailure(t *testing.T) {
	for _, oversize := range []bool{false, true} {
		t.Run(map[bool]string{false: "malformed", true: "oversize"}[oversize], func(t *testing.T) {
			c, done := mockBrowser(t, func(envelope) any {
				data := "%%%"
				if oversize {
					data = strings.Repeat("A", maxMessage+1)
				}
				return map[string]any{"data": data}
			})
			defer done()
			_, e := screenshot(context.Background(), c, "s")
			if oversize && e == nil {
				t.Fatal("oversize response accepted")
			}
			if !oversize && e == nil {
				t.Fatal("malformed base64 accepted")
			}
		})
	}
}

func TestStaleAndAmbiguousNodeNeverInputs(t *testing.T) {
	for _, mode := range []string{"replaced", "document", "duplicate", "truncated", "diagnostic"} {
		t.Run(mode, func(t *testing.T) {
			var changed atomic.Bool
			var effects atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				if strings.HasPrefix(q.Method, "Input.") || q.Method == "DOM.focus" {
					effects.Add(1)
				}
				switch q.Method {
				case "Page.getFrameTree":
					loader := "doc"
					if changed.Load() && mode == "document" {
						loader = "other"
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": loader, "url": "http://localhost/"}}}
				case "Accessibility.enable":
					return map[string]any{}
				case "Accessibility.getFullAXTree":
					backend := 4
					if changed.Load() && mode == "replaced" {
						backend = 5
					}
					n := map[string]any{"nodeId": "1", "backendDOMNodeId": backend, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": "Do it"}}
					nodes := []any{n}
					if changed.Load() && mode == "duplicate" {
						nodes = append(nodes, map[string]any{"nodeId": "2", "backendDOMNodeId": 6, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": "Do it"}})
					}
					return map[string]any{"nodes": nodes}
				default:
					t.Errorf("unexpected input-stage call %s", q.Method)
					return map[string]any{}
				}
			})
			defer done()
			sn, e := snapshot(context.Background(), c, "session", domain.BrowserIdentity{}, domain.BrowserPage{ID: "page"})
			if e != nil {
				t.Fatal(e)
			}
			changed.Store(true)
			if mode == "truncated" {
				sn.Truncated = true
			}
			if mode == "diagnostic" {
				sn.Nodes[0].BackendID = 0
			}
			performed, _, e := act(context.Background(), c, "session", domain.BrowserIdentity{}, sn.Page, domain.BrowserRequest{Operation: "click", Prior: sn, Node: sn.Nodes[0].Ref}, func() error { t.Fatal("stale action reached fence"); return nil })
			if e == nil || performed || effects.Load() != 0 {
				t.Fatalf("unsafe stale action: %v %v %d", e, performed, effects.Load())
			}
		})
	}
}

func TestAXPasswordPrivacyAndCrossOriginRefusal(t *testing.T) {
	for _, cross := range []bool{false, true} {
		t.Run(map[bool]string{false: "password", true: "cross-origin"}[cross], func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					tree := map[string]any{"frame": map[string]any{"id": "main", "loaderId": "doc", "url": "http://localhost/"}}
					if cross {
						tree["childFrames"] = []any{map[string]any{"frame": map[string]any{"id": "child", "url": "http://foreign.invalid/"}}}
					}
					return map[string]any{"frameTree": tree}
				case "Accessibility.enable":
					return map[string]any{}
				case "Accessibility.getFullAXTree":
					return map[string]any{"nodes": []any{map[string]any{"nodeId": "1", "backendDOMNodeId": 4, "role": map[string]any{"value": "textbox"}, "name": map[string]any{"value": "Password"}, "value": map[string]any{"value": "secret"}}}}
				case "DOM.describeNode":
					return map[string]any{"node": map[string]any{"attributes": []string{"type", "password", "value", "secret"}}}
				default:
					t.Errorf("unexpected %s", q.Method)
					return map[string]any{}
				}
			})
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "p"})
			if cross {
				if e == nil {
					t.Fatal("cross-origin frame silently omitted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if len(sn.Nodes) != 1 || !sn.Nodes[0].Password || sn.Nodes[0].Value != "" {
				t.Fatalf("password privacy missing: %+v", sn)
			}
		})
	}
}

func TestRedactedNamePreservesFingerprintAuthorization(t *testing.T) {
	n := domain.BrowserNode{BackendID: 4, Frame: "main", Role: "button", Name: "sensitive-label"}
	n.Fingerprint = fingerprint(n)
	old := n
	old.Name = "[REDACTED]"
	if !uniqueFreshNode([]domain.BrowserNode{n}, old) {
		t.Fatal("redacted display name blocked unchanged fingerprint")
	}
	duplicate := n
	duplicate.BackendID = 5
	duplicate.Fingerprint = fingerprint(duplicate)
	if uniqueFreshNode([]domain.BrowserNode{n, duplicate}, old) {
		t.Fatal("duplicate raw semantic label accepted")
	}
}
func TestAXStatesRejectArbitraryText(t *testing.T) {
	for _, bad := range []any{"secret-value", 42, nil} {
		if _, ok := axState(bad); ok {
			t.Fatalf("retained text state %v", bad)
		}
	}
	for _, good := range []any{true, false, "true", "false", "mixed"} {
		if _, ok := axState(good); !ok {
			t.Fatal("missing boolean state")
		}
	}
}
