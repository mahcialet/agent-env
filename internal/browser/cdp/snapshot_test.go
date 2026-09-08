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

func TestFrameClassificationUsesSecurityOrigin(t *testing.T) {
	for _, tc := range []struct {
		name, pageURL, childURL, origin string
		allowed                         bool
	}{
		{"http", "https://site.test/", "https://site.test/frame", "https://site.test", true},
		{"blob", "https://site.test/", "blob:https://site.test/uuid", "https://site.test", true},
		{"inherited-blank", "https://site.test/", "about:blank", "https://site.test", true},
		{"inherited-srcdoc", "https://site.test/", "about:srcdoc", "https://site.test", true},
		{"inherited-blank-cdp-opaque", "https://site.test/", "about:blank", "://", true},
		{"inherited-srcdoc-cdp-opaque", "https://site.test/", "about:srcdoc", "://", true},
		{"sandboxed-blank", "https://site.test/", "about:blank", "://", false},
		{"sandboxed-srcdoc", "https://site.test/", "about:srcdoc", "null", false},
		{"sandboxed-same-url", "https://site.test/", "https://site.test/frame", "://", false},
		{"missing-origin", "https://site.test/", "about:blank", "", false},
		{"foreign-origin", "https://site.test/", "about:blank", "https://foreign.test", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "d", "url": tc.pageURL, "securityOrigin": "https://site.test"}, "childFrames": []any{map[string]any{"frame": map[string]any{"id": "child", "loaderId": "c", "url": tc.childURL, "securityOrigin": tc.origin}}}}}
				case "DOM.getFrameOwner":
					return map[string]any{"backendNodeId": 77}
				case "Page.createIsolatedWorld":
					var p struct {
						FrameID             string
						GrantUniveralAccess bool
					}
					json.Unmarshal(q.Params, &p)
					if p.FrameID != "main" || p.GrantUniveralAccess {
						t.Error("origin probe escaped verified parent")
					}
					return map[string]any{"executionContextId": 88}
				case "DOM.resolveNode":
					var p struct{ ExecutionContextID int }
					json.Unmarshal(q.Params, &p)
					if p.ExecutionContextID != 88 {
						t.Error("origin proof used page realm")
					}
					return map[string]any{"object": map[string]any{"objectId": "isolated-owner"}}
				case "Runtime.callFunctionOn":
					var p struct{ FunctionDeclaration string }
					json.Unmarshal(q.Params, &p)
					if !strings.Contains(p.FunctionDeclaration, "Object.getOwnPropertyDescriptor") {
						t.Error("origin proof trusted page property")
					}
					return map[string]any{"result": map[string]any{"value": strings.Contains(tc.name, "cdp-opaque")}}
				case "Accessibility.getFullAXTree":
					var p struct{ FrameID string }
					json.Unmarshal(q.Params, &p)
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 1, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": p.FrameID}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{})
			if tc.allowed {
				if e != nil || sn == nil || len(sn.Nodes) != 2 {
					t.Fatalf("same-origin child rejected: %v %+v", e, sn)
				}
			} else if e == nil {
				t.Fatal("opaque or foreign frame accepted")
			}
		})
	}
}

func TestDOMNameTruncationIsReported(t *testing.T) {
	for _, size := range []int{128, 129} {
		t.Run(strings.Repeat("n", size), func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				if q.Method == "Page.getFrameTree" {
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main"}}}
				}
				if q.Method != "DOMSnapshot.captureSnapshot" {
					return map[string]any{}
				}
				return map[string]any{"strings": []string{strings.Repeat("n", size), "main"}, "documents": []any{map[string]any{"frameId": 1, "nodes": map[string]any{"nodeType": []int{1}, "nodeName": []int{0}}}}}
			})
			defer done()
			raw, truncated, e := domSnapshot(context.Background(), c, "s")
			if e != nil {
				t.Fatal(e)
			}
			if truncated != (size > 128) {
				t.Fatalf("truncation flag=%v for %d-byte name: %s", truncated, size, raw)
			}
			var saved struct{ Truncated bool }
			json.Unmarshal(raw, &saved)
			if saved.Truncated != truncated {
				t.Fatal("artifact truncation disagrees")
			}
		})
	}
}

