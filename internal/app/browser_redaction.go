package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

type browserTextProof struct {
	Version      int                          `json:"version"`
	Fingerprints []evidence.SecretFingerprint `json:"fingerprints"`
}

func (s *Service) browserTextFingerprints(ctx context.Context, l domain.Lease, runs []domain.CommandRun) ([]evidence.SecretFingerprint, error) {
	artifacts, err := s.Store.Artifacts(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	proofs := map[string]domain.Artifact{}
	for _, a := range artifacts {
		if a.Kind == "browser-redaction" {
			if _, exists := proofs[a.RunID]; exists {
				return nil, errors.New("ambiguous browser redaction proof")
			}
			proofs[a.RunID] = a
		}
	}
	var result []evidence.SecretFingerprint
	seen := map[evidence.SecretFingerprint]bool{}
	for _, run := range runs {
		if run.Name != "browser-set-text" {
			continue
		}
		a, ok := proofs[run.ID]
		if !ok {
			return nil, errors.New("browser input redaction proof is missing")
		}
		expected := filepath.Join(s.Home, "leases", l.ID, "artifacts", run.ID, "redaction.json")
		if a.LeaseID != l.ID || filepath.Clean(a.Path) != filepath.Clean(expected) {
			return nil, errors.New("browser redaction proof path is invalid")
		}
		root := filepath.Clean(s.Home)
		for p := expected; p != root; p = filepath.Dir(p) {
			if p == filepath.Dir(p) {
				return nil, errors.New("redaction proof escapes home")
			}
			st, e := os.Lstat(p)
			if e != nil {
				return nil, e
			}
			if st.Mode()&os.ModeSymlink != 0 || p == expected && !st.Mode().IsRegular() {
				return nil, errors.New("redaction proof must be regular without symlinks")
			}
		}
		f, e := os.Open(expected)
		if e != nil {
			return nil, e
		}
		data, e := io.ReadAll(io.LimitReader(f, 4097))
		f.Close()
		if e != nil || len(data) > 4096 {
			return nil, errors.New("invalid redaction proof size")
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != a.Digest {
			return nil, errors.New("browser redaction proof digest mismatch")
		}
		var proof browserTextProof
		if e = json.Unmarshal(data, &proof); e != nil || proof.Version != 1 || len(proof.Fingerprints) > 1 {
			return nil, errors.New("invalid browser redaction proof")
		}
		for _, fingerprint := range proof.Fingerprints {
			if !seen[fingerprint] {
				seen[fingerprint] = true
				result = append(result, fingerprint)
			}
		}
	}
	if _, err = evidence.RedactFingerprints("", result); err != nil {
		return nil, err
	}
	return result, nil
}

func appendBrowserFingerprint(values []evidence.SecretFingerprint, value evidence.SecretFingerprint) []evidence.SecretFingerprint {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// Redaction can expand strings, so the persisted representation must satisfy
// the capture limits independently of the provider's pre-redaction limits.
func boundRedactedBrowserCapture(o *domain.BrowserObservation) {
	total := 0
	retain := func(fields ...*string) bool {
		size := 0
		for _, field := range fields {
			if len(*field) > 4096 {
				*field = "[TRUNCATED]"
				o.Truncated = true
			}
			size += len(*field)
		}
		if total+size > 65536 {
			o.Truncated = true
			return false
		}
		total += size
		return true
	}
	console := o.Console[:0]
	for _, v := range o.Console {
		if len(console) >= 256 {
			o.Truncated = true
			continue
		}
		if retain(&v.Type, &v.Text) {
			console = append(console, v)
		}
	}
	o.Console = console
	network := o.Network[:0]
	for _, v := range o.Network {
		if len(network) >= 256 {
			o.Truncated = true
			continue
		}
		if retain(&v.ID, &v.URL, &v.Method, &v.Type, &v.Failure) {
			network = append(network, v)
		}
	}
	o.Network = network
}
