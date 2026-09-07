package compose

func pruneConfig(raw map[string]any, selected []string) {
	all, _ := raw["services"].(map[string]any)
	kept := map[string]any{}
	used := map[string]map[string]bool{"volumes": {}, "networks": {}, "configs": {}, "secrets": {}}
	for _, name := range selected {
		service, _ := all[name].(map[string]any)
		kept[name] = service
		if networks, ok := service["networks"].(map[string]any); ok {
			for n := range networks {
				used["networks"][n] = true
			}
		}
		for _, kind := range []string{"volumes", "configs", "secrets"} {
			entries, _ := service[kind].([]any)
			for _, entry := range entries {
				if name, ok := entry.(string); ok {
					used[kind][name] = true
					continue
				}
				value, _ := entry.(map[string]any)
				if kind == "volumes" && value["type"] != "volume" {
					continue
				}
				if source, ok := value["source"].(string); ok {
					used[kind][source] = true
				}
			}
		}
	}
	raw["services"] = kept
	for kind, names := range used {
		resources, _ := raw[kind].(map[string]any)
		subset := map[string]any{}
		for name := range names {
			if value, ok := resources[name]; ok {
				subset[name] = value
			}
		}
		if len(subset) > 0 {
			raw[kind] = subset
		} else {
			delete(raw, kind)
		}
	}
}
