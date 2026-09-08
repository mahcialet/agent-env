package uihelper

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/execx"
)

type BuildOptions struct{ SDK, JDK, Platform, BuildTools, Output string }

// Build explicitly invokes installed build tools; normal Go builds never call it.
func Build(ctx context.Context, o BuildOptions) (Metadata, error) {
	var empty Metadata
	if o.SDK == "" || o.JDK == "" || o.Output == "" {
		return empty, fmt.Errorf("SDK, JDK and output directory are required")
	}
	if !strings.HasPrefix(o.Platform, "android-") || strings.ContainsAny(o.Platform, "/\\") || o.BuildTools == "" || strings.ContainsAny(o.BuildTools, "/\\") || o.BuildTools == ".." {
		return empty, fmt.Errorf("invalid platform/build-tools version")
	}
	var err error
	o.SDK, err = filepath.Abs(o.SDK)
	if err != nil {
		return empty, err
	}
	o.JDK, err = filepath.Abs(o.JDK)
	if err != nil {
		return empty, err
	}
	o.Output, err = filepath.Abs(o.Output)
	if err != nil {
		return empty, err
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	java := filepath.Join(o.JDK, "bin", "java"+suffix)
	javac := filepath.Join(o.JDK, "bin", "javac"+suffix)
	keytool := filepath.Join(o.JDK, "bin", "keytool"+suffix)
	bt := filepath.Join(o.SDK, "build-tools", o.BuildTools)
	android := filepath.Join(o.SDK, "platforms", o.Platform, "android.jar")
	for _, p := range []string{java, javac, keytool, android, filepath.Join(bt, "aapt2"+suffix), filepath.Join(bt, "lib", "d8.jar"), filepath.Join(bt, "lib", "apksigner.jar")} {
		st, e := os.Stat(p)
		if e != nil || !st.Mode().IsRegular() {
			return empty, fmt.Errorf("required build input unavailable: %s", p)
		}
	}
	// Refuse overwriting existing release artifacts or user files.
	if err := os.Mkdir(o.Output, 0700); err != nil {
		return empty, err
	}
	work, err := os.MkdirTemp(o.Output, "build-")
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(work)
	for _, name := range []string{"Observer.java", "AndroidManifest.xml"} {
		b, _ := source.ReadFile("source/" + name)
		if err = os.WriteFile(filepath.Join(work, name), b, 0600); err != nil {
			return empty, err
		}
	}
	classes := filepath.Join(work, "classes")
	dex := filepath.Join(work, "dex")
	for _, d := range []string{classes, dex} {
		if err = os.Mkdir(d, 0700); err != nil {
			return empty, err
		}
	}
	run := func(name string, args ...string) error {
		_, e := (execx.OSRunner{}).Run(ctx, execx.Command{Name: name, Args: args, Dir: work, Timeout: 2 * time.Minute})
		return e
	}
	if err = run(javac, "-source", "8", "-target", "8", "-classpath", android, "-d", classes, filepath.Join(work, "Observer.java")); err != nil {
		return empty, err
	}
	args := []string{"-cp", filepath.Join(bt, "lib", "d8.jar"), "com.android.tools.r8.D8", "--lib", android, "--min-api", "26", "--output", dex}
	err = filepath.WalkDir(classes, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && strings.HasSuffix(p, ".class") {
			args = append(args, p)
		}
		return nil
	})
	if err != nil {
		return empty, err
	}
	if err = run(java, args...); err != nil {
		return empty, err
	}
	resources := filepath.Join(work, "resources.apk")
	if err = run(filepath.Join(bt, "aapt2"+suffix), "link", "-I", android, "--manifest", filepath.Join(work, "AndroidManifest.xml"), "-o", resources); err != nil {
		return empty, err
	}
	unsigned := filepath.Join(work, "unsigned.apk")
	if err = addDex(resources, filepath.Join(dex, "classes.dex"), unsigned); err != nil {
		return empty, err
	}
	var random [24]byte
	if _, err = rand.Read(random[:]); err != nil {
		return empty, err
	}
	password := hex.EncodeToString(random[:])
	passfile := filepath.Join(work, "password")
	if err = os.WriteFile(passfile, []byte(password), 0600); err != nil {
		return empty, err
	}
	key := filepath.Join(work, "signing.p12")
	if err = run(keytool, "-genkeypair", "-keystore", key, "-storepass:file", passfile, "-keypass:file", passfile, "-alias", "observer", "-keyalg", "RSA", "-validity", "3650", "-dname", "CN=Agent env observer local build"); err != nil {
		return empty, err
	}
	apk := filepath.Join(o.Output, "observer.apk")
	if err = run(java, "-jar", filepath.Join(bt, "lib", "apksigner.jar"), "sign", "--ks", key, "--ks-pass", "file:"+passfile, "--out", apk, unsigned); err != nil {
		return empty, err
	}
	if err = run(java, "-jar", filepath.Join(bt, "lib", "apksigner.jar"), "verify", apk); err != nil {
		return empty, err
	}
	if err = os.Chmod(apk, 0600); err != nil {
		return empty, err
	}
	b, err := regular(apk, MaxAPKBytes)
	if err != nil {
		return empty, err
	}
	sum := sha256.Sum256(b)
	m := Metadata{Version: Version, Package: Package, SourceSHA256: SourceDigest(), APKSHA256: hex.EncodeToString(sum[:]), Platform: o.Platform, BuildTools: o.BuildTools}
	encoded, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return empty, err
	}
	err = os.WriteFile(filepath.Join(o.Output, "observer.json"), append(encoded, '\n'), 0600)
	return m, err
}
func addDex(resources, dex, out string) (err error) {
	r, err := zip.OpenReader(resources)
	if err != nil {
		return err
	}
	defer r.Close()
	f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	w := zip.NewWriter(f)
	defer func() {
		if e := w.Close(); err == nil {
			err = e
		}
	}()
	for _, entry := range r.File {
		if err = w.Copy(entry); err != nil {
			return err
		}
	}
	dst, err := w.Create("classes.dex")
	if err != nil {
		return err
	}
	src, err := os.Open(dex)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(dst, src)
	return err
}
