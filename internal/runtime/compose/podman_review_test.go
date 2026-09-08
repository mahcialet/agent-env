package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

const podmanReviewInfo = `{"host":{"hostname":"fixture","os":"linux","arch":"amd64"},"store":{"graphRoot":"/graph","runRoot":"/run","volumePath":"/volumes"},"version":{"version":"5.4.2"}}`

func TestPodmanRemoteInspectPreservesUDPAndChecksTCP(t *testing.T) {
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = udp.Close() })
	_, udpPort, _ := net.SplitHostPort(udp.LocalAddr().String())
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tcp.Close() })
	_, tcpPort, _ := net.SplitHostPort(tcp.Addr().String())
	for _, scenario := range []struct {
		name, tcpPort string
		ready         bool
	}{
		{"udp_only", "", true}, {"mixed_reachable", tcpPort, true},
		// Port zero cannot be a listening TCP endpoint; no close/rebind port race.
		{"mixed_unreachable", "0", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			p := testPodmanIdentity(t)
			p.URL = "ssh://fixture@machine/run/user/1000/podman/podman.sock"
			ports := map[string]any{"53/udp": []map[string]string{{"HostIP": "127.0.0.1", "HostPort": udpPort}}}
			if scenario.tcpPort != "" {
				ports["80/tcp"] = []map[string]string{{"HostIP": "127.0.0.1", "HostPort": scenario.tcpPort}}
			}
			data, err := json.Marshal([]map[string]any{{"Id": "container", "Config": map[string]any{"Labels": map[string]string{"io.podman.compose.project": "ae_fixture", "io.podman.compose.service": "web", leaseLabel: "lease", runtimeLabel: "runtime"}}, "State": map[string]any{"Running": true, "Status": "running"}, "NetworkSettings": map[string]any{"Ports": ports}}})
			if err != nil {
				t.Fatal(err)
			}
			c := podmanClient{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
				flags := p.flags()
				if q.Name != p.Executable || len(q.Args) < len(flags) || !reflect.DeepEqual(q.Args[:len(flags)], flags) {
					t.Fatalf("unpinned request: %#v", q)
				}
				args := q.Args[len(flags):]
				switch {
				case reflect.DeepEqual(args, []string{"info", "--format", "json"}):
					return execx.Result{Stdout: podmanReviewInfo}, nil
				case reflect.DeepEqual(args, []string{"ps", "--all", "--quiet", "--filter", "label=io.podman.compose.project=ae_fixture"}):
					return execx.Result{Stdout: "container"}, nil
				case reflect.DeepEqual(args, []string{"inspect", "--type", "container", "container"}):
					return execx.Result{Stdout: string(data)}, nil
				case reflect.DeepEqual(args, []string{"network", "ls", "--quiet", "--filter", "label=io.podman.compose.project=ae_fixture"}), reflect.DeepEqual(args, []string{"volume", "ls", "--quiet", "--filter", "label=io.podman.compose.project=ae_fixture"}):
					return execx.Result{}, nil
				default:
					t.Fatalf("unexpected request: %#v", q)
					return execx.Result{}, nil
				}
			})}
			p.Fingerprint, _, err = c.fingerprint(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			o, err := c.Inspect(context.Background(), domain.Runtime{Name: "runtime", LeaseID: "lease", Project: "ae_fixture", Context: p.encode(), Services: []string{"web"}})
			if err != nil {
				t.Fatal(err)
			}
			if !o.Exists || o.Ready != scenario.ready || o.Endpoints["web/53/udp"] != udp.LocalAddr().String() {
				t.Fatalf("UDP/readiness lost: %#v", o)
			}
			if scenario.tcpPort != "" && scenario.ready && o.Endpoints["web/80/tcp"] != tcp.Addr().String() {
				t.Fatalf("reachable TCP lost: %#v", o)
			}
			if !scenario.ready && (o.Endpoints["web/80/tcp"] != "" || len(o.Diagnostics) != 1 || !strings.Contains(o.Diagnostics[0], "web/80/tcp")) {
				t.Fatalf("unreachable TCP accepted: %#v", o)
			}
			if scenario.ready && len(o.Diagnostics) != 0 {
				t.Fatal(o.Diagnostics)
			}
		})
	}
}

