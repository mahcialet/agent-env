package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/remotesource"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"github.com/oklog/ulid/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type executorProcess struct {
	starts, stops, logs int
	running             bool
}

func (p *executorProcess) Prepare(_ context.Context, r domain.Runtime) (domain.Runtime, error) {
	return r, nil
}
func (p *executorProcess) Start(_ context.Context, r domain.Runtime) (domain.Runtime, error) {
	p.starts++
	p.running = true
	r.Process.State = "running"
	return r, nil
}
func (p *executorProcess) Inspect(context.Context, domain.Runtime) (app.RuntimeObservation, error) {
	return app.RuntimeObservation{Ready: p.running, Exists: p.running}, nil
}
func (p *executorProcess) Logs(context.Context, domain.Runtime) (string, error) {
	p.logs++
	return "provider-redacted-log", nil
}
func (p *executorProcess) Destroy(context.Context, domain.Runtime) error {
	p.stops++
	p.running = false
	return nil
}
func exGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, b)
	}
}
func executorFixture(t *testing.T) (*AppExecutor, protocol.Operation, *executorProcess, *sqlite.Store) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "source")
	exGit(t, "", "init", repo)
	text := `version: 1
sources:
 app: {repository: ., default_ref: HEAD}
runtimes:
 api:
  type: process
  source: app
  working_directory: .
  command: [go, version]
components:
 api: {runtime: api}
stacks:
 local: {roots: [api]}
`
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	exGit(t, repo, "add", ".agent-env.yaml")
	exGit(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "manifest")
	home := filepath.Join(t.TempDir(), "worker")
	cas, err := blobstore.New(filepath.Join(home, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := remotesource.Build(context.Background(), app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	process := &executorProcess{}
	e := &AppExecutor{Home: home, CAS: cas, Factory: func(m *domain.Management) *app.Service {
		return &app.Service{Home: home, Store: db, Process: process, Management: m}
	}}
	op := protocol.Operation{ID: "readable-retry-operation", LeaseID: ulid.Make().String(), ControllerID: "controller-a", HostID: "linux-a", HostInstanceID: "instance-a", Epoch: 1, Kind: "create"}
	pbytes, _ := json.Marshal(pkg)
	opts, _ := json.Marshal(Request{Owner: "fixture", TTL: time.Hour})
	env := protocol.CreateRequest{OperationID: op.ID, RepositoryID: pkg.RepositoryID, Stack: pkg.Stack, ManifestDigest: pkg.ManifestDigest, PlanDigest: pkg.PlanDigest, SourceSetDigest: pkg.SourceSetDigest, Manifest: pkg.Manifest, RequiredCapabilities: pkg.RequiredCapabilities, AndroidSlots: pkg.AndroidSlots, Package: pbytes, ControlBlobDigest: pkg.ManifestBlobDigest, Options: opts}
	for _, s := range pkg.Sources {
		env.Sources = append(env.Sources, protocol.Source{Alias: s.Alias, Commit: s.Commit, BundleDigest: s.BlobDigest})
	}
	op.Payload, _ = json.Marshal(env)
	return e, op, process, db
}
func executePrepared(t *testing.T, e *AppExecutor, op protocol.Operation) protocol.Result {
	t.Helper()
	if err := e.Prepare(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	r := e.Execute(context.Background(), op)
	if r.State != "completed" {
		t.Fatalf("operation %s: %+v %s", op.Kind, r, r.Payload)
	}
	return r
}
func nextOperation(op protocol.Operation, kind string, req Request) protocol.Operation {
	op.ID = ulid.Make().String()
	op.Kind = kind
	op.Payload, _ = json.Marshal(req)
	return op
}
func TestAppExecutorCommittedCreateLifecycleAndRecoveryNoReplay(t *testing.T) {
	ctx := context.Background()
	e, op, process, db := executorFixture(t)
	result := executePrepared(t, e, op)
	var out Response
	if err := json.Unmarshal(result.Payload, &out); err != nil {
		t.Fatal(err)
	}
	if out.Lease == nil || out.Lease.ID != op.LeaseID || sameManagement(*out.Lease, op) != nil || process.starts != 1 || out.EndpointScope != "worker-local" {
		t.Fatalf("create outcome %+v", out)
	}
	for _, kind := range []string{"test", "ui", "browser"} {
		recovery := nextOperation(op, kind, Request{Name: "must-not-run"})
		r := e.Recover(ctx, recovery)
		if r.State != "uncertain" || process.starts != 1 || process.stops != 0 {
			t.Fatal("uncertain operation replayed")
		}
	}
	executePrepared(t, e, nextOperation(op, "show", Request{}))
	executePrepared(t, e, nextOperation(op, "renew", Request{TTL: 2 * time.Hour}))
	r := executePrepared(t, e, nextOperation(op, "logs", Request{Component: "api"}))
	if !bytes.Contains(r.Payload, []byte("provider-redacted-log")) || process.logs != 1 {
		t.Fatal("logs bypassed runtime provider")
	}
	// Duplicate Execute cannot cross the prepared/effect boundary a second time.
	if r = e.Execute(ctx, op); r.State != "failed" || process.starts != 1 {
		t.Fatal("executor replayed create")
	}
	destroy := nextOperation(op, "destroy", Request{})
	r = executePrepared(t, e, destroy)
	if !r.CleanupConfirmed || process.stops != 1 {
		t.Fatalf("cleanup %+v", r)
	}
	r = e.Recover(ctx, destroy)
	if r.State != "completed" || !r.CleanupConfirmed || process.stops != 1 {
		t.Fatal("known cleanup recovery replayed effects")
	}
	lease, err := db.Get(ctx, op.LeaseID)
	if err != nil || lease.Observed != "released" {
		t.Fatal("cleanup not durable")
	}
}
func TestAppExecutorRejectsEnvelopeAndAuthorityBeforeEffects(t *testing.T) {
	ctx := context.Background()
	e, op, process, _ := executorFixture(t)
	for _, field := range []string{"manifest_digest", "plan_digest", "source_set_digest", "repository_id", "control_blob_digest", "stack", "operation_id", "required_capabilities", "android_slots", "options"} {
		t.Run(field, func(t *testing.T) {
			var payload map[string]any
			json.Unmarshal(op.Payload, &payload)
			switch field {
			case "required_capabilities":
				payload[field] = []string{"git"}
			case "android_slots":
				payload[field] = 1
			case "options":
				payload[field] = map[string]any{"management": map[string]any{"controller_id": "other"}}
			default:
				payload[field] = "tampered"
			}
			bad := op
			bad.Payload, _ = json.Marshal(payload)
			if err := e.Prepare(ctx, bad); err == nil {
				t.Fatal("tampered envelope accepted")
			}
			if process.starts != 0 {
				t.Fatal("rejected input performed runtime effect")
			}
		})
	}
	executePrepared(t, e, op)
	next := nextOperation(op, "destroy", Request{})
	next.Epoch = 2
	if err := e.Prepare(ctx, next); err == nil {
		t.Fatal("foreign assignment prepared")
	}
	if r := e.Recover(ctx, next); r.State != "uncertain" || bytes.Contains(r.Payload, []byte(`"lease":`)) {
		t.Fatal("foreign recovery exposed lease")
	}
	if process.stops != 0 {
		t.Fatal("foreign assignment stopped runtime")
	}
	// Recheck management after Prepare, before even a read-only disclosure.
	show := nextOperation(op, "show", Request{})
	if err := e.Prepare(ctx, show); err != nil {
		t.Fatal(err)
	}
	show.Epoch++

	if r := e.Execute(ctx, show); r.State != "failed" || bytes.Contains(r.Payload, []byte(`"lease":`)) {
		t.Fatal("execution ignored changed management")
	}
}
func TestAppExecutorRegisteredArtifactDigestPathAndRunLogs(t *testing.T) {
	ctx := context.Background()
	e, op, _, db := executorFixture(t)
	executePrepared(t, e, op)
	run := domain.CommandRun{ID: ulid.Make().String(), LeaseID: op.LeaseID, Name: "fixture", Status: "passed", StartedAt: time.Now(), FinishedAt: time.Now()}
	data := []byte("launch-time-redacted output")
	path := filepath.Join(e.Home, "leases", op.LeaseID, "fixture.log")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	a := domain.Artifact{ID: ulid.Make().String(), LeaseID: op.LeaseID, RunID: run.ID, Kind: "stdout", Path: path, Digest: hex.EncodeToString(sum[:]), CreatedAt: time.Now()}
	run.StdoutPath = path
	if err := db.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveArtifact(ctx, a); err != nil {
		t.Fatal(err)
	}
	r := executePrepared(t, e, nextOperation(op, "artifact", Request{Run: run.ID}))
	if len(r.Artifacts) != 1 || r.Artifacts[0].Digest != a.Digest {
		t.Fatal("registered artifact missing")
	}
	r = executePrepared(t, e, nextOperation(op, "logs", Request{Run: run.ID}))
	if !bytes.Contains(r.Payload, data) {
		t.Fatal("retained redacted log missing")
	}
	if err := os.WriteFile(path, []byte("tampered file"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := nextOperation(op, "artifact", Request{Name: a.ID})
	if err := e.Prepare(ctx, bad); err != nil {
		t.Fatal(err)
	}
	if r = e.Execute(ctx, bad); r.State != "failed" || len(r.Artifacts) != 0 {
		t.Fatal("corrupt artifact published")
	}
	a.ID = ulid.Make().String()
	a.Path = filepath.Join(t.TempDir(), "outside")
	os.WriteFile(a.Path, data, 0600)
	db.SaveArtifact(ctx, a)
	bad = nextOperation(op, "artifact", Request{Name: a.ID})
	if err := e.Prepare(ctx, bad); err != nil {
		t.Fatal(err)
	}
	if r = e.Execute(ctx, bad); r.State != "failed" || len(r.Artifacts) != 0 {
		t.Fatal("outside artifact published")
	}
}
func TestOwnedReadRejectsTraversalSymlinksAndOversize(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "data"), []byte("abcd"), 0600)
	for _, p := range []string{"../data", "/data", "a/../data", "a\\data", "data:stream"} {
		if _, err := readOwned(root, p, 4); err == nil {
			t.Fatal("unsafe path accepted", p)
		}
	}
	if _, err := readOwned(root, "data", 3); err == nil {
		t.Fatal("oversize accepted")
	}
	if b, err := readOwned(root, "data", 4); err != nil || string(b) != "abcd" {
		t.Fatal("exact limit failed")
	}
	if err := os.Symlink(filepath.Join(root, "data"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := readOwned(root, "link", 4); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestExecutorResultBoundRetainsUncertainty(t *testing.T) {
	for _, state := range []string{"completed", "failed", "uncertain"} {
		r := responseResult(state, Response{Value: string(bytes.Repeat([]byte("x"), 4<<20))}, nil)
		if len(r.Payload) > 4<<20 || !bytes.Contains(r.Payload, []byte("exceeds metadata")) {
			t.Fatal("unbounded result")
		}
		if state == "uncertain" && r.State != "uncertain" {
			t.Fatal("lost uncertainty")
		}
	}
}

func TestCreateValidationFailureRetainsDurableNoEffectProof(t *testing.T) {
	for _, failure := range []string{"mode", "ttl", "provider"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			e, op, process, db := executorFixture(t)
			j, err := OpenJournal(e.Home, op.HostID)
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			if err = j.Bind(ctx, op.ControllerID); err != nil {
				t.Fatal(err)
			}
			op.HostInstanceID = j.Identity.HostInstanceID
			var envelope protocol.CreateRequest
			if err = json.Unmarshal(op.Payload, &envelope); err != nil {
				t.Fatal(err)
			}
			var request Request
			if err = json.Unmarshal(envelope.Options, &request); err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "mode":
				request.Mode = "unsupported"
			case "ttl":
				request.TTL = 100000 * time.Hour
			case "provider":
				e.Factory = func(m *domain.Management) *app.Service { return &app.Service{Store: db, Management: m} }
			}
			envelope.Options, _ = json.Marshal(request)
			op.Payload, _ = json.Marshal(envelope)
			receipt, err := j.Receive(ctx, op)
			if err != nil {
				t.Fatal(err)
			}
			runner := Runner{Journal: j, Executor: e, Transport: &fakeTransport{}}
			if err = runner.handle(ctx, receipt); err != nil {
				t.Fatal(err)
			}
			stored, err := j.Get(ctx, op.ID)
			if err != nil || stored.Result == nil || stored.Result.State != "failed" {
				t.Fatalf("validation result: %+v %v", stored, err)
			}
			if process.starts != 0 {
				t.Fatal("validation started runtime")
			}
			if _, err = db.Get(ctx, op.LeaseID); err == nil {
				t.Fatal("validation reserved lease")
			}
			destroy := op
			destroy.ID = "cleanup-validation"
			destroy.Kind = "destroy"
			destroy.Payload = json.RawMessage(`{}`)
			proof, err := j.ProvesNoEffects(ctx, destroy)
			if err != nil || !proof {
				t.Fatalf("pre-Reserve validation lacks durable proof: %v %v", proof, err)
			}
			receipt, err = j.Receive(ctx, destroy)
			if err != nil {
				t.Fatal(err)
			}
			if err = runner.handle(ctx, receipt); err != nil {
				t.Fatal(err)
			}
			stored, err = j.Get(ctx, destroy.ID)
			if err != nil || stored.Result == nil || !stored.Result.CleanupConfirmed || stored.Result.LocalState != "released" {
				t.Fatalf("proof cleanup: %+v %v", stored, err)
			}
		})
	}
}

type reserveBoundaryStore struct {
	app.Store
	reserve func(context.Context, domain.Lease, int) error
}

func (s reserveBoundaryStore) Reserve(ctx context.Context, l domain.Lease, max int) error {
	return s.reserve(ctx, l, max)
}
func TestCreateEffectBoundaryPrecedesReserveAndKeepsAmbiguousFailure(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("fence-blocked-%v", blocked), func(t *testing.T) {
			ctx := context.Background()
			e, op, process, db := executorFixture(t)
			j, err := OpenJournal(e.Home, op.HostID)
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			if err = j.Bind(ctx, op.ControllerID); err != nil {
				t.Fatal(err)
			}
			op.HostInstanceID = j.Identity.HostInstanceID
			reserves := 0
			e.Factory = func(m *domain.Management) *app.Service {
				return &app.Service{Store: reserveBoundaryStore{Store: db, reserve: func(ctx context.Context, l domain.Lease, max int) error {
					reserves++
					receipt, err := j.Get(ctx, op.ID)
					if err != nil || receipt.State != "effect_started" {
						t.Fatalf("Reserve before durable boundary: %+v %v", receipt, err)
					}
					return errors.New("ambiguous reserve failure")
				}}, Process: process, Management: m}
			}
			receipt, err := j.Receive(ctx, op)
			if err != nil {
				t.Fatal(err)
			}
			runner := Runner{Journal: j, Executor: e, Transport: &fakeTransport{}}
			fences := 0
			err = runner.handleWithFence(ctx, receipt, func() error {
				fences++
				if blocked && fences == 2 {
					return errors.New("fenced immediately before reservation")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if process.starts != 0 {
				t.Fatal("failed boundary started process")
			}
			stored, resultErr := j.Get(ctx, op.ID)
			wantState := "uncertain"
			if blocked {
				wantState = "failed"
			}
			if resultErr != nil || stored.Result == nil || stored.Result.State != wantState {
				t.Fatalf("boundary result: %+v %v want %s", stored, resultErr, wantState)
			}

			if (blocked && reserves != 0) || (!blocked && reserves != 1) {
				t.Fatalf("reserve count %d blocked=%v", reserves, blocked)
			}
			destroy := op
			destroy.ID = "destroy-after-boundary"
			destroy.Kind = "destroy"
			destroy.Payload = json.RawMessage(`{}`)
			proof, err := j.ProvesNoEffects(ctx, destroy)
			if err != nil || proof != blocked {
				t.Fatalf("durable boundary proof=%v blocked=%v err=%v", proof, blocked, err)
			}
		})
	}
}

type compensatedCreateProcess struct{ *executorProcess }

func (p compensatedCreateProcess) Start(_ context.Context, r domain.Runtime) (domain.Runtime, error) {
	p.starts++
	return r, errors.New("fixture process start failed")
}
func TestCompensatedCreateResultDeliversUncertaintyBeforeExplicitDestroy(t *testing.T) {
	ctx := context.Background()
	e, op, process, db := executorFixture(t)
	e.Factory = func(m *domain.Management) *app.Service {
		return &app.Service{Store: db, Process: compensatedCreateProcess{process}, Management: m}
	}
	j, err := OpenJournal(e.Home, op.HostID)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err = j.Bind(ctx, op.ControllerID); err != nil {
		t.Fatal(err)
	}
	op.HostInstanceID = j.Identity.HostInstanceID
	receipt, err := j.Receive(ctx, op)
	if err != nil {
		t.Fatal(err)
	}
	transport := &createResultReviewTransport{}
	runner := Runner{Journal: j, Executor: e, Transport: transport}
	if err = runner.handle(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	stored, err := j.Get(ctx, op.ID)
	if err != nil || stored.State != "completed" || stored.Result == nil {
		t.Fatalf("create receipt not acknowledged: %s %v", stored.State, err)
	}
	var response Response
	if err = json.Unmarshal(stored.Result.Payload, &response); err != nil {
		t.Fatal(err)
	}
	if response.Lease == nil || response.Lease.Observed != "released" {
		t.Fatalf("fixture did not compensate create: %+v", response.Lease)
	}
	if stored.Result.State != "uncertain" || stored.Result.CleanupConfirmed || stored.Result.LocalState == "released" {
		t.Fatalf("create falsely finalized assignment: state=%s local=%s cleanup=%v", stored.Result.State, stored.Result.LocalState, stored.Result.CleanupConfirmed)
	}
	recovered := e.Recover(ctx, op)
	if recovered.State != "uncertain" || recovered.CleanupConfirmed || recovered.LocalState == "released" {
		t.Fatalf("recovery cannot assert create release: %+v", recovered)
	}
	destroy := op
	destroy.ID = "destroy-compensated-create"
	destroy.Kind = "destroy"
	destroy.Payload = json.RawMessage(`{}`)
	receipt, err = j.Receive(ctx, destroy)
	if err != nil {
		t.Fatal(err)
	}
	if err = runner.handle(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	stored, err = j.Get(ctx, destroy.ID)
	if err != nil || stored.Result == nil || !stored.Result.CleanupConfirmed || stored.Result.State != "completed" {
		t.Fatalf("explicit destroy missing release proof: %v", err)
	}
	if process.starts != 1 {
		t.Fatalf("create replayed effects: starts=%d", process.starts)
	}
}

type createResultReviewTransport struct{ fakeTransport }

func (t *createResultReviewTransport) Complete(_ context.Context, r protocol.Result) error {
	if r.LocalState == "released" && (!r.CleanupConfirmed || r.State == "uncertain") {
		return errors.New("release requires cleanup proof")
	}
	t.acks++
	return nil
}
