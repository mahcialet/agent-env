package flutter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

// The test executable is also a portable Flutter stand-in: no shell scripts or
// interpreter dependency are needed to exercise the native argv boundary.
func init() {
	if os.Getenv("AGENT_ENV_FLUTTER_TEST_HELPER") != "1" {
		return
	}
	if reflect.DeepEqual(os.Args[1:], []string{"--version", "--machine"}) {
		fmt.Print(`{"frameworkVersion":"native-fixture"}`)
		os.Exit(0)
	}
	if !reflect.DeepEqual(os.Args[1:], []string{"build", "apk", "literal space;$(no-shell)"}) {
		os.Exit(2)
	}
	if _, err := os.Stat("pubspec.yaml"); err != nil {
		os.Exit(3)
	}
	if err := os.WriteFile("native.apk", []byte("native build"), 0600); err != nil {
		os.Exit(4)
	}
	fmt.Print("native output")
	os.Exit(0)
}

func TestNativeBuildPreservesArgvAndProjectDirectory(t *testing.T) {
	t.Setenv("AGENT_ENV_FLUTTER_TEST_HELPER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := project(t)
	r, err := (Adapter{}).Build(context.Background(), root, "app space", []string{executable, "build", "apk", "literal space;$(no-shell)"}, "native.apk", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Stdout != "native output" || r.Version != "native-fixture" || len(r.Digest) != 64 {
		t.Fatalf("native evidence: %+v", r)
	}
}

type runnerFunc func(context.Context, execx.Command) (execx.Result, error)

func (f runnerFunc) Run(ctx context.Context, c execx.Command) (execx.Result, error) { return f(ctx, c) }

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app space", "android"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app space", "pubspec.yaml"), []byte("name: fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestBuildArgvDirectoryAndEvidence(t *testing.T) {
	root := project(t)
	var commands []execx.Command
	argv := []string{"flutter", "build", "apk", "--dart-define=URL=hello world;$(untouched)"}
	data := []byte("APK fixture bytes")
	a := Adapter{Runner: runnerFunc(func(_ context.Context, c execx.Command) (execx.Result, error) {
		commands = append(commands, c)
		if len(commands) == 1 {
			return execx.Result{Stdout: `{"frameworkVersion":"3.44.0","channel":"stable"}`}, nil
		}
		if err := os.WriteFile(filepath.Join(c.Dir, "result.apk"), data, 0600); err != nil {
			return execx.Result{}, err
		}
		return execx.Result{Stdout: "build output", Stderr: "build diagnostics"}, nil
	})}
	r, err := a.Build(context.Background(), root, "app space", argv, "result.apk", 3*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 2 || !reflect.DeepEqual(commands[0].Args, []string{"--version", "--machine"}) || !reflect.DeepEqual(commands[1].Args, argv[1:]) || commands[1].Dir != filepath.Join(root, "app space") || commands[1].Timeout != 3*time.Minute {
		t.Fatalf("commands %+v", commands)
	}
	h := sha256.Sum256(data)
	if r.Digest != hex.EncodeToString(h[:]) || r.Version != "3.44.0" || r.Stdout != "build output" || r.Stderr != "build diagnostics" || r.ArtifactPath != filepath.Join(root, "app space", "result.apk") {
		t.Fatalf("result %+v", r)
	}
}

func TestDoctorFailsMissingMalformedAndEmptyVersion(t *testing.T) {
	for _, output := range []string{"", "Flutter 3.44", `{}`, `{"frameworkVersion":" "}`, `{"frameworkVersion":42}`, `{"frameworkVersion":"3"} trailing`} {
		t.Run(output, func(t *testing.T) {
			a := Adapter{Runner: runnerFunc(func(_ context.Context, c execx.Command) (execx.Result, error) {
				if c.Timeout <= 0 {
					t.Fatal("unbounded doctor")
				}
				return execx.Result{Stdout: output}, nil
			})}
			if _, err := a.Doctor(context.Background(), "flutter"); err == nil {
				t.Fatal("invalid version accepted")
			}
		})
	}
	want := errors.New("tool not found")
	a := Adapter{Runner: runnerFunc(func(context.Context, execx.Command) (execx.Result, error) { return execx.Result{}, want })}
	if _, err := a.Doctor(context.Background(), "flutter"); !errors.Is(err, want) {
		t.Fatalf("missing executable: %v", err)
	}
}

func TestBuildFailureRetainsLogsAndCancellation(t *testing.T) {
	root := project(t)
	calls := 0
	a := Adapter{Runner: runnerFunc(func(ctx context.Context, c execx.Command) (execx.Result, error) {
		calls++
		if calls == 1 {
			return execx.Result{Stdout: `{"frameworkVersion":"3"}`}, nil
		}
		if c.Timeout <= 0 {
			t.Fatal("unbounded default build")
		}
		return execx.Result{Stdout: "partial output", Stderr: "failed"}, context.Canceled
	})}
	r, err := a.Build(context.Background(), root, "app space", []string{"flutter", "build", "apk"}, "out.apk", 0)
	if !errors.Is(err, context.Canceled) || r.Stdout != "partial output" || r.Stderr != "failed" || r.Digest != "" {
		t.Fatalf("failure evidence %+v %v", r, err)
	}
}

func TestInvalidPrerequisitesDoNotRunCommands(t *testing.T) {
	for _, mode := range []string{"missing-project", "missing-pubspec", "missing-android", "directory-pubspec", "escape-project", "escape-artifact", "absolute-artifact", "backslash-artifact", "not-apk", "relative-executable"} {
		t.Run(mode, func(t *testing.T) {
			root := project(t)
			dir, apk := "app space", "result.apk"
			argv := []string{"flutter", "build", "apk"}
			switch mode {
			case "missing-project":
				dir = "absent"
			case "missing-pubspec", "directory-pubspec":
				p := filepath.Join(root, dir, "pubspec.yaml")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if mode == "directory-pubspec" {
					if err := os.Mkdir(p, 0700); err != nil {
						t.Fatal(err)
					}
				}
			case "missing-android":
				if err := os.Remove(filepath.Join(root, dir, "android")); err != nil {
					t.Fatal(err)
				}
			case "escape-project":
				dir = "../outside"
			case "escape-artifact":
				apk = "../outside.apk"
			case "absolute-artifact":
				apk = filepath.Join(root, "outside.apk")
			case "backslash-artifact":
				apk = `build\result.apk`
			case "not-apk":
				apk = "result.zip"
			case "relative-executable":
				argv[0] = "tools/flutter"
			}
			a := Adapter{Runner: runnerFunc(func(context.Context, execx.Command) (execx.Result, error) {
				t.Fatal("command ran for invalid prerequisite")
				return execx.Result{}, nil
			})}
			if _, err := a.Build(context.Background(), root, dir, argv, apk, time.Minute); err == nil {
				t.Fatal("invalid prerequisite accepted")
			}
		})
	}
}

func TestBuildRejectsMissingAndNonregularArtifact(t *testing.T) {
	for _, mode := range []string{"missing", "directory"} {
		t.Run(mode, func(t *testing.T) {
			root := project(t)
			a := Adapter{Runner: runnerFunc(func(_ context.Context, c execx.Command) (execx.Result, error) {
				if c.Dir == "" {
					return execx.Result{Stdout: `{"frameworkVersion":"3"}`}, nil
				}
				if mode == "directory" {
					if err := os.Mkdir(filepath.Join(c.Dir, "result.apk"), 0700); err != nil {
						return execx.Result{}, err
					}
				}
				return execx.Result{Stdout: "built"}, nil
			})}
			r, err := a.Build(context.Background(), root, "app space", []string{"flutter", "build", "apk"}, "result.apk", time.Minute)
			if err == nil || r.Digest != "" || r.Stdout != "built" {
				t.Fatalf("invalid artifact result %+v %v", r, err)
			}
		})
	}
}

func TestBuildRejectsSymlinkArtifactsAndLatePathEscape(t *testing.T) {
	for _, mode := range []string{"artifact-internal", "artifact-external", "parent-internal", "parent-external", "project-external"} {
		t.Run(mode, func(t *testing.T) {
			root := project(t)
			outside := t.TempDir()
			target := filepath.Join(root, "app space", "target")
			if strings.Contains(mode, "external") {
				target = outside
			}
			if err := os.MkdirAll(target, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "result.apk"), []byte("other apk"), 0600); err != nil {
				t.Fatal(err)
			}
			// Check symlink support before invoking the fake build. Windows may lack
			// developer mode; ordinary path/argv validation still runs natively there.
			probe := filepath.Join(root, "probe")
			if err := os.Symlink(target, probe); err != nil {
				t.Skipf("host cannot create symlink: %v", err)
			}
			if err := os.Remove(probe); err != nil {
				t.Fatal(err)
			}
			apk := "result.apk"
			if strings.HasPrefix(mode, "parent") {
				apk = "build/result.apk"
			}
			a := Adapter{Runner: runnerFunc(func(_ context.Context, c execx.Command) (execx.Result, error) {
				if c.Dir == "" {
					return execx.Result{Stdout: `{"frameworkVersion":"3"}`}, nil
				}
				var err error
				switch {
				case mode == "project-external":
					err = os.Rename(c.Dir, c.Dir+"-old")
					if err == nil {
						err = os.Symlink(outside, c.Dir)
					}
				case strings.HasPrefix(mode, "parent"):
					err = os.Symlink(target, filepath.Join(c.Dir, "build"))
				default:
					err = os.Symlink(filepath.Join(target, "result.apk"), filepath.Join(c.Dir, "result.apk"))
				}
				return execx.Result{}, err
			})}
			if _, err := a.Build(context.Background(), root, "app space", []string{"flutter", "build", "apk"}, apk, time.Minute); err == nil {
				t.Fatal("build-created symlink accepted")
			}
		})
	}
}

func TestBuildRejectsInternalProjectSymlinks(t *testing.T) {
	for _, mode := range []string{"directory", "ancestor", "source-root"} {
		for _, late := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/late=%t", mode, late), func(t *testing.T) {
				root := project(t)
				dir := "app space"
				link := filepath.Join(root, dir)
				if mode == "ancestor" {
					if err := os.Mkdir(filepath.Join(root, "parent"), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(link, filepath.Join(root, "parent", dir)); err != nil {
						t.Fatal(err)
					}
					link = filepath.Join(root, "parent")
					dir = "parent/app space"
				} else if mode == "source-root" {
					root = filepath.Join(root, dir)
					dir = "."
					link = root
				}
				probe := filepath.Join(t.TempDir(), "probe")
				if err := os.Symlink(link, probe); err != nil {
					t.Skipf("host cannot create symlink: %v", err)
				}
				if err := os.Remove(probe); err != nil {
					t.Fatal(err)
				}
				replace := func() {
					t.Helper()
					if err := os.Rename(link, link+"-real"); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(link+"-real", link); err != nil {
						t.Fatal(err)
					}
				}
				if !late {
					replace()
				}
				calls := 0
				a := Adapter{Runner: runnerFunc(func(_ context.Context, c execx.Command) (execx.Result, error) {
					calls++
					if c.Dir == "" {
						return execx.Result{Stdout: `{"frameworkVersion":"3"}`}, nil
					}
					if late {
						replace()
					}
					return execx.Result{}, os.WriteFile(filepath.Join(c.Dir, "result.apk"), []byte("valid APK"), 0600)
				})}
				result, err := a.Build(context.Background(), root, dir, []string{"flutter", "build", "apk"}, "result.apk", time.Minute)
				if err == nil || result.Digest != "" {
					t.Fatalf("project symlink accepted: %+v %v", result, err)
				}
				if !late && calls != 0 {
					t.Fatalf("invalid project ran %d commands", calls)
				}
				if late && calls != 2 {
					t.Fatalf("late validation did not exercise build: %d commands", calls)
				}
			})
		}
	}
}
