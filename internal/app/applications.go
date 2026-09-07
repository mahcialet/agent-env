package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/stack"
)

type FlutterProvider interface {
	Validate(string, string, string) (string, error)
	Doctor(context.Context, string) (string, error)
	Build(context.Context, string, string, []string, string, time.Duration) (domain.ApplicationBuildResult, error)
}

// AndroidApplicationProvider contains only generic Android operations. Resource
// identity and the shared ADB-server policy remain the Android adapter's concern.
type AndroidApplicationProvider interface {
	InstallAPK(context.Context, domain.Runtime, string) error
	PackageInstalled(context.Context, domain.Runtime, string) (bool, error)
	Reverse(context.Context, domain.Runtime, int, int) error
	ReverseMappings(context.Context, domain.Runtime) (map[int]int, error)
	RemoveReverse(context.Context, domain.Runtime, int, int) error
	LaunchActivity(context.Context, domain.Runtime, string, string) error
}

func applicationSelected(apps []domain.Application, name string) bool {
	for _, a := range apps {
		if a.Name == name {
			return true
		}
	}
	return false
}
func applicationRuntime(l domain.Lease, name string) (domain.Runtime, error) {
	for _, r := range l.Runtimes {
		if r.Name == name && r.Type == "android-emulator" && r.Android != nil {
			return r, nil
		}
	}
	return domain.Runtime{}, fmt.Errorf("selected Android runtime %q missing", name)
}
func applicationSource(l domain.Lease, a domain.Application) (domain.Source, error) {
	for _, src := range l.Sources {
		if src.Alias == a.Source && src.Commit == a.SourceCommit {
			return src, nil
		}
	}
	return domain.Source{}, errors.New("application pinned source identity mismatch")
}
func (s *Service) buildApplications(ctx context.Context, l *domain.Lease) error {
	for i := range l.Applications {
		a := &l.Applications[i]
		src, err := applicationSource(*l, *a)
		if err != nil {
			return err
		}
		if err = s.inspectTestSources(ctx, l, evidence.InheritedSecrets()); err != nil {
			return err
		}
		if _, err = s.Flutter.Validate(src.WorktreePath, a.ProjectDirectory, a.Artifact); err != nil {
			return fmt.Errorf("%w: application %s project: %v", ErrPrerequisite, a.Name, err)
		}
		a.State = "building"
		a.BuildUnconfirmed = true
		if err = s.persist(ctx, *l); err != nil {
			return err
		}
		if err = s.event(ctx, l.ID, "application_build_requested", a.Name); err != nil {
			return err
		}
		timeout := 20 * time.Minute
		if a.Timeout != "" {
			timeout, err = time.ParseDuration(a.Timeout)
			if err != nil {
				return err
			}
		}
		result, buildErr := s.Flutter.Build(ctx, src.WorktreePath, a.ProjectDirectory, a.Command, a.Artifact, timeout)
		a.BuildUnconfirmed = errors.Is(buildErr, execx.ErrProcessTreeUnconfirmed) || errors.Is(buildErr, execx.ErrOutputIncomplete)
		a.Build = result
		// Logs live in redacted retained artifacts, never in the lease's JSON payload.
		a.Build.Stdout, a.Build.Stderr = "", ""
		a.State = "built"
		if buildErr != nil {
			a.State = "build_failed"
		}
		if operationLost(ctx) {
			return errors.Join(buildErr, domain.ErrLockLost)
		}
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		e := errors.Join(s.saveTextArtifact(finish, *l, "application-build/"+a.Name, a.Name+"-build.stdout.log", result.Stdout), s.saveTextArtifact(finish, *l, "application-build/"+a.Name, a.Name+"-build.stderr.log", result.Stderr), s.persist(finish, *l))
		if buildErr == nil {
			e = errors.Join(e, s.inspectTestSources(finish, l, evidence.InheritedSecrets()))
		}
		cancel()
		if err = errors.Join(buildErr, e); err != nil {
			return fmt.Errorf("build application %s: %w", a.Name, err)
		}
	}
	return nil
}

func loopbackPort(address string) (int, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}
	ip := net.ParseIP(host)
	// adb reverse tcp:N connects on this same host. Do not silently discard a
	// remote endpoint address (including a remote Docker context).
	if ip == nil || !ip.IsLoopback() {
		return 0, fmt.Errorf("reverse endpoint must be a loopback TCP address: %s", address)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("invalid endpoint TCP port: %s", address)
	}
	return n, nil
}

