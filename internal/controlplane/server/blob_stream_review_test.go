package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/controlplane/store"
)

type blobDeadlineWriter struct {
	http.ResponseWriter
	clears *atomic.Int32
	delay  time.Duration
}

func (w *blobDeadlineWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *blobDeadlineWriter) SetReadDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		w.clears.Add(1)
	}
	return http.NewResponseController(w.ResponseWriter).SetReadDeadline(deadline)
}
func (w *blobDeadlineWriter) SetWriteDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		w.clears.Add(1)
	}
	return http.NewResponseController(w.ResponseWriter).SetWriteDeadline(deadline)
}
func (w *blobDeadlineWriter) Write(p []byte) (int, error) {
	if w.delay > 0 {
		time.Sleep(w.delay)
		w.delay = 0
	}
	return w.ResponseWriter.Write(p)
}

func TestAuthorizedBlobStreamsOutliveServerMetadataDeadline(t *testing.T) {
	for _, method := range []string{"upload", "download", "forbidden-kind", "unenrolled"} {
		t.Run(method, func(t *testing.T) {
			root := t.TempDir()
			db, err := store.Open(filepath.Join(root, "controller.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			cas, err := blobstore.New(filepath.Join(root, "blobs"))
			if err != nil {
				t.Fatal(err)
			}
			data := []byte("ab")
			digest := fmt.Sprintf("%x", sha256.Sum256(data))
			if _, err = cas.Put(context.Background(), bytes.NewReader(data), digest, blobstore.SourceLimit); err != nil {
				t.Fatal(err)
			}
			ca := certificateAuthority(t)
			serverCert, _ := ca.leaf(t, 20, true)
			clientCert, clientLeaf := ca.leaf(t, 21, false)
			if method != "unenrolled" {
				if err = db.Enroll(context.Background(), protocol.Enrollment{Fingerprint: auth.Fingerprint(clientLeaf), Role: protocol.RoleClient}); err != nil {
					t.Fatal(err)
				}
			}
			handler := New(db, cas, "fixture").Handler()
			var clears atomic.Int32
			const timeout = 100 * time.Millisecond
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				delay := time.Duration(0)
				if method == "download" {
					delay = 3 * timeout
				}
				handler.ServeHTTP(&blobDeadlineWriter{ResponseWriter: w, clears: &clears, delay: delay}, r)
			}))
			server.Config.ReadTimeout = timeout
			server.Config.WriteTimeout = timeout
			server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientCAs: ca.pool, ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS13}
			server.StartTLS()
			defer server.Close()
			transport := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{clientCert}, RootCAs: ca.pool, MinVersion: tls.VersionTLS13}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
			kind := "source"
			verb := http.MethodGet
			var body io.Reader
			if method != "download" {
				verb = http.MethodPut
				body = bytes.NewReader(data)
			}
			if method == "forbidden-kind" {
				kind = "artifact"
			}
			if method == "upload" {
				reader, writer := io.Pipe()
				defer reader.Close()
				defer writer.Close()
				body = reader
				go func() {
					defer writer.Close()
					if _, err := writer.Write(data[:1]); err != nil {
						return
					}
					time.Sleep(3 * timeout)
					_, _ = writer.Write(data[1:])
				}()
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, verb, server.URL+"/v1/blobs/"+digest+"?kind="+kind, body)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("blob transfer failed across metadata deadline: %v", err)
			}
			defer response.Body.Close()
			result, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("blob body failed across metadata deadline: %v", err)
			}
			if method == "forbidden-kind" || method == "unenrolled" {
				if response.StatusCode != http.StatusForbidden || clears.Load() != 0 {
					t.Fatalf("unauthorized deadline change: status=%d clears=%d", response.StatusCode, clears.Load())
				}
				return
			}
			if response.StatusCode != http.StatusOK || clears.Load() != 2 {
				t.Fatalf("authorized stream: status=%d clears=%d body=%s", response.StatusCode, clears.Load(), result)
			}
			if method == "download" && !bytes.Equal(result, data) {
				t.Fatalf("download changed bytes: %q", result)
			}
			if method == "upload" && !bytes.Contains(result, []byte(digest)) {
				t.Fatalf("missing verified upload receipt: %s", result)
			}
		})
	}
}
