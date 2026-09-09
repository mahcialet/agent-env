package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/mahcialet/agent-env/internal/browser/cdp"
	"github.com/mahcialet/agent-env/internal/buildinfo"
	"github.com/mahcialet/agent-env/internal/controlplane/auth"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/controlplane/server"
	controllerstore "github.com/mahcialet/agent-env/internal/controlplane/store"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/policy"
	"github.com/mahcialet/agent-env/internal/remotesource"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
	"github.com/mahcialet/agent-env/internal/worker"
	"github.com/spf13/cobra"
)

func addServices(root *cobra.Command, f *remoteFlags, emit func(any) error, errOut io.Writer) {
	controller := &cobra.Command{Use: "control-plane", Short: "Run or enroll identities in a single control plane"}
	address := "127.0.0.1:7443"
	serve := &cobra.Command{Use: "serve", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		config, e := auth.LoadTLS(f.CA, f.Cert, f.Key, true)
		if e != nil {
			return e
		}
		home, e := paths.Resolve()
		if e != nil {
			return e
		}
		home = filepath.Join(home, "control-plane")
		store, e := controllerstore.Open(filepath.Join(home, "controller.db"))
		if e != nil {
			return e
		}
		defer store.Close()
		cas, e := blobstore.New(filepath.Join(home, "blobs"))
		if e != nil {
			return e
		}
		handler := server.New(store, cas, buildinfo.Current().Version)
		listener, e := net.Listen("tcp", address)
		if e != nil {
			return e
		}
		defer listener.Close()
		fmt.Fprintf(errOut, "Controller %s listening on https://%s\n", store.ID, listener.Addr().String())
		return handler.Run(cmd.Context(), listener, config)
	}}
	serve.Flags().StringVar(&address, "listen", address, "controller TLS listen address")
	controller.AddCommand(serve)
	var enrollmentCert, role, host string
	enroll := &cobra.Command{Use: "enroll", Args: cobra.NoArgs, Short: "Enroll a pre-provisioned certificate in local controller state", RunE: func(cmd *cobra.Command, _ []string) error {
		fingerprint, e := auth.ReadFingerprint(enrollmentCert)
		if e != nil {
			return e
		}
		home, e := paths.Resolve()
		if e != nil {
			return e
		}
		store, e := controllerstore.Open(filepath.Join(home, "control-plane", "controller.db"))
		if e != nil {
			return e
		}
		defer store.Close()
		enrollment := protocol.Enrollment{Fingerprint: fingerprint, Role: role, HostID: host}
		if e = store.Enroll(cmd.Context(), enrollment); e != nil {
			return e
		}
		return emit(enrollment)
	}}
	enroll.Flags().StringVar(&enrollmentCert, "certificate", "", "certificate PEM to enroll")
	enroll.Flags().StringVar(&role, "role", "", "client or worker")
	enroll.Flags().StringVar(&host, "host-id", "", "worker host ID (worker role only)")
	controller.AddCommand(enroll)
	root.AddCommand(controller)
	var workerHost string
	var maximum, androidSlots int
	workerCommand := &cobra.Command{Use: "worker", Short: "Operate a worker that connects outbound to its controller"}
	workerServe := &cobra.Command{Use: "serve", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if maximum <= 0 || maximum > policy.Defaults().MaxActive {
			return fmt.Errorf("max-leases must be between 1 and the worker local policy limit %d", policy.Defaults().MaxActive)
		}
		if androidSlots < 0 || androidSlots > sqlite.AndroidSlotCapacity {
			return fmt.Errorf("android-slots must be between 0 and the worker allocator limit %d", sqlite.AndroidSlotCapacity)
		}
		c, e := f.client()
		if e != nil {
			return e
		}
		home, e := paths.Resolve()
		if e != nil {
			return e
		}
		// Canonicalize only the explicitly selected private root. Managed child
		// paths still undergo their own no-symlink and ownership checks.
		if e = os.MkdirAll(home, 0700); e != nil {
			return e
		}
		home, e = filepath.EvalSymlinks(home)
		if e != nil {
			return e
		}
		journal, e := worker.OpenJournal(home, workerHost)
		if e != nil {
			return e
		}
		defer journal.Close()
		local, e := sqlite.Open(filepath.Join(home, "state.db"))
		if e != nil {
			return e
		}
		defer local.Close()
		cas, e := blobstore.New(filepath.Join(home, "remote-blobs"))
		if e != nil {
			return e
		}
		factory := func(authority *domain.Management) *app.Service {
			s := serviceForStore(home, local, errOut, errOut)
			s.BrowserProvider = cdp.Client{}
			s.Management = authority
			return s
		}
		capabilities := workerCapabilities(cmd.Context(), factory(nil), androidSlots)
		executor := &worker.AppExecutor{Home: home, CAS: cas, Factory: factory}
		runner := &worker.Runner{Journal: journal, Transport: c, Executor: executor, Registration: protocol.RegisterRequest{ProductVersion: buildinfo.Current().Version, OS: runtime.GOOS, Arch: runtime.GOARCH, Capabilities: capabilities, Capacity: protocol.Capacity{MaxLeases: maximum, AndroidSlots: androidSlots}}}
		runner.Download = func(ctx context.Context, op protocol.Operation) error {
			if op.Kind != "create" {
				return nil
			}
			var request protocol.CreateRequest
			if e := json.Unmarshal(op.Payload, &request); e != nil {
				return e
			}
			var pkg remotesource.Package
			if e := json.Unmarshal(request.Package, &pkg); e != nil {
				return e
			}
			if len(pkg.Manifest) == 0 || string(pkg.Manifest) == "null" {
				pkg.Manifest = request.Manifest
			}
			if e := remotesource.Validate(pkg); e != nil {
				return e
			}
			digests := map[string]bool{pkg.ManifestBlobDigest: true}
			for _, source := range pkg.Sources {
				digests[source.BlobDigest] = true
			}
			for digest := range digests {
				if existing, _, e := cas.Open(ctx, digest, blobstore.SourceLimit); e == nil {
					existing.Close()
					continue
				}
				body, e := c.Download(ctx, digest, "source")
				if e != nil {
					return e
				}
				_, e = cas.Put(ctx, body, digest, blobstore.SourceLimit)
				body.Close()
				if e != nil {
					return e
				}
			}
			return nil
		}
		runner.Upload = func(ctx context.Context, result protocol.Result) error {
			for _, artifact := range result.Artifacts {
				file, blob, e := cas.Open(ctx, artifact.Digest, blobstore.ArtifactLimit)
				if e != nil {
					return e
				}
				if blob.Size != artifact.Size {
					file.Close()
					return errors.New("stored artifact size mismatch")
				}
				_, e = c.Upload(ctx, file, artifact.Digest, "artifact")
				file.Close()
				if e != nil {
					return e
				}
			}
			return nil
		}
		fmt.Fprintf(errOut, "Worker %s instance %s connecting outbound\n", journal.Identity.HostID, journal.Identity.HostInstanceID)
		return runner.Run(cmd.Context())
	}}
	workerServe.Flags().StringVar(&workerHost, "host-id", "", "enrolled worker host ID")
	workerServe.Flags().IntVar(&maximum, "max-leases", 1, "maximum controller lease reservations")
	workerServe.Flags().IntVar(&androidSlots, "android-slots", 0, fmt.Sprintf("maximum emulator reservations (0-%d)", sqlite.AndroidSlotCapacity))
	workerCommand.AddCommand(workerServe)
	root.AddCommand(workerCommand)
}

func workerCapabilities(ctx context.Context, s *app.Service, androidSlots int) []string {
	result := []string{"persistent-process", "browser-cdp"}
	if _, e := execx.LookPath("git"); e == nil {
		result = append(result, "git")
	}
	if providers, ok := s.Runtime.(interface {
		DoctorFor(context.Context, domain.ComposeProviderName) (map[string]string, error)
	}); ok {
		for _, provider := range []domain.ComposeProviderName{domain.ComposeProviderDocker, domain.ComposeProviderPodman} {
			probe, cancel := context.WithTimeout(ctx, 15*time.Second)
			_, e := providers.DoctorFor(probe, provider)
			cancel()
			if e == nil {
				if provider == domain.ComposeProviderDocker {
					result = append(result, "compose.docker")
				} else {
					result = append(result, "compose.podman")
				}
			}
		}
	}
	if androidSlots > 0 && s.Android != nil {
		probe, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, e := s.Android.Doctor(probe)
		cancel()
		if e == nil {
			result = append(result, "android-emulator")
			if _, e = execx.LookPath("flutter"); e == nil {
				result = append(result, "flutter-android")
			}
		}
	}
	sort.Strings(result)
	return result
}
