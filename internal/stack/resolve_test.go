package stack

import (
	"os"
	"reflect"
	"testing"

	"github.com/mahcialet/agent-env/internal/config"
)

func TestClosure(t *testing.T) {
	b, e := os.ReadFile("../../testdata/manifests/api-dashboard.yaml")
	if e != nil {
		t.Fatal(e)
	}
	m, e := config.Parse(b)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name string
		want []string
	}{{"api", []string{"api"}}, {"dashboard", []string{"api", "dashboard"}}, {"full", []string{"api", "dashboard"}}} {
		got, e := Resolve(m, tc.name)
		if e != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s got %v, %v", tc.name, got, e)
		}
	}
	if _, e := Resolve(m, "missing"); e == nil {
		t.Fatal("unknown stack accepted")
	}
	c := m.Components["api"]
	c.DependsOn = []string{"dashboard"}
	m.Components["api"] = c
	if _, e := Resolve(m, "api"); e == nil {
		t.Fatal("cycle accepted")
	}
}

func TestDeterminismAndNoMutation(t *testing.T) {
	m := &config.Manifest{Version: 1, Sources: map[string]config.Source{"src": {Repository: "."}}, Runtimes: map[string]config.Runtime{"rt": {Type: "compose", Source: "src", Files: []string{"compose.yaml"}}}, Components: map[string]config.Component{}, Stacks: map[string]config.Stack{"all": {Roots: []string{"z", "b"}}}}
	for _, n := range []string{"z", "a", "b"} {
		m.Components[n] = config.Component{Runtime: "rt", ComposeServices: []string{n}}
	}
	c := m.Components["z"]
	c.DependsOn = []string{"b", "a"}
	m.Components["z"] = c
	before := config.Digest(m)
	want := []string{"b", "a", "z"}
	for i := 0; i < 20; i++ {
		got, e := Resolve(m, "all")
		if e != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v %v", got, e)
		}
	}
	if config.Digest(m) != before {
		t.Fatal("resolver mutated manifest")
	}
}
