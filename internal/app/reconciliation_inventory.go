package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mahcialet/agent-env/internal/domain"
)

// RuntimeInventory is optional so other runtime providers can explicitly report
// inventory as unsupported without pretending to have inspected external state.
type RuntimeInventory interface {
	Inventory(context.Context, string) ([]domain.Resource, error)
}

// ComposeInventoryDoctor discovers engines without requiring Compose startup tools.
type ComposeInventoryDoctor interface {
	InventoryDoctorFor(context.Context, domain.ComposeProviderName) (map[string]string, error)
}

// ComposeProviderDiscovery lists installed providers without starting engines.
type ComposeProviderDiscovery interface {
	AvailableComposeProviders() []domain.ComposeProviderName
}

// ComposeProviderInventory observes an explicit provider and persisted engine identity.
type ComposeProviderInventory interface {
	InventoryFor(context.Context, domain.ComposeProviderName, string) ([]domain.Resource, error)
}

type composeInventoryTarget struct {
	provider domain.ComposeProviderName
	identity string
}

func (t composeInventoryTarget) projectKey(project string) string {
	return string(t.provider) + "\x00" + t.identity + "\x00" + project
}

// Inventory returns anomalous resources only. It does not save rows, emit
// mutating reconciliation events, adopt resources, or perform cleanup.
func (s *Service) Inventory(ctx context.Context) ([]domain.Resource, error) {
	result := []domain.Resource{}
	if s.Store == nil {
		return result, fmt.Errorf("inventory requires a lease store")
	}
	leases, err := s.Store.List(ctx)
	if err != nil {
		return result, err
	}
	byID := map[string]domain.Lease{}
	projects := map[string]string{}
	paths := map[string]domain.Source{}
	contexts := map[composeInventoryTarget]bool{}
	providers := map[domain.ComposeProviderName]bool{}
	for _, l := range leases {
		byID[l.ID] = l
		for _, r := range l.Runtimes {
			if r.Type == "android-emulator" {
				continue
			}
			target := composeInventoryTarget{domain.EffectiveComposeProvider(r.Provider), r.Context}
			providers[target.provider] = true
			projects[target.projectKey(r.Project)] = l.ID
			if r.Context != "" {
				contexts[target] = true
			}
		}
		for _, source := range l.Sources {
			if source.WorktreePath != "" {
				paths[filepath.Clean(source.WorktreePath)] = source
			}
		}
	}
	var failures []error
	// Global discovery must find orphan projects even when no Compose rows survive.
	if s.Runtime == nil {
		failures = append(failures, fmt.Errorf("runtime inventory provider is missing"))
	} else {
		inventory, ok := s.Runtime.(RuntimeInventory)
		if !ok {
			failures = append(failures, fmt.Errorf("runtime provider does not support global inventory"))
		} else {
			if discovery, ok := s.Runtime.(ComposeProviderDiscovery); ok {
				for _, provider := range discovery.AvailableComposeProviders() {
					providers[provider] = true
				}
			} else {
				// Legacy adapters have only Docker, whose active context was always inspected.
				providers[domain.ComposeProviderDocker] = true
			}
			for provider := range providers {
				var doctor map[string]string
				var e error
				if selected, ok := s.Runtime.(ComposeInventoryDoctor); ok {
					doctor, e = selected.InventoryDoctorFor(ctx, provider)
				} else if selected, ok := s.Runtime.(ComposeProviderDoctor); ok {
					doctor, e = selected.DoctorFor(ctx, provider)
				} else if provider == domain.ComposeProviderDocker {
					doctor, e = s.Runtime.Doctor(ctx)
				} else {
					e = fmt.Errorf("inventory cannot discover provider %s", provider)
				}
				if e != nil {
					failures = append(failures, e)
				} else if doctor["context"] == "" {
					failures = append(failures, fmt.Errorf("runtime doctor returned no active %s connection identity", provider))
				} else {
					contexts[composeInventoryTarget{provider, doctor["context"]}] = true
				}
			}
			targets := make([]composeInventoryTarget, 0, len(contexts))
			for target := range contexts {
				targets = append(targets, target)
			}
			sort.Slice(targets, func(i, j int) bool { return targets[i].projectKey("") < targets[j].projectKey("") })
			for _, target := range targets {
				contextName := target.identity
				var items []domain.Resource
				var e error
				selected, explicit := s.Runtime.(ComposeProviderInventory)
				if explicit {
					items, e = selected.InventoryFor(ctx, target.provider, contextName)
				} else if target.provider == domain.ComposeProviderDocker {
					items, e = inventory.Inventory(ctx, contextName)
				} else {
					e = fmt.Errorf("runtime inventory cannot select provider %s", target.provider)
				}
				if e != nil {
					failures = append(failures, fmt.Errorf("inventory provider %s context %s: %w", target.provider, contextName, e))
				}
				// Project ownership can be established by an explicit agent label on
				// a container, never by an ae-style project-name guess.
				labelledProjects := map[string]string{}
				ambiguous := map[string]bool{}
				for _, item := range items {
					p := item.Metadata["project"]
					if p != "" && item.LeaseID != "" {
						if prior := labelledProjects[p]; prior != "" && prior != item.LeaseID {
							ambiguous[p] = true
						}
						labelledProjects[p] = item.LeaseID
					}
				}
				for _, item := range items {
					if item.Metadata == nil {
						item.Metadata = map[string]string{}
					}
					p := item.Metadata["project"]
					known := projects[target.projectKey(p)]
					owner := item.LeaseID
					status, diagnostic := "", ""
					switch {
					case explicit && item.Metadata["provider"] != string(target.provider):
						status, diagnostic = "identity_mismatch", "provider returned a resource from a different provider"
					case item.Metadata["context"] != contextName:
						status, diagnostic = "identity_mismatch", "provider returned a resource from a different context"
					case ambiguous[p]:
						status, diagnostic = "identity_mismatch", "project carries conflicting lease ownership labels"
					case known != "" && owner != "" && owner != known:
						status, diagnostic = "identity_mismatch", "project registry owner differs from resource lease label"
					case known != "" && byID[known].Desired == "released":
						status, diagnostic = "orphaned", "released lease still has external resources"
					case known != "":
						continue
					case owner != "":
						if _, exists := byID[owner]; exists {
							status, diagnostic = "identity_mismatch", "resource lease exists but this project/context is not registered"
						} else {
							status, diagnostic = "orphaned", "managed resource lease has no registry record"
						}
					case labelledProjects[p] != "":
						status, diagnostic = "orphaned", "project has agent-labelled resources but no registered project identity"
					default:
						status, diagnostic = "foreign", "Compose resource has no established agent ownership; do not adopt or delete"
					}
					item.Metadata["status"], item.Metadata["diagnostic"] = status, diagnostic
					result = append(result, item)
				}
			}
		}
	}
	if s.Home == "" {
		failures = append(failures, fmt.Errorf("state home is required for filesystem inventory"))
	} else {
		home, e := filepath.Abs(s.Home)
		if e != nil {
			failures = append(failures, e)
		} else {
			for _, area := range []string{"worktrees", "leases"} {
				root := filepath.Join(home, area)
				rootInfo, rootErr := os.Lstat(root)
				if errors.Is(rootErr, os.ErrNotExist) {
					continue
				}
				if rootErr != nil {
					failures = append(failures, rootErr)
					continue
				}
				if rootInfo.Mode()&os.ModeSymlink != 0 {
					result = append(result, pathFinding(area, root, "", "unknown", "managed root is a symlink; not followed"))
					continue
				}
				entries, e := os.ReadDir(root)
				if errors.Is(e, os.ErrNotExist) {
					continue
				}
				if e != nil {
					failures = append(failures, e)
					continue
				}
				for _, entry := range entries {
					id := entry.Name()
					p := filepath.Join(root, id)
					if entry.Type()&os.ModeSymlink != 0 {
						result = append(result, pathFinding(area, p, id, "unknown", "managed-root entry is a symlink; not followed"))
						continue
					}
					if !entry.IsDir() {
						continue
					}
					if area == "leases" {
						if _, exists := byID[id]; !exists {
							result = append(result, pathFinding("lease-directory", p, id, "orphaned", "lease evidence directory has no registry record; retain evidence"))
						}
						continue
					}
					sources, e := os.ReadDir(p)
					if e != nil {
						failures = append(failures, e)
						continue
					}
					for _, sourceEntry := range sources {
						worktree := filepath.Join(p, sourceEntry.Name())
						if sourceEntry.Type()&os.ModeSymlink != 0 {
							result = append(result, pathFinding("worktree", worktree, id, "unknown", "worktree path is a symlink; not followed"))
							continue
						}
						if !sourceEntry.IsDir() {
							continue
						}
						if _, exists := paths[filepath.Clean(worktree)]; exists {
							continue
						}
						finding := pathFinding("worktree", worktree, id, "orphaned", "worktree path is absent from the registry")
						if s.Source == nil {
							finding.Metadata["status"], finding.Metadata["diagnostic"] = "unknown", "source provider is missing; worktree identity unverified"
						} else {
							resolved, e := s.Source.Resolve(ctx, worktree, "HEAD")
							if e != nil {
								finding.Metadata["status"], finding.Metadata["diagnostic"] = "unknown", "cannot resolve Git identity: "+e.Error()
							} else {
								resolved.WorktreePath = worktree
								resolved.Alias = sourceEntry.Name()
								o, e := s.Source.Inspect(ctx, resolved)
								if e != nil {
									finding.Metadata["status"], finding.Metadata["diagnostic"] = "unknown", "cannot inspect Git identity: "+e.Error()
								} else {
									finding.Metadata["repository_id"], finding.Metadata["commit"] = resolved.RepositoryID, o.Commit
									if !o.Exists || !o.Registered || o.Commit != resolved.Commit {
										finding.Metadata["status"], finding.Metadata["diagnostic"] = "unknown", "candidate lacks a matching registered Git worktree identity"
									}
									if o.TrackedDirty {
										finding.Metadata["tracked_dirty"] = "true"
									}
								}
							}
						}
						result = append(result, finding)
					}
				}
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, errors.Join(failures...)
}

func pathFinding(kind, path, id, status, diagnostic string) domain.Resource {
	return domain.Resource{ID: kind + ":" + path, LeaseID: id, Kind: kind, ExternalID: path, Metadata: map[string]string{"path": path, "status": status, "diagnostic": strings.TrimSpace(diagnostic)}}
}
