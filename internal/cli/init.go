package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mahcialet/agent-env/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func addInit(root *cobra.Command, emit func(any) error) {
	root.AddCommand(&cobra.Command{Use: "init [repository]", Args: cobra.MaximumNArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		repo := "."
		if len(args) > 0 {
			repo = args[0]
		}
		absolute, err := filepath.Abs(repo)
		if err != nil {
			return err
		}
		var file string
		for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
			if info, err := os.Stat(filepath.Join(absolute, name)); err == nil && !info.IsDir() {
				if file != "" {
					return errors.New("multiple Compose candidates found; write an explicit .agent-env.yaml selecting files")
				}
				file = name
			}
		}
		if file == "" {
			return errors.New("no root Compose file found; write .agent-env.yaml with explicit runtime files")
		}
		b, err := os.ReadFile(filepath.Join(absolute, file))
		if err != nil {
			return err
		}
		var compose struct {
			Services map[string]any `yaml:"services"`
		}
		if err = yaml.Unmarshal(b, &compose); err != nil {
			return err
		}
		services := make([]string, 0, len(compose.Services))
		for name := range compose.Services {
			services = append(services, name)
		}
		sort.Strings(services)
		m := config.Manifest{Version: 1, Sources: map[string]config.Source{"main": {Repository: ".", DefaultRef: "HEAD"}}, Runtimes: map[string]config.Runtime{"main": {Type: "compose", Source: "main", ProjectDirectory: ".", Files: []string{file}}}, Components: map[string]config.Component{"app": {Runtime: "main", ComposeServices: services}}, Stacks: map[string]config.Stack{"app": {Description: "Review this generated stack before creating a lease", Roots: []string{"app"}}}}
		if err := config.Validate(&m); err != nil {
			return err
		}
		data, err := yaml.Marshal(m)
		if err != nil {
			return err
		}
		path := filepath.Join(absolute, ".agent-env.yaml")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("manifest already exists or cannot be created: %w", err)
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return err
		}
		return emit(map[string]any{"path": path, "review_required": true, "stack": "app"})
	}})
}
