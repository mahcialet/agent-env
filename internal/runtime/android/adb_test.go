package android

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/execx"
)

type serverRunner struct {
	base  *testRunner
	ready atomic.Bool
}

func (r *serverRunner) probe(ctx context.Context) (int, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	return 41, r.ready.Load(), nil
}

func (r *serverRunner) Run(ctx context.Context, c execx.Command) (execx.Result, error) {
	if err := ctx.Err(); err != nil {
		return execx.Result{}, err
	}
	if len(c.Args) > 0 && c.Args[len(c.Args)-1] == "devices" {
		if strings.Join(c.Args, " ") != "-H 127.0.0.1 -P 5037 devices" {
			return execx.Result{}, errors.New("server probe could auto-start")
		}
		if !r.ready.Load() {
			return execx.Result{}, errors.New("no local ADB server")
		}
	}
	return r.base.Run(ctx, c)
}

type serverProcesses struct {
	base       *testProcess
	runner     *serverRunner
	mu         sync.Mutex
	starts     []string
	noReady    bool
	cancel     context.CancelFunc
	startError error
}

func (p *serverProcesses) Start(ctx context.Context, c execx.Command, out, errout string) (execx.ProcessIdentity, error) {
	if len(c.Args) > 0 && c.Args[0] == "-L" {
		p.mu.Lock()
		p.starts = append(p.starts, "shared-adb")
		p.mu.Unlock()
		if strings.Join(c.Args, " ") != "-L tcp:localhost:5037 start-server" {
			return execx.ProcessIdentity{}, errors.New("unscoped shared server startup")
		}
		if err := os.WriteFile(out, []byte("SDK server startup\n"), 0600); err != nil {
			return execx.ProcessIdentity{}, err
		}
		if err := os.WriteFile(errout, nil, 0600); err != nil {
			return execx.ProcessIdentity{}, err
		}
		if !p.noReady {
			p.runner.ready.Store(true)
		}
		if p.cancel != nil {
			p.cancel()
		}
		return execx.ProcessIdentity{PID: 84, StartID: "shared-server-job"}, p.startError
	}
	p.mu.Lock()
	p.starts = append(p.starts, "emulator")
	p.mu.Unlock()
	return p.base.Start(ctx, c, out, errout)
}
func (p *serverProcesses) Alive(ctx context.Context, id execx.ProcessIdentity) (bool, error) {
	if id.PID == 84 {
		return true, errors.New("shared server must never be used as emulator identity")
	}
	return p.base.Alive(ctx, id)
}

func TestSharedADBStartsOutsideEmulatorContainment(t *testing.T) {
	a, r, emulator := fixture(t)
	runner := &serverRunner{base: a.Runner.(*testRunner)}
	processes := &serverProcesses{base: emulator, runner: runner}
	a.Runner = runner
	a.Processes = processes
	a.ProbeADB = runner.probe
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(processes.starts, ",") != "shared-adb,emulator" {
		t.Fatalf("startup containment/order: %v", processes.starts)
	}
	if r.Android.ProcessID != 42 || r.Android.ProcessStart == "shared-server-job" {
		t.Fatal("shared server became lease-owned emulator")
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !runner.ready.Load() {
		t.Fatal("lease cleanup stopped shared server")
	}
	for _, name := range []string{"adb-server-start.json", "adb-server.stdout.log", "adb-server.stderr.log"} {
		if _, err := os.Stat(filepath.Join(r.Directory, name)); err != nil {
			t.Fatalf("shared startup diagnostics not retained: %v", err)
		}
	}
}

func TestSharedADBReadinessFailureNeverLaunchesEmulator(t *testing.T) {
	for _, mode := range []string{"canceled", "unavailable", "canceled_while_waiting"} {
		t.Run(mode, func(t *testing.T) {
			a, r, emulator := fixture(t)
			runner := &serverRunner{base: a.Runner.(*testRunner)}
			processes := &serverProcesses{base: emulator, runner: runner, noReady: true}
			a.Runner = runner
			a.Processes = processes
			a.ProbeADB = runner.probe
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wantErr := context.Canceled
			if mode == "canceled" {
				processes.cancel = cancel
			} else if mode == "canceled_while_waiting" {
				a.ProbeADB = func(ctx context.Context) (int, bool, error) {
					if len(processes.starts) > 0 {
						// Return an ordinary unready observation, then exercise the
						// wait loop's cancellation path without a wall-clock race.
						cancel()
						return 41, false, nil
					}
					return runner.probe(ctx)
				}
			} else {
				wantErr = errors.New("shared server unavailable after startup")
				// Inject the readiness failure only after startup. A wall-clock
				// deadline can expire during unrelated fixture I/O on busy CI.
				a.ProbeADB = func(ctx context.Context) (int, bool, error) {
					if len(processes.starts) > 0 {
						return 0, false, wantErr
					}
					return runner.probe(ctx)
				}
			}
			r, err := a.Create(ctx, r)
			if !errors.Is(err, wantErr) {
				t.Fatalf("readiness error = %v, want %v", err, wantErr)
			}
			if strings.Join(processes.starts, ",") != "shared-adb" || emulator.alive.Load() || r.Android.ProcessID != 0 {
				t.Fatalf("emulator launched before shared server was ready: %v %+v", processes.starts, r.Android)
			}
			if err := a.Destroy(context.Background(), r); err != nil {
				t.Fatalf("pre-emulator failure cannot compensate: %v", err)
			}
		})
	}
}

func TestExistingSharedADBNeedsNoDetachedStartup(t *testing.T) {
	a, r, emulator := fixture(t)
	runner := &serverRunner{base: a.Runner.(*testRunner)}
	runner.ready.Store(true)
	processes := &serverProcesses{base: emulator, runner: runner}
	a.Runner = runner
	a.Processes = processes
	a.ProbeADB = runner.probe
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(processes.starts, ",") != "emulator" {
		t.Fatalf("existing server replaced: %v", processes.starts)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func fakeVersionServer(t *testing.T, response string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				request := make([]byte, len("000chost:version"))
				if _, err := io.ReadFull(c, request); err != nil {
					return
				}
				if string(request) != "000chost:version" {
					t.Errorf("operational request sent to shared server: %q", request)
					return
				}
				_, _ = io.WriteString(c, response)
			}()
		}
	}()
	return listener.Addr().String()
}

