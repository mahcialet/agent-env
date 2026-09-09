package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/remotesource"
	"github.com/oklog/ulid/v2"
)

// Request contains typed operation options, never a forwarded shell command.
type Request struct {
	Run       string             `json:"run,omitempty"`
	Component string             `json:"component,omitempty"`
	TTL       time.Duration      `json:"ttl,omitempty"`
	Owner     string             `json:"owner,omitempty"`
	Purpose   string             `json:"purpose,omitempty"`
	Mode      string             `json:"mode,omitempty"`
	Force     bool               `json:"force,omitempty"`
	DryRun    bool               `json:"dry_run,omitempty"`
	Name      string             `json:"name,omitempty"`
	UI        app.UIOptions      `json:"ui,omitempty"`
	Browser   app.BrowserOptions `json:"browser,omitempty"`
}
type Response struct {
	Artifacts     []ArtifactReference `json:"artifacts,omitempty"`
	Lease         *domain.Lease       `json:"lease,omitempty"`
	Run           *domain.CommandRun  `json:"run,omitempty"`
	Value         any                 `json:"value,omitempty"`
	Error         string              `json:"error,omitempty"`
	EndpointScope string              `json:"endpoint_scope"`
}
type prepared struct {
	operation protocol.Operation
	request   Request
	options   app.PlanOptions
	source    app.SourceProvider
}

// AppExecutor adapts assigned operations to the existing worker-local lifecycle.
// Factory must return an operation-local Service backed by the worker's store.
type AppExecutor struct {
	Home     string
	CAS      *blobstore.Store
	Factory  func(*domain.Management) *app.Service
	mu       sync.Mutex
	prepared map[string]prepared
}

