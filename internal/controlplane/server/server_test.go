package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/client"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/controlplane/store"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type certs struct {
	ca   *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func certificateAuthority(t *testing.T) certs {
	t.Helper()
	k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, e := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if e != nil {
		t.Fatal(e)
	}
	ca, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	return certs{ca: ca, key: k, pool: pool}
}
func (c certs) leaf(t *testing.T, n int, server bool) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(int64(n)), Subject: pkix.Name{CommonName: "test peer"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	der, e := x509.CreateCertificate(rand.Reader, tmpl, c.ca, &k.PublicKey, c.key)
	if e != nil {
		t.Fatal(e)
	}
	leaf, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k, Leaf: leaf}, leaf
}

type fixture struct {
	s             *Server
	url           string
	tls           *tls.Config
	ca            certs
	admin, worker *client.Client
	workerCert    tls.Certificate
	store         *store.Store
	ctx           context.Context
}

func serveFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	db, e := store.Open(filepath.Join(root, "controller.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	blobs, e := blobstore.New(filepath.Join(root, "blobs"))
	if e != nil {
		t.Fatal(e)
	}
	ca := certificateAuthority(t)
	serverCert, _ := ca.leaf(t, 2, true)
	adminCert, adminLeaf := ca.leaf(t, 3, false)
	workerCert, workerLeaf := ca.leaf(t, 4, false)
	for _, en := range []protocol.Enrollment{{Fingerprint: auth.Fingerprint(adminLeaf), Role: "client"}, {Fingerprint: auth.Fingerprint(workerLeaf), Role: "worker", HostID: "host-a"}} {
		if e = db.Enroll(context.Background(), en); e != nil {
			t.Fatal(e)
		}
	}
	s := New(db, blobs, "test")
	ctx, cancel := context.WithCancel(context.Background())
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientCAs: ca.pool}
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx, ln, cfg) }()
	url := "https://" + ln.Addr().String()
	admin, e := client.New(url, &tls.Config{Certificates: []tls.Certificate{adminCert}, RootCAs: ca.pool})
	if e != nil {
		t.Fatal(e)
	}
	worker, e := client.New(url, &tls.Config{Certificates: []tls.Certificate{workerCert}, RootCAs: ca.pool})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		admin.Close()
		worker.Close()
		cancel()
		ln.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("server did not stop")
		}
	})
	return fixture{s: s, url: url, tls: cfg, ca: ca, admin: admin, worker: worker, workerCert: workerCert, store: db, ctx: ctx}
}
func statusCode(err error, n int) bool {
	var e *client.StatusError
	return errors.As(err, &e) && e.StatusCode == n
}
func TestMTLSEnrollmentRolesAndControllerIdentity(t *testing.T) {
	f := serveFixture(t)
	ctx := context.Background()
	info, e := f.admin.Info(ctx)
	if e != nil || info.ControllerID != f.store.ID {
		t.Fatalf("info %+v %v", info, e)
	}
	unauthorized, _ := f.ca.leaf(t, 5, false)
	u, e := client.New(f.url, &tls.Config{Certificates: []tls.Certificate{unauthorized}, RootCAs: f.ca.pool})
	if e != nil {
		t.Fatal(e)
	}
	defer u.Close()
	if _, e = u.Info(ctx); !statusCode(e, 403) {
		t.Fatalf("unenrolled trusted certificate accepted: %v", e)
	}
	wrongCA := certificateAuthority(t)
	wrong, _ := wrongCA.leaf(t, 6, false)
	bad, e := client.New(f.url, &tls.Config{Certificates: []tls.Certificate{wrong}, RootCAs: f.ca.pool})
	if e != nil {
		t.Fatal(e)
	}
	defer bad.Close()
	if _, e = bad.Info(ctx); e == nil {
		t.Fatal("untrusted certificate accepted")
	}
	req := protocol.RegisterRequest{WorkerIdentity: protocol.WorkerIdentity{ControllerID: info.ControllerID, HostID: "host-a", HostInstanceID: "instance", Incarnation: "boot"}, ProtocolVersion: 1, ProductVersion: "test", OS: "linux", Arch: "amd64", Capabilities: []string{"process"}, Capacity: protocol.Capacity{MaxLeases: 1}}
	if _, e = f.admin.Register(ctx, req); !statusCode(e, 403) {
		t.Fatalf("client registered worker: %v", e)
	}
	badReq := req
	badReq.HostID = "host-b"
	if _, e = f.worker.Register(ctx, badReq); !statusCode(e, 403) {
		t.Fatalf("certificate adopted another host: %v", e)
	}
	badReq = req
	badReq.ControllerID = "wrong"
	if _, e = f.worker.Register(ctx, badReq); !statusCode(e, 409) {
		t.Fatal("wrong controller accepted")
	}
	badReq = req
	badReq.ProductVersion = "other"
	if result, err := f.worker.Register(ctx, badReq); err != nil || result.Host.Compatible || result.Host.CompatibilityError == "" {
		t.Fatalf("incompatible worker not visible: %+v %v", result, err)
	}
	if _, e = f.worker.Poll(ctx, protocol.PollRequest{WorkerIdentity: req.WorkerIdentity}); !statusCode(e, 400) {
		t.Fatalf("incompatible worker polled: %v", e)
	}
	if host, err := f.admin.GetHost(ctx, req.HostID); err != nil || host.Compatible || host.ProductVersion != "other" {
		t.Fatalf("incompatible inventory missing: %+v %v", host, err)
	}
	result, e := f.worker.Register(ctx, req)
	if e != nil || result.Host.HostInstanceID != "instance" {
		t.Fatalf("register %+v %v", result, e)
	}
	if _, e = f.worker.ListLeases(ctx); !statusCode(e, 403) {
		t.Fatal("worker read client lease inventory")
	}
	replacement := req
	replacement.HostInstanceID = "new-home"
	if _, e = f.worker.Register(ctx, replacement); !statusCode(e, 409) {
		t.Fatal("missing worker home identity silently adopted")
	}
	old := req.WorkerIdentity
	old.Incarnation = "old"
	if e = f.worker.Heartbeat(ctx, old); !statusCode(e, 409) {
		t.Fatal("stale incarnation heartbeat accepted")
	}
}
func TestMTLSAssignPollResultsAndBlobACL(t *testing.T) {
	f := serveFixture(t)
	ctx := context.Background()
	w := protocol.WorkerIdentity{ControllerID: f.store.ID, HostID: "host-a", HostInstanceID: "instance", Incarnation: "boot"}
	_, e := f.worker.Register(ctx, protocol.RegisterRequest{WorkerIdentity: w, ProtocolVersion: 1, ProductVersion: "test", OS: "linux", Arch: "amd64", Capabilities: []string{"process"}, Capacity: protocol.Capacity{MaxLeases: 1}})
	if e != nil {
		t.Fatal(e)
	}
	source := []byte("opaque source bundle")
	sum := sha256.Sum256(source)
	digest := hex.EncodeToString(sum[:])
	if _, e = f.worker.Upload(ctx, bytes.NewReader(source), digest, "source"); !statusCode(e, 403) {
		t.Fatal("worker published source package")
	}
	if _, e = f.admin.Upload(ctx, bytes.NewReader(source), strings.Repeat("a", 64), "source"); !statusCode(e, 400) {
		t.Fatal("mismatched source digest accepted")
	}
	if _, e = f.admin.Upload(ctx, bytes.NewReader(source), digest, "source"); e != nil {
		t.Fatal(e)
	}
	if _, e = f.worker.Download(ctx, digest, "source"); !statusCode(e, 403) {
		t.Fatal("worker downloaded unassigned source")
	}
	q := protocol.CreateRequest{OperationID: "create", Stack: "app", ManifestDigest: digest, PlanDigest: digest, Manifest: json.RawMessage("{}"), ControlBlobDigest: digest, Package: json.RawMessage(`{"manifest_blob_digest":"` + digest + `","sources":[]}`), RequiredCapabilities: []string{"process"}, Options: json.RawMessage(`{"owner":"user@host"}`)}
	op, e := f.admin.Create(ctx, q)
	if e != nil {
		t.Fatal(e)
	}
	duplicate, e := f.admin.Create(ctx, q)
	if e != nil || duplicate.LeaseID != op.LeaseID {
		t.Fatal("network create replay duplicated placement")
	}
	l, e := f.admin.GetLease(ctx, op.LeaseID)
	if e != nil || l.Owner != "user@host" {
		t.Fatal("owner metadata lost")
	}
	downloaded, e := f.worker.Download(ctx, digest, "source")
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(downloaded)
	downloaded.Close()
	if e != nil || !bytes.Equal(got, source) {
		t.Fatal("assigned blob mismatch")
	}
	polled, e := f.worker.Poll(ctx, protocol.PollRequest{WorkerIdentity: w, WaitSeconds: 0})
	if e != nil || polled.Operation == nil || polled.Operation.ID != op.ID {
		t.Fatal("assignment not delivered")
	}
	data := []byte("redacted artifact")
	sum = sha256.Sum256(data)
	artifact := hex.EncodeToString(sum[:])
	blob, e := f.worker.Upload(ctx, bytes.NewReader(data), artifact, "artifact")
	if e != nil {
		t.Fatal(e)
	}
	result := protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "ready", Artifacts: []protocol.Blob{blob}, Payload: json.RawMessage(`{"ok":true}`)}
	if e = f.worker.Complete(ctx, result); e != nil {
		t.Fatal(e)
	}
	recorded, e := f.admin.GetOperation(ctx, op.ID)
	if e != nil || recorded.Result == nil || recorded.Result.Artifacts[0].Digest != artifact {
		t.Fatal("artifact result lost")
	}
	file, e := f.admin.Download(ctx, artifact, "artifact")
	if e != nil {
		t.Fatal(e)
	}
	got, e = io.ReadAll(file)
	file.Close()
	if e != nil || !bytes.Equal(data, got) {
		t.Fatal("artifact bytes changed")
	}
	if _, e = f.admin.Remove(ctx, "host-a"); !statusCode(e, 409) {
		t.Fatal("active host removed")
	}
	if _, e = f.admin.Drain(ctx, "host-a"); e != nil {
		t.Fatal(e)
	}
	if _, e = f.admin.Undrain(ctx, "host-a"); e != nil {
		t.Fatal(e)
	}
}
func TestControllerRunRejectsSecondLiveInstance(t *testing.T) {
	f := serveFixture(t)
	ctx := context.Background()
	if _, e := f.admin.Info(ctx); e != nil {
		t.Fatal(e)
	}
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	started := time.Now()
	if e = f.s.Run(ctx, ln, f.tls); e == nil {
		t.Fatal("second controller instance started")
	}
	if time.Since(started) > time.Second {
		t.Fatal("second controller lock did not fail promptly")
	}
}
