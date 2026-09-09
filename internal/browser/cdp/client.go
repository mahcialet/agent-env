package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type Client struct{}

const maxBrowserPages = 128

type target struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
}

func scrubURL(s string) string {
	u, e := url.Parse(s)
	if e != nil {
		return "[invalid URL]"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
func safeURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && u.User == nil && ((u.Scheme == "http" || u.Scheme == "https") && u.Host != "" || s == "about:blank")
}
func connect(ctx context.Context, r domain.Runtime, b domain.BrowserBinding, verify func(context.Context) error) (*connection, domain.BrowserIdentity, error) {
	var id domain.BrowserIdentity
	if verify == nil {
		return nil, id, errors.New("native ownership verifier required")
	}
	if e := verify(ctx); e != nil {
		return nil, id, e
	}
	p := r.Process
	if p == nil {
		return nil, id, errors.New("process identity missing")
	}
	port := p.Ports[b.CDPPort]
	if port < 1 || port > 65535 || p.ProcessID < 1 || p.ProcessStart == "" {
		return nil, id, errors.New("invalid browser ownership")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://127.0.0.1:%d/json/version", port), nil)
	if e != nil {
		return nil, id, e
	}
	resp, e := privateHTTP().Do(req)
	if e != nil {
		return nil, id, errors.New("browser discovery unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, id, errors.New("browser discovery failed")
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if e != nil || len(raw) > 65536 {
		return nil, id, errors.New("browser discovery exceeds limit")
	}
	var d struct {
		Browser   string
		Protocol  string `json:"Protocol-Version"`
		WebSocket string `json:"webSocketDebuggerUrl"`
	}
	if json.Unmarshal(raw, &d) != nil || !validEndpoint(d.WebSocket, port) {
		return nil, id, errors.New("foreign or malformed browser websocket")
	}
	c, e := dial(ctx, d.WebSocket)
	if e != nil {
		return nil, id, e
	}
	fail := func(e error) (*connection, domain.BrowserIdentity, error) { c.close(); return nil, id, e }
	var v struct {
		Product  string
		Protocol string `json:"protocolVersion"`
	}
	if e = c.call(ctx, "", "Browser.getVersion", nil, &v); e != nil {
		return fail(e)
	}
	if v.Product != d.Browser || v.Protocol != d.Protocol || v.Product == "" || v.Protocol == "" {
		return fail(errors.New("browser version identity mismatch"))
	}
	var proc struct {
		ProcessInfo []struct {
			Type string
			ID   int
		}
	}
	if e = c.call(ctx, "", "SystemInfo.getProcessInfo", nil, &proc); e != nil {
		return fail(e)
	}
	matches := 0
	for _, x := range proc.ProcessInfo {
		if x.Type == "browser" {
			if x.ID != p.ProcessID {
				return fail(errors.New("browser PID does not match owned process root"))
			}
			matches++
		}
	}
	if matches != 1 {
		return fail(errors.New("browser PID identity unavailable"))
	}
	var cmd struct{ Arguments []string }
	if e = c.call(ctx, "", "Browser.getBrowserCommandLine", nil, &cmd); e != nil {
		return fail(e)
	}
	if e = validateFlags(cmd.Arguments, p.StateDirectory, port); e != nil {
		return fail(e)
	}
	if e = verify(ctx); e != nil {
		return fail(e)
	}
	id = domain.BrowserIdentity{Runtime: b.Runtime, PID: p.ProcessID, Birth: p.ProcessStart, Port: port, WebSocket: d.WebSocket, Product: v.Product, Protocol: v.Protocol}
	return c, id, nil
}
func validateFlags(args []string, state string, port int) error {
	want := map[string]string{"--enable-automation": "", "--headless": "new", "--remote-debugging-address": "127.0.0.1", "--remote-debugging-port": strconv.Itoa(port), "--user-data-dir": filepath.Join(state, "profile")}
	seen := map[string]bool{}
	for _, a := range args {
		k, v, _ := strings.Cut(a, "=")
		w, ok := want[k]
		if !ok {
			continue
		}
		matches := v == w
		if k == "--user-data-dir" {
			matches = filepath.IsAbs(v) && filepath.Clean(v) == filepath.Clean(w)
		}
		if seen[k] || !matches {
			return errors.New("browser required launch flags do not match private binding")
		}
		seen[k] = true
	}
	if len(seen) != len(want) {
		return errors.New("browser required launch flags missing")
	}
	root, e := filepath.EvalSymlinks(state)
	if e != nil {
		return errors.New("private browser state unavailable")
	}
	profile, e := filepath.EvalSymlinks(filepath.Join(state, "profile"))
	if e != nil {
		return errors.New("private browser profile unavailable")
	}
	rel, e := filepath.Rel(root, profile)
	if e != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("browser profile escapes private state")
	}
	return nil
}
func pages(ctx context.Context, c *connection) ([]domain.BrowserPage, error) {
	var x struct{ TargetInfos []target }
	if e := c.call(ctx, "", "Target.getTargets", nil, &x); e != nil {
		return nil, e
	}
	out := []domain.BrowserPage{}
	for _, t := range x.TargetInfos {
		if t.Type == "page" {
			out = append(out, domain.BrowserPage{ID: t.TargetID, URL: scrubURL(t.URL), Title: t.Title})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > maxBrowserPages {
		return nil, errors.New("browser page limit exceeded")
	}
	return out, nil
}
func (Client) Observe(ctx context.Context, r domain.Runtime, b domain.BrowserBinding, q domain.BrowserRequest, verify func(context.Context) error) (o domain.BrowserObservation, err error) {
	defer func() {
		var known confirmedError
		o.Confirmed = err == nil || !o.ActionPerformed || errors.As(err, &known)
	}()
	c, id, e := connect(ctx, r, b, verify)
	if e != nil {
		return o, e
	}
	defer c.close()
	o.Identity = id
	if q.Prior != nil && q.Prior.Identity != id {
		return o, errors.New("stale browser identity")
	}
	o.Pages, e = pages(ctx, c)
	if e != nil {
		return o, e
	}
	if q.Operation == "capabilities" || q.Operation == "pages" {
		if q.Operation == "capabilities" {
			o.Detail = "operations: pages,page-create,page-close,navigate,snapshot,dom-snapshot,screenshot,click,set-text,key,scroll,wait,console,network; headless Chromium with Browser/SystemInfo/Target/Page/Accessibility/DOM/DOMSnapshot/Input/Runtime/Network methods required; limits: 128 pages, 32 same-origin frames, 2048 nodes, 1 MiB snapshots, 256 capture records, 64 KiB capture text, 10 second captures; iframe input and cross-origin frame observation unsupported"
		}
		o.Confirmed = true
		return o, nil
	}
	mutate := func() error { return verify(ctx) }
	if q.Operation == "page-create" {
		if len(o.Pages) >= maxBrowserPages {
			return o, errors.New("browser page limit reached")
		}
		if !safeURL(q.URL) {
			return o, errors.New("unsupported navigation URL")
		}
		if e = mutate(); e != nil {
			return o, e
		}
		var x struct {
			TargetID string `json:"targetId"`
		}
		o.ActionPerformed = true
		e = c.call(ctx, "", "Target.createTarget", map[string]any{"url": q.URL}, &x)
		o.Page = domain.BrowserPage{ID: x.TargetID, URL: scrubURL(q.URL)}
		o.Confirmed = e == nil
		return o, e
	}
	selected := q.Page
	if q.Prior != nil {
		if selected != "" && selected != q.Prior.Page.ID {
			return o, errors.New("snapshot page mismatch")
		}
		selected = q.Prior.Page.ID
	}
	if selected == "" && len(o.Pages) == 1 {
		selected = o.Pages[0].ID
	}
	for _, p := range o.Pages {
		if p.ID == selected {
			o.Page = p
		}
	}
	if o.Page.ID == "" {
		return o, errors.New("select exactly one existing page")
	}
	if q.Operation == "page-close" {
		if e = mutate(); e != nil {
			return o, e
		}
		var x struct{ Success bool }
		o.ActionPerformed = true
		e = c.call(ctx, "", "Target.closeTarget", map[string]any{"targetId": selected}, &x)
		if e == nil && !x.Success {
			e = confirmedError{errors.New("page close not confirmed")}
		}
		o.Confirmed = e == nil
		return o, e
	}
	var attach struct {
		SessionID string `json:"sessionId"`
	}
	if e = c.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": selected, "flatten": true}, &attach); e != nil {
		return o, e
	}
	s := attach.SessionID
	if e = c.call(ctx, s, "Page.enable", nil, nil); e != nil {
		return o, e
	}
	switch q.Operation {
	case "snapshot":
		o.Snapshot, e = snapshot(ctx, c, s, id, o.Page)
	case "dom-snapshot":
		o.DOM, o.Truncated, e = domSnapshot(ctx, c, s)
	case "screenshot":
		o.PNG, e = screenshot(ctx, c, s)
	case "navigate":
		if !safeURL(q.URL) {
			return o, errors.New("unsupported navigation URL")
		}
		if e = mutate(); e != nil {
			return o, e
		}
		var x struct{ ErrorText string }
		o.ActionPerformed = true
		e = c.call(ctx, s, "Page.navigate", map[string]any{"url": q.URL}, &x)
		if e == nil && x.ErrorText != "" {
			e = confirmedError{errors.New("navigation failed")}
		}
	case "click", "set-text", "key", "scroll":
		o.ActionPerformed, o.ReadbackEqual, e = act(ctx, c, s, id, o.Page, q, mutate)
	case "wait":
		o.Snapshot, e = wait(ctx, c, s, id, o.Page, q)
	case "console", "network":
		e = capture(ctx, c, s, q, &o)
	default:
		e = errors.New("unsupported browser operation")
	}
	o.Confirmed = e == nil
	return o, e
}
func wait(ctx context.Context, c *connection, s string, id domain.BrowserIdentity, p domain.BrowserPage, q domain.BrowserRequest) (*domain.BrowserSnapshot, error) {
	if _, bounded := ctx.Deadline(); !bounded {
		return nil, errors.New("wait requires a bounded operation context")
	}
	for {
		sn, e := snapshot(ctx, c, s, id, p)
		if errors.Is(e, errIncompleteFrameOrigin) || errors.Is(e, errFrameObservationChanged) {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ctx.Err(), e)
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}
		if e != nil {
			return nil, e
		}
		matched := false
		switch q.WaitFor {
		case "url":
			// Match transient URL components without publishing them as evidence.
			current, err := frameDocument(ctx, c, s)
			if err != nil {
				return nil, err
			}
			matched = documentIdentity(current) == sn.Document && strings.Contains(current.Frame.URL+current.Frame.URLFragment, q.Contains)
		case "text", "gone":
			for _, n := range sn.Nodes {
				if n.Ignored {
					continue
				}
				if (q.Role == "" || n.Role == q.Role) && strings.Contains(n.Name, q.Contains) {
					matched = true
				}
			}
			if q.WaitFor == "gone" {
				if sn.Truncated {
					return nil, errors.New("cannot confirm absence from a truncated semantic snapshot")
				}
				matched = !matched
			}
		case "load":
			var x struct{ Result struct{ Value string } }
			e = c.call(ctx, s, "Runtime.evaluate", map[string]any{"expression": "document.readyState", "returnByValue": true}, &x)
			if e == nil && x.Result.Value == "complete" {
				// Navigation can occur after snapshot's own consistency check.
				// Bind the later load predicate to that same document.
				current, err := frameDocument(ctx, c, s)
				if err != nil {
					return nil, err
				}
				matched = documentIdentity(current) == sn.Document
			}
		default:
			return nil, errors.New("unsupported wait condition")
		}
		if matched {
			return sn, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func screenshot(ctx context.Context, c *connection, s string) ([]byte, error) {
	var x struct{ Data string }
	if e := c.call(ctx, s, "Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": false}, &x); e != nil {
		return nil, e
	}
	return base64.StdEncoding.Strict().DecodeString(x.Data)
}
