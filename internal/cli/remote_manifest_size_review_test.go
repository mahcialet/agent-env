package cli

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/client"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/controlplane/server"
	controllerstore "github.com/mahcialet/agent-env/internal/controlplane/store"
	"github.com/mahcialet/agent-env/internal/remotesource"
)

func TestRemoteCreateNearManifestLimitRoundtrip(t *testing.T) {
	ctx := context.Background()
	repo := filepath.Join(t.TempDir(), "repository")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		if b, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("git: %v %s", e, b)
		}
	}
	git("init", repo)
	manifest := `version: 1
sources:
 app: {repository: ., default_ref: HEAD}
runtimes:
 api:
  type: process
  source: app
  working_directory: .
  command: [go, version]
  env: {FILL: "%s"}
components:
 api: {runtime: api}
stacks:
 local: {roots: [api]}
`
	write := func(padding string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(strings.Replace(manifest, "%s", padding, 1)), 0600); err != nil {
			t.Fatal(err)
		}
		git("-C", repo, "add", ".agent-env.yaml")
		git("-C", repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "manifest")
	}
	write("")
	cas, err := blobstore.New(filepath.Join(t.TempDir(), "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	initial, err := remotesource.Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	const manifestSize = (4 << 20) - 128
	write(strings.Repeat("a", manifestSize-len(initial.Manifest)))
	pkg, err := remotesource.Build(ctx, app.PlanOptions{Repository: repo, Stack: "local"}, cas)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Manifest) != manifestSize {
		t.Fatalf("canonical manifest length=%d want %d", len(pkg.Manifest), manifestSize)
	}

	db, err := controllerstore.Open(filepath.Join(t.TempDir(), "controller.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	controller := server.New(db, cas, "fixture")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "manifest transport fixture CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	if err = db.Enroll(ctx, protocol.Enrollment{Fingerprint: auth.Fingerprint(leaf), Role: "client"}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(controller.Handler())
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS13}
	ts.StartTLS()
	defer ts.Close()
	c, err := client.New(ts.URL, &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	workerTemplate := *template
	workerTemplate.SerialNumber = big.NewInt(2)
	workerTemplate.Subject = pkix.Name{CommonName: "manifest transport fixture worker"}
	workerTemplate.IsCA = false
	workerTemplate.KeyUsage = x509.KeyUsageDigitalSignature
	workerDER, err := x509.CreateCertificate(rand.Reader, &workerTemplate, leaf, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	workerLeaf, err := x509.ParseCertificate(workerDER)
	if err != nil {
		t.Fatal(err)
	}
	workerCert := tls.Certificate{Certificate: [][]byte{workerDER, der}, PrivateKey: key, Leaf: workerLeaf}
	if err = db.Enroll(ctx, protocol.Enrollment{Fingerprint: auth.Fingerprint(workerLeaf), Role: "worker", HostID: "fixture"}); err != nil {
		t.Fatal(err)
	}
	workerClient, err := client.New(ts.URL, &tls.Config{Certificates: []tls.Certificate{workerCert}, RootCAs: pool})
	if err != nil {
		t.Fatal(err)
	}
	defer workerClient.Close()
	w := protocol.WorkerIdentity{ControllerID: db.ID, HostID: "fixture", HostInstanceID: "home", Incarnation: "boot"}
	if _, err = workerClient.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: protocol.Version, ProductVersion: "fixture", OS: "linux", Arch: "amd64", Capabilities: pkg.RequiredCapabilities, Capacity: protocol.Capacity{MaxLeases: 1}}); err != nil {
		t.Fatal(err)
	}
	root := New(io.Discard, io.Discard)
	cmd, _, err := root.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.ParseFlags([]string{"--stack=local", "--ttl=1h", "--owner=fixture"}); err != nil {
		t.Fatal(err)
	}
	flags := remoteFlags{Host: "fixture"}
	op, err := flags.create(ctx, c, cmd, []string{repo}, "near-limit-create")
	if err != nil {
		t.Fatalf("near-limit create transport failed: %v", err)
	}
	if op.ID != "near-limit-create" {
		t.Fatalf("unexpected operation %+v", op)
	}
	got, err := c.GetOperation(ctx, op.ID)
	if err != nil {
		t.Fatalf("operation response exceeded metadata bound: %v", err)
	}
	pollResult, err := workerClient.Poll(ctx, protocol.PollRequest{WorkerIdentity: w})
	polled := pollResult.Operation
	if err != nil || polled == nil {
		t.Fatalf("poll: %v %+v", err, polled)
	}
	var envelope protocol.CreateRequest
	if err = json.Unmarshal(got.Payload, &envelope); err != nil {
		t.Fatal(err)
	}
	var compact remotesource.Package
	if err = json.Unmarshal(envelope.Package, &compact); err != nil {
		t.Fatal(err)
	}
	if len(compact.Manifest) != 0 && string(compact.Manifest) != "null" {
		t.Fatal("canonical manifest was duplicated or changed")
	}
	var receivedManifest, originalManifest any
	if err = json.Unmarshal(envelope.Manifest, &receivedManifest); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(pkg.Manifest, &originalManifest); err != nil {
		t.Fatal(err)
	}
	receivedBytes, _ := json.Marshal(receivedManifest)
	originalBytes, _ := json.Marshal(originalManifest)
	if !bytes.Equal(receivedBytes, originalBytes) {
		t.Fatal("canonical manifest meaning changed")
	}
	compact.Manifest = envelope.Manifest
	if err = remotesource.Validate(compact); err != nil {
		t.Fatalf("worker hydrated package invalid: %v", err)
	}
	wire, err := json.Marshal(protocol.PollResult{Operation: polled})
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) >= 8<<20 {
		t.Fatalf("poll response exceeds metadata envelope: %d", len(wire))
	}
	// Reconstruct the prior duplicate wire form to prove this fixture exercises
	// the actual 8 MiB boundary, not merely the compact struct representation.
	envelope.Package, _ = json.Marshal(compact)
	legacy, _ := json.Marshal(envelope)
	if len(legacy) <= 8<<20 {
		t.Fatalf("fixture does not reproduce old oversize request: %d", len(legacy))
	}
	if _, err = c.Create(ctx, envelope); err == nil {
		t.Fatal("oversized legacy request unexpectedly passed unchanged server bound")
	} else {
		var status *client.StatusError
		clientSizeRefusal := strings.Contains(err.Error(), "request exceeds metadata limit")
		serverSizeRefusal := errors.As(err, &status) && status.StatusCode == 400 && strings.Contains(err.Error(), "oversized request JSON")
		if !clientSizeRefusal && !serverSizeRefusal {
			t.Fatalf("legacy request failed outside unchanged size bound: %v", err)
		}
	}
	if !bytes.Equal(polled.Payload, got.Payload) {
		t.Fatal("stored/polled compact payload changed")
	}
}
