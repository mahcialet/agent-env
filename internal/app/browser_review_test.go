package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBrowserRejectsChangedStoredManifest(t *testing.T) {
	_, l, _, _ := browserFixture(t)
	if _, _, err := selectBrowser(l, ""); err != nil {
		t.Fatal(err)
	}
	l.Manifest = []byte(strings.Replace(string(l.Manifest), "\"web\"", "\"other\"", 1))
	if _, _, err := selectBrowser(l, ""); err == nil {
		t.Fatal("changed manifest accepted with original digest")
	}
}

func TestBrowserCaptureBoundsAfterRedaction(t *testing.T) {
	for _, op := range []string{"console", "network"} {
		for _, repeats := range []int{500, 100} {
			t.Run(op+strings.Repeat("x", repeats/100), func(t *testing.T) {
				s, l, p, _ := browserFixture(t)
				ctx := context.Background()
				sn, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
				if err != nil {
					t.Fatal(err)
				}
				p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
					return domain.BrowserObservation{Confirmed: true, ReadbackEqual: true}, nil
				}
				_, err = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "set-text", Text: "qz", Node: "n1"}, Snapshot: sn.Run.ID})
				if err != nil {
					t.Fatal(err)
				}
				p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
					o := domain.BrowserObservation{Confirmed: true}
					count := 70
					if repeats == 500 {
						count = 40
					}
					for i := 0; i < count; i++ {
						v := strings.Repeat("qz/", repeats)
						if op == "console" {
							o.Console = append(o.Console, domain.BrowserConsole{Type: "log", Text: v})
						} else {
							o.Network = append(o.Network, domain.BrowserNetwork{ID: "id", URL: "https://example.test/", Method: v, Type: "fetch"})
						}
					}
					return o, nil
				}
				got, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: op, Duration: time.Millisecond}})
				if err != nil {
					t.Fatal(err)
				}
				check := func(o domain.BrowserObservation) {
					t.Helper()
					if !o.Truncated {
						t.Error("redaction expansion not marked truncated")
					}
					total := 0
					field := func(v string) {
						t.Helper()
						total += len(v)
						if len(v) > 4096 {
							t.Errorf("field size %d", len(v))
						}
						if strings.Contains(v, "qz") {
							t.Error("secret retained")
						}
					}
					for _, v := range o.Console {
						field(v.Type)
						field(v.Text)
					}
					for _, v := range o.Network {
						field(v.ID)
						field(v.URL)
						field(v.Method)
						field(v.Type)
						field(v.Failure)
					}
					if total > 65536 {
						t.Errorf("aggregate size %d", total)
					}
				}
				check(got.Observation)
				found := false
				for _, a := range got.Artifacts {
					if a.Kind == "browser-run" || strings.HasSuffix(a.Path, "run.json") {
						found = true
						raw, e := os.ReadFile(a.Path)
						if e != nil {
							t.Fatal(e)
						}
						var saved BrowserResult
						if e = json.Unmarshal(raw, &saved); e != nil {
							t.Fatal(e)
						}
						check(saved.Observation)
					}
				}
				if !found {
					t.Fatal("registered run.json missing")
				}
			})
		}
	}
}

func TestBrowserDOMBoundsAfterRedaction(t *testing.T) {
	for _, count := range []int{1000, 2048} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s, l, p, _ := browserFixture(t)
			ctx := context.Background()
			initial, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
			if err != nil {
				t.Fatal(err)
			}
			p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
				return domain.BrowserObservation{Confirmed: true, ReadbackEqual: true}, nil
			}
			if _, err = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "set-text", Text: "qz", Node: "n1"}, Snapshot: initial.Run.ID}); err != nil {
				t.Fatal(err)
			}
			nodes := make([]map[string]any, count)
			for i := range nodes {
				nodes[i] = map[string]any{"document": 0, "index": i, "parent": 0, "backend_id": i + 1, "type": 1, "name": strings.Repeat("qz-", 42)}
			}
			raw, _ := json.Marshal(map[string]any{"version": 1, "nodes": nodes, "truncated": false})
			if len(raw) > 1<<20 {
				t.Fatal("fixture exceeds provider cap")
			}
			p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
				return domain.BrowserObservation{Confirmed: true, DOM: raw}, nil
			}
			got, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "dom-snapshot"}})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, a := range got.Artifacts {
				if a.Kind != "browser-dom" {
					continue
				}
				found = true
				saved, e := os.ReadFile(a.Path)
				if e != nil {
					t.Fatal(e)
				}
				if len(saved) > 1<<20 {
					t.Fatalf("persisted DOM exceeds 1MiB: %d", len(saved))
				}
				if strings.Contains(string(saved), "qz") {
					t.Fatal("DOM secret persisted")
				}
				var doc struct {
					Version int
					Nodes   []struct {
						Index     int
						BackendID int `json:"backend_id"`
						Name      string
					}
					Truncated bool
				}
				if e = json.Unmarshal(saved, &doc); e != nil {
					t.Fatal(e)
				}
				if doc.Version != 1 || len(doc.Nodes) == 0 {
					t.Fatal("missing DOM structure")
				}
				for i, n := range doc.Nodes {
					if n.Index != i || n.BackendID != i+1 {
						t.Fatal("retained DOM identity changed")
					}
				}
				omitted := len(doc.Nodes) < count
				if doc.Truncated != omitted || got.Observation.Truncated != omitted {
					t.Fatalf("omitted=%v document=%v observation=%v", omitted, doc.Truncated, got.Observation.Truncated)
				}
				if (count == 2048) != omitted {
					t.Fatalf("unexpected retained node count %d of %d", len(doc.Nodes), count)
				}
			}
			if !found {
				t.Fatal("missing DOM artifact")
			}
			runs, e := s.Store.Runs(ctx, l.ID)
			if e != nil {
				t.Fatal(e)
			}
			found = false
			for _, run := range runs {
				if run.ID == got.Run.ID {
					found = true
					if run.Status != "passed" {
						t.Fatalf("read-only run remains %s", run.Status)
					}
				}
			}
			if !found {
				t.Fatal("missing durable run")
			}
		})
	}
}

func TestBrowserDOMEncodedBoundary(t *testing.T) {
	const limit = 1 << 20
	for _, size := range []int{limit - 1, limit, limit + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			prefix := `{"version":1,"nodes":[{"backend_id":7,"name":"`
			suffix := `"}],"truncated":false}`
			input := []byte(prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix)
			output, truncated, err := boundBrowserDOM(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(output) > limit || truncated != (size > limit) {
				t.Fatalf("size=%d output=%d truncated=%v", size, len(output), truncated)
			}
			var doc struct {
				Nodes []struct {
					BackendID int `json:"backend_id"`
				}
				Truncated bool
			}
			if err = json.Unmarshal(output, &doc); err != nil {
				t.Fatal(err)
			}
			if doc.Truncated != truncated {
				t.Fatal("artifact and observation disagree")
			}
			if !truncated && (len(doc.Nodes) != 1 || doc.Nodes[0].BackendID != 7) {
				t.Fatal("untruncated identity changed")
			}
			if truncated && len(doc.Nodes) != 0 {
				t.Fatal("truncation set without omission")
			}
		})
	}
}
