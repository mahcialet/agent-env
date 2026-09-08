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
		URLFragment    string
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
	h := sha256.Sum256([]byte(tree.Frame.URL + tree.Frame.URLFragment))
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

// Prefer browser-reported serialized origins over document URLs: sandboxing
// can make a same-URL document opaque. Chromium inherited-origin placeholders
// require the separate native same-origin access proof below, never a URL guess.
var errIncompleteFrameOrigin = errors.New("iframe origin is not yet available")
var errFrameObservationChanged = errors.New("frame identity changed during semantic observation")

func serializedOrigin(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Opaque != "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	return u.Scheme + "://" + u.Host, true
}

// Chromium can report "://" for an inherited about:blank document. Ask its
// own same-origin access check instead of inferring permission from that URL.
// The isolated parent world has no universal access and cannot see page-script
// overrides of the native contentDocument getter. Sandboxed opaque frames yield
// null even though CDP itself could inspect them with debugger privileges.
func inheritedFrameAccess(ctx context.Context, c *connection, session, parent, child string) bool {
	if parent == "" {
		return false
	}
	var owner struct{ BackendNodeID int }
	if c.call(ctx, session, "DOM.getFrameOwner", map[string]any{"frameId": child}, &owner) != nil || owner.BackendNodeID <= 0 {
		return false
	}
	var world struct{ ExecutionContextID int }
	if c.call(ctx, session, "Page.createIsolatedWorld", map[string]any{"frameId": parent, "worldName": "agent-env-origin-check", "grantUniveralAccess": false}, &world) != nil || world.ExecutionContextID <= 0 {
		return false
	}
	var resolved struct{ Object struct{ ObjectID string } }
	if c.call(ctx, session, "DOM.resolveNode", map[string]any{"backendNodeId": owner.BackendNodeID, "executionContextId": world.ExecutionContextID}, &resolved) != nil || resolved.Object.ObjectID == "" {
		return false
	}
	defer c.call(ctx, session, "Runtime.releaseObject", map[string]any{"objectId": resolved.Object.ObjectID}, nil)
	var result struct {
		Result           struct{ Value bool }
		ExceptionDetails json.RawMessage
	}
	err := c.call(ctx, session, "Runtime.callFunctionOn", map[string]any{"objectId": resolved.Object.ObjectID, "functionDeclaration": "function(){const p=this instanceof HTMLIFrameElement?HTMLIFrameElement.prototype:this instanceof HTMLFrameElement?HTMLFrameElement.prototype:null;return !!p && Object.getOwnPropertyDescriptor(p,'contentDocument').get.call(this)!==null;}", "returnByValue": true}, &result)
	return err == nil && len(result.ExceptionDetails) == 0 && result.Result.Value
}

func approvedFrames(ctx context.Context, c *connection, s string, tree frameTree) ([]frameTree, error) {
	var frames []frameTree
	var walk func(frameTree, string) error
	rootOrigin, rootKnown := serializedOrigin(tree.Frame.SecurityOrigin)
	walk = func(f frameTree, parent string) error {
		if f.Frame.ID != tree.Frame.ID {
			childOrigin, childKnown := serializedOrigin(f.Frame.SecurityOrigin)
			if !childKnown && f.Frame.URL == "" {
				return errIncompleteFrameOrigin
			}
			inherited := false
			if rootKnown && !childKnown && (f.Frame.URL == "about:blank" || f.Frame.URL == "about:srcdoc") {
				inherited = inheritedFrameAccess(ctx, c, s, parent, f.Frame.ID)
			}
			if !rootKnown || (!inherited && (!childKnown || childOrigin != rootOrigin)) {
				return errors.New("cross-origin or opaque iframe observation is unsupported")
			}
		}
		if len(frames) >= 32 {
			return errors.New("browser frame limit exceeded")
		}
		frames = append(frames, f)
		for _, ch := range f.ChildFrames {
			if e := walk(ch, f.Frame.ID); e != nil {
				return e
			}
		}
		return nil
	}
	if e := walk(tree, ""); e != nil {
		return nil, e
	}
	return frames, nil
}

func recheckFrames(ctx context.Context, c *connection, s string, tree frameTree, page string) error {
	latest, e := frameDocument(ctx, c, s)
	if e != nil {
		return e
	}
	before, _ := json.Marshal(tree)
	after, _ := json.Marshal(latest)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		return errFrameObservationChanged
	}
	frames, e := approvedFrames(ctx, c, s, latest)
	if e != nil {
		return e
	}
	return validateFrameTargets(ctx, c, s, frames, page)
}

