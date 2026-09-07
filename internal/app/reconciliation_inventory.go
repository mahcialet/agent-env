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
	contexts := map[string]bool{}
	for _, l := range leases {
		byID[l.ID] = l
		for _, r := range l.Runtimes {
			projects[r.Context+"\x00"+r.Project] = l.ID
			if r.Context != "" {
				contexts[r.Context] = true
			}
		}
		for _, source := range l.Sources {
			if source.WorktreePath != "" {
				paths[filepath.Clean(source.WorktreePath)] = source
			}
		}
	}
	var failures []error
	if s.Runtime == nil {
		failures = append(failures, fmt.Errorf("runtime inventory provider is missing"))
	} else {
		inventory, ok := s.Runtime.(RuntimeInventory)
		if !ok {
			failures = append(failures, fmt.Errorf("runtime provider does not support global inventory"))
		} else {
			doctor, e := s.Runtime.Doctor(ctx)
			if e != nil {
				failures = append(failures, e)
			} else if doctor["context"] == "" {
				failures = append(failures, fmt.Errorf("runtime doctor returned no active Docker context"))
			} else {
				contexts[doctor["context"]] = true
			}
			for _, contextName := range sortedKeys(contexts) {
				items, e := inventory.Inventory(ctx, contextName)
				if e != nil {
					failures = append(failures, fmt.Errorf("inventory context %s: %w", contextName, e))
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
					known := projects[contextName+"\x00"+p]
					owner := item.LeaseID
					status, diagnostic := "", ""
					switch {
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
