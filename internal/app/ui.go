package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

type AndroidUIProvider interface {
	ObserveUI(context.Context, domain.Runtime, domain.UIRequest) (domain.UIObservation, error)
}

type UIOptions struct {
	Operation, Application, Runtime, Snapshot, Node, Text, Contains string
	AllWindows                                                      bool
	Timeout, Duration, Since                                        time.Duration
	X, Y, ToX, ToY                                                  int
}
type UIResult struct {
	Run         domain.CommandRun    `json:"run"`
	Snapshot    *domain.UISnapshot   `json:"snapshot,omitempty"`
	Artifacts   []domain.Artifact    `json:"artifacts"`
	Observation domain.UIObservation `json:"observation"`
	Width       int                  `json:"width,omitempty"`
	Height      int                  `json:"height,omitempty"`
}

func validateUIOptions(o *UIOptions) error {
	switch o.Operation {
	case "snapshot", "screenshot", "tap", "set-text", "tap-coordinate", "back", "home", "swipe", "wait", "logcat":
	default:
		return errors.New("unknown UI operation")
	}
	if o.Application != "" && o.Runtime != "" {
		return errors.New("application and runtime selectors conflict")
	}
	semantic := o.Operation == "tap" || o.Operation == "set-text"
	if o.Operation == "wait" && (o.Application == "" || o.Runtime != "" || o.AllWindows) {
		return errors.New("wait requires an application selector and cannot include all windows")
	}
	if semantic != (o.Snapshot != "" && o.Node != "") || !semantic && (o.Snapshot != "" || o.Node != "") {
		return errors.New("semantic actions require both snapshot and node; other operations reject them")
	}
	if !utf8.ValidString(o.Text) || len(o.Text) > 4096 {
		return errors.New("text must be valid UTF-8 and at most 4096 bytes")
	}
	if o.Operation != "set-text" && o.Text != "" {
		return errors.New("text is only valid for set-text")
	}
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.Timeout <= 0 || o.Timeout > 60*time.Second {
		return errors.New("UI timeout must be positive and at most 60s")
	}
	if o.Duration == 0 {
		o.Duration = 300 * time.Millisecond
	}
	if o.Duration < 50*time.Millisecond || o.Duration > 2*time.Second {
		return errors.New("swipe duration must be 50–2000ms")
	}
	if o.Since == 0 {
		o.Since = 30 * time.Second
	}
	if o.Since < time.Second || o.Since > time.Hour {
		return errors.New("logcat since must be 1s–1h")
	}
	if o.X < 0 || o.Y < 0 || o.ToX < 0 || o.ToY < 0 || o.X > 32767 || o.Y > 32767 || o.ToX > 32767 || o.ToY > 32767 {
		return errors.New("coordinates must be 0–32767")
	}
	if o.Operation == "wait" && (o.Contains == "" || len(o.Contains) > 4096) {
		return errors.New("wait requires bounded nonempty contains text")
	}
	if o.Operation == "logcat" && o.Application == "" {
		return errors.New("logcat requires an application")
	}
	return nil
}

func selectUIRuntime(l domain.Lease, o UIOptions) (domain.Runtime, string, error) {
	name, pkg := o.Runtime, ""
	if o.Application != "" {
		count := 0
		for _, a := range l.Applications {
			if a.Name == o.Application {
				count++
				name, pkg = a.Runtime, a.Package
			}
		}
		if count != 1 || pkg == "" {
			return domain.Runtime{}, "", errors.New("missing or ambiguous application selection")
		}
	}
	var selected domain.Runtime
	count := 0
	for _, r := range l.Runtimes {
		if r.Type == "android-emulator" && (name == "" || name == r.Name) {
			selected = r
			count++
		}
	}
	if count != 1 {
		return selected, "", errors.New("exactly one allocated Android runtime is required")
	}
	if selected.LeaseID != l.ID || selected.Android == nil || selected.Android.Serial == "" || !selected.Started {
		return selected, "", errors.New("Android runtime identity is incomplete")
	}
	for _, r := range l.Runtimes {
		if r.Name != selected.Name && r.Android != nil && r.Android.Serial == selected.Android.Serial {
			return selected, "", errors.New("duplicate Android serial identity")
		}
	}
	if o.AllWindows {
		pkg = ""
	}
	return selected, pkg, nil
}

