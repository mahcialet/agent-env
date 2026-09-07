// Package flutter builds Android applications without owning device resources.
package flutter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
)

type Adapter struct{ Runner execx.Runner }

// Result describes the local build, not a reproducible or retained APK archive.
// The caller must redact command output before persisting it as evidence.
type Result = domain.ApplicationBuildResult

func (a Adapter) runner() execx.Runner {
	if a.Runner != nil {
		return a.Runner
	}
	return execx.OSRunner{}
}

func validExecutable(executable string) bool {
	return strings.TrimSpace(executable) != "" && !strings.ContainsAny(executable, "\x00\r\n") && (filepath.IsAbs(executable) || !strings.ContainsAny(executable, `/\\:`))
}

// Doctor queries machine version output only. It does not invoke flutter doctor,
// install toolchains, accept licenses or create an environment.
func (a Adapter) Doctor(ctx context.Context, executable string) (string, error) {
	if !validExecutable(executable) {
		return "", fmt.Errorf("Flutter executable must be a PATH name or absolute host path")
	}
	r, err := a.runner().Run(ctx, execx.Command{Name: executable, Args: []string{"--version", "--machine"}, Timeout: 30 * time.Second})
	if err != nil {
		return "", fmt.Errorf("Flutter version prerequisite: %w", err)
	}
	var version struct {
		FrameworkVersion string `json:"frameworkVersion"`
	}
	if err := json.Unmarshal([]byte(r.Stdout), &version); err != nil {
		return "", fmt.Errorf("Flutter machine version prerequisite: %w", err)
	}
	if strings.TrimSpace(version.FrameworkVersion) == "" {
		return "", fmt.Errorf("Flutter machine version has no frameworkVersion")
	}
	return version.FrameworkVersion, nil
}

// Validate checks project prerequisites and both source and project confinement.
// An APK need not exist before building; existing symlink path components are
// rejected even when the symlink's destination is inside the source.
func (a Adapter) Validate(sourceRoot, projectDir, artifact string) (string, error) {
	dir, err := paths.Within(sourceRoot, projectDir)
	if err != nil {
		return "", err
	}
	// Within rejects escapes but intentionally permits internal symlinks. Check
	// every lexical project component as well, including project_directory=.
	// Stop at the allocated source root: host ancestors may legitimately be
	// aliases (for example the native macOS temporary directory).
	for probe := dir; ; probe = filepath.Dir(probe) {
		st, err := os.Lstat(probe)
		if err != nil {
			return "", fmt.Errorf("Flutter project directory: %w", err)
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("Flutter project path contains a symlink: %s", probe)
		}
		if probe == filepath.Clean(sourceRoot) {
			break
		}
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("Flutter project directory is missing or not a directory: %s", dir)
	}
	pubspec, err := paths.Within(sourceRoot, filepath.ToSlash(filepath.Join(projectDir, "pubspec.yaml")))
	if err != nil {
		return "", err
	}
	st, err = os.Stat(pubspec)
	if err != nil || !st.Mode().IsRegular() {
		return "", fmt.Errorf("Flutter project requires regular pubspec.yaml")
	}
	android, err := paths.Within(sourceRoot, filepath.ToSlash(filepath.Join(projectDir, "android")))
	if err != nil {
		return "", err
	}
	st, err = os.Stat(android)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("Flutter project requires android directory")
	}
	if _, err := artifactPath(dir, artifact); err != nil {
		return "", err
	}
	return dir, nil
}

func artifactPath(dir, artifact string) (string, error) {
	if artifact == "" || !strings.EqualFold(filepath.Ext(artifact), ".apk") {
		return "", fmt.Errorf("Flutter artifact must name an APK file")
	}
	path, err := paths.Within(dir, artifact)
	if err != nil {
		return "", err
	}
	for probe := path; probe != filepath.Clean(dir); probe = filepath.Dir(probe) {
		st, err := os.Lstat(probe)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("Flutter APK path contains a symlink: %s", probe)
		}
	}
	return path, nil
}

func (a Adapter) Build(ctx context.Context, sourceRoot, projectDir string, argv []string, artifact string, timeout time.Duration) (Result, error) {
	var out Result
	if len(argv) == 0 || !validExecutable(argv[0]) {
		return out, fmt.Errorf("Flutter build requires argv with a PATH executable or absolute host path")
	}
	for _, arg := range argv {
		if strings.ContainsRune(arg, '\x00') {
			return out, fmt.Errorf("Flutter build argv contains NUL")
		}
	}
	out.Executable = argv[0]
	var err error
	out.Directory, err = a.Validate(sourceRoot, projectDir, artifact)
	if err != nil {
		return out, err
	}
	out.Version, err = a.Doctor(ctx, argv[0])
	if err != nil {
		return out, err
	}
	if timeout <= 0 {
		timeout = 20 * time.Minute
	}
	r, err := a.runner().Run(ctx, execx.Command{Name: argv[0], Args: append([]string(nil), argv[1:]...), Dir: out.Directory, Timeout: timeout})
	out.Stdout, out.Stderr = r.Stdout, r.Stderr
	if err != nil {
		return out, fmt.Errorf("Flutter build: %w", err)
	}
	// Builds may create or replace path components; inspect confinement again.
	if _, err := a.Validate(sourceRoot, projectDir, artifact); err != nil {
		return out, err
	}
	out.ArtifactPath, err = artifactPath(out.Directory, artifact)
	if err != nil {
		return out, err
	}
	st, err := os.Lstat(out.ArtifactPath)
	if err != nil {
		return out, fmt.Errorf("Flutter APK: %w", err)
	}
	if !st.Mode().IsRegular() {
		return out, fmt.Errorf("Flutter APK is not a regular file")
	}
	f, err := os.Open(out.ArtifactPath)
	if err != nil {
		return out, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return out, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(st, opened) {
		return out, fmt.Errorf("Flutter APK changed while opening")
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return out, err
	}
	out.Digest = hex.EncodeToString(h.Sum(nil))
	return out, nil
}
