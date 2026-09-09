package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

// Process output survives writable AVD cleanup. Read only the fixed diagnostics
// beneath the allocated runtime; never follow a replaced path into other state.
func AndroidProcessLogEntries(home, leaseID string, runtime domain.Runtime) (map[string]string, error) {
	return androidProcessLogEntries(home, leaseID, runtime, -1)
}

// A nonnegative budget is for display. The legacy evidence reader is retained
// for callers whose durable cleanup policy requires complete captured files.
func androidProcessLogEntries(home, leaseID string, runtime domain.Runtime, remaining int64) (map[string]string, error) {
	entries := map[string]string{}
	for _, part := range []string{leaseID, runtime.Name} {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\\:`) {
			return nil, errors.New("invalid Android log identity")
		}
	}
	relative := filepath.Join("leases", leaseID, "android", runtime.Name)
	if home == "" || filepath.Clean(runtime.Directory) != filepath.Join(home, relative) {
		return nil, errors.New("Android log directory differs from the lease allocation")
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	current := ""
	for _, part := range []string{"leases", leaseID, "android", runtime.Name} {
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if os.IsNotExist(err) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe Android log directory: %s", current)
		}
	}
	logs, err := root.OpenRoot(relative)
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	for _, name := range []string{"emulator.stdout.log", "emulator.stderr.log", "adb-server.stdout.log", "adb-server.stderr.log"} {
		info, err := logs.Lstat(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("Android log is not a regular file: %s", name)
		}
		var data []byte
		if remaining < 0 {
			data, err = logs.ReadFile(name)
		} else {
			if info.Size() > remaining {
				return nil, errors.New("Android log response exceeds the remaining display byte limit")
			}
			file, openErr := logs.Open(name)
			if openErr != nil {
				return nil, openErr
			}
			data, err = io.ReadAll(io.LimitReader(file, remaining+1))
			err = errors.Join(err, file.Close())
			if int64(len(data)) > remaining {
				return nil, errors.New("Android log response exceeds the remaining display byte limit")
			}
			remaining -= int64(len(data))
		}
		if err != nil {
			return nil, err
		}
		entries[runtime.Name+"/"+name] = evidence.RedactString(string(data), evidence.InheritedSecrets())
	}
	return entries, nil
}