func TestNativeADBVersionProbe(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		want           int
		bad            bool
	}{{"compatible", "OKAY00040029", 41, false}, {"different", "OKAY0004002a", 42, false}, {"malformed-length", "OKAYffff", 0, true}, {"malformed-body", "OKAY0004xxxx", 0, true}, {"failed", "FAIL", 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			address := fakeVersionServer(t, tc.response)
			got, exists, err := probeADBVersion(context.Background(), address)
			if !exists || (err != nil) != tc.bad || (!tc.bad && got != tc.want) {
				t.Fatalf("version=%d exists=%t err=%v", got, exists, err)
			}
		})
	}
}

func TestIncompatibleOrMalformedSharedServerNeverInvokesOperationalADB(t *testing.T) {
	for _, response := range []string{"OKAY0004002a", "OKAY0004xxxx"} {
		t.Run(response, func(t *testing.T) {
			a, r, p := fixture(t)
			address := fakeVersionServer(t, response)
			a.ProbeADB = func(ctx context.Context) (int, bool, error) { return probeADBVersion(ctx, address) }
			if _, err := a.Create(context.Background(), r); !errors.Is(err, app.ErrPrerequisite) {
				t.Fatalf("unsafe shared server must be a prerequisite failure: %v", err)
			}
			if p.command.Name != "" {
				t.Fatal("incompatible server triggered detached startup")
			}
			runner := a.Runner.(*testRunner)
			runner.mu.Lock()
			for _, cmd := range runner.commands {
				if strings.Join(cmd.Args, " ") != "version" {
					t.Errorf("operational SDK invocation could replace server: %+v", cmd)
				}
			}
			runner.mu.Unlock()
			if err := a.Destroy(context.Background(), r); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInspectRefusesServerReplacementWithoutRestart(t *testing.T) {
	a, r, _ := fixture(t)
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	a.ProbeADB = func(context.Context) (int, bool, error) { return 42, true, nil }
	runner := a.Runner.(*testRunner)
	runner.mu.Lock()
	runner.commands = nil
	runner.mu.Unlock()
	if _, err := a.Inspect(context.Background(), r); err == nil {
		t.Fatal("replaced server accepted")
	}
	runner.mu.Lock()
	for _, cmd := range runner.commands {
		if strings.Join(cmd.Args, " ") != "version" {
			t.Errorf("incompatible server received SDK request: %+v", cmd)
		}
	}
	runner.mu.Unlock()
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestRacingStarterErrorAcceptsCompatibleSharedServer(t *testing.T) {
	a, r, emulator := fixture(t)
	runner := &serverRunner{base: a.Runner.(*testRunner)}
	processes := &serverProcesses{base: emulator, runner: runner, startError: fmt.Errorf("starter exited before identity capture")}
	a.Runner = runner
	a.Processes = processes
	a.ProbeADB = runner.probe
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatalf("compatible shared service rejected because starter exited: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(r.Directory, "adb-server-start.json"))
	if err != nil || !strings.Contains(string(data), "starter exited") {
		t.Fatalf("startup diagnostic missing: %s %v", data, err)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}
