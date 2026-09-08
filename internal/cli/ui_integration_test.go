//go:build flutterintegration

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/runtime/android/uihelper"
)

// Explicit opt-in requires the real Flutter/Compose/Emulator prerequisites and
// a verified companion. Compile-only checks do not exercise these acceptances.
func TestRealAndroidUIObserver(t *testing.T) {
	if _, err := uihelper.Load(os.Getenv("AGENT_ENV_UI_HELPER")); err != nil {
		t.Fatalf("real UI companion prerequisite: %v", err)
	}
	realFlutterAndroidBackendLease(t, true)
}

const observerFlutterWidgets = `
class ObserverFixture extends StatefulWidget {
 const ObserverFixture({super.key});
 @override State<ObserverFixture> createState() => _ObserverFixtureState();
}
class _ObserverFixtureState extends State<ObserverFixture> {
 int count = 0;
 final entry = TextEditingController();
 final password = TextEditingController();
 @override void dispose() { entry.dispose(); password.dispose(); super.dispose(); }
 @override Widget build(BuildContext context) => Scaffold(
  appBar: AppBar(title: const Text('agent-env fixture')),
  body: ListView(padding: const EdgeInsets.all(16), children: [
   Text('Count$count'),
   ElevatedButton(onPressed: () { setState(() { count++; }); debugPrint('agent-env-ui-count-$count'); }, child: Text('Increment$count')),
   ElevatedButton(onPressed: () { showDialog<void>(context: context, builder: (context) => const AlertDialog(title: Text('Observer dialog'))); }, child: const Text('Open observer dialog')),
   TextField(controller: entry, autofocus: true, decoration: const InputDecoration(labelText: 'Observer input')),
   TextField(controller: password, obscureText: true, decoration: const InputDecoration(labelText: 'Observer password')),
   for (int i = 0; i < 30; i++) Padding(padding: const EdgeInsets.all(12), child: Text('Scrollable item $i')),
  ]),
 );
}
`

func executeRealUI(ctx context.Context, args ...string) (app.UIResult, string, error) {
	var out, errOut bytes.Buffer
	cmd := New(&out, &errOut)
	cmd.SetArgs(append([]string{"--output", "json", "ui"}, args...))
	err := cmd.ExecuteContext(ctx)
	var envelope struct {
		SchemaVersion int          `json:"schema_version"`
		Data          app.UIResult `json:"data"`
	}
	if err == nil {
		err = json.Unmarshal(out.Bytes(), &envelope)
		if err == nil && envelope.SchemaVersion != 1 {
			return envelope.Data, out.String(), &realUISchemaError{}
		}
	}
	return envelope.Data, out.String() + errOut.String(), err
}

type realUISchemaError struct{}

