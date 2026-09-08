package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

type BrowserProvider interface {
	Observe(context.Context, domain.Runtime, domain.BrowserBinding, domain.BrowserRequest, func(context.Context) error) (domain.BrowserObservation, error)
}
type BrowserOptions struct {
	domain.BrowserRequest
	Browser, Snapshot string
	Timeout           time.Duration
}
type BrowserResult struct {
	Run         domain.CommandRun         `json:"run"`
	Observation domain.BrowserObservation `json:"observation"`
	Snapshot    *domain.BrowserSnapshot   `json:"snapshot,omitempty"`
	Artifacts   []domain.Artifact         `json:"artifacts"`
	Width       int                       `json:"width,omitempty"`
	Height      int                       `json:"height,omitempty"`
}

func browserMutation(op string) bool {
	switch op {
	case "click", "set-text", "key", "scroll", "navigate", "page-create", "page-close":
		return true
	}
	return false
}
func browserSemantic(op string) bool {
	return op == "click" || op == "set-text" || op == "key" || op == "scroll"
}
func validateBrowserOptions(o *BrowserOptions) error {
	switch o.Operation {
	case "capabilities", "pages", "page-create", "page-close", "navigate", "snapshot", "dom-snapshot", "screenshot", "click", "set-text", "key", "scroll", "wait", "console", "network":
	default:
		return errors.New("unknown browser operation")
	}
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.Timeout <= 0 || o.Timeout > 60*time.Second {
		return errors.New("browser timeout must be positive and at most 60s")
	}
	if o.Duration == 0 {
		o.Duration = time.Second
	}
	if o.Duration < time.Millisecond || o.Duration > 10*time.Second {
		return errors.New("browser capture duration must be 1ms..10s")
	}
	if browserSemantic(o.Operation) != (o.Snapshot != "" && o.Node != "") || !browserSemantic(o.Operation) && (o.Snapshot != "" || o.Node != "") {
		return errors.New("browser semantic input requires snapshot and node; other operations reject them")
	}
	if o.Prior != nil {
		return errors.New("browser prior snapshot must be loaded from registered evidence")
	}
	if !utf8.ValidString(o.Text) || len(o.Text) > 4096 || strings.ContainsRune(o.Text, 0) {
		return errors.New("browser text must be valid UTF-8, at most 4096 bytes, without NUL")
	}
	if o.Operation != "set-text" && o.Text != "" {
		return errors.New("text requires set-text")
	}
	if o.Operation == "key" {
		switch o.Key {
		case "Enter", "Tab", "Escape", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Home", "End", "PageUp", "PageDown":
		default:
			return errors.New("unsupported browser key")
		}
	} else if o.Key != "" {
		return errors.New("key requires key operation")
	}
	if o.DeltaX < -10000 || o.DeltaX > 10000 || o.DeltaY < -10000 || o.DeltaY > 10000 {
		return errors.New("scroll deltas must be bounded to 10000")
	}
	if o.Operation != "scroll" && (o.DeltaX != 0 || o.DeltaY != 0) {
		return errors.New("scroll deltas require scroll")
	}
	if o.Operation == "navigate" || o.Operation == "page-create" {
		if o.URL == "" && o.Operation == "page-create" {
			o.URL = "about:blank"
		}
		if err := browserURL(o.URL); err != nil {
			return err
		}
	} else if o.URL != "" {
		return errors.New("URL requires navigate or page-create")
	}
	if o.Operation == "page-close" && o.Page == "" {
		return errors.New("page-close requires explicit page")
	}
	if len(o.Contains) > 4096 || len(o.Role) > 128 {
		return errors.New("wait predicate is too large")
	}
	if o.Operation == "wait" {
		switch o.WaitFor {
		case "load":
		case "url":
			if o.Contains == "" || o.Role != "" {
				return errors.New("URL wait requires a nonempty --contains and does not accept --role")
			}
		case "text", "gone":
			if o.Contains == "" && o.Role == "" {
				return errors.New("wait requires contains or role")
			}
		default:
			return errors.New("wait-for must be load, url, text or gone")
		}
	}
	return nil
}
func browserURL(raw string) error {
	if raw == "about:blank" {
		return nil
	}
	u, e := url.Parse(raw)
	if e != nil || len(raw) > 8192 || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return errors.New("browser navigation requires http(s) URL without user credentials, or about:blank")
	}
	return nil
}
func selectBrowser(l domain.Lease, name string) (domain.Runtime, domain.BrowserBinding, error) {
	var m config.Manifest
	if err := json.Unmarshal(l.Manifest, &m); err != nil {
		return domain.Runtime{}, domain.BrowserBinding{}, errors.New("invalid stored browser manifest")
	}
	if err := config.Validate(&m); err != nil {
		return domain.Runtime{}, domain.BrowserBinding{}, errors.New("stored browser contract is invalid")
	}
	var b domain.BrowserBinding
	var r domain.Runtime
	count := 0
	for n, v := range m.Browsers {
		if name != "" && n != name {
			continue
		}
		for _, candidate := range l.Runtimes {
			if candidate.Name == v.Runtime {
				count++
				b = domain.BrowserBinding{Name: n, Runtime: v.Runtime, CDPPort: v.CDPPort}
				r = candidate
			}
		}
	}
	if count != 1 || r.Type != "process" || r.LeaseID != l.ID || r.Process == nil || !r.Started || r.Process.ProcessID <= 0 || r.Process.ProcessStart == "" {
		return r, b, errors.New("exactly one allocated, identified browser process is required")
	}
	return r, b, nil
}

