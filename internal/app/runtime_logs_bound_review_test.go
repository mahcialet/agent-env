package app

import (
	"context"
	"fmt"
	"github.com/mahcialet/agent-env/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeLogEntriesBoundsRetainedFilesAndAggregate(t *testing.T) {
	for _, sizes := range [][]int{{(2 << 20) + 1}, {(1 << 20) + 1, (1 << 20) + 1}} {
		t.Run(fmt.Sprint(sizes), func(t *testing.T) {
			s, _, lease := commandFixture(t)
			ctx := context.Background()
			for i, size := range sizes {
				if err := s.saveTextArtifact(ctx, lease, "compose-log/runtime/service", fmt.Sprintf("logs%d.txt", i), strings.Repeat("x", size)); err != nil {
					t.Fatal(err)
				}
			}
			lease.Observed = "released"
			entries, err := s.RuntimeLogEntries(ctx, lease, "")
			if err == nil || !strings.Contains(err.Error(), "exceeds 2 MiB") || entries != nil {
				t.Fatalf("unbounded retained logs accepted: entries=%d error=%v", len(entries), err)
			}
		})
	}
}

type unboundedLogTrap struct {
	RuntimeProvider
	t *testing.T
}

func (p unboundedLogTrap) Logs(context.Context, domain.Runtime) (string, error) {
	p.t.Fatal("display reached unlimited runtime Logs")
	return "", nil
}
func TestRuntimeLogDisplayRefusesUnboundedProviderFallback(t *testing.T) {
	s, _, lease := commandFixture(t)
	s.Runtime = unboundedLogTrap{t: t}
	lease.Runtimes = []domain.Runtime{{Name: "compose", Type: "compose"}}
	if _, err := s.RuntimeLogEntries(context.Background(), lease, ""); err == nil || !strings.Contains(err.Error(), "bounded runtime log provider") {
		t.Fatalf("unbounded provider accepted: %v", err)
	}
}
func TestRuntimeLogDisplayBoundsAndroidFilesBeforeReading(t *testing.T) {
	s, _, lease := commandFixture(t)
	runtime := domain.Runtime{Name: "emulator", Type: "android-emulator"}
	directory := filepath.Join(s.Home, "leases", lease.ID, "android", runtime.Name)
	runtime.Directory = directory
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	large := strings.Repeat("x", (2<<20)+1)
	if err := os.WriteFile(filepath.Join(directory, "emulator.stdout.log"), []byte(large), 0600); err != nil {
		t.Fatal(err)
	}
	lease.Runtimes = []domain.Runtime{runtime}
	if entries, err := s.RuntimeLogEntries(context.Background(), lease, ""); err == nil || !strings.Contains(err.Error(), "Android log response exceeds") || entries != nil {
		t.Fatalf("unbounded Android display: entries=%d error=%v", len(entries), err)
	}
	legacy, err := AndroidProcessLogEntries(s.Home, lease.ID, runtime)
	if err != nil || legacy["emulator/emulator.stdout.log"] != large {
		t.Fatalf("legacy Android evidence reader changed: %v", err)
	}
}
