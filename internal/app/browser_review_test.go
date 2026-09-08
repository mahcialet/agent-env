package app

import (
	"context"
	"encoding/json"
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
