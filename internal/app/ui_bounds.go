package app

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/mahcialet/agent-env/internal/domain"
)

const uiEvidenceLimit = 1 << 20

// boundUIEvidence checks the actual post-redaction serialized envelopes, including
// metadata and escaping. Node order/identities remain intact; an omitted suffix
// or field is explicit and cannot subsequently authorize semantic input.
func boundUIEvidence(o *domain.UIObservation, snapshot *domain.UISnapshot) error {
	o.Truncated = o.Truncated || o.Snapshot.Truncated
	field := func(s *string) {
		if utf8.RuneCountInString(*s) > 4096 {
			*s = "[TRUNCATED]"
			o.Truncated = true
			o.Snapshot.Truncated = true
		}
	}
	field(&o.Detail)
	for i := range o.Snapshot.Windows {
		w := &o.Snapshot.Windows[i]
		for _, s := range []*string{&w.Title, &w.RootPackage, &w.RootClass} {
			field(s)
		}
	}
	for i := range o.Snapshot.Nodes {
		n := &o.Snapshot.Nodes[i]
		for _, s := range []*string{&n.Package, &n.Class, &n.ResourceID, &n.Text, &n.Description, &n.Hint} {
			field(s)
		}
	}
	nodes := o.Snapshot.Nodes
	if len(nodes) > 1000 {
		o.Snapshot.Nodes = nodes[:1000]
		o.Snapshot.Truncated = true
		o.Truncated = true
		nodes = o.Snapshot.Nodes
	}
	fits := func() bool {
		if snapshot != nil {
			snapshot.Tree = o.Snapshot
			b, e := json.Marshal(snapshot)
			if e != nil || len(b) > uiEvidenceLimit {
				return false
			}
		}
		b, e := json.Marshal(o)
		return e == nil && len(b) <= uiEvidenceLimit
	}
	if fits() {
		return nil
	}
	o.Snapshot.Truncated = true
	o.Truncated = true
	o.Snapshot.Nodes = nil
	if !fits() {
		return errors.New("UI evidence metadata exceeds size limit")
	}
	if len(nodes) == 0 {
		return errors.New("UI evidence metadata exceeds size limit")
	}
	low, high := 0, len(nodes)-1
	for low < high {
		mid := low + (high-low+1)/2
		o.Snapshot.Nodes = nodes[:mid]
		if fits() {
			low = mid
		} else {
			high = mid - 1
		}
	}
	o.Snapshot.Nodes = nodes[:low]
	if !fits() {
		return errors.New("UI evidence exceeds size limit")
	}
	return nil
}

// The device completed, but its evidence is invalid. Keep a bounded failed
// observation rather than confusing validation failure with an unfinished effect.
func rejectUIEvidence(o *domain.UIObservation) {
	o.Status = "invalid"
	o.Detail = "UI evidence rejected: metadata exceeds size limit."
	o.Snapshot = domain.UITree{Truncated: true}
	o.Truncated = true
	o.Backend = ""
	o.AfterFingerprint = ""
	o.Since = ""
	o.Log = ""
}
