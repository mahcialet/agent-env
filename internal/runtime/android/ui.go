package android

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/runtime/android/uihelper"
)

const uiResponseLimit = 1 << 20

var installedUIPath = regexp.MustCompile(`^/data/app/[A-Za-z0-9_./=+~-]+/base\.apk$`)

// ObserveUI never accepts a caller-supplied ADB target. Every device command
// independently reuses the existing runtime identity and shared-server gate.
func (a Adapter) ObserveUI(ctx context.Context, r domain.Runtime, q domain.UIRequest) (o domain.UIObservation, err error) {
	o.Version, o.Backend, o.Confirmed = 1, "android-shell-v1", true
	if q.Version != 1 || q.Package != "" && !applicationPackage.MatchString(q.Package) {
		return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: invalid UI request")
	}
	call := func(limit int, args ...string) (execx.Result, error) {
		v, e := a.applicationADBLimited(ctx, r, 30*time.Second, limit, args...)
		if e != nil {
			var preflight *adbPreflightError
			if !errors.As(e, &preflight) && (errors.Is(e, execx.ErrProcessTreeUnconfirmed) || errors.Is(e, execx.ErrOutputIncomplete)) {
				o.Confirmed = false
			}
			return v, uiSafeError(e)
		}
		return v, nil
	}
	switch q.Operation {
	case "snapshot", "tap", "set-text", "quiesce":
		if len(q.Text) > 4096 || !utf8.ValidString(q.Text) {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: text exceeds UTF-8 limit")
		}
		meta, e := uihelper.Load(os.Getenv("AGENT_ENV_UI_HELPER"))
		if e != nil {
			if q.Operation == "quiesce" {
				const prefix = "uiautomation-v1:source="
				if strings.HasPrefix(q.ExpectedBackend, prefix) {
					parts := strings.Split(strings.TrimPrefix(q.ExpectedBackend, prefix), ":apk=")
					if len(parts) == 2 && len(parts[0]) == 64 && len(parts[1]) == 64 {
						meta = uihelper.Metadata{Version: 1, Package: uihelper.Package, SourceSHA256: parts[0], APKSHA256: parts[1]}
						o.Backend = q.ExpectedBackend
						e = nil
					}
				}
			}
		}
		if e != nil {
			return o, fmt.Errorf("%w: AGENTENV-UI-UNAVAILABLE: build and configure the verified UI helper: %v", app.ErrPrerequisite, e)
		}
		if o.Backend == "" {
			o.Backend = fmt.Sprintf("uiautomation-v%d:source=%s:apk=%s", meta.Version, meta.SourceSHA256, meta.APKSHA256)
		}
		if (q.Operation == "tap" || q.Operation == "set-text") && q.ExpectedBackend != o.Backend {
			o.Status = "stale"
			return o, nil
		}
		v, e := call(4096, "shell", "getprop", "ro.build.version.sdk")
		if e != nil {
			return o, e
		}
		api, e := strconv.Atoi(strings.TrimSpace(v.Stdout))
		if e != nil || api < 26 {
			return o, fmt.Errorf("%w: AGENTENV-UI-UNAVAILABLE: Android API 26 or later required", app.ErrPrerequisite)
		}
		// A template may already contain a different package. Never replace it.
		path, e := a.uiHelperPath(ctx, r)
		if e != nil {
			return o, e
		}
		if path == "" {
			if q.Operation == "quiesce" {
				o.Status = "ok"
				return o, nil
			}
			o.Confirmed = false
			// Install bytes that were verified by Load, rather than reopening the
			// mutable configured path after verification.
			apk, readErr := os.ReadFile(meta.APKPath)
			if readErr != nil {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: verified helper read failed: %w", readErr)
			}
			sum := sha256.Sum256(apk)
			if hex.EncodeToString(sum[:]) != meta.APKSHA256 {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper changed after verification")
			}
			tmp, tempErr := os.CreateTemp(r.Directory, ".agent-env-observer-*.apk")
			if tempErr != nil {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper staging failed: %w", tempErr)
			}
			tmpPath := tmp.Name()
			defer os.Remove(tmpPath)
			if _, tempErr = tmp.Write(apk); tempErr == nil {
				tempErr = tmp.Close()
			} else {
				_ = tmp.Close()
			}
			if tempErr != nil {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper staging failed: %w", tempErr)
			}
			v, e = call(16384, "install", tmpPath)
			if e != nil {
				return o, e
			}
			if !uiInstallConfirmed(v.Stdout) {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper installation unconfirmed")
			}
			o.Confirmed = true
			o.HelperInstalled = true
			path, e = a.uiHelperPath(ctx, r)
			if e != nil {
				return o, e
			}
		}
		path, ok := strings.CutPrefix(path, "package:")
		if !ok || !installedUIPath.MatchString(path) || strings.Contains(path, "..") {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: installed helper path is ambiguous")
		}
		v, e = call(4096, "shell", "sha256sum", path)
		if e != nil {
			return o, e
		}
		fields := strings.Fields(v.Stdout)
		if len(fields) != 2 || fields[0] != meta.APKSHA256 || fields[1] != path {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: installed helper digest conflicts with configured build")
		}
		if q.Operation == "quiesce" {
			o.Confirmed = false
			v, e = call(4096, "shell", "am", "force-stop", uihelper.Package)
			if e != nil {
				return o, e
			}
			if strings.TrimSpace(v.Stdout+v.Stderr) != "" {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper stop unconfirmed")
			}
			v, e = a.applicationADBLimited(ctx, r, 10*time.Second, 4096, "shell", "pidof", uihelper.Package)
			var exit *execx.ExitError
			if !errors.As(e, &exit) || exit.Result.ExitCode != 1 || strings.TrimSpace(v.Stdout+v.Stderr) != "" || ctx.Err() != nil || errors.Is(e, execx.ErrProcessTreeUnconfirmed) || errors.Is(e, execx.ErrOutputIncomplete) {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: helper termination unconfirmed")
			}
			o.Confirmed = true
			o.Status = "ok"
			return o, nil
		}
		data, e := json.Marshal(q)
		if e != nil {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: invalid request")
		}
		o.Confirmed = false
		v, e = call(2*uiResponseLimit, "shell", "am", "instrument", "-w", "-r", "-e", "request", base64.StdEncoding.EncodeToString(data), uihelper.Runner)
		if e != nil {
			return o, e
		}
		parsed, e := decodeUIResponse(v.Stdout)
		if e != nil {
			return o, e
		}
		parsed.Backend = o.Backend
		parsed.Confirmed = true
		parsed.HelperInstalled = o.HelperInstalled
		return parsed, nil
	case "screenshot":
		v, e := call(16<<20, "exec-out", "screencap", "-p")
		if e != nil {
			return o, e
		}
		o.Binary = []byte(v.Stdout)
		o.Status = "ok"
		return o, nil
	case "back", "home", "tap-coordinate", "swipe":
		args := []string{"shell", "input"}
		switch q.Operation {
		case "back":
			args = append(args, "keyevent", "4")
		case "home":
			args = append(args, "keyevent", "3")
		case "tap-coordinate", "swipe":
			if q.X < 0 || q.Y < 0 || q.ToX < 0 || q.ToY < 0 || q.X > 32767 || q.Y > 32767 || q.ToX > 32767 || q.ToY > 32767 {
				return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: coordinates out of range")
			}
			if q.Operation == "tap-coordinate" {
				args = append(args, "tap", strconv.Itoa(q.X), strconv.Itoa(q.Y))
			} else {
				if q.DurationMS < 50 || q.DurationMS > 2000 {
					return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: swipe duration out of range")
				}
				args = append(args, "swipe", strconv.Itoa(q.X), strconv.Itoa(q.Y), strconv.Itoa(q.ToX), strconv.Itoa(q.ToY), strconv.Itoa(q.DurationMS))
			}
		}
		o.Confirmed = false
		v, e := call(4096, args...)
		if e != nil {
			return o, e
		}
		o.Confirmed = true
		if strings.TrimSpace(v.Stdout+v.Stderr) != "" {
			o.Status = "uncertain"
			return o, nil
		}
		o.Status = "ok"
		o.ActionPerformed = true
		return o, nil
	case "logcat":
		if q.Package == "" || q.SinceSeconds < 1 || q.SinceSeconds > 3600 {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: logcat requires package and since 1s..1h")
		}
		pid, e := a.uiPID(ctx, r, q.Package)
		if e != nil {
			return o, e
		}
		v, e := call(4096, "shell", "date", "+%s")
		if e != nil {
			return o, e
		}
		epoch, e := strconv.ParseInt(strings.TrimSpace(v.Stdout), 10, 64)
		if e != nil || epoch <= 0 {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: invalid device clock")
		}
		cutoff := epoch - int64(q.SinceSeconds)
		// Capture a bounded envelope larger than the final 256 KiB artifact limit.
		// Logcat is reduced locally after the process has completed, so ordinary
		// high-volume output does not become an unconfirmed running command.
		v, e = call(16<<20, "shell", "logcat", "-d", "-v", "epoch", "--pid", strconv.Itoa(pid), "-t", "2000")
		if e != nil {
			return o, e
		}
		after, e := a.uiPID(ctx, r, q.Package)
		if e != nil || after != pid {
			return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: package PID changed during log capture")
		}
		o.Binary, o.Truncated = boundedUILog(v.Stdout, cutoff)
		o.PID = pid
		o.Since = strconv.FormatInt(cutoff, 10) + ".000"
		o.Status = "ok"
		return o, nil
	default:
		return o, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: unsupported operation")
	}
}

