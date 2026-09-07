package policy

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectedClosureAndHazards(t *testing.T) {
	root := t.TempDir()
	c := Config{Services: map[string]Service{"api": {DependsOn: map[string]json.RawMessage{"db": nil}}, "db": {}, "unused": {Privileged: true}}}
	d, err := Defaults().Evaluate(c, []string{"api"}, []string{root})
	if err != nil || len(d) != 0 {
		t.Fatalf("unselected service rejected: %v %v", d, err)
	}
	c.Services["db"] = Service{ContainerName: "global", Privileged: true, NetworkMode: "host", Ports: []Port{{Published: "5432"}}, Volumes: []Mount{{Type: "bind", Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"}}}
	d, err = Defaults().Evaluate(c, []string{"api"}, []string{root})
	if err != nil || len(d) != 6 {
		t.Fatalf("expected six policy findings: %v %v", d, err)
	}
	c.Services["db"] = Service{Volumes: []Mount{{Type: "bind", Source: filepath.Join(root, "config"), Target: "/config"}}, Ports: []Port{{Published: "0"}}}
	d, err = Defaults().Evaluate(c, []string{"api"}, []string{root})
	if err != nil || len(d) != 0 {
		t.Fatalf("safe isolated config rejected %v %v", d, err)
	}
}

func TestTTLAndUnknownService(t *testing.T) {
	p := Defaults()
	if got, err := p.TTL(0); err != nil || got != 4*time.Hour {
		t.Fatalf("%v %v", got, err)
	}
	for _, ttl := range []time.Duration{-time.Second, 25 * time.Hour} {
		if _, err := p.TTL(ttl); err == nil {
			t.Fatal("invalid TTL accepted")
		}
	}
	if _, err := Services(Config{}, []string{"missing"}); err == nil {
		t.Fatal("unknown service accepted")
	}
}
