package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type StatusError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *StatusError) Error() string   { return fmt.Sprintf("controller %s: %s", e.Code, e.Message) }
func (e *StatusError) Temporary() bool { return e.StatusCode == 429 || e.StatusCode >= 500 }

type Client struct {
	base string
	http *http.Client
}

func New(base string, cfg *tls.Config) (*Client, error) {
	u, e := url.Parse(base)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("controller URL must be an HTTPS origin")
	}
	if cfg == nil || cfg.InsecureSkipVerify || len(cfg.Certificates) == 0 {
		return nil, errors.New("verified mTLS client configuration required")
	}
	tlsConfig := cfg.Clone()
	tlsConfig.MinVersion = tls.VersionTLS13
	tr := &http.Transport{TLSClientConfig: tlsConfig, DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 90 * time.Second}
	return &Client{base: strings.TrimSuffix(base, "/"), http: &http.Client{Transport: tr, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Close() { c.http.CloseIdleConnections() }
func status(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	var e protocol.Error
	if json.Unmarshal(b, &e) != nil || e.Code == "" {
		e = protocol.Error{Code: "http", Message: "controller request failed"}
	}
	return &StatusError{StatusCode: resp.StatusCode, Code: e.Code, Message: e.Message}
}
func (c *Client) request(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		if len(b) > 8<<20 {
			return errors.New("controller request exceeds metadata limit")
		}
		body = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if e != nil {
		return e
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return status(resp)
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if e != nil {
		return e
	}
	if len(data) > 8<<20 {
		return errors.New("controller response exceeds metadata limit")
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
func (c *Client) Info(ctx context.Context) (protocol.Info, error) {
	var o protocol.Info
	e := c.request(ctx, "GET", "/v1/info", nil, &o)
	return o, e
}
func (c *Client) Register(ctx context.Context, q protocol.RegisterRequest) (protocol.RegisterResult, error) {
	var o protocol.RegisterResult
	e := c.request(ctx, "POST", "/v1/register", q, &o)
	return o, e
}
func (c *Client) Heartbeat(ctx context.Context, q protocol.WorkerIdentity) error {
	return c.request(ctx, "POST", "/v1/heartbeat", q, nil)
}
func (c *Client) Poll(ctx context.Context, q protocol.PollRequest) (protocol.PollResult, error) {
	var o protocol.PollResult
	e := c.request(ctx, "POST", "/v1/poll", q, &o)
	return o, e
}
func (c *Client) Complete(ctx context.Context, q protocol.Result) error {
	return c.request(ctx, "POST", "/v1/results", q, nil)
}
func (c *Client) Create(ctx context.Context, q protocol.CreateRequest) (protocol.Operation, error) {
	var o protocol.Operation
	e := c.request(ctx, "POST", "/v1/leases", q, &o)
	return o, e
}
func (c *Client) Submit(ctx context.Context, q protocol.SubmitRequest) (protocol.Operation, error) {
	var o protocol.Operation
	e := c.request(ctx, "POST", "/v1/operations", q, &o)
	return o, e
}
func (c *Client) GetOperation(ctx context.Context, id string) (protocol.Operation, error) {
	var o protocol.Operation
	e := c.request(ctx, "GET", "/v1/operations/"+url.PathEscape(id), nil, &o)
	return o, e
}
func (c *Client) GetLease(ctx context.Context, id string) (protocol.Lease, error) {
	var o protocol.Lease
	e := c.request(ctx, "GET", "/v1/leases/"+url.PathEscape(id), nil, &o)
	return o, e
}
func (c *Client) ListLeases(ctx context.Context) ([]protocol.Lease, error) {
	var o []protocol.Lease
	e := c.request(ctx, "GET", "/v1/leases", nil, &o)
	return o, e
}
func (c *Client) ListHosts(ctx context.Context) ([]protocol.Host, error) {
	var o []protocol.Host
	e := c.request(ctx, "GET", "/v1/hosts", nil, &o)
	return o, e
}
func (c *Client) GetHost(ctx context.Context, id string) (protocol.Host, error) {
	var o protocol.Host
	e := c.request(ctx, "GET", "/v1/hosts/"+url.PathEscape(id), nil, &o)
	return o, e
}
func (c *Client) Drain(ctx context.Context, id string) (protocol.Host, error) {
	var o protocol.Host
	e := c.request(ctx, "POST", "/v1/hosts/"+url.PathEscape(id)+"/drain", nil, &o)
	return o, e
}
func (c *Client) Undrain(ctx context.Context, id string) (protocol.Host, error) {
	var o protocol.Host
	e := c.request(ctx, "POST", "/v1/hosts/"+url.PathEscape(id)+"/undrain", nil, &o)
	return o, e
}
func (c *Client) Remove(ctx context.Context, id string) (protocol.Host, error) {
	var o protocol.Host
	e := c.request(ctx, "POST", "/v1/hosts/"+url.PathEscape(id)+"/remove", nil, &o)
	return o, e
}

// blobRequest shares connection/TLS/header protections with metadata requests,
// but a streaming body is governed by the caller's context, not a whole-request
// metadata timer. Copying the client leaves concurrent metadata calls bounded.
func (c *Client) blobRequest(req *http.Request) (*http.Response, error) {
	streaming := *c.http
	streaming.Timeout = 0
	return streaming.Do(req)
}
func (c *Client) Upload(ctx context.Context, r io.Reader, digest, kind string) (protocol.Blob, error) {
	var o protocol.Blob
	req, e := http.NewRequestWithContext(ctx, "PUT", c.base+"/v1/blobs/"+url.PathEscape(digest)+"?kind="+url.QueryEscape(kind), r)
	if e != nil {
		return o, e
	}
	// Cancellation must also unblock an upload reader waiting for more input.
	// net/http owns request bodies, but may wait for its body-writing goroutine
	// before returning from Do; explicitly close a cancelable streaming body.
	if req.Body != nil {
		stop := context.AfterFunc(ctx, func() { _ = req.Body.Close() })
		defer stop()
	}
	resp, e := c.blobRequest(req)
	if e != nil {
		// Closing a blocked body can win over the transport's context error.
		// Keep both causes so cancellation remains observable to callers.
		return o, errors.Join(e, ctx.Err())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return o, status(resp)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if e != nil {
		return o, e
	}
	e = json.Unmarshal(b, &o)
	return o, e
}
func (c *Client) Download(ctx context.Context, digest, kind string) (io.ReadCloser, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", c.base+"/v1/blobs/"+url.PathEscape(digest)+"?kind="+url.QueryEscape(kind), nil)
	if e != nil {
		return nil, e
	}
	resp, e := c.blobRequest(req)
	if e != nil {
		return nil, e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, status(resp)
	}
	return resp.Body, nil
}