func TestGoneCannotSucceedWithTruncatedSnapshot(t *testing.T) {
	for _, kind := range []string{"node-limit", "name-limit"} {
		t.Run(kind, func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": "d", "url": "http://localhost/", "securityOrigin": "http://localhost"}}}
				case "Accessibility.getFullAXTree":
					nodes := []any{}
					count := 2049
					if kind == "name-limit" {
						count = 1
					}
					for i := 0; i < count; i++ {
						name := "other"
						if kind == "name-limit" {
							name = strings.Repeat("x", 4097) + "needle"
						} else if i == count-1 {
							name = "needle"
						}
						nodes = append(nodes, map[string]any{"backendDOMNodeId": i + 1, "role": map[string]any{"value": "button"}, "name": map[string]any{"value": name}})
					}
					return map[string]any{"nodes": nodes}
				default:
					return map[string]any{}
				}
			})
			defer done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if _, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{}, domain.BrowserRequest{WaitFor: "gone", Contains: "needle"}); e == nil {
				t.Fatal("false absence from incomplete snapshot")
			}
		})
	}
}

func TestOutOfProcessFrameCannotBeSilentlyOmitted(t *testing.T) {
	for _, parent := range []string{"main", "other-page", ""} {
		t.Run("parent-"+parent, func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "url": "https://site.test/", "securityOrigin": "https://site.test"}}}
				case "Target.getTargets":
					return map[string]any{"targetInfos": []any{map[string]any{"type": "iframe", "targetId": "opaque-child", "parentId": parent, "parentFrameId": parent}}}
				case "DOM.getDocument":
					return map[string]any{"root": map[string]any{"nodeType": 9, "children": []any{map[string]any{"nodeType": 1, "frameId": "opaque-child"}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			_, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"})
			if parent == "other-page" {
				if e != nil {
					t.Fatalf("unrelated page rejected: %v", e)
				}
			} else if e == nil {
				t.Fatal("opaque OOPIF omitted without warning")
			}
		})
	}
}

func TestWaitRetriesOnlyUncommittedOriginWithoutPartialEvidence(t *testing.T) {
	for _, kind := range []string{"becomes-known", "never-known", "committed-opaque"} {
		t.Run(kind, func(t *testing.T) {
			var frames atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					n := frames.Add(1)
					u, origin := "", "://"
					if kind == "becomes-known" && n >= 2 {
						u = "https://site.test/child"
						origin = "https://site.test"
					}
					if kind == "committed-opaque" {
						u = "about:srcdoc"
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "url": "https://site.test/", "securityOrigin": "https://site.test"}, "childFrames": []any{map[string]any{"frame": map[string]any{"id": "child", "url": u, "securityOrigin": origin}}}}}
				case "Accessibility.getFullAXTree":
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 1, "role": map[string]any{"value": "heading"}, "name": map[string]any{"value": "ready"}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			sn, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"}, domain.BrowserRequest{WaitFor: "text", Contains: "ready"})
			if kind == "becomes-known" {
				if e != nil || sn == nil || frames.Load() < 2 {
					t.Fatalf("did not reobserve: %v", e)
				}
			} else {
				if e == nil || sn != nil {
					t.Fatal("unproven origin produced evidence")
				}
				if kind == "committed-opaque" && frames.Load() != 1 {
					t.Fatal("committed opaque frame was retried")
				}
			}
		})
	}
}

func TestSnapshotDiscardsAXWhenFrameProofChanges(t *testing.T) {
	for _, kind := range []string{"loader-origin", "parent-topology", "oopif"} {
		t.Run(kind, func(t *testing.T) {
			var changed atomic.Bool
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					child := map[string]any{"id": "child", "loaderId": "approved", "url": "https://site.test/child", "securityOrigin": "https://site.test"}
					if changed.Load() && kind == "loader-origin" {
						child["loaderId"] = "new"
						child["securityOrigin"] = "https://foreign.test"
					}
					children := []any{map[string]any{"frame": child}}
					if changed.Load() && kind == "parent-topology" {
						children = []any{map[string]any{"frame": map[string]any{"id": "inserted-parent", "url": "https://site.test/", "securityOrigin": "https://site.test"}, "childFrames": children}}
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "url": "https://site.test/", "securityOrigin": "https://site.test"}, "childFrames": children}}
				case "Target.getTargets":
					if changed.Load() && kind == "oopif" {
						return map[string]any{"targetInfos": []any{map[string]any{"type": "iframe", "parentId": "main", "parentFrameId": "main"}}}
					}
					return map[string]any{}
				case "Accessibility.getFullAXTree":
					changed.Store(true)
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 1, "role": map[string]any{"value": "heading"}, "name": map[string]any{"value": "unapproved-new-document-text"}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			sn, e := snapshot(context.Background(), c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"})
			if e == nil || sn != nil {
				t.Fatal("published AX under stale origin/loader/topology proof")
			}
		})
	}
}

