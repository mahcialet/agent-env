package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
)

// Inventory observes labelled resources in an explicit Docker context. It never
// changes projects or adopts resources whose ownership is not established.
func (c dockerClient) Inventory(ctx context.Context, contextName string) ([]domain.Resource, error) {
	result := []domain.Resource{}
	if contextName == "" || strings.ContainsAny(contextName, "\x00\r\n") {
		return result, fmt.Errorf("inventory requires a recorded Docker context")
	}
	base := []string{"--context", contextName}
	out, err := c.run(ctx, "", append(append([]string{}, base...), "compose", "ls", "--all", "--format", "json")...)
	if err != nil {
		return result, err
	}
	var projects []struct{ Name, Status, ConfigFiles string }
	if err := json.Unmarshal([]byte(out.Stdout), &projects); err != nil {
		return result, fmt.Errorf("invalid Compose project inventory JSON: %w", err)
	}
	for _, p := range projects {
		if p.Name == "" {
			return result, fmt.Errorf("Compose inventory returned a project without identity")
		}
		result = append(result, domain.Resource{ID: contextName + ":project:" + p.Name, Kind: "compose-project", ExternalID: p.Name, Metadata: map[string]string{"context": contextName, "project": p.Name, "observed_status": p.Status, "config_files": p.ConfigFiles, "ownership": "unknown"}})
	}
	return c.inventoryResources(ctx, contextName, result)
}

// inventoryResources observes labelled native resources independently of Compose.
func (c dockerClient) inventoryResources(ctx context.Context, contextName string, result []domain.Resource) ([]domain.Resource, error) {
	base := []string{"--context", contextName}
	for _, kind := range []string{"container", "network", "volume"} {
		set := map[string]bool{}
		for _, label := range []string{leaseLabel, projectLabel} {
			args := append([]string{}, base...)
			if kind == "container" {
				args = append(args, "ps", "--all", "--quiet")
			} else {
				args = append(args, kind, "ls", "--quiet")
			}
			args = append(args, "--filter", "label="+label)
			out, err := c.run(ctx, "", args...)
			if err != nil {
				return result, err
			}
			for _, id := range ids(out.Stdout) {
				set[id] = true
			}
		}
		all := make([]string, 0, len(set))
		for id := range set {
			all = append(all, id)
		}
		sort.Strings(all)
		if len(all) == 0 {
			continue
		}
		args := append([]string{}, base...)
		if kind == "container" {
			args = append(args, "inspect", "--type", "container")
		} else {
			args = append(args, kind, "inspect")
		}
		args = append(args, all...)
		out, err := c.run(ctx, "", args...)
		if err != nil {
			return result, err
		}
		if kind == "container" {
			var data []containerData
			if err := json.Unmarshal([]byte(out.Stdout), &data); err != nil {
				return result, fmt.Errorf("invalid container inventory JSON: %w", err)
			}
			if len(data) != len(all) {
				return result, fmt.Errorf("container inventory inspection is incomplete")
			}
			for _, v := range data {
				if v.ID == "" {
					return result, fmt.Errorf("container inventory has empty ID")
				}
				item := inventoryResource(contextName, kind, v.ID, v.Config.Labels)
				item.Metadata["service"] = v.Config.Labels["com.docker.compose.service"]
				item.Metadata["observed_status"] = v.State.Status
				item.Metadata["image"] = v.Config.Image
				item.Metadata["image_id"] = v.Image
				result = append(result, item)
			}
		} else {
			var data []namedData
			if err := json.Unmarshal([]byte(out.Stdout), &data); err != nil {
				return result, fmt.Errorf("invalid %s inventory JSON: %w", kind, err)
			}
			if len(data) != len(all) {
				return result, fmt.Errorf("%s inventory inspection is incomplete", kind)
			}
			for _, v := range data {
				id := v.ID
				if kind == "volume" {
					id = v.Name
				}
				if id == "" {
					return result, fmt.Errorf("%s inventory has empty ID", kind)
				}
				item := inventoryResource(contextName, kind, id, v.Labels)
				item.Metadata["name"] = v.Name
				result = append(result, item)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func inventoryResource(contextName, kind, id string, labels map[string]string) domain.Resource {
	ownership := "unknown"
	if labels[leaseLabel] != "" {
		ownership = "labelled"
	}
	return domain.Resource{ID: contextName + ":" + kind + ":" + id, LeaseID: labels[leaseLabel], Runtime: labels[runtimeLabel], Kind: kind, ExternalID: id, Metadata: map[string]string{"context": contextName, "project": labels[projectLabel], "lease": labels[leaseLabel], "ownership": ownership}}
}
