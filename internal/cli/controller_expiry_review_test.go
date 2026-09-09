package cli

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	controlstore "github.com/mahcialet/agent-env/internal/controlplane/store"
)

type controllerReadyWriter struct {
	ready chan struct{}
	once  sync.Once
}

func (w *controllerReadyWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "listening on https://") {
		w.once.Do(func() { close(w.ready) })
	}
	return len(p), nil
}

func TestControllerServeQueuesExpiryWithoutPolling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AGENT_ENV_HOME", home)
	dbPath := filepath.Join(home, "control-plane", "controller.db")
	s, err := controlstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	identity := protocol.WorkerIdentity{ControllerID: s.ID, HostID: "host", HostInstanceID: "instance", Incarnation: "boot"}
	if _, err = s.Register(ctx, protocol.RegisterRequest{WorkerIdentity: identity, ProtocolVersion: protocol.Version, ProductVersion: "test", OS: "linux", Arch: "amd64", Capacity: protocol.Capacity{MaxLeases: 1}}); err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	op, err := s.Create(ctx, protocol.CreateRequest{OperationID: "create", Stack: "local", Manifest: json.RawMessage(`{}`), Package: json.RawMessage(`{"manifest_blob_digest":"` + digest + `","sources":[]}`), ControlBlobDigest: digest, ManifestDigest: digest, PlanDigest: digest})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Poll(ctx, identity); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Complete(ctx, protocol.Result{WorkerIdentity: identity, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "ready"}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	// Reuse Go's test certificate only to start the real CLI TLS listener; no
	// HTTP polling or worker connection may schedule the expiry in this test.
	tlsFixture := httptest.NewTLSServer(nil)
	cert := tlsFixture.TLS.Certificates[0]
	tlsFixture.Close()
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath := filepath.Join(home, "cert.pem"), filepath.Join(home, "key.pem")
	if err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	ready := &controllerReadyWriter{ready: make(chan struct{})}
	cmd := New(io.Discard, ready)
	cmd.SetArgs([]string{"--tls-ca", certPath, "--tls-cert", certPath, "--tls-key", keyPath, "control-plane", "serve", "--listen", "127.0.0.1:0"})
	done := make(chan error, 1)
	go func() { done <- cmd.ExecuteContext(runCtx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			t.Error("controller did not stop")
		}
	})
	select {
	case <-ready.ready:
	case <-time.After(10 * time.Second):
		t.Fatal("controller did not start")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA busy_timeout=3000"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE lease_lifetimes SET expires_at=? WHERE lease_id=?", time.Now().Add(-time.Second).UnixNano(), op.LeaseID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM operations WHERE lease_id=? AND kind='destroy' AND state='queued'", op.LeaseID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("actual CLI server did not queue expired offline lease")
		}
		time.Sleep(20 * time.Millisecond)
	}
	var state string
	if err = db.QueryRow("SELECT state FROM leases WHERE id=?", op.LeaseID).Scan(&state); err != nil || state != "READY" {
		t.Fatalf("expiry freed capacity: %s %v", state, err)
	}
}
