package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/policy"
	androidruntime "github.com/mahcialet/agent-env/internal/runtime/android"
	"github.com/mahcialet/agent-env/internal/runtime/compose"
	flutterruntime "github.com/mahcialet/agent-env/internal/runtime/flutter"
	"github.com/mahcialet/agent-env/internal/source/gitcli"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }
func ExitCode(err error) int {
	if errors.Is(err, app.ErrPrerequisite) {
		return 3
	}
	var e *ExitError
	if errors.As(err, &e) {
		return e.Code
	}
	return 2
}
func doctorManifest(ctx context.Context, path, runtimeType string, android app.AndroidProvider) (string, error) {
	m, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if runtimeType == "android-emulator" {
		names := make([]string, 0, len(m.Runtimes))
		for name := range m.Runtimes {
			names = append(names, name)
		}
		sort.Strings(names)
		var issues []error
		for _, name := range names {
			runtime := m.Runtimes[name]
			if runtime.Type != "android-emulator" {
				continue
			}
			_, err := android.Validate(ctx, runtime.AVD)
			if err != nil {
				issues = append(issues, fmt.Errorf("Android runtime %s (AVD %s): %w", name, runtime.AVD, err))
			}
		}
		if err := errors.Join(issues...); err != nil {
			return "", err
		}
	}
	return config.Digest(m), nil
}
func coded(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{code, err}
}

type runtimeAdapter struct{ client compose.Client }

func (r runtimeAdapter) Inventory(ctx context.Context, name string) ([]domain.Resource, error) {
	return r.client.Inventory(ctx, name)
}

func (r runtimeAdapter) Doctor(ctx context.Context) (map[string]string, error) {
	return r.client.Doctor(ctx)
}
func (r runtimeAdapter) Render(ctx context.Context, v domain.Runtime, roots []string) (app.Rendered, error) {
	x, err := r.client.Render(ctx, v, roots)
	return app.Rendered{JSON: x.JSON, Digest: x.Digest, Services: x.Services}, err
}
func (r runtimeAdapter) Up(ctx context.Context, v domain.Runtime) error { return r.client.Up(ctx, v) }
func (r runtimeAdapter) Down(ctx context.Context, v domain.Runtime) error {
	return r.client.Down(ctx, v)
}
func (r runtimeAdapter) Logs(ctx context.Context, v domain.Runtime) (string, error) {
	return r.client.Logs(ctx, v)
}
func (r runtimeAdapter) Inspect(ctx context.Context, v domain.Runtime) (app.RuntimeObservation, error) {
	x, err := r.client.Inspect(ctx, v)
	return app.RuntimeObservation{Ready: x.Ready, Exists: x.Exists, Resources: x.Resources, Diagnostics: x.Diagnostics, Endpoints: x.Endpoints}, err
}

