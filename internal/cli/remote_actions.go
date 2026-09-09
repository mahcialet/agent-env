package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/worker"
	"github.com/spf13/cobra"
)

// remoteActionPayload translates the registered UI/browser flag schema into the
// typed worker request. It does not open local state, submit work, or send argv.
// Application-level semantic validation still runs in app on the assigned worker.
func remoteActionPayload(cmd *cobra.Command, args []string) (string, json.RawMessage, error) {
	if cmd.Parent() == nil {
		return "", nil, errors.New("remote action requires a UI or browser subcommand")
	}
	family := cmd.Parent().Name()
	if family != "ui" && family != "browser" {
		return "", nil, errors.New("unsupported remote action family")
	}
	if cmd.Name() == "set-text" {
		return "", nil, protocol.ErrTransientInput
	}
	if err := cmd.ValidateArgs(args); err != nil {
		return "", nil, err
	}
	if err := cmd.ValidateRequiredFlags(); err != nil {
		return "", nil, err
	}
	f := cmd.Flags()
	if family == "ui" && cmd.Name() == "recover" {
		runID, err := f.GetString("run")
		if err != nil {
			return "", nil, err
		}
		payload, err := json.Marshal(worker.Request{Name: runID})
		return "ui-recover", payload, err
	}
	timeout, err := f.GetDuration("timeout")
	if err != nil {
		return "", nil, err
	}
	if f.Changed("timeout") && timeout <= 0 {
		return "", nil, errors.New("timeout must be positive")
	}
	duration := time.Duration(0)
	if f.Lookup("duration") != nil {
		duration, err = f.GetDuration("duration")
		if err != nil {
			return "", nil, err
		}
		if f.Changed("duration") && duration <= 0 {
			return "", nil, errors.New("duration must be positive")
		}
	}
	get := func(name string) string { value, _ := f.GetString(name); return value }
	integer := func(name string) int { value, _ := f.GetInt(name); return value }
	var request worker.Request
	switch family {
	case "ui":
		since := time.Duration(0)
		if f.Lookup("since") != nil {
			since, err = f.GetDuration("since")
			if err != nil {
				return "", nil, err
			}
			if f.Changed("since") && since <= 0 {
				return "", nil, errors.New("since must be positive")
			}
		}
		all, _ := f.GetBool("all-windows")
		request.UI = app.UIOptions{Operation: cmd.Name(), Application: get("application"), Runtime: get("runtime"), Snapshot: get("snapshot"), Node: get("node"), Text: get("text"), Contains: get("contains"), AllWindows: all, Timeout: timeout, Duration: duration, Since: since, X: integer("x"), Y: integer("y"), ToX: integer("to-x"), ToY: integer("to-y")}
	case "browser":
		request.Browser = app.BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: cmd.Name(), Page: get("page"), URL: get("url"), Node: get("node"), Text: get("text"), Key: get("key"), Contains: get("contains"), Role: get("role"), WaitFor: get("wait-for"), DeltaX: integer("delta-x"), DeltaY: integer("delta-y"), Duration: duration}, Browser: get("browser"), Snapshot: get("snapshot"), Timeout: timeout}
	default:
		return "", nil, fmt.Errorf("unsupported remote action %s", family)
	}
	payload, err := json.Marshal(request)
	if err == nil {
		err = protocol.ValidateDurablePayload(family, payload)
	}
	if err != nil {
		return "", nil, err
	}
	return family, payload, err
}