// Browser holds the lease fence through protocol effects and evidence finalization.
func (s *Service) Browser(ctx context.Context, id string, o BrowserOptions) (result BrowserResult, err error) {
	if err = validateBrowserOptions(&o); err != nil {
		return result, fmt.Errorf("AGENTENV-BROWSER-INPUT: %w", err)
	}
	if s.Store == nil || s.Process == nil || s.BrowserProvider == nil {
		return result, errors.New("browser store/process/provider unavailable")
	}
	ctx, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, release()) }()
	l, err := s.Store.Get(ctx, id)
	if err != nil {
		return result, err
	}
	mutation := browserMutation(o.Operation)
	if l.Desired != "active" || !l.ExpiresAt.After(time.Now()) || l.Observed != "ready" && (mutation || l.Observed != "degraded") {
		return result, errors.New("browser requires active unexpired ready lease; diagnostics also allow degraded")
	}
	if err = applicationCleanupBarrier(l); err != nil {
		return result, err
	}
	runs, err := s.Store.Runs(ctx, id)
	if err != nil {
		return result, err
	}
	for _, run := range runs {
		if run.Status == "running" {
			return result, errors.New("unfinished command blocks browser operations")
		}
	}
	if o.Snapshot != "" {
		prior, e := s.loadBrowserSnapshot(ctx, l, o.Snapshot)
		if e != nil {
			return result, e
		}
		if o.Browser != "" && o.Browser != prior.Browser || o.Page != "" && o.Page != prior.Page.ID {
			return result, errors.New("AGENTENV-BROWSER-STALE: snapshot browser/page mismatch")
		}
		for _, run := range runs {
			if run.Name == "browser-navigate" && run.StartedAt.After(prior.CapturedAt) && len(run.Argv) >= 4 && run.Argv[2] == "--browser" && run.Argv[3] == prior.Browser {
				return result, errors.New("AGENTENV-BROWSER-STALE: navigation requires a fresh snapshot")
			}
		}
		if prior.Truncated {
			return result, errors.New("AGENTENV-BROWSER-STALE: truncated snapshot cannot authorize input")
		}
		matches := 0
		for _, n := range prior.Nodes {
			if n.Ref == o.Node && n.BackendID > 0 && n.Fingerprint != "" {
				matches++
			}
		}
		if matches != 1 {
			return result, errors.New("AGENTENV-BROWSER-STALE: missing or ambiguous snapshot node")
		}
		o.Browser, o.Page, o.Prior = prior.Browser, prior.Page.ID, prior
	}
	fingerprints, err := s.browserTextFingerprints(ctx, l, runs)
	if err != nil {
		return result, err
	}
	r, b, err := selectBrowser(l, o.Browser)
	if err != nil {
		return result, err
	}
	verify := func(check context.Context) error {
		if e := check.Err(); e != nil {
			return e
		}
		if !l.ExpiresAt.After(time.Now()) {
			return errors.New("browser lease expired")
		}
		if operationLost(ctx) {
			return domain.ErrLockLost
		}
		observation, e := s.Process.Inspect(check, r)
		if e != nil {
			return errors.New("browser process ownership cannot be confirmed")
		}
		if !observation.Exists || !observation.Ready {
			return errors.New("browser process is not live and owned")
		}
		return ownedResources(l.ID, observation.Resources)
	}
	if err = verify(ctx); err != nil {
		return result, err
	}
	if o.Prior != nil && (o.Prior.Identity.Runtime != r.Name || o.Prior.Identity.PID != r.Process.ProcessID || o.Prior.Identity.Birth != r.Process.ProcessStart || o.Prior.Identity.Port != r.Process.Ports[b.CDPPort]) {
		return result, errors.New("AGENTENV-BROWSER-STALE: process identity changed")
	}
	if o.Operation == "set-text" && o.Text != "" {
		next := appendBrowserFingerprint(append([]evidence.SecretFingerprint{}, fingerprints...), evidence.FingerprintSecret(o.Text))
		if _, e := evidence.RedactFingerprints("", next); e != nil {
			return result, e
		}
	}
	result.Run = domain.CommandRun{ID: newID(), LeaseID: id, Name: "browser-" + o.Operation, Argv: []string{"browser", o.Operation, "--browser", b.Name}, Status: "running", StartedAt: time.Now().UTC(), ExitCode: -1}
	if browserSemantic(o.Operation) {
		// Record the validated target before input, including when CDP later
		// disconnects and no post-action snapshot can be obtained.
		result.Run.Argv = append(result.Run.Argv, "--page", o.Page, "--snapshot", o.Snapshot, "--node", o.Node)
	}
	if o.Operation == "set-text" {
		result.Run.Argv = append(result.Run.Argv, "--text", "[REDACTED]")
	}
	if err = s.Store.SaveRun(ctx, result.Run); err != nil {
		return result, err
	}
	saveContext := ctx
	directory := filepath.Join(s.Home, "leases", id, "artifacts", result.Run.ID)
	save := func(kind, name string, data []byte) error {
		if len(data) > 16<<20 {
			return errors.New("browser artifact exceeds 16 MiB")
		}
		if e := os.MkdirAll(directory, 0700); e != nil {
			return e
		}
		path := filepath.Join(directory, name)
		if e := evidence.AtomicWrite(path, data, 0600); e != nil {
			return e
		}
		digest := sha256.Sum256(data)
		a := domain.Artifact{ID: newID(), LeaseID: id, RunID: result.Run.ID, Kind: kind, Path: path, Digest: hex.EncodeToString(digest[:]), CreatedAt: time.Now().UTC()}
		if e := s.Store.SaveArtifact(saveContext, a); e != nil {
			return e
		}
		result.Artifacts = append(result.Artifacts, a)
		return nil
	}
	if o.Operation == "set-text" {
		proof := browserTextProof{Version: 1}
		if o.Text != "" {
			proof.Fingerprints = []evidence.SecretFingerprint{evidence.FingerprintSecret(o.Text)}
		}
		proofData, proofErr := json.Marshal(proof)
		if proofErr == nil {
			proofErr = save("browser-redaction", "redaction.json", proofData)
		}
		if proofErr != nil {
			return result, proofErr
		}
		for _, fingerprint := range proof.Fingerprints {
			fingerprints = appendBrowserFingerprint(fingerprints, fingerprint)
		}
	}
	deadline, cancel := context.WithTimeout(ctx, o.Timeout)
	result.Observation, err = s.BrowserProvider.Observe(deadline, r, b, o.BrowserRequest, verify)
	cancel()
	secrets := evidence.InheritedSecrets()
	if o.Text != "" {
		secrets = append(secrets, o.Text)
	}
	if err != nil {
		message, redactErr := evidence.RedactFingerprints(evidence.RedactString(err.Error(), secrets), fingerprints)
		if redactErr != nil {
			err = errors.New("browser operation failed; diagnostic redaction unavailable")
		} else {
			err = errors.New(message)
		}
	}
	if operationLost(ctx) {
		result.Observation = domain.BrowserObservation{}
		return result, errors.Join(err, domain.ErrLockLost)
	}
	finish, done := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer done()
	saveContext = finish
	var persistErr error
	if result.Observation.Snapshot != nil {
		snap := result.Observation.Snapshot
		snap.Version = 1
		snap.ID = result.Run.ID
		snap.LeaseID = id
		snap.Browser = b.Name
		snap.CapturedAt = time.Now().UTC()
		snap.Identity = result.Observation.Identity
	}
	dom, png := result.Observation.DOM, result.Observation.PNG
	result.Observation.DOM = nil
	result.Observation.PNG = nil
	// Redact individual JSON strings rather than serialized escapes, preserving valid JSON.
	data, e := redactBrowserJSON(result.Observation, secrets, fingerprints)
	if e == nil {
		e = json.Unmarshal(data, &result.Observation)
	}
	if e != nil {
		persistErr = e
		result.Observation = domain.BrowserObservation{}
	}
	if err == nil && persistErr == nil && result.Observation.Snapshot != nil {
		result.Snapshot = result.Observation.Snapshot
		data, e := json.Marshal(result.Snapshot)
		if e == nil && len(data) > 2<<20 {
			e = errors.New("browser snapshot exceeds 2 MiB")
		}
		if e == nil {
			e = save("browser-snapshot", "snapshot.json", data)
		}
		persistErr = errors.Join(persistErr, e)
		persistErr = errors.Join(persistErr, save("browser-snapshot-text", "snapshot.txt", []byte(BrowserSnapshotText(*result.Snapshot))))
	}
	if err == nil && len(dom) > 0 {
		var v any
		e := json.Unmarshal(dom, &v)
		if e == nil {
			dom, e = redactBrowserJSON(v, secrets, fingerprints)
		}
		if e == nil {
			e = save("browser-dom", "dom-snapshot.json", dom)
		}
		persistErr = errors.Join(persistErr, e)
	}
	if err == nil && len(png) > 0 {
		result.Width, result.Height, e = validateUIPNG(png)
		if e == nil {
			e = save("browser-screenshot", "screenshot.png", png)
		}
		persistErr = errors.Join(persistErr, e)
	}
	if err == nil && o.Operation == "screenshot" && len(png) == 0 {
		err = errors.New("browser screenshot is empty")
	}
	if err != nil {
		result.Run.Notes = append(result.Run.Notes, err.Error())
	}
	result.Run.FinishedAt = time.Now().UTC()
	result.Run.Status = "passed"
	result.Run.ExitCode = 0
	if err != nil {
		result.Run.Status = "failed"
		result.Run.ExitCode = 1
	}
	if mutation && !result.Observation.Confirmed {
		result.Run.Status = "running"
		result.Run.FinishedAt = time.Time{}
		result.Run.Notes = append(result.Run.Notes, "Browser input completion is uncertain; do not replay. Preserve cleanup barrier.")
		if err == nil {
			err = errors.New("browser input completion is uncertain")
		}
	}
	resultData, e := json.Marshal(result)
	if e == nil {
		e = save("browser-result", "run.json", resultData)
	}
	persistErr = errors.Join(persistErr, e)
	if persistErr != nil {
		return result, errors.Join(err, persistErr)
	}
	if e := s.Store.SaveRun(finish, result.Run); e != nil {
		return result, errors.Join(err, e)
	}
	return result, err
}
func redactBrowserJSON(v any, secrets []string, fingerprints []evidence.SecretFingerprint) ([]byte, error) {
	data, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var tree any
	if e = json.Unmarshal(data, &tree); e != nil {
		return nil, e
	}
	var redactErr error
	var walk func(any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case string:
			redacted, err := evidence.RedactFingerprints(evidence.RedactString(x, secrets), fingerprints)
			if err != nil {
				redactErr = err
				return "[REDACTED]"
			}
			return redacted
		case []any:
			for i := range x {
				x[i] = walk(x[i])
			}
		case map[string]any:
			for k := range x {
				x[k] = walk(x[k])
			}
		}
		return v
	}
	tree = walk(tree)
	if redactErr != nil {
		return nil, redactErr
	}
	data, e = json.Marshal(tree)
	if e != nil {
		return nil, e
	}
	if original, ok := v.(domain.BrowserObservation); ok {
		// Authority fields come from validated bindings/native/CDP identities, not page text.
		// Ordinary input such as "web" must not destroy a future snapshot's browser binding.
		var clean domain.BrowserObservation
		if e = json.Unmarshal(data, &clean); e != nil {
			return nil, e
		}
		clean.Identity = original.Identity
		clean.Page.ID = original.Page.ID
		for i := range clean.Pages {
			clean.Pages[i].ID = original.Pages[i].ID
		}
		for i := range clean.Network {
			clean.Network[i].ID = original.Network[i].ID
		}
		if clean.Snapshot != nil && original.Snapshot != nil {
			clean.Snapshot.Identity = original.Snapshot.Identity
			clean.Snapshot.ID = original.Snapshot.ID
			clean.Snapshot.LeaseID = original.Snapshot.LeaseID
			clean.Snapshot.Browser = original.Snapshot.Browser
			clean.Snapshot.Document = original.Snapshot.Document
			clean.Snapshot.Page.ID = original.Snapshot.Page.ID
			for i := range clean.Snapshot.Nodes {
				clean.Snapshot.Nodes[i].Ref = original.Snapshot.Nodes[i].Ref
				clean.Snapshot.Nodes[i].Frame = original.Snapshot.Nodes[i].Frame
				clean.Snapshot.Nodes[i].Role = original.Snapshot.Nodes[i].Role
				clean.Snapshot.Nodes[i].States = original.Snapshot.Nodes[i].States
				clean.Snapshot.Nodes[i].Fingerprint = original.Snapshot.Nodes[i].Fingerprint
			}
		}
		return json.Marshal(clean)
	}
	return data, nil
}
func BrowserSnapshotText(s domain.BrowserSnapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Snapshot %s browser=%s page=%s truncated=%t\n%q %q\n", s.ID, s.Browser, s.Page.ID, s.Truncated, s.Page.URL, s.Page.Title)
	for _, n := range s.Nodes {
		fmt.Fprintf(&b, "[%s] %s %q disabled=%t editable=%t password=%t\n", n.Ref, n.Role, n.Name, n.Disabled, n.Editable, n.Password)
	}
	return b.String()
}
func (s *Service) loadBrowserSnapshot(ctx context.Context, l domain.Lease, id string) (*domain.BrowserSnapshot, error) {
	if id == "" || filepath.Base(id) != id || strings.ContainsAny(id, "/\\") || id == "." || id == ".." {
		return nil, errors.New("invalid browser snapshot ID")
	}
	artifacts, err := s.Store.Artifacts(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	expected := filepath.Join(s.Home, "leases", l.ID, "artifacts", id, "snapshot.json")
	var found *domain.Artifact
	for _, a := range artifacts {
		if a.Kind == "browser-snapshot" && a.RunID == id {
			if found != nil {
				return nil, errors.New("ambiguous browser snapshot")
			}
			copy := a
			found = &copy
		}
	}
	if found == nil || found.LeaseID != l.ID || filepath.Clean(found.Path) != filepath.Clean(expected) {
		return nil, errors.New("browser snapshot is not registered for this lease")
	}
	root := filepath.Clean(s.Home)
	for p := expected; p != root; p = filepath.Dir(p) {
		if p == filepath.Dir(p) {
			return nil, errors.New("snapshot escapes home")
		}
		st, e := os.Lstat(p)
		if e != nil {
			return nil, e
		}
		if st.Mode()&os.ModeSymlink != 0 || p == expected && !st.Mode().IsRegular() {
			return nil, errors.New("snapshot must be regular without symlinks")
		}
	}
	f, err := os.Open(expected)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		return nil, errors.New("invalid snapshot size")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != found.Digest {
		return nil, errors.New("browser snapshot digest mismatch")
	}
	var snap domain.BrowserSnapshot
	if err = json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if snap.Version != 1 || snap.ID != id || snap.LeaseID != l.ID {
		return nil, errors.New("browser snapshot identity mismatch")
	}
	return &snap, nil
}
