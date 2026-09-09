package execx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func platformCommand(ctx context.Context, spec Command) (*exec.Cmd, error) {
	name := spec.Name
	// Windows LookPath probes explicit relative names immediately, before
	// Cmd.Dir applies. Resolve those names against the requested child cwd;
	// bare program names retain their existing PATH lookup semantics.
	if spec.Dir != "" && !filepath.IsAbs(name) && filepath.VolumeName(name) == "" && !strings.HasPrefix(name, `/`) && !strings.HasPrefix(name, `\`) && strings.ContainsAny(name, `/\`) {
		var err error
		name, err = filepath.Abs(filepath.Join(spec.Dir, name))
		if err != nil {
			return nil, err
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return exec.CommandContext(ctx, path, spec.Args...), nil
	}
	// Batch files require cmd.exe; no user supplied shell program is accepted.
	// A wrapper's %* expansion is a second parsing pass, so protect both passes.
	for _, s := range append([]string{path}, spec.Args...) {
		if strings.ContainsAny(s, "\x00\r\n") {
			return nil, fmt.Errorf("batch arguments cannot contain NUL or line breaks")
		}
	}
	parts := []string{escapeMeta(path)}
	for _, arg := range spec.Args {
		parts = append(parts, escapeMeta(escapeMeta(windowsQuote(arg))))
	}
	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	cmd := exec.CommandContext(ctx, comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: windowsQuote(comspec) + ` /d /s /v:off /c "` + strings.Join(parts, " ") + `"`}
	return cmd, nil
}

func windowsQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	slash := 0
	for _, c := range s {
		if c == '\\' {
			slash++
			continue
		}
		if c == '"' {
			b.WriteString(strings.Repeat("\\", slash*2+1))
		} else {
			b.WriteString(strings.Repeat("\\", slash))
		}
		slash = 0
		b.WriteRune(c)
	}
	b.WriteString(strings.Repeat("\\", slash*2))
	b.WriteByte('"')
	return b.String()
}
func escapeMeta(s string) string {
	var b strings.Builder
	for _, c := range s {
		if strings.ContainsRune("()[]%!^\"`<>&|;, *?", c) {
			b.WriteByte('^')
		}
		b.WriteRune(c)
	}
	return b.String()
}
