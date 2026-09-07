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

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/oklog/ulid/v2"
)

type Store interface {
	Reserve(context.Context, domain.Lease, int) error
	Save(context.Context, domain.Lease) error
	Get(context.Context, string) (domain.Lease, error)
	List(context.Context) ([]domain.Lease, error)
	Event(context.Context, domain.Event) error
	Events(context.Context, string) ([]domain.Event, error)
	SaveRun(context.Context, domain.CommandRun) error
	Runs(context.Context, string) ([]domain.CommandRun, error)
	RequestRunCancel(context.Context, string) error
	RunCancellationRequested(context.Context, string) (bool, error)
	SaveArtifact(context.Context, domain.Artifact) error
	Artifacts(context.Context, string) ([]domain.Artifact, error)
	Acquire(context.Context, string, string, time.Duration) (func() error, error)
	AcquireContext(context.Context, string, string, time.Duration) (context.Context, func() error, error)
}

var ErrPrerequisite = errors.New("required external tool unavailable")

type Rendered struct {
	JSON     []byte
	Digest   string
	Services []string
}
type RuntimeObservation struct {
	Ready, Exists bool
	Resources     []domain.Resource
	Diagnostics   []string
	Endpoints     map[string]string
}
type RuntimeProvider interface {
	Doctor(context.Context) (map[string]string, error)
	Render(context.Context, domain.Runtime, []string) (Rendered, error)
	Up(context.Context, domain.Runtime) error
	Down(context.Context, domain.Runtime) error
	Inspect(context.Context, domain.Runtime) (RuntimeObservation, error)
	Logs(context.Context, domain.Runtime) (string, error)
}
type SourceDiff interface {
	Diff(context.Context, domain.Source) (string, error)
}

type Service struct {
	Store             Store
	Source            SourceProvider
	Runtime           RuntimeProvider
	Home              string
	Policy            policy.Policy
	Runner            execx.Runner
	Stdout, Stderr    io.Writer
	ReadinessTimeout  time.Duration
	ReadinessInterval time.Duration
}
type CreateOptions struct {
	Owner, Purpose, Mode string
	TTL                  time.Duration
}

func newID() string { return ulid.Make().String() }
func projectName(id, name string) string {
	sum := sha256.Sum256([]byte(name))
	return "ae-" + strings.ToLower(id) + "-" + hex.EncodeToString(sum[:4])
}
func (s *Service) event(ctx context.Context, id, kind, message string) error {
	return s.Store.Event(ctx, domain.Event{ID: newID(), LeaseID: id, Time: time.Now().UTC(), Type: kind, Message: evidence.RedactString(message, evidence.InheritedSecrets())})
}
func (s *Service) defaults() policy.Policy {
	if s.Policy.MaxTTL == 0 {
		return policy.Defaults()
	}
	return s.Policy
}

