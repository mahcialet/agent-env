package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/stack"
)

var ErrTestFailed = errors.New("named test failed")

// Test executes only an argv declared by the lease's immutable manifest. The
// durable operation lock prevents cleanup while a test still owns its worktree.
func (s *Service) Test(ctx context.Context, leaseID, name string) (run domain.CommandRun, err error) {
	if s.Store == nil || s.Source == nil || s.Runner == nil {
		return run, errors.New("store, source and command runner are required")
	}
	ctx, release, err := s.acquireManagedOperation(ctx, leaseID)
	if err != nil {
		return run, err
	}
	defer func() { err = errors.Join(err, release()) }()
	lease, err := s.Store.Get(ctx, leaseID)
	if err != nil {
		return run, err
	}
	if lease.Desired != "active" || lease.Observed != "ready" {
		return run, fmt.Errorf("named tests require a ready lease; observed %s", lease.Observed)
	}
	if !lease.ExpiresAt.After(time.Now()) {
		return run, errors.New("lease expired; renew it before running a test")
	}
	var manifest config.Manifest
	if err = json.Unmarshal(lease.Manifest, &manifest); err != nil {
		return run, fmt.Errorf("decode pinned manifest: %w", err)
	}
	if err = config.Validate(&manifest); err != nil {
		return run, err
	}
	if config.Digest(&manifest) != lease.ManifestDigest {
		return run, errors.New("pinned manifest digest mismatch")
	}
	spec, ok := manifest.Tests[name]
	if !ok {
		return run, fmt.Errorf("unknown named test %q", name)
	}
	required, err := stack.Resolve(&manifest, spec.Stack)
	if err != nil {
		return run, err
	}
	selected := make(map[string]bool)
	for _, component := range lease.Components {
		selected[component.Name] = true
	}
	for _, component := range required {
		if !selected[component] {
			return run, fmt.Errorf("test %s requires stack %s component %s, absent from this lease", name, spec.Stack, component)
		}
	}
	var source domain.Source
	for _, candidate := range lease.Sources {
		if candidate.Alias == spec.Source {
			source = candidate
			break
		}
	}
	if source.WorktreePath == "" {
		return run, fmt.Errorf("test source %s was not allocated", spec.Source)
	}
	dir, err := paths.Within(source.WorktreePath, spec.WorkingDirectory)
	if err != nil {
		return run, err
	}
	if err = paths.ValidateExecutionDirectory(dir); err != nil {
		return run, err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return run, err
	}
	if !info.IsDir() {
		return run, errors.New("test working_directory is not a directory")
	}
	env := make(map[string]string, len(spec.Env))
	for key, value := range spec.Env {
		env[key], err = expandTestValue(value, lease.ID, func(name string) (string, error) { return s.androidTestSerial(ctx, lease, name) })
		if err != nil {
			return run, fmt.Errorf("test environment %s: %w", key, err)
		}
	}
	argv := make([]string, len(spec.Command))
	for i, value := range spec.Command {
		argv[i], err = expandTestValue(value, lease.ID, func(name string) (string, error) { return s.androidTestSerial(ctx, lease, name) })
		if err != nil {
			return run, fmt.Errorf("test argv %d: %w", i, err)
		}
	}
	secrets := append(evidence.InheritedSecrets(), evidence.Secrets(env)...)
	if err = s.inspectTestSources(ctx, &lease, secrets); err != nil {
		return run, err
	}
	timeout := 10 * time.Minute
	if spec.Timeout != "" {
		timeout, err = time.ParseDuration(spec.Timeout)
		if err != nil || timeout <= 0 {
			return run, errors.New("test timeout must be a positive duration")
		}
	}
	lease.HeartbeatAt = time.Now().UTC()
	if err = s.persist(ctx, lease); err != nil {
		return run, err
	}
	runID := newID()
	directory := filepath.Join(s.Home, "leases", lease.ID, "artifacts", runID)
	if err = os.MkdirAll(directory, 0700); err != nil {
		return run, err
	}
	run = domain.CommandRun{ID: runID, LeaseID: lease.ID, Name: name, Source: source.Alias, Directory: dir, StartedAt: time.Now().UTC(), ExitCode: -1, Status: "running", StdoutPath: filepath.Join(directory, "stdout.log"), StderrPath: filepath.Join(directory, "stderr.log")}
	if len(lease.Applications) > 0 {
		run.Notes = []string{"Named tests may build and reinstall a test APK. Creation APK digest is provenance only; this run does not prove execution of that exact APK."}
	}
	for _, arg := range argv {
		run.Argv = append(run.Argv, evidence.RedactString(arg, secrets))
	}
	out, err := os.OpenFile(run.StdoutPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return run, err
	}
	defer out.Close()
	stderr, err := os.OpenFile(run.StderrPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return run, err
	}
	defer stderr.Close()
	if err = s.event(ctx, lease.ID, "test_start_requested", name+" "+run.ID); err != nil {
		return run, err
	}
	if err = s.Store.SaveRun(ctx, run); err != nil {
		return run, err
	}
	stdoutDst, stderrDst := io.Writer(out), io.Writer(stderr)
	if s.Stdout != nil {
		stdoutDst = io.MultiWriter(out, s.Stdout)
	}
	if s.Stderr != nil {
		stderrDst = io.MultiWriter(stderr, s.Stderr)
	}
	stdoutRedactor, stderrRedactor := evidence.NewRedactor(stdoutDst, secrets), evidence.NewRedactor(stderrDst, secrets)
	result := execx.Result{ExitCode: -1}
	var commandErr error
	for _, note := range run.Notes {
		if _, err := fmt.Fprintln(stdoutRedactor, note); err != nil {
			commandErr = err
			break
		}
	}
	if commandErr == nil {
		result, commandErr = s.runWithCancellation(ctx, run.ID, execx.Command{Name: argv[0], Args: argv[1:], Dir: dir, Env: env, Timeout: timeout, Stdout: stdoutRedactor, Stderr: stderrRedactor})
	}
	streamErr := errors.Join(stdoutRedactor.Close(), stderrRedactor.Close(), out.Sync(), stderr.Sync(), out.Close(), stderr.Close())
	run.FinishedAt = time.Now().UTC()
	run.ExitCode = result.ExitCode
	run.Status = "passed"
	if commandErr != nil || result.ExitCode != 0 || streamErr != nil {
		run.Status = "failed"
	}
	if errors.Is(commandErr, context.DeadlineExceeded) {
		run.Status = "timed_out"
	}
	if errors.Is(commandErr, context.Canceled) {
		run.Status = "canceled"
	}
	// Final evidence survives cancellation and failed registry writes; the local
	// descriptor is retained alongside the logs for recovery and diagnosis.
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	if operationLost(ctx) {
		data, _ := json.MarshalIndent(run, "", "  ")
		writeErr := evidence.AtomicWrite(filepath.Join(directory, "run.json"), append(data, '\n'), 0600)
		return run, errors.Join(domain.ErrLockLost, commandErr, writeErr)
	}
	var outcomeErrors, finalErrors []error
	finalErrors = append(finalErrors, streamErr)
	if errors.Is(commandErr, execx.ErrProcessTreeUnconfirmed) || errors.Is(commandErr, execx.ErrOutputIncomplete) {
		finalErrors = append(finalErrors, commandErr)
	}
	if commandErr != nil || result.ExitCode != 0 || streamErr != nil {
		outcomeErrors = append(outcomeErrors, fmt.Errorf("%w: %s (exit %d; status %s)", ErrTestFailed, name, run.ExitCode, run.Status))
		if commandErr != nil {
			diagnostic := evidence.RedactString(commandErr.Error(), secrets)
			outcomeErrors = append(outcomeErrors, errors.New(diagnostic))
			finalErrors = append(finalErrors, s.event(finishCtx, lease.ID, "test_command_error", diagnostic))
		}
	}
	if err = s.inspectTestSources(finishCtx, &lease, secrets); err != nil {
		finalErrors = append(finalErrors, err)
	}
	lease.HeartbeatAt = time.Now().UTC()
	finalErrors = append(finalErrors, s.persist(finishCtx, lease))
	data, marshalErr := json.MarshalIndent(run, "", "  ")
	finalErrors = append(finalErrors, marshalErr)
	if marshalErr == nil {
		finalErrors = append(finalErrors, evidence.AtomicWrite(filepath.Join(directory, "run.json"), append(data, '\n'), 0600))
	}
	for _, artifact := range []struct{ kind, path string }{{"test-stdout", run.StdoutPath}, {"test-stderr", run.StderrPath}, {"test-run", filepath.Join(directory, "run.json")}} {
		finalErrors = append(finalErrors, s.recordRunArtifact(finishCtx, run, artifact.kind, artifact.path))
	}
	// A typed executable lookup failure proves no process could produce the
	// declared outputs. Missing outputs remain reported failures, but must not
	// leave an otherwise fully recorded run masquerading as an active process.
	var lookupError *exec.Error
	lookupFailed := result.ExitCode == -1 && errors.As(commandErr, &lookupError)
	for index, path := range spec.Artifacts {
		artifactErr := s.collectTestArtifact(finishCtx, run, source.WorktreePath, path, filepath.Join(directory, fmt.Sprintf("artifact-%d", index+1)), secrets)
		if lookupFailed && errors.Is(artifactErr, os.ErrNotExist) {
			outcomeErrors = append(outcomeErrors, artifactErr)
		} else {
			finalErrors = append(finalErrors, artifactErr)
		}
	}
	finalErrors = append(finalErrors, s.event(finishCtx, lease.ID, "test_finished", name+" "+run.Status+" "+run.ID))
	// The terminal row is the durable cleanup barrier: publish it only after
	// confirmed process termination and every required evidence write succeeds.
	if errors.Join(finalErrors...) == nil {
		finalErrors = append(finalErrors, s.Store.SaveRun(finishCtx, run))
	}
	finalErrors = append(finalErrors, outcomeErrors...)
	if joined := errors.Join(finalErrors...); joined != nil {
		safe := errors.New(evidence.RedactString(joined.Error(), secrets))
		if errors.Is(joined, ErrTestFailed) {
			return run, errors.Join(ErrTestFailed, safe)
		}
		return run, safe
	}
	return run, nil
}