func strictJSON(data []byte, v any) error {
	if len(data) == 0 {
		data = []byte("{}")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values")
	}
	return nil
}
func management(op protocol.Operation) *domain.Management {
	return &domain.Management{ControllerID: op.ControllerID, HostID: op.HostID, HostInstanceID: op.HostInstanceID, AssignmentEpoch: uint64(op.Epoch)}
}
func operationValid(op protocol.Operation) error {
	if !validIdentity(op.ID) {
		return errors.New("invalid operation identity")
	}
	for _, id := range []string{op.LeaseID} {
		if _, err := ulid.ParseStrict(id); err != nil {
			return errors.New("invalid operation identity")
		}
	}
	for _, id := range []string{op.ControllerID, op.HostID, op.HostInstanceID} {
		if id == "" || len(id) > 256 || strings.ContainsAny(id, "\x00\n\r") {
			return errors.New("invalid management identity")
		}
	}
	if op.Epoch < 1 {
		return errors.New("invalid assignment epoch")
	}
	switch op.Kind {
	case "create", "show", "renew", "reconcile", "destroy", "test", "logs", "artifact", "ui", "ui-recover", "browser":
	default:
		return errors.New("unsupported worker operation")
	}
	return nil
}
func (e *AppExecutor) service(op protocol.Operation, source app.SourceProvider) (*app.Service, error) {
	if e.Factory == nil {
		return nil, errors.New("worker service factory unavailable")
	}
	s := e.Factory(management(op))
	if s == nil || s.Store == nil {
		return nil, errors.New("worker service unavailable")
	}
	s.Management = management(op)
	s.Home = e.Home
	if source != nil {
		s.Source = source
	}
	return s, nil
}
func sameManagement(l domain.Lease, op protocol.Operation) error {
	if l.Management == nil || *l.Management != *management(op) {
		return errors.New("lease belongs to another controller assignment")
	}
	return nil
}
func (e *AppExecutor) Prepare(ctx context.Context, op protocol.Operation) error {
	if err := operationValid(op); err != nil {
		return err
	}
	if e.CAS == nil {
		return errors.New("worker blob store unavailable")
	}
	var pkg remotesource.Package
	var req Request
	if op.Kind == "create" {
		var envelope protocol.CreateRequest
		if err := strictJSON(op.Payload, &envelope); err != nil {
			return err
		}
		if err := strictJSON(envelope.Package, &pkg); err != nil {
			return err
		}
		if err := remotesource.Validate(pkg); err != nil {
			return err
		}
		if envelope.OperationID != op.ID || envelope.Stack != pkg.Stack || envelope.ManifestDigest != pkg.ManifestDigest || envelope.PlanDigest != pkg.PlanDigest || envelope.SourceSetDigest != pkg.SourceSetDigest || envelope.RepositoryID != pkg.RepositoryID || envelope.ControlBlobDigest != pkg.ManifestBlobDigest || envelope.AndroidSlots != pkg.AndroidSlots || !sameJSON(envelope.Manifest, pkg.Manifest) || !reflect.DeepEqual(envelope.RequiredCapabilities, pkg.RequiredCapabilities) {
			return errors.New("create envelope differs from committed package")
		}
		if envelope.HostID != "" && envelope.HostID != op.HostID {
			return errors.New("create target host mismatch")
		}
		if len(envelope.Sources) != len(pkg.Sources) {
			return errors.New("source envelope mismatch")
		}
		for i, s := range pkg.Sources {
			got := envelope.Sources[i]
			if got.Alias != s.Alias || got.Commit != s.Commit || got.BundleDigest != s.BlobDigest {
				return errors.New("source envelope mismatch")
			}
		}
		var optionEnvelope struct {
			Options json.RawMessage `json:"options"`
		}
		if err := json.Unmarshal(op.Payload, &optionEnvelope); err != nil {
			return err
		}
		if err := strictJSON(optionEnvelope.Options, &req); err != nil {
			return err
		}
	} else {
		if err := strictJSON(op.Payload, &req); err != nil {
			return err
		}
		s, err := e.service(op, nil)
		if err != nil {
			return err
		}
		lease, err := s.Store.Get(ctx, op.LeaseID)
		if err != nil {
			return err
		}
		if err = sameManagement(lease, op); err != nil {
			return err
		}
		data, err := e.loadPackage(op.LeaseID)
		if err != nil {
			return err
		}
		if err = strictJSON(data, &pkg); err != nil {
			return err
		}
		if pkg.ManifestDigest != lease.ManifestDigest || pkg.SourceSetDigest != lease.SourceSetDigest || pkg.Stack != lease.Stack {
			return errors.New("retained source package differs from lease")
		}
	}
	if err := validateRequest(op, req); err != nil {
		return err
	}
	options, provider, err := remotesource.Materialize(ctx, pkg, e.Home, e.CAS)
	if err != nil {
		return err
	}
	if op.Kind == "create" {
		if err = e.retainPackage(op.LeaseID, pkg); err != nil {
			return err
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.prepared == nil {
		e.prepared = map[string]prepared{}
	}
	e.prepared[op.ID] = prepared{op, req, options, provider}
	return nil
}
func (e *AppExecutor) Execute(ctx context.Context, op protocol.Operation) protocol.Result {
	return e.execute(ctx, op, nil)
}

// ExecuteWithEffectBoundary preserves durable absence proof until Create reaches
// its first lease reservation. All earlier checks remain read-only diagnostics.
func (e *AppExecutor) ExecuteWithEffectBoundary(ctx context.Context, op protocol.Operation, beforeEffects func(context.Context) error) protocol.Result {
	if op.Kind != "create" || beforeEffects == nil {
		return responseResult("failed", Response{}, errors.New("create effect boundary required"))
	}
	return e.execute(ctx, op, beforeEffects)
}
func (e *AppExecutor) execute(ctx context.Context, op protocol.Operation, beforeEffects func(context.Context) error) protocol.Result {
	e.mu.Lock()
	p, ok := e.prepared[op.ID]
	delete(e.prepared, op.ID)
	e.mu.Unlock()
	if !ok || !reflect.DeepEqual(p.operation, op) {
		return responseResult("failed", Response{}, errors.New("operation was not prepared"))
	}
	s, err := e.service(op, p.source)
	if err != nil {
		return responseResult("failed", Response{}, err)
	}
	if op.Kind != "create" {
		lease, e := s.Store.Get(ctx, op.LeaseID)
		if e != nil {
			return responseResult("failed", Response{}, e)
		}
		if e = sameManagement(lease, op); e != nil {
			return responseResult("failed", Response{}, e)
		}
	}
	var out Response
	var lease domain.Lease
	var artifacts []protocol.Blob
	switch op.Kind {
	case "create":
		lease, err = s.Create(ctx, p.options, app.CreateOptions{BeforeReserve: beforeEffects, LeaseID: op.LeaseID, Management: management(op), Owner: p.request.Owner, Purpose: p.request.Purpose, Mode: p.request.Mode, TTL: p.request.TTL})
		out.Lease = &lease
	case "show":
		lease, err = s.Show(ctx, op.LeaseID)
		out.Lease = &lease
	case "renew":
		lease, err = s.Renew(ctx, op.LeaseID, p.request.TTL)
		out.Lease = &lease
	case "reconcile":
		lease, err = s.Reconcile(ctx, op.LeaseID)
		out.Lease = &lease
	case "destroy":
		lease, err = s.Destroy(ctx, op.LeaseID, p.request.Force, p.request.DryRun)
		out.Lease = &lease
	case "test":
		run, runErr := s.Test(ctx, op.LeaseID, p.request.Name)
		err = runErr
		out.Run = &run
	case "ui":
		v, runErr := s.UI(ctx, op.LeaseID, p.request.UI)
		err = runErr
		out.Run = &v.Run
		out.Value = v
	case "ui-recover":
		v, runErr := s.RecoverUI(ctx, op.LeaseID, p.request.Name)
		err = runErr
		out.Value = v
	case "browser":
		v, runErr := s.Browser(ctx, op.LeaseID, p.request.Browser)
		err = runErr
		out.Run = &v.Run
		out.Value = v
	case "logs":
		lease, err = s.Store.Get(ctx, op.LeaseID)
		if err == nil {
			err = sameManagement(lease, op)
		}
		if err == nil {
			out.Value, err = e.logs(ctx, s, op, p.request)
		}
	case "artifact":
		out.Value, artifacts, err = e.artifacts(ctx, s, op, p.request.Name, p.request.Run)
	}
	if out.Lease == nil {
		if l, getErr := s.Store.Get(ctx, op.LeaseID); getErr == nil && sameManagement(l, op) == nil {
			lease = l
			out.Lease = &lease
		}
	}
	if out.Run != nil && out.Run.ID != "" && out.Run.Status != "running" {
		refs, blobs, e := e.artifacts(ctx, s, op, "", out.Run.ID)
		if e != nil {
			err = errors.Join(err, e)
		} else {
			out.Artifacts = refs
			artifacts = append(artifacts, blobs...)
		}
	}
	state := "completed"
	if err != nil {
		state = "failed"
	}
	if out.Run != nil && out.Run.Status == "running" {
		state = "uncertain"
	}
	result := responseResult(state, out, err)
	if len(artifacts) > 1024 {
		result = responseResult("failed", Response{}, errors.New("artifact reference limit exceeded; retained evidence remains on worker"))
		artifacts = nil
	}
	result.Artifacts = artifacts
	if out.Lease != nil {
		result.LocalState = out.Lease.Observed
	}
	result.CleanupConfirmed = op.Kind == "destroy" && !p.request.DryRun && err == nil && lease.Observed == "released"
	return result
}
func (e *AppExecutor) Recover(ctx context.Context, op protocol.Operation) protocol.Result {
	if err := operationValid(op); err != nil {
		return responseResult("uncertain", Response{}, err)
	}
	s, err := e.service(op, nil)
	if err != nil {
		return responseResult("uncertain", Response{}, err)
	}
	lease, err := s.Store.Get(ctx, op.LeaseID)
	if err != nil {
		return responseResult("uncertain", Response{}, err)
	}
	if err = sameManagement(lease, op); err != nil {
		return responseResult("uncertain", Response{}, err)
	}
	result := responseResult("uncertain", Response{Lease: &lease}, errors.New("previous execution may have taken effect; mutation was not replayed"))
	result.LocalState = lease.Observed
	if op.Kind == "destroy" && lease.Observed == "released" {
		result = responseResult("completed", Response{Lease: &lease}, nil)
		result.LocalState = lease.Observed
		result.CleanupConfirmed = true
	}
	return result
}
func responseResult(state string, out Response, err error) protocol.Result {
	out.EndpointScope = "worker-local"
	if err != nil {
		out.Error = evidence.RedactString(err.Error(), evidence.InheritedSecrets())
	}
	data, e := json.Marshal(out)
	if e != nil {
		data = []byte(`{"error":"cannot encode worker result","endpoint_scope":"worker-local"}`)
		state = "failed"
	}
	if len(data) > 4<<20 {
		if state != "uncertain" {
			state = "failed"
		}
		data = []byte(`{"error":"worker response exceeds metadata transfer limit; operation was not replayed and evidence remains on worker; request a narrower component, run, or artifact","endpoint_scope":"worker-local"}`)
	}
	return protocol.Result{State: state, Payload: data}
}
func (e *AppExecutor) packageRoot() (string, error) {
	root := filepath.Join(e.Home, "remote-lease-inputs")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("unsafe retained input directory")
	}
	return root, nil
}
func (e *AppExecutor) retainPackage(id string, p remotesource.Package) error {
	root, err := e.packageRoot()
	if err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err == nil {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		err = decoder.Decode(&value)
		if err == nil {
			data, err = json.Marshal(value)
		}
	}
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp(root, ".incoming-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if err = os.WriteFile(filepath.Join(temp, "package.json"), data, 0600); err != nil {
		return err
	}
	if err = os.Rename(temp, filepath.Join(root, id)); err != nil {
		existing, e := e.loadPackage(id)
		if e != nil {
			return e
		}
		if !bytes.Equal(existing, data) {
			return errors.New("lease source package changed")
		}
	}
	return nil
}
func (e *AppExecutor) loadPackage(id string) ([]byte, error) {
	root, err := e.packageRoot()
	if err != nil {
		return nil, err
	}
	return readOwned(root, id+"/package.json", 8<<20)
}
func readOwned(root, relative string, limit int64) ([]byte, error) {
	if relative == "" || strings.ContainsAny(relative, "\\:\x00") || strings.HasPrefix(relative, "/") {
		return nil, errors.New("invalid retained artifact path")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	parts := strings.Split(relative, "/")
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return nil, errors.New("unsafe retained artifact path")
		}
		st, e := r.Lstat(filepath.Join(parts[:i+1]...))
		if e != nil {
			return nil, e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("retained artifact symlink refused")
		}
		if i == len(parts)-1 && !st.Mode().IsRegular() {
			return nil, errors.New("retained artifact is not a regular file")
		}
	}
	f, err := r.Open(filepath.FromSlash(relative))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("retained artifact exceeds transfer limit")
	}
	return data, nil
}