func (a Adapter) uiPID(ctx context.Context, r domain.Runtime, pkg string) (int, error) {
	v, e := a.applicationADBLimited(ctx, r, 10*time.Second, 4096, "shell", "pidof", pkg)
	if e != nil {
		return 0, uiSafeError(e)
	}
	f := strings.Fields(v.Stdout)
	if len(f) != 1 {
		return 0, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: package must have one current PID")
	}
	pid, e := strconv.Atoi(f[0])
	if e != nil || pid <= 0 {
		return 0, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: invalid package PID")
	}
	return pid, nil
}

func decodeUIResponse(output string) (domain.UIObservation, error) {
	var o domain.UIObservation
	fail := func() (domain.UIObservation, error) {
		return domain.UIObservation{}, fmt.Errorf("AGENTENV-UI-UNAVAILABLE: incomplete or invalid helper response")
	}
	if len(output) > 2*uiResponseLimit {
		return fail()
	}
	count := 0
	completed := false
	codes := 0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "INSTRUMENTATION_CODE:") {
			codes++
			completed = line == "INSTRUMENTATION_CODE: 0"
		}
		if value, ok := strings.CutPrefix(line, "INSTRUMENTATION_RESULT: response64="); ok {
			count++
			b, e := base64.StdEncoding.DecodeString(value)
			if e != nil || len(b) > uiResponseLimit {
				return fail()
			}
			if json.Unmarshal(b, &o) != nil {
				return fail()
			}
			o.Raw = b
		}
	}
	if count != 1 || codes != 1 || !completed || o.Version != 1 || len(o.Snapshot.Nodes) > 1000 {
		return fail()
	}
	switch o.Status {
	case "ok", "unavailable", "stale", "ambiguous", "refused", "uncertain":
	default:
		return fail()
	}
	seen := map[string]bool{}
	for _, n := range o.Snapshot.Nodes {
		if n.Ref == "" || seen[n.Ref] || len(n.Fingerprint) != 64 {
			return fail()
		}
		seen[n.Ref] = true
		if _, e := hex.DecodeString(n.Fingerprint); e != nil {
			return fail()
		}
		if n.Editable || n.Password {
			for _, value := range []string{n.Text, n.Description, n.Hint} {
				if value != "" && value != "[REDACTED]" {
					return fail()
				}
			}
		}
	}
	return o, nil
}