func snapshot(ctx context.Context, c *connection, s string, id domain.BrowserIdentity, p domain.BrowserPage) (*domain.BrowserSnapshot, error) {
	tree, e := frameDocument(ctx, c, s)
	if e != nil {
		return nil, e
	}
	p.URL = scrubURL(tree.Frame.URL)

	sn := &domain.BrowserSnapshot{Version: 1, Identity: id, Page: p, Document: documentIdentity(tree), CapturedAt: time.Now().UTC(), Nodes: []domain.BrowserNode{}}
	frames, e := approvedFrames(ctx, c, s, tree)
	if e != nil {
		return nil, e
	}
	if e = validateFrameTargets(ctx, c, s, frames, p.ID); e != nil {
		return nil, e
	}
	if e = c.call(ctx, s, "Accessibility.enable", nil, nil); e != nil {
		return nil, e
	}
frameNodes:
	for _, f := range frames {
		var x struct{ Nodes []axNode }
		if e = c.call(ctx, s, "Accessibility.getFullAXTree", map[string]any{"frameId": f.Frame.ID}, &x); e != nil {
			return nil, errors.New("iframe accessibility session unsupported")
		}
		for _, a := range x.Nodes {
			// Reaching the cap is complete when no further node exists. Read
			// remaining frames until an actual omitted node proves truncation.
			if len(sn.Nodes) == 2048 {
				sn.Truncated = true
				break frameNodes
			}
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
	if e = recheckFrames(ctx, c, s, tree, p.ID); e != nil {
		return nil, e
	}
	return sn, nil
}

// DOM evidence intentionally excludes every text/attribute value and input value.
// It retains bounded structure and layout; semantic labels belong to AX evidence.
func domSnapshot(ctx context.Context, c *connection, s string) ([]byte, bool, error) {
	tree, e := frameDocument(ctx, c, s)
	if e != nil {
		return nil, false, e
	}
	frames, e := approvedFrames(ctx, c, s, tree)
	if e != nil {
		return nil, false, e
	}
	if e = validateFrameTargets(ctx, c, s, frames, tree.Frame.ID); e != nil {
		return nil, false, e
	}
	approved := map[string]bool{}
	for _, f := range frames {
		approved[f.Frame.ID] = true
	}

	var x struct {
		Strings   []string
		Documents []struct {
			FrameID *int
			Nodes   struct {
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
	for _, d := range x.Documents {
		if d.FrameID == nil || *d.FrameID < 0 || *d.FrameID >= len(x.Strings) || !approved[x.Strings[*d.FrameID]] {
			return nil, false, errors.New("DOM document frame is not approved")
		}
	}
	if e = recheckFrames(ctx, c, s, tree, tree.Frame.ID); e != nil {
		return nil, false, e
	}
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
				out.Truncated = true
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

func validateFrameTargets(ctx context.Context, c *connection, s string, frames []frameTree, page string) error {
	var targets struct {
		TargetInfos []struct {
			Type          string
			TargetID      string
			ParentID      string
			ParentFrameID string
		}
	}
	if e := c.call(ctx, "", "Target.getTargets", nil, &targets); e != nil {
		return e
	}
	if len(targets.TargetInfos) > 512 {
		return errors.New("browser target census limit exceeded")
	}
	frameIDs := map[string]bool{}
	for _, f := range frames {
		frameIDs[f.Frame.ID] = true
	}
	var owners map[string]bool
	for _, t := range targets.TargetInfos {
		if t.Type == "iframe" && t.ParentID == "" && t.ParentFrameID == "" {
			var e error
			owners, e = frameOwners(ctx, c, s)
			if e != nil {
				return e
			}
			break
		}
	}
	for _, t := range targets.TargetInfos {
		if t.Type == "iframe" && (t.ParentID == page || frameIDs[t.ParentFrameID] || owners[t.TargetID]) {
			return errors.New("cross-origin or opaque iframe observation is unsupported: out-of-process frame")
		}
	}
	return nil
}

// Frame-owner elements belong to this page session, unlike the browser-wide
// target list. Chrome may omit parent metadata for OOPIF targets; correlate
// their target IDs with these frame IDs instead of blocking unrelated tabs.
func frameOwners(ctx context.Context, c *connection, s string) (map[string]bool, error) {
	type domNode struct {
		NodeType        int
		FrameID         string
		Children        []domNode
		ShadowRoots     []domNode
		ContentDocument *domNode
		TemplateContent *domNode
	}
	var result struct{ Root *domNode }
	if e := c.call(ctx, s, "DOM.getDocument", map[string]any{"depth": -1, "pierce": true}, &result); e != nil {
		return nil, e
	}
	owners := map[string]bool{}
	pending := []*domNode{result.Root}
	count := 0
	for len(pending) > 0 {
		n := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if n == nil || n.NodeType == 0 {
			return nil, errors.New("page frame-owner census unavailable")
		}
		count++
		if count > 65536 {
			return nil, errors.New("page frame-owner census limit exceeded")
		}
		if n.FrameID != "" {
			owners[n.FrameID] = true
		}
		for i := range n.Children {
			pending = append(pending, &n.Children[i])
		}
		for i := range n.ShadowRoots {
			pending = append(pending, &n.ShadowRoots[i])
		}
		if n.ContentDocument != nil {
			pending = append(pending, n.ContentDocument)
		}
		if n.TemplateContent != nil {
			pending = append(pending, n.TemplateContent)
		}
	}
	return owners, nil
}