type ArtifactReference struct {
	Artifact domain.Artifact `json:"artifact"`
	Blob     protocol.Blob   `json:"blob"`
}

func (e *AppExecutor) artifacts(ctx context.Context, s *app.Service, op protocol.Operation, name, runFilter string) ([]ArtifactReference, []protocol.Blob, error) {
	lease, err := s.Store.Get(ctx, op.LeaseID)
	if err != nil {
		return nil, nil, err
	}
	if err = sameManagement(lease, op); err != nil {
		return nil, nil, err
	}
	registered, err := s.Store.Artifacts(ctx, op.LeaseID)
	if err != nil {
		return nil, nil, err
	}
	var refs []ArtifactReference
	var blobs []protocol.Blob
	for _, a := range registered {
		if a.LeaseID != op.LeaseID {
			return nil, nil, errors.New("foreign artifact in lease registry")
		}
		if runFilter != "" && a.RunID != runFilter {
			continue
		}
		if name != "" && a.ID != name {
			continue
		}
		prefix := "leases/" + op.LeaseID + "/"
		relative, err := filepath.Rel(e.Home, a.Path)
		if err != nil {
			return nil, nil, err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, prefix) {
			return nil, nil, errors.New("artifact is outside assigned lease")
		}
		data, err := readOwned(e.Home, relative, blobstore.ArtifactLimit)
		if err != nil {
			return nil, nil, err
		}
		b, err := e.CAS.Put(ctx, bytes.NewReader(data), a.Digest, blobstore.ArtifactLimit)
		if err != nil {
			return nil, nil, err
		}
		wire := protocol.Blob{Digest: b.Digest, Size: b.Size}
		refs = append(refs, ArtifactReference{a, wire})
		blobs = append(blobs, wire)
	}
	if name != "" && len(refs) == 0 {
		return nil, nil, fmt.Errorf("registered artifact not found")
	}
	return refs, blobs, nil
}

