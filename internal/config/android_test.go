package config

import (
	"strings"
	"testing"
)

const androidManifest = `version: 1
sources:
  self: {repository: ., default_ref: HEAD}
runtimes:
  device: {type: android-emulator, source: self, avd: Pixel.API_35}
components:
  device: {runtime: device, provides: [android-emulator]}
stacks:
  android: {roots: [device]}
`

func TestAndroidManifestAcceptsTemplateWithoutComposeServices(t *testing.T) {
	m, err := Parse([]byte(androidManifest))
	if err != nil {
		t.Fatal(err)
	}
	if m.Runtimes["device"].AVD != "Pixel.API_35" || len(m.Components["device"].ComposeServices) != 0 {
		t.Fatalf("Android manifest lost identity: %+v", m)
	}
	before := Digest(m)
	r := m.Runtimes["device"]
	r.AVD = "Pixel.API_36"
	m.Runtimes["device"] = r
	if before == Digest(m) {
		t.Fatal("template change did not change manifest digest")
	}
}
func TestAndroidManifestRejectsInvalidRuntimeContracts(t *testing.T) {
	for _, tc := range []struct{ name, old, next, want string }{
		{"missing template", ", avd: Pixel.API_35", "", "valid avd"},
		{"traversal template", "Pixel.API_35", "../Pixel", "valid avd"},
		{"absolute template", "Pixel.API_35", "/Pixel", "valid avd"},
		{"unknown source", "source: self", "source: missing", "unknown source"},
		{"missing source", "source: self, ", "", "unknown source"},
		{"compose files", "avd: Pixel.API_35", "avd: Pixel.API_35, files: [compose.yaml]", "cannot use Compose files"},
		{"compose directory", "avd: Pixel.API_35", "avd: Pixel.API_35, project_directory: .", "cannot use Compose files"},
		{"compose services", "runtime: device, provides", "runtime: device, compose_services: [api], provides", "cannot use Compose services"},
		{"compose endpoints", "runtime: device, provides", "runtime: device, endpoints: {api: {service: api, target: 8080}}, provides", "cannot use Compose services"},
		{"compose readiness", "runtime: device, provides", "runtime: device, readiness: [{type: compose}], provides", "cannot use Compose readiness"},
		{"unknown Android key", "avd: Pixel.API_35", "avd: Pixel.API_35, serial: emulator-5554", "field serial not found"},
		{"compose avd", "type: android-emulator", "type: compose, files: [compose.yaml]", "cannot use avd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(androidManifest, tc.old, tc.next, 1)
			if body == androidManifest {
				t.Fatal("fixture unchanged")
			}
			_, err := Parse([]byte(body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q: %v", tc.want, err)
			}
		})
	}
}
func TestAndroidManifestSupportsMixedDependencyClosure(t *testing.T) {
	m, err := Parse([]byte(androidManifest))
	if err != nil {
		t.Fatal(err)
	}
	m.Runtimes["backend"] = Runtime{Type: "compose", Source: "self", Files: []string{"compose.yaml"}}
	m.Components["api"] = Component{Runtime: "backend", ComposeServices: []string{"api"}}
	c := m.Components["device"]
	c.DependsOn = []string{"api"}
	c.Readiness = []Probe{{Type: "command", Source: "self", Command: []string{"git", "status"}}, {Type: "http", URL: "http://127.0.0.1:8080/health"}}
	m.Components["device"] = c
	if err := Validate(m); err != nil {
		t.Fatal(err)
	}
}
