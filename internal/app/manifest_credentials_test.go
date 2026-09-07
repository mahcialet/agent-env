package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

func TestInheritedCredentialDoesNotMatchManifestSchema(t *testing.T) {
	// The Windows hosted runner sets this PostgreSQL default. It remains a
	// credential, but must not match the fixed manifest schema field "roots".
	t.Setenv("PGPASSWORD", "root")
	if len(evidence.Secrets(map[string]string{"PGPASSWORD": "root"})) != 1 {
		t.Fatal("PostgreSQL password must remain classified as a credential")
	}
	s, options, _, _, _ := lifecycleFixture(t)
	lease, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Observed != "ready" {
		t.Fatal("schema collision prevented lease creation")
	}
	if _, err := ownedConfiguration([]byte(`{"services":{"api":{}}}`), domain.Runtime{Project: "project", Name: "runtime", Services: []string{"api"}}, "lease"); err != nil {
		t.Fatalf("safe Compose snapshot rejected: %v", err)
	}

	s, options, _, _, _ = lifecycleFixture(t)
	manifestPath := filepath.Join(options.Repository, ".agent-env.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "review: {roots:", "review: {description: root, roots:", 1))
	if !strings.Contains(string(data), "description: root") {
		t.Fatal("fixture description missing")
	}
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), options, CreateOptions{Owner: "tester"}); err == nil || !strings.Contains(err.Error(), "manifest contains inherited credentials") {
		t.Fatal("literal inherited credential was not rejected before persistence")
	}
	leases, err := s.Store.List(context.Background())
	if err != nil || len(leases) != 0 {
		t.Fatal("rejected manifest persisted a lease")
	}
}