func (s *Service) Create(ctx context.Context, o PlanOptions, options CreateOptions) (lease domain.Lease, err error) {
	if s.Store == nil || s.Source == nil || s.Runtime == nil {
		return lease, errors.New("store, source and runtime providers are required")
	}
	if options.Mode == "" {
		options.Mode = "review"
	}
	if options.Mode != "review" {
		return lease, fmt.Errorf("mode %q unsupported; use review", options.Mode)
	}
	if options.Owner == "" {
		return lease, errors.New("owner is required")
	}
	p := s.defaults()
	ttl, err := p.TTL(options.TTL)
	if err != nil {
		return lease, err
	}
	plan, err := BuildPlan(ctx, o, s.Source)
	if err != nil {
		return lease, err
	}
	for name, test := range plan.Manifest.Tests {
		for key, value := range test.Env {
			if len(evidence.Secrets(map[string]string{key: value})) > 0 && !envPlaceholder(value) {
				return lease, fmt.Errorf("test %s environment key %s must use ${env:NAME}; literal credentials are not persisted", name, key)
			}
		}
	}
	doctor, err := s.Runtime.Doctor(ctx)
	if err != nil {
		return lease, fmt.Errorf("%w: %v", ErrPrerequisite, err)
	}
	if doctor["context"] == "" {
		return lease, errors.New("Docker context identity is missing")
	}
	home, err := filepath.Abs(s.Home)
	if err != nil || s.Home == "" {
		return lease, errors.New("absolute state home is required")
	}
	manifest, err := config.CanonicalJSON(plan.Manifest)
	if err != nil {
		return lease, err
	}
	if evidence.RedactString(string(manifest), evidence.InheritedSecrets()) != string(manifest) {
		return lease, errors.New("manifest contains inherited credentials; replace literal values with ${env:NAME} references")
	}
	now := time.Now().UTC()
	id := newID()
	lease = domain.Lease{ID: id, Owner: options.Owner, Purpose: options.Purpose, Mode: options.Mode, Repository: plan.Repository, Stack: plan.Stack, Desired: "active", Observed: "requested", CreatedAt: now, HeartbeatAt: now, ExpiresAt: now.Add(ttl), ManifestDigest: plan.ManifestDigest, ManifestPath: plan.ManifestPath, ManifestCommit: plan.ManifestCommit, ManifestModified: plan.ManifestModified, SourceSetDigest: plan.SourceSetDigest, Manifest: manifest, Sources: plan.Sources, Components: plan.Components, Runtimes: plan.Runtimes, Resources: []domain.Resource{}, Diagnostics: append([]string{}, plan.Diagnostics...)}
	roots := map[string]string{}
	for i := range lease.Sources {
		source := &lease.Sources[i]
		source.WorktreePath = filepath.Join(home, "worktrees", id, source.Alias)
		roots[source.Alias] = source.WorktreePath
	}
	for i := range lease.Runtimes {
		r := &lease.Runtimes[i]
		r.LeaseID = id
		root := roots[r.Source]
		r.Project = projectName(id, r.Name)
		r.Context = doctor["context"]
		r.Directory = filepath.Join(root, filepath.FromSlash(r.Directory))
		for j, p := range r.Files {
			r.Files[j] = filepath.Join(root, filepath.FromSlash(p))
		}
	}
	if err = s.Store.Reserve(ctx, lease, p.MaxActive); err != nil {
		return lease, err
	}
	ctx, release, err := s.Store.AcquireContext(ctx, lease.ID, newID(), 2*time.Minute)
	if err != nil {
		return lease, err
	}
	defer func() { err = errors.Join(err, release()) }()
	if err = s.event(ctx, id, "allocation_requested", "Reserved immutable source identities, paths and runtime projects"); err != nil {
		return lease, err
	}
	lease.Observed = "allocating"
	if err = s.persist(ctx, lease); err != nil {
		return lease, err
	}
	doctorData, e := json.MarshalIndent(doctor, "", "  ")
	if e != nil {
		return lease, e
	}
	if err = s.saveTextArtifact(ctx, lease, "docker-info", "docker-info.json", string(doctorData)); err != nil {
		return lease, err
	}
	err = s.allocate(ctx, &lease, home)
	if err == nil {
		return lease, nil
	}
	// Compensation must survive cancellation of the original request.
	if operationLost(ctx) {
		return lease, errors.Join(err, domain.ErrLockLost)
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	defer cancel()
	lease.Diagnostics = append(lease.Diagnostics, evidence.RedactString(err.Error(), evidence.InheritedSecrets()))
	lease.Observed = "failed"
	evidenceErr := s.event(cleanupCtx, id, "allocation_failed", err.Error())
	saveErr := s.persist(cleanupCtx, lease)
	cleanupErr := s.cleanup(cleanupCtx, &lease, false)
	return lease, errors.Join(err, evidenceErr, saveErr, cleanupErr)
}

func operationLost(ctx context.Context) bool {
	if errors.Is(context.Cause(ctx), domain.ErrLockLost) {
		return true
	}
	if state, ok := ctx.(interface{ LockLost() bool }); ok {
		return state.LockLost()
	}
	return false
}

func (s *Service) allocate(ctx context.Context, l *domain.Lease, home string) error {
	roots := make([]string, 0, len(l.Sources))
	for _, source := range l.Sources {
		if err := s.event(ctx, l.ID, "source_materialize_requested", source.Alias+" "+source.Commit); err != nil {
			return err
		}
		if err := s.Source.Materialize(ctx, source); err != nil {
			return fmt.Errorf("materialize source %s: %w", source.Alias, err)
		}
		if err := s.event(ctx, l.ID, "source_materialized", source.Alias); err != nil {
			return err
		}
		roots = append(roots, source.WorktreePath)
	}
	for i := range l.Runtimes {
		r := &l.Runtimes[i]
		var sourceRoot string
		for _, source := range l.Sources {
			if source.Alias == r.Source {
				sourceRoot = source.WorktreePath
			}
		}
		for _, p := range append(append([]string{}, r.Files...), r.Directory) {
			if err := withinExisting(sourceRoot, p); err != nil {
				return fmt.Errorf("runtime %s path: %w", r.Name, err)
			}
		}
		if err := s.event(ctx, l.ID, "runtime_render_requested", r.Name); err != nil {
			return err
		}
		rendered, err := s.Runtime.Render(ctx, *r, roots)
		if err != nil {
			return err
		}
		r.Services = rendered.Services
		endpointData, err := endpointConfiguration(rendered.JSON, *l, *r)
		if err != nil {
			return err
		}
		data, err := ownedConfiguration(endpointData, *r, l.ID)
		if err != nil {
			return err
		}
		generated := filepath.Join(home, "leases", l.ID, "generated")
		if err := os.MkdirAll(generated, 0700); err != nil {
			return err
		}
		r.ConfigPath = filepath.Join(generated, r.Project+".json")
		if err := os.WriteFile(r.ConfigPath, data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		r.ConfigDigest = hex.EncodeToString(sum[:])
		// Persist intent before Up: a failed command can still have allocated resources.
		r.Started = true
		l.Observed = "starting"
		if err := s.persist(ctx, *l); err != nil {
			return err
		}
		if err := s.event(ctx, l.ID, "runtime_up_requested", r.Name); err != nil {
			return err
		}
		if err := s.Runtime.Up(ctx, *r); err != nil {
			return fmt.Errorf("start runtime %s: %w", r.Name, err)
		}
		if err := s.event(ctx, l.ID, "runtime_started", r.Name); err != nil {
			return err
		}
	}
	if err := s.waitReady(ctx, l); err != nil {
		return err
	}
	if err := s.probeReady(ctx, l); err != nil {
		return err
	}
	l.Observed = "ready"
	l.HeartbeatAt = time.Now().UTC()
	if err := s.persist(ctx, *l); err != nil {
		return err
	}
	return s.event(ctx, l.ID, "lease_ready", "All selected runtime services are ready")
}

func withinExisting(root, path string) error {
	r, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(r, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s escapes allocated source %s", path, root)
	}
	return nil
}

func ownedConfiguration(data []byte, r domain.Runtime, id string) ([]byte, error) {
	if evidence.RedactString(string(data), evidence.InheritedSecrets()) != string(data) {
		return nil, errors.New("rendered configuration contains inherited credentials; use secret files before persisting an execution snapshot")
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	services, ok := raw["services"].(map[string]any)
	if !ok {
		return nil, errors.New("rendered configuration has no services")
	}
	if len(r.Services) == 0 {
		return nil, errors.New("rendered service closure is empty")
	}
	for name, value := range services {
		if service, ok := value.(map[string]any); ok {
			if environment, ok := service["environment"].(map[string]any); ok {
				for key, value := range environment {
					fileReference := strings.HasSuffix(strings.ToUpper(key), "_FILE") && strings.HasPrefix(fmt.Sprint(value), "/") && !strings.ContainsAny(fmt.Sprint(value), "\x00\r\n")
					if value != nil && !fileReference && len(evidence.Secrets(map[string]string{key: fmt.Sprint(value)})) > 0 {
						return nil, fmt.Errorf("service %s environment key %s contains resolved credentials; use a secret file instead of persisting credential-bearing configuration", name, key)
					}
				}
			}
		}
	}
	for _, name := range r.Services {
		service, ok := services[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rendered closure service %s missing", name)
		}
		labels := map[string]any{}
		if existing, ok := service["labels"].(map[string]any); ok {
			for k, v := range existing {
				labels[k] = v
			}
		}
		labels["io.agent-env.lease"] = id
		labels["io.agent-env.runtime"] = r.Name
		service["labels"] = labels
	}
	for _, kind := range []string{"networks", "volumes"} {
		resources, _ := raw[kind].(map[string]any)
		for name, value := range resources {
			resource, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("rendered %s resource %s is invalid", kind, name)
			}
			labels := map[string]any{}
			if existing, ok := resource["labels"].(map[string]any); ok {
				for k, v := range existing {
					labels[k] = v
				}
			}
			labels["io.agent-env.lease"] = id
			labels["io.agent-env.runtime"] = r.Name
			resource["labels"] = labels
		}
	}
	raw["name"] = r.Project
	result, err := json.MarshalIndent(raw, "", "  ")
	return append(result, '\n'), err
}

func (s *Service) waitReady(ctx context.Context, l *domain.Lease) error {
	timeout := s.ReadinessTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	interval := s.ReadinessInterval
	if interval <= 0 {
		interval = time.Second
	}
	var manifest config.Manifest
	if err := json.Unmarshal(l.Manifest, &manifest); err != nil {
		return err
	}
	for _, component := range l.Components {
		for _, probe := range manifest.Components[component.Name].Readiness {
			if probe.Type != "compose" {
				continue
			}
			if probe.Timeout != "" {
				value, _ := time.ParseDuration(probe.Timeout)
				if value < timeout {
					timeout = value
				}
			}
			if probe.Interval != "" {
				value, _ := time.ParseDuration(probe.Interval)
				if value < interval {
					interval = value
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		allReady := true
		resources := []domain.Resource{}
		diagnostics := []string{}
		for _, r := range l.Runtimes {
			o, err := s.Runtime.Inspect(ctx, r)
			if err != nil {
				return fmt.Errorf("inspect runtime %s: %w", r.Name, err)
			}
			if err := ownedResources(l.ID, o.Resources); err != nil {
				return err
			}
			for i := range o.Resources {
				o.Resources[i].LeaseID = l.ID
				o.Resources[i].ID = l.ID + ":" + o.Resources[i].ID
			}
			resources = append(resources, o.Resources...)
			diagnostics = append(diagnostics, o.Diagnostics...)
			allReady = allReady && o.Ready && o.Exists
		}
		l.Resources = resources
		l.Diagnostics = diagnostics
		if err := s.persist(ctx, *l); err != nil {
			return err
		}
		if allReady {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("readiness deadline: %w: %s", ctx.Err(), strings.Join(diagnostics, "; "))
		case <-time.After(interval):
		}
	}
}

func ownedResources(id string, resources []domain.Resource) error {
	for _, r := range resources {
		if owner := r.Metadata["lease"]; owner != id {
			return fmt.Errorf("resource %s belongs to lease %s; quarantine without deleting", r.ExternalID, owner)
		}
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Lease, error) {
	return s.Store.Get(ctx, id)
}
func (s *Service) Show(ctx context.Context, id string) (domain.Lease, error) {
	return s.Reconcile(ctx, id)
}
func (s *Service) List(ctx context.Context, cached bool) ([]domain.Lease, error) {
	leases, err := s.Store.List(ctx)
	if err != nil || cached {
		return leases, err
	}
	var errs []error
	for i := range leases {
		l, err := s.Reconcile(ctx, leases[i].ID)
		if err != nil {
			errs = append(errs, err)
		}
		if l.ID != "" {
			leases[i] = l
		}
	}
	return leases, errors.Join(errs...)
}

func (s *Service) Renew(ctx context.Context, id string, ttl time.Duration) (lease domain.Lease, err error) {
	p := s.defaults()
	ttl, err = p.TTL(ttl)
	if err != nil {
		return lease, err
	}
	ctx, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		return lease, err
	}
	defer func() { err = errors.Join(err, release()) }()
	lease, err = s.Store.Get(ctx, id)
	if err != nil {
		return lease, err
	}
	if lease.Observed == "released" || lease.Observed == "releasing" || lease.Observed == "quarantined" {
		return lease, fmt.Errorf("cannot renew %s lease", lease.Observed)
	}
	now := time.Now().UTC()
	lease.HeartbeatAt = now
	lease.ExpiresAt = now.Add(ttl)
	if err = s.event(ctx, id, "lease_renew_requested", ttl.String()); err != nil {
		return lease, err
	}
	err = s.persist(ctx, lease)
	return lease, err
}

func (s *Service) Destroy(ctx context.Context, id string, force, dryRun bool) (lease domain.Lease, err error) {
	if dryRun {
		lease, err = s.Store.Get(ctx, id)
		if err != nil {
			return lease, err
		}
		return s.previewDestroy(ctx, lease, force)
	}
	ctx, release, err := s.acquireDestroyAfterCancellation(ctx, id)
	if err != nil {
		return lease, err
	}
	defer func() { err = errors.Join(err, release()) }()
	lease, err = s.Store.Get(ctx, id)
	if err != nil {
		return lease, err
	}
	lease.HeartbeatAt = time.Now().UTC()
	if err = s.event(ctx, id, "lease_destroy_requested", fmt.Sprintf("force=%t", force)); err != nil {
		return lease, err
	}
	err = s.cleanup(ctx, &lease, force)
	return lease, err
}

func (s *Service) quarantine(ctx context.Context, l *domain.Lease, cause error) error {
	l.Observed = "quarantined"
	l.Diagnostics = append(l.Diagnostics, evidence.RedactString(cause.Error(), evidence.InheritedSecrets()))
	return errors.Join(cause, s.persist(ctx, *l), s.event(ctx, l.ID, "lease_quarantined", cause.Error()))
}

func (s *Service) cleanup(ctx context.Context, l *domain.Lease, force bool) error {
	// Verify all source identities and tracked changes before deleting any resource.
	for _, source := range l.Sources {
		o, err := s.Source.Inspect(ctx, source)
		if err != nil {
			return s.quarantine(ctx, l, err)
		}
		if o.Exists && !o.Registered {
			return s.quarantine(ctx, l, fmt.Errorf("source %s exists without registered ownership", source.Alias))
		}
		if o.Exists && o.Commit != source.Commit {
			return s.quarantine(ctx, l, fmt.Errorf("source %s commit identity changed", source.Alias))
		}
		if o.TrackedDirty {
			if !force {
				return s.quarantine(ctx, l, fmt.Errorf("source %s has tracked changes; inspect diff or explicitly force with evidence", source.Alias))
			}
			diff, ok := s.Source.(SourceDiff)
			if !ok {
				return s.quarantine(ctx, l, errors.New("forced tracked cleanup requires source diff evidence provider"))
			}
			if err := s.event(ctx, l.ID, "forced_diff_requested", source.Alias); err != nil {
				return err
			}
			data, err := diff.Diff(ctx, source)
			if err != nil {
				return s.quarantine(ctx, l, err)
			}
			if err := s.saveTextArtifact(ctx, *l, "tracked-diff", source.Alias+".patch", data); err != nil {
				return s.quarantine(ctx, l, err)
			}
		}
	}
	l.Desired = "released"
	l.Observed = "releasing"
	if err := s.persist(ctx, *l); err != nil {
		return err
	}
	for i := len(l.Runtimes) - 1; i >= 0; i-- {
		r := &l.Runtimes[i]
		o, err := s.Runtime.Inspect(ctx, *r)
		if err != nil {
			return s.quarantine(ctx, l, err)
		}
		if err := ownedResources(l.ID, o.Resources); err != nil {
			return s.quarantine(ctx, l, err)
		}
		if o.Exists {
			if err := s.event(ctx, l.ID, "runtime_logs_requested", r.Name); err != nil {
				return err
			}
			for _, service := range r.Services {
				scoped := *r
				scoped.Services = []string{service}
				logs, err := s.Runtime.Logs(ctx, scoped)
				if err != nil {
					return s.quarantine(ctx, l, fmt.Errorf("collect logs for %s/%s before cleanup: %w", r.Name, service, err))
				}
				if err := s.saveTextArtifact(ctx, *l, "compose-log/"+r.Name+"/"+service, r.Project+"-"+service+".log", logs); err != nil {
					return s.quarantine(ctx, l, err)
				}
			}
			if err := s.event(ctx, l.ID, "runtime_down_requested", r.Name); err != nil {
				return err
			}
			if err := s.Runtime.Down(ctx, *r); err != nil {
				return s.quarantine(ctx, l, fmt.Errorf("remove runtime %s: %w", r.Name, err))
			}
			remaining, err := s.Runtime.Inspect(ctx, *r)
			if err != nil || remaining.Exists {
				if err == nil {
					err = errors.New("runtime resources remain after cleanup")
				}
				return s.quarantine(ctx, l, err)
			}
		}
		r.Started = false
		if err := s.persist(ctx, *l); err != nil {
			return err
		}
	}
	for i := len(l.Sources) - 1; i >= 0; i-- {
		source := l.Sources[i]
		if err := s.event(ctx, l.ID, "source_remove_requested", source.Alias); err != nil {
			return err
		}
		if err := s.Source.Remove(ctx, source, force); err != nil {
			return s.quarantine(ctx, l, fmt.Errorf("remove source %s: %w", source.Alias, err))
		}
		o, err := s.Source.Inspect(ctx, source)
		if err != nil || o.Exists || o.Registered {
			if err == nil {
				err = errors.New("source remains after cleanup")
			}
			return s.quarantine(ctx, l, err)
		}
	}
	l.Observed = "released"
	l.Resources = []domain.Resource{}
	if err := s.persist(ctx, *l); err != nil {
		return err
	}
	return s.event(ctx, l.ID, "lease_released", "Runtime resources and managed sources removed")
}

func (s *Service) saveTextArtifact(ctx context.Context, l domain.Lease, kind, name, data string) error {
	data = evidence.RedactString(data, evidence.InheritedSecrets())
	id := newID()
	directory := filepath.Join(s.Home, "leases", l.ID, "artifacts", id)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	path := filepath.Join(directory, filepath.Base(name))
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(data))
	return s.Store.SaveArtifact(ctx, domain.Artifact{ID: id, LeaseID: l.ID, Kind: kind, Path: path, Digest: hex.EncodeToString(sum[:]), CreatedAt: time.Now().UTC()})
}

func (s *Service) Reconcile(ctx context.Context, id string) (lease domain.Lease, err error) {
	ctx, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		return lease, err
	}
	defer func() { err = errors.Join(err, release()) }()
	lease, err = s.Store.Get(ctx, id)
	if err != nil {
		return lease, err
	}
	diagnostics := []string{}
	resources := []domain.Resource{}
	unknown, dirty, leftovers, ready := false, false, false, true
	for _, source := range lease.Sources {
		o, e := s.Source.Inspect(ctx, source)
		if e != nil {
			unknown = true
			diagnostics = append(diagnostics, "source "+source.Alias+": "+e.Error())
			continue
		}
		leftovers = leftovers || o.Exists || o.Registered
		if !o.Exists || !o.Registered {
			ready = false
			diagnostics = append(diagnostics, "source "+source.Alias+" is missing or unregistered")
		}
		if o.Exists && o.Commit != source.Commit {
			dirty = true
			diagnostics = append(diagnostics, "source "+source.Alias+" commit identity changed")
		}
		if o.TrackedDirty {
			dirty = true
			diagnostics = append(diagnostics, "source "+source.Alias+" has tracked changes")
		}
	}
	for _, r := range lease.Runtimes {
		o, e := s.Runtime.Inspect(ctx, r)
		if e != nil {
			unknown = true
			diagnostics = append(diagnostics, "runtime "+r.Name+": "+e.Error())
			continue
		}
		if e := ownedResources(lease.ID, o.Resources); e != nil {
			dirty = true
			diagnostics = append(diagnostics, e.Error())
		}
		leftovers = leftovers || o.Exists
		ready = ready && o.Exists && o.Ready
		for i := range o.Resources {
			o.Resources[i].LeaseID = lease.ID
			o.Resources[i].ID = lease.ID + ":" + o.Resources[i].ID
		}
		resources = append(resources, o.Resources...)
		diagnostics = append(diagnostics, o.Diagnostics...)
	}
	wasReleased := lease.Observed == "released"
	if ready && lease.Desired == "active" && !wasReleased {
		events, e := s.Store.Events(ctx, lease.ID)
		if e != nil {
			return lease, e
		}
		completed := false
		for _, event := range events {
			if event.Type == "lease_ready" {
				completed = true
				break
			}
		}
		if !completed {
			ready = false
			diagnostics = append(diagnostics, "allocation readiness was never completed; inspect the failed allocation before recovery")
		} else if e := s.probeHealth(ctx, lease); e != nil {
			ready = false
			diagnostics = append(diagnostics, e.Error())
		}
	}
	switch {
	case dirty || wasReleased && leftovers:
		lease.Observed = "quarantined"
	case unknown:
		if lease.Observed != "quarantined" {
			lease.Observed = "unknown"
		}
	case wasReleased:
		lease.Observed = "released"
		diagnostics = []string{}
	case lease.Observed == "quarantined":
	case ready && lease.Desired == "active":
		lease.Observed = "ready"
	default:
		lease.Observed = "degraded"
	}
	if wasReleased && leftovers {
		diagnostics = append(diagnostics, "released lease still owns external resources")
	}
	now := time.Now().UTC()
	if !wasReleased && !lease.ExpiresAt.After(now) {
		diagnostics = append(diagnostics, "lease expired at "+lease.ExpiresAt.Format(time.RFC3339)+"; renew explicitly or inspect the GC plan")
	}
	runs, err := s.Store.Runs(ctx, lease.ID)
	if err != nil {
		return lease, err
	}
	_, heartbeatGrace := s.gcGrace()
	for _, run := range runs {
		if run.Status != "running" {
			continue
		}
		message := "command run " + run.ID + " remains running in the registry; process completion is unverified and GC is blocked"
		if !lease.HeartbeatAt.Add(heartbeatGrace).After(now) {
			message = "stale " + message
		}
		diagnostics = append(diagnostics, message)
	}
	for i := range diagnostics {
		diagnostics[i] = evidence.RedactString(diagnostics[i], evidence.InheritedSecrets())
	}
	lease.Diagnostics = diagnostics
	lease.HeartbeatAt = time.Now().UTC()
	if !unknown {
		lease.Resources = resources
	}
	if err = s.persist(ctx, lease); err != nil {
		return lease, err
	}
	err = s.event(ctx, id, "lease_reconciled", lease.Observed+": "+strings.Join(diagnostics, "; "))
	return lease, err
}

func (s *Service) GC(ctx context.Context, apply bool) ([]domain.Lease, error) {
	leases, err := s.Store.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Lease{}
	var errs []error
	for _, l := range leases {
		eligible, checkErr := s.gcEligible(ctx, l, time.Now())
		if checkErr != nil {
			errs = append(errs, checkErr)
			continue
		}
		if !eligible {
			continue
		}
		if apply {
			l, err = s.gcOne(ctx, l.ID)
			if err != nil {
				errs = append(errs, err)
			}
		}
		result = append(result, l)
	}
	return result, errors.Join(errs...)
}

func (s *Service) gcOne(ctx context.Context, id string) (l domain.Lease, err error) {
	ctx, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		return l, err
	}
	defer func() { err = errors.Join(err, release()) }()
	l, err = s.Store.Get(ctx, id)
	if err != nil {
		return l, err
	}
	eligible, err := s.gcEligible(ctx, l, time.Now())
	if err != nil {
		return l, err
	}
	if !eligible {
		return l, nil
	}
	err = s.cleanup(ctx, &l, false)
	return l, err
}

func (s *Service) gcGrace() (time.Duration, time.Duration) {
	p := s.defaults()
	if p.GCGrace == 0 {
		p.GCGrace = 5 * time.Minute
	}
	if p.HeartbeatGrace == 0 {
		p.HeartbeatGrace = time.Minute
	}
	return p.GCGrace, p.HeartbeatGrace
}

func (s *Service) gcEligible(ctx context.Context, l domain.Lease, now time.Time) (bool, error) {
	expiryGrace, heartbeatGrace := s.gcGrace()
	if !domain.GCEligibleWithGrace(l, now, expiryGrace, heartbeatGrace) {
		return false, nil
	}
	runs, err := s.Store.Runs(ctx, l.ID)
	if err != nil {
		return false, err
	}
	for _, run := range runs {
		if run.Status == "running" {
			return false, nil
		}
	}
	return true, nil
}

func envPlaceholder(value string) bool {
	if !strings.HasPrefix(value, "${env:") || !strings.HasSuffix(value, "}") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(value, "${env:"), "}")
	if name == "" {
		return false
	}
	for i, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
