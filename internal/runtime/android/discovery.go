// Package android manages isolated, lease-owned Android Emulator resources.
package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

// Adapter keeps SDK commands and persistent processes behind injected native
// execution boundaries. It never changes host AVDs or installs SDK packages.
type Adapter struct {
	Runner    execx.Runner
	Processes execx.DetachedProcess
	// ProbeADB injects the read-only shared-server protocol boundary in tests.
	ProbeADB func(context.Context) (protocol int, exists bool, err error)
}

func (a Adapter) runner() execx.Runner {
	if a.Runner != nil {
		return a.Runner
	}
	return execx.OSRunner{}
}

func (a Adapter) processes() execx.DetachedProcess {
	if a.Processes != nil {
		return a.Processes
	}
	return execx.NativeDetached{}
}

func executable(root, dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(root, dir, name)
}

func sdkRoot() (string, error) {
	root := os.Getenv("ANDROID_HOME")
	if other := os.Getenv("ANDROID_SDK_ROOT"); root == "" {
		root = other
	} else if other != "" && filepath.Clean(root) != filepath.Clean(other) {
		one, oneErr := os.Stat(root)
		two, twoErr := os.Stat(other)
		if oneErr != nil || twoErr != nil || !os.SameFile(one, two) {
			return "", fmt.Errorf("ANDROID_HOME and ANDROID_SDK_ROOT disagree")
		}
	}
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		switch runtime.GOOS {
		case "darwin":
			root = filepath.Join(home, "Library", "Android", "sdk")
		case "windows":
			local := os.Getenv("LOCALAPPDATA")
			if local == "" {
				local = filepath.Join(home, "AppData", "Local")
			}
			root = filepath.Join(local, "Android", "Sdk")
		default:
			root = filepath.Join(home, "Android", "Sdk")
		}
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("Android SDK path must be absolute")
	}
	root = filepath.Clean(root)
	for _, p := range []string{executable(root, "emulator", "emulator"), executable(root, "platform-tools", "adb")} {
		st, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("Android SDK prerequisite %s: %w", p, err)
		}
		if !st.Mode().IsRegular() {
			return "", fmt.Errorf("Android SDK tool is not a regular file: %s", p)
		}
	}
	return root, nil
}

func avdRoot() (string, error) {
	if p := os.Getenv("ANDROID_AVD_HOME"); p != "" {
		if !filepath.IsAbs(p) {
			return "", fmt.Errorf("ANDROID_AVD_HOME must be absolute")
		}
		return filepath.Clean(p), nil
	}
	if p := os.Getenv("ANDROID_USER_HOME"); p != "" {
		if !filepath.IsAbs(p) {
			return "", fmt.Errorf("ANDROID_USER_HOME must be absolute")
		}
		return filepath.Join(p, "avd"), nil
	}
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".android", "avd"), err
}

func (a Adapter) Doctor(ctx context.Context) (map[string]string, error) {
	result := map[string]string{}
	root, err := sdkRoot()
	if err != nil {
		result["android"] = "unavailable"
		return result, err
	}
	result["sdk"] = root
	for _, tool := range []struct{ key, dir, name, arg string }{{"emulator", "emulator", "emulator", "-version"}, {"adb", "platform-tools", "adb", "version"}, {"acceleration", "emulator", "emulator", "-accel-check"}} {
		r, err := a.runner().Run(ctx, execx.Command{Name: executable(root, tool.dir, tool.name), Args: []string{tool.arg}, Timeout: 15 * time.Second})
		if err != nil {
			result[tool.key] = "unavailable"
			return result, fmt.Errorf("Android %s prerequisite: %w", tool.key, err)
		}
		result[tool.key] = strings.TrimSpace(r.Stdout + "\n" + r.Stderr)
	}
	protocol, err := parseADBProtocol(result["adb"])
	if err != nil {
		return result, err
	}
	ready, err := a.compatibleADB(ctx, protocol)
	if err != nil {
		result["adb_server"] = "unavailable"
		return result, err
	}
	result["adb_server"] = "not running; separate SDK server startup required"
	if ready {
		result["adb_server"] = fmt.Sprintf("compatible protocol %d", protocol)
	}
	avds, err := avdRoot()
	if err != nil {
		return result, err
	}
	entries, err := os.ReadDir(avds)
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ini") {
			names = append(names, strings.TrimSuffix(e.Name(), ".ini"))
		}
	}
	sort.Strings(names)
	result["templates"] = strings.Join(names, ",")
	return result, nil
}