func TestWaitRetriesFrameChangesButNeverPublishesPartialEvidence(t *testing.T) {
	for _, continuous := range []bool{false, true} {
		t.Run(map[bool]string{false: "settles", true: "deadline"}[continuous], func(t *testing.T) {
			var version atomic.Int32
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "loaderId": strings.Repeat("x", int(version.Load())+1), "url": "https://site.test/", "securityOrigin": "https://site.test"}}}
				case "Accessibility.getFullAXTree":
					old := version.Load()
					if continuous || old == 0 {
						version.Add(1)
					}
					name := "stable"
					if old == 0 || continuous {
						name = "partial"
					}
					return map[string]any{"nodes": []any{map[string]any{"backendDOMNodeId": 1, "role": map[string]any{"value": "heading"}, "name": map[string]any{"value": name}}}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			sn, e := wait(ctx, c, "s", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"}, domain.BrowserRequest{WaitFor: "text", Contains: "stable"})
			if continuous {
				if e == nil || sn != nil {
					t.Fatal("continuous navigation produced evidence")
				}
			} else if e != nil || sn == nil || sn.Nodes[0].Name != "stable" {
				t.Fatalf("did not wait for proved stable frame: %v", e)
			}
		})
	}
}

func TestDOMSnapshotOriginAndDocumentProof(t *testing.T) {
	for _, kind := range []string{"same-origin", "cross-origin", "opaque", "navigation", "oopif", "unapproved-document", "missing-frame"} {
		t.Run(kind, func(t *testing.T) {
			var captured atomic.Bool
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					origin := "https://site.test"
					if kind == "cross-origin" || (kind == "navigation" && captured.Load()) {
						origin = "https://foreign.test"
					}
					if kind == "opaque" {
						origin = "://"
					}
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "url": "https://site.test/", "securityOrigin": "https://site.test"}, "childFrames": []any{map[string]any{"frame": map[string]any{"id": "child", "url": "https://site.test/frame", "securityOrigin": origin}}}}}
				case "Target.getTargets":
					if kind == "oopif" && captured.Load() {
						return map[string]any{"targetInfos": []any{map[string]any{"type": "iframe", "parentId": "main"}}}
					}
					return map[string]any{}
				case "DOMSnapshot.captureSnapshot":
					captured.Store(true)
					frame := "child"
					if kind == "unapproved-document" {
						frame = "foreign"
					}
					doc := map[string]any{"frameId": 1, "nodes": map[string]any{"nodeType": []int{1}, "nodeName": []int{0}}}
					if kind == "missing-frame" {
						delete(doc, "frameId")
					}
					return map[string]any{"strings": []string{"DIV", frame}, "documents": []any{doc}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			raw, _, err := domSnapshot(context.Background(), c, "s")
			if kind == "same-origin" {
				if err != nil || raw == nil {
					t.Fatalf("same-origin rejected: %v", err)
				}
			} else if err == nil || raw != nil {
				t.Fatal("unapproved DOM evidence published")
			}
		})
	}
}

func TestUnparentedIframeTargetsArePageScoped(t *testing.T) {
	for _, related := range []bool{false, true} {
		t.Run(map[bool]string{false: "other-tab", true: "selected-tab"}[related], func(t *testing.T) {
			c, done := mockBrowser(t, func(q envelope) any {
				switch q.Method {
				case "Page.getFrameTree":
					return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": "main", "url": "https://site.test/", "securityOrigin": "https://site.test"}}}
				case "Target.getTargets":
					return map[string]any{"targetInfos": []any{map[string]any{"targetId": "oopif", "type": "iframe"}}}
				case "DOM.getDocument":
					if q.Session != "selected-session" {
						t.Error("frame owner census escaped selected page session")
					}
					children := []any{}
					if related {
						children = append(children, map[string]any{"nodeType": 1, "nodeName": "IFRAME", "frameId": "oopif"})
					}
					return map[string]any{"root": map[string]any{"nodeType": 9, "children": children}}
				default:
					return map[string]any{}
				}
			})
			defer done()
			sn, err := snapshot(context.Background(), c, "selected-session", domain.BrowserIdentity{}, domain.BrowserPage{ID: "main"})
			if related {
				if err == nil || sn != nil {
					t.Fatal("selected OOPIF accepted")
				}
			} else if err != nil || sn == nil {
				t.Fatalf("unrelated OOPIF blocked selected page: %v", err)
			}
		})
	}
}
