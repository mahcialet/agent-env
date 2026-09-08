// Package process owns lease-bound persistent native processes, without app orchestration.
package process

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
)

type Client struct{ Processes execx.ManagedProcess }
type Observation struct {
	Exists, Ready bool
	Endpoints     map[string]string
	Resources     []domain.Resource
	Diagnostics   []string
}
type ownership struct{ Lease, Runtime, Source, Commit string }
type receipt struct {
	Owner    ownership
	Identity execx.ProcessIdentity
}

func (c Client) native() execx.ManagedProcess {
	if c.Processes != nil {
		return c.Processes
	}
	return execx.NativeDetached{}
}
func owner(r domain.Runtime) ownership {
	return ownership{r.LeaseID, r.Name, r.Directory, r.Process.SourceCommit}
}
func uncertain(err error) error { return errors.Join(domain.ErrResourceIdentity, err) }

func (c Client) Prepare(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if r.Process == nil || r.LeaseID == "" || r.Name == "" || r.Process.SourceCommit == "" {
		return r, errors.New("incomplete process source identity")
	}
	p := *r.Process
	r.Process = &p
	root, err := filepath.EvalSymlinks(r.Directory)
	if err != nil {
		return r, err
	}
	r.Directory = root
	cwd, err := paths.Within(root, p.WorkingDirectory)
	if err != nil {
		return r, err
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return r, err
	}
	info, err := os.Stat(cwd)
	if err != nil {
		return r, err
	}
	if !info.IsDir() {
		return r, errors.New("working directory is not a directory")
	}
	p.ResolvedDirectory = cwd
	if len(p.Command) == 0 || strings.Contains(p.Command[0], "${") {
		return r, errors.New("literal executable is required")
	}
	exe := p.Command[0]
	origin := "host"
	if strings.ContainsAny(exe, "/\\") {
		exe, err = paths.Within(root, exe)
		origin = "source"
	} else {
		exe, err = execx.LookPath(exe)
	}
	if err != nil {
		return r, err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return r, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return r, err
	}
	ext := strings.ToLower(filepath.Ext(exe))
	if ext == ".cmd" || ext == ".bat" {
		return r, errors.New("persistent process requires a native executable")
	}
	digest, err := hashExecutable(exe)
	if err != nil {
		return r, err
	}
	p.Executable, p.ExecutableOrigin, p.ExecutableDigest = exe, origin, digest
	if !filepath.IsAbs(p.Directory) {
		return r, errors.New("runtime directory must be absolute")
	}
	if err = os.MkdirAll(filepath.Dir(p.Directory), 0700); err != nil {
		return r, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(p.Directory))
	if err != nil {
		return r, err
	}
	p.Directory = filepath.Join(parent, filepath.Base(p.Directory))
	if contained(root, p.Directory) || contained(p.Directory, root) {
		return r, errors.New("runtime directory must be separate from source")
	}
	p.StateDirectory = filepath.Join(p.Directory, "state")
	p.StdoutPath = filepath.Join(p.Directory, "stdout.log")
	p.StderrPath = filepath.Join(p.Directory, "stderr.log")
	err = os.Mkdir(p.Directory, 0700)
	if os.IsExist(err) {
		if err = validateOwner(r); err != nil {
			return r, err
		}
	} else if err != nil {
		return r, err
	} else {
		if err = writeJSON(filepath.Join(p.Directory, "owner.json"), owner(r)); err != nil {
			return r, err
		}
	}
	if err = os.Mkdir(p.StateDirectory, 0700); err != nil && !os.IsExist(err) {
		return r, err
	}
	for _, name := range []string{p.StdoutPath, p.StderrPath} {
		f, e := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e == nil {
			e = f.Close()
		}
		if e != nil && !os.IsExist(e) {
			return r, e
		}
	}
	if err = validatePaths(r, true); err != nil {
		return r, err
	}
	p.State = "prepared"
	return r, nil
}

