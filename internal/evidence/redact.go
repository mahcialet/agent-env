// Package evidence captures artifacts without persisting configured secrets.
package evidence

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

const replacement = "[REDACTED]"

// Secrets selects nonempty values for keys commonly used for credentials.
// It returns values only; callers must never log this collection itself.
func Secrets(env map[string]string) []string {
	var values []string
	for key, value := range env {
		if secretKey(key) {
			values = append(values, value)
		}
	}
	return normalize(values)
}

func secretKey(key string) bool {
	key = strings.ToUpper(key)
	// Hugging Face's public boolean control is not an authentication token.
	if key == "HF_HUB_DISABLE_IMPLICIT_TOKEN" {
		return false
	}
	return strings.Contains(key, "TOKEN") || strings.Contains(key, "PASSWORD") || strings.Contains(key, "SECRET") || strings.Contains(key, "KEY")
}

// InheritedSecrets selects only credential-like values. It does not expose or
// return a copy of the complete process environment.
func InheritedSecrets() []string {
	selected := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if ok && secretKey(key) {
			selected[key] = value
		}
	}
	return Secrets(selected)
}

func normalize(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i]) != len(result[j]) {
			return len(result[i]) > len(result[j])
		}
		return result[i] < result[j]
	})
	return result
}

// Redactor is a streaming writer. It retains at most the longest secret's byte
// length and emits only bytes that cannot begin a still-incomplete match.
// Matches use the longest value at each position; replacements are never rescanned.
// Flush and Close finalize the stream without closing the destination.
type Redactor struct {
	mu      sync.Mutex
	dst     io.Writer
	values  [][]byte
	max     int
	pending []byte
	err     error
	closed  bool
}

func NewRedactor(dst io.Writer, secrets []string) *Redactor {
	r := &Redactor{dst: dst}
	for _, value := range normalize(secrets) {
		r.values = append(r.values, []byte(value))
		if len(value) > r.max {
			r.max = len(value)
		}
	}
	return r
}

func (r *Redactor) emit(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	n, err := r.dst.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		r.err = err
	}
	return err
}

func (r *Redactor) consume(out []byte) []byte {
	for _, value := range r.values {
		if bytes.HasPrefix(r.pending, value) {
			out = append(out, replacement...)
			r.pending = r.pending[len(value):]
			return out
		}
	}
	out = append(out, r.pending[0])
	r.pending = r.pending[1:]
	return out
}

func (r *Redactor) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return 0, r.err
	}
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	if r.max == 0 {
		n, err := r.dst.Write(p)
		if err == nil && n != len(p) {
			err = io.ErrShortWrite
		}
		r.err = err
		return n, err
	}
	out := make([]byte, 0, 4096)
	for i, b := range p {
		r.pending = append(r.pending, b)
		for len(r.pending) >= r.max {
			out = r.consume(out)
		}
		if len(out) >= 4096 {
			if err := r.emit(out); err != nil {
				return i + 1, err
			}
			out = out[:0]
		}
	}
	if err := r.emit(out); err != nil {
		return len(p), err
	}
	return len(p), nil
}

func (r *Redactor) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	if r.closed {
		return nil
	}
	r.closed = true
	out := make([]byte, 0, 4096)
	for len(r.pending) != 0 {
		out = r.consume(out)
		if len(out) >= 4096 {
			if err := r.emit(out); err != nil {
				return err
			}
			out = out[:0]
		}
	}
	return r.emit(out)
}

func (r *Redactor) Close() error { return r.Flush() }

func RedactString(text string, secrets []string) string {
	var out strings.Builder
	r := NewRedactor(&out, secrets)
	_, _ = r.Write([]byte(text))
	_ = r.Close()
	return out.String()
}
