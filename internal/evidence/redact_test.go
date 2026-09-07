package evidence

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestSecretSelection(t *testing.T) {
	env := map[string]string{"API_TOKEN": "long-token", "password": "pw", "client_SECRET": "secret", "SSH_KEY": "key", "HOME": "visible", "EMPTY_TOKEN": "", "DUP_TOKEN": "pw"}
	want := []string{"long-token", "secret", "key", "pw"}
	if got := Secrets(env); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	t.Setenv("AGENT_ENV_TEST_PRIVATE_TOKEN", "select-this-value")
	t.Setenv("AGENT_ENV_TEST_ORDINARY", "do-not-select-this-value")
	values := InheritedSecrets()
	if !contains(values, "select-this-value") || contains(values, "do-not-select-this-value") {
		t.Fatal("inherited selection did not honor secret keys")
	}
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestRedactionEveryByteBoundary(t *testing.T) {
	secrets := []string{"token", "token-long", "日本語秘密", `a"quote`, "aba", ""}
	input := `prefix token-long token 日本語秘密 a"quote ababa suffix`
	want := `prefix [REDACTED] [REDACTED] [REDACTED] [REDACTED] [REDACTED]ba suffix`
	if got := RedactString(input, secrets); got != want {
		t.Fatalf("%q want %q", got, want)
	}
	for split := 0; split <= len(input); split++ {
		var out bytes.Buffer
		r := NewRedactor(&out, secrets)
		if n, err := r.Write([]byte(input[:split])); err != nil || n != split {
			t.Fatal(n, err)
		}
		if _, err := r.Write([]byte(input[split:])); err != nil {
			t.Fatal(err)
		}
		if err := r.Flush(); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Fatalf("split %d: %q", split, out.String())
		}
	}
	var out bytes.Buffer
	r := NewRedactor(&out, secrets)
	for _, b := range []byte(input) {
		if _, err := r.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if out.String() != want {
		t.Fatal(out.String())
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte("late")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write after finalization: %v", err)
	}
}

func TestRedactorDoesNotRetainCommandOutput(t *testing.T) {
	r := NewRedactor(io.Discard, []string{"private-value"})
	payload := bytes.Repeat([]byte("no secret in these output bytes "), 32768)
	if n, err := r.Write(payload); err != nil || n != len(payload) {
		t.Fatal(n, err)
	}
	if len(r.pending) >= len("private-value") {
		t.Fatalf("retained %d bytes", len(r.pending))
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

type brokenWriter struct {
	err   error
	short bool
}

func (w brokenWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) / 2, nil
	}
	return 0, w.err
}
func TestRedactorPropagatesErrors(t *testing.T) {
	sentinel := errors.New("destination unavailable")
	for _, secrets := range [][]string{nil, {"secret"}} {
		for _, writer := range []brokenWriter{{err: sentinel}, {short: true}} {
			r := NewRedactor(writer, secrets)
			_, err := r.Write([]byte(strings.Repeat("ordinary text ", 1000)))
			want := sentinel
			if writer.short {
				want = io.ErrShortWrite
			}
			if !errors.Is(err, want) {
				t.Fatalf("write error %v want %v", err, want)
			}
			if err := r.Flush(); !errors.Is(err, want) {
				t.Fatalf("flush lost error: %v", err)
			}
			if _, err := r.Write([]byte("more")); !errors.Is(err, want) {
				t.Fatalf("write lost error: %v", err)
			}
		}
	}
	r := NewRedactor(brokenWriter{err: sentinel}, []string{"secret"})
	if _, err := r.Write([]byte("sec")); err != nil {
		t.Fatal(err)
	}
	if err := r.Flush(); !errors.Is(err, sentinel) {
		t.Fatalf("final drain: %v", err)
	}
}