func expandTestValue(value, leaseID string, android ...func(string) (string, error)) (string, error) {
	var result strings.Builder
	for {
		before, after, found := strings.Cut(value, "${")
		result.WriteString(before)
		if !found {
			return result.String(), nil
		}
		expression, rest, closed := strings.Cut(after, "}")
		if !closed {
			return "", errors.New("unterminated interpolation")
		}
		switch {
		case expression == "lease_id":
			result.WriteString(leaseID)
		case strings.HasPrefix(expression, "android:"):
			parts := strings.Split(expression, ":")
			if len(parts) != 3 || parts[1] == "" || parts[2] != "serial" || len(android) == 0 {
				return "", fmt.Errorf("unsupported Android interpolation %q", expression)
			}
			serial, err := android[0](parts[1])
			if err != nil {
				return "", err
			}
			result.WriteString(serial)
		case strings.HasPrefix(expression, "env:"):
			key := strings.TrimPrefix(expression, "env:")
			resolved, ok := os.LookupEnv(key)
			if !ok || key == "" {
				return "", fmt.Errorf("required environment variable %q is not set", key)
			}
			result.WriteString(resolved)
		default:
			return "", fmt.Errorf("unsupported interpolation %q", expression)
		}
		value = rest
	}
}

func (s *Service) inspectTestSources(ctx context.Context, lease *domain.Lease, secrets []string) error {
	for _, source := range lease.Sources {
		observation, err := s.Source.Inspect(ctx, source)
		if err == nil && (!observation.Exists || !observation.Registered || observation.Commit != source.Commit) {
			err = fmt.Errorf("source %s ownership or pinned commit changed", source.Alias)
		}
		if err == nil && observation.TrackedDirty {
			err = fmt.Errorf("source %s has tracked changes; review lease quarantined", source.Alias)
		}
		if err != nil {
			return s.quarantine(ctx, lease, errors.New(evidence.RedactString(err.Error(), secrets)))
		}
	}
	return nil
}