func (g gitSource) Diff(ctx context.Context, s domain.Source) (string, error) {
	if _, err := g.client.Inspect(ctx, worktree(s)); err != nil {
		return "", err
	}
	result, err := g.client.Runner.Run(ctx, execx.Command{Name: "git", Args: []string{"diff", "HEAD", "--binary", "--no-ext-diff", "--no-textconv", "--"}, Dir: s.WorktreePath, UnsetEnv: []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR"}, Timeout: 30 * time.Second})
	return result.Stdout, err
}

func localOwner() string {
	if owner := os.Getenv("AGENT_ENV_OWNER"); owner != "" {
		return owner
	}
	host, _ := os.Hostname()
	name := "local"
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	return name + "@" + host + "/" + strings.ToLower(ulid.Make().String())
}

func ownerMatches(recorded, current string, explicit bool) bool {
	if explicit {
		return recorded == current
	}
	prefix := func(value string) string {
		at := strings.LastIndex(value, "/")
		if at < 0 {
			return value
		}
		if _, err := ulid.Parse(value[at+1:]); err != nil {
			return value
		}
		return value[:at]
	}
	return prefix(recorded) == prefix(current)
}

func openService(out, errOut io.Writer) (*app.Service, func() error, error) {
	home, err := paths.Resolve()
	if err != nil {
		return nil, nil, err
	}
	if err = os.MkdirAll(home, 0700); err != nil {
		return nil, nil, err
	}
	store, err := sqlite.Open(filepath.Join(home, "state.db"))
	if err != nil {
		return nil, nil, err
	}
	return serviceForStore(home, store, out, errOut), store.Close, nil
}

func serviceForStore(home string, store app.Store, out, errOut io.Writer) *app.Service {
	runner := execx.OSRunner{}
	p := policy.Defaults()
	android := androidruntime.Adapter{Runner: runner, Processes: execx.NativeDetached{}}
	return &app.Service{Home: home, Store: store, Source: gitSource{gitcli.Client{Runner: runner}}, Runtime: runtimeAdapter{compose.Client{Runner: runner, Policy: p}}, Android: android, Flutter: flutterruntime.Adapter{Runner: runner}, AndroidApplications: android, Policy: p, Runner: runner, Stdout: out, Stderr: errOut}
}

func addLifecycle(root *cobra.Command, output *string, emit func(any) error, out, errOut io.Writer) {
	owner := localOwner()
	root.PersistentFlags().StringVar(&owner, "owner", owner, "advisory lease owner (or AGENT_ENV_OWNER)")
	withFactory := func(factory func(io.Writer, io.Writer) (*app.Service, func() error, error), fn func(*app.Service) error) error {
		stream := out
		if *output == "json" {
			stream = errOut
		}
		s, close, err := factory(stream, errOut)
		if err != nil {
			return coded(7, err)
		}
		defer close()
		return fn(s)
	}
	with := func(fn func(*app.Service) error) error { return withFactory(openService, fn) }
	createOptions := app.CreateOptions{Mode: "review"}
	planOptions := app.PlanOptions{}
	create := &cobra.Command{Use: "create [repository]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		planOptions.Repository = "."
		if len(args) > 0 {
			planOptions.Repository = args[0]
		}
		createOptions.Owner = owner
		return withFactory(openCreateService, func(s *app.Service) error {
			l, err := s.Create(cmd.Context(), planOptions, createOptions)
			if l.ID != "" {
				if e := emit(l); e != nil {
					return e
				}
			}
			if err != nil {
				if errors.Is(err, app.ErrPrerequisite) {
					return coded(3, err)
				}
				if l.ID == "" {
					return coded(2, err)
				}
				if l.Observed == "quarantined" {
					return coded(6, err)
				}
			}
			return coded(4, err)
		})
	}}
	create.Flags().StringVar(&planOptions.Stack, "stack", "", "named component stack")
	create.Flags().StringVar(&planOptions.Ref, "ref", "", "single-source ref")
	create.Flags().StringVar(&planOptions.ManifestPath, "manifest", "", "explicit manifest path relative to repository")
	create.Flags().StringToStringVar(&planOptions.SourceRefs, "source", nil, "source alias=ref overrides")
	create.Flags().StringVar(&createOptions.Purpose, "purpose", "", "human purpose")
	create.Flags().StringVar(&createOptions.Mode, "mode", "review", "lease mode (review)")
	create.Flags().DurationVar(&createOptions.TTL, "ttl", 0, "lease lifetime (default host policy)")
	root.AddCommand(create)
	cached, mine := false, false
	state := ""
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return with(func(s *app.Service) error {
			leases, err := s.List(cmd.Context(), cached)
			if err != nil {
				return coded(7, err)
			}
			filtered := []domain.Lease{}
			for _, l := range leases {
				if mine && !ownerMatches(l.Owner, owner, root.PersistentFlags().Changed("owner") || os.Getenv("AGENT_ENV_OWNER") != "") {
					continue
				}
				if state != "" && state != l.Observed && state != l.Desired {
					continue
				}
				filtered = append(filtered, l)
			}
			return emit(filtered)
		})
	}}
	list.Flags().BoolVar(&cached, "cached", false, "registry-only view without resource inspection")
	list.Flags().BoolVar(&mine, "mine", false, "filter by configured owner")
	list.Flags().StringVar(&state, "state", "", "filter desired or observed state")
	root.AddCommand(list)
	root.AddCommand(&cobra.Command{Use: "show <lease-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			l, err := s.Show(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			events, err := s.Store.Events(cmd.Context(), l.ID)
			if err != nil {
				return err
			}
			runs, err := s.Store.Runs(cmd.Context(), l.ID)
			if err != nil {
				return err
			}
			artifacts, err := s.Store.Artifacts(cmd.Context(), l.ID)
			if err != nil {
				return err
			}
			return emit(map[string]any{"lease": l, "events": events, "runs": runs, "artifacts": artifacts})
		})
	}})
	ttl := time.Duration(0)
	renew := &cobra.Command{Use: "renew <lease-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			l, err := s.Renew(cmd.Context(), args[0], ttl)
			if err != nil {
				return err
			}
			return emit(l)
		})
	}}
	renew.Flags().DurationVar(&ttl, "ttl", 0, "lease lifetime from now")
	root.AddCommand(renew)
	force, dryRun := false, false
	destroy := &cobra.Command{Use: "destroy <lease-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			l, err := s.Destroy(cmd.Context(), args[0], force, dryRun)
			if l.ID != "" {
				if e := emit(l); e != nil {
					return e
				}
			}
			return coded(6, err)
		})
	}}
	destroy.Flags().BoolVar(&force, "force", false, "explicitly discard tracked edits after preserving evidence")
	destroy.Flags().BoolVar(&dryRun, "dry-run", false, "inspect cleanup plan without deleting resources")
	root.AddCommand(destroy)
	apply := false
	gc := &cobra.Command{Use: "gc", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return with(func(s *app.Service) error {
			leases, err := s.GC(cmd.Context(), apply)
			if e := emit(map[string]any{"apply": apply, "leases": leases}); e != nil {
				return e
			}
			return coded(6, err)
		})
	}}
	gc.Flags().BoolVar(&apply, "apply", false, "delete eligible expired leases (default: plan only)")
	root.AddCommand(gc)
	root.AddCommand(&cobra.Command{Use: "reconcile [lease-id]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			if len(args) == 1 {
				l, err := s.Reconcile(cmd.Context(), args[0])
				if err != nil {
					return coded(7, err)
				}
				return emit(l)
			}
			leases, err := s.List(cmd.Context(), false)
			if err != nil {
				return coded(7, err)
			}
			inventory, err := s.Inventory(cmd.Context())
			if err != nil {
				return coded(7, err)
			}
			return emit(map[string]any{"leases": leases, "inventory": inventory})
		})
	}})
	root.AddCommand(&cobra.Command{Use: "capabilities <lease-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			l, err := s.Show(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			caps := []string{}
			seen := map[string]bool{}
			for _, c := range l.Components {
				for _, v := range c.Capabilities {
					if !seen[v] {
						seen[v] = true
						caps = append(caps, v)
					}
				}
			}
			endpoints, err := s.Endpoints(cmd.Context(), l)
			if err != nil {
				return err
			}
			return emit(map[string]any{"lease_id": l.ID, "capabilities": caps, "endpoints": endpoints, "observed_state": l.Observed})
		})
	}})
	doctorRuntime := "compose"
	doctor := &cobra.Command{Use: "doctor [repository|lease-id]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if doctorRuntime != "compose" && doctorRuntime != "android-emulator" && doctorRuntime != "flutter-android" {
			return coded(2, errors.New("--runtime must be compose, android-emulator or flutter-android"))
		}
		if len(args) == 1 {
			if _, e := ulid.ParseStrict(args[0]); e == nil {
				return with(func(s *app.Service) error {
					lease, err := s.Show(cmd.Context(), args[0])
					if err != nil {
						return coded(7, err)
					}
					if err := emit(map[string]any{"lease": lease, "ok": lease.Observed == "ready", "diagnostics": lease.Diagnostics}); err != nil {
						return err
					}
					if lease.Observed != "ready" {
						return coded(3, fmt.Errorf("lease is %s", lease.Observed))
					}
					return nil
				})
			}
		}
		checks := map[string]any{}
		var issues []string
		tools := []string{"git"}
		if doctorRuntime == "compose" {
			tools = append(tools, "docker")
		}
		for _, name := range tools {
			p, err := execx.LookPath(name)
			if err != nil {
				issues = append(issues, "missing "+name+": install it and add it to PATH")
			} else {
				checks[name] = p
			}
		}
		if len(issues) == 0 && (doctorRuntime != "flutter-android" || len(args) == 0) {
			var d map[string]string
			var err error
			if doctorRuntime == "android-emulator" {
				d, err = (androidruntime.Adapter{Runner: execx.OSRunner{}, Processes: execx.NativeDetached{}}).Doctor(cmd.Context())
			} else if doctorRuntime == "flutter-android" {
				var version string
				version, err = (flutterruntime.Adapter{}).Doctor(cmd.Context(), "flutter")
				d = map[string]string{"flutter": version}
			} else {
				d, err = (compose.Client{Runner: execx.OSRunner{}, Policy: policy.Defaults()}).Doctor(cmd.Context())
			}
			checks["runtime"] = d
			if err != nil {
				issues = append(issues, err.Error())
			}
		}
		if len(args) > 0 {
			android := androidruntime.Adapter{Runner: execx.OSRunner{}, Processes: execx.NativeDetached{}}
			var m string
			var err error
			if doctorRuntime == "flutter-android" {
				var applications map[string]any
				m, applications, err = doctorFlutterManifest(cmd.Context(), args[0], flutterruntime.Adapter{}, android)
				checks["applications"] = applications
			} else {
				m, err = doctorManifest(cmd.Context(), args[0], doctorRuntime, android)
			}
			if err != nil {
				issues = append(issues, err.Error())
			} else {
				checks["manifest_digest"] = m
			}
		}
		checks["ok"] = len(issues) == 0
		checks["diagnostics"] = issues
		if err := emit(checks); err != nil {
			return err
		}
		if len(issues) > 0 {
			return coded(3, errors.New(strings.Join(issues, "; ")))
		}
		return nil
	}}
	doctor.Flags().StringVar(&doctorRuntime, "runtime", "compose", "prerequisites for compose, android-emulator or flutter-android")
	root.AddCommand(doctor)
	root.AddCommand(&cobra.Command{Use: "test <lease-id> <test-name>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			run, err := s.Test(cmd.Context(), args[0], args[1])
			if run.ID != "" {
				if e := emit(run); e != nil {
					return e
				}
			}
			if errors.Is(err, app.ErrTestFailed) {
				return coded(5, err)
			}
			return err
		})
	}})
	runID, component := "", ""
	logs := &cobra.Command{Use: "logs <lease-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return with(func(s *app.Service) error {
			l, err := s.Get(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if runID != "" && component != "" {
				return errors.New("--run and --component select different log sources; use one at a time")
			}
			entries := map[string]string{}
			if runID != "" {
				runs, err := s.Store.Runs(cmd.Context(), l.ID)
				if err != nil {
					return err
				}
				found := false
				for _, run := range runs {
					if run.ID != runID {
						continue
					}
					found = true
					for name, path := range map[string]string{"stdout": run.StdoutPath, "stderr": run.StderrPath} {
						b, err := os.ReadFile(path)
						if err != nil {
							return err
						}
						entries[name] = string(b)
					}
				}
				if !found {
					return errors.New("command run not found for lease")
				}
			} else {
				entries, err = runtimeLogEntries(cmd.Context(), s, l, component)
				if err != nil {
					return err
				}
			}
			if *output == "json" {
				return emit(map[string]any{"lease_id": l.ID, "logs": entries})
			}
			keys := make([]string, 0, len(entries))
			for k := range entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintln(out, "== "+k+" ==")
				if _, err := io.WriteString(out, entries[k]); err != nil {
					return err
				}
			}
			return nil
		})
	}}
	logs.Flags().StringVar(&runID, "run", "", "recorded command run ID")
	logs.Flags().StringVar(&component, "component", "", "only services owned by the selected component")
	logs.MarkFlagsMutuallyExclusive("run", "component")
	root.AddCommand(logs)
	addInit(root, emit)
}

