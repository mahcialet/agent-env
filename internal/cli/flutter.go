package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/config"
)

type flutterPrerequisites interface {
	Doctor(context.Context, string) (string, error)
	Validate(string, string, string) (string, error)
}

// Repository diagnostics inspect existing source checkouts. Immutable worktree
// materialization and all selected dependency prerequisites belong to create.
func doctorFlutterManifest(ctx context.Context, repository string, flutter flutterPrerequisites, android app.AndroidProvider) (string, map[string]any, error) {
	report := map[string]any{}
	m, err := config.Load(repository)
	if err != nil {
		return "", report, err
	}
	abs, err := filepath.Abs(repository)
	if err != nil {
		return "", report, err
	}
	names := make([]string, 0, len(m.Applications))
	for name := range m.Applications {
		names = append(names, name)
	}
	sort.Strings(names)
	var issues []error
	checkedRuntimes := map[string]bool{}
	for _, name := range names {
		a := m.Applications[name]
		source := m.Sources[a.Source].Repository
		if !filepath.IsAbs(source) {
			source = filepath.Join(abs, filepath.FromSlash(source))
		}
		entry := map[string]string{"source": a.Source, "runtime": a.Runtime, "executable": a.Build.Command[0]}
		report[name] = entry
		directory, err := flutter.Validate(source, a.ProjectDirectory, a.Build.Artifact)
		entry["directory"] = directory
		if err != nil {
			issues = append(issues, fmt.Errorf("Flutter application %s project: %w", name, err))
		}
		version, err := flutter.Doctor(ctx, a.Build.Command[0])
		entry["flutter_version"] = version
		if err != nil {
			issues = append(issues, fmt.Errorf("Flutter application %s executable: %w", name, err))
		}
		if !checkedRuntimes[a.Runtime] {
			checkedRuntimes[a.Runtime] = true
			r := m.Runtimes[a.Runtime]
			if _, err := android.Validate(ctx, r.AVD); err != nil {
				issues = append(issues, fmt.Errorf("Android runtime %s (AVD %s): %w", a.Runtime, r.AVD, err))
			}
		}
	}
	return config.Digest(m), report, errors.Join(issues...)
}
