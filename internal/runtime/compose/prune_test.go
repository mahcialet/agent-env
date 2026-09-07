package compose

import (
	"encoding/json"
	"testing"
)

func TestPruneRemovesUnselectedCleanupTargets(t *testing.T) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(`{"services":{"api":{"networks":{"default":null},"volumes":[{"type":"volume","source":"api-data"}],"secrets":[{"source":"api-key"}]},"dashboard":{"volumes":[{"type":"volume","source":"global-data"}]}},"networks":{"default":{"name":"ae_test_default"},"unused":{"name":"global-network"}},"volumes":{"api-data":{"name":"ae_test_data"},"global-data":{"name":"unrelated-global-data"}},"secrets":{"api-key":{"file":"key"},"unused":{"file":"other"}}}`), &raw); err != nil {
		t.Fatal(err)
	}
	pruneConfig(raw, []string{"api"})
	for _, kind := range []string{"services", "networks", "volumes", "secrets"} {
		if len(raw[kind].(map[string]any)) != 1 {
			t.Fatalf("unselected %s retained: %v", kind, raw[kind])
		}
	}
	if _, exists := raw["volumes"].(map[string]any)["global-data"]; exists {
		t.Fatal("foreign global volume remains a down target")
	}
}
