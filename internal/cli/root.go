// Package cli owns argument parsing, adapter wiring, and stable output.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/source/gitcli"
	"github.com/spf13/cobra"
)

const Version = "0.1.0-dev"

type Envelope struct {
	SchemaVersion int `json:"schema_version"`
	Data          any `json:"data"`
}

func New(out, errOut io.Writer) *cobra.Command {
	output := "table"
	root := &cobra.Command{Use: "agent-env", Short: "Pinned, isolated local development environment leases", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVar(&output, "output", "table", "output format: table or json")
	root.PersistentPreRunE = func(*cobra.Command, []string) error {
		if output != "json" && output != "table" {
			return fmt.Errorf("invalid output %q; use table or json", output)
		}
		return nil
	}
	emit := func(v any) error {
		if output == "json" {
			return json.NewEncoder(out).Encode(Envelope{1, v})
		}
		switch p := v.(type) {
		case app.Plan:
			fmt.Fprintf(out, "Repository: %s\nStack: %s\n", p.Repository, p.Stack)
			for _, s := range p.Sources {
				fmt.Fprintf(out, "Source %-12s %s -> %s\n", s.Alias, s.RequestedRef, s.Commit)
			}
			for _, c := range p.Components {
				fmt.Fprintf(out, "Component %-12s runtime=%s services=%v\n", c.Name, c.Runtime, c.Services)
			}
		case []domain.Lease:
			w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSTACK\tOWNER\tDESIRED\tOBSERVED\tEXPIRES")
			for _, l := range p {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", l.ID, l.Stack, l.Owner, l.Desired, l.Observed, l.ExpiresAt.Format(time.RFC3339))
			}
			return w.Flush()
		default:
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(b))
		}
		return nil
	}
	root.AddCommand(&cobra.Command{Use: "version", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		if output == "json" {
			return emit(map[string]string{"version": Version})
		}
		_, err := fmt.Fprintln(out, "agent-env "+Version)
		return err
	}})
	root.AddCommand(&cobra.Command{Use: "validate [repository]", Args: cobra.MaximumNArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		repo := "."
		if len(args) > 0 {
			repo = args[0]
		}
		m, err := config.Load(repo)
		if err != nil {
			return err
		}
		return emit(map[string]any{"valid": true, "manifest_digest": config.Digest(m)})
	}})
	options := app.PlanOptions{}
	plan := &cobra.Command{Use: "plan [repository]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		options.Repository = "."
		if len(args) > 0 {
			options.Repository = args[0]
		}
		p, err := app.BuildPlan(cmd.Context(), options, gitSource{gitcli.Client{Runner: execx.OSRunner{}}})
		if err != nil {
			return err
		}
		return emit(p)
	}}
	plan.Flags().StringVar(&options.Stack, "stack", "", "named component stack")
	plan.Flags().StringVar(&options.Ref, "ref", "", "requested ref for a single-source manifest")
	plan.Flags().StringVar(&options.ManifestPath, "manifest", "", "explicit manifest path relative to repository")
	plan.Flags().StringToStringVar(&options.SourceRefs, "source", nil, "source alias=ref overrides")
	root.AddCommand(plan)
	addLifecycle(root, &output, emit, out, errOut)
	return root
}

type gitSource struct{ client gitcli.Client }

func (g gitSource) Resolve(ctx context.Context, repo, ref string) (domain.Source, error) {
	s, err := g.client.Resolve(ctx, repo, ref)
	return domain.Source{RepositoryID: s.RepositoryID, RepositoryPath: s.RepositoryPath, RequestedRef: s.RequestedRef, Commit: s.Commit, ResolvedAt: time.Now().UTC()}, err
}
func worktree(s domain.Source) gitcli.Worktree {
	return gitcli.Worktree{Resolved: gitcli.Resolved{RepositoryID: s.RepositoryID, RepositoryPath: s.RepositoryPath, RequestedRef: s.RequestedRef, Commit: s.Commit}, Path: s.WorktreePath}
}
func (g gitSource) Materialize(ctx context.Context, s domain.Source) error {
	w := worktree(s)
	_, err := g.client.Materialize(ctx, w.Resolved, w.Path)
	return err
}
func (g gitSource) Inspect(ctx context.Context, s domain.Source) (app.SourceObservation, error) {
	o, err := g.client.Inspect(ctx, worktree(s))
	return app.SourceObservation{Exists: o.Exists, Registered: o.Registered, TrackedDirty: o.TrackedDirty, Commit: o.Commit}, err
}
func (g gitSource) Remove(ctx context.Context, s domain.Source, force bool) error {
	return g.client.Remove(ctx, worktree(s), force)
}
