package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

var applicationPackage = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
var applicationActivity = regexp.MustCompile(`^(\.[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*|[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+)$`)

// applicationADB shares device ownership and server policy with Emulator
// observation. It cannot start or replace the shared server and never chooses
// an implicit device. A command's duration is bounded independently of probes.
func (a Adapter) applicationADB(ctx context.Context, r domain.Runtime, timeout time.Duration, args ...string) (execx.Result, error) {
	if r.Type != "android-emulator" || r.Android == nil {
		return execx.Result{}, identityError("application operation requires an Android Emulator runtime")
	}
	observed, err := a.Inspect(ctx, r)
	if err != nil {
		return execx.Result{}, err
	}
	if !observed.Exists || !observed.Ready {
		return execx.Result{}, fmt.Errorf("owned Android Emulator is not ready for application operation")
	}
	protocol, err := a.adbProtocol(ctx, r.Android.SDKPath)
	if err != nil {
		return execx.Result{}, err
	}
	available, err := a.compatibleADB(ctx, protocol)
	if err != nil {
		return execx.Result{}, err
	}
	if !available {
		return execx.Result{}, fmt.Errorf("shared local ADB server is unavailable")
	}
	scoped := []string{"-H", "127.0.0.1", "-P", "5037", "-s", r.Android.Serial}
	return a.runner().Run(ctx, execx.Command{Name: executable(r.Android.SDKPath, "platform-tools", "adb"), Args: append(scoped, args...), UnsetEnv: adbRoutingEnvironment(), Timeout: timeout})
}

func (a Adapter) InstallAPK(ctx context.Context, r domain.Runtime, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("APK path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("APK must be a regular file")
	}
	result, err := a.applicationADB(ctx, r, 2*time.Minute, "install", "-r", path)
	if err != nil {
		return err
	}
	// adb implementations can report package-manager failure in successful
	// command output. Require the explicit installation success marker.
	success := false
	for _, line := range strings.Split(result.Stdout+"\n"+result.Stderr, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "failure") || strings.HasPrefix(lower, "error") || strings.Contains(lower, "exception") {
			return fmt.Errorf("ADB install reported failure")
		}
		if line == "Success" {
			success = true
		}
	}
	if !success {
		return fmt.Errorf("ADB install did not confirm success")
	}
	return nil
}

func (a Adapter) PackageInstalled(ctx context.Context, r domain.Runtime, pkg string) (bool, error) {
	if !applicationPackage.MatchString(pkg) {
		return false, fmt.Errorf("invalid Android application package")
	}
	result, err := a.applicationADB(ctx, r, 10*time.Second, "shell", "pm", "list", "packages", "--user", "0", pkg)
	if err != nil {
		return false, err
	}
	installed := false
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, ok := strings.CutPrefix(line, "package:")
		if !ok || !applicationPackage.MatchString(name) {
			return false, fmt.Errorf("malformed Android package listing")
		}
		if name == pkg {
			installed = true
		}
	}
	if strings.TrimSpace(result.Stderr) != "" {
		return false, fmt.Errorf("Android package listing reported diagnostics")
	}
	return installed, nil
}

func applicationTCPPort(port int) (string, error) {
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("Android reverse TCP port must be between 1 and 65535")
	}
	return "tcp:" + strconv.Itoa(port), nil
}

func (a Adapter) Reverse(ctx context.Context, r domain.Runtime, devicePort, hostPort int) error {
	device, err := applicationTCPPort(devicePort)
	if err != nil {
		return err
	}
	host, err := applicationTCPPort(hostPort)
	if err != nil {
		return err
	}
	_, err = a.applicationADB(ctx, r, 10*time.Second, "reverse", "--no-rebind", device, host)
	return err
}

func parseReverseMappings(output string) (map[int]int, error) {
	result := map[int]int{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed Android reverse mapping")
		}
		// The transport label belongs to adbd and need not equal the Emulator
		// serial. Device selection is provided by the scoped -s invocation.
		ports := [2]int{}
		for i, value := range fields[1:] {
			raw, ok := strings.CutPrefix(value, "tcp:")
			port, err := strconv.Atoi(raw)
			if !ok || err != nil || port < 1 || port > 65535 || raw != strconv.Itoa(port) {
				return nil, fmt.Errorf("unsupported Android reverse mapping endpoint")
			}
			ports[i] = port
		}
		if _, exists := result[ports[0]]; exists {
			return nil, fmt.Errorf("duplicate Android reverse device port")
		}
		result[ports[0]] = ports[1]
	}
	return result, nil
}

func (a Adapter) ReverseMappings(ctx context.Context, r domain.Runtime) (map[int]int, error) {
	result, err := a.applicationADB(ctx, r, 10*time.Second, "reverse", "--list")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.Stderr) != "" {
		return nil, fmt.Errorf("Android reverse listing reported diagnostics")
	}
	return parseReverseMappings(result.Stdout)
}

func (a Adapter) RemoveReverse(ctx context.Context, r domain.Runtime, devicePort, expectedHostPort int) error {
	device, err := applicationTCPPort(devicePort)
	if err != nil {
		return err
	}
	if _, err = applicationTCPPort(expectedHostPort); err != nil {
		return err
	}
	mappings, err := a.ReverseMappings(ctx, r)
	if err != nil {
		return err
	}
	host, exists := mappings[devicePort]
	if !exists {
		return nil
	}
	if host != expectedHostPort {
		return identityError("Android reverse mapping differs from recorded host endpoint")
	}
	_, err = a.applicationADB(ctx, r, 10*time.Second, "reverse", "--remove", device)
	return err
}

func (a Adapter) LaunchActivity(ctx context.Context, r domain.Runtime, pkg, activity string) error {
	if !applicationPackage.MatchString(pkg) || !applicationActivity.MatchString(activity) {
		return fmt.Errorf("invalid Android package or activity")
	}
	result, err := a.applicationADB(ctx, r, 30*time.Second, "shell", "am", "start", "-W", "--user", "0", "-n", pkg+"/"+activity)
	if err != nil {
		return err
	}
	return activityStartResult(result.Stdout + "\n" + result.Stderr)
}

func activityStartResult(output string) error {
	successful := false
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "error") || strings.Contains(lower, "exception") || strings.HasPrefix(lower, "failure") {
			return fmt.Errorf("Android activity launch reported failure")
		}
		if status, ok := strings.CutPrefix(lower, "status:"); ok {
			if strings.TrimSpace(status) != "ok" {
				return fmt.Errorf("Android activity launch did not report OK status")
			}
			successful = true
		}
	}
	if !successful {
		return fmt.Errorf("Android activity launch did not confirm success")
	}
	return nil
}
