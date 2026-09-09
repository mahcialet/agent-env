package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/client"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/remotesource"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

type remoteFlags struct {
	Controller, CA, Cert, Key, OperationID, Host string
	Wait                                         time.Duration
}

func (f *remoteFlags) client() (*client.Client, error) {
	if f.Controller == "" || f.CA == "" || f.Cert == "" || f.Key == "" {
		return nil, errors.New("remote mode requires --controller and explicit CA, certificate, and private key files")
	}
	config, e := auth.LoadTLS(f.CA, f.Cert, f.Key, false)
	if e != nil {
		return nil, e
	}
	return client.New(f.Controller, config)
}
func addRemote(root *cobra.Command, emit func(any) error, errOut io.Writer) {
	f := &remoteFlags{}
	flags := root.PersistentFlags()
	flags.StringVar(&f.Controller, "controller", "", "explicit HTTPS controller URL")
	flags.StringVar(&f.CA, "tls-ca", "", "trusted CA PEM file for remote mode")
	flags.StringVar(&f.Cert, "tls-cert", "", "enrolled identity certificate PEM file")
	flags.StringVar(&f.Key, "tls-key", "", "identity private key PEM file")
	flags.StringVar(&f.OperationID, "operation-id", "", "stable remote operation ID for safe retries")
	flags.StringVar(&f.Host, "host", "", "explicit worker placement (safety checks still apply)")
	flags.DurationVar(&f.Wait, "wait", 10*time.Minute, "maximum remote operation wait")
	// Wrapping leaves every local command's original parsing and behavior intact.
	// An explicit controller never silently falls back to local execution.
	var wrap func(*cobra.Command)
	wrap = func(command *cobra.Command) {
		for _, child := range command.Commands() {
			wrap(child)
		}
		if command.RunE == nil {
			return
		}
		local := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			if f.Controller == "" {
				return local(cmd, args)
			}
			return f.run(cmd, args, emit, errOut)
		}
	}
	wrap(root)
	addRemoteArtifacts(root, f, emit, errOut)
	addRemoteOperations(root, f, emit)
	addRemoteHosts(root, f, emit)
	addServices(root, f, emit, errOut)
}
func (f *remoteFlags) run(cmd *cobra.Command, args []string, emit func(any) error, errOut io.Writer) error {
	if f.Wait < 0 {
		return errors.New("--wait cannot be negative")
	}
	name := cmd.Name()
	// Validate supported route before opening a connection or reading sources.
	switch name {
	case "create", "list", "show", "renew", "reconcile", "destroy", "test", "logs", "artifacts":
	default:
		ancestor := cmd.Parent()
		if ancestor == nil || (ancestor.Name() != "ui" && ancestor.Name() != "browser") {
			return fmt.Errorf("%s is unavailable with --controller", cmd.CommandPath())
		}
	}
	c, e := f.client()
	if e != nil {
		return e
	}
	defer c.Close()
	ctx := cmd.Context()
	if name == "list" {
		v, e := c.ListLeases(ctx)
		if e != nil {
			return e
		}
		mine, _ := cmd.Flags().GetBool("mine")
		state, _ := cmd.Flags().GetString("state")
		owner, _ := cmd.Flags().GetString("owner")
		filtered := make([]protocol.Lease, 0, len(v))
		for _, lease := range v {
			if mine && !ownerMatches(lease.Owner, owner, cmd.Flags().Changed("owner") || os.Getenv("AGENT_ENV_OWNER") != "") {
				continue
			}
			if state != "" && !strings.EqualFold(lease.State, state) {
				continue
			}
			filtered = append(filtered, lease)
		}
		return emit(filtered)
	}
	if name == "show" {
		v, e := c.GetLease(ctx, args[0])
		if e != nil {
			return e
		}
		return emit(v)
	}
	id := f.OperationID
	if id == "" {
		id = ulid.Make().String()
	}
	fmt.Fprintf(errOut, "Remote operation: %s\n", id)
	var op protocol.Operation
	if name == "create" {
		op, e = f.create(ctx, c, cmd, args, id)
	} else {
		if len(args) == 0 {
			return errors.New("remote operation requires an explicit lease ID")
		}
		kind := name
		if kind == "artifacts" {
			kind = "artifact"
		}
		payload := map[string]any{}
		for _, flag := range []string{"force", "dry-run"} {
			if cmd.Flags().Lookup(flag) != nil {
				v, _ := cmd.Flags().GetBool(flag)
				key := flag
				if key == "dry-run" {
					key = "dry_run"
				}
				payload[key] = v
			}
		}
		if cmd.Flags().Lookup("ttl") != nil {
			v, _ := cmd.Flags().GetDuration("ttl")
			payload["ttl"] = v
		}
		for _, flag := range []string{"run", "component"} {
			if cmd.Flags().Lookup(flag) != nil {
				v, _ := cmd.Flags().GetString(flag)
				payload[flag] = v
			}
		}
		if name == "test" {
			payload["name"] = args[1]
		}
		raw, _ := json.Marshal(payload)
		if p := cmd.Parent(); p != nil && (p.Name() == "ui" || p.Name() == "browser") {
			kind, raw, e = remoteActionPayload(cmd, args)
			if e != nil {
				return e
			}
		}
		op, e = c.Submit(ctx, protocol.SubmitRequest{OperationID: id, LeaseID: args[0], Kind: kind, Payload: raw})
	}
	if e != nil {
		return fmt.Errorf("operation %s submission: %w (inspect/retry this same ID before another mutation)", id, e)
	}
	return f.wait(ctx, c, op, emit)
}
func (f *remoteFlags) create(ctx context.Context, c *client.Client, cmd *cobra.Command, args []string, id string) (protocol.Operation, error) {
	var zero protocol.Operation
	options := app.PlanOptions{Repository: "."}
	if len(args) > 0 {
		options.Repository = args[0]
	}
	options.Stack, _ = cmd.Flags().GetString("stack")
	options.Ref, _ = cmd.Flags().GetString("ref")
	options.ManifestPath, _ = cmd.Flags().GetString("manifest")
	options.SourceRefs, _ = cmd.Flags().GetStringToString("source")
	temp, e := os.MkdirTemp("", "agent-env-source-")
	if e != nil {
		return zero, e
	}
	defer os.RemoveAll(temp)
	cas, e := blobstore.New(filepath.Join(temp, "blobs"))
	if e != nil {
		return zero, e
	}
	pkg, e := remotesource.Build(ctx, options, cas)
	if e != nil {
		return zero, e
	}
	digests := map[string]bool{pkg.ManifestBlobDigest: true}
	var sources []protocol.Source
	for _, source := range pkg.Sources {
		digests[source.BlobDigest] = true
		sources = append(sources, protocol.Source{Alias: source.Alias, Commit: source.Commit, BundleDigest: source.BlobDigest})
	}
	for digest := range digests {
		file, _, e := cas.Open(ctx, digest, blobstore.SourceLimit)
		if e != nil {
			return zero, e
		}
		_, e = c.Upload(ctx, file, digest, "source")
		file.Close()
		if e != nil {
			return zero, e
		}
	}
	raw, e := json.Marshal(pkg)
	if e != nil {
		return zero, e
	}
	ttl, _ := cmd.Flags().GetDuration("ttl")
	purpose, _ := cmd.Flags().GetString("purpose")
	mode, _ := cmd.Flags().GetString("mode")
	owner, _ := cmd.Flags().GetString("owner")
	requestOptions, _ := json.Marshal(map[string]any{"ttl": ttl, "purpose": purpose, "mode": mode, "owner": owner})
	return c.Create(ctx, protocol.CreateRequest{OperationID: id, HostID: f.Host, Stack: pkg.Stack, ManifestDigest: pkg.ManifestDigest, PlanDigest: pkg.PlanDigest, SourceSetDigest: pkg.SourceSetDigest, RepositoryID: pkg.RepositoryID, Manifest: pkg.Manifest, Package: raw, Sources: sources, RequiredCapabilities: pkg.RequiredCapabilities, AndroidSlots: pkg.AndroidSlots, ControlBlobDigest: pkg.ManifestBlobDigest, Options: requestOptions})
}
func (f *remoteFlags) wait(ctx context.Context, c *client.Client, op protocol.Operation, emit func(any) error) error {
	if f.Wait < 0 {
		return errors.New("--wait cannot be negative")
	}
	if f.Wait == 0 {
		return emit(op)
	}
	ctx, cancel := context.WithTimeout(ctx, f.Wait)
	defer cancel()
	for op.Result == nil {
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("operation %s is still pending or delivery is uncertain: %w; inspect this operation ID", op.ID, ctx.Err())
		case <-timer.C:
		}
		updated, e := c.GetOperation(ctx, op.ID)
		if e != nil {
			return fmt.Errorf("inspect operation %s before retrying: %w", op.ID, e)
		}
		op = updated
	}
	if e := emit(op); e != nil {
		return e
	}
	if op.Result.State != "completed" {
		return fmt.Errorf("operation %s ended %s; inspect recorded evidence before retrying", op.ID, op.Result.State)
	}
	return nil
}
func addRemoteOperations(root *cobra.Command, f *remoteFlags, emit func(any) error) {
	cmd := &cobra.Command{Use: "operation <operation-id>", Args: cobra.ExactArgs(1), Short: "Inspect or wait for a durable remote operation", RunE: func(cmd *cobra.Command, args []string) error {
		if f.Controller == "" {
			return errors.New("operation requires --controller")
		}
		c, e := f.client()
		if e != nil {
			return e
		}
		op, e := c.GetOperation(cmd.Context(), args[0])
		if e != nil {
			return e
		}
		return f.wait(cmd.Context(), c, op, emit)
	}}
	root.AddCommand(cmd)
}
func addRemoteHosts(root *cobra.Command, f *remoteFlags, emit func(any) error) {
	hosts := &cobra.Command{Use: "hosts", Short: "Inspect and administer registered workers"}
	for _, action := range []string{"list", "show", "drain", "undrain", "remove"} {
		action := action
		argc := cobra.ExactArgs(1)
		use := action + " <host-id>"
		if action == "list" {
			argc = cobra.NoArgs
			use = action
		}
		hosts.AddCommand(&cobra.Command{Use: use, Args: argc, RunE: func(cmd *cobra.Command, args []string) error {
			if f.Controller == "" {
				return errors.New("hosts requires --controller")
			}
			c, e := f.client()
			if e != nil {
				return e
			}
			var value any
			switch action {
			case "list":
				value, e = c.ListHosts(cmd.Context())
			case "show":
				value, e = c.GetHost(cmd.Context(), args[0])
			case "drain":
				value, e = c.Drain(cmd.Context(), args[0])
			case "undrain":
				value, e = c.Undrain(cmd.Context(), args[0])
			case "remove":
				value, e = c.Remove(cmd.Context(), args[0])
			}
			if e != nil {
				return e
			}
			return emit(value)
		}})
	}
	root.AddCommand(hosts)
}
