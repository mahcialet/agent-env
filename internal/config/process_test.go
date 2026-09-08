package config

import (
	"encoding/json"
	"strings"
	"testing"
)

const processManifest = `version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  api:
    type: process
    source: app
    working_directory: .
    command: [./bin/server, '--port=${port:http}', '${runtime_dir}/profile', '${lease_id}']
    ports:
      http: {protocol: tcp}
    env: {TOKEN: '${env:API_TOKEN}'}
components:
  api:
    runtime: api
    endpoints:
      http: {runtime_port: http}
    readiness: [{type: http, url: 'http://127.0.0.1:${endpoint:http}/health'}]
stacks:
  local: {roots: [api]}
`

func TestProcessManifestContract(t *testing.T) {
	m, err := Parse([]byte(processManifest))
	if err != nil {
		t.Fatal(err)
	}
	if m.Runtimes["api"].Ports["http"].Protocol != "tcp" || m.Components["api"].Endpoints["http"].RuntimePort != "http" {
		t.Fatal("lost process endpoint")
	}
	b, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	var restored Manifest
	if err = json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	if err = Validate(&restored); err != nil {
		t.Fatal(err)
	}
}

func TestProcessManifestNegativeFixtures(t *testing.T) {
	cases := map[string][2]string{
		"missing cwd":                     {"    working_directory: .\n", ""},
		"escaping cwd":                    {"working_directory: .", "working_directory: ../outside"},
		"absolute cwd":                    {"working_directory: .", "working_directory: /tmp"},
		"windows cwd":                     {"working_directory: .", `working_directory: C:\outside`},
		"cwd interpolation":               {"working_directory: .", "working_directory: '${runtime_dir}'"},
		"argv scalar":                     {"command: [./bin/server, '--port=${port:http}', '${runtime_dir}/profile', '${lease_id}']", "command: server"},
		"argv empty":                      {"command: [./bin/server, '--port=${port:http}', '${runtime_dir}/profile', '${lease_id}']", "command: []"},
		"argv0 reference":                 {"./bin/server", "'${env:SERVER}'"},
		"argv0 escape":                    {"./bin/server", "../server"},
		"argv0 absolute":                  {"./bin/server", "/bin/server"},
		"windows wrapper":                 {"./bin/server", "./bin/server.cmd"},
		"unknown ref":                     {"${port:http}", "${port:missing}"},
		"source ref":                      {"${runtime_dir}", "${source:app}"},
		"unterminated ref":                {"${runtime_dir}", "${runtime_dir"},
		"udp":                             {"protocol: tcp", "protocol: udp"},
		"missing protocol":                {"protocol: tcp", ""},
		"fixed port":                      {"protocol: tcp", "protocol: tcp, host_port: 8080"},
		"bad env key":                     {"TOKEN:", "'BAD=KEY':"},
		"case env collision":              {"TOKEN: '${env:API_TOKEN}'", "TOKEN: x, token: y"},
		"readiness missing endpoint":      {"${endpoint:http}", "${endpoint:unknown}"},
		"readiness endpoint scope":        {"${endpoint:http}", "${endpoint:api.http}"},
		"readiness unknown interpolation": {"${endpoint:http}", "${port:http}"},
		"unknown endpoint":                {"runtime_port: http", "runtime_port: other"},
		"compose service":                 {"runtime_port: http", "runtime_port: http, service: null"},
		"compose target":                  {"runtime_port: http", "runtime_port: http, target: 0"},
		"compose protocol":                {"runtime_port: http", "runtime_port: http, protocol: ''"},
		"compose services":                {"    runtime: api\n", "    runtime: api\n    compose_services: []\n"},
		"compose readiness":               {"type: http, url: 'http://127.0.0.1:${endpoint:http}/health'", "type: compose"},
		"reserved name":                   {"  api:", "  CON:"},
		"trailing dot":                    {"  api:", "  api.:"},
		"dot name":                        {"  api:", "  ..:"},
	}
	for _, field := range []string{"provider", "files", "project_directory", "avd"} {
		for _, value := range []string{"null", "''", "[]"} {
			cases[field+value] = [2]string{"    type: process\n", "    type: process\n    " + field + ": " + value + "\n"}
		}
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			b := strings.Replace(processManifest, change[0], change[1], 1)
			if b == processManifest {
				t.Fatal("invalid fixture")
			}
			if _, err := Parse([]byte(b)); err == nil {
				t.Fatal("accepted invalid process manifest")
			}
		})
	}
}

func TestProcessPresenceRejectsYAMLMergeAndAliases(t *testing.T) {
	b := strings.Replace(processManifest, "    type: process", "    <<: &unsupported {files: null}\n    type: process", 1)
	if _, err := Parse([]byte(b)); err == nil {
		t.Fatal("merged incompatible field accepted")
	}
	b = strings.Replace(processManifest, "  api:\n    type: process", "  API: &runtime\n    type: process", 1)
	b = strings.Replace(b, "components:\n", "  api: *runtime\ncomponents:\n", 1)
	if _, err := Parse([]byte(b)); err == nil {
		t.Fatal("case-colliding runtime alias accepted")
	}
}

func TestProcessFieldsDoNotChangeLegacyCanonicalShape(t *testing.T) {
	r := Runtime{Type: "compose", Source: "app", ProjectDirectory: ".", Files: []string{"compose.yaml"}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"compose","source":"app","project_directory":".","files":["compose.yaml"]}`
	if string(b) != want {
		t.Fatalf("legacy canonical shape changed: %s", b)
	}
	for _, field := range []string{"working_directory", "command", "env", "ports"} {
		b := strings.Replace(processManifest, "    type: process\n", "    type: compose\n    files: [compose.yaml]\n", 1)
		// A null process-only field must not silently become an unused Compose default.
		start := strings.Index(b, "    working_directory:")
		end := strings.Index(b, "components:")
		b = b[:start] + "    " + field + ": null\n" + b[end:]
		if _, err := Parse([]byte(b)); err == nil {
			t.Fatalf("compose accepted process field %s", field)
		}
	}
}
