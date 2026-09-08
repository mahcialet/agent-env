package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type fakeProcesses struct {
	starts, stops       int
	spec                execx.Command
	id                  execx.ProcessIdentity
	observed            execx.ProcessObservation
	startErr, errorStop error
	hook                func()
}

func (f *fakeProcesses) Start(_ context.Context, s execx.Command, _, _ string) (execx.ProcessIdentity, error) {
	f.starts++
	f.spec = s
	if f.hook != nil {
		f.hook()
	}
	return f.id, f.startErr
}
func (f *fakeProcesses) Alive(context.Context, execx.ProcessIdentity) (bool, error) {
	return f.observed.Alive, nil
}
func (f *fakeProcesses) Stop(context.Context, execx.ProcessIdentity) error { return f.errorStop }
func (f *fakeProcesses) Observe(context.Context, execx.ProcessIdentity) (execx.ProcessObservation, error) {
	return f.observed, nil
}
func (f *fakeProcesses) Terminate(context.Context, execx.ProcessIdentity, time.Duration) error {
	f.stops++
	if f.errorStop == nil {
		f.observed = execx.ProcessObservation{}
	}
	return f.errorStop
}

func fixture(t *testing.T) (Client, domain.Runtime, *fakeProcesses) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source 日本語")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	native := filepath.Join(source, "fixture helper"+filepath.Ext(exe))
	out, err := os.OpenFile(native, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	f := &fakeProcesses{id: execx.ProcessIdentity{PID: 12345, StartID: "test-identity"}, observed: execx.ProcessObservation{Alive: true, RootAlive: true}}
	r := domain.Runtime{Name: "server", LeaseID: "lease-one", Type: "process", Directory: source, Process: &domain.PersistentProcess{WorkingDirectory: ".", Command: []string{"./" + filepath.Base(native)}, Env: map[string]string{}, Ports: map[string]int{}, SourceCommit: strings.Repeat("a", 40), Directory: filepath.Join(root, "private 日本語"), State: "planned"}}
	return Client{Processes: f}, r, f
}
func prepared(t *testing.T) (Client, domain.Runtime, *fakeProcesses) {
	t.Helper()
	c, r, f := fixture(t)
	r, err := c.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return c, r, f
}
func launched(t *testing.T) (Client, domain.Runtime, *fakeProcesses) {
	t.Helper()
	c, r, f := prepared(t)
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return c, r, f
}

