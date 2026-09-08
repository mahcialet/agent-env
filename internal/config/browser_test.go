package config

import (
	"encoding/json"
	"strings"
	"testing"
)

const browserManifest = `version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  chrome:
    type: process
    source: app
    working_directory: .
    command: [chrome, '--headless=new', '--enable-automation', '--user-data-dir=${runtime_dir}/profile', '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=${port:cdp}', 'about:blank']
    ports:
      cdp: {protocol: tcp}
browsers:
  web: {type: chromium-cdp, runtime: chrome, cdp_port: cdp}
components:
  page: {runtime: chrome}
stacks:
  local: {roots: [page]}
`

func TestBrowserManifestContract(t *testing.T) {
	m, err := Parse([]byte(browserManifest))
	if err != nil {
		t.Fatal(err)
	}
	want := Browser{Type: "chromium-cdp", Runtime: "chrome", CDPPort: "cdp"}
	if m.Browsers["web"] != want {
		t.Fatalf("binding: %#v", m.Browsers)
	}
	encoded, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	var restored Manifest
	if err = json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if err = Validate(&restored); err != nil {
		t.Fatal(err)
	}
	if restored.Browsers["web"] != want {
		t.Fatal("canonical binding lost")
	}
}

func TestBrowserManifestNegativeFixtures(t *testing.T) {
	cases := map[string][2]string{
		"unknown type":      {"type: chromium-cdp", "type: webdriver"},
		"null type":         {"type: chromium-cdp", "type: null"},
		"missing type":      {"type: chromium-cdp, ", ""},
		"missing runtime":   {"runtime: chrome, ", ""},
		"null runtime":      {"runtime: chrome, ", "runtime: null, "},
		"unknown runtime":   {"runtime: chrome, ", "runtime: other, "},
		"missing port":      {", cdp_port: cdp", ""},
		"unknown port":      {"cdp_port: cdp", "cdp_port: unknown"},
		"null port":         {"cdp_port: cdp", "cdp_port: null"},
		"udp":               {"protocol: tcp", "protocol: udp"},
		"unknown field":     {"cdp_port: cdp", "cdp_port: cdp, profile: null"},
		"binding null":      {"{type: chromium-cdp, runtime: chrome, cdp_port: cdp}", "null"},
		"binding empty":     {"{type: chromium-cdp, runtime: chrome, cdp_port: cdp}", "{}"},
		"case collision":    {"components:\n", "  WEB: {type: chromium-cdp, runtime: chrome, cdp_port: cdp}\ncomponents:\n"},
		"duplicate runtime": {"components:\n", "  other: {type: chromium-cdp, runtime: chrome, cdp_port: cdp}\ncomponents:\n"},
		"duplicate key":     {"components:\n", "  web: {type: chromium-cdp, runtime: chrome, cdp_port: cdp}\ncomponents:\n"},
		"reserved name":     {"  web:", "  CON:"},
		"trailing dot":      {"  web:", "  web.:"},
		"invalid name":      {"  web:", "  ../web:"},
		"merged null type":  {"{type: chromium-cdp, runtime: chrome, cdp_port: cdp}", "{<<: &browser {type: null}, runtime: chrome, cdp_port: cdp}"},
		"alias collision":   {"  web: {type:", "  WEB: &browser {type:"},
	}
	for _, flag := range []string{"--headless=new", "--enable-automation", "--user-data-dir=${runtime_dir}/profile", "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=${port:cdp}"} {
		cases["missing "+flag] = [2]string{"'" + flag + "', ", ""}
		cases["duplicate "+flag] = [2]string{"'" + flag + "'", "'" + flag + "', '" + flag + "'"}
		cases["single dash "+flag] = [2]string{flag, strings.TrimPrefix(flag, "-")}
		cases["slash "+flag] = [2]string{flag, "/" + strings.TrimPrefix(flag, "--")}
		cases["uppercase "+flag] = [2]string{flag, strings.ToUpper(flag)}
	}
	for _, flag := range []string{"--", "--remote-debugging-pipe", "--profile-directory=Default", "--guest", "--incognito", "--user-data-dir=/outside", "--remote-debugging-address=0.0.0.0", "--remote-debugging-port=9222", "--user-data-dir", "--headless"} {
		cases["extra "+flag] = [2]string{"'about:blank'", "'" + flag + "', 'about:blank'"}
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(browserManifest, change[0], change[1], 1)
			if name == "alias collision" {
				text = strings.Replace(text, "components:\n", "  web: *browser\ncomponents:\n", 1)
			}
			if text == browserManifest {
				t.Fatal("fixture replacement failed")
			}
			if _, err := Parse([]byte(text)); err == nil {
				t.Fatal("invalid browser manifest accepted")
			}
		})
	}
	for _, value := range []string{"null", "{}", "[]", "''"} {
		text := strings.Replace(browserManifest, "browsers:\n  web: {type: chromium-cdp, runtime: chrome, cdp_port: cdp}", "browsers: "+value, 1)
		if _, err := Parse([]byte(text)); err == nil {
			t.Fatalf("accepted browsers: %s", value)
		}
	}
}

func TestBrowserRequiresProcessRuntime(t *testing.T) {
	m, err := Parse([]byte(browserManifest))
	if err != nil {
		t.Fatal(err)
	}
	m.Runtimes["chrome"] = Runtime{Type: "compose", Source: "app", ProjectDirectory: ".", Files: []string{"compose.yaml"}}
	m.Components["page"] = Component{Runtime: "chrome", ComposeServices: []string{"web"}}
	if err := Validate(m); err == nil {
		t.Fatal("browser accepted Compose runtime")
	}
}

func TestBrowserAbsentPreservesLegacyCanonicalShape(t *testing.T) {
	m, err := Parse([]byte(processManifest))
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"browsers"`) {
		t.Fatal("legacy manifest gained browsers field")
	}
	m.Browsers = map[string]Browser{}
	empty, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(empty) != string(b) {
		t.Fatal("empty additive binding changed legacy snapshot")
	}
	// A port alone never creates an implicit browser capability.
	if len(m.Browsers) != 0 {
		t.Fatal("implicit browser binding")
	}
}