func (e *AppExecutor) logs(ctx context.Context, s *app.Service, op protocol.Operation, req Request) (map[string]string, error) {
	lease, err := s.Store.Get(ctx, op.LeaseID)
	if err != nil {
		return nil, err
	}
	if err = sameManagement(lease, op); err != nil {
		return nil, err
	}
	if req.Run == "" {
		component := req.Component
		if component == "" {
			component = req.Name
		}
		return s.RuntimeLogEntries(ctx, lease, component)
	}
	runs, err := s.Store.Runs(ctx, op.LeaseID)
	if err != nil {
		return nil, err
	}
	var run *domain.CommandRun
	for i := range runs {
		if runs[i].ID == req.Run && runs[i].LeaseID == op.LeaseID {
			run = &runs[i]
			break
		}
	}
	if run == nil {
		return nil, errors.New("command run not found for lease")
	}
	registered, err := s.Store.Artifacts(ctx, op.LeaseID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for name, path := range map[string]string{"stdout": run.StdoutPath, "stderr": run.StderrPath} {
		if path == "" {
			continue
		}
		found := false
		for _, a := range registered {
			if a.LeaseID == op.LeaseID && a.RunID == run.ID && filepath.Clean(a.Path) == filepath.Clean(path) {
				relative, err := filepath.Rel(e.Home, path)
				if err != nil {
					return nil, err
				}
				relative = filepath.ToSlash(relative)
				if !strings.HasPrefix(relative, "leases/"+op.LeaseID+"/") {
					return nil, errors.New("run log is outside assigned lease")
				}
				data, err := readOwned(e.Home, relative, 1<<20)
				if err != nil {
					return nil, err
				}
				if _, err = e.CAS.Put(ctx, bytes.NewReader(data), a.Digest, blobstore.ArtifactLimit); err != nil {
					return nil, err
				}
				out[name] = string(data)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("run log lacks registered retained evidence")
		}
	}
	return out, nil
}

func sameJSON(a, b []byte) bool {
	var x, y any
	first := json.NewDecoder(bytes.NewReader(a))
	first.UseNumber()
	second := json.NewDecoder(bytes.NewReader(b))
	second.UseNumber()
	return first.Decode(&x) == nil && second.Decode(&y) == nil && reflect.DeepEqual(x, y)
}

func validateRequest(op protocol.Operation, req Request) error {
	if req.Run != "" && req.Component != "" {
		return errors.New("run and component select different log sources")
	}
	if op.Kind != "logs" && req.Component != "" {
		return errors.New("component filter is only supported for logs")
	}
	if op.Kind != "logs" && op.Kind != "artifact" && req.Run != "" {
		return errors.New("run filter is only supported for logs and artifacts")
	}
	if req.TTL < 0 {
		return errors.New("negative TTL")
	}
	if op.Kind == "test" && req.Name == "" {
		return errors.New("test name is required")
	}
	return nil
}