func (s *Service) recordRunArtifact(ctx context.Context, run domain.CommandRun, kind, path string) error {
	digest, err := evidence.FileDigest(path)
	if err != nil {
		return err
	}
	return s.Store.SaveArtifact(ctx, domain.Artifact{ID: newID(), LeaseID: run.LeaseID, RunID: run.ID, Kind: kind, Path: path, Digest: digest, CreatedAt: time.Now().UTC()})
}

func (s *Service) collectTestArtifact(ctx context.Context, run domain.CommandRun, root, relative, destination string, secrets []string) error {
	path, err := paths.Within(root, relative)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("test artifact %s: %w", relative, err)
	}
	if info.IsDir() {
		return filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			sub, err := filepath.Rel(path, current)
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				st, e := os.Stat(current)
				if e != nil {
					return e
				}
				if st.IsDir() {
					return errors.New("artifact directory symlinks are not followed")
				}
			}
			return s.collectTestArtifact(ctx, run, root, filepath.ToSlash(rel), filepath.Join(destination, evidence.RedactString(sub, secrets)), secrets)
		})
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("test artifact %s is not a regular file", relative)
	}
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer output.Close()
	redactor := evidence.NewRedactor(output, secrets)
	_, copyErr := io.Copy(redactor, input)
	if err = errors.Join(copyErr, redactor.Close(), output.Sync(), output.Close()); err != nil {
		return err
	}
	return s.recordRunArtifact(ctx, run, "test-output", destination)
}

func (s *Service) androidTestSerial(ctx context.Context, l domain.Lease, name string) (string, error) {
	r, err := applicationRuntime(l, name)
	if err != nil {
		return "", err
	}
	o, err := s.inspectRuntime(ctx, r)
	if err != nil {
		return "", err
	}
	if !o.Exists || !o.Ready || r.Android.Serial == "" {
		return "", fmt.Errorf("Android runtime %s is not owned and ready", name)
	}
	return r.Android.Serial, nil
}