func TestPodmanInventoryWithoutComposeDiscoversNativeOrphans(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "podman")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := os.WriteFile(binary, []byte("fixture executable"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	t.Setenv("CONTAINER_CONNECTION", "")
	t.Setenv("CONTAINER_HOST", "unix:///fixture/podman.sock")
	t.Setenv("CONTAINER_SSHKEY", "")
	p := podmanIdentity{Version: 1, Executable: binary, URL: "unix:///fixture/podman.sock"}
	fault := ""
	c := Client{Runner: podmanRunnerFunc(func(_ context.Context, q execx.Command) (execx.Result, error) {
		flags := p.flags()
		if q.Name != binary || len(q.Args) < len(flags) || !reflect.DeepEqual(q.Args[:len(flags)], flags) {
			return execx.Result{}, fmt.Errorf("inventory invoked unpinned/non-native executable: %#v", q)
		}
		args := q.Args[len(flags):]
		if reflect.DeepEqual(args, []string{"info", "--format", "json"}) {
			return execx.Result{Stdout: podmanReviewInfo}, nil
		}
		for _, kind := range []string{"container", "network", "volume"} {
			list := []string{kind, "ls", "--quiet"}
			inspect := []string{kind, "inspect", kind + "-id"}
			if kind == "container" {
				list = []string{"ps", "--all", "--quiet"}
				inspect = []string{"inspect", "--type", "container", "container-id"}
			}
			for _, label := range []string{leaseLabel, "io.podman.compose.project"} {
				if reflect.DeepEqual(args, append(append([]string{}, list...), "--filter", "label="+label)) {
					if fault == "network failure" && kind == "network" {
						return execx.Result{ExitCode: 125, Stderr: "network unavailable"}, nil
					}
					return execx.Result{Stdout: kind + "-id"}, nil
				}
			}
			if reflect.DeepEqual(args, inspect) {
				labels := map[string]string{"io.podman.compose.project": "ae_orphan", leaseLabel: "gone-lease", runtimeLabel: "runtime"}
				if fault == "conflicting labels" {
					labels[projectLabel] = "ae_other"
				}
				row := map[string]any{"Id": kind + "-id", "Name": kind + "-id", "Labels": labels}
				if kind == "container" {
					labels["io.podman.compose.service"] = "web"
					delete(row, "Labels")
					row["Config"] = map[string]any{"Labels": labels}
				}
				data, err := json.Marshal([]map[string]any{row})
				if err != nil {
					return execx.Result{}, err
				}
				return execx.Result{Stdout: string(data)}, nil
			}
		}
		return execx.Result{}, fmt.Errorf("unexpected inventory command: %#v", q)
	})}
	details, err := c.InventoryDoctorFor(context.Background(), domain.ComposeProviderPodman)
	if err != nil {
		t.Fatal(err)
	}
	p, err = decodePodmanIdentity(details["context"])
	if err != nil || p.ComposeExecutable != "" {
		t.Fatalf("inventory doctor requires compose: %#v %v", p, err)
	}
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing_compose", true: "stale_recorded_compose"}[stale], func(t *testing.T) {
			identity := p
			if stale {
				identity.ComposeExecutable = filepath.Join(directory, "removed-podman-compose")
			}
			items, err := c.InventoryFor(context.Background(), domain.ComposeProviderPodman, identity.encode())
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 3 {
				t.Fatalf("lost native orphan resources: %#v", items)
			}
			seen := map[string]bool{}
			for _, item := range items {
				seen[item.Kind] = true
				if item.ExternalID != item.Kind+"-id" || item.LeaseID != "gone-lease" || item.Runtime != "runtime" || item.Metadata["project"] != "ae_orphan" || item.Metadata["context"] != identity.encode() || item.Metadata["provider"] != "podman-compose" || item.Metadata["ownership"] != "labelled" || !strings.HasPrefix(item.ID, "podman-compose:") {
					t.Fatalf("lost identity: %#v", item)
				}
			}
			if !seen["container"] || !seen["network"] || !seen["volume"] {
				t.Fatal(seen)
			}
		})
	}
	for _, scenario := range []struct {
		name, wantError string
		retained        int
	}{
		{"conflicting labels", "conflicting", 0},
		{"network failure", "exit 125", 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fault = scenario.name
			defer func() { fault = "" }()
			items, err := c.InventoryFor(context.Background(), domain.ComposeProviderPodman, p.encode())
			if err == nil || !strings.Contains(err.Error(), scenario.wantError) || len(items) != scenario.retained {
				t.Fatalf("inventory lost error or partial evidence: %v, %#v", err, items)
			}
		})
	}
}