func (c Client) Start(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	if r.Process == nil {
		return r, errors.New("missing process snapshot")
	}
	p := *r.Process
	r.Process = &p
	if p.State != "launching" {
		return r, errors.New("durable launching intent required")
	}
	p.State = "prepared"
	if err := validatePaths(r, true); err != nil {
		return r, err
	}
	if _, err := os.Lstat(filepath.Join(p.Directory, "launch.json")); !os.IsNotExist(err) {
		return r, uncertain(errors.New("launch receipt already exists or is inaccessible"))
	}
	digest, err := hashExecutable(p.Executable)
	if err != nil {
		return r, err
	}
	if digest != p.ExecutableDigest {
		return r, uncertain(errors.New("executable changed after preparation"))
	}
	cwd, err := paths.Within(r.Directory, p.WorkingDirectory)
	if err != nil {
		return r, err
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil || cwd != p.ResolvedDirectory {
		return r, uncertain(errors.New("working directory changed after preparation"))
	}
	for _, port := range p.Ports {
		if port < 1 || port > 65535 {
			return r, errors.New("invalid reserved process port")
		}
		listener, e := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if e != nil {
			return r, errors.New("reserved process port is occupied")
		}
		_ = listener.Close()
	}
	spec, secrets, err := command(r)
	if err != nil {
		return r, err
	}
	fingerprints, err := fingerprintSecrets(secrets)
	if err != nil {
		return r, err
	}
	if err = writeJSON(filepath.Join(p.Directory, "redaction.json"), redactionEvidence{Owner: owner(r), Version: 1, Secrets: fingerprints}); err != nil {
		return r, err
	}
	p.State = "launching"
	id, startErr := c.native().Start(ctx, spec, p.StdoutPath, p.StderrPath)
	p.ProcessID, p.ProcessStart = id.PID, id.StartID
	if id.PID != 0 {
		p.State = "running"
		err = writeJSON(filepath.Join(p.Directory, "launch.json"), receipt{Owner: owner(r), Identity: id})
		if err != nil {
			return r, uncertain(errors.Join(errors.New("cannot persist process launch receipt"), err, startErr))
		}
	}
	if startErr != nil {
		if id.PID == 0 && errors.Is(startErr, execx.ErrProcessNotStarted) {
			p.State = "prepared"
		}
		return r, startErr
	}
	if id.PID <= 0 || id.StartID == "" {
		return r, uncertain(errors.New("native launch returned incomplete identity"))
	}
	return r, nil
}

func (c Client) Inspect(ctx context.Context, r domain.Runtime) (Observation, error) {
	o := Observation{Endpoints: map[string]string{}}
	id, err := identity(r)
	if err != nil {
		return o, err
	}
	if id.PID == 0 {
		return o, nil
	}
	observed, err := c.native().Observe(ctx, id)
	if err != nil {
		return o, uncertain(err)
	}
	o.Exists = observed.Alive
	o.Ready = observed.Alive && observed.RootAlive
	if observed.Alive {
		o.Resources = []domain.Resource{{ID: r.Name + "-process", LeaseID: r.LeaseID, Runtime: r.Name, Kind: "process", ExternalID: strconv.Itoa(id.PID), Metadata: map[string]string{"start_id": id.StartID, "lease": r.LeaseID, "runtime": r.Name}}}
		for name, port := range r.Process.Ports {
			o.Endpoints[name] = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	if observed.Alive && !observed.RootAlive {
		o.Diagnostics = append(o.Diagnostics, "foreground process exited while descendants remain")
	}
	return o, nil
}

func (c Client) Destroy(ctx context.Context, r domain.Runtime) error {
	id, err := identity(r)
	if err != nil {
		return err
	}
	if id.PID != 0 {
		if err = c.native().Terminate(ctx, id, 2*time.Second); err != nil {
			return uncertain(err)
		}
		o, e := c.native().Observe(ctx, id)
		if e != nil {
			return uncertain(e)
		}
		if o.Alive {
			return uncertain(errors.New("process tree remains alive"))
		}
	}
	if r.Process == nil {
		return errors.New("missing process snapshot")
	}
	if _, e := os.Lstat(r.Process.Directory); os.IsNotExist(e) && id.PID == 0 && (r.Process.State == "planned" || r.Process.State == "reserved" || r.Process.State == "released") {
		return nil
	}
	if err = validatePaths(r, false); err != nil {
		return err
	}
	return os.RemoveAll(r.Process.StateDirectory)
}

func (c Client) Logs(ctx context.Context, r domain.Runtime) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := validatePaths(r, false); err != nil {
		return "", err
	}
	var rec redactionEvidence
	err := readJSON(filepath.Join(r.Process.Directory, "redaction.json"), &rec)
	var fingerprints []secretFingerprint
	if err == nil {
		if rec.Owner != owner(r) || rec.Version != 1 {
			return "", uncertain(errors.New("log redaction receipt ownership/version mismatch"))
		}
		fingerprints = rec.Secrets
	} else if os.IsNotExist(err) && r.Process.State == "prepared" && r.Process.ProcessID == 0 {
		_, secrets, e := command(r)
		if e != nil {
			return "", e
		}
		fingerprints, e = fingerprintSecrets(secrets)
		if e != nil {
			return "", e
		}
	} else {
		return "", uncertain(errors.New("durable log redaction evidence is unavailable"))
	}
	redactor, err := newFingerprintRedactor(fingerprints)
	if err != nil {
		return "", uncertain(err)
	}
	var out strings.Builder
	for _, path := range []string{r.Process.StdoutPath, r.Process.StderrPath} {
		f, e := os.Open(path)
		if e != nil {
			return "", e
		}
		data, e := io.ReadAll(io.LimitReader(f, 1024*1024+1+int64(redactor.longest)))
		_ = f.Close()
		if e != nil {
			return "", e
		}
		redacted, e := redactor.redact(ctx, data, 1024*1024)
		if e != nil {
			return "", e
		}
		out.WriteString(filepath.Base(path) + ":\n")
		out.WriteString(redacted)
	}
	return out.String(), nil
}

func identity(r domain.Runtime) (execx.ProcessIdentity, error) {
	if r.Process != nil && r.Process.ProcessID == 0 && (r.Process.State == "planned" || r.Process.State == "reserved" || r.Process.State == "released") {
		if _, err := os.Lstat(r.Process.Directory); os.IsNotExist(err) {
			return execx.ProcessIdentity{}, nil
		}
	}
	if err := validatePaths(r, false); err != nil {
		return execx.ProcessIdentity{}, err
	}
	p := r.Process
	id := execx.ProcessIdentity{PID: p.ProcessID, StartID: p.ProcessStart}
	var rec receipt
	err := readJSON(filepath.Join(p.Directory, "launch.json"), &rec)
	if os.IsNotExist(err) && (p.State == "prepared" || p.State == "planned" || p.State == "reserved") && id.PID == 0 {
		return id, nil
	}
	if err != nil { // A returned native identity still permits compensation when writing its receipt failed.
		if os.IsNotExist(err) && id.PID > 0 && id.StartID != "" {
			return id, nil
		}
		return id, uncertain(err)
	}
	if rec.Owner != owner(r) || rec.Identity.PID <= 0 || rec.Identity.StartID == "" {
		return id, uncertain(errors.New("launch receipt ownership mismatch"))
	}
	if id.PID != 0 && id != rec.Identity {
		return id, uncertain(errors.New("launch receipt process identity mismatch"))
	}
	return rec.Identity, nil
}

func command(r domain.Runtime) (execx.Command, []string, error) {
	p := r.Process
	spec := execx.Command{Name: p.Executable, Dir: p.ResolvedDirectory, Env: map[string]string{}}
	secrets := evidence.InheritedSecrets()
	expand := func(value string) (string, error) {
		var result strings.Builder
		for {
			before, after, ok := strings.Cut(value, "${")
			result.WriteString(before)
			if !ok {
				break
			}
			key, rest, closed := strings.Cut(after, "}")
			if !closed {
				return "", errors.New("unterminated process interpolation")
			}
			switch {
			case key == "runtime_dir":
				result.WriteString(p.StateDirectory)
			case key == "lease_id":
				result.WriteString(r.LeaseID)
			case strings.HasPrefix(key, "port:"):
				port, ok := p.Ports[strings.TrimPrefix(key, "port:")]
				if !ok || port <= 0 {
					return "", errors.New("unknown process port interpolation")
				}
				result.WriteString(strconv.Itoa(port))
			case strings.HasPrefix(key, "env:"):
				v, ok := os.LookupEnv(strings.TrimPrefix(key, "env:"))
				if !ok {
					return "", errors.New("required process environment reference is unavailable")
				}
				result.WriteString(v)
				secrets = append(secrets, v)
			default:
				return "", errors.New("unsupported process interpolation")
			}
			value = rest
		}
		return result.String(), nil
	}
	for _, arg := range p.Command[1:] {
		value, err := expand(arg)
		if err != nil {
			return spec, nil, err
		}
		spec.Args = append(spec.Args, value)
	}
	for key, value := range p.Env {
		v, err := expand(value)
		if err != nil {
			return spec, nil, err
		}
		spec.Env[key] = v
	}
	secrets = append(secrets, evidence.Secrets(spec.Env)...)
	return spec, secrets, nil
}
func hashExecutable(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("executable must be a regular file")
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func contained(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func validateOwner(r domain.Runtime) error {
	if r.Process == nil {
		return errors.New("missing process snapshot")
	}
	resolved, err := filepath.EvalSymlinks(r.Process.Directory)
	if err != nil {
		return uncertain(err)
	}
	if resolved != r.Process.Directory {
		return uncertain(errors.New("runtime directory alias changed"))
	}
	var have ownership
	if err = readJSON(filepath.Join(resolved, "owner.json"), &have); err != nil {
		return uncertain(err)
	}
	if have != owner(r) {
		return uncertain(errors.New("runtime ownership marker mismatch"))
	}
	return nil
}
func validatePaths(r domain.Runtime, stateRequired bool) error {
	if err := validateOwner(r); err != nil {
		return err
	}
	p := r.Process
	for _, entry := range []struct {
		path, name string
		directory  bool
	}{{p.StateDirectory, "state", true}, {p.StdoutPath, "stdout.log", false}, {p.StderrPath, "stderr.log", false}} {
		if entry.path != filepath.Join(p.Directory, entry.name) {
			return uncertain(errors.New("runtime evidence path changed"))
		}
		info, err := os.Lstat(entry.path)
		if os.IsNotExist(err) && entry.directory && !stateRequired {
			continue
		}
		if err != nil {
			return uncertain(err)
		}
		if info.Mode()&os.ModeSymlink != 0 || entry.directory && !info.IsDir() || !entry.directory && !info.Mode().IsRegular() {
			return uncertain(errors.New("runtime path is not a private native file/directory"))
		}
	}
	return nil
}
func readJSON(path string, dst any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("identity evidence is not a regular file")
	}
	if info.Size() > 1024*1024 {
		return errors.New("oversized identity evidence")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
func writeJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".identity-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if _, err = os.Lstat(path); !os.IsNotExist(err) {
		return fmt.Errorf("identity evidence already exists or is inaccessible: %s", filepath.Base(path))
	}
	return os.Rename(temp, path)
}
