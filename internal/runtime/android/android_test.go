package android

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type testRunner struct {
	mu       sync.Mutex
	commands []execx.Command
	boot     string
}

func (r *testRunner) Run(_ context.Context, c execx.Command) (execx.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, c)
	if len(c.Args) > 6 && c.Args[6] == "shell" {
		return execx.Result{Stdout: r.boot}, nil
	}
	return execx.Result{Stdout: "fake prerequisite version"}, nil
}

type testProcess struct {
	alive         atomic.Bool
	mu            sync.Mutex
	name          string
	listener      net.Listener
	conversations [][]string
	command       execx.Command
	startError    error
	partial       bool
	onName        func()
}

func (p *testProcess) Start(_ context.Context, c execx.Command, stdout, stderr string) (execx.ProcessIdentity, error) {
	p.command = c
	if p.startError != nil {
		if p.partial {
			return execx.ProcessIdentity{PID: 42}, p.startError
		}
		return execx.ProcessIdentity{}, p.startError
	}
	port := c.Args[3]
	p.name = c.Args[1]
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return execx.ProcessIdentity{}, err
	}
	p.listener = l
	p.alive.Store(true)
	if err := os.WriteFile(stdout, []byte("emulator stdout\n"), 0600); err != nil {
		return execx.ProcessIdentity{}, err
	}
	if err := os.WriteFile(stderr, []byte("emulator stderr\n"), 0600); err != nil {
		return execx.ProcessIdentity{}, err
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go p.serve(conn)
		}
	}()
	return execx.ProcessIdentity{PID: 42, StartID: "process-birth"}, nil
}
func (p *testProcess) Alive(_ context.Context, id execx.ProcessIdentity) (bool, error) {
	return p.alive.Load() && id.PID == 42 && id.StartID == "process-birth", nil
}
func (p *testProcess) serve(c net.Conn) {
	defer c.Close()
	p.mu.Lock()
	index := len(p.conversations)
	p.conversations = append(p.conversations, nil)
	p.mu.Unlock()
	fmt.Fprint(c, "Android Console: type 'help' for commands\r\nOK\r\n")
	s := bufio.NewScanner(c)
	for s.Scan() {
		command := s.Text()
		p.mu.Lock()
		p.conversations[index] = append(p.conversations[index], command)
		name := p.name
		p.mu.Unlock()
		switch command {
		case "avd name":
			p.mu.Lock()
			hook := p.onName
			p.mu.Unlock()
			if hook != nil {
				hook()
			}
			fmt.Fprintf(c, "%s\r\nOK\r\n", name)
		case "kill":
			p.alive.Store(false)
			p.listener.Close()
			fmt.Fprint(c, "OK: killing emulator, bye bye\r\n")
			return
		default:
			fmt.Fprint(c, "KO: unsupported\r\n")
		}
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) (Adapter, domain.Runtime, *testProcess) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "Android 日本語 with spaces")
	sdk := filepath.Join(base, "sdk")
	avds := filepath.Join(base, "templates")
	template := filepath.Join(avds, "Pixel.avd")
	writeFixture(t, executable(sdk, "emulator", "emulator"), "")
	writeFixture(t, executable(sdk, "platform-tools", "adb"), "")
	writeFixture(t, filepath.Join(sdk, "system-images", "test", "system.img"), "system")
	writeFixture(t, filepath.Join(sdk, "system-images", "test", "userdata.img"), "seed")
	writeFixture(t, filepath.Join(avds, "Pixel.ini"), "path="+template+"\n")
	writeFixture(t, filepath.Join(template, "config.ini"), "image.sysdir.1=system-images/test/\nhw.cpu.arch=x86_64\nhw.ramSize=1024\n")
	writeFixture(t, filepath.Join(template, "userdata-qemu.img"), "private original user data")
	t.Setenv("ANDROID_HOME", sdk)
	t.Setenv("ANDROID_SDK_ROOT", sdk)
	t.Setenv("ANDROID_AVD_HOME", avds)
	p := &testProcess{}
	a := Adapter{Runner: &testRunner{boot: "1\n"}, Processes: p}
	d, err := a.Validate(context.Background(), "Pixel")
	if err != nil {
		t.Fatal(err)
	}
	r := domain.Runtime{LeaseID: "lease-one", Name: "android", Type: "android-emulator", Directory: filepath.Join(base, "leases", "lease-one", "android", "android"), Started: true}
	port := 0
	for n := 5554; n <= 5682; n += 2 {
		if portAvailable(n) && portAvailable(n+1) {
			port = n
			break
		}
	}
	if port == 0 {
		t.Fatal("no test emulator port pair available")
	}
	d.ConsolePort = port
	d.ADBPort = port + 1
	d.Serial = "emulator-" + strconv.Itoa(port)
	d.AVDName = "ae-lease-one"
	d.AVDHome = filepath.Join(r.Directory, "avd")
	d.AVDPath = filepath.Join(d.AVDHome, d.AVDName+".avd")
	r.Android = &d
	t.Cleanup(func() {
		if p.listener != nil {
			p.listener.Close()
		}
	})
	return a, r, p
}

