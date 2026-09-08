package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func publicationGate(data []byte) error {
	var w struct {
		Jobs map[string]struct {
			Needs    any              `yaml:"needs"`
			If       string           `yaml:"if"`
			Continue bool             `yaml:"continue-on-error"`
			Steps    []map[string]any `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if e := yaml.Unmarshal(data, &w); e != nil {
		return e
	}
	pub, ok := w.Jobs["publish"]
	if !ok {
		return fmt.Errorf("missing publish job")
	}
	needs, ok := pub.Needs.([]any)
	if !ok {
		return fmt.Errorf("publication must depend on build and smoke")
	}
	seen := map[string]bool{}
	for _, n := range needs {
		seen[fmt.Sprint(n)] = true
	}
	if !seen["build"] || !seen["smoke"] {
		return fmt.Errorf("missing validation dependency")
	}
	for _, name := range []string{"build", "smoke", "publish"} {
		j, ok := w.Jobs[name]
		if !ok || j.If != "" || j.Continue {
			return fmt.Errorf("validation gate bypass")
		}
		for _, s := range j.Steps {
			if _, ok := s["continue-on-error"]; ok {
				return fmt.Errorf("step failure ignored")
			}
			if _, ok := s["if"]; ok {
				return fmt.Errorf("conditional validation")
			}
		}
	}
	if w.Jobs["smoke"].Needs != "build" {
		return fmt.Errorf("smoke must consume built bytes")
	}
	upload, download := "", ""
	repeated := false
	for _, s := range w.Jobs["build"].Steps {
		if strings.HasPrefix(fmt.Sprint(s["uses"]), "actions/upload-artifact@") {
			with, _ := s["with"].(map[string]any)
			upload = fmt.Sprint(with["name"])
		}
		if strings.Contains(fmt.Sprint(s["run"]), "release-repeat") {
			repeated = true
		}
	}
	for _, s := range pub.Steps {
		if _, ok := s["run"]; ok {
			return fmt.Errorf("publish must not rebuild")
		}
		if strings.HasPrefix(fmt.Sprint(s["uses"]), "actions/download-artifact@") {
			with, _ := s["with"].(map[string]any)
			download = fmt.Sprint(with["name"])
		}
	}
	if upload == "" || download != upload || !repeated {
		return fmt.Errorf("publication must consume repeated validated artifact")
	}
	return nil
}
func TestReleasePublicationGate(t *testing.T) {
	root, e := repositoryRoot()
	if e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if e != nil {
		t.Fatal(e)
	}
	if e = publicationGate(data); e != nil {
		t.Fatal(e)
	}
	for _, replacement := range []struct{ old, new string }{{"needs: [build, smoke]", "needs: [build]"}, {"needs: [build, smoke]", "needs: [build, smoke]\n    if: always()"}, {"name: release-candidate", "name: wrong-candidate"}, {"release-repeat", "release-check"}} {
		t.Run(replacement.new, func(t *testing.T) {
			changed := strings.Replace(string(data), replacement.old, replacement.new, 1)
			if changed == string(data) {
				t.Fatal("fixture did not mutate workflow")
			}
			if e = publicationGate([]byte(changed)); e == nil {
				t.Fatal("unsafe publication gate accepted")
			}
		})
	}
}
