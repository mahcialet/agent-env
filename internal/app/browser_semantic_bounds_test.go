package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestBrowserSemanticBoundsAfterRedaction(t *testing.T) {
	for _, tc := range []struct {
		name           string
		nodes, repeats int
	}{{"field", 1, 500}, {"aggregate", 700, 100}, {"large-aggregate", 1100, 100}} {
		t.Run(tc.name, func(t *testing.T) {
			s, l, p, _ := browserFixture(t)
			ctx := context.Background()
			initial, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
			if err != nil {
				t.Fatal(err)
			}
			p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
				return domain.BrowserObservation{Confirmed: true, ReadbackEqual: true}, nil
			}
			_, err = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "set-text", Text: "qz", Node: "n1"}, Snapshot: initial.Run.ID})
			if err != nil {
				t.Fatal(err)
			}
			p.fn = func(context.Context, domain.BrowserRequest) (domain.BrowserObservation, error) {
				sn := &domain.BrowserSnapshot{Page: domain.BrowserPage{ID: "page"}, Document: "doc"}
				for i := 0; i < tc.nodes; i++ {
					sn.Nodes = append(sn.Nodes, domain.BrowserNode{Ref: fmt.Sprintf("n%d", i+1), BackendID: i + 1, Frame: "frame", Role: "button", Name: strings.Repeat("qz/", tc.repeats), Value: strings.Repeat("qz/", tc.repeats), Fingerprint: "digest"})
				}
				raw, _ := json.Marshal(sn)
				if len(raw) > 1<<20 {
					t.Fatal("fixture exceeds original provider budget")
				}
				return domain.BrowserObservation{Confirmed: true, Snapshot: sn}, nil
			}
			got, err := s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "snapshot"}})
			if err != nil {
				t.Fatalf("bounded read-only observation failed: %v", err)
			}
			if got.Run.Status == "running" {
				t.Fatal("read-only snapshot left cleanup barrier")
			}
			check := func(sn *domain.BrowserSnapshot) {
				t.Helper()
				if sn == nil {
					t.Fatal("missing snapshot")
				}
				if !sn.Truncated {
					t.Error("expanded snapshot not marked truncated")
				}
				raw, e := json.Marshal(sn)
				if e != nil {
					t.Fatal(e)
				}
				if len(raw) > 1<<20 {
					t.Errorf("semantic JSON bytes=%d", len(raw))
				}
				if strings.Contains(string(raw), "qz") {
					t.Fatal("secret in bounded snapshot")
				}
				for i, n := range sn.Nodes {
					if n.Ref != fmt.Sprintf("n%d", i+1) || n.BackendID != i+1 || n.Fingerprint != "digest" {
						t.Fatal("retained target identity changed")
					}
					if len(n.Name) > 4096 || len(n.Value) > 4096 {
						t.Fatalf("AX field sizes %d/%d", len(n.Name), len(n.Value))
					}
				}
			}
			check(got.Snapshot)
			check(got.Observation.Snapshot)
			found := false
			for _, a := range got.Artifacts {
				if a.Kind == "browser-snapshot" {
					found = true
					raw, e := os.ReadFile(a.Path)
					if e != nil {
						t.Fatal(e)
					}
					if len(raw) > 1<<20 {
						t.Fatalf("artifact bytes=%d", len(raw))
					}
					var sn domain.BrowserSnapshot
					if e = json.Unmarshal(raw, &sn); e != nil {
						t.Fatal(e)
					}
					check(&sn)
				}
			}
			if !found {
				t.Fatal("registered semantic snapshot missing")
			}
			runs, e := s.Store.Runs(ctx, l.ID)
			if e != nil {
				t.Fatal(e)
			}
			saved := false
			for _, run := range runs {
				if run.ID == got.Run.ID {
					saved = true
					if run.Status != "passed" {
						t.Fatalf("persisted snapshot status=%s", run.Status)
					}
				}
			}
			if !saved {
				t.Fatal("snapshot run not persisted")
			}
			calls := p.calls
			_, err = s.Browser(ctx, l.ID, BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "click", Node: "n1"}, Snapshot: got.Run.ID})
			if err == nil || p.calls != calls {
				t.Fatal("truncated snapshot authorized input")
			}
		})
	}
}
