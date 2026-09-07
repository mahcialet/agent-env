package android

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

type console struct {
	conn   net.Conn
	reader *bufio.Reader
	ctx    context.Context
	stop   func() bool
}

func (c *console) close() {
	if c.stop != nil {
		c.stop()
	}
	_ = c.conn.Close()
}

func (c *console) response() (string, error) {
	var lines []string
	total := 0
	for {
		bytes, err := c.reader.ReadSlice('\n')
		total += len(bytes)
		if total > 16*1024 {
			return "", fmt.Errorf("emulator console response exceeds limit")
		}
		if err != nil {
			return "", err
		}
		line := strings.TrimSpace(string(bytes))
		if line == "OK" || strings.HasPrefix(line, "OK:") {
			return strings.Join(lines, "\n"), nil
		}
		if strings.HasPrefix(line, "KO") {
			return "", fmt.Errorf("emulator console rejected command")
		}
		lines = append(lines, line)
	}
}

func (c *console) command(command string) (string, error) {
	if err := c.ctx.Err(); err != nil {
		return "", err
	}
	if _, err := io.WriteString(c.conn, command+"\n"); err != nil {
		return "", err
	}
	return c.response()
}

// connectConsole only speaks to the reserved loopback console. Ownership is
// checked again on this same TCP connection before any destructive command.
func connectConsole(ctx context.Context, port int, name string) (*console, error) {
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	c := &console{conn: conn, reader: bufio.NewReaderSize(conn, 16*1024), ctx: ctx}
	c.stop = context.AfterFunc(ctx, func() { _ = conn.Close() })
	fail := func(err error) (*console, error) { c.close(); return nil, errors.Join(err, ctx.Err()) }
	banner, err := c.response()
	if err != nil {
		return fail(err)
	}
	if !strings.Contains(banner, "Android Console") {
		return fail(errors.Join(domain.ErrResourceIdentity, fmt.Errorf("unexpected emulator console banner")))
	}
	if strings.Contains(banner, "Authentication required") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fail(err)
		}
		path := filepath.Join(home, ".emulator_console_auth_token")
		st, err := os.Lstat(path)
		if err != nil {
			return fail(fmt.Errorf("emulator console authentication token unavailable"))
		}
		if !st.Mode().IsRegular() || st.Size() > 1024 {
			return fail(fmt.Errorf("invalid emulator console authentication token file"))
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return fail(fmt.Errorf("emulator console authentication token unreadable"))
		}
		token := strings.TrimSpace(string(b))
		if token == "" || strings.ContainsAny(token, "\r\n\x00\t ") {
			return fail(fmt.Errorf("invalid emulator console authentication token"))
		}
		if _, err = c.command("auth " + token); err != nil {
			return fail(fmt.Errorf("emulator console authentication failed"))
		}
	}
	observed, err := c.command("avd name")
	if err != nil {
		return fail(err)
	}
	if strings.TrimSpace(observed) != name {
		return fail(errors.Join(domain.ErrResourceIdentity, fmt.Errorf("emulator console AVD identity mismatch")))
	}
	return c, nil
}

func portAvailable(port int) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