func boundedUILog(raw string, cutoff int64) ([]byte, bool) {
	raw = evidence.RedactString(raw, evidence.InheritedSecrets())
	var out strings.Builder
	count := 0
	truncated := false
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		sec, err := strconv.ParseFloat(f[0], 64)
		if err != nil || math.IsNaN(sec) || math.IsInf(sec, 0) || sec < float64(cutoff) {
			continue
		}
		if count == 2000 || out.Len()+len(line)+1 > 256<<10 {
			truncated = true
			break
		}
		out.WriteString(line)
		out.WriteByte('\n')
		count++
	}
	// The device tail itself is capped even if local filtering reduced line count.
	return []byte(out.String()), truncated || len(strings.Split(strings.TrimSpace(raw), "\n")) >= 2000
}

// Never retain device output or request-bearing ExitError.Command here. Even
// malformed instrumentation diagnostics could echo the base64 text payload.
func uiSafeError(err error) error {
	safe := fmt.Errorf("AGENTENV-UI-UNAVAILABLE: owned device command failed")
	for _, marker := range []error{context.Canceled, context.DeadlineExceeded, domain.ErrLockLost, domain.ErrResourceIdentity, execx.ErrProcessTreeUnconfirmed, execx.ErrOutputIncomplete} {
		if errors.Is(err, marker) {
			safe = errors.Join(safe, marker)
		}
	}
	if errors.Is(err, exec.ErrNotFound) {
		safe = errors.Join(app.ErrPrerequisite, safe)
	}
	return safe
}

func (a Adapter) uiHelperPath(ctx context.Context, r domain.Runtime) (string, error) {
	v, e := a.applicationADBLimited(ctx, r, 10*time.Second, 4096, "shell", "pm", "path", uihelper.Package)
	if e != nil {
		var exit *execx.ExitError
		if errors.As(e, &exit) && exit.Result.ExitCode == 1 && strings.TrimSpace(v.Stdout+v.Stderr) == "" && ctx.Err() == nil && !errors.Is(e, execx.ErrProcessTreeUnconfirmed) && !errors.Is(e, execx.ErrOutputIncomplete) {
			return "", nil
		}
		return "", uiSafeError(e)
	}
	return strings.TrimSpace(v.Stdout), nil
}

func uiInstallConfirmed(output string) bool {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "Success" {
			count++
		}
		if strings.Contains(line, "Failure") || strings.HasPrefix(line, "Error") {
			return false
		}
	}
	return count == 1
}
