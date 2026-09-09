package cli

import (
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/worker"
	"github.com/spf13/cobra"
)

func remoteActionCommand(t *testing.T, family, action string, flags []string) *cobra.Command {
	t.Helper()
	root := New(io.Discard, io.Discard)
	cmd, _, e := root.Find([]string{family, action})
	if e != nil {
		t.Fatal(e)
	}
	if e = cmd.ParseFlags(flags); e != nil {
		t.Fatal(e)
	}
	return cmd
}

func TestRemoteUIFlagsPreserveTypedValues(t *testing.T) {
	cmd := remoteActionCommand(t, "ui", "tap", []string{"--application", "app", "--snapshot", "snapshot-01", "--node", "n7", "--timeout", "8s"})
	kind, payload, e := remoteActionPayload(cmd, []string{"lease-01"})
	if e != nil {
		t.Fatal(e)
	}
	var request worker.Request
	if e = json.Unmarshal(payload, &request); e != nil {
		t.Fatal(e)
	}
	want := app.UIOptions{Operation: "tap", Application: "app", Snapshot: "snapshot-01", Node: "n7", Timeout: 8 * time.Second}
	if kind != "ui" || !reflect.DeepEqual(request.UI, want) {
		t.Fatalf("typed UI options changed: %s %+v", kind, request.UI)
	}
	if strings.Contains(string(payload), "--text") {
		t.Fatal("raw argv in protocol")
	}
}

func TestRemoteBrowserFlagsPreserveTypedValues(t *testing.T) {
	cmd := remoteActionCommand(t, "browser", "scroll", []string{"--browser", "chromium", "--page", "page-01", "--snapshot", "snapshot-01", "--node", "n2", "--delta-x", "-25", "--delta-y", "99", "--timeout", "9s"})
	kind, payload, e := remoteActionPayload(cmd, []string{"lease-01"})
	if e != nil {
		t.Fatal(e)
	}
	var request worker.Request
	if e = json.Unmarshal(payload, &request); e != nil {
		t.Fatal(e)
	}
	want := app.BrowserOptions{BrowserRequest: domain.BrowserRequest{Operation: "scroll", Page: "page-01", Node: "n2", DeltaX: -25, DeltaY: 99}, Browser: "chromium", Snapshot: "snapshot-01", Timeout: 9 * time.Second}
	if kind != "browser" || !reflect.DeepEqual(request.Browser, want) {
		t.Fatalf("typed browser options changed: %s %+v", kind, request.Browser)
	}
	if request.Browser.Prior != nil {
		t.Fatal("client supplied trusted prior snapshot")
	}
}

func TestRemoteActionDefaultsAndRequiredFlags(t *testing.T) {
	for _, tc := range []struct {
		family, action string
		flags          []string
		args           []string
		rejected       bool
	}{
		{"ui", "snapshot", nil, []string{"lease"}, false},
		{"browser", "snapshot", nil, []string{"lease"}, false},
		{"ui", "tap", nil, []string{"lease"}, true},
		{"ui", "swipe", nil, []string{"lease"}, true},
		{"ui", "logcat", nil, []string{"lease"}, true},
		{"ui", "snapshot", []string{"--timeout", "0s"}, []string{"lease"}, true},
		{"ui", "logcat", []string{"--application", "app", "--since", "0s"}, []string{"lease"}, true},
		{"browser", "console", []string{"--duration", "0s"}, []string{"lease"}, true},
		{"browser", "click", []string{"--node", "n1"}, []string{"lease"}, true},
		{"browser", "page-close", nil, []string{"lease"}, true},
		{"ui", "snapshot", nil, nil, true},
		{"browser", "snapshot", nil, []string{"lease", "extra"}, true},
	} {
		t.Run(tc.family+"/"+tc.action+"/"+strings.Join(tc.flags, " "), func(t *testing.T) {
			cmd := remoteActionCommand(t, tc.family, tc.action, tc.flags)
			_, payload, e := remoteActionPayload(cmd, tc.args)
			if (e != nil) != tc.rejected {
				t.Fatalf("validation: %v", e)
			}
			if e != nil {
				return
			}
			var request worker.Request
			if e = json.Unmarshal(payload, &request); e != nil {
				t.Fatal(e)
			}
			if tc.family == "ui" && (request.UI.Timeout != 30*time.Second || request.UI.Duration != 0 || request.UI.Since != 0) {
				t.Fatalf("UI defaults changed: %+v", request.UI)
			}
			if tc.family == "browser" && (request.Browser.Timeout != 30*time.Second || request.Browser.Duration != 0 || request.Browser.WaitFor != "") {
				t.Fatalf("browser defaults changed: %+v", request.Browser)
			}
		})
	}
}

