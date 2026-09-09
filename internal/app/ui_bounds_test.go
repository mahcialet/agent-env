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

func TestUIAuditUIRedactionBounds(t *testing.T) {
	for _, tc := range []struct {
		name           string
		count, repeats int
	}{{"field", 1, 500}, {"aggregate", 550, 100}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUDIT_PRIVATE_SECRET", "qz")
			s, db, l, p := uiFixture(t)
			p.observe = func(context.Context, domain.UIRequest) (domain.UIObservation, error) {
				o := domain.UIObservation{Version: 1, Status: "ok", Confirmed: true, Backend: "test-v1"}
				for i := 0; i < tc.count; i++ {
					o.Snapshot.Nodes = append(o.Snapshot.Nodes, domain.UINode{Ref: fmt.Sprintf("n%d", i+1), Fingerprint: strings.Repeat("a", 64), Text: strings.Repeat("qz/", tc.repeats), Description: strings.Repeat("qz/", tc.repeats)})
				}
				raw, _ := json.Marshal(o)
				t.Logf("input bytes=%d", len(raw))
				if len(raw) > 700000 {
					t.Fatal("fixture exceeds helper budget")
				}
				return o, nil
			}
			result, e := s.UI(context.Background(), l.ID, UIOptions{Operation: "snapshot"})
			if e != nil {
				t.Fatal(e)
			}
			if result.Snapshot == nil || len(result.Snapshot.Tree.Nodes) == 0 || !result.Snapshot.Tree.Truncated || !result.Observation.Truncated {
				t.Fatalf("missing retained evidence/truncation: %+v", result)
			}
			if tc.name == "field" && len(result.Snapshot.Tree.Nodes) != 1 {
				t.Fatal("field truncation unnecessarily dropped the node")
			}
			if tc.name == "aggregate" && len(result.Snapshot.Tree.Nodes) >= tc.count {
				t.Fatal("oversized aggregate retained every node")
			}
			for i, n := range result.Snapshot.Tree.Nodes {
				if n.Ref != fmt.Sprintf("n%d", i+1) {
					t.Fatal("node identity/order changed")
				}
				if n.Fingerprint != "" {
					t.Fatal("secret-derived fingerprint retained")
				}
			}
			for _, n := range result.Snapshot.Tree.Nodes {
				if len([]rune(n.Text)) > 4096 {
					t.Errorf("field chars=%d", len([]rune(n.Text)))
					break
				}
			}
			seen := map[string]bool{}
			for _, a := range result.Artifacts {
				if a.Kind == "ui-snapshot" || a.Kind == "ui-result" {
					seen[a.Kind] = true
					raw, e := os.ReadFile(a.Path)
					if e != nil {
						t.Fatal(e)
					}
					if len(raw) > 1<<20 || strings.Contains(string(raw), "qz") {
						t.Fatalf("published size/secret violation kind=%s bytes=%d", a.Kind, len(raw))
					}
					if !strings.Contains(string(raw), `"truncated":true`) {
						t.Fatal("artifact does not disclose truncation")
					}
				}
			}
			if !seen["ui-snapshot"] || !seen["ui-result"] {
				t.Fatalf("missing registered evidence: %v", seen)
			}
			runs, e := db.Runs(context.Background(), l.ID)
			if e != nil || len(runs) != 1 || runs[0].Status != "passed" {
				t.Fatalf("completed read-only capture left barrier: %+v %v", runs, e)
			}
			for _, a := range result.Artifacts {
				if a.Kind == "ui-snapshot" {
					raw, e := os.ReadFile(a.Path)
					if e != nil {
						t.Fatal(e)
					}
					t.Logf("snapshot bytes=%d truncated=%v", len(raw), result.Snapshot.Tree.Truncated)
					if len(raw) > 1<<20 {
						t.Errorf("registered snapshot exceeds1MiB: %d", len(raw))
					}
				}
			}
		})
	}
}

func TestUIFieldBoundaryAndSerializedObservationBoundary(t *testing.T) {
	for _, n := range []int{4095, 4096, 4097} {
		t.Run(fmt.Sprintf("field-%d", n), func(t *testing.T) {
			o := domain.UIObservation{Snapshot: domain.UITree{Nodes: []domain.UINode{{Ref: "n1", Fingerprint: "stable", Text: strings.Repeat("界", n)}}}}
			if e := boundUIEvidence(&o, nil); e != nil {
				t.Fatal(e)
			}
			want := n > 4096
			if o.Truncated != want || o.Snapshot.Truncated != want || len(o.Snapshot.Nodes) != 1 || o.Snapshot.Nodes[0].Fingerprint != "stable" {
				t.Fatalf("incorrect boundary/result: %+v", o)
			}
			if !want && o.Snapshot.Nodes[0].Text != strings.Repeat("界", n) {
				t.Fatal("in-bound field changed")
			}
		})
	}
	for _, delta := range []int{-1, 0, 1} {
		t.Run(fmt.Sprintf("serialized-%d", delta), func(t *testing.T) {
			o := domain.UIObservation{Status: "ok", Confirmed: true, Log: "x"}
			for i := 0; i < 140; i++ {
				o.Snapshot.Nodes = append(o.Snapshot.Nodes, domain.UINode{Ref: fmt.Sprintf("n%d", i), Fingerprint: "stable", Text: strings.Repeat("\x00", 1000)})
			}
			b, e := json.Marshal(o)
			if e != nil {
				t.Fatal(e)
			}
			padding := (1 << 20) + delta - len(b) + 1
			if padding < 0 || padding > 256<<10 {
				t.Fatalf("invalid fixture padding%d", padding)
			}
			o.Log = strings.Repeat("x", padding)
			b, e = json.Marshal(o)
			if e != nil || len(b) != (1<<20)+delta {
				t.Fatalf("fixture bytes=%d err=%v", len(b), e)
			}
			if e = boundUIEvidence(&o, nil); e != nil {
				t.Fatal(e)
			}
			b, e = json.Marshal(o)
			if e != nil || len(b) > 1<<20 || o.Truncated != (delta > 0) || o.Snapshot.Truncated != (delta > 0) {
				t.Fatalf("bytes%d truncated%v err%v", len(b), o.Truncated, e)
			}
			if len(o.Snapshot.Nodes) == 0 {
				t.Fatal("discarded all evidence")
			}
			if delta > 0 && len(o.Snapshot.Nodes) >= 140 {
				t.Fatal("oversized evidence marked truncated without omitting nodes")
			}
			if delta <= 0 && len(o.Snapshot.Nodes) != 140 {
				t.Fatal("exact valid evidence shortened")
			}
			for i, n := range o.Snapshot.Nodes {
				if n.Ref != fmt.Sprintf("n%d", i) || n.Fingerprint != "stable" {
					t.Fatal("identity changed")
				}
			}
		})
	}
}
