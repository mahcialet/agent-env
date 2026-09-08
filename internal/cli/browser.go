package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/browser/cdp"
	"github.com/spf13/cobra"
)

func browserCommand(output *string, emit func(any) error, out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "browser", Short: "Observe and automate an owned headless Chromium browser through CDP"}
	for _, operation := range []string{"capabilities", "pages", "page-create", "page-close", "navigate", "snapshot", "dom-snapshot", "screenshot", "click", "set-text", "key", "scroll", "wait", "console", "network"} {
		op := operation
		options := app.BrowserOptions{}
		options.Operation = op
		command := &cobra.Command{Use: op + " <lease>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			for _, flag := range []string{"timeout", "duration"} {
				if cmd.Flags().Changed(flag) {
					value, err := cmd.Flags().GetDuration(flag)
					if err != nil || value <= 0 {
						return fmt.Errorf("--%s must be positive", flag)
					}
				}
			}
			service, closeStore, err := openService(out, errOut)
			if err != nil {
				return err
			}
			defer closeStore()
			service.BrowserProvider = cdp.Client{}
			result, opErr := service.Browser(cmd.Context(), args[0], options)
			if result.Run.ID == "" {
				return opErr
			}
			if *output == "json" {
				return errors.Join(opErr, emit(result))
			}
			fmt.Fprintf(out, "Browser %s: %s run=%s\n", op, result.Run.Status, result.Run.ID)
			if result.Snapshot != nil {
				fmt.Fprint(out, app.BrowserSnapshotText(*result.Snapshot))
			}
			for _, page := range result.Observation.Pages {
				fmt.Fprintf(out, "Page %s %q %q\n", page.ID, page.Title, page.URL)
			}
			for _, artifact := range result.Artifacts {
				fmt.Fprintf(out, "Artifact %s %s\n", artifact.Kind, artifact.Path)
			}
			if result.Observation.Detail != "" {
				fmt.Fprintln(out, result.Observation.Detail)
			}
			return opErr
		}}
		command.Flags().StringVar(&options.Browser, "browser", "", "explicit repository browser binding")
		command.Flags().DurationVar(&options.Timeout, "timeout", 30*time.Second, "bounded operation timeout (maximum 60s)")
		if op != "capabilities" && op != "pages" && op != "page-create" {
			command.Flags().StringVar(&options.Page, "page", "", "page target ID (required if ambiguous)")
		}
		switch op {
		case "page-create", "navigate":
			command.Flags().StringVar(&options.URL, "url", "", "http(s) URL or about:blank")
		case "click", "set-text", "key", "scroll":
			command.Flags().StringVar(&options.Snapshot, "snapshot", "", "registered browser snapshot ID")
			command.Flags().StringVar(&options.Node, "node", "", "node reference from the snapshot")
			if op == "set-text" {
				command.Flags().StringVar(&options.Text, "text", "", "Unicode text (redacted in evidence)")
			}
			if op == "key" {
				command.Flags().StringVar(&options.Key, "key", "", "allowlisted key name")
			}
			if op == "scroll" {
				command.Flags().IntVar(&options.DeltaX, "delta-x", 0, "horizontal wheel delta")
				command.Flags().IntVar(&options.DeltaY, "delta-y", 0, "vertical wheel delta")
			}
		case "console", "network":
			command.Flags().DurationVar(&options.Duration, "duration", time.Second, "capture window, maximum 10s; no historical capture")
		case "wait":
			command.Flags().StringVar(&options.WaitFor, "wait-for", "load", "load, url, text or gone")
			command.Flags().StringVar(&options.Contains, "contains", "", "bounded substring predicate")
			command.Flags().StringVar(&options.Role, "role", "", "AX role predicate")
		}
		switch op {
		case "navigate":
			_ = command.MarkFlagRequired("url")
		case "page-close":
			_ = command.MarkFlagRequired("page")
		case "click", "set-text", "key", "scroll":
			_ = command.MarkFlagRequired("snapshot")
			_ = command.MarkFlagRequired("node")
			if op == "set-text" {
				_ = command.MarkFlagRequired("text")
			}
			if op == "key" {
				_ = command.MarkFlagRequired("key")
			}
		}
		root.AddCommand(command)
	}
	return root
}
