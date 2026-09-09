//go:build multihostintegration

package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/worker"
)

type multiHostProcess struct {
	cmd     *exec.Cmd
	log     *os.File
	stopped bool
}

func (p *multiHostProcess) stop() {
	if !p.stopped {
		p.stopped = true
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		_ = p.log.Close()
	}
}

// This fixture uses independently built native CLI processes and real TLS sockets.
// Its two worker roots share one physical machine; it does not prove VM/network
// topology behavior or native execution on an OS where it has not been run.
func TestMultiHostNativeCLI(t *testing.T) {
	base, createErr := os.MkdirTemp("", "agent-env-multihost-native-")
	if createErr != nil {
		t.Fatal(createErr)
	}
	retain := false
	t.Cleanup(func() {
		if retain {
			t.Logf("retained uncertain fixture state: %s", base)
		} else if err := os.RemoveAll(base); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	ctx := context.Background()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_COUNT", "0")
	root, e := filepath.Abs(filepath.Join("..", ".."))
	if e != nil {
		t.Fatal(e)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	binary := filepath.Join(base, "agent-env"+suffix)
	run := func(dir, name string, args ...string) string {
		t.Helper()
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(cctx, name, args...)
		cmd.Dir = dir
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, e, out)
		}
		return string(out)
	}
	run(root, "go", "build", "-o", binary, "./cmd/agent-env")
	ca, credentials := multiHostCertificates(t, base)
	controllerHome := filepath.Join(base, "controller")
	clientHome := filepath.Join(base, "client")
	workerHomes := []string{filepath.Join(base, "worker-a"), filepath.Join(base, "worker-b")}
	for _, home := range append([]string{controllerHome, clientHome}, workerHomes...) {
		if e = os.MkdirAll(home, 0700); e != nil {
			t.Fatal(e)
		}
	}
	env := func(home string) []string {
		result := []string{}
		for _, v := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(v), "AGENT_ENV_HOME=") {
				result = append(result, v)
			}
		}
		return append(result, "AGENT_ENV_HOME="+home)
	}
	invoke := func(home string, args ...string) (json.RawMessage, error) {
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(cctx, binary, append([]string{"--output=json", "--owner=multihost-fixture"}, args...)...)
		cmd.Env = env(home)
		var out, errOut bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errOut
		e := cmd.Run()
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		decodeErr := json.Unmarshal(out.Bytes(), &envelope)
		if e != nil {
			return envelope.Data, fmt.Errorf("CLI %v: %w\n%s\n%s", args, e, errOut.String(), out.String())
		}
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", out.String(), decodeErr)
		}
		return envelope.Data, nil
	}
	must := func(home string, args ...string) json.RawMessage {
		t.Helper()
		raw, e := invoke(home, args...)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	for _, identity := range []string{"client", "worker-a", "worker-b"} {
		args := []string{"control-plane", "enroll", "--certificate", credentials[identity][0], "--role", protocol.RoleWorker, "--host-id", identity}
		if identity == "client" {
			args = []string{"control-plane", "enroll", "--certificate", credentials[identity][0], "--role", protocol.RoleClient}
		}
		must(controllerHome, args...)
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	url := "https://" + address
	tlsArgs := func(identity string) []string {
		return []string{"--tls-ca", ca, "--tls-cert", credentials[identity][0], "--tls-key", credentials[identity][1]}
	}
	remoteArgs := append(tlsArgs("client"), "--controller", url, "--wait", "90s")
	remote := func(args ...string) json.RawMessage {
		t.Helper()
		return must(clientHome, append(append([]string{}, remoteArgs...), args...)...)
	}
	processes := []*multiHostProcess{}
	t.Cleanup(func() {
		for i := len(processes) - 1; i >= 0; i-- {
			p := processes[i]
			p.stop()
			if t.Failed() {
				content, _ := os.ReadFile(p.log.Name())
				t.Logf("service %s output:\n%s", filepath.Base(p.log.Name()), content)
			}
		}
	})
	start := func(home, label string, args ...string) *multiHostProcess {
		t.Helper()
		log, e := os.CreateTemp(base, label+"-*.log")
		if e != nil {
			t.Fatal(e)
		}
		cmd := exec.Command(binary, args...)
		cmd.Env = env(home)
		cmd.Stdout = log
		cmd.Stderr = log
		if e = cmd.Start(); e != nil {
			t.Fatal(e)
		}
		p := &multiHostProcess{cmd: cmd, log: log}
		processes = append(processes, p)
		return p
	}
	startController := func() *multiHostProcess {
		return start(controllerHome, "controller", append(tlsArgs("server"), "control-plane", "serve", "--listen", address)...)
	}
	controller := startController()
	startWorker := func(index int) *multiHostProcess {
		identity := []string{"worker-a", "worker-b"}[index]
		return start(workerHomes[index], identity, append(tlsArgs(identity), "--controller", url, "worker", "serve", "--host-id", identity, "--max-leases", "2", "--android-slots", "0")...)
	}
	workers := []*multiHostProcess{startWorker(0), startWorker(1)}
	waitHosts := func() []protocol.Host {
		t.Helper()
		deadline := time.Now().Add(90 * time.Second)
		var last error
		for time.Now().Before(deadline) {
			raw, e := invoke(clientHome, append(append([]string{}, remoteArgs...), "hosts", "list")...)
			last = e
			if e == nil {
				var hosts []protocol.Host
				if e = json.Unmarshal(raw, &hosts); e == nil && len(hosts) == 2 && hosts[0].Online && hosts[1].Online {
					return hosts
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("two TLS workers did not register: %v", last)
		return nil
	}
	hosts := waitHosts()
	for _, identity := range []string{"worker-a", "unenrolled"} {
		args := append(tlsArgs(identity), "--controller", url, "list")
		if _, err := invoke(clientHome, args...); err == nil {
			t.Fatalf("%s certificate gained client role", identity)
		}
	}

	controllerID := hosts[0].ControllerID
	instances := map[string]string{}
	for _, host := range hosts {
		instances[host.HostID] = host.HostInstanceID
		if host.ControllerID != controllerID {
			t.Fatal("workers registered with different controller IDs")
		}
	}
	// Cleanup is remote and assignment-authorized while services remain available.
	// Register after service cleanup so this executes first, including failed tests.
	leaseIDs := []string{}
	t.Cleanup(func() {
		for _, id := range leaseIDs {
			if _, e := invoke(clientHome, append(append([]string{}, remoteArgs...), "destroy", id)...); e != nil {
				retain = true
				t.Errorf("remote fixture cleanup %s: %v", id, e)
			}
		}
	})
	repo := multiHostRepository(t, base, suffix, run)
	if _, e = invoke(clientHome, append(append([]string{}, remoteArgs...), "create", repo, "--host", "not-enrolled", "--stack", "review")...); e == nil {
		t.Fatal("explicit unenrolled host accepted")
	}
	create := func(host string) domain.Lease {
		t.Helper()
		args := []string{"create", repo, "--stack", "review"}
		if host != "" {
			args = append(args, "--host", host)
		}
		var op protocol.Operation
		raw, invokeErr := invoke(clientHome, append(append([]string{}, remoteArgs...), args...)...)
		if len(raw) > 0 {
			if e = json.Unmarshal(raw, &op); e != nil {
				t.Fatal(e)
			}
		}
		if op.LeaseID != "" {
			leaseIDs = append(leaseIDs, op.LeaseID)
		}
		if invokeErr != nil {
			t.Fatal(invokeErr)
		}
		if op.Result == nil || op.Result.State != "completed" {
			t.Fatalf("create result: %+v", op)
		}
		var response worker.Response
		if e = json.Unmarshal(op.Result.Payload, &response); e != nil {
			t.Fatal(e)
		}
		if response.Lease == nil {
			t.Fatalf("missing worker lease: %s", op.Result.Payload)
		}
		l := *response.Lease
		if l.ID != op.LeaseID || l.Management == nil || l.Management.HostID != op.HostID || l.Management.HostInstanceID != op.HostInstanceID || l.Management.ControllerID != controllerID || int64(l.Management.AssignmentEpoch) != op.Epoch || l.Observed != "ready" || len(l.Runtimes) != 2 || len(l.Components) != 2 || len(l.Sources) != 1 {
			t.Fatalf("whole-lease placement/identity: %+v", l)
		}
		for _, r := range l.Runtimes {
			if r.LeaseID != l.ID || r.Process == nil || !strings.Contains(r.Process.StateDirectory, l.ID) {
				t.Fatalf("runtime identity separated from lease: %+v", r)
			}
		}
		if response.EndpointScope != "worker-local" {
			t.Fatal("worker endpoint scope missing")
		}
		return l
	}
	first := create("worker-a")
	remote("hosts", "drain", "worker-a")
	if _, e = invoke(clientHome, append(append([]string{}, remoteArgs...), "create", repo, "--host", "worker-a", "--stack", "review")...); e == nil {
		t.Fatal("explicit placement bypassed drain")
	}
	second := create("")
	if second.Management.HostID != "worker-b" {
		t.Fatalf("drained host selected: %+v", second.Management)
	}
	var leases []protocol.Lease
	if e = json.Unmarshal(remote("list"), &leases); e != nil || len(leases) != 2 {
		t.Fatalf("remote list: %+v %v", leases, e)
	}
	if first.Sources[0].WorktreePath == second.Sources[0].WorktreePath {
		t.Fatal("worker worktrees overlap")
	}
	ports := map[int]bool{}
	for _, l := range []domain.Lease{first, second} {
		for _, r := range l.Runtimes {
			port := r.Process.Ports["http"]
			if port == 0 || ports[port] {
				t.Fatalf("runtime endpoint collision: %d", port)
			}
			ports[port] = true
			multiHostProbe(t, port)
		}
	}
	if _, e = invoke(workerHomes[0], "destroy", first.ID, "--force"); e == nil {
		t.Fatal("local force bypassed controller ownership")
	}
	// An outage must leave native workloads intact; restarting the same state root
	// must preserve authority, assignments and worker identities rather than recreate.
	controller.stop()
	for _, r := range first.Runtimes {
		multiHostProbe(t, r.Process.Ports["http"])
	}
	controller = startController()
	_ = controller
	after := waitHosts()
	for _, host := range after {
		if host.ControllerID != controllerID || host.HostInstanceID != instances[host.HostID] {
			t.Fatalf("controller restart replaced identity: %+v", host)
		}
	}
	workers[1].stop()
	for _, r := range second.Runtimes {
		multiHostProbe(t, r.Process.Ports["http"])
	}
	workers[1] = startWorker(1)
	after = waitHosts()
	for _, host := range after {
		if host.HostInstanceID != instances[host.HostID] {
			t.Fatalf("worker restart changed persistent instance: %+v", host)
		}
	}
	var reconciled protocol.Operation
	if e = json.Unmarshal(remote("reconcile", second.ID), &reconciled); e != nil || reconciled.Result == nil || reconciled.Result.State != "completed" {
		t.Fatalf("restart reconciliation: %+v %v", reconciled, e)
	}
	var recovered worker.Response
	if e = json.Unmarshal(reconciled.Result.Payload, &recovered); e != nil || recovered.Lease == nil {
		t.Fatalf("reconcile lease: %s %v", reconciled.Result.Payload, e)
	}
	for i, r := range recovered.Lease.Runtimes {
		if r.Process.ProcessID != second.Runtimes[i].Process.ProcessID || r.Process.ProcessStart != second.Runtimes[i].Process.ProcessStart {
			t.Fatal("worker restart relaunched workload")
		}
	}
	var destroyed protocol.Operation
	if e = json.Unmarshal(remote("destroy", first.ID), &destroyed); e != nil || destroyed.Result == nil || !destroyed.Result.CleanupConfirmed {
		t.Fatalf("destroy lacks cleanup proof: %+v %v", destroyed, e)
	}
	leaseIDs = leaseIDs[1:]
	for _, r := range second.Runtimes {
		multiHostProbe(t, r.Process.Ports["http"])
	}
	var shown protocol.Lease
	if e = json.Unmarshal(remote("show", second.ID), &shown); e != nil || shown.HostID != second.Management.HostID || shown.Epoch != int64(second.Management.AssignmentEpoch) {
		t.Fatalf("destroy affected other assignment: %+v %v", shown, e)
	}
	t.Logf("native %s/%s: real TLS controller/client/two worker processes, committed two-runtime leases, placement/drain/isolation/local refusal/outage/controller+worker restart/cleanup passed; same physical host only", runtime.GOOS, runtime.GOARCH)
}

func multiHostProbe(t *testing.T, port int) {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	response, e := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 256))
	if response.StatusCode != 200 || !strings.HasPrefix(string(body), "fixture pid=") {
		t.Fatalf("workload response: %d %q", response.StatusCode, body)
	}
}

func multiHostRepository(t *testing.T, base, suffix string, run func(string, string, ...string) string) string {
	t.Helper()
	repo := filepath.Join(base, "committed repository 日本語")
	if e := os.Mkdir(repo, 0700); e != nil {
		t.Fatal(e)
	}
	source := filepath.Join(base, "fixture.go")
	program := `package main
import("flag";"fmt";"net/http";"os";"time")
func main(){port:=flag.Int("port",0,"");flag.Parse();go func(){time.Sleep(8*time.Minute);os.Exit(9)}();http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){fmt.Fprintf(w,"fixture pid=%d",os.Getpid())});if e:=http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d",*port),nil);e!=nil{panic(e)}}
`
	if e := os.WriteFile(source, []byte(program), 0600); e != nil {
		t.Fatal(e)
	}
	helper := "fixture" + suffix
	run(base, "go", "build", "-o", filepath.Join(repo, helper), source)
	command, _ := json.Marshal("./" + helper)
	manifest := fmt.Sprintf(`version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  first:
    type: process
    source: self
    working_directory: .
    command: [%s, "--port", "${port:http}"]
    ports: {http: {protocol: tcp}}
  second:
    type: process
    source: self
    working_directory: .
    command: [%s, "--port", "${port:http}"]
    ports: {http: {protocol: tcp}}
components:
  first:
    runtime: first
    endpoints: {http: {runtime_port: http}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:http}/", timeout: 30s}
  second:
    runtime: second
    endpoints: {http: {runtime_port: http}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:http}/", timeout: 30s}
stacks:
  review: {roots: [first, second]}
`, command, command)
	if e := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); e != nil {
		t.Fatal(e)
	}
	run(repo, "git", "-c", "init.templateDir=", "init", "-b", "main")
	run(repo, "git", "add", ".agent-env.yaml", helper)
	run(repo, "git", "update-index", "--chmod=+x", helper)
	run(repo, "git", "-c", "commit.gpgsign=false", "-c", "user.name=Multi Host Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "Committed multi-host fixture")
	return repo
}

func multiHostCertificates(t *testing.T, base string) (string, map[string][2]string) {
	t.Helper()
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	caTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Fixture CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, e := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, pub, key)
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(base, "ca.pem")
	if e = os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); e != nil {
		t.Fatal(e)
	}
	result := map[string][2]string{}
	for i, name := range []string{"server", "client", "worker-a", "worker-b", "unenrolled"} {
		pub, leaf, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(int64(i + 2)), Subject: pkix.Name{CommonName: name}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
		if name == "server" {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		}
		der, e = x509.CreateCertificate(rand.Reader, template, caTemplate, pub, key)
		if e != nil {
			t.Fatal(e)
		}
		private, e := x509.MarshalPKCS8PrivateKey(leaf)
		if e != nil {
			t.Fatal(e)
		}
		certPath, keyPath := filepath.Join(base, name+".pem"), filepath.Join(base, name+".key")
		if e = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600); e != nil {
			t.Fatal(e)
		}
		result[name] = [2]string{certPath, keyPath}
	}
	return ca, result
}