func (s *Service) startApplications(ctx context.Context, l *domain.Lease) error {
	if len(l.Applications) == 0 {
		return nil
	}
	endpoints, err := s.Endpoints(ctx, *l)
	if err != nil {
		return err
	}
	for i := range l.Applications {
		a := &l.Applications[i]
		r, e := applicationRuntime(*l, a.Runtime)
		if e != nil {
			return e
		}
		src, e := applicationSource(*l, *a)
		if e != nil {
			return e
		}
		dir, e := paths.Within(src.WorktreePath, a.ProjectDirectory)
		if e != nil {
			return e
		}
		apk, e := paths.Within(dir, a.Artifact)
		if e != nil {
			return e
		}
		if e = regularApplicationAPK(dir, apk); e != nil {
			return e
		}
		digest, e := evidence.FileDigest(apk)
		if e != nil {
			return e
		}
		if apk != a.Build.ArtifactPath || digest != a.Build.Digest || digest == "" {
			return errors.New("APK identity changed after build")
		}
		existing, e := s.AndroidApplications.PackageInstalled(ctx, r, a.Package)
		if e != nil {
			return e
		}
		if existing {
			return fmt.Errorf("declared package %s already exists; cannot attribute this APK to a fresh install", a.Package)
		}
		a.State = "installing"
		if e = s.persist(ctx, *l); e != nil {
			return e
		}
		if e = s.event(ctx, l.ID, "application_install_requested", a.Name); e != nil {
			return e
		}
		if e = s.AndroidApplications.InstallAPK(ctx, r, apk); e != nil {
			return e
		}
		present, e := s.AndroidApplications.PackageInstalled(ctx, r, a.Package)
		if e != nil {
			return e
		}
		if !present {
			return fmt.Errorf("installed APK does not contain declared package %s", a.Package)
		}
		a.InstalledDigest = digest
		if e = s.persist(ctx, *l); e != nil {
			return e
		}
		for j := range a.Reverse {
			b := &a.Reverse[j]
			b.HostPort, e = loopbackPort(endpoints[b.Endpoint])
			if e != nil {
				return fmt.Errorf("endpoint %s: %w", b.Endpoint, e)
			}
			current, e := s.AndroidApplications.ReverseMappings(ctx, r)
			if e != nil {
				return e
			}
			if _, exists := current[b.DevicePort]; exists {
				return fmt.Errorf("device port %d already mapped; refusing ownership claim", b.DevicePort)
			}
			b.Requested = true
			if e = s.persist(ctx, *l); e != nil {
				return e
			}
			if e = s.event(ctx, l.ID, "application_reverse_requested", fmt.Sprintf("%s tcp:%d -> tcp:%d", a.Name, b.DevicePort, b.HostPort)); e != nil {
				return e
			}
			if e = s.AndroidApplications.Reverse(ctx, r, b.DevicePort, b.HostPort); e != nil {
				return e
			}
			current, e = s.AndroidApplications.ReverseMappings(ctx, r)
			if e != nil {
				return e
			}
			if current[b.DevicePort] != b.HostPort {
				return errors.New("reverse mapping was not established")
			}
			b.Established = true
			if e = s.persist(ctx, *l); e != nil {
				return e
			}
		}
		a.State = "launching"
		if e = s.persist(ctx, *l); e != nil {
			return e
		}
		if e = s.event(ctx, l.ID, "application_launch_requested", a.Name); e != nil {
			return e
		}
		if e = s.AndroidApplications.LaunchActivity(ctx, r, a.Package, a.Activity); e != nil {
			return e
		}
		a.State = "ready"
		if e = s.persist(ctx, *l); e != nil {
			return e
		}
		if e = s.event(ctx, l.ID, "application_ready", a.Name); e != nil {
			return e
		}
	}
	return nil
}

