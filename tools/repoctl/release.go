package main

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

type releaseTarget struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

var releaseTargets = []releaseTarget{{"windows", "amd64"}, {"windows", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}

type releaseIdentity struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Dirty     string `json:"dirty"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}
type releaseAsset struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}
type releaseArtifact struct {
	releaseTarget
	Archive          string          `json:"archive"`
	SHA256           string          `json:"sha256"`
	Executable       string          `json:"executable"`
	ExecutableSHA256 string          `json:"executable_sha256"`
	BuildInfo        releaseIdentity `json:"build_info"`
	Assets           []releaseAsset  `json:"assets"`
}
type releaseManifest struct {
	SchemaVersion   int               `json:"schema_version"`
	Product         string            `json:"product"`
	Version         string            `json:"version"`
	Tag             string            `json:"tag"`
	Commit          string            `json:"commit"`
	SourceTimestamp int64             `json:"source_timestamp"`
	GoVersion       string            `json:"go_version"`
	Artifacts       []releaseArtifact `json:"artifacts"`
}

func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func archiveNames(version string, t releaseTarget) (prefix, name, exe string) {
	prefix = "agent-env_v" + version + "_" + t.GOOS + "_" + t.GOARCH
	name = prefix + ".tar.gz"
	exe = "agent-env"
	if t.GOOS == "windows" {
		name = prefix + ".zip"
		exe += ".exe"
	}
	return
}
func releaseFlags(version, commit string) string {
	const pkg = "github.com/mahcialet/agent-env/internal/buildinfo."
	return "-s -w -X " + pkg + "ReleaseRecord=agent-env-release-v1|" + version + "|" + commit + "|false|end"
}

// This deterministic bilingual text is the README.txt source; no local paths.
func releaseReadme(version string) []byte {
	return []byte("agent-env " + version + "\n\nExtract the directory and run agent-env version or agent-env --help.\nNo Go installation or shell is required to run the executable.\nAGENT_ENV_HOME sets an absolute writable state directory.\nOptional capabilities require Git; Docker/Compose; Android SDK/Emulator;\nor Flutter/Java/Android build tools. These tools are not bundled.\n\nディレクトリを展開し、agent-env version または agent-env --help を実行します。\n実行に Go やシェルは不要です。AGENT_ENV_HOME で状態保存先の絶対パスを指定できます。\n機能に応じて Git、Docker/Compose、Android SDK/Emulator、Flutter/Java が別途必要です。\n")
}
func executeRelease(root string, args []string, out, errOut io.Writer) error {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(errOut)
	version := fs.String("version", "", "release version without v")
	selectedTag := fs.String("tag", "", "release tag (alternative to version)")
	tagEnv := fs.Bool("tag-env", false, "read tag from AGENT_ENV_RELEASE_TAG")
	var dir string
	if args[0] == "release-build" {
		fs.StringVar(&dir, "out", "", "new output directory")
	} else {
		fs.StringVar(&dir, "dir", "", "release directory")
	}
	if e := fs.Parse(args[1:]); e != nil {
		return e
	}
	if *tagEnv {
		if *selectedTag != "" {
			return fmt.Errorf("tag and tag-env are mutually exclusive")
		}
		*selectedTag = os.Getenv("AGENT_ENV_RELEASE_TAG")
		if *selectedTag == "" {
			return fmt.Errorf("missing AGENT_ENV_RELEASE_TAG")
		}
	}
	if *selectedTag != "" {
		if *version != "" || !releaseTag.MatchString(*selectedTag) {
			return fmt.Errorf("specify either valid tag or version")
		}
		*version = strings.TrimPrefix(*selectedTag, "v")
	}
	if fs.NArg() != 0 || *version == "" || dir == "" {
		return fmt.Errorf("AGENTENV-RELEASE-001: version and directory flags required")
	}
	dir, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	tag, mt, e := releaseVersion(root, *version)
	if e != nil {
		return e
	}
	commit, e := gitOut(root, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	switch args[0] {
	case "release-build":
		e = buildRelease(root, dir, *version, tag, commit, mt, out, errOut)
	case "release-repeat":
		e = repeatRelease(root, dir, *version, tag, commit, mt, out, errOut)
	case "release-check":
		_, e = checkRelease(root, dir, *version, tag, commit, mt)
	case "release-smoke":
		e = smokeRelease(root, dir, *version, tag, commit, mt, out)
	default:
		return fmt.Errorf("unknown release command")
	}
	if e == nil {
		fmt.Fprintln(out, args[0]+" passed")
	}
	return e
}
func buildRelease(root, dir, version, tag, commit string, mt time.Time, out, errOut io.Writer) error {
	if _, e := os.Lstat(dir); !os.IsNotExist(e) {
		return fmt.Errorf("output must not exist: %s", dir)
	}
	// Keep construction out of the source worktree until its final cleanliness
	// check. Destination siblings may be unignored Git inputs.
	stage, e := os.MkdirTemp("", "agent-env-release-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	payload, e := os.MkdirTemp("", "agent-env-build-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(payload)
	source, cleanup, e := privateReleaseSource(root, commit, tag)
	if e != nil {
		return e
	}
	defer cleanup()
	license, e := gitOut(source, "show", commit+":LICENSE")
	if e != nil {
		return e
	}
	licenseBytes := []byte(license + "\n")
	manifest := releaseManifest{SchemaVersion: 1, Product: "agent-env", Version: version, Tag: tag, Commit: commit, SourceTimestamp: mt.Unix()}
	for _, t := range releaseTargets {
		prefix, name, exe := archiveNames(version, t)
		bin := filepath.Join(payload, exe)
		cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=true", "-ldflags", releaseFlags(version, commit), "-o", bin, "./cmd/agent-env")
		cmd.Dir = source
		cmd.Env = releaseBuildEnv(t)
		cmd.Stdout = out
		cmd.Stderr = errOut
		fmt.Fprintln(out, "build", t.GOOS, t.GOARCH)
		if e = cmd.Run(); e != nil {
			return e
		}
		b, e := os.ReadFile(bin)
		if e != nil {
			return e
		}
		info, e := inspectReleaseBinary(b, version, commit, t)
		if e != nil {
			return e
		}
		if manifest.GoVersion == "" {
			manifest.GoVersion = info.GoVersion
		}
		if info.GoVersion != manifest.GoVersion {
			return fmt.Errorf("toolchain changed during build")
		}
		if e = os.WriteFile(filepath.Join(payload, "LICENSE"), licenseBytes, 0644); e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(payload, "README.txt"), releaseReadme(version), 0644); e != nil {
			return e
		}
		if e = writeArchive(filepath.Join(stage, name), payload, prefix, t.GOOS == "windows", mt); e != nil {
			return e
		}
		ar, e := os.ReadFile(filepath.Join(stage, name))
		if e != nil {
			return e
		}
		manifest.Artifacts = append(manifest.Artifacts, releaseArtifact{t, name, digest(ar), prefix + "/" + exe, digest(b), info, []releaseAsset{}})
		if e = os.Remove(bin); e != nil {
			return e
		}
	}
	data, e := json.MarshalIndent(manifest, "", "  ")
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(stage, "release-manifest.json"), append(data, '\n'), 0644); e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(stage, "checksums.txt"), releaseChecksums(manifest), 0644); e != nil {
		return e
	}
	// Recheck source after builds to catch edits during construction.
	finalTag, finalTime, e := releaseVersion(root, version)
	if e != nil {
		return e
	}
	finalCommit, e := gitOut(root, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	if finalTag != tag || !finalTime.Equal(mt) || finalCommit != commit {
		return fmt.Errorf("release source changed during build")
	}
	if _, e = checkRelease(source, stage, version, tag, commit, mt); e != nil {
		return e
	}
	if _, e = checkRelease(root, stage, version, tag, commit, mt); e != nil {
		return e
	}
	return publishReleaseDirectory(stage, dir)
}

// Transfer verified bytes only after source validation. A destination-local
// sibling retains same-filesystem rename even when os.TempDir is on another disk.
func publishReleaseDirectory(stage, dir string) error {
	if _, e := os.Lstat(dir); !os.IsNotExist(e) {
		return fmt.Errorf("output appeared during build")
	}
	parent := filepath.Dir(dir)
	if e := os.MkdirAll(parent, 0755); e != nil {
		return e
	}
	transfer, e := os.MkdirTemp(parent, ".agent-env-publish-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(transfer)
	candidate := filepath.Join(transfer, "candidate")
	if e = copyVerifiedRelease(stage, candidate); e != nil {
		return e
	}
	if _, e = os.Lstat(dir); !os.IsNotExist(e) {
		return fmt.Errorf("output appeared during publication")
	}
	return os.Rename(candidate, dir)
}
func releaseBuildEnv(t releaseTarget) []string {
	controlled := map[string]string{"CGO_ENABLED": "0", "GOOS": t.GOOS, "GOARCH": t.GOARCH, "GOFLAGS": "", "GOENV": "off", "GOWORK": "off", "GOEXPERIMENT": "", "GOAMD64": "v1", "GOARM64": "v8.0", "GIT_NO_REPLACE_OBJECTS": "1"}
	env := []string{}
	for _, s := range releaseGitEnv() {
		k, _, _ := strings.Cut(s, "=")
		if _, ok := controlled[strings.ToUpper(k)]; !ok {
			env = append(env, s)
		}
	}
	keys := []string{}
	for k := range controlled {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, k+"="+controlled[k])
	}
	return env
}
func releaseChecksums(m releaseManifest) []byte {
	lines := []string{}
	artifacts := append([]releaseArtifact(nil), m.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Archive < artifacts[j].Archive })
	for _, a := range artifacts {
		lines = append(lines, a.SHA256+"  "+a.Archive+"\n")
	}
	return []byte(strings.Join(lines, ""))
}
func inspectReleaseBinary(b []byte, version, commit string, t releaseTarget) (releaseIdentity, error) {
	info, e := buildinfo.Read(bytes.NewReader(b))
	if e != nil {
		return releaseIdentity{}, e
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	expected := map[string]string{"GOOS": t.GOOS, "GOARCH": t.GOARCH, "CGO_ENABLED": "0", "-trimpath": "true", "vcs.revision": commit, "vcs.modified": "false"}
	for k, v := range expected {
		if settings[k] != v {
			return releaseIdentity{}, fmt.Errorf("binary %s mismatch: got %q expected %q", k, settings[k], v)
		}
	}
	if info.Path != "github.com/mahcialet/agent-env/cmd/agent-env" || !strings.HasPrefix(info.GoVersion, "go1.") {
		return releaseIdentity{}, fmt.Errorf("unexpected executable build identity")
	}
	records := regexp.MustCompile(`agent-env-release-v1\|[^|\x00]{1,100}\|[a-f0-9]{40,64}\|false\|end`).FindAll(b, -1)
	expectedRecord := []byte("agent-env-release-v1|" + version + "|" + commit + "|false|end")
	if len(records) != 1 || !bytes.Equal(records[0], expectedRecord) {
		return releaseIdentity{}, fmt.Errorf("static release record mismatch")
	}
	return releaseIdentity{version, commit, "false", info.GoVersion, t.GOOS, t.GOARCH}, nil
}
func regularRead(path string, limit int64) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("invalid release file %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readOpenedReleaseFile(f, limit)
}

type releaseFileReader interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

func readOpenedReleaseFile(f releaseFileReader, limit int64) ([]byte, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if limit < 0 || !st.Mode().IsRegular() || st.Size() > limit {
		return nil, fmt.Errorf("invalid release file size or type")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, err
	}
	var extra [1]byte
	if n, err := io.ReadFull(f, extra[:]); n != 0 {
		return nil, fmt.Errorf("release file exceeds read limit")
	} else if err != io.EOF {
		return nil, err
	}
	return data, nil
}
func checkRelease(root, dir, version, tag, commit string, mt time.Time) (releaseManifest, error) {
	var m releaseManifest
	b, e := regularRead(filepath.Join(dir, "release-manifest.json"), 1<<20)
	if e != nil {
		return m, e
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&m); e != nil {
		return m, e
	}
	canonical, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		return m, e
	}
	if !bytes.Equal(b, append(canonical, '\n')) {
		return m, fmt.Errorf("manifest must use canonical generated JSON")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return m, fmt.Errorf("trailing manifest data")
	}
	if m.SchemaVersion != 1 || m.Product != "agent-env" || m.Version != version || m.Tag != tag || m.Commit != commit || m.SourceTimestamp != mt.Unix() || len(m.Artifacts) != len(releaseTargets) {
		return m, fmt.Errorf("release manifest identity mismatch")
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return m, e
	}
	if len(entries) != 8 {
		return m, fmt.Errorf("expected exactly eight release files")
	}
	license, e := gitOut(root, "show", commit+":LICENSE")
	if e != nil {
		return m, e
	}
	for i, t := range releaseTargets {
		a := m.Artifacts[i]
		prefix, name, exe := archiveNames(version, t)
		if a.releaseTarget != t || a.Archive != name || a.Executable != prefix+"/"+exe || a.Assets == nil || len(a.Assets) != 0 {
			return m, fmt.Errorf("target/member/asset mismatch")
		}
		ar, e := regularRead(filepath.Join(dir, name), 256<<20)
		if e != nil {
			return m, e
		}
		if digest(ar) != a.SHA256 {
			return m, fmt.Errorf("archive checksum mismatch")
		}
		files, e := readReleaseArchive(filepath.Join(dir, name), prefix, t.GOOS == "windows", mt)
		if e != nil {
			return m, e
		}
		binary := files[exe]
		if digest(binary) != a.ExecutableSHA256 {
			return m, fmt.Errorf("executable checksum mismatch")
		}
		info, e := inspectReleaseBinary(binary, version, commit, t)
		if e != nil {
			return m, e
		}
		if !reflect.DeepEqual(info, a.BuildInfo) || info.GoVersion != m.GoVersion {
			return m, fmt.Errorf("build info mismatch")
		}
		if !bytes.Equal(files["LICENSE"], []byte(license+"\n")) || !bytes.Equal(files["README.txt"], releaseReadme(version)) {
			return m, fmt.Errorf("license/readme mismatch")
		}
		for _, p := range []string{root, os.TempDir()} {
			if releaseBinaryContainsPath(binary, p) {
				return m, fmt.Errorf("local path leaked in binary")
			}
		}
	}
	sums, e := regularRead(filepath.Join(dir, "checksums.txt"), 1<<20)
	if e != nil {
		return m, e
	}
	if !bytes.Equal(sums, releaseChecksums(m)) {
		return m, fmt.Errorf("checksums file mismatch")
	}
	return m, nil
}

func repeatRelease(root, dir, version, tag, commit string, mt time.Time, out, errOut io.Writer) error {
	if _, e := checkRelease(root, dir, version, tag, commit, mt); e != nil {
		return e
	}
	base, e := os.MkdirTemp("", "agent-env-repeat-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(base)
	next := filepath.Join(base, "candidate")
	if e = buildRelease(root, next, version, tag, commit, mt, out, errOut); e != nil {
		return e
	}
	return compareReleaseDirectories(dir, next)
}

// releaseBinaryContainsPath checks known host directory prefixes, exempting
// only their occurrences within module/import identities embedded by Go.
func releaseBinaryContainsPath(binary []byte, path string) bool {
	var modules []string
	if info, err := buildinfo.Read(bytes.NewReader(binary)); err == nil {
		modules = append(modules, info.Path, info.Main.Path)
		for _, dep := range info.Deps {
			modules = append(modules, dep.Path)
		}
	}
	return releaseBinaryContainsPathWithModules(binary, path, modules)
}

func releaseBinaryContainsPathWithModules(binary []byte, path string, modules []string) bool {
	if path == "" || filepath.Clean(path) == filepath.VolumeName(path)+string(filepath.Separator) {
		return false
	}
	for _, prefix := range []string{path + string(filepath.Separator), filepath.ToSlash(path) + "/"} {
		for offset := 0; offset < len(binary); {
			i := bytes.Index(binary[offset:], []byte(prefix))
			if i < 0 {
				break
			}
			i += offset
			knownModule := false
			for _, module := range modules {
				// Host paths can occur within a longer module path. Exempt only
				// the exact bytes proven to be part of that known identity.
				identity := module + "/"
				for from := 0; module != "" && from < len(identity); {
					within := strings.Index(identity[from:], prefix)
					if within < 0 {
						break
					}
					within += from
					start := i - within
					if start >= 0 && bytes.HasPrefix(binary[start:], []byte(identity)) {
						knownModule = true
						break
					}
					from = within + 1
				}
				if knownModule {
					break
				}
			}
			if !knownModule {
				return true
			}
			offset = i + 1
		}
	}
	return false
}
