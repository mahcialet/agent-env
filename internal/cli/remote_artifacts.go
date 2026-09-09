package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/blobstore"
	"github.com/spf13/cobra"
)

func addRemoteArtifacts(root *cobra.Command, f *remoteFlags, emit func(any) error, errOut io.Writer) {
	root.AddCommand(&cobra.Command{Use: "artifacts <lease-id>", Args: cobra.ExactArgs(1), Short: "Publish registered worker artifacts and return immutable references", RunE: func(cmd *cobra.Command, args []string) error {
		if f.Controller == "" {
			return errors.New("artifacts requires --controller; local artifact metadata is available in show")
		}
		return f.run(cmd, args, emit, errOut)
	}})
	var destination string
	fetch := &cobra.Command{Use: "artifact-download <digest>", Args: cobra.ExactArgs(1), Short: "Download a registered controller artifact after digest verification", RunE: func(cmd *cobra.Command, args []string) error {
		if f.Controller == "" || destination == "" {
			return errors.New("artifact-download requires --controller and --destination")
		}
		if !blobstore.ValidDigest(args[0]) {
			return errors.New("invalid SHA-256 artifact digest")
		}
		c, e := f.client()
		if e != nil {
			return e
		}
		defer c.Close()
		body, e := c.Download(cmd.Context(), args[0], "artifact")
		if e != nil {
			return e
		}
		defer body.Close()
		temp, e := os.MkdirTemp("", "agent-env-artifact-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(temp)
		cas, e := blobstore.New(filepath.Join(temp, "blobs"))
		if e != nil {
			return e
		}
		blob, e := cas.Put(cmd.Context(), body, args[0], blobstore.ArtifactLimit)
		if e != nil {
			return e
		}
		source, _, e := cas.Open(cmd.Context(), blob.Digest, blobstore.ArtifactLimit)
		if e != nil {
			return e
		}
		defer source.Close()
		output, e := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		_, copyErr := io.Copy(output, source)
		syncErr := output.Sync()
		closeErr := output.Close()
		if e = errors.Join(copyErr, syncErr, closeErr); e != nil {
			return fmt.Errorf("download write failed; partial destination retained: %w", e)
		}
		return emit(map[string]any{"digest": blob.Digest, "size": blob.Size, "destination": destination})
	}}
	fetch.Flags().StringVar(&destination, "destination", "", "new local output file (never overwritten)")
	root.AddCommand(fetch)
}
