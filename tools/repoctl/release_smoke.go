package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func smokeRelease(root, dir, version, tag, commit string, mt time.Time, out io.Writer) error {
	m, e := checkRelease(root, dir, version, tag, commit, mt)
	if e != nil {
		return e
	}
	t := releaseTarget{runtime.GOOS, runtime.GOARCH}
	prefix, name, exe := archiveNames(version, t)
	files, e := readReleaseArchive(filepath.Join(dir, name), prefix, t.GOOS == "windows", mt)
	if e != nil {
		return e
	}
	var identity releaseIdentity
	for _, a := range m.Artifacts {
		if a.releaseTarget == t {
			identity = a.BuildInfo
		}
	}
	return smokeReleaseFiles(files, exe, identity, out)
}
func smokeReleaseFiles(files map[string][]byte, exe string, identity releaseIdentity, out io.Writer) error {
	base, e := os.MkdirTemp("", "agent-env smoke 日本語 ")
	if e != nil {
		return e
	}
	defer os.RemoveAll(base)
	bindir := filepath.Join(base, "unpacked 配布")
	cwd := filepath.Join(base, "outside source 作業")
	state := filepath.Join(base, "state 状態")
	home := filepath.Join(base, "default home")
	emptyPath := filepath.Join(base, "empty PATH")
	for _, p := range []string{bindir, cwd, home, emptyPath} {
		if e = os.MkdirAll(p, 0700); e != nil {
			return e
		}
	}
	for n, b := range files {
		mode := os.FileMode(0644)
		if n == exe {
			mode = 0755
		}
		if e = os.WriteFile(filepath.Join(bindir, n), b, mode); e != nil {
			return e
		}
	}
	env := []string{}
	for _, s := range os.Environ() {
		k, _, _ := strings.Cut(s, "=")
		switch strings.ToUpper(k) {
		case "PATH", "AGENT_ENV_HOME", "HOME", "USERPROFILE", "LOCALAPPDATA", "XDG_STATE_HOME":
			continue
		}
		env = append(env, s)
	}
	env = append(env, "PATH="+emptyPath, "AGENT_ENV_HOME="+state, "HOME="+home, "USERPROFILE="+home, "LOCALAPPDATA="+home, "XDG_STATE_HOME="+home)
	run := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, filepath.Join(bindir, exe), args...)
		cmd.Dir = cwd
		cmd.Env = env
		var errOut bytes.Buffer
		cmd.Stderr = &errOut
		b, e := cmd.Output()
		if e != nil {
			return nil, fmt.Errorf("smoke %v: %w: %s", args, e, errOut.String())
		}
		return b, nil
	}
	b, e := run("version", "--output", "json")
	if e != nil {
		return e
	}
	var envelope struct {
		SchemaVersion int             `json:"schema_version"`
		Data          releaseIdentity `json:"data"`
	}
	if e = json.Unmarshal(b, &envelope); e != nil {
		return e
	}
	if envelope.SchemaVersion != 1 || !reflect.DeepEqual(envelope.Data, identity) {
		return fmt.Errorf("native version identity mismatch: %s", b)
	}
	if _, e = run("--help"); e != nil {
		return e
	}
	if _, e = os.Stat(state); !os.IsNotExist(e) {
		return fmt.Errorf("version/help unexpectedly initialized state")
	}
	if _, e = run("list", "--output", "json"); e != nil {
		return e
	}
	if st, e := os.Stat(state); e != nil || !st.IsDir() {
		return fmt.Errorf("list did not create overridden state")
	}
	for _, p := range []string{cwd, home} {
		entries, e := os.ReadDir(p)
		if e != nil {
			return e
		}
		if len(entries) != 0 {
			return fmt.Errorf("state leaked into %s", p)
		}
	}
	entries, e := os.ReadDir(bindir)
	if e != nil {
		return e
	}
	if len(entries) != len(files) {
		return fmt.Errorf("state leaked beside binary")
	}
	for n, want := range files {
		got, e := os.ReadFile(filepath.Join(bindir, n))
		if e != nil {
			return e
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("distribution file changed")
		}
	}
	fmt.Fprintf(out, "native smoke passed: %s/%s; version/help/list outside source with empty PATH and Unicode state root\n", identity.GOOS, identity.GOARCH)
	return nil
}