// UI holds the existing operation fence from state selection through evidence finalization.
func (s *Service) UI(ctx context.Context, id string, o UIOptions) (result UIResult, err error) {
	if err = validateUIOptions(&o); err != nil {
		err = errors.Join(domain.ErrUIInput, err)
		return
	}
	if s.Store == nil || s.AndroidUI == nil {
		return result, errors.New("UI store/provider unavailable")
	}
	inputCtx := ctx
	ctx, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		if _, lookupErr := s.Store.Get(inputCtx, id); errors.Is(lookupErr, sqlite.ErrNotFound) {
			return result, lookupErr
		}
		return result, err
	}
	defer func() { err = errors.Join(err, release()) }()
	l, err := s.Store.Get(ctx, id)
	if err != nil {
		return result, err
	}
	mutation := o.Operation == "tap" || o.Operation == "set-text" || o.Operation == "tap-coordinate" || o.Operation == "back" || o.Operation == "home" || o.Operation == "swipe"
	if l.Desired != "active" || !l.ExpiresAt.After(time.Now()) || l.Observed != "ready" && (mutation || l.Observed != "degraded") {
		return result, errors.New("UI operation requires active unexpired ready lease (diagnostics also permit degraded)")
	}
	if err = applicationCleanupBarrier(l); err != nil {
		return
	}
	runs, err := s.Store.Runs(ctx, id)
	if err != nil {
		return result, err
	}
	for _, r := range runs {
		if r.Status == "running" {
			return result, errors.New("unfinished command blocks UI operations")
		}
	}
	var prior *domain.UISnapshot
	if o.Snapshot != "" {
		prior, err = s.loadUISnapshot(ctx, l, o.Snapshot)
		if err != nil {
			return result, errors.Join(domain.ErrUIInput, err)
		}
		if o.Runtime != "" && o.Runtime != prior.Runtime {
			return result, errors.Join(domain.ErrUIInput, errors.New("snapshot runtime selection mismatch"))
		}
		if o.Runtime == "" && o.Application == "" {
			o.Runtime = prior.Runtime
		}
	}
	r, pkg, err := selectUIRuntime(l, o)
	if err != nil {
		return result, errors.Join(domain.ErrUIInput, err)
	}
	req := domain.UIRequest{Version: 1, Operation: o.Operation, Package: pkg, Text: o.Text, X: o.X, Y: o.Y, ToX: o.ToX, ToY: o.ToY, DurationMS: int(o.Duration / time.Millisecond), SinceSeconds: int(o.Since / time.Second)}
	if prior != nil {
		if prior.Runtime != r.Name || prior.Serial != r.Android.Serial || o.Application != "" && prior.Package != pkg || o.AllWindows {
			return result, errors.Join(domain.ErrUIInput, errors.New("snapshot runtime or scope mismatch"))
		}
		if prior.Tree.Truncated {
			return result, errors.New("AGENTENV-UI-STALE: truncated snapshot cannot authorize input")
		}
		count := 0
		for _, n := range prior.Tree.Nodes {
			if n.Ref == o.Node {
				req.ExpectedFingerprint = n.Fingerprint
				count++
			}
		}
		if count != 1 || req.ExpectedFingerprint == "" {
			return result, errors.New("AGENTENV-UI-STALE: missing or ambiguous snapshot node")
		}
		matches := 0
		for _, n := range prior.Tree.Nodes {
			if n.Fingerprint == req.ExpectedFingerprint {
				matches++
			}
		}
		if matches != 1 {
			return result, errors.New("AGENTENV-UI-AMBIGUOUS: snapshot fingerprint is not unique")
		}
		req.Package = prior.Package
		req.ExpectedBackend = prior.Backend
	}
	result.Run = domain.CommandRun{ID: newID(), LeaseID: id, Name: "ui-" + o.Operation, Argv: []string{"ui", o.Operation, "--runtime", r.Name}, Status: "running", StartedAt: time.Now().UTC(), ExitCode: -1, Notes: []string{"Android serial: " + r.Android.Serial}}
	if o.Application != "" {
		result.Run.Argv = append(result.Run.Argv, "--application", o.Application)
	}
	if o.Operation == "wait" {
		result.Run.Argv = append(result.Run.Argv, "--contains", evidence.RedactString(o.Contains, evidence.InheritedSecrets()), "--timeout", o.Timeout.String())
	}

	if req.Package != "" {
		result.Run.Notes = append(result.Run.Notes, "Application selector package: "+req.Package)
		if o.Operation == "snapshot" || o.Operation == "tap" || o.Operation == "set-text" || o.Operation == "wait" || o.Operation == "logcat" {
			result.Run.Notes = append(result.Run.Notes, "Package scope enforced: "+req.Package)
		}
	}
	if prior != nil {
		result.Run.Argv = append(result.Run.Argv, "--snapshot", prior.ID, "--node", o.Node)
		result.Run.Notes = append(result.Run.Notes, "Expected fingerprint: "+req.ExpectedFingerprint)
	}
	if o.Operation == "tap-coordinate" || o.Operation == "swipe" {
		result.Run.Argv = append(result.Run.Argv, "--x", fmt.Sprint(o.X), "--y", fmt.Sprint(o.Y))
		if o.Operation == "swipe" {
			result.Run.Argv = append(result.Run.Argv, "--to-x", fmt.Sprint(o.ToX), "--to-y", fmt.Sprint(o.ToY), "--duration", o.Duration.String())
		}
	}
	if o.Operation == "set-text" {
		result.Run.Argv = append(result.Run.Argv, "--text", "[REDACTED]")
	}
	directory := filepath.Join(s.Home, "leases", id, "artifacts", result.Run.ID)
	if err = os.MkdirAll(directory, 0700); err != nil {
		return
	}
	if err = s.Store.SaveRun(ctx, result.Run); err != nil {
		return
	}
	deadline, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	if req.Operation == "wait" {
		req.Operation = "snapshot"
	}
	var effectErr error
	helperInstalled := false
	for {
		result.Observation, effectErr = s.AndroidUI.ObserveUI(deadline, r, req)
		helperInstalled = helperInstalled || result.Observation.HelperInstalled
		if effectErr != nil || o.Operation != "wait" || result.Observation.Status != "ok" || uiContains(result.Observation.Snapshot, o.Contains) {
			break
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-deadline.Done():
			timer.Stop()
			effectErr = deadline.Err()
			result.Observation.Status = "timeout"
			result.Observation.Detail = "wait predicate was not satisfied before the deadline"
		case <-timer.C:
		}
		if effectErr != nil {
			break
		}
	}
	result.Observation.HelperInstalled = helperInstalled
	if result.Observation.HelperInstalled {
		result.Run.Notes = append(result.Run.Notes, "Observer companion installed on this owned runtime during this operation; backend="+result.Observation.Backend)
	} else if result.Observation.Backend != "" {
		result.Run.Notes = append(result.Run.Notes, "Observer companion verified on this owned runtime; backend="+result.Observation.Backend)
	}
	// A local request deadline may interrupt self-targeting instrumentation while
	// the operation fence remains held. Quiesce only that verified companion, never
	// retry the original input or treat stopping the helper as action success.
	hostUnconfirmed := errors.Is(effectErr, execx.ErrProcessTreeUnconfirmed) || errors.Is(effectErr, execx.ErrOutputIncomplete)
	if hostUnconfirmed {
		result.Observation.Confirmed = false
	}
	helperOperation := req.Operation == "snapshot" || req.Operation == "tap" || req.Operation == "set-text"
	if !hostUnconfirmed && !result.Observation.Confirmed && helperOperation && ctx.Err() == nil && !operationLost(ctx) {
		result.Run.Notes = append(result.Run.Notes, "Recovery requested: verify and quiesce only the observer companion; do not retry original operation.")
		if checkErr := s.Store.SaveRun(ctx, result.Run); checkErr != nil {
			effectErr = errors.Join(effectErr, checkErr)
		} else {
			recoveryCtx, recoveryCancel := context.WithTimeout(ctx, 10*time.Second)
			quiesced, recoveryErr := s.AndroidUI.ObserveUI(recoveryCtx, r, domain.UIRequest{Version: 1, Operation: "quiesce"})
			recoveryCancel()
			if recoveryErr == nil && quiesced.Confirmed && quiesced.Status == "ok" {
				result.Observation.Confirmed = true
				if result.Observation.Backend == "" {
					result.Observation.Backend = quiesced.Backend
				}
				result.Observation.Detail = "Interrupted operation; verified observer companion quiescence; original outcome remains uncertain."
				result.Observation.Status = "uncertain"
				effectErr = errors.Join(effectErr, errors.New("UI operation interrupted; companion stopped; original outcome remains uncertain"))
			} else {
				effectErr = errors.Join(effectErr, recoveryErr, errors.New("UI companion quiescence is unconfirmed"))
			}
		}
	}
	secrets := evidence.InheritedSecrets()
	if o.Text != "" {
		secrets = append(secrets, o.Text)
	}
	sanitizeUIObservation(&result.Observation, secrets)
	if effectErr != nil {
		prerequisite := errors.Is(effectErr, ErrPrerequisite)
		effectErr = errors.New(evidence.RedactString(effectErr.Error(), secrets))
		if prerequisite {
			effectErr = errors.Join(ErrPrerequisite, effectErr)
		}
	}
	if operationLost(ctx) {
		return result, errors.Join(effectErr, domain.ErrLockLost)
	}
	finish, done := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer done()
	save := func(kind, name string, data []byte) error {
		path := filepath.Join(directory, name)
		if e := evidence.AtomicWrite(path, data, 0600); e != nil {
			return e
		}
		digest := sha256.Sum256(data)
		a := domain.Artifact{ID: newID(), LeaseID: id, RunID: result.Run.ID, Kind: kind, Path: path, Digest: hex.EncodeToString(digest[:]), CreatedAt: time.Now().UTC()}
		if e := s.Store.SaveArtifact(finish, a); e != nil {
			return e
		}
		result.Artifacts = append(result.Artifacts, a)
		return nil
	}
	var persistErr error
	if len(result.Observation.Raw) > 0 {
		persistErr = save("ui-raw", "raw.json", result.Observation.Raw)
	}
	if (o.Operation == "snapshot" || o.Operation == "wait") && result.Observation.Status == "ok" {
		result.Snapshot = &domain.UISnapshot{Version: 1, ID: result.Run.ID, LeaseID: id, Runtime: r.Name, Serial: r.Android.Serial, Package: req.Package, Backend: result.Observation.Backend, CapturedAt: time.Now().UTC(), Tree: result.Observation.Snapshot}
		data, e := json.Marshal(result.Snapshot)
		if e == nil && len(data) > 2<<20 {
			effectErr = errors.Join(effectErr, errors.New("normalized snapshot exceeds limit"))
			result.Snapshot = nil
			result.Observation.Snapshot = domain.UITree{}
			result.Observation.Detail = "Normalized snapshot rejected: exceeds size limit."
			result.Observation.Status = "invalid"
		} else if e == nil {
			e = save("ui-snapshot", "snapshot.json", data)
		}
		persistErr = errors.Join(persistErr, e)
	}
	if o.Operation == "screenshot" && result.Observation.Status == "ok" {
		result.Width, result.Height, err = validateUIPNG(result.Observation.Binary)
		if err == nil {
			persistErr = errors.Join(persistErr, save("ui-screenshot", "screenshot.png", result.Observation.Binary))
		} else {
			effectErr = errors.Join(effectErr, err)
			result.Observation.Status = "invalid"
			result.Observation.Detail = "Screenshot rejected: incomplete, invalid or oversized PNG."
		}
	}
	if o.Operation == "logcat" && result.Observation.Status == "ok" {
		data := []byte(evidence.RedactString(string(result.Observation.Binary), secrets))
		if len(data) > 256<<10 {
			data = data[:256<<10]
			result.Observation.Truncated = true
		}
		result.Observation.Log = string(data)
		persistErr = errors.Join(persistErr, save("ui-logcat", "logcat.log", data))
	}
	result.Observation.Raw = nil
	result.Observation.Binary = nil
	if mutation && result.Observation.Status == "ok" && (!result.Observation.ActionPerformed || o.Operation == "set-text" && !result.Observation.ReadbackEqual) {
		effectErr = errors.Join(effectErr, errors.New("UI mutation lacks confirmed action/read-back evidence"))
	}
	if result.Observation.Status != "ok" && effectErr == nil {
		code := map[string]string{"stale": "AGENTENV-UI-STALE", "ambiguous": "AGENTENV-UI-AMBIGUOUS", "unavailable": "AGENTENV-UI-UNAVAILABLE"}[result.Observation.Status]
		if code == "" {
			code = "AGENTENV-UI-REFUSED"
		}
		effectErr = fmt.Errorf("%s: UI operation refused: %s", code, result.Observation.Status)
	}
	data, e := json.Marshal(result.Observation)
	if e == nil {
		e = save("ui-result", "result.json", data)
	}
	persistErr = errors.Join(persistErr, e)
	if persistErr != nil || !result.Observation.Confirmed {
		classification := "UI recovery classification: termination-unconfirmed"
		if persistErr != nil {
			classification = "UI recovery classification: evidence-incomplete"
		}
		if hostUnconfirmed {
			classification = "UI recovery classification: host-process-unconfirmed"
		}
		result.Run.Notes = append(result.Run.Notes, classification)
		persistErr = errors.Join(persistErr, s.Store.SaveRun(finish, result.Run))
		return result, errors.Join(effectErr, persistErr, errors.New("UI evidence or remote completion unconfirmed; running cleanup barrier retained"))
	}
	result.Run.FinishedAt = time.Now().UTC()
	result.Run.Status = "passed"
	result.Run.ExitCode = 0
	if effectErr != nil {
		result.Run.Status = "failed"
		result.Run.ExitCode = 1
	}
	l.HeartbeatAt = time.Now().UTC()
	if err = s.persist(finish, l); err != nil {
		return result, errors.Join(effectErr, err)
	}
	err = s.Store.SaveRun(finish, result.Run)
	return result, errors.Join(effectErr, err)
}