func (*realUISchemaError) Error() string { return "unexpected UI envelope version" }
func realUICommand(t *testing.T, ctx context.Context, args ...string) app.UIResult {
	t.Helper()
	result, output, err := executeRealUI(ctx, args...)
	if err != nil {
		t.Fatalf("real UI %s: %v; output=%s", args[0], err, output)
	}
	if result.Run.Status != "passed" || result.Observation.Status != "ok" {
		t.Fatalf("real UI %s did not succeed: %+v", args[0], result)
	}
	return result
}
func realUINode(t *testing.T, result app.UIResult, predicate func(domain.UINode) bool, label string) domain.UINode {
	t.Helper()
	if result.Snapshot == nil || result.Snapshot.Tree.Truncated {
		t.Fatalf("missing/partial snapshot while finding %s", label)
	}
	var found domain.UINode
	count := 0
	for _, node := range result.Snapshot.Tree.Nodes {
		if predicate(node) {
			found = node
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one %s, found %d: %+v", label, count, result.Snapshot.Tree.Nodes)
	}
	return found
}
func realUIArtifact(t *testing.T, result app.UIResult, kind string) []byte {
	t.Helper()
	var data []byte
	count := 0
	for _, artifact := range result.Artifacts {
		if artifact.Kind != kind {
			continue
		}
		count++
		if artifact.LeaseID != result.Run.LeaseID || artifact.RunID != result.Run.ID {
			t.Fatal("artifact has wrong lease/run identity")
		}
		var err error
		data, err = os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if artifact.Digest != hex.EncodeToString(digest[:]) {
			t.Fatal("artifact digest mismatch")
		}
	}
	if count != 1 {
		t.Fatalf("expected one %s artifact, got %d", kind, count)
	}
	return data
}
func exerciseRealAndroidUI(t *testing.T, ctx context.Context, s *app.Service, leases []domain.Lease) {
	t.Helper()
	first := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	sibling := realUICommand(t, ctx, "snapshot", leases[1].ID, "--application", "mobile-app")
	if first.Snapshot == nil || sibling.Snapshot == nil || first.Snapshot.ID == sibling.Snapshot.ID || first.Snapshot.Serial == sibling.Snapshot.Serial || first.Snapshot.Package != "dev.agentenv.lease_fixture" {
		t.Fatal("real snapshots did not retain isolated lease/runtime/package identity")
	}
	for i, snapshot := range []app.UIResult{first, sibling} {
		if snapshot.Snapshot.LeaseID != leases[i].ID || snapshot.Snapshot.Runtime != "phone" {
			t.Fatal("snapshot scope mismatch")
		}
		realUIArtifact(t, snapshot, "ui-snapshot")
		realUIArtifact(t, snapshot, "ui-raw")
		realUINode(t, snapshot, func(n domain.UINode) bool { return n.Text == "Count0" || n.Description == "Count0" }, "initial Count0")
	}
	increment := realUINode(t, first, func(n domain.UINode) bool {
		return n.Clickable && strings.Contains(n.Text+" "+n.Description, "Increment0")
	}, "Increment0 button")
	if _, _, err := executeRealUI(ctx, "tap", leases[1].ID, "--snapshot", first.Snapshot.ID, "--node", increment.Ref); err == nil {
		t.Fatal("cross-lease snapshot authorized input")
	}
	realUICommand(t, ctx, "tap", leases[0].ID, "--snapshot", first.Snapshot.ID, "--node", increment.Ref)
	changed := realUICommand(t, ctx, "wait", leases[0].ID, "--application", "mobile-app", "--contains", "Count1", "--timeout", "20s")
	realUINode(t, changed, func(n domain.UINode) bool { return n.Text == "Count1" || n.Description == "Count1" }, "changed Count1")
	if _, _, err := executeRealUI(ctx, "tap", leases[0].ID, "--snapshot", first.Snapshot.ID, "--node", increment.Ref); err == nil || !strings.Contains(err.Error(), "AGENTENV-UI-STALE") {
		t.Fatalf("stale semantic action was not rejected with stable code: %v", err)
	}
	afterStale := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	realUINode(t, afterStale, func(n domain.UINode) bool { return n.Text == "Count1" || n.Description == "Count1" }, "counter unchanged after stale rejection")
	// Focus belongs to the current accessibility node. Never address fields using
	// ordinals from before a keyboard/layout change.
	field := realUINode(t, afterStale, func(n domain.UINode) bool { return n.Editable && !n.Password }, "ordinary editable field")
	if !field.Focused {
		realUICommand(t, ctx, "tap", leases[0].ID, "--snapshot", afterStale.Snapshot.ID, "--node", field.Ref)
	}
	payloads := []string{"ascii-replacement@example.invalid", "日本語🙂 café Ελληνικά", "置換二回目 — 🧪"}
	for _, payload := range payloads {
		focused := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
		field = realUINode(t, focused, func(n domain.UINode) bool { return n.Editable && !n.Password && n.Focused }, "focused editable field")
		replacement := realUICommand(t, ctx, "set-text", leases[0].ID, "--snapshot", focused.Snapshot.ID, "--node", field.Ref, "--text", payload)
		if !replacement.Observation.ActionPerformed || !replacement.Observation.ReadbackEqual {
			t.Fatal("Unicode replacement lacked actual read-back confirmation")
		}
		for _, artifact := range replacement.Artifacts {
			b, err := os.ReadFile(artifact.Path)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range payloads {
				if bytes.Contains(b, []byte(secret)) {
					t.Fatalf("text payload retained in %s", artifact.Kind)
				}
			}
		}
	}
	redacted := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	for _, artifact := range redacted.Artifacts {
		b, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range payloads {
			if bytes.Contains(b, []byte(secret)) {
				t.Fatalf("editable text exposed by subsequent %s", artifact.Kind)
			}
		}
	}
	passwordFound := false
	for _, node := range redacted.Snapshot.Tree.Nodes {
		if node.Password {
			passwordFound = true
		}
		if (node.Editable || node.Password) && node.Text != "" && node.Text != "[REDACTED]" {
			t.Fatal("editable/password node exposed text")
		}
	}
	if !passwordFound {
		t.Fatal("real Flutter password semantics missing")
	}
	screenshot := realUICommand(t, ctx, "screenshot", leases[0].ID, "--application", "mobile-app")
	imageData := realUIArtifact(t, screenshot, "ui-screenshot")
	image, err := png.Decode(bytes.NewReader(imageData))
	if err != nil {
		t.Fatal(err)
	}
	if image.Bounds().Dx() != screenshot.Width || image.Bounds().Dy() != screenshot.Height || screenshot.Width <= 0 || screenshot.Height <= 0 {
		t.Fatal("PNG dimensions mismatch")
	}
	realUICommand(t, ctx, "tap-coordinate", leases[0].ID, "--runtime", "phone", "--x", "1", "--y", "1")
	// UiAutomation can suppress the IME; Back is tested against an explicit
	// dialog rather than assuming it only dismisses a keyboard.
	dialogScreen := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	dialogButton := realUINode(t, dialogScreen, func(n domain.UINode) bool {
		return n.Clickable && strings.Contains(n.Text+" "+n.Description, "Open observer dialog")
	}, "dialog button")
	realUICommand(t, ctx, "tap", leases[0].ID, "--snapshot", dialogScreen.Snapshot.ID, "--node", dialogButton.Ref)
	dialog := realUICommand(t, ctx, "wait", leases[0].ID, "--application", "mobile-app", "--contains", "Observer dialog", "--timeout", "20s")
	realUINode(t, dialog, func(n domain.UINode) bool {
		return n.Visible && (n.Text == "Observer dialog" || n.Description == "Observer dialog")
	}, "visible dialog title")
	realUICommand(t, ctx, "back", leases[0].ID, "--runtime", "phone")
	realUICommand(t, ctx, "wait", leases[0].ID, "--application", "mobile-app", "--contains", "Count1", "--timeout", "20s")
	beforeSwipe := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	for _, node := range beforeSwipe.Snapshot.Tree.Nodes {
		if node.Visible && (node.Text == "Observer dialog" || node.Description == "Observer dialog") {
			t.Fatal("Back did not dismiss the dialog")
		}
	}
	// The IME may resize the viewport. Derive this explicit gesture from the
	// current scroll container, not from the full screenshot dimensions.
	scroll := realUINode(t, beforeSwipe, func(n domain.UINode) bool { return n.Visible && n.Scrollable }, "visible scroll container")
	x := (scroll.Bounds[0] + scroll.Bounds[2]) / 2
	height := scroll.Bounds[3] - scroll.Bounds[1]
	if height < 80 {
		t.Fatal("scroll viewport too small for the fixture gesture")
	}
	realUICommand(t, ctx, "swipe", leases[0].ID, "--runtime", "phone", "--x", strconv.Itoa(x), "--y", strconv.Itoa(scroll.Bounds[3]-height/8), "--to-x", strconv.Itoa(x), "--to-y", strconv.Itoa(scroll.Bounds[1]+height/8), "--duration", "300ms")
	// A genuine scroll must expose later fixture content, not merely exit zero.
	scrolled := realUICommand(t, ctx, "snapshot", leases[0].ID, "--application", "mobile-app")
	beforeLabels := map[string]bool{}
	for _, node := range beforeSwipe.Snapshot.Tree.Nodes {
		if node.Visible {
			beforeLabels[node.Text+" "+node.Description] = true
		}
	}
	newScrollable := false
	for _, node := range scrolled.Snapshot.Tree.Nodes {
		label := node.Text + " " + node.Description
		if node.Visible && strings.Contains(label, "Scrollable item") && !beforeLabels[label] {
			newScrollable = true
		}
	}
	if !newScrollable {
		t.Fatal("swipe did not reveal new scrollable semantics")
	}
	realUICommand(t, ctx, "home", leases[0].ID, "--runtime", "phone")
	home := realUICommand(t, ctx, "snapshot", leases[0].ID, "--runtime", "phone")
	outside := false
	for _, node := range home.Snapshot.Tree.Nodes {
		if node.Visible && node.Package == "dev.agentenv.lease_fixture" {
			t.Fatal("Home left application semantics visible")
		}
		if node.Visible && node.Package != "" && node.Package != "dev.agentenv.lease_fixture" {
			outside = true
		}
	}
	if !outside {
		t.Fatal("Home did not expose system UI")
	}
	launch, err := s.Test(ctx, leases[0].ID, "resume-app")
	if err != nil || launch.Status != "passed" {
		t.Fatalf("resume owned app after Home: %v", err)
	}
	logs := realUICommand(t, ctx, "logcat", leases[0].ID, "--application", "mobile-app", "--since", "1h")
	logData := realUIArtifact(t, logs, "ui-logcat")
	if logs.Observation.PID <= 0 || logs.Observation.Since == "" || len(logData) == 0 || len(logData) > 256<<10 || bytes.Count(logData, []byte("\n")) > 2000 {
		t.Fatal("logcat evidence lacked bounded current-PID attribution/content")
	}
	// Place timeout last: any erroneously retained running barrier will be caught
	// by the required normal destroy and sibling-isolation checks in the caller.
	started := time.Now()
	if _, _, err := executeRealUI(ctx, "wait", leases[0].ID, "--application", "mobile-app", "--contains", "not-present-observer-sentinel", "--timeout", "5s"); err == nil || !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("missing predicate wait did not report timeout: %v", err)
	}
	if time.Since(started) > 20*time.Second {
		t.Fatal("bounded wait continued beyond its completion allowance")
	}
	sibling = realUICommand(t, ctx, "snapshot", leases[1].ID, "--application", "mobile-app")
	realUINode(t, sibling, func(n domain.UINode) bool { return n.Text == "Count0" || n.Description == "Count0" }, "unmodified sibling Count0")
	t.Log("real CLI snapshot, stale/cross-lease rejection, ASCII/Unicode replacement, privacy, PNG, explicit coordinates, navigation, swipe, PID logcat and bounded wait exercised")
}
