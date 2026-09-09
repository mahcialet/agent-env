package execx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrCrossOSExecution identifies a direct executable that crosses the worker's
// native process-ownership boundary. It does not certify scripts or descendants.
var ErrCrossOSExecution = errors.New("cross-OS process execution is unsupported")

// ValidateNativeExecutable checks an already resolved executable path. Relative
// paths are interpreted against the child directory, just as process startup
// does. Ordinary lookup/access errors remain the responsibility of os/exec.
func ValidateNativeExecutable(path, dir string) error {
	return validateNativeExecutable(path, dir, runtime.GOOS)
}

func validateNativeExecutable(path, dir, goos string) error {
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	if goos == "windows" {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		// Windows cannot account for Linux descendants behind this bridge using
		// its native process ownership and cleanup mechanisms.
		if strings.EqualFold(filepath.Base(path), "wsl.exe") {
			return fmt.Errorf("%w: run a separate Linux worker inside WSL instead of launching wsl.exe", ErrCrossOSExecution)
		}
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	var header [64]byte
	if _, err := f.ReadAt(header[:], 0); err != nil || string(header[:2]) != "MZ" {
		return nil
	}
	// PE's signature offset is a uint32. Read only the signature, never allocate
	// from untrusted executable metadata or scan the entire file.
	offset := int64(binary.LittleEndian.Uint32(header[60:64]))
	if offset < int64(len(header)) || offset > info.Size()-4 {
		return nil
	}
	var signature [4]byte
	if _, err := f.ReadAt(signature[:], offset); err == nil && string(signature[:]) == "PE\x00\x00" {
		return fmt.Errorf("%w: Windows executable on %s; use native tools and a worker in the same OS", ErrCrossOSExecution, goos)
	}
	return nil
}
