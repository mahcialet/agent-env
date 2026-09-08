package cdp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type frameTree struct {
	Frame struct {
		ID             string
		LoaderID       string
		URL            string
		SecurityOrigin string
	}
	ChildFrames []frameTree
}
type axValue struct {
	Type  string
	Value any
}
type axNode struct {
	NodeID           string
	Ignored          bool
	Role             axValue
	Name             axValue
	Value            axValue
	BackendDOMNodeID int
	FrameID          string
	Properties       []struct {
		Name  string
		Value axValue
	}
}

func frameDocument(ctx context.Context, c *connection, s string) (frameTree, error) {
	var x struct{ FrameTree frameTree }
	e := c.call(ctx, s, "Page.getFrameTree", nil, &x)
	return x.FrameTree, e
}
func valueString(v axValue) string {
	if s, ok := v.Value.(string); ok {
		return s
	}
	return ""
}
func fingerprint(n domain.BrowserNode) string {
	n.Ref = ""
	n.Value = ""
	n.Fingerprint = ""
	raw, _ := json.Marshal(n)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func documentIdentity(tree frameTree) string {
	h := sha256.Sum256([]byte(tree.Frame.URL))
	return tree.Frame.ID + ":" + tree.Frame.LoaderID + ":" + hex.EncodeToString(h[:])
}
func axState(v any) (string, bool) {
	switch x := v.(type) {
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case string:
		switch x {
		case "true", "false", "mixed":
			return x, true
		}
	}
	return "", false
}
func snapshot(ctx context.Context, c *connection, s string, id domain.BrowserIdentity, p domain.BrowserPage) (*domain.BrowserSnapshot, error) {
	tree, e := frameDocument(ctx, c, s)
	if e != nil {
		return nil, e
	}
	p.URL = scrubURL(tree.Frame.URL)
	sn := &domain.BrowserSnapshot{Version: 1, Identity: id, Page: p, Document: documentIdentity(tree), CapturedAt: time.Now().UTC(), Nodes: []domain.BrowserNode{}}
	var frames []frameTree
	var walk func(frameTree) error
	rootURL, _ := url.Parse(tree.Frame.URL)
	walk = func(f frameTree) error {
		u, _ := url.Parse(f.Frame.URL)
		if f.Frame.ID != tree.Frame.ID && u != nil && u.Scheme != "about" && rootURL != nil && (u.Scheme != rootURL.Scheme || u.Host != rootURL.Host) {
			return errors.New("cross-origin iframe observation is unsupported")
		}
		frames = append(frames, f)
		for _, ch := range f.ChildFrames {
			if e := walk(ch); e != nil {
				return e
			}
		}
		return nil
	}
	if e = walk(tree); e != nil {
		return nil, e
	}
	if len(frames) > 32 {
		return nil, errors.New("browser frame limit exceeded")
	}
	if e = c.call(ctx, s, "Accessibility.enable", nil, nil); e != nil {
		return nil, e
	}
	for _, f := range frames {
		var x struct{ Nodes []axNode }
		if e = c.call(ctx, s, "Accessibility.getFullAXTree", map[string]any{"frameId": f.Frame.ID}, &x); e != nil {
			return nil, errors.New("iframe accessibility session unsupported")
		}
		for _, a := range x.Nodes {
			n := domain.BrowserNode{BackendID: a.BackendDOMNodeID, Frame: f.Frame.ID, Role: valueString(a.Role), Name: valueString(a.Name), Ignored: a.Ignored}
			for _, v := range a.Properties {
				switch v.Name {
				case "checked", "selected", "expanded", "readonly", "required", "focusable", "focused", "multiselectable":
					if state, ok := axState(v.Value.Value); ok {
						if n.States == nil {
							n.States = map[string]string{}
						}
						n.States[v.Name] = state
					}
				case "disabled":
					n.Disabled = v.Value.Value == true
				case "editable":
					n.Editable = v.Value.Value != nil && v.Value.Value != false
				case "password", "protected":
					n.Password = v.Value.Value == true
				}
			}
			n.Password = n.Password || strings.Contains(strings.ToLower(n.Role), "password")
			if n.Role == "textbox" || n.Role == "searchbox" {
				n.Editable = true
				if n.BackendID > 0 {
					var d struct{ Node struct{ Attributes []string } }
					if e = c.call(ctx, s, "DOM.describeNode", map[string]any{"backendNodeId": n.BackendID}, &d); e != nil {
						return nil, errors.New("input privacy classification unavailable")
					}
					for i := 0; i+1 < len(d.Node.Attributes); i += 2 {
						if strings.EqualFold(d.Node.Attributes[i], "type") && strings.EqualFold(d.Node.Attributes[i+1], "password") {
							n.Password = true
						}
					}
				}
			}
			if !n.Editable && !n.Password {
				n.Value = valueString(a.Value)
			}
			if n.Password {
				n.Value = ""
			}
			if len(n.Name) > 4096 {
				n.Name = "[TRUNCATED]"
				sn.Truncated = true
			}
			if len(n.Value) > 4096 {
				n.Value = "[TRUNCATED]"
				sn.Truncated = true
			}
			n.Fingerprint = fingerprint(n)
			sn.Nodes = append(sn.Nodes, n)
			if len(sn.Nodes) >= 2048 {
				sn.Truncated = true
				break
			}
		}
		if sn.Truncated {
			break
		}
	}
	sort.SliceStable(sn.Nodes, func(i, j int) bool {
		a, b := sn.Nodes[i], sn.Nodes[j]
		if a.Frame != b.Frame {
			return a.Frame < b.Frame
		}
		if a.BackendID != b.BackendID {
			return a.BackendID < b.BackendID
		}
		return a.Fingerprint < b.Fingerprint
	})
	for i := range sn.Nodes {
		sn.Nodes[i].Ref = fmt.Sprintf("n%d", i+1)
	}
	raw, _ := json.Marshal(sn)
	for len(raw) > 1<<20 && len(sn.Nodes) > 0 {
		sn.Truncated = true
		sn.Nodes = sn.Nodes[:len(sn.Nodes)/2]
		raw, _ = json.Marshal(sn)
	}
	if len(raw) > 1<<20 {
		return nil, errors.New("semantic snapshot metadata exceeds byte limit")
	}
	return sn, nil
}

// DOM evidence intentionally excludes every text/attribute value and input value.
// It retains bounded structure and layout; semantic labels belong to AX evidence.
func domSnapshot(ctx context.Context, c *connection, s string) ([]byte, bool, error) {
	var x struct {
		Strings   []string
		Documents []struct {
			Nodes struct {
				NodeType      []int
				NodeName      []int
				ParentIndex   []int
				BackendNodeID []int
			}
			Layout struct {
				NodeIndex []int
				Bounds    [][]float64
			}
		}
	}
	if e := c.call(ctx, s, "DOMSnapshot.captureSnapshot", map[string]any{"computedStyles": []string{}, "includeDOMRects": true}, &x); e != nil {
		return nil, false, e
	}
	type node struct {
		Document  int       `json:"document"`
		Index     int       `json:"index"`
		Parent    int       `json:"parent"`
		BackendID int       `json:"backend_id"`
		Type      int       `json:"type"`
		Name      string    `json:"name"`
		Bounds    []float64 `json:"bounds,omitempty"`
	}
	out := struct {
		Version   int    `json:"version"`
		Nodes     []node `json:"nodes"`
		Truncated bool   `json:"truncated"`
	}{Version: 1, Nodes: []node{}}
	for di, d := range x.Documents {
		bounds := map[int][]float64{}
		for i, n := range d.Layout.NodeIndex {
			if i < len(d.Layout.Bounds) {
				bounds[n] = d.Layout.Bounds[i]
			}
		}
		for i, t := range d.Nodes.NodeType {
			if len(out.Nodes) >= 2048 {
				out.Truncated = true
				break
			}
			n := node{Document: di, Index: i, Type: t, Bounds: bounds[i]}
			if i < len(d.Nodes.ParentIndex) {
				n.Parent = d.Nodes.ParentIndex[i]
			}
			if i < len(d.Nodes.BackendNodeID) {
				n.BackendID = d.Nodes.BackendNodeID[i]
			}
			if i < len(d.Nodes.NodeName) {
				j := d.Nodes.NodeName[i]
				if j >= 0 && j < len(x.Strings) {
					n.Name = x.Strings[j]
				}
			}
			if len(n.Name) > 128 {
				n.Name = "[truncated]"
			}
			out.Nodes = append(out.Nodes, n)
		}
		if out.Truncated {
			break
		}
	}
	raw, e := json.Marshal(out)
	if len(raw) > 1<<20 {
		return nil, false, errors.New("DOM snapshot byte limit exceeded")
	}
	return raw, out.Truncated, e
}
