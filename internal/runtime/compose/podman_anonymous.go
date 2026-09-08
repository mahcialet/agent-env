package compose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
)

type podmanAnonymousVolume struct {
	Name, Mountpoint, CreatedAt, Driver string
	Anonymous                           bool
	Labels                              map[string]string
}

func (v podmanAnonymousVolume) proof() (string, error) {
	if !v.Anonymous || v.Name == "" || v.Mountpoint == "" || v.CreatedAt == "" || len(v.Labels) != 0 {
		return "", fmt.Errorf("volume %s lacks unambiguous native anonymous-volume identity", v.Name)
	}
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}
func (c podmanClient) anonymousVolumes(ctx context.Context, r domain.Runtime, observation Observation) ([]domain.Resource, error) {
	p, err := decodePodmanIdentity(r.Context)
	if err != nil {
		return nil, err
	}
	var result []domain.Resource
	seen := map[string]bool{}
	for _, owned := range observation.Resources {
		if owned.Kind != "compose_container" && owned.Kind != "container" {
			continue
		}
		out, err := c.command(ctx, p, "inspect", "--type", "container", owned.ExternalID)
		if err != nil {
			return result, err
		}
		var containers []struct {
			ID     string
			Mounts []struct{ Type, Name string }
		}
		if err = json.Unmarshal([]byte(out.Stdout), &containers); err != nil || len(containers) != 1 || containers[0].ID != owned.ExternalID {
			return result, fmt.Errorf("cannot confirm anonymous-volume attachment identity")
		}
		for _, mount := range containers[0].Mounts {
			if mount.Type != "volume" || mount.Name == "" || seen[mount.Name] {
				continue
			}
			seen[mount.Name] = true
			volumeOut, err := c.command(ctx, p, "volume", "inspect", mount.Name)
			if err != nil {
				return result, err
			}
			var volumes []podmanAnonymousVolume
			if err = json.Unmarshal([]byte(volumeOut.Stdout), &volumes); err != nil || len(volumes) != 1 {
				return result, fmt.Errorf("incomplete attached volume inspection")
			}
			if !volumes[0].Anonymous {
				continue
			}
			proof, err := volumes[0].proof()
			if err != nil {
				return result, err
			}
			result = append(result, resource(r, "volume", mount.Name, map[string]string{"project": r.Project, "context": r.Context, "lease": r.LeaseID, "runtime": r.Name, "anonymous": "true", "volume_fingerprint": proof, "attached_container": owned.ExternalID, "provider": "podman-compose"}))
		}
	}
	return result, nil
}
func (c podmanClient) removeAnonymousVolumes(ctx context.Context, r domain.Runtime, volumes []domain.Resource) error {
	p, err := decodePodmanIdentity(r.Context)
	if err != nil {
		return err
	}
	for _, owned := range volumes {
		// Existence is observed through a successful list; an inspect failure is never absence.
		listing, err := c.command(ctx, p, "volume", "ls", "--quiet")
		if err != nil {
			return err
		}
		found := false
		for _, name := range strings.Fields(listing.Stdout) {
			if name == owned.ExternalID {
				found = true
			}
		}
		if !found {
			continue
		}
		out, err := c.command(ctx, p, "volume", "inspect", owned.ExternalID)
		if err != nil {
			return err
		}
		var data []podmanAnonymousVolume
		if err = json.Unmarshal([]byte(out.Stdout), &data); err != nil || len(data) != 1 {
			return fmt.Errorf("cannot revalidate anonymous volume")
		}
		proof, err := data[0].proof()
		if err != nil {
			return err
		}
		if proof != owned.Metadata["volume_fingerprint"] {
			return fmt.Errorf("anonymous volume identity changed; quarantine")
		}
		users, err := c.command(ctx, p, "ps", "--all", "--quiet", "--filter", "volume="+owned.ExternalID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(users.Stdout) != "" {
			return fmt.Errorf("anonymous volume still attached; quarantine")
		}
		current, _, err := c.fingerprint(ctx, p)
		if err != nil {
			return err
		}
		if current != p.Fingerprint {
			return fmt.Errorf("Podman engine fingerprint changed before anonymous-volume cleanup")
		}
		if _, err = c.command(ctx, p, "volume", "rm", owned.ExternalID); err != nil {
			return err
		}
		listing, err = c.command(ctx, p, "volume", "ls", "--quiet")
		if err != nil {
			return err
		}
		for _, name := range strings.Fields(listing.Stdout) {
			if name == owned.ExternalID {
				return fmt.Errorf("anonymous volume remained after cleanup")
			}
		}
	}
	return nil
}

func validatePodmanCleanupEvidence(r domain.Runtime, proof domain.Resource) error {
	m := proof.Metadata
	if proof.Kind != "volume" || proof.ExternalID == "" || proof.Runtime != r.Name || m["anonymous"] != "true" || m["provider"] != "podman-compose" || m["lease"] != r.LeaseID || m["runtime"] != r.Name || m["project"] != r.Project || m["context"] != r.Context || m["attached_container"] == "" {
		return fmt.Errorf("anonymous-volume cleanup evidence is outside recorded runtime scope")
	}
	bytes, err := hex.DecodeString(m["volume_fingerprint"])
	if err != nil || len(bytes) != sha256.Size {
		return fmt.Errorf("invalid anonymous-volume cleanup fingerprint")
	}
	return nil
}
func (c podmanClient) PrepareCleanup(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	evidence := map[string]domain.Resource{}
	for _, proof := range r.CleanupEvidence {
		if err := validatePodmanCleanupEvidence(r, proof); err != nil {
			return r, err
		}
		if prior, ok := evidence[proof.ExternalID]; ok && prior.Metadata["volume_fingerprint"] != proof.Metadata["volume_fingerprint"] {
			return r, fmt.Errorf("conflicting anonymous-volume cleanup evidence")
		}
		evidence[proof.ExternalID] = proof
	}
	observation, err := c.Inspect(ctx, r)
	if err != nil {
		return r, err
	}
	for _, proof := range observation.Resources {
		if proof.Metadata["anonymous"] != "true" {
			continue
		}
		if err := validatePodmanCleanupEvidence(r, proof); err != nil {
			return r, err
		}
		if prior, ok := evidence[proof.ExternalID]; ok && prior.Metadata["volume_fingerprint"] != proof.Metadata["volume_fingerprint"] {
			return r, fmt.Errorf("anonymous volume changed since cleanup evidence was recorded")
		}
		evidence[proof.ExternalID] = proof
	}
	r.CleanupEvidence = nil
	names := make([]string, 0, len(evidence))
	for name := range evidence {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		r.CleanupEvidence = append(r.CleanupEvidence, evidence[name])
	}
	return r, nil
}

func (c podmanClient) retainedAnonymousVolumes(ctx context.Context, r domain.Runtime, live []domain.Resource) ([]domain.Resource, error) {
	if len(r.CleanupEvidence) == 0 {
		return nil, nil
	}
	p, err := decodePodmanIdentity(r.Context)
	if err != nil {
		return nil, err
	}
	listing, err := c.command(ctx, p, "volume", "ls", "--quiet")
	if err != nil {
		return nil, err
	}
	exists := map[string]bool{}
	for _, name := range strings.Fields(listing.Stdout) {
		exists[name] = true
	}
	seen := map[string]string{}
	for _, resource := range live {
		if resource.Metadata["anonymous"] == "true" {
			seen[resource.ExternalID] = resource.Metadata["volume_fingerprint"]
		}
	}
	var result []domain.Resource
	for _, proof := range r.CleanupEvidence {
		if err = validatePodmanCleanupEvidence(r, proof); err != nil {
			return result, err
		}
		if !exists[proof.ExternalID] {
			continue
		}
		out, err := c.command(ctx, p, "volume", "inspect", proof.ExternalID)
		if err != nil {
			return result, err
		}
		var volumes []podmanAnonymousVolume
		if err = json.Unmarshal([]byte(out.Stdout), &volumes); err != nil || len(volumes) != 1 {
			return result, fmt.Errorf("cannot inspect retained anonymous volume")
		}
		fingerprint, err := volumes[0].proof()
		if err != nil {
			return result, err
		}
		if fingerprint != proof.Metadata["volume_fingerprint"] {
			return result, fmt.Errorf("retained anonymous volume identity changed")
		}
		if existing, ok := seen[proof.ExternalID]; ok {
			if existing != fingerprint {
				return result, fmt.Errorf("conflicting live anonymous volume identity")
			}
			continue
		}
		result = append(result, proof)
	}
	return result, nil
}
