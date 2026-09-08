package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// RecoverUI confirms termination of a previously interrupted companion operation.
// It cannot turn that operation into success or waive incomplete evidence from a
// successful response. Original evidence remains immutable and is referenced by
// the separately registered recovery evidence.
func (s *Service) RecoverUI(ctx context.Context, id, runID string) (result UIResult, err error) {
	if s.Store == nil || s.AndroidUI == nil {
		return result, errors.New("UI store/provider unavailable")
	}
	if runID == "" || filepath.Base(runID) != runID || strings.ContainsAny(runID, "/\\") || runID == "." || runID == ".." {
		return result, errors.Join(domain.ErrUIInput, errors.New("invalid UI run ID"))
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
	lease, err := s.Store.Get(ctx, id)
	if err != nil {
		return result, err
	}
	if lease.Observed == "released" {
		return result, errors.New("released lease cannot recover UI operations")
	}
	runs, err := s.Store.Runs(ctx, id)
	if err != nil {
		return result, err
	}
	found := 0
	for _, run := range runs {
		if run.ID == runID {
			result.Run = run
			found++
		} else if run.Status == "running" {
			return result, errors.New("another unfinished command prevents isolated UI recovery")
		}
	}
	if found != 1 || result.Run.LeaseID != id || result.Run.Status != "running" {
		return result, errors.Join(domain.ErrUIInput, errors.New("recovery requires one registered running UI helper run"))
	}
	args := result.Run.Argv
	if len(args) < 4 || args[0] != "ui" || args[2] != "--runtime" || result.Run.Name != "ui-"+args[1] {
		return result, errors.New("UI run intent is not a supported helper operation")
	}
	switch args[1] {
	case "snapshot", "wait", "tap", "set-text":
	default:
		return result, errors.New("native commands and unrelated tests cannot use helper recovery")
	}
	runtime, _, err := selectUIRuntime(lease, UIOptions{Runtime: args[3]})
	if err != nil {
		return result, err
	}
	serialNotes := 0
	eligible := false
	for _, note := range result.Run.Notes {
		if note == "UI recovery classification: termination-unconfirmed" {
			eligible = true
		}
		if note == "UI recovery classification: evidence-incomplete" || note == "UI recovery classification: host-process-unconfirmed" {
			return result, errors.New("original UI evidence is incomplete; helper recovery cannot waive its barrier")
		}
		if strings.HasPrefix(note, "Android serial: ") {
			serialNotes++
			if note != "Android serial: "+runtime.Android.Serial {
				return result, errors.New("recorded UI serial differs from current lease runtime")
			}
		}
	}
	if !eligible {
		return result, errors.New("UI run lacks durable termination-unconfirmed eligibility; unclassified interruption remains blocked")
	}
	if serialNotes != 1 {
		return result, errors.New("UI run lacks unique durable serial identity")
	}
	artifacts, err := s.Store.Artifacts(ctx, id)
	if err != nil {
		return result, err
	}
	var original *domain.Artifact
	for _, a := range artifacts {
		if a.RunID == runID && a.Kind == "ui-result" {
			if original != nil {
				return result, errors.New("ambiguous original UI result evidence")
			}
			copy := a
			original = &copy
		}
	}
	if original == nil {
		return result, errors.New("original UI result evidence is missing; helper recovery cannot waive incomplete evidence")
	}
	data, err := s.readUIRecoveryEvidence(id, runID, *original)
	if err != nil {
		return result, err
	}
	var previous domain.UIObservation
	if err = json.Unmarshal(data, &previous); err != nil {
		return result, err
	}
	if previous.Status != "" && previous.Status != "uncertain" {
		return result, errors.New("successful UI response has an incomplete evidence barrier; helper recovery cannot waive it")
	}
	recoveryID := newID()
	result.Run.Notes = append(result.Run.Notes, "Explicit observer companion quiescence requested: "+recoveryID+"; original input must not be retried.")
	if err = s.Store.SaveRun(ctx, result.Run); err != nil {
		return result, err
	}
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	result.Observation, err = s.AndroidUI.ObserveUI(bounded, runtime, domain.UIRequest{Version: 1, Operation: "quiesce"})
	cancel()
	secrets := evidence.InheritedSecrets()
	sanitizeUIObservation(&result.Observation, secrets)
	if operationLost(ctx) {
		return result, errors.Join(err, domain.ErrLockLost)
	}
	effectErr := err
	if effectErr != nil {
		effectErr = errors.New(evidence.RedactString(effectErr.Error(), secrets))
	}
	if !result.Observation.Confirmed || result.Observation.Status != "ok" {
		effectErr = errors.Join(effectErr, errors.New("verified observer companion quiescence remains unconfirmed"))
	}
	result.Observation.Binary = nil
	result.Observation.Raw = nil
	record := struct {
		Version         int                  `json:"version"`
		RecoveryID      string               `json:"recovery_id"`
		LeaseID         string               `json:"lease_id"`
		RunID           string               `json:"run_id"`
		Runtime         string               `json:"runtime"`
		Serial          string               `json:"serial"`
		OriginalResult  domain.Artifact      `json:"original_result"`
		OriginalOutcome string               `json:"original_outcome"`
		RecoveredAt     time.Time            `json:"recovered_at"`
		Quiescence      domain.UIObservation `json:"quiescence"`
		Error           string               `json:"error,omitempty"`
	}{Version: 1, RecoveryID: recoveryID, LeaseID: id, RunID: runID, Runtime: runtime.Name, Serial: runtime.Android.Serial, OriginalResult: *original, OriginalOutcome: "failed/uncertain; original result retained; no input retried", RecoveredAt: time.Now().UTC(), Quiescence: result.Observation}
	if effectErr != nil {
		record.Error = effectErr.Error()
	}
	data, err = json.Marshal(record)
	if err != nil {
		return result, errors.Join(effectErr, err)
	}
	directory := filepath.Join(s.Home, "leases", id, "artifacts", runID, recoveryID)
	if err = os.Mkdir(directory, 0700); err != nil {
		return result, errors.Join(effectErr, err)
	}
	path := filepath.Join(directory, "recovery.json")
	if err = evidence.AtomicWrite(path, data, 0600); err != nil {
		return result, errors.Join(effectErr, err)
	}
	digest := sha256.Sum256(data)
	artifact := domain.Artifact{ID: recoveryID, LeaseID: id, RunID: runID, Kind: "ui-recovery", Path: path, Digest: hex.EncodeToString(digest[:]), CreatedAt: time.Now().UTC()}
	// This explicit recovery does not detach finalization from caller cancellation.
	// An interrupted recovery therefore retains its running barrier for inspection.
	if err = s.Store.SaveArtifact(ctx, artifact); err != nil {
		return result, errors.Join(effectErr, err)
	}
	result.Artifacts = []domain.Artifact{artifact}
	if effectErr != nil {
		return result, effectErr
	}
	result.Run.Status = "failed"
	result.Run.ExitCode = 1
	result.Run.FinishedAt = time.Now().UTC()
	result.Run.Notes = append(result.Run.Notes, "Observer companion termination confirmed; original outcome remains failed/uncertain. Recovery evidence: "+recoveryID)
	if err = s.Store.SaveRun(ctx, result.Run); err != nil {
		return result, err
	}
	diagnostics := lease.Diagnostics[:0]
	changed := false
	for _, diagnostic := range lease.Diagnostics {
		stale := "stale command run " + runID + " remains running in registry; process completion is unverified and GC is blocked"
		running := "running command run " + runID + " remains running in registry; process completion is unverified and GC is blocked"
		if diagnostic == stale || diagnostic == running {
			changed = true
			continue
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	if changed {
		lease.Diagnostics = diagnostics
		err = s.persist(ctx, lease)
	}
	return result, err
}

func (s *Service) readUIRecoveryEvidence(id, runID string, a domain.Artifact) ([]byte, error) {
	expected := filepath.Join(s.Home, "leases", id, "artifacts", runID, "result.json")
	if a.LeaseID != id || a.RunID != runID || filepath.Clean(a.Path) != filepath.Clean(expected) {
		return nil, errors.New("original UI result is outside its registered private run path")
	}
	root := filepath.Clean(s.Home)
	for p := expected; p != root; p = filepath.Dir(p) {
		if p == filepath.Dir(p) {
			return nil, errors.New("UI evidence escapes state root")
		}
		st, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 || p == expected && !st.Mode().IsRegular() {
			return nil, errors.New("UI evidence must be regular with no symlink path")
		}
	}
	f, err := os.Open(expected)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 2<<20 {
		return nil, errors.New("original UI result exceeds bound")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != a.Digest {
		return nil, fmt.Errorf("original UI result digest mismatch")
	}
	return data, nil
}