func applicationConsistent(l domain.Lease, a domain.Application) error {
	var m config.Manifest
	if err := json.Unmarshal(l.Manifest, &m); err != nil {
		return err
	}
	if config.Digest(&m) != l.ManifestDigest {
		return errors.New("manifest digest mismatch")
	}
	spec, ok := m.Applications[a.Name]
	if !ok || spec.Type != a.Type || spec.Source != a.Source || spec.Runtime != a.Runtime || spec.Package != a.Package || spec.Activity != a.Activity || spec.ProjectDirectory != a.ProjectDirectory || spec.Build.Artifact != a.Artifact || spec.Build.Timeout != a.Timeout || !reflect.DeepEqual(spec.Build.Command, a.Command) {
		return errors.New("application declaration differs from pinned manifest")
	}
	src, err := applicationSource(l, a)
	if err != nil {
		return err
	}
	expected := filepath.Join(src.WorktreePath, filepath.FromSlash(a.ProjectDirectory), filepath.FromSlash(a.Artifact))
	if a.Build.ArtifactPath != expected || !validApplicationDigest(a.Build.Digest) || a.InstalledDigest != a.Build.Digest || a.Build.Version == "" {
		return errors.New("application build/install identity is incomplete or inconsistent")
	}
	if len(a.Reverse) != len(spec.Reverse) {
		return errors.New("reverse declaration count changed")
	}
	for i, b := range a.Reverse {
		if b.DevicePort != spec.Reverse[i].DevicePort || b.Endpoint != spec.Reverse[i].Endpoint || !b.Requested || !b.Established || b.HostPort < 1 {
			return errors.New("reverse identity is incomplete or inconsistent")
		}
	}
	return nil
}
func (s *Service) observeApplication(ctx context.Context, l domain.Lease, a *domain.Application) error {
	if err := applicationConsistent(l, *a); err != nil {
		a.State = "degraded"
		return err
	}
	if s.AndroidApplications == nil {
		return errors.New("Android application provider unavailable")
	}
	r, err := applicationRuntime(l, a.Runtime)
	if err != nil {
		return err
	}
	installed, err := s.AndroidApplications.PackageInstalled(ctx, r, a.Package)
	if err != nil {
		return err
	}
	if !installed {
		a.State = "degraded"
		return errors.New("declared package is missing")
	}
	mappings, err := s.AndroidApplications.ReverseMappings(ctx, r)
	if err != nil {
		return err
	}
	endpoints, err := s.Endpoints(ctx, l)
	if err != nil {
		return err
	}
	for _, b := range a.Reverse {
		currentPort, e := loopbackPort(endpoints[b.Endpoint])
		if e != nil || currentPort != b.HostPort {
			a.State = "degraded"
			return fmt.Errorf("endpoint %s differs from recorded reverse target", b.Endpoint)
		}
		if mappings[b.DevicePort] != b.HostPort {
			a.State = "degraded"
			return fmt.Errorf("reverse tcp:%d no longer points to recorded host port %d", b.DevicePort, b.HostPort)
		}
	}
	// Activity foreground/running state is intentionally not a health condition.
	a.State = "ready"
	return nil
}

func (s *Service) cleanupApplications(ctx context.Context, l *domain.Lease, r domain.Runtime) error {
	for i := len(l.Applications) - 1; i >= 0; i-- {
		a := &l.Applications[i]
		if a.Runtime != r.Name {
			continue
		}
		for j := len(a.Reverse) - 1; j >= 0; j-- {
			b := &a.Reverse[j]
			if !b.Requested {
				continue
			}
			o, err := s.inspectRuntime(ctx, r)
			if err != nil {
				return err
			}
			if o.Exists {
				if s.AndroidApplications == nil {
					return errors.New("Android application provider unavailable")
				}
				if !b.Established {
					mappings, err := s.AndroidApplications.ReverseMappings(ctx, r)
					if err != nil {
						return err
					}
					if _, present := mappings[b.DevicePort]; present {
						return fmt.Errorf("%w: reverse tcp:%d was requested but ownership was never confirmed", domain.ErrResourceIdentity, b.DevicePort)
					}
					b.Requested = false
					if err = s.persist(ctx, *l); err != nil {
						return applicationCleanupWriteError{err}
					}
					continue
				}
				if err = s.event(ctx, l.ID, "application_reverse_remove_requested", a.Name); err != nil {
					return applicationCleanupWriteError{err}
				}
				if err = s.AndroidApplications.RemoveReverse(ctx, r, b.DevicePort, b.HostPort); err != nil {
					return err
				}
			}
			b.Requested, b.Established = false, false
			if err = s.persist(ctx, *l); err != nil {
				return applicationCleanupWriteError{err}
			}
		}
	}
	return nil
}
func releaseApplications(l *domain.Lease, runtime string) {
	for i := range l.Applications {
		if l.Applications[i].Runtime == runtime {
			l.Applications[i].State = "released"
		}
	}
}

func regularApplicationAPK(dir, path string) error {
	for p := path; p != filepath.Clean(dir); p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("APK path changed to a symlink after build")
		}
		if p == path && !st.Mode().IsRegular() {
			return errors.New("APK is not a regular file")
		}
	}
	return nil
}
func validApplicationDigest(digest string) bool {
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == 32
}

func applicationSetConsistent(l domain.Lease) error {
	var m config.Manifest
	err := json.Unmarshal(l.Manifest, &m)
	if len(l.Applications) == 0 && len(m.Applications) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	if config.Digest(&m) != l.ManifestDigest {
		return errors.New("application manifest digest mismatch")
	}
	components, err := stack.Resolve(&m, l.Stack)
	if err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, c := range components {
		if name := m.Components[c].Application; name != "" {
			expected[name] = true
		}
	}
	for _, a := range l.Applications {
		if !expected[a.Name] {
			return errors.New("unexpected or duplicate application record")
		}
		delete(expected, a.Name)
	}
	if len(expected) > 0 {
		return errors.New("selected application record missing")
	}
	return nil
}

type applicationCleanupWriteError struct{ error }

func (e applicationCleanupWriteError) Unwrap() error { return e.error }
