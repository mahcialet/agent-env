package cli

import (
	"errors"
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"io"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/spf13/cobra"
)

func uiCommand(output *string, emit func(any) error, out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "ui", Short: "Observe or interact with a lease-owned Android UI"}
	for _, op := range []string{"snapshot", "screenshot", "tap", "set-text", "tap-coordinate", "back", "home", "swipe", "wait", "logcat"} {
		var o app.UIOptions
		o.Operation = op
		cmd := &cobra.Command{Use: op + " LEASE", Short: "Android UI " + op, Args: cobra.ExactArgs(1)}
		f := cmd.Flags()
		f.StringVar(&o.Application, "application", "", "Select durable application and package")
		f.StringVar(&o.Runtime, "runtime", "", "Select allocated Android runtime")
		f.DurationVar(&o.Timeout, "timeout", 30*time.Second, "Operation timeout (up to 60s)")
		switch op {
		case "snapshot", "wait":
			f.BoolVar(&o.AllWindows, "all-windows", false, "Include system windows")
		case "tap", "set-text":
			f.StringVar(&o.Snapshot, "snapshot", "", "Registered snapshot ID")
			f.StringVar(&o.Node, "node", "", "Snapshot-local node reference")
			_ = cmd.MarkFlagRequired("snapshot")
			_ = cmd.MarkFlagRequired("node")
		}
		if op == "set-text" {
			f.StringVar(&o.Text, "text", "", "Replacement UTF-8 text (not retained)")
			_ = cmd.MarkFlagRequired("text")
		}
		if op == "wait" {
			f.StringVar(&o.Contains, "contains", "", "Wait for visible semantic text")
			_ = cmd.MarkFlagRequired("contains")
		}
		if op == "logcat" {
			f.DurationVar(&o.Since, "since", 30*time.Second, "Current-PID device log lookback (1s..1h)")
			_ = cmd.MarkFlagRequired("application")
		}
		if op == "tap-coordinate" || op == "swipe" {
			f.IntVar(&o.X, "x", 0, "Start X in device pixels")
			f.IntVar(&o.Y, "y", 0, "Start Y in device pixels")
			_ = cmd.MarkFlagRequired("x")
			_ = cmd.MarkFlagRequired("y")
		}
		if op == "swipe" {
			f.IntVar(&o.ToX, "to-x", 0, "End X in device pixels")
			f.IntVar(&o.ToY, "to-y", 0, "End Y in device pixels")
			f.DurationVar(&o.Duration, "duration", 300*time.Millisecond, "Swipe duration (50ms..2s)")
			_ = cmd.MarkFlagRequired("to-x")
			_ = cmd.MarkFlagRequired("to-y")
		}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			for _, name := range []string{"timeout", "since", "duration"} {
				if cmd.Flags().Changed(name) {
					value, e := cmd.Flags().GetDuration(name)
					if e == nil && value == 0 {
						return uiExit(fmt.Errorf("%w: --%s must be positive", domain.ErrUIInput, name))
					}
				}
			}
			svc, closeStore, err := openService(out, errOut)
			if err != nil {
				return uiExit(err)
			}
			defer closeStore()
			result, err := svc.UI(cmd.Context(), args[0], o)
			if err != nil {
				return uiExit(err)
			}
			if *output == "json" {
				return emit(result)
			}
			if result.Snapshot != nil {
				s := result.Snapshot
				fmt.Fprintf(out, "Snapshot %s  runtime=%s  serial=%s  package=%s  truncated=%t\n", s.ID, s.Runtime, s.Serial, s.Package, s.Tree.Truncated)
				for _, n := range s.Tree.Nodes {
					label := strings.TrimSpace(n.Description + " " + n.Text)
					fmt.Fprintf(out, "%s parent=%s %s %q bounds=%v enabled=%t editable=%t focused=%t\n", n.Ref, n.Parent, n.Class, label, n.Bounds, n.Enabled, n.Editable, n.Focused)
				}
			} else {
				fmt.Fprintf(out, "%s: %s\n", o.Operation, result.Observation.Status)
				writeUILog(out, result.Observation.Log)
			}
			for _, artifact := range result.Artifacts {
				fmt.Fprintf(out, "%s: %s\n", artifact.Kind, artifact.Path)
			}
			return nil
		}
		root.AddCommand(cmd)
	}
	var runID string
	recoverCmd := &cobra.Command{Use: "recover LEASE", Short: "Confirm an interrupted observer helper stopped without retrying input", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		svc, closeStore, err := openService(out, errOut)
		if err != nil {
			return uiExit(err)
		}
		defer closeStore()
		result, err := svc.RecoverUI(cmd.Context(), args[0], runID)
		if err != nil {
			return uiExit(err)
		}
		return emit(result)
	}}
	recoverCmd.Flags().StringVar(&runID, "run", "", "Interrupted observer run ID")
	_ = recoverCmd.MarkFlagRequired("run")
	root.AddCommand(recoverCmd)
	return root
}

func uiExit(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, app.ErrPrerequisite) {
		return coded(3, err)
	}
	if errors.Is(err, domain.ErrUIInput) || errors.Is(err, sqlite.ErrNotFound) || strings.Contains(err.Error(), "AGENTENV-UI-STALE") || strings.Contains(err.Error(), "AGENTENV-UI-AMBIGUOUS") {
		return coded(2, err)
	}
	return coded(7, err)
}

func writeUILog(out io.Writer, log string) {
	if log != "" {
		fmt.Fprint(out, log)
		if !strings.HasSuffix(log, "\n") {
			fmt.Fprintln(out)
		}
	}
}
