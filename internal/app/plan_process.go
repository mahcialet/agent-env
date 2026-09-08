package app

import (
	"fmt"
	"github.com/mahcialet/agent-env/internal/config"
	"github.com/mahcialet/agent-env/internal/evidence"
)

func validateProcessSecrets(m config.Manifest) error {
	for name, r := range m.Runtimes {
		if r.Type != "process" {
			continue
		}
		for key, value := range r.Env {
			if len(evidence.Secrets(map[string]string{key: value})) > 0 && !envPlaceholder(value) {
				return fmt.Errorf("process runtime %s secret environment %s must use an explicit host reference", name, key)
			}
		}
		if evidence.ContainsSecretString(r, evidence.InheritedSecrets()) {
			return fmt.Errorf("process runtime %s contains literal inherited credentials", name)
		}
	}
	for name, component := range m.Components {
		if m.Runtimes[component.Runtime].Type == "process" && evidence.ContainsSecretString(component.Readiness, evidence.InheritedSecrets()) {
			return fmt.Errorf("process component %s readiness contains literal inherited credentials", name)
		}
	}
	return nil
}
