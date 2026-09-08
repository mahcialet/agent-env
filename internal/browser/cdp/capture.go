package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func capture(ctx context.Context, c *connection, s string, q domain.BrowserRequest, o *domain.BrowserObservation) error {
	d := q.Duration
	if d <= 0 || d > 10*time.Second {
		return errors.New("capture duration must be greater than zero and at most 10 seconds")
	}
	method := "Runtime.enable"
	methods := []string{"Runtime.consoleAPICalled"}
	started := float64(time.Now().UnixMilli())
	if q.Operation == "network" {
		method = "Network.enable"
		methods = []string{"Network.requestWillBeSent", "Network.responseReceived", "Network.loadingFailed"}
	}
	events, unsubscribe, err := c.subscribe(s, methods...)
	if err != nil {
		return err
	}
	defer unsubscribe()
	if e := c.call(ctx, s, method, nil, nil); e != nil {
		return e
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	bytes := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return errors.New("browser disconnected during capture")
		case <-timer.C:
			pending, err := unsubscribe()
			if pending {
				o.Truncated = true
			}
			return err
		case ev := <-events:
			if ev.Session != s {
				continue
			}
			if q.Operation == "console" && ev.Method == "Runtime.consoleAPICalled" {
				var x struct {
					Type      string
					Timestamp float64
					Args      []struct {
						Type  string
						Value json.RawMessage
					}
				}
				if json.Unmarshal(ev.Params, &x) != nil {
					return errors.New("malformed console event")
				}
				if x.Timestamp < started {
					continue
				}
				parts := []string{}
				for _, a := range x.Args {
					if a.Type != "string" {
						continue
					}
					var v string
					if json.Unmarshal(a.Value, &v) == nil {
						parts = append(parts, v)
					}
				}
				v := strings.Join(parts, " ")
				size := boundCaptureStrings(o, &x.Type, &v)
				if bytes+size > 65536 || len(o.Console) >= 256 {
					o.Truncated = true
					continue
				}
				bytes += size
				o.Console = append(o.Console, domain.BrowserConsole{Type: x.Type, Text: v, Timestamp: x.Timestamp})
			}
			if q.Operation == "network" {
				var x struct {
					RequestID string
					Timestamp float64
					Type      string
					ErrorText string
					Request   struct {
						URL    string
						Method string
					}
					Response struct {
						URL    string
						Status int
					}
				}
				if json.Unmarshal(ev.Params, &x) != nil {
					return errors.New("malformed network event")
				}
				n := domain.BrowserNetwork{ID: x.RequestID, Timestamp: x.Timestamp, Type: x.Type}
				switch ev.Method {
				case "Network.requestWillBeSent":
					n.URL = scrubURL(x.Request.URL)
					n.Method = x.Request.Method
				case "Network.responseReceived":
					n.URL = scrubURL(x.Response.URL)
					n.Status = x.Response.Status
				case "Network.loadingFailed":
					n.Failure = "request failed"
				default:
					continue
				}
				size := boundCaptureStrings(o, &n.ID, &n.URL, &n.Method, &n.Type, &n.Failure)
				if bytes+size > 65536 || len(o.Network) >= 256 {
					o.Truncated = true
					continue
				}
				bytes += size
				o.Network = append(o.Network, n)
			}
		}
	}
}

// Every retained string participates in both limits, including metadata. Replacing
// oversized values preserves UTF-8 and avoids leaking partial sensitive strings.
func boundCaptureStrings(o *domain.BrowserObservation, fields ...*string) int {
	size := 0
	for _, field := range fields {
		if len(*field) > 4096 {
			*field = "[TRUNCATED]"
			o.Truncated = true
		}
		size += len(*field)
	}
	return size
}