func TestRemoteUIRecoveryUsesRunIdentity(t *testing.T) {
	cmd := remoteActionCommand(t, "ui", "recover", []string{"--run", "run-01"})
	kind, payload, e := remoteActionPayload(cmd, []string{"lease"})
	if e != nil {
		t.Fatal(e)
	}
	var request worker.Request
	if e = json.Unmarshal(payload, &request); e != nil {
		t.Fatal(e)
	}
	if kind != "ui-recover" || request.Name != "run-01" || request.UI.Operation != "" {
		t.Fatalf("wrong recovery routing: %s %+v", kind, request)
	}
	if _, _, e = remoteActionPayload(remoteActionCommand(t, "ui", "recover", nil), []string{"lease"}); e == nil {
		t.Fatal("missing run accepted")
	}
}

func TestRemoteActionSpecificOptionalFlags(t *testing.T) {
	for _, tc := range []struct {
		family, action string
		flags          []string
		check          func(worker.Request) bool
	}{
		{"ui", "swipe", []string{"--x", "0", "--y", "2", "--to-x", "3", "--to-y", "4", "--duration", "500ms"}, func(r worker.Request) bool {
			return r.UI.X == 0 && r.UI.Y == 2 && r.UI.ToX == 3 && r.UI.ToY == 4 && r.UI.Duration == 500*time.Millisecond
		}},
		{"ui", "wait", []string{"--contains", "表示", "--all-windows"}, func(r worker.Request) bool { return r.UI.Contains == "表示" && r.UI.AllWindows }},
		{"ui", "logcat", []string{"--application", "app", "--since", "12s"}, func(r worker.Request) bool { return r.UI.Application == "app" && r.UI.Since == 12*time.Second }},
		{"browser", "wait", []string{"--wait-for", "url", "--contains", "target", "--role", "button"}, func(r worker.Request) bool {
			return r.Browser.WaitFor == "url" && r.Browser.Contains == "target" && r.Browser.Role == "button"
		}},
		{"browser", "console", []string{"--duration", "250ms"}, func(r worker.Request) bool { return r.Browser.Duration == 250*time.Millisecond }},
	} {
		t.Run(tc.family+"/"+tc.action, func(t *testing.T) {
			cmd := remoteActionCommand(t, tc.family, tc.action, tc.flags)
			_, raw, e := remoteActionPayload(cmd, []string{"lease"})
			if e != nil {
				t.Fatal(e)
			}
			var r worker.Request
			if e = json.Unmarshal(raw, &r); e != nil {
				t.Fatal(e)
			}
			if !tc.check(r) {
				t.Fatalf("action flag lost: %+v", r)
			}
		})
	}
}

func TestRemoteTextInputRefusedBeforePayloadSubmission(t *testing.T) {
	for _, family := range []string{"ui", "browser"} {
		for _, value := range []string{"", "remote-secret-canary"} {
			t.Run(family+"/set-text/"+value, func(t *testing.T) {
				cmd := remoteActionCommand(t, family, "set-text", []string{"--text", value})
				_, payload, err := remoteActionPayload(cmd, []string{"lease-id"})
				if !errors.Is(err, protocol.ErrTransientInput) || len(payload) != 0 || strings.Contains(err.Error(), "remote-secret-canary") {
					t.Fatalf("input not safely rejected before submission: payload=%s err=%v", payload, err)
				}
			})
		}
	}
}
