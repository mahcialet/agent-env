package app

import (
	"encoding/json"
	"testing"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

func TestDeclaredEndpointPublishesLoopbackDynamically(t *testing.T) {
	m := config.Manifest{Components: map[string]config.Component{"api": {Endpoints: map[string]config.Endpoint{"http": {Service: "api", Target: 8080}}}}}
	manifest, _ := json.Marshal(m)
	l := domain.Lease{Manifest: manifest, Components: []domain.Component{{Name: "api", Runtime: "backend"}}}
	r := domain.Runtime{Name: "backend"}
	input := []byte(`{"services":{"api":{"image":"example","ports":[{"target":8080,"published":"0","host_ip":"0.0.0.0"},{"target":9000,"published":"0"}]}}}`)
	b, err := endpointConfiguration(input, l, r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err = json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	ports := raw["services"].(map[string]any)["api"].(map[string]any)["ports"].([]any)
	if len(ports) != 2 {
		t.Fatalf("duplicate publishing: %s", b)
	}
	generated := ports[1].(map[string]any)
	if generated["published"] != "0" || generated["host_ip"] != "127.0.0.1" || generated["target"] != float64(8080) {
		t.Fatalf("unsafe endpoint: %s", b)
	}
}