func TestCreateUsesPrivateAVDAndOwnedConsoleCleanup(t *testing.T) {
	a, r, p := fixture(t)
	ctx := context.Background()
	r, err := a.Create(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Android.State != "booting" || r.Android.ProcessID != 42 {
		t.Fatalf("runtime not launched: %+v", r.Android)
	}
	if p.command.Env["ANDROID_AVD_HOME"] != r.Android.AVDHome || p.command.Name != executable(r.Android.SDKPath, "emulator", "emulator") {
		t.Fatalf("native command mismatch: %+v", p.command)
	}
	if _, err := os.Stat(filepath.Join(r.Android.AVDPath, "userdata-qemu.img")); !os.IsNotExist(err) {
		t.Fatal("template mutable userdata was copied")
	}
	cfg, err := readINI(filepath.Join(r.Android.AVDPath, "config.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["disk.dataPartition.path"] != filepath.Join(r.Android.AVDPath, "userdata-qemu.img") {
		t.Fatal("writable path not isolated")
	}
	ob, err := a.Inspect(ctx, r)
	if err != nil || !ob.Ready || len(ob.Resources) != 1 || ob.Resources[0].Metadata["serial"] != r.Android.Serial || ob.Resources[0].Metadata["lease"] != r.LeaseID || ob.Resources[0].ID != r.Name+":emulator" {
		t.Fatalf("observation %+v: %v", ob, err)
	}
	if err := a.Destroy(ctx, r); err != nil {
		t.Fatal(err)
	}
	if p.alive.Load() {
		t.Fatal("emulator remains alive")
	}
	if _, err := os.Stat(r.Android.AVDHome); !os.IsNotExist(err) {
		t.Fatal("writable AVD remains")
	}
	for _, name := range []string{"android-owner.json", "emulator.stdout.log", "emulator.stderr.log"} {
		if _, err := os.Stat(filepath.Join(r.Directory, name)); err != nil {
			t.Fatalf("missing retained evidence %s: %v", name, err)
		}
	}
	if err := a.Destroy(ctx, r); err != nil {
		t.Fatalf("repeat destroy: %v", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	found := false
	for _, commands := range p.conversations {
		for i, c := range commands {
			if c == "kill" {
				found = true
				if i == 0 || commands[i-1] != "avd name" {
					t.Fatalf("kill without same-connection ownership: %v", commands)
				}
			}
		}
	}
	if !found {
		t.Fatal("no console kill recorded")
	}
}

func TestDestroyRefusesReusedConsoleAndMarkerTampering(t *testing.T) {
	for _, mode := range []string{"console", "marker", "missing"} {
		t.Run(mode, func(t *testing.T) {
			a, r, p := fixture(t)
			r, err := a.Create(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "console":
				p.mu.Lock()
				p.name = "unrelated-avd"
				p.mu.Unlock()
			case "marker":
				m, err := loadMarker(r)
				if err != nil {
					t.Fatal(err)
				}
				m.Android.AVDPath = filepath.Join(t.TempDir(), "other.avd")
				if err := saveMarker(r, m); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(markerPath(r)); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.Destroy(context.Background(), r); !errors.Is(err, domain.ErrResourceIdentity) {
				t.Fatalf("wanted quarantine, got %v", err)
			}
			if !p.alive.Load() {
				t.Fatal("unproven emulator stopped")
			}
			if _, err := os.Stat(r.Android.AVDPath); err != nil {
				t.Fatal("unproven writable AVD removed")
			}
		})
	}
}

func TestPartialLaunchRecovery(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(strconv.FormatBool(partial), func(t *testing.T) {
			a, r, p := fixture(t)
			p.startError = errors.New("launch failed")
			p.partial = partial
			if partial {
				p.startError = errors.Join(p.startError, execx.ErrProcessTreeUnconfirmed)
			}
			r, err := a.Create(context.Background(), r)
			if err == nil {
				t.Fatal("expected launch error")
			}
			err = a.Destroy(context.Background(), r)
			if partial {
				if !errors.Is(err, domain.ErrResourceIdentity) {
					t.Fatalf("partial launch should quarantine: %v", err)
				}
			} else if err != nil {
				t.Fatalf("prelaunch failure should compensate: %v", err)
			}
		})
	}
}

func TestInspectRecoversIdentityFromDurableMarker(t *testing.T) {
	a, r, _ := fixture(t)
	original := *r.Android
	created, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	r.Android = &original
	ob, err := a.Inspect(context.Background(), r)
	if err != nil || !ob.Ready {
		t.Fatalf("recover identity: %+v %v", ob, err)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if created.Android.ProcessID == 0 {
		t.Fatal("missing process identity")
	}
}

func TestMissingProcessReconcilesAbsentAndLeavesSibling(t *testing.T) {
	a, r, p := fixture(t)
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b, s, q := fixture(t)
	s, err = b.Create(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if r.Android.ConsolePort == s.Android.ConsolePort || r.Android.AVDPath == s.Android.AVDPath {
		t.Fatal("concurrent writable state collided")
	}
	p.alive.Store(false)
	p.listener.Close()
	ob, err := a.Inspect(context.Background(), r)
	if err != nil || ob.Exists || ob.Ready {
		t.Fatalf("dead process observation: %+v %v", ob, err)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	ob, err = b.Inspect(context.Background(), s)
	if err != nil || !ob.Ready || !q.alive.Load() {
		t.Fatalf("sibling affected: %+v %v", ob, err)
	}
	if err := b.Destroy(context.Background(), s); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsMissingAndUnsafePrerequisites(t *testing.T) {
	for _, mode := range []string{"sdk", "template", "image", "lock", "external-write"} {
		t.Run(mode, func(t *testing.T) {
			a, r, _ := fixture(t)
			switch mode {
			case "sdk":
				t.Setenv("ANDROID_HOME", t.TempDir())
				t.Setenv("ANDROID_SDK_ROOT", os.Getenv("ANDROID_HOME"))
			case "template":
				if err := os.Remove(filepath.Join(r.Android.TemplatePath, "config.ini")); err != nil {
					t.Fatal(err)
				}
			case "image":
				if err := os.Remove(filepath.Join(r.Android.SystemImage, "userdata.img")); err != nil {
					t.Fatal(err)
				}
			case "lock":
				writeFixture(t, filepath.Join(r.Android.TemplatePath, "hardware-qemu.ini.lock"), "")
			case "external-write":
				f, err := os.OpenFile(filepath.Join(r.Android.TemplatePath, "config.ini"), os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				_, err = f.WriteString("sdcard.path=" + filepath.Join(t.TempDir(), "shared.img") + "\n")
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := a.Validate(context.Background(), "Pixel"); err == nil {
				t.Fatal("unsafe prerequisite accepted")
			}
		})
	}
}

func TestDoctorReportsTemplateAndVersions(t *testing.T) {
	a, _, _ := fixture(t)
	report, err := a.Doctor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report["templates"] != "Pixel" || !strings.Contains(report["emulator"], "version") || report["acceleration"] == "" {
		t.Fatalf("doctor report: %v", report)
	}
}

func TestBootingLaunchReturnsForAppReadinessAndCompensation(t *testing.T) {
	a, r, p := fixture(t)
	a.Runner = &testRunner{boot: "0"}
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	ob, err := a.Inspect(context.Background(), r)
	if err != nil || ob.Ready || !ob.Exists {
		t.Fatalf("booting observation: %+v %v", ob, err)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if p.alive.Load() {
		t.Fatal("compensation left emulator alive")
	}
}

func TestCanceledDestroyPreservesPreparedAndStoppedState(t *testing.T) {
	for _, phase := range []string{"prepared", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			a, r, _ := fixture(t)
			if err := os.MkdirAll(r.Directory, 0700); err != nil {
				t.Fatal(err)
			}
			if err := makePrivateAVD(r); err != nil {
				t.Fatal(err)
			}
			if err := saveMarker(r, marker{LeaseID: r.LeaseID, Runtime: r.Name, Android: *r.Android, Phase: phase}); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := a.Destroy(ctx, r); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled destroy: %v", err)
			}
			if _, err := os.Stat(r.Android.AVDPath); err != nil {
				t.Fatal("canceled cleanup removed writable state")
			}
			m, err := loadMarker(r)
			if err != nil || m.Phase != phase {
				t.Fatalf("canceled cleanup changed marker: %+v %v", m, err)
			}
		})
	}
}

func TestCancellationDuringIdentityHandshakeCannotKill(t *testing.T) {
	a, r, p := fixture(t)
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.mu.Lock()
	p.onName = cancel
	p.mu.Unlock()
	if err := a.Destroy(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatalf("handshake cancellation: %v", err)
	}
	if !p.alive.Load() {
		t.Fatal("canceled handshake killed emulator")
	}
	if _, err := os.Stat(r.Android.AVDPath); err != nil {
		t.Fatal("canceled handshake removed writable AVD")
	}
	p.mu.Lock()
	p.onName = nil
	for _, commands := range p.conversations {
		for _, command := range commands {
			if command == "kill" {
				t.Error("kill sent after handshake cancellation")
			}
		}
	}
	p.mu.Unlock()
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessPinsLocalADBServer(t *testing.T) {
	a, r, _ := fixture(t)
	t.Setenv("ADB_SERVER_SOCKET", "tcp:remote.example:5555")
	t.Setenv("ANDROID_ADB_SERVER_PORT", "6000")
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Inspect(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	runner := a.Runner.(*testRunner)
	runner.mu.Lock()
	commands := append([]execx.Command(nil), runner.commands...)
	runner.mu.Unlock()
	found := false
	for _, cmd := range commands {
		if len(cmd.Args) > 6 && cmd.Args[6] == "shell" {
			found = true
			if strings.Join(cmd.Args[:6], " ") != "-H 127.0.0.1 -P 5037 -s "+r.Android.Serial {
				t.Fatalf("unscoped adb: %+v", cmd)
			}
			for _, key := range []string{"ADB_SERVER_SOCKET", "ANDROID_ADB_SERVER_ADDRESS", "ANDROID_ADB_SERVER_PORT", "ANDROID_SERIAL"} {
				if !strings.Contains(strings.Join(cmd.UnsetEnv, ","), key) {
					t.Fatalf("adb inherited %s", key)
				}
			}
		}
	}
	if !found {
		t.Fatal("no adb readiness invocation")
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestReleasedObservationRefusesReappearedWritableState(t *testing.T) {
	a, r, _ := fixture(t)
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(r.Android.AVDHome, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Inspect(context.Background(), r); !errors.Is(err, domain.ErrResourceIdentity) {
		t.Fatalf("reappeared writable state accepted: %v", err)
	}
}

func TestModernSystemImageSeedsPrivateDataWithoutFabricatedImage(t *testing.T) {
	a, r, _ := fixture(t)
	if err := os.Remove(filepath.Join(r.Android.SystemImage, "userdata.img")); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(r.Android.SystemImage, "data", "local", "seed"), "SDK data seed")
	if _, err := a.Validate(context.Background(), "Pixel"); err != nil {
		t.Fatalf("modern image rejected: %v", err)
	}
	r, err := a.Create(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := readINI(filepath.Join(r.Android.AVDPath, "config.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg["disk.dataPartition.initPath"]; ok {
		t.Fatal("modern image references nonexistent userdata.img")
	}
	if _, err := os.Stat(filepath.Join(r.Android.SystemImage, "userdata.img")); !os.IsNotExist(err) {
		t.Fatal("adapter fabricated shared SDK userdata")
	}
	if err := a.Destroy(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestStoppedMarkerNeverAuthorizesCleanupOfReappearedResource(t *testing.T) {
	for _, mode := range []string{"tree-only", "port-only", "both"} {
		t.Run(mode, func(t *testing.T) {
			a, r, p := fixture(t)
			r, err := a.Create(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			m, err := loadMarker(r)
			if err != nil {
				t.Fatal(err)
			}
			m.Phase = "stopped"
			if err := saveMarker(r, m); err != nil {
				t.Fatal(err)
			}
			if mode == "tree-only" {
				p.listener.Close()
			}
			if mode == "port-only" {
				p.alive.Store(false)
			}
			if _, err := a.Inspect(context.Background(), r); !errors.Is(err, domain.ErrResourceIdentity) {
				t.Fatalf("stopped active resource accepted: %v", err)
			}
			if err := a.Destroy(context.Background(), r); !errors.Is(err, domain.ErrResourceIdentity) {
				t.Fatalf("stopped active resource cleanup accepted: %v", err)
			}
			if mode != "port-only" && !p.alive.Load() {
				t.Fatal("stopped marker authorized a kill")
			}
			if _, err := os.Stat(r.Android.AVDPath); err != nil {
				t.Fatal("stopped marker authorized writable cleanup")
			}
			p.mu.Lock()
			for _, commands := range p.conversations {
				for _, command := range commands {
					if command == "kill" {
						t.Error("unexpected kill")
					}
				}
			}
			p.mu.Unlock()
			p.alive.Store(false)
			p.listener.Close()
			if err := a.Destroy(context.Background(), r); err != nil {
				t.Fatal(err)
			}
		})
	}
}
