package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReleaseBinaryKnownModulePaths(t *testing.T) {
	for _, tc := range []struct {
		name, binary, root string
		want               bool
	}{
		{"module matches checkout basename", "github.com/mahcialet/agent-env/internal/cli.Run", "/agent-env", false},
		{"module matches temporary basename", "example.invalid/tmp/internal.Run", "/tmp", false},
		{"concatenated string pool", "not support deadline/agent-env/internal/leaked-source.gomethod ABI and", "/agent-env", true},
		{"absolute source", "/agent-env/internal/cli.go", "/agent-env", true},
		{"source after nul", "prefix\x00/agent-env/internal/cli.go", "/agent-env", true},
		{"source after assignment", "path=/agent-env/internal/cli.go", "/agent-env", true},
		{"quoted source", "\"/agent-env/internal/cli.go\"", "/agent-env", true},
		{"temp path", "\x00/tmp/build/main.go", "/tmp", true},
		{"windows source", "\x00C:\\agent-env\\internal\\cli.go", "C:\\agent-env", filepath.Separator == '\\'},
		{"windows slash source", "\x00C:/agent-env/internal/cli.go", "C:/agent-env", true},
		{"later genuine path", "github.com/mahcialet/agent-env/internal/cli.Run\x00/agent-env/internal/cli.go", "/agent-env", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := releaseBinaryContainsPathWithModules([]byte(tc.binary), tc.root, []string{"github.com/mahcialet/agent-env", "example.invalid/tmp"}); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReleaseBinaryDetectsConcatenatedPathLiteral(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/fixture\n\ngo 1.26.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"/agent-env/internal/leaked-source.go\") }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(t.TempDir(), "fixture.exe")
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !releaseBinaryContainsPath(binary, "/agent-env") {
		t.Fatal("real binary source path literal escaped detection")
	}
}

func TestReleaseBinaryShortCheckoutPaths(t *testing.T) {
	for _, root := range []string{"/a", "/ab", "/abc"} {
		t.Run(root, func(t *testing.T) {
			if !releaseBinaryContainsPath([]byte("prefix"+root+"/private.go"), root) {
				t.Fatal("short checkout leak accepted")
			}
			module := "example.invalid" + root
			if releaseBinaryContainsPathWithModules([]byte(module+"/internal.Run"), root, []string{module}) {
				t.Fatal("module identity treated as a host path")
			}
		})
	}
}
