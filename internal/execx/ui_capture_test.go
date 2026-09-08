package execx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCaptureChild(t *testing.T) {
	if os.Getenv("AGENT_ENV_CAPTURE_CHILD") != "1" {
		return
	}
	_, _ = io.WriteString(os.Stdout, strings.Repeat("o", 8192))
	_, _ = io.WriteString(os.Stderr, strings.Repeat("e", 8192))
	os.Exit(0)
}
func TestCaptureNativeBoundsAndDefault(t *testing.T) {
	for _, limit := range []int{0, 128, 8192} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			r, err := (OSRunner{}).Run(context.Background(), Command{Name: os.Args[0], Args: []string{"-test.run=^TestCaptureChild$"}, Env: map[string]string{"AGENT_ENV_CAPTURE_CHILD": "1", "GORACE": "atexit_sleep_ms=0"}, CaptureLimit: limit, Timeout: 10 * time.Second})
			if limit == 128 {
				if !errors.Is(err, ErrOutputIncomplete) {
					t.Fatalf("missing incomplete marker: %v", err)
				}
				if len(r.Stdout) > limit || len(r.Stderr) > limit {
					t.Fatalf("unbounded output: %d %d", len(r.Stdout), len(r.Stderr))
				}
				if len(r.Stdout) != limit {
					t.Fatalf("stdout prefix lost: %d", len(r.Stdout))
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if r.Stdout != strings.Repeat("o", 8192) || r.Stderr != strings.Repeat("e", 8192) {
					t.Fatal("default/exact-limit output changed")
				}
			}
		})
	}
}
func TestCaptureCopyCannotBypassLimit(t *testing.T) {
	b := &captureBuffer{limit: 17}
	if _, ok := any(b).(io.ReaderFrom); ok {
		t.Fatal("promoted ReaderFrom bypasses Write limit")
	}
	n, err := io.Copy(b, bytes.NewBufferString(strings.Repeat("x", 100)))
	if n != 17 || b.Len() != 17 || !errors.Is(err, ErrOutputIncomplete) {
		t.Fatalf("copy bypass: n=%d len=%d err=%v", n, b.Len(), err)
	}
}
