// Package cdp observes and controls an already owned browser; it never starts one.
package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const maxMessage = 8 << 20

type confirmedError struct{ error }

type envelope struct {
	ID      int             `json:"id"`
	Session string          `json:"sessionId,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int
		Message string
	} `json:"error,omitempty"`
}
type eventSubscription struct {
	session  string
	methods  map[string]bool
	events   chan envelope
	overflow bool
}

type connection struct {
	ws      *websocket.Conn
	mu      sync.Mutex
	next    int
	pending map[int]chan envelope
	capture *eventSubscription
	done    chan struct{}
	once    sync.Once
}

func dial(ctx context.Context, endpoint string) (*connection, error) {
	d := websocket.Dialer{Proxy: nil, HandshakeTimeout: 5 * time.Second, NetDialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext}
	w, resp, e := d.DialContext(ctx, endpoint, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if e != nil {
		return nil, errors.New("browser websocket connection failed")
	}
	c := &connection{ws: w, pending: map[int]chan envelope{}, done: make(chan struct{})}
	w.SetReadLimit(maxMessage)
	go c.read()
	return c, nil
}
func (c *connection) close() { c.once.Do(func() { close(c.done); c.ws.Close() }) }
func (c *connection) read() {
	defer c.close()
	for {
		var m envelope
		if c.ws.ReadJSON(&m) != nil {
			return
		}
		if m.ID != 0 {
			c.mu.Lock()
			ch := c.pending[m.ID]
			c.mu.Unlock()
			if ch != nil {
				select {
				case ch <- m:
				default:
				}
			}
		} else {
			c.mu.Lock()
			subscription := c.capture
			overflow := false
			if subscription != nil && subscription.session == m.Session && subscription.methods[m.Method] {
				select {
				case subscription.events <- m:
				default:
					overflow = true
					subscription.overflow = true
				}
			}
			c.mu.Unlock()
			if overflow {
				return
			}
		}
	}
}

// subscribe installs the sole operation-local capture before its CDP domain is
// enabled. Ordinary commands and unrelated sessions never accumulate events.
func (c *connection) subscribe(session string, methods ...string) (<-chan envelope, func() (bool, error), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.capture != nil {
		return nil, nil, errors.New("CDP capture already subscribed")
	}
	select {
	case <-c.done:
		return nil, nil, errors.New("CDP disconnected")
	default:
	}
	subscription := &eventSubscription{session: session, methods: map[string]bool{}, events: make(chan envelope, 512)}
	for _, method := range methods {
		subscription.methods[method] = true
	}
	c.capture = subscription
	cancel := func() (bool, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.capture == subscription {
			c.capture = nil
		}
		// Stop admission and inspect pending evidence under the dispatch lock.
		// An overflow must remain an error even if deadline selection wins the
		// race with the reader closing the connection.
		if subscription.overflow {
			return false, errors.New("CDP capture event queue overflow")
		}
		select {
		case <-c.done:
			return false, errors.New("browser disconnected during capture")
		default:
		}
		return len(subscription.events) > 0, nil
	}
	return subscription.events, cancel, nil
}

func (c *connection) call(ctx context.Context, session, method string, p any, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stopCancellation := context.AfterFunc(ctx, c.close)
	defer stopCancellation()
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-c.done:
		return errors.New("CDP disconnected")
	default:
	}
	raw, e := json.Marshal(p)
	if e != nil {
		return e
	}
	ch := make(chan envelope, 1)
	c.mu.Lock()
	c.next++
	id := c.next
	c.pending[id] = ch
	deadline, _ := ctx.Deadline()
	c.ws.SetWriteDeadline(deadline)
	e = c.ws.WriteJSON(envelope{ID: id, Session: session, Method: method, Params: raw})
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
	if e != nil {
		return errors.New("CDP write failed; effect may be uncertain")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return errors.New("CDP disconnected; effect may be uncertain")
	case m := <-ch:
		if m.Error != nil {
			return fmt.Errorf("CDP %s failed (%d)", method, m.Error.Code)
		}
		if out != nil {
			return json.Unmarshal(m.Result, out)
		}
		return nil
	}
}
func validEndpoint(s string, port int) bool {
	u, e := url.Parse(s)
	if e != nil {
		return false
	}
	id := strings.TrimPrefix(u.Path, "/devtools/browser/")
	if len(id) < 1 || len(id) > 128 || u.RawPath != "" {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return e == nil && u.Scheme == "ws" && u.Host == net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) && u.User == nil && u.RawQuery == "" && u.Fragment == "" && strings.HasPrefix(u.Path, "/devtools/browser/") && len(strings.TrimPrefix(u.Path, "/devtools/browser/")) > 0 && !strings.Contains(strings.TrimPrefix(u.Path, "/devtools/browser/"), "/")
}
func privateHTTP() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, DisableKeepAlives: true}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("browser discovery redirect refused") }}
}
