package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"

	"github.com/mahcialet/agent-env/internal/domain"
)

func act(ctx context.Context, c *connection, s string, id domain.BrowserIdentity, p domain.BrowserPage, q domain.BrowserRequest, verify func() error) (performed, readback bool, err error) {
	if q.Prior == nil || q.Node == "" {
		return false, false, errors.New("semantic input requires snapshot and node")
	}
	if q.Prior.Truncated {
		return false, false, errors.New("truncated snapshot cannot authorize input")
	}
	var old *domain.BrowserNode
	for i := range q.Prior.Nodes {
		if q.Prior.Nodes[i].Ref == q.Node {
			if old != nil {
				return false, false, errors.New("ambiguous snapshot reference")
			}
			old = &q.Prior.Nodes[i]
		}
	}
	if old == nil || old.BackendID <= 0 || old.Ignored || old.Disabled {
		return false, false, errors.New("node is diagnostic-only or unavailable")
	}
	fresh, e := snapshot(ctx, c, s, id, p)
	if e != nil {
		return false, false, e
	}
	if fresh.Truncated {
		return false, false, errors.New("truncated fresh snapshot cannot authorize input")
	}
	if fresh.Document != q.Prior.Document {
		return false, false, errors.New("stale document")
	}
	if !uniqueFreshNode(fresh.Nodes, *old) {
		return false, false, errors.New("stale or ambiguous semantic node")
	}
	tree, e := frameDocument(ctx, c, s)
	if e != nil {
		return false, false, e
	}
	if old.Frame != tree.Frame.ID {
		return false, false, errors.New("iframe input is unsupported")
	}
	resolveParams := map[string]any{"backendNodeId": old.BackendID}
	{
		var world struct {
			ExecutionContextID int `json:"executionContextId"`
		}
		if e = c.call(ctx, s, "Page.createIsolatedWorld", map[string]any{"frameId": old.Frame, "worldName": "agent-env-input", "grantUniveralAccess": false}, &world); e != nil {
			return false, false, e
		}
		if world.ExecutionContextID == 0 {
			return false, false, errors.New("input verification context unavailable")
		}
		resolveParams["executionContextId"] = world.ExecutionContextID
	}
	var resolved struct {
		Object struct {
			ObjectID string `json:"objectId"`
		}
	}
	if e = c.call(ctx, s, "DOM.resolveNode", resolveParams, &resolved); e != nil {
		return false, false, e
	}
	object := resolved.Object.ObjectID
	if object == "" {
		return false, false, errors.New("node cannot be resolved")
	}
	defer c.call(ctx, s, "Runtime.releaseObject", map[string]any{"objectId": object}, nil)
	x, y, e := nodeHit(ctx, c, s, object, old.BackendID)
	if e != nil {
		return false, false, e
	}
	if q.Operation == "set-text" && !old.Editable {
		return false, false, errors.New("node is not editable")
	}
	if q.Operation == "key" && !allowedKey(q.Key) {
		return false, false, errors.New("unsupported key")
	}
	if e = verify(); e != nil {
		return false, false, e
	}
	latest, e := frameDocument(ctx, c, s)
	if e != nil {
		return false, false, e
	}
	if documentIdentity(latest) != fresh.Document {
		return false, false, errors.New("document changed before input")
	}
	final, e := snapshot(ctx, c, s, id, p)
	if e != nil {
		return false, false, e
	}
	if final.Document != fresh.Document || final.Truncated {
		return false, false, errors.New("document changed before input")
	}
	if !uniqueFreshNode(final.Nodes, *old) {
		return false, false, errors.New("semantic node changed during ownership verification")
	}
	x, y, e = nodeHit(ctx, c, s, object, old.BackendID)
	if e != nil {
		return false, false, e
	}
	performed = true
	switch q.Operation {
	case "click":
		e = c.call(ctx, s, "Input.dispatchMouseEvent", map[string]any{"type": "mousePressed", "x": x, "y": y, "button": "left", "clickCount": 1}, nil)
		if e == nil {
			e = c.call(ctx, s, "Input.dispatchMouseEvent", map[string]any{"type": "mouseReleased", "x": x, "y": y, "button": "left", "clickCount": 1}, nil)
		}
	case "scroll":
		e = c.call(ctx, s, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": x, "y": y, "deltaX": q.DeltaX, "deltaY": q.DeltaY}, nil)
	case "key", "set-text":
		// Activate the selected page before focusing its control: background pages
		// can retain activeElement without delivering synchronous focus handlers.
		if e = c.call(ctx, s, "Page.bringToFront", nil, nil); e != nil {
			return true, false, e
		}

		activated, activationErr := snapshot(ctx, c, s, id, p)
		if activationErr != nil {
			return true, false, activationErr
		}
		if activated.Document != fresh.Document || activated.Truncated || !uniqueFreshNode(activated.Nodes, *old) {
			return true, false, errors.New("semantic node changed during page activation")
		}
		if _, _, e = nodeHit(ctx, c, s, object, old.BackendID); e != nil {
			return true, false, e
		}
		e = c.call(ctx, s, "DOM.focus", map[string]any{"backendNodeId": old.BackendID}, nil)
		if e != nil {
			return true, false, e
		}
		if e = verifyFocus(ctx, c, s, object); e != nil {
			return true, false, e
		}
		if q.Operation == "key" {
			e = key(ctx, c, s, q.Key, 0)
		} else {
			e = selectAll(ctx, c, s, runtime.GOOS)
			if e == nil {
				e = verifyFocus(ctx, c, s, object)
			}
			if e == nil {
				if q.Text == "" {
					e = key(ctx, c, s, "Backspace", 0)
				} else {
					e = c.call(ctx, s, "Input.insertText", map[string]any{"text": q.Text}, nil)
				}
			}
			if e == nil {
				var rb struct {
					Result           struct{ Value *bool }
					ExceptionDetails json.RawMessage
				}
				e = c.call(ctx, s, "Runtime.callFunctionOn", map[string]any{"objectId": object, "functionDeclaration": "function(expected){return this.isConnected && (typeof this.value==='string'?this.value:this.textContent)===expected;}", "arguments": []map[string]any{{"value": q.Text}}, "returnByValue": true}, &rb)
				if e == nil && (len(rb.ExceptionDetails) > 0 || rb.Result.Value == nil) {
					e = errors.New("text replacement readback unavailable")
				}
				readback = e == nil && *rb.Result.Value
				if e == nil && !readback {
					e = confirmedError{errors.New("text replacement readback differs")}
				}
			}
		}
	}
	return performed, readback, e
}
func allowedKey(k string) bool {
	switch k {
	case "Enter", "Tab", "Escape", "Backspace", "Delete", "ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown", "Space":
		return true
	}
	return false
}

// Chromium on macOS does not reliably translate a synthetic Meta+A into a
// native Cocoa editing command. CDP's explicit command keeps selection in the
// browser input pipeline on every OS without a JavaScript value assignment.
func selectAll(ctx context.Context, c *connection, s, platform string) error {
	modifier := 2
	if platform == "darwin" {
		modifier = 4
	}
	p := map[string]any{"type": "rawKeyDown", "key": "a", "code": "KeyA", "windowsVirtualKeyCode": 65, "modifiers": modifier, "commands": []string{"selectAll"}}
	if e := c.call(ctx, s, "Input.dispatchKeyEvent", p, nil); e != nil {
		return e
	}
	delete(p, "commands")
	p["type"] = "keyUp"
	return c.call(ctx, s, "Input.dispatchKeyEvent", p, nil)
}

func key(ctx context.Context, c *connection, s, k string, mod int) error {
	codes := map[string]int{"Enter": 13, "Tab": 9, "Escape": 27, "Backspace": 8, "Delete": 46, "ArrowLeft": 37, "ArrowRight": 39, "ArrowUp": 38, "ArrowDown": 40, "Home": 36, "End": 35, "PageUp": 33, "PageDown": 34, "Space": 32, "a": 65}
	p := map[string]any{"type": "keyDown", "key": k, "windowsVirtualKeyCode": codes[k], "modifiers": mod}
	if e := c.call(ctx, s, "Input.dispatchKeyEvent", p, nil); e != nil {
		return e
	}
	p["type"] = "keyUp"
	return c.call(ctx, s, "Input.dispatchKeyEvent", p, nil)
}

func nodeHit(ctx context.Context, c *connection, s, object string, backend int) (float64, float64, error) {
	var box struct{ Model struct{ Content []float64 } }
	if e := c.call(ctx, s, "DOM.getBoxModel", map[string]any{"backendNodeId": backend}, &box); e != nil {
		return 0, 0, e
	}
	if len(box.Model.Content) != 8 {
		return 0, 0, errors.New("node has no current layout")
	}
	x := (box.Model.Content[0] + box.Model.Content[4]) / 2
	y := (box.Model.Content[1] + box.Model.Content[5]) / 2
	// Check every enclosing root from the target outward. Closed shadow roots
	// are reachable from their children even when host.shadowRoot is null.
	// Each enclosing host must itself pass hit testing, so overlays still refuse.
	var hit struct {
		Result           struct{ Value bool }
		ExceptionDetails json.RawMessage
	}
	if e := c.call(ctx, s, "Runtime.callFunctionOn", map[string]any{"objectId": object, "functionDeclaration": `function(x,y){
if(!this.isConnected)return false;
let node=this;
for(let depth=0;depth<128;depth++){
 const root=node.getRootNode();
 const hit=root.elementFromPoint(x,y);
 if(!hit || (hit!==node && !node.contains(hit)))return false;
 if(root===this.ownerDocument)return true;
 if(!(root instanceof ShadowRoot))return false;
 node=root.host;
}
return false;
}`, "arguments": []map[string]any{{"value": x}, {"value": y}}, "returnByValue": true}, &hit); e != nil {
		return 0, 0, e
	}
	if len(hit.ExceptionDetails) > 0 || !hit.Result.Value {
		return 0, 0, errors.New("node is obscured or outside the viewport")
	}
	return x, y, nil
}

func uniqueFreshNode(nodes []domain.BrowserNode, old domain.BrowserNode) bool {
	var candidate domain.BrowserNode
	matches := 0
	for _, n := range nodes {
		if n.Frame == old.Frame && n.BackendID == old.BackendID && n.Fingerprint == old.Fingerprint {
			candidate = n
			matches++
		}
	}
	if matches != 1 {
		return false
	}
	semantic := 0
	for _, n := range nodes {
		if n.Frame == candidate.Frame && n.Role == candidate.Role && n.Name == candidate.Name && !n.Ignored {
			semantic++
		}
	}
	return semantic == 1
}

// verifyFocus runs in an isolated world so page overrides cannot forge the
// active-element or connectivity checks. Walk outward from the resolved target:
// getRootNode exposes its closed roots without relying on host.shadowRoot.
func verifyFocus(ctx context.Context, c *connection, session, object string) error {
	var x struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	e := c.call(ctx, session, "Runtime.callFunctionOn", map[string]any{"objectId": object, "functionDeclaration": `function(){
if(!this.isConnected||!this.ownerDocument.hasFocus())return false;
let node=this;
for(let depth=0;depth<128;depth++){
 const root=node.getRootNode();
 if(root.activeElement!==node)return false;
 if(root===this.ownerDocument)return true;
 if(!(root instanceof ShadowRoot))return false;
 node=root.host;
}
return false;
}`, "returnByValue": true}, &x)
	if e != nil {
		return e
	}
	if len(x.ExceptionDetails) > 0 || !x.Result.Value {
		return errors.New("input target no longer has focus")
	}
	return nil
}
