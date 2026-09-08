package buildinfo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Build actual executables: go test binaries do not carry the VCS settings of
// a normal go build, so an in-process Current call cannot prove this fallback.
func TestCurrentEmbeddedVCSIdentity(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture, output := t.TempDir(), t.TempDir()
	copyFile := func(relative string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repoRoot, relative))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(fixture, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "internal/buildinfo/buildinfo.go"} {
		copyFile(name)
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "internal", "assets"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			copyFile(filepath.Join("internal", "assets", entry.Name()))
		}
	}
	main := []byte("package main\nimport (\"encoding/json\"; \"os\"; \"github.com/mahcialet/agent-env/internal/buildinfo\")\nfunc main() { json.NewEncoder(os.Stdout).Encode(buildinfo.Current()) }\n")
	if err := os.WriteFile(filepath.Join(fixture, "main.go"), main, 0644); err != nil {
		t.Fatal(err)
	}
	env := make([]string, 0, len(os.Environ()))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		key = strings.ToUpper(key)
		if strings.HasPrefix(key, "GIT_") || key == "GOFLAGS" || key == "GOWORK" || key == "GOOS" || key == "GOARCH" || key == "CGO_ENABLED" || key == "GOTOOLCHAIN" {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "GOFLAGS=", "GOWORK=off", "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	run := func(command string, args ...string) string {
		t.Helper()
		cmd := exec.Command(command, args...)
		cmd.Dir, cmd.Env = fixture, env
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", command, args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	run("git", "init", "--template=")
	run("git", "add", ".")
	run("git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--no-gpg-sign", "-m", "fixture")
	commit := run("git", "rev-parse", "HEAD")
	build := func(flags ...string) Info {
		t.Helper()
		binary := filepath.Join(output, "identity")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		args := append([]string{"build", "-o", binary}, flags...)
		run("go", append(args, ".")...)
		cmd := exec.Command(binary)
		cmd.Dir = output
		for _, value := range env {
			key, _, _ := strings.Cut(value, "=")
			if !strings.EqualFold(key, "PATH") {
				cmd.Env = append(cmd.Env, value)
			}
		}
		cmd.Env = append(cmd.Env, "PATH=")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run outside checkout with empty PATH: %v\n%s", err, data)
		}
		var got Info
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	assert := func(got Info, version, revision, dirty string) {
		t.Helper()
		if got.Version != version || got.Commit != revision || got.Dirty != dirty {
			t.Fatalf("identity = %+v, want version=%s commit=%s dirty=%s", got, version, revision, dirty)
		}
	}
	assert(build(), "devel", commit, "false")
	if err := os.WriteFile(filepath.Join(fixture, "main.go"), append(main, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	assert(build(), "devel", commit, "true")
	assert(build("-buildvcs=false"), "devel", "unknown", "unknown")
	assert(build("-ldflags=-X github.com/mahcialet/agent-env/internal/buildinfo.Version=custom -X github.com/mahcialet/agent-env/internal/buildinfo.Commit=explicit -X github.com/mahcialet/agent-env/internal/buildinfo.Dirty=false"), "custom", "explicit", "false")
	revision := strings.Repeat("a", 40)
	assert(build("-ldflags=-X github.com/mahcialet/agent-env/internal/buildinfo.ReleaseRecord="+ReleaseRecordPrefix+"1.2.3|"+revision+"|false|end"), "1.2.3", revision, "false")
}
