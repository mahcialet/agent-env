package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBlobTransferOutlivesMetadataTimeout(t *testing.T) {
	for _, method := range []string{"upload", "download"} {
		t.Run(method, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/info" {
					<-r.Context().Done()
					return
				}
				if r.Method == http.MethodPut {
					first := make([]byte, 1)
					if _, err := io.ReadFull(r.Body, first); err != nil {
						return
					}
					close(started)
					if _, err := io.Copy(io.Discard, r.Body); err != nil {
						return
					}
					_, _ = io.WriteString(w, `{"digest":"fixture","size":2}`)
					return
				}
				_, _ = io.WriteString(w, "a")
				w.(http.Flusher).Flush()
				close(started)
				select {
				case <-release:
					_, _ = io.WriteString(w, "b")
				case <-r.Context().Done():
				}
			}))
			defer server.Close()
			httpClient := server.Client()
			httpClient.Timeout = 50 * time.Millisecond
			c := &Client{base: server.URL, http: httpClient}
			defer c.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			result := make(chan error, 1)
			if method == "upload" {
				reader, writer := io.Pipe()
				defer reader.Close()
				defer writer.Close()
				go func() {
					defer writer.Close()
					if _, err := io.WriteString(writer, "a"); err != nil {
						return
					}
					select {
					case <-release:
						_, _ = io.WriteString(writer, "b")
					case <-ctx.Done():
					}
				}()
				go func() {
					blob, err := c.Upload(ctx, reader, "fixture", "source")
					if err == nil && blob.Size != 2 {
						err = errors.New("upload receipt lost bytes")
					}
					result <- err
				}()
			} else {
				go func() {
					body, err := c.Download(ctx, "fixture", "source")
					if err != nil {
						result <- err
						return
					}
					defer body.Close()
					data, err := io.ReadAll(body)
					if err == nil && string(data) != "ab" {
						err = errors.New("download lost bytes")
					}
					result <- err
				}()
			}
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("blob did not begin")
			}
			// The metadata request consumes the entire configured timeout while the
			// blob is deliberately unfinished. Blob completion must remain caller-owned.
			if _, err := c.Info(ctx); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("metadata lost deadline: %v", err)
			}
			close(release)
			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("blob used metadata timeout: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("blob did not finish")
			}
			if c.http.Timeout != 50*time.Millisecond {
				t.Fatal("stream mutated shared metadata timeout")
			}
		})
	}
}

func TestBlobTransferPreservesCallerCancellation(t *testing.T) {
	for _, method := range []string{"upload", "download", "upload-deadline", "download-deadline"} {
		t.Run(method, func(t *testing.T) {
			started := make(chan struct{})
			finish := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					first := make([]byte, 1)
					if _, err := io.ReadFull(r.Body, first); err != nil {
						return
					}
					close(started)
					_, _ = io.Copy(io.Discard, r.Body)
					<-finish
					return
				}
				_, _ = io.WriteString(w, "prefix")
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
				// Keep the stream unfinished until the client observes cancellation;
				// otherwise graceful EOF can legitimately win that race.
				<-finish
			}))
			defer func() { close(finish); server.Close() }()
			c := &Client{base: server.URL, http: server.Client()}
			defer c.Close()
			ctx, cancel := context.WithCancel(context.Background())
			if strings.HasSuffix(method, "-deadline") {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			}
			defer cancel()
			result := make(chan error, 1)
			if strings.HasPrefix(method, "upload") {
				reader, writer := io.Pipe()
				defer reader.Close()
				defer writer.Close()
				go func() { _, _ = io.Copy(writer, strings.NewReader("a")) }()
				go func() { _, err := c.Upload(ctx, reader, "fixture", "source"); result <- err }()
			} else {
				go func() {
					body, err := c.Download(ctx, "fixture", "source")
					if err != nil {
						result <- err
						return
					}
					defer body.Close()
					_, err = io.ReadAll(body)
					result <- err
				}()
			}
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("blob did not begin")
			}
			want := context.Canceled
			if strings.HasSuffix(method, "-deadline") {
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			select {
			case err := <-result:
				if !errors.Is(err, want) {
					t.Fatalf("caller cancellation lost: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("blob ignored cancellation")
			}
		})
	}
}