func TestStartInterpolatesWithoutSnapshotSecrets(t *testing.T) {
	c, r, f := prepared(t)
	t.Setenv("PROCESS_FIXTURE_SECRET", "private-example-value")
	r.Process.Command = append(r.Process.Command, "${runtime_dir}", "${lease_id}", "${env:PROCESS_FIXTURE_SECRET}")
	r.Process.Env["TOKEN"] = "${env:PROCESS_FIXTURE_SECRET}"
	r.Process.State = "launching"
	got, err := c.Start(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if f.spec.Args[0] != r.Process.StateDirectory || f.spec.Args[1] != r.LeaseID || f.spec.Args[2] != "private-example-value" {
		t.Fatalf("unexpected expanded argv: %q", f.spec.Args)
	}
	if got.Process.Env["TOKEN"] != "${env:PROCESS_FIXTURE_SECRET}" {
		t.Fatal("snapshot stored secret")
	}
	if got.Process.ProcessID != f.id.PID {
		t.Fatal("identity absent")
	}
	if err = os.WriteFile(got.Process.StdoutPath, []byte("ready private-example-value"), 0600); err != nil {
		t.Fatal(err)
	}
	logs, err := c.Logs(context.Background(), got)
	if err != nil || strings.Contains(logs, "private-example-value") {
		t.Fatalf("logs not redacted: %q %v", logs, err)
	}
}

func TestReservedPortOccupationPreventsLaunch(t *testing.T) {
	c, r, f := prepared(t)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	r.Process.Ports["http"] = l.Addr().(*net.TCPAddr).Port
	r.Process.State = "launching"
	r, err = c.Start(context.Background(), r)
	if err == nil || f.starts != 0 || r.Process.State != "prepared" {
		t.Fatalf("unsafe preflight: %v %d %s", err, f.starts, r.Process.State)
	}
	if err = c.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryUsesReceiptAndChecksMismatch(t *testing.T) {
	c, r, f := launched(t)
	r.Process.ProcessID = 0
	r.Process.ProcessStart = ""
	r.Process.State = "launching"
	o, err := c.Inspect(context.Background(), r)
	if err != nil || !o.Exists || !o.Ready {
		t.Fatalf("receipt recovery failed: %+v %v", o, err)
	}
	r.Process.ProcessID = f.id.PID + 1
	r.Process.ProcessStart = f.id.StartID
	if _, err = c.Inspect(context.Background(), r); err == nil {
		t.Fatal("mismatched snapshot accepted")
	}
	if err = c.Destroy(context.Background(), r); err == nil || f.stops != 0 {
		t.Fatal("mismatched process terminated")
	}
}

func TestReceiptFailurePreservesReturnedIdentity(t *testing.T) {
	c, r, f := prepared(t)
	f.hook = func() {
		if err := os.Mkdir(filepath.Join(r.Process.Directory, "launch.json"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if err == nil || r.Process.ProcessID != f.id.PID {
		t.Fatal("receipt failure lost native identity")
	}
	if err = c.Destroy(context.Background(), r); err == nil || f.stops != 0 {
		t.Fatal("ambiguous receipt allowed termination")
	}
}

func TestMissingLaunchingReceiptIsUncertain(t *testing.T) {
	c, r, f := prepared(t)
	r.Process.State = "launching"
	if _, err := c.Inspect(context.Background(), r); err == nil {
		t.Fatal("unrecorded launch reported absent")
	}
	if err := c.Destroy(context.Background(), r); err == nil || f.stops != 0 {
		t.Fatal("unrecorded launch silently cleaned")
	}
}

func TestExitedRootWithDescendantsIsNotReady(t *testing.T) {
	c, r, f := launched(t)
	f.observed = execx.ProcessObservation{Alive: true, RootAlive: false}
	o, err := c.Inspect(context.Background(), r)
	if err != nil || !o.Exists || o.Ready || len(o.Diagnostics) == 0 {
		t.Fatalf("bad observation: %+v %v", o, err)
	}
}

func TestDestroyPreservesLogsAndReceipt(t *testing.T) {
	c, r, f := launched(t)
	if err := c.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if f.stops != 1 {
		t.Fatal("termination absent")
	}
	if _, err := os.Stat(r.Process.StateDirectory); !os.IsNotExist(err) {
		t.Fatal("mutable state remains")
	}
	for _, path := range []string{r.Process.StdoutPath, r.Process.StderrPath, filepath.Join(r.Process.Directory, "launch.json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	o, err := c.Inspect(context.Background(), r)
	if err != nil || o.Exists {
		t.Fatalf("released inspect: %+v %v", o, err)
	}
}

func TestPrepareRejectsUnownedRootAndSourceEscape(t *testing.T) {
	for _, kind := range []string{"root", "cwd", "executable"} {
		t.Run(kind, func(t *testing.T) {
			c, r, _ := fixture(t)
			switch kind {
			case "root":
				if err := os.Mkdir(r.Process.Directory, 0700); err != nil {
					t.Fatal(err)
				}
			case "cwd":
				r.Process.WorkingDirectory = "../"
			case "executable":
				r.Process.Command[0] = "../unowned"
			}
			if _, err := c.Prepare(context.Background(), r); err == nil {
				t.Fatal("unsafe preparation accepted")
			}
		})
	}
}

func TestStartRejectsChangedExecutableAndEvidenceAlias(t *testing.T) {
	for _, kind := range []string{"binary", "logs", "owner"} {
		t.Run(kind, func(t *testing.T) {
			c, r, f := prepared(t)
			switch kind {
			case "binary":
				if err := os.WriteFile(r.Process.Executable, []byte("changed"), 0700); err != nil {
					t.Fatal(err)
				}
			case "logs":
				r.Process.StdoutPath = filepath.Join(t.TempDir(), "foreign.log")
			case "owner":
				r.LeaseID = "other-lease"
			}
			r.Process.State = "launching"
			if _, err := c.Start(context.Background(), r); err == nil || f.starts != 0 {
				t.Fatal("changed evidence accepted")
			}
		})
	}
}

func TestPersistentNativeHelper(t *testing.T) {
	if os.Getenv("AGENT_ENV_PERSISTENT_HELPER") != "1" {
		return
	}
	fmt.Println("native-ready")
	fmt.Fprintln(os.Stderr, "native-error-stream")
	if os.Getenv("AGENT_ENV_PERSISTENT_EXIT") == "1" {
		os.Exit(3)
	}
	for {
		time.Sleep(time.Hour)
	}
}
func TestNativeProcessSurvivesStartAndCleans(t *testing.T) {
	c, r, _ := fixture(t)
	c.Processes = nil
	r.Process.Command = append(r.Process.Command, "-test.run=^TestPersistentNativeHelper$")
	r.Process.Env["AGENT_ENV_PERSISTENT_HELPER"] = "1"
	r, err := c.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	r.Process.State = "launching"
	r, err = c.Start(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Destroy(context.Background(), r); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		logs, err := c.Logs(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(logs, "native-ready") && strings.Contains(logs, "native-error-stream") {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("helper did not emit logs")
		case <-time.After(20 * time.Millisecond):
		}
	}
	o, err := c.Inspect(ctx, r)
	if err != nil || !o.Exists || !o.Ready {
		t.Fatalf("native process not alive: %+v %v", o, err)
	}
}

func TestNativeImmediateExitDoesNotRestart(t *testing.T) {
	c, r, _ := fixture(t)
	c.Processes = nil
	r.Process.Command = append(r.Process.Command, "-test.run=^TestPersistentNativeHelper$")
	r.Process.Env["AGENT_ENV_PERSISTENT_HELPER"] = "1"
	r.Process.Env["AGENT_ENV_PERSISTENT_EXIT"] = "1"
	r, err := c.Prepare(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	r.Process.State = "launching"
	r, err = c.Start(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		o, err := c.Inspect(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if !o.Exists {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("exited helper remains alive")
		case <-time.After(20 * time.Millisecond):
		}
	}
	if err = c.Destroy(ctx, r); err != nil {
		t.Fatal(err)
	}
}

func TestPlannedMissingDirectoryIsAbsent(t *testing.T) {
	c, r, f := fixture(t)
	o, err := c.Inspect(context.Background(), r)
	if err != nil || o.Exists {
		t.Fatal(o, err)
	}
	if err = c.Destroy(context.Background(), r); err != nil || f.stops != 0 {
		t.Fatal(err)
	}
}

func TestNativeStartFailureWithoutIdentityStaysUncertain(t *testing.T) {
	c, r, f := prepared(t)
	f.id = execx.ProcessIdentity{}
	f.startErr = errors.New("launch may have started")
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if err == nil || r.Process.State != "launching" {
		t.Fatal("failed launch lost uncertainty")
	}
	if _, err = c.Inspect(context.Background(), r); err == nil {
		t.Fatal("ambiguous launch reported absent")
	}
}

func TestEndpointUsesReservedPort(t *testing.T) {
	c, r, _ := launched(t)
	r.Process.Ports["web"] = 45678
	o, err := c.Inspect(context.Background(), r)
	if err != nil || o.Endpoints["web"] != net.JoinHostPort("127.0.0.1", strconv.Itoa(45678)) {
		t.Fatalf("endpoint mismatch: %+v %v", o, err)
	}
}

func TestBoundedLogsDoNotExposeSecretAcrossBoundary(t *testing.T) {
	c, r, _ := prepared(t)
	secret := "boundary-secret-value"
	t.Setenv("PROCESS_BOUNDARY_SECRET", secret)
	r.Process.Env["TOKEN"] = "${env:PROCESS_BOUNDARY_SECRET}"
	contents := strings.Repeat("x", 1024*1024-5) + secret + strings.Repeat("x", 4096)
	if err := os.WriteFile(r.Process.StdoutPath, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	logs, err := c.Logs(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs, "bound") || strings.Contains(logs, secret) || !strings.Contains(logs, "[log truncated]") {
		t.Fatalf("boundary redaction/truncation failed (length %d)", len(logs))
	}
	if len(logs) > 1024*1024+200 {
		t.Fatal("logs not bounded")
	}
}

func TestPrepareRejectsSymlinkCWDAndRuntimeRoot(t *testing.T) {
	for _, kind := range []string{"cwd", "runtime"} {
		t.Run(kind, func(t *testing.T) {
			c, r, _ := fixture(t)
			outside := t.TempDir()
			link := filepath.Join(r.Directory, "alias")
			if kind == "runtime" {
				link = r.Process.Directory
			} else {
				r.Process.WorkingDirectory = "alias"
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if _, err := c.Prepare(context.Background(), r); err == nil {
				t.Fatal("symlink boundary escape accepted")
			}
		})
	}
}

func TestLogsRedactOriginalSecretsWhenEnvironmentChanges(t *testing.T) {
	for _, mode := range []string{"changed", "unset"} {
		t.Run(mode, func(t *testing.T) {
			c, r, _ := prepared(t)
			const key = "PERSISTENT_ORIGINAL_SECRET"
			const secret = "original-secret-from-launch"
			t.Setenv(key, secret)
			r.Process.Env["TOKEN"] = "${env:" + key + "}"
			r.Process.State = "launching"
			r, err := c.Start(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(r.Process.StdoutPath, []byte("output "+secret), 0600); err != nil {
				t.Fatal(err)
			}
			contents, err := os.ReadFile(filepath.Join(r.Process.Directory, "launch.json"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(contents), secret) {
				t.Fatal("receipt contains plaintext secret")
			}
			if mode == "changed" {
				t.Setenv(key, "different-current-value")
			} else {
				if err = os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			logs, err := c.Logs(context.Background(), r)
			if err != nil || strings.Contains(logs, secret) || !strings.Contains(logs, "[REDACTED]") {
				t.Fatalf("old secret exposed or logs unavailable: %q %v", logs, err)
			}
		})
	}
}
func TestLogsRequireDurableRedactionVersion(t *testing.T) {
	c, r, _ := launched(t)
	var rec redactionEvidence
	if err := readJSON(filepath.Join(r.Process.Directory, "redaction.json"), &rec); err != nil {
		t.Fatal(err)
	}
	rec.Version = 0
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r.Process.Directory, "redaction.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Logs(context.Background(), r); err == nil {
		t.Fatal("missing redaction contract accepted")
	}
}
func TestReleasedUnlaunchedMissingDirectoryIsAbsent(t *testing.T) {
	c, r, _ := fixture(t)
	r.Process.State = "released"
	o, err := c.Inspect(context.Background(), r)
	if err != nil || o.Exists {
		t.Fatalf("released observation %+v %v", o, err)
	}
}

func TestReceiptFailureStillHasDurableLogRedactionForCompensation(t *testing.T) {
	c, r, f := prepared(t)
	t.Setenv("PROCESS_COMPENSATION_SECRET", "compensation-secret-value")
	r.Process.Env["TOKEN"] = "${env:PROCESS_COMPENSATION_SECRET}"
	f.hook = func() {
		if err := os.Mkdir(filepath.Join(r.Process.Directory, "launch.json"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if err == nil || r.Process.ProcessID == 0 {
		t.Fatal("expected receipt failure with known identity")
	}
	// Simulate removal of the failed write obstruction; no launch receipt exists.
	if err = os.Remove(filepath.Join(r.Process.Directory, "launch.json")); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(r.Process.StdoutPath, []byte("compensation-secret-value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Unsetenv("PROCESS_COMPENSATION_SECRET"); err != nil {
		t.Fatal(err)
	}
	logs, err := c.Logs(context.Background(), r)
	if err != nil || strings.Contains(logs, "compensation-secret-value") {
		t.Fatalf("unsafe compensation logs: %q %v", logs, err)
	}
	if err = c.Destroy(context.Background(), r); err != nil || f.stops != 1 {
		t.Fatal("known launch identity not compensated", err)
	}
}

func TestProvenNoSpawnFailureCanReleaseWithoutQuarantine(t *testing.T) {
	c, r, f := prepared(t)
	f.id = execx.ProcessIdentity{}
	f.startErr = errors.Join(execx.ErrProcessNotStarted, errors.New("permission denied before spawn"))
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if err == nil || r.Process.State != "prepared" {
		t.Fatal("proven no-spawn failure retained launching uncertainty")
	}
	if _, err = c.Logs(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	o, err := c.Inspect(context.Background(), r)
	if err != nil || o.Exists {
		t.Fatalf("no-spawn observation: %+v %v", o, err)
	}
	if err = c.Destroy(context.Background(), r); err != nil || f.stops != 0 {
		t.Fatal("safe no-spawn release failed", err)
	}
}

func TestNativeNonExecutableFileIsProvenNotStarted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable permission prerequisite")
	}
	c, r, _ := prepared(t)
	c.Processes = nil
	if err := os.Chmod(r.Process.Executable, 0600); err != nil {
		t.Fatal(err)
	}
	r.Process.State = "launching"
	r, err := c.Start(context.Background(), r)
	if !errors.Is(err, execx.ErrProcessNotStarted) || r.Process.State != "prepared" {
		t.Fatalf("permission error not classified before spawn: state=%s err=%v", r.Process.State, err)
	}
	if err = c.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestDigestRedactionCoversOverlapsAndBoundsExpansion(t *testing.T) {
	fingerprints, err := fingerprintSecrets([]string{"abc", "bcdef"})
	if err != nil {
		t.Fatal(err)
	}
	redactor, err := newFingerprintRedactor(fingerprints)
	if err != nil {
		t.Fatal(err)
	}
	got, err := redactor.redact(context.Background(), []byte("abcdef end"), 100)
	if err != nil || got != "[REDACTED] end" {
		t.Fatalf("overlapping values not covered: %q %v", got, err)
	}
	fingerprints, err = fingerprintSecrets([]string{"x"})
	if err != nil {
		t.Fatal(err)
	}
	redactor, err = newFingerprintRedactor(fingerprints)
	if err != nil {
		t.Fatal(err)
	}
	got, err = redactor.redact(context.Background(), []byte(strings.Repeat("x", 100)), 100)
	if err != nil || len(got) > 120 || strings.Contains(got, "x") {
		t.Fatalf("redaction expansion not bounded: %d %v", len(got), err)
	}
}
