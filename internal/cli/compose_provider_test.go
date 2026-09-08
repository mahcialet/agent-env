package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

type selectedDoctor struct {
	calls []domain.ComposeProviderName
	fail  bool
}

func (d *selectedDoctor) DoctorFor(_ context.Context, p domain.ComposeProviderName) (map[string]string, error) {
	d.calls = append(d.calls, p)
	if d.fail {
		return nil, errors.New("missing selected tool")
	}
	return map[string]string{"provider": string(p)}, nil
}
func TestComposeDoctorSelectsManifestProvidersWithoutFallback(t *testing.T) {
	root := t.TempDir()
	m := `version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  api: {type: compose, provider: podman-compose, source: self, project_directory: ., files: [compose.yaml]}
components:
  api: {runtime: api, compose_services: [api]}
stacks:
  review: {roots: [api]}
`
	if err := os.WriteFile(filepath.Join(root, ".agent-env.yaml"), []byte(m), 0600); err != nil {
		t.Fatal(err)
	}
	for _, fail := range []bool{false, true} {
		d := &selectedDoctor{fail: fail}
		report, err := doctorCompose(context.Background(), root, "", d)
		if (err != nil) != fail || !reflect.DeepEqual(d.calls, []domain.ComposeProviderName{domain.ComposeProviderPodman}) || len(report) != 1 {
			t.Fatalf("doctor selected wrong provider: %v %v %v", d.calls, report, err)
		}
	}
	d := &selectedDoctor{}
	if _, err := doctorCompose(context.Background(), "", "other", d); err == nil || len(d.calls) != 0 {
		t.Fatal("unknown provider reached doctor")
	}
	d = &selectedDoctor{}
	if _, err := doctorCompose(context.Background(), "", "", d); err != nil || !reflect.DeepEqual(d.calls, []domain.ComposeProviderName{domain.ComposeProviderDocker}) {
		t.Fatal("legacy default changed")
	}
}
