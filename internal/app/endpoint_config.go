package app

import (
	"encoding/json"
	"fmt"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/domain"
)

// Declared endpoints request isolated ephemeral loopback publishing. This changes
// only the generated execution snapshot; source Compose files stay untouched.
func endpointConfiguration(data []byte, l domain.Lease, r domain.Runtime) ([]byte, error) {
	var m config.Manifest
	if err := json.Unmarshal(l.Manifest, &m); err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	services, _ := raw["services"].(map[string]any)
	for _, component := range l.Components {
		if component.Runtime != r.Name {
			continue
		}
		for _, name := range sortedKeys(m.Components[component.Name].Endpoints) {
			endpoint := m.Components[component.Name].Endpoints[name]
			service, ok := services[endpoint.Service].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("endpoint %s.%s references missing selected service", component.Name, name)
			}
			protocol := endpoint.Protocol
			if protocol == "" {
				protocol = "tcp"
			}
			ports, _ := service["ports"].([]any)
			kept := []any{}
			for _, port := range ports {
				value, _ := port.(map[string]any)
				target, _ := value["target"].(float64)
				p, _ := value["protocol"].(string)
				if p == "" {
					p = "tcp"
				}
				if int(target) != endpoint.Target || p != protocol {
					kept = append(kept, port)
				}
			}
			service["ports"] = append(kept, map[string]any{"target": endpoint.Target, "published": "0", "host_ip": "127.0.0.1", "protocol": protocol, "mode": "ingress"})
		}
	}
	return json.Marshal(raw)
}
