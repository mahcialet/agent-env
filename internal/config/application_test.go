package config

import (
	"strings"
	"testing"
)

const flutterManifest = `version: 1
sources:
  app: {repository: .}
runtimes:
  backend: {type: compose, source: app, files: [compose.yaml]}
  phone: {type: android-emulator, source: app, avd: Pixel}
applications:
  mobile-app:
    type: flutter-android
    source: app
    runtime: phone
    project_directory: mobile
    build:
      command: [flutter, build, apk, --debug]
      artifact: build/app.apk
    package: com.example.app
    activity: .MainActivity
    reverse:
      - {device_port: 8080, endpoint: api.http}
components:
  api:
    runtime: backend
    compose_services: [api]
    endpoints:
      http: {service: api, target: 8080}
  mobile:
    runtime: phone
    application: mobile-app
    depends_on: [api]
stacks:
  mobile: {roots: [mobile]}
`

func TestFlutterManifestContract(t *testing.T) {
	m, err := Parse([]byte(flutterManifest))
	if err != nil {
		t.Fatal(err)
	}
	if m.Applications["mobile-app"].Build.Command[0] != "flutter" || m.Components["mobile"].Application != "mobile-app" {
		t.Fatal("application not decoded")
	}
	for _, tc := range []struct{ name, from, to, want string }{
		{"unknown application field", "    package:", "    typo: true\n    package:", "field typo not found"},
		{"unknown build field", "      artifact:", "      typo: true\n      artifact:", "field typo not found"},
		{"wrong type", "type: flutter-android", "type: android-emulator", "unsupported type"},
		{"missing source", "    source: app", "    source: missing", "unknown source"},
		{"wrong runtime", "    runtime: phone", "    runtime: backend", "android-emulator"},
		{"missing runtime", "    runtime: phone", "    runtime: absent", "android-emulator"},
		{"escaped project", "project_directory: mobile", "project_directory: ../mobile", "escapes"},
		{"absolute project", "project_directory: mobile", "project_directory: C:/mobile", "source-relative"},
		{"shell command", "command: [flutter, build, apk, --debug]", "command: flutter build apk", "cannot unmarshal"},
		{"empty argv", "command: [flutter, build, apk, --debug]", "command: []", "argv"},
		{"empty executable", "command: [flutter, build, apk, --debug]", "command: ['', build]", "argv"},
		{"invalid timeout", "      artifact:", "      timeout: -1s\n      artifact:", "positive duration"},
		{"empty artifact", "artifact: build/app.apk", "artifact: ''", "nonempty .apk"},
		{"non apk", "artifact: build/app.apk", "artifact: build/app.aab", "nonempty .apk"},
		{"artifact parent", "artifact: build/app.apk", "artifact: ../app.apk", "escapes"},
		{"artifact windows", "artifact: build/app.apk", "artifact: C:/app.apk", "source-relative"},
		{"artifact backslash", "artifact: build/app.apk", `artifact: 'build\app.apk'`, "nonportable"},
		{"package missing", "package: com.example.app", "package: ''", "valid Android package"},
		{"package injection", "package: com.example.app", "package: '-x'", "valid Android package"},
		{"activity invalid", "activity: .MainActivity", "activity: com.example/.MainActivity", "valid Android activity"},
		{"activity remote expansion", "activity: .MainActivity", "activity: .Main$Activity", "valid Android activity"},
		{"activity unqualified", "activity: .MainActivity", "activity: MainActivity", "valid Android activity"},
		{"unknown binding field", "device_port: 8080", "unknown: 8080", "field unknown not found"},
		{"port low", "device_port: 8080", "device_port: 0", "1..65535"},
		{"port high", "device_port: 8080", "device_port: 65536", "1..65535"},
		{"endpoint absent", "endpoint: api.http", "endpoint: api.unknown", "unknown endpoint"},
		{"endpoint UDP", "target: 8080", "target: 8080, protocol: udp", "Compose TCP"},
		{"not dependency", "depends_on: [api]", "depends_on: []", "must belong to a dependency"},
		{"unknown application", "application: mobile-app", "application: missing", "unknown application"},
		{"mismatched component", "    application: mobile-app", "    application: mobile-app\n    compose_services: [api]", "cannot use Compose"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(flutterManifest, tc.from, tc.to, 1)
			if body == flutterManifest {
				t.Fatal("fixture unchanged")
			}
			_, err := Parse([]byte(body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}

func TestFlutterApplicationCollisionsAndDependencyClosure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{"duplicate package", func(m *Manifest) { a := m.Applications["mobile-app"]; a.Reverse = nil; m.Applications["second"] = a }, "duplicate package"},
		{"duplicate device port", func(m *Manifest) {
			a := m.Applications["mobile-app"]
			a.Package = "com.example.other"
			m.Applications["second"] = a
		}, "duplicate device_port"},
		{"duplicate in one application", func(m *Manifest) {
			a := m.Applications["mobile-app"]
			a.Reverse = append(a.Reverse, a.Reverse[0])
			m.Applications["mobile-app"] = a
		}, "duplicate device_port"},
		{"other runtime allowed", func(m *Manifest) {
			m.Runtimes["other"] = m.Runtimes["phone"]
			a := m.Applications["mobile-app"]
			a.Runtime = "other"
			m.Applications["second"] = a
		}, ""},
		{"transitive dependency", func(m *Manifest) {
			m.Components["middle"] = Component{Runtime: "phone", DependsOn: []string{"api"}}
			c := m.Components["mobile"]
			c.DependsOn = []string{"middle"}
			m.Components["mobile"] = c
		}, ""},
		{"second selector lacks dependency", func(m *Manifest) { c := m.Components["mobile"]; c.DependsOn = nil; m.Components["second"] = c }, "must belong to a dependency"},
		{"component runtime mismatch", func(m *Manifest) {
			m.Runtimes["other"] = m.Runtimes["phone"]
			c := m.Components["mobile"]
			c.Runtime = "other"
			m.Components["mobile"] = c
		}, "does not match"},
		{"NUL argv", func(m *Manifest) {
			a := m.Applications["mobile-app"]
			a.Build.Command = append(a.Build.Command, "bad\x00argument")
			m.Applications["mobile-app"] = a
		}, "NUL"},
		{"literal argv", func(m *Manifest) {
			a := m.Applications["mobile-app"]
			a.Build.Command = []string{"/sdk with spaces/日本語/flutter", "build", "apk", "--dart-define=NAME=$(literal)"}
			a.Build.Timeout = "1m"
			a.Activity = "com.example.MainActivity"
			m.Applications["mobile-app"] = a
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Parse([]byte(flutterManifest))
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(m)
			err = Validate(m)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q; got %v", tc.want, err)
			}
		})
	}
}

func TestApplicationEndpointReferenceWithDots(t *testing.T) {
	m, err := Parse([]byte(flutterManifest))
	if err != nil {
		t.Fatal(err)
	}
	c := m.Components["api"]
	c.Endpoints = map[string]Endpoint{"http.public": {Service: "api", Target: 8080}}
	m.Components["api"] = c
	cn, en, err := ResolveEndpointReference(m, "api.http.public")
	if err != nil || cn != "api" || en != "http.public" {
		t.Fatalf("%q %q %v", cn, en, err)
	}
	c.Endpoints = map[string]Endpoint{"public": {Service: "api", Target: 8080}}
	m.Components["api.http"] = c
	if _, _, err := ResolveEndpointReference(m, "api.http.public"); err == nil {
		t.Fatal("ambiguous dotted reference accepted")
	}
}
