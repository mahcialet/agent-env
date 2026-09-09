package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/evidence"
	"github.com/mahcialet/agent-env/internal/remotesource"
)

type retainedPublication struct {
	write   func(string, []byte, os.FileMode) error
	prepare func(string) error
	rename  func(string, string) error
	confirm func(string, string, string, []byte) error
}

func (e *AppExecutor) packageRoot() (string, error) {
	root := filepath.Join(e.Home, "remote-lease-inputs")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	st, err := os.Lstat(root)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("unsafe retained input directory")
	}
	return root, nil
}
func (e *AppExecutor) retainPackage(id string, p remotesource.Package) error {
	return e.retainPackageWithPublication(id, p, nativeRetainedPublication())
}

func (e *AppExecutor) retainPackageWithPublication(id string, p remotesource.Package, publication retainedPublication) error {
	root, err := e.packageRoot()
	if err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err == nil {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		err = decoder.Decode(&value)
		if err == nil {
			data, err = json.Marshal(value)
		}
	}
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp(root, ".incoming-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if publication.write == nil {
		publication.write = evidence.AtomicWrite
	}
	if err = publication.write(filepath.Join(temp, "package.json"), data, 0600); err != nil {
		return err
	}
	if err = publication.prepare(temp); err != nil {
		return err
	}
	destination := filepath.Join(root, id)
	if err = publication.rename(temp, destination); err != nil {
		existing, e := e.loadPackage(id)
		if e != nil {
			return e
		}
		if !bytes.Equal(existing, data) {
			return errors.New("lease source package changed")
		}
		return publication.confirm(temp, destination, root, data)
	}
	return publication.confirm("", destination, root, data)
}
func (e *AppExecutor) loadPackage(id string) ([]byte, error) {
	root, err := e.packageRoot()
	if err != nil {
		return nil, err
	}
	return readOwned(root, id+"/package.json", 8<<20)
}
