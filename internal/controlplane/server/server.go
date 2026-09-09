package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/controlplane/store"
	"github.com/mahcialet/agent-env/internal/instance"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BlobStore interface {
	Put(context.Context, io.Reader, string, int64) (blobstore.Blob, error)
	Open(context.Context, string, int64) (*os.File, blobstore.Blob, error)
}
type Server struct {
	Store          *store.Store
	Blobs          BlobStore
	ProductVersion string
}

func New(s *store.Store, b BlobStore, version string) *Server {
	return &Server{Store: s, Blobs: b, ProductVersion: version}
}
func (s *Server) Run(ctx context.Context, listener net.Listener, tlsConfig *tls.Config) error {
	if tlsConfig == nil || tlsConfig.ClientCAs == nil || len(tlsConfig.Certificates) == 0 {
		return errors.New("controller requires explicit CA and server certificate")
	}
	release, err := instance.Acquire(filepath.Join(filepath.Dir(s.Store.Path), "controller.lock"))
	if err != nil {
		return err
	}
	defer release()
	cfg := tlsConfig.Clone()
	cfg.MinVersion = tls.VersionTLS13
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	srv := &http.Server{Handler: s.Handler(), TLSConfig: cfg, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute, IdleTimeout: 30 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(c)
			srv.Close()
		case <-done:
		}
	}()
	err = srv.Serve(tls.NewListener(listener, cfg))
	if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		return nil
	}
	return err
}
func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.serve) }
func fail(w http.ResponseWriter, err error) {
	var f *store.Fault
	status := http.StatusInternalServerError
	e := protocol.Error{Code: "internal", Message: "controller operation failed"}
	if errors.As(err, &f) {
		e.Code = f.Code
		e.Message = f.Message
		switch f.Code {
		case "invalid", "version":
			status = 400
		case "forbidden":
			status = 403
		case "not_found":
			status = 404
		case "conflict", "identity", "capacity":
			status = 409
		case "unavailable":
			status = 503
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(e)
}
func reject(code, msg string) error { return &store.Fault{Code: code, Message: msg} }
func send(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return reject("invalid", "invalid or oversized request JSON")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return reject("invalid", "request must contain one JSON object")
	}
	return nil
}
func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		fail(w, reject("forbidden", "verified mTLS identity required"))
		return
	}
	e, err := s.Store.Authorize(r.Context(), auth.Fingerprint(r.TLS.PeerCertificates[0]))
	if err != nil {
		fail(w, err)
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "v1" {
		fail(w, reject("not_found", "endpoint not found"))
		return
	}
	if path == "v1/info" && r.Method == "GET" {
		send(w, protocol.Info{ControllerID: s.Store.ID, ProtocolVersion: protocol.Version, ProductVersion: s.ProductVersion})
		return
	}
	if len(parts) == 3 && parts[1] == "blobs" {
		s.blob(w, r, e, parts[2])
		return
	}
	if e.Role == protocol.RoleWorker {
		if r.Method != "POST" {
			fail(w, reject("forbidden", "worker endpoint requires POST"))
			return
		}
		switch path {
		case "v1/register":
			var q protocol.RegisterRequest
			if err = decode(w, r, &q); err == nil {
				if q.HostID != e.HostID {
					err = reject("forbidden", "certificate is enrolled for another host")
				} else if q.ProductVersion != s.ProductVersion {
					err = reject("version", "incompatible product version")
				} else {
					var h protocol.Host
					h, err = s.Store.Register(r.Context(), q)
					if err == nil {
						send(w, protocol.RegisterResult{Info: protocol.Info{ControllerID: s.Store.ID, ProtocolVersion: protocol.Version, ProductVersion: s.ProductVersion}, Host: h})
						return
					}
				}
			}
		case "v1/heartbeat":
			var q protocol.WorkerIdentity
			if err = decode(w, r, &q); err == nil {
				if q.HostID != e.HostID {
					err = reject("forbidden", "certificate host mismatch")
				} else {
					var h protocol.Host
					h, err = s.Store.Heartbeat(r.Context(), q)
					if err == nil {
						send(w, h)
						return
					}
				}
			}
		case "v1/poll":
			var q protocol.PollRequest
			if err = decode(w, r, &q); err == nil {
				if q.HostID != e.HostID {
					err = reject("forbidden", "certificate host mismatch")
				} else if q.WaitSeconds < 0 || q.WaitSeconds > 30 {
					err = reject("invalid", "poll wait must be between 0 and 30 seconds")
				} else {
					deadline := time.Now().Add(time.Duration(q.WaitSeconds) * time.Second)
					for {
						var op *protocol.Operation
						op, err = s.Store.Poll(r.Context(), q.WorkerIdentity)
						if err != nil {
							break
						}
						if op != nil || !time.Now().Before(deadline) {
							send(w, protocol.PollResult{Operation: op})
							return
						}
						timer := time.NewTimer(100 * time.Millisecond)
						select {
						case <-r.Context().Done():
							timer.Stop()
							return
						case <-timer.C:
						}
					}
				}
			}
		case "v1/results":
			var q protocol.Result
			if err = decode(w, r, &q); err == nil {
				if q.HostID != e.HostID {
					err = reject("forbidden", "certificate host mismatch")
				} else {
					for _, a := range q.Artifacts {
						if s.Blobs == nil {
							err = reject("unavailable", "blob store unavailable")
							break
						}
						var f *os.File
						var b blobstore.Blob
						f, b, err = s.Blobs.Open(r.Context(), a.Digest, blobstore.ArtifactLimit)
						if err != nil {
							err = reject("invalid", "result artifact is not available")
							break
						}
						f.Close()
						if b.Size != a.Size {
							err = reject("invalid", "result artifact size mismatch")
							break
						}
					}
					if err == nil {
						var op protocol.Operation
						op, err = s.Store.Complete(r.Context(), q)
						if err == nil {
							send(w, op)
							return
						}
					}
				}
			}
		default:
			err = reject("forbidden", "endpoint is not available to workers")
		}
		fail(w, err)
		return
	}
	if e.Role != protocol.RoleClient {
		fail(w, reject("forbidden", "unknown enrollment role"))
		return
	}
	switch {
	case path == "v1/leases" && r.Method == "POST":
		var q protocol.CreateRequest
		if err = decode(w, r, &q); err == nil {
			var refs []string
			refs, err = store.SourceBlobs(q)
			if err == nil {
				for _, d := range refs {
					if s.Blobs == nil {
						err = reject("unavailable", "blob store unavailable")
						break
					}
					var f *os.File
					f, _, err = s.Blobs.Open(r.Context(), d, blobstore.SourceLimit)
					if err != nil {
						err = reject("invalid", "source bundle is not available")
						break
					}
					f.Close()
				}
			}
			if err == nil {
				var op protocol.Operation
				op, err = s.Store.Create(r.Context(), q)
				if err == nil {
					send(w, op)
					return
				}
			}
		}
	case path == "v1/operations" && r.Method == "POST":
		var q protocol.SubmitRequest
		if err = decode(w, r, &q); err == nil {
			var op protocol.Operation
			op, err = s.Store.Submit(r.Context(), q)
			if err == nil {
				send(w, op)
				return
			}
		}
	case path == "v1/leases" && r.Method == "GET":
		var leases []protocol.Lease
		leases, err = s.Store.ListLeases(r.Context())
		if err == nil {
			send(w, leases)
			return
		}
	case path == "v1/hosts" && r.Method == "GET":
		var hosts []protocol.Host
		hosts, err = s.Store.ListHosts(r.Context())
		if err == nil {
			send(w, hosts)
			return
		}
	case len(parts) == 3 && parts[1] == "leases" && r.Method == "GET" && store.ValidID(parts[2]):
		var l protocol.Lease
		l, err = s.Store.GetLease(r.Context(), parts[2])
		if err == nil {
			send(w, l)
			return
		}
	case len(parts) == 3 && parts[1] == "operations" && r.Method == "GET" && store.ValidID(parts[2]):
		var op protocol.Operation
		op, err = s.Store.GetOperation(r.Context(), parts[2])
		if err == nil {
			send(w, op)
			return
		}
	case len(parts) == 3 && parts[1] == "hosts" && r.Method == "GET" && store.ValidID(parts[2]):
		var h protocol.Host
		h, err = s.Store.GetHost(r.Context(), parts[2])
		if err == nil {
			send(w, h)
			return
		}
	case len(parts) == 4 && parts[1] == "hosts" && r.Method == "POST" && store.ValidID(parts[2]):
		var h protocol.Host
		switch parts[3] {
		case "drain":
			h, err = s.Store.Drain(r.Context(), parts[2])
		case "undrain":
			h, err = s.Store.Undrain(r.Context(), parts[2])
		case "remove":
			h, err = s.Store.Remove(r.Context(), parts[2])
		default:
			err = reject("not_found", "host action not found")
		}
		if err == nil {
			send(w, h)
			return
		}
	default:
		err = reject("forbidden", "endpoint is not available to clients")
	}
	fail(w, err)
}
func (s *Server) blob(w http.ResponseWriter, r *http.Request, e protocol.Enrollment, digest string) {
	if !store.ValidDigest(digest) {
		fail(w, reject("invalid", "invalid blob digest"))
		return
	}
	kind := r.URL.Query().Get("kind")
	limit := blobstore.SourceLimit
	if kind == "artifact" {
		limit = blobstore.ArtifactLimit
	} else if kind != "source" {
		fail(w, reject("invalid", "blob kind required"))
		return
	}
	if s.Blobs == nil {
		fail(w, reject("unavailable", "blob store unavailable"))
		return
	}
	if r.Method == "PUT" {
		if (e.Role == protocol.RoleClient && kind != "source") || (e.Role == protocol.RoleWorker && kind != "artifact") {
			fail(w, reject("forbidden", "role cannot publish this blob kind"))
			return
		}
		b, err := s.Blobs.Put(r.Context(), http.MaxBytesReader(w, r.Body, limit), digest, limit)
		if err != nil {
			fail(w, reject("invalid", "blob bytes do not match the required digest or size"))
			return
		}
		send(w, protocol.Blob{Digest: b.Digest, Size: b.Size})
		return
	}
	if r.Method != "GET" {
		fail(w, reject("forbidden", "blob method not permitted"))
		return
	}
	if err := s.Store.AuthorizeBlob(r.Context(), e, digest, kind); err != nil {
		fail(w, err)
		return
	}
	f, b, err := s.Blobs.Open(r.Context(), digest, limit)
	if err != nil {
		fail(w, reject("not_found", "verified blob not found"))
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-SHA256", b.Digest)
	http.ServeContent(w, r, "blob", time.Time{}, f)
}
