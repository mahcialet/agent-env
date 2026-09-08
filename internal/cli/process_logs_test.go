package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
)

type processLogProvider struct {
	app.PersistentProcessProvider
	names []string
}

func (p *processLogProvider) Logs(_ context.Context, r domain.Runtime) (string, error) {
	p.names = append(p.names, r.Name)
	return "process:" + r.Name, nil
}
func TestProcessComponentLogsRouteAndRetainIsolatedArtifacts(t *testing.T) {
	lease := logLease()
	lease.Components = append(lease.Components, domain.Component{Name: "browser", Runtime: "browser"}, domain.Component{Name: "server", Runtime: "server"})
	lease.Runtimes = append(lease.Runtimes, domain.Runtime{Name: "browser", Type: "process"}, domain.Runtime{Name: "server", Type: "process"})
	native := &processLogProvider{}
	compose := &logRuntime{}
	service := &app.Service{Store: logStore{}, Runtime: compose, Process: native}
	logs, err := runtimeLogEntries(context.Background(), service, lease, "browser")
	if err != nil || len(logs) != 1 || logs["browser"] != "process:browser" || len(compose.calls) != 0 || len(native.names) != 1 {
		t.Fatalf("provider/component scope: %v %v", logs, err)
	}
	var artifacts []domain.Artifact
	for _, name := range []string{"browser", "server"} {
		path := filepath.Join(t.TempDir(), name+".log")
		if err := os.WriteFile(path, []byte("retained:"+name), 0600); err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, domain.Artifact{ID: name, Kind: "process-log/" + name, Path: path})
	}
	lease.Desired, lease.Observed = "released", "released"
	service.Store = logStore{artifacts: artifacts}
	service.Process = nil
	logs, err = runtimeLogEntries(context.Background(), service, lease, "browser")
	if err != nil || len(logs) != 1 || logs["browser"] != "retained:browser" {
		t.Fatalf("retained scope: %v %v", logs, err)
	}
	logs, err = runtimeLogEntries(context.Background(), service, lease, "")
	if err != nil || len(logs) != 2 {
		t.Fatalf("aggregate retained: %v %v", logs, err)
	}
}
