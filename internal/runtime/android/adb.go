package android

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

func adbRoutingEnvironment() []string {
	return []string{"ADB_SERVER_SOCKET", "ANDROID_ADB_SERVER_ADDRESS", "ANDROID_ADB_SERVER_PORT", "ANDROID_SERIAL"}
}

// Probe only the documented host:version service. Invoking an SDK adb client
// here is unsafe: even -H may kill an existing incompatible server automatically.
func probeADBVersion(ctx context.Context, address string) (int, bool, error) {
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", address)
	if errors.Is(err, syscall.ECONNREFUSED) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline := time.Now().Add(3 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	if _, err = io.WriteString(conn, "000chost:version"); err != nil {
		return 0, true, err
	}
	status := make([]byte, 4)
	if _, err = io.ReadFull(conn, status); err != nil {
		return 0, true, errors.Join(err, ctx.Err())
	}
	if string(status) != "OKAY" {
		return 0, true, fmt.Errorf("shared local ADB server rejected version probe")
	}
	if _, err = io.ReadFull(conn, status); err != nil {
		return 0, true, err
	}
	size, err := strconv.ParseUint(string(status), 16, 16)
	if err != nil || size == 0 || size > 8 {
		return 0, true, fmt.Errorf("malformed shared ADB version length")
	}
	body := make([]byte, int(size))
	if _, err = io.ReadFull(conn, body); err != nil {
		return 0, true, err
	}
	version, err := strconv.ParseUint(string(body), 16, 31)
	if err != nil || version == 0 {
		return 0, true, fmt.Errorf("malformed shared ADB version")
	}
	return int(version), true, nil
}

func (a Adapter) probeADB(ctx context.Context) (int, bool, error) {
	if a.ProbeADB != nil {
		return a.ProbeADB(ctx)
	}
	return probeADBVersion(ctx, "127.0.0.1:5037")
}

func parseADBProtocol(text string) (int, error) {
	for _, line := range strings.Split(text, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "Android Debug Bridge version 1.0."); ok {
			version, err := strconv.Atoi(value)
			if err == nil && version > 0 {
				return version, nil
			}
		}
	}
	return 0, fmt.Errorf("SDK adb version does not identify a supported server protocol")
}

func (a Adapter) adbProtocol(ctx context.Context, sdk string) (int, error) {
	result, err := a.runner().Run(ctx, execx.Command{Name: executable(sdk, "platform-tools", "adb"), Args: []string{"version"}, UnsetEnv: adbRoutingEnvironment(), Timeout: 5 * time.Second})
	if err != nil {
		return 0, err
	}
	return parseADBProtocol(result.Stdout + "\n" + result.Stderr)
}

func (a Adapter) compatibleADB(ctx context.Context, expected int) (bool, error) {
	version, exists, err := a.probeADB(ctx)
	if err != nil {
		return false, fmt.Errorf("shared local ADB server protocol: %w", err)
	}
	if !exists {
		return false, nil
	}
	if version != expected {
		return false, fmt.Errorf("shared local ADB server protocol %d differs from SDK protocol %d; refusing automatic replacement", version, expected)
	}
	return true, nil
}

// The shared SDK server must exist outside the Emulator's process containment.
// Otherwise Windows can inherit it into the Emulator Job Object and keep a
// destroyed lease alive indefinitely. This startup is a host SDK service, not
// a lease-owned emulator: never stop it as part of lease compensation.
func (a Adapter) ensureADBServer(ctx context.Context, r domain.Runtime) error {
	expected, err := a.adbProtocol(ctx, r.Android.SDKPath)
	if err != nil {
		return err
	}
	ready, err := a.compatibleADB(ctx, expected)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := execx.Command{Name: executable(r.Android.SDKPath, "platform-tools", "adb"), Args: []string{"-L", "tcp:localhost:5037", "start-server"}, Dir: r.Directory, UnsetEnv: adbRoutingEnvironment()}
	id, startErr := a.processes().Start(ctx, cmd, filepath.Join(r.Directory, "adb-server.stdout.log"), filepath.Join(r.Directory, "adb-server.stderr.log"))
	// Keep launch evidence even when a canceled startup has incomplete identity.
	// Do not copy this shared-service identity into AndroidEmulator.ProcessID.
	startDiagnostic := ""
	if startErr != nil {
		startDiagnostic = startErr.Error()
	}
	record, err := json.Marshal(struct {
		Process execx.ProcessIdentity `json:"process"`
		Shared  bool                  `json:"shared"`
		Error   string                `json:"error,omitempty"`
	}{Process: id, Shared: true, Error: startDiagnostic})
	if err == nil {
		err = os.WriteFile(filepath.Join(r.Directory, "adb-server-start.json"), record, 0600)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("start shared local ADB server"), startErr, err)
	}
	readyCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for {
		if err := readyCtx.Err(); err != nil {
			return errors.Join(fmt.Errorf("shared local ADB server readiness: %w", err), startErr)
		}
		ready, err := a.compatibleADB(readyCtx, expected)
		if err != nil {
			return errors.Join(err, startErr)
		}
		if ready {
			return nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-readyCtx.Done():
			timer.Stop()
			return errors.Join(fmt.Errorf("shared local ADB server readiness: %w", readyCtx.Err()), startErr)
		case <-timer.C:
		}
	}
}