// runtimeLogEntries isolates both live and retained logs using the component's
// exact runtime/service contract. Aggregate historical logs cannot be split.
func runtimeLogEntries(ctx context.Context, s *app.Service, lease domain.Lease, component string) (map[string]string, error) {
	entries := map[string]string{}
	selectedRuntime := ""
	selectedAndroid := false
	selectedServices := []string{}
	allowedKinds := map[string]bool{}
	if component != "" {
		for _, c := range lease.Components {
			if c.Name == component {
				selectedRuntime = c.Runtime
				if c.Application != "" {
					allowedKinds["application-build/"+c.Application] = true
				}
				selectedServices = append([]string(nil), c.Services...)
				for _, service := range c.Services {
					allowedKinds["compose-log/"+c.Runtime+"/"+service] = true
				}
				break
			}
		}
		for _, runtime := range lease.Runtimes {
			if runtime.Name == selectedRuntime {
				selectedAndroid = runtime.Type == "android-emulator"
				break
			}
		}
		if selectedRuntime == "" || (!selectedAndroid && len(selectedServices) == 0) {
			return nil, errors.New("component has no allocated runtime services in this lease")
		}
	}
	artifacts, err := s.Store.Artifacts(ctx, lease.ID)
	if err != nil {
		return nil, err
	}
	legacy, matched := false, false
	for _, artifact := range artifacts {
		if artifact.Kind == "compose-log" {
			legacy = true
			if component != "" {
				continue
			}
		} else if strings.HasPrefix(artifact.Kind, "compose-log/") || strings.HasPrefix(artifact.Kind, "application-build/") {
			if component != "" && !allowedKinds[artifact.Kind] {
				continue
			}
		} else {
			continue
		}
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			return nil, err
		}
		entries[artifact.ID] = string(data)
		matched = true
	}
	for _, runtime := range lease.Runtimes {
		if runtime.Type != "android-emulator" || (component != "" && runtime.Name != selectedRuntime) {
			continue
		}
		logs, err := androidProcessLogEntries(s.Home, lease.ID, runtime)
		if err != nil {
			return nil, err
		}
		for name, data := range logs {
			entries[name] = data
			matched = true
		}
	}
	if lease.Observed == "released" {
		if component != "" && !selectedAndroid && legacy && !matched {
			return nil, errors.New("legacy aggregate logs cannot isolate this component; omit --component to inspect the retained aggregate")
		}
		return entries, nil
	}
	for _, runtime := range lease.Runtimes {
		if runtime.Type == "android-emulator" {
			continue
		}
		if component != "" {
			if runtime.Name != selectedRuntime {
				continue
			}
			runtime.Services = append([]string(nil), selectedServices...)
		}
		logs, err := s.Runtime.Logs(ctx, runtime)
		if err != nil {
			return nil, err
		}
		entries[runtime.Name] = evidence.RedactString(logs, evidence.InheritedSecrets())
	}
	return entries, nil
}