func uiContains(tree domain.UITree, want string) bool {
	for _, n := range tree.Nodes {
		if n.Visible && !n.Editable && !n.Password && (strings.Contains(n.Text, want) || strings.Contains(n.Description, want) || strings.Contains(n.Hint, want)) {
			return true
		}
	}
	return false
}
func sanitizeUIObservation(o *domain.UIObservation, secrets []string) {
	windowSecret := false
	for i := range o.Snapshot.Windows {
		w := &o.Snapshot.Windows[i]
		before := w.Title + "\x00" + w.RootPackage + "\x00" + w.RootClass
		w.Title = evidence.RedactString(w.Title, secrets)
		w.RootPackage = evidence.RedactString(w.RootPackage, secrets)
		w.RootClass = evidence.RedactString(w.RootClass, secrets)
		windowSecret = windowSecret || before != w.Title+"\x00"+w.RootPackage+"\x00"+w.RootClass
	}
	if windowSecret {
		for i := range o.Snapshot.Windows {
			o.Snapshot.Windows[i].Key = ""
		}
	}
	nodeSecret := false
	for i := range o.Snapshot.Nodes {
		n := &o.Snapshot.Nodes[i]
		before := n.Text + "\x00" + n.Description + "\x00" + n.Hint
		if n.Editable || n.Password {
			n.Text = "[REDACTED]"
			n.Description = "[REDACTED]"
			n.Hint = "[REDACTED]"
		}
		n.Text = evidence.RedactString(n.Text, secrets)
		n.Description = evidence.RedactString(n.Description, secrets)
		n.Hint = evidence.RedactString(n.Hint, secrets)
		nodeSecret = nodeSecret || before != n.Text+"\x00"+n.Description+"\x00"+n.Hint
	}
	if nodeSecret {
		for i := range o.Snapshot.Nodes {
			o.Snapshot.Nodes[i].Fingerprint = ""
		}
	}
	o.Detail = evidence.RedactString(o.Detail, secrets)
	o.Log = evidence.RedactString(o.Log, secrets)
	if len(o.Log) > 256<<10 {
		o.Log = o.Log[:256<<10]
		o.Truncated = true
	}
	// Decode raw as the known typed protocol to discard unrecognized payload fields.
	if len(o.Raw) > 0 {
		var raw domain.UIObservation
		if len(o.Raw) > 1<<20 || json.Unmarshal(o.Raw, &raw) != nil {
			o.Raw = []byte(`{"detail":"raw evidence rejected: invalid or oversized protocol"}`)
		} else {
			raw.Raw = nil
			sanitizeUIObservation(&raw, secrets)
			normalized, marshalErr := json.Marshal(raw)
			if marshalErr != nil || len(normalized) > 1<<20 {
				o.Raw = []byte(`{"detail":"raw evidence rejected after redaction: size bound"}`)
			} else {
				o.Raw = normalized
			}
		}
	}
}
func validateUIPNG(data []byte) (int, int, error) {
	if len(data) == 0 || len(data) > 16<<20 {
		return 0, 0, errors.New("PNG exceeds 16MiB or is empty")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
		return 0, 0, errors.New("PNG exceeds 16 megapixels")
	}
	reader := bytes.NewReader(data)
	if _, err = png.Decode(reader); err != nil {
		return 0, 0, err
	}
	if reader.Len() != 0 {
		return 0, 0, errors.New("PNG has trailing data")
	}
	return cfg.Width, cfg.Height, nil
}
func (s *Service) loadUISnapshot(ctx context.Context, l domain.Lease, id string) (*domain.UISnapshot, error) {
	if id == "" || filepath.Base(id) != id || strings.ContainsAny(id, "/\\") || id == "." || id == ".." {
		return nil, errors.New("invalid snapshot ID")
	}
	artifacts, err := s.Store.Artifacts(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	expected := filepath.Join(s.Home, "leases", l.ID, "artifacts", id, "snapshot.json")
	var found *domain.Artifact
	for _, a := range artifacts {
		if a.Kind == "ui-snapshot" && a.RunID == id {
			if found != nil {
				return nil, errors.New("ambiguous registered snapshot")
			}
			copy := a
			found = &copy
		}
	}
	if found == nil || found.LeaseID != l.ID || filepath.Clean(found.Path) != filepath.Clean(expected) {
		return nil, errors.New("snapshot is not registered for this lease")
	}
	root := filepath.Clean(s.Home)
	for p := expected; p != root; p = filepath.Dir(p) {
		if p == filepath.Dir(p) {
			return nil, errors.New("snapshot path escapes state root")
		}
		st, e := os.Lstat(p)
		if e != nil {
			return nil, e
		}
		if st.Mode()&os.ModeSymlink != 0 || p == expected && !st.Mode().IsRegular() {
			return nil, errors.New("snapshot path must contain no symlinks and be regular")
		}
	}
	f, err := os.Open(expected)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil || len(data) > 2<<20 {
		return nil, errors.New("snapshot exceeds size bound")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != found.Digest {
		return nil, errors.New("snapshot digest mismatch")
	}
	var snapshot domain.UISnapshot
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	if snapshot.Version != 1 || snapshot.ID != id || snapshot.LeaseID != l.ID {
		return nil, errors.New("snapshot identity mismatch")
	}
	return &snapshot, nil
}
