package app

import (
	"context"
	"fmt"

	"github.com/mahcialet/agent-env/internal/domain"
)

func (s *Service) previewDestroy(ctx context.Context, l domain.Lease, force bool) (domain.Lease, error) {
	l.Diagnostics = append(append([]string{}, l.Diagnostics...), "dry-run: no state or resources changed")
	for _, source := range l.Sources {
		o, err := s.Source.Inspect(ctx, source)
		if err != nil {
			l.Diagnostics = append(l.Diagnostics, "would quarantine source "+source.Alias+": "+err.Error())
			continue
		}
		if o.TrackedDirty && !force {
			l.Diagnostics = append(l.Diagnostics, "would quarantine tracked changes in "+source.Alias)
		} else if o.Exists || o.Registered {
			l.Diagnostics = append(l.Diagnostics, fmt.Sprintf("would remove source %s at %s (force=%t)", source.Alias, source.WorktreePath, force))
		}
	}
	for _, r := range l.Runtimes {
		o, err := s.inspectRuntime(ctx, r)
		if err != nil {
			l.Diagnostics = append(l.Diagnostics, "would quarantine runtime "+r.Name+": "+err.Error())
			continue
		}
		if err := ownedResources(l.ID, o.Resources); err != nil {
			l.Diagnostics = append(l.Diagnostics, "would quarantine: "+err.Error())
		} else if r.Type == "android-emulator" && r.Android != nil {
			a := r.Android
			l.Diagnostics = append(l.Diagnostics, fmt.Sprintf("would verify ownership, stop Android Emulator %s (%s) if running, and remove private writable AVD state at %s; retain runtime evidence and process logs", a.AVDName, a.Serial, a.AVDPath))
		} else if o.Exists {
			l.Diagnostics = append(l.Diagnostics, "would retain logs and remove Compose project "+r.Project+" in context "+r.Context)
		}
	}
	return l, nil
}