func safeName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func readINI(path string) (map[string]string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 1024*1024 {
		return nil, fmt.Errorf("AVD configuration must be a regular file of at most 1 MiB: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid AVD configuration line")
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if _, ok := result[k]; ok {
			return nil, fmt.Errorf("duplicate AVD configuration key %s", k)
		}
		if strings.ContainsAny(k+v, "\x00\r\n") {
			return nil, fmt.Errorf("invalid AVD configuration characters")
		}
		result[k] = v
	}
	return result, nil
}

func within(root, path string) bool {
	r, err := filepath.Rel(root, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

func (a Adapter) Validate(ctx context.Context, template string) (domain.AndroidEmulator, error) {
	var out domain.AndroidEmulator
	if !safeName(template) {
		return out, fmt.Errorf("invalid Android AVD template name")
	}
	root, err := sdkRoot()
	if err != nil {
		return out, err
	}
	base, err := avdRoot()
	if err != nil {
		return out, err
	}
	ini, err := readINI(filepath.Join(base, template+".ini"))
	if err != nil {
		return out, fmt.Errorf("AVD template %s: %w", template, err)
	}
	path := ini["path"]
	if path == "" && ini["path.rel"] != "" {
		path = filepath.Join(filepath.Dir(base), filepath.FromSlash(ini["path.rel"]))
	}
	if !filepath.IsAbs(path) {
		return out, fmt.Errorf("AVD template path must be absolute")
	}
	path = filepath.Clean(path)
	if err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("AVD template contains a symlink: %s", p)
		}
		if strings.HasSuffix(d.Name(), ".lock") {
			return fmt.Errorf("AVD template has active or stale lock: %s", p)
		}
		return nil
	}); err != nil {
		return out, err
	}
	cfg, err := readINI(filepath.Join(path, "config.ini"))
	if err != nil {
		return out, err
	}
	image := filepath.FromSlash(cfg["image.sysdir.1"])
	if image == "" {
		return out, fmt.Errorf("AVD template has no system image")
	}
	if !filepath.IsAbs(image) {
		image = filepath.Join(root, image)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return out, err
	}
	image, err = filepath.EvalSymlinks(image)
	if err != nil {
		return out, fmt.Errorf("AVD system image: %w", err)
	}
	if !within(resolvedRoot, image) {
		return out, fmt.Errorf("AVD system image must be inside SDK")
	}
	for _, name := range []string{"system.img"} {
		st, err := os.Stat(filepath.Join(image, name))
		if err != nil || !st.Mode().IsRegular() {
			return out, fmt.Errorf("AVD system image requires %s", name)
		}
	}
	// Older SDK images ship a userdata.img seed. Current images instead ship a
	// data/ tree and let Emulator construct the private data partition itself.
	if st, err := os.Stat(filepath.Join(image, "userdata.img")); err != nil || !st.Mode().IsRegular() {
		if st, err := os.Stat(filepath.Join(image, "data")); err != nil || !st.IsDir() {
			return out, fmt.Errorf("AVD system image requires userdata.img or data directory")
		}
	}
	// Templates supply device configuration, never writable images. Reject explicit
	// external writable paths rather than silently inheriting host AVD state.
	for _, k := range []string{"sdcard.path", "disk.dataPartition.path", "disk.cachePartition.path", "snapshot.storage.path"} {
		if p := cfg[k]; p != "" {
			if !filepath.IsAbs(p) {
				p = filepath.Join(path, p)
			}
			if !within(path, filepath.Clean(p)) {
				return out, fmt.Errorf("AVD template %s escapes writable template directory", k)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	out.Template = template
	out.SDKPath = root
	out.TemplatePath = path
	out.SystemImage = image
	return out, nil
}
