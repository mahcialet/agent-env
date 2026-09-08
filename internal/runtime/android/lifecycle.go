package android

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/execx"
)

type marker struct {
	LeaseID string                 `json:"lease_id"`
	Runtime string                 `json:"runtime"`
	Android domain.AndroidEmulator `json:"android"`
	Phase   string                 `json:"phase"`
}

func identityError(message string) error {
	return errors.Join(domain.ErrResourceIdentity, errors.New(message))
}
func markerPath(r domain.Runtime) string { return filepath.Join(r.Directory, "android-owner.json") }

func checkLayout(r domain.Runtime) error {
	d := r.Android
	if d == nil || r.LeaseID == "" || r.Name == "" || !safeName(d.AVDName) || !filepath.IsAbs(r.Directory) ||
		filepath.Base(r.Directory) != r.Name || filepath.Base(filepath.Dir(r.Directory)) != "android" || filepath.Base(filepath.Dir(filepath.Dir(r.Directory))) != r.LeaseID ||
		d.AVDHome != filepath.Join(r.Directory, "avd") || d.AVDPath != filepath.Join(d.AVDHome, d.AVDName+".avd") ||
		d.ConsolePort < 5554 || d.ConsolePort > 5682 || d.ConsolePort%2 != 0 || d.ADBPort != d.ConsolePort+1 || d.Serial != "emulator-"+strconv.Itoa(d.ConsolePort) {
		return identityError("invalid Android resource identity or writable layout")
	}
	for _, path := range []string{filepath.Dir(filepath.Dir(filepath.Dir(r.Directory))), filepath.Dir(filepath.Dir(r.Directory)), filepath.Dir(r.Directory), r.Directory, d.AVDHome, d.AVDPath, markerPath(r)} {
		st, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return identityError("Android writable ownership path is a symlink")
		}
	}
	return nil
}

func saveMarker(r domain.Runtime, m marker) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(r.Directory, ".android-owner-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, markerPath(r))
}

func loadMarker(r domain.Runtime) (marker, error) {
	var m marker
	if err := checkLayout(r); err != nil {
		return m, err
	}
	st, err := os.Stat(markerPath(r))
	if err != nil {
		return m, err
	}
	if !st.Mode().IsRegular() || st.Size() > 32*1024 {
		return m, identityError("invalid Android ownership marker")
	}
	b, err := os.ReadFile(markerPath(r))
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(b, &m); err != nil {
		return m, identityError("unreadable Android ownership marker")
	}
	expected, observed := *r.Android, m.Android
	// A process may have started durably before the caller could save its identity.
	// Recover from the marker only when the registry has not recorded a different one.
	if expected.ProcessID != 0 && (expected.ProcessID != observed.ProcessID || expected.ProcessStart != observed.ProcessStart) {
		return m, identityError("Android process identity changed")
	}
	expected.ProcessID = 0
	expected.ProcessStart = ""
	expected.State = ""
	observed.ProcessID = 0
	observed.ProcessStart = ""
	observed.State = ""
	if m.LeaseID != r.LeaseID || m.Runtime != r.Name || !reflect.DeepEqual(expected, observed) {
		return m, identityError("Android ownership marker disagrees with registry")
	}
	switch m.Phase {
	case "prepared", "launching", "launched", "stopped", "destroyed":
	default:
		return m, identityError("unknown Android ownership phase")
	}
	return m, nil
}

func makePrivateAVD(r domain.Runtime) error {
	d := r.Android
	cfg, err := readINI(filepath.Join(d.TemplatePath, "config.ini"))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(d.AVDHome, 0700); err != nil {
		return err
	}
	if err = os.Mkdir(d.AVDPath, 0700); err != nil {
		return err
	}
	// Deliberately do not copy the template's userdata, cache, SD card, locks or
	// snapshots. The emulator initializes private userdata from the SDK seed.
	out := map[string]string{"AvdId": d.AVDName, "avd.ini.displayname": d.AVDName, "avd.ini.encoding": "UTF-8", "image.sysdir.1": d.SystemImage + string(filepath.Separator), "disk.dataPartition.path": filepath.Join(d.AVDPath, "userdata-qemu.img"), "disk.cachePartition.path": filepath.Join(d.AVDPath, "cache.img"), "hw.sdCard": "no", "snapshot.present": "false"}
	if st, err := os.Stat(filepath.Join(d.SystemImage, "userdata.img")); err == nil && st.Mode().IsRegular() {
		out["disk.dataPartition.initPath"] = filepath.Join(d.SystemImage, "userdata.img")
	}
	for _, k := range []string{"abi.type", "hw.cpu.arch", "hw.cpu.ncore", "hw.ramSize", "hw.lcd.width", "hw.lcd.height", "hw.lcd.density", "hw.gpu.enabled", "hw.gpu.mode", "hw.keyboard", "hw.mainKeys", "hw.dPad", "hw.battery", "hw.accelerometer", "hw.gps", "hw.audioInput", "hw.audioOutput", "hw.camera.back", "hw.camera.front", "disk.dataPartition.size", "tag.id", "tag.display", "target", "PlayStore.enabled"} {
		if v := cfg[k]; v != "" {
			out[k] = v
		}
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var text strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&text, "%s=%s\n", k, out[k])
	}
	if err = os.WriteFile(filepath.Join(d.AVDPath, "config.ini"), []byte(text.String()), 0600); err != nil {
		return err
	}
	ini := fmt.Sprintf("avd.ini.encoding=UTF-8\npath=%s\n", d.AVDPath)
	return os.WriteFile(filepath.Join(d.AVDHome, d.AVDName+".ini"), []byte(ini), 0600)
}

func (a Adapter) Create(ctx context.Context, r domain.Runtime) (domain.Runtime, error) {
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if err := checkLayout(r); err != nil {
		return r, err
	}
	if _, err := os.Lstat(markerPath(r)); !os.IsNotExist(err) {
		return r, identityError("Android allocation already has ownership evidence")
	}
	if _, err := os.Lstat(r.Android.AVDHome); !os.IsNotExist(err) {
		return r, identityError("Android writable allocation already exists")
	}
	if !portAvailable(r.Android.ConsolePort) || !portAvailable(r.Android.ADBPort) {
		return r, fmt.Errorf("reserved Android port pair is occupied by an external process")
	}
	if err := os.MkdirAll(r.Directory, 0700); err != nil {
		return r, err
	}
	m := marker{LeaseID: r.LeaseID, Runtime: r.Name, Android: *r.Android, Phase: "prepared"}
	if err := saveMarker(r, m); err != nil {
		return r, err
	}
	if err := makePrivateAVD(r); err != nil {
		return r, err
	}
	childEnv := emulatorEnvironment(*r.Android, runtime.GOOS)
	if err := os.MkdirAll(childEnv["TMPDIR"], 0700); err != nil {
		return r, err
	}
	if err := a.ensureADBServer(ctx, r); err != nil {
		if ctx.Err() != nil {
			return r, err
		}
		return r, errors.Join(app.ErrPrerequisite, err)
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	m.Phase = "launching"
	if err := saveMarker(r, m); err != nil {
		return r, err
	}
	d := r.Android
	cmd := execx.Command{Name: executable(d.SDKPath, "emulator", "emulator"), Args: []string{"-avd", d.AVDName, "-port", strconv.Itoa(d.ConsolePort), "-no-window", "-no-audio", "-no-boot-anim", "-no-snapshot", "-no-cache", "-wipe-data", "-netsim-args", "--no-web-ui"}, Dir: r.Directory, Env: childEnv, UnsetEnv: adbRoutingEnvironment()}
	id, err := a.processes().Start(ctx, cmd, filepath.Join(r.Directory, "emulator.stdout.log"), filepath.Join(r.Directory, "emulator.stderr.log"))
	d.ProcessID = id.PID
	d.ProcessStart = id.StartID
	d.State = "booting"
	m.Android = *d
	if err != nil {
		// Even a failed launch can have created a live process. The durable launching
		// phase deliberately requires manual recovery when its identity is incomplete.
		if id.PID == 0 && !errors.Is(err, execx.ErrProcessTreeUnconfirmed) {
			m.Phase = "prepared"
		}
		return r, errors.Join(err, saveMarker(r, m))
	}
	m.Phase = "launched"
	if err := saveMarker(r, m); err != nil {
		return r, errors.Join(identityError("cannot persist launched Android identity"), err)
	}
	// The app owns the readiness budget, operation fencing and compensation.
	// Persistent launch is complete here; Inspect supplies bounded boot observations.
	return r, nil
}

// The emulator client discovers netsimd via TMPDIR, while the daemon uses
// XDG_RUNTIME_DIR on Linux, LOCALAPPDATA/Temp on Windows, and the native temp
// directory on macOS. Keep all of these in one owned namespace so a sibling
// cannot attach to this process tree's radio/network service. The web UI alone
// is disabled at launch; guest networking and radio simulation remain enabled.
func emulatorEnvironment(d domain.AndroidEmulator, goos string) map[string]string {
	localData := filepath.Join(d.AVDHome, "emulator-data")
	temp := filepath.Join(localData, "Temp")
	env := map[string]string{
		"ANDROID_AVD_HOME": d.AVDHome, "ANDROID_HOME": d.SDKPath, "ANDROID_SDK_ROOT": d.SDKPath,
		"TMPDIR": temp, "TMP": temp, "TEMP": temp, "XDG_RUNTIME_DIR": temp,
		// The client uses instance 1. A private discovery directory disambiguates
		// it; the HCI listener must use an ephemeral port instead of shared 6402.
		"NETSIM_INSTANCE": "1", "NETSIM_HCI_PORT": "0",
	}
	if goos == "windows" {
		env["LOCALAPPDATA"] = localData
	}
	return env
}

func (a Adapter) Inspect(ctx context.Context, r domain.Runtime) (app.RuntimeObservation, error) {
	var out app.RuntimeObservation
	m, err := loadMarker(r)
	if os.IsNotExist(err) {
		if _, e := os.Lstat(r.Android.AVDHome); os.IsNotExist(e) && r.Android.ProcessID == 0 {
			return out, nil
		}
		return out, identityError("Android ownership evidence is missing")
	}
	if err != nil {
		return out, err
	}
	d := m.Android
	if m.Phase == "destroyed" {
		if _, err := os.Lstat(d.AVDHome); !os.IsNotExist(err) {
			return out, identityError("destroyed Android writable state unexpectedly reappeared")
		}
		return out, nil
	}
	if m.Phase == "stopped" {
		return out, a.assertStopped(ctx, d)
	}
	if m.Phase == "prepared" {
		return out, nil
	}
	if d.ProcessID == 0 || d.ProcessStart == "" {
		return out, identityError("Android launch has no confirmed process identity")
	}
	alive, err := a.processes().Alive(ctx, execx.ProcessIdentity{PID: d.ProcessID, StartID: d.ProcessStart})
	if err != nil {
		return out, errors.Join(domain.ErrResourceIdentity, err)
	}
	c, err := connectConsole(ctx, d.ConsolePort, d.AVDName)
	if err != nil {
		if errors.Is(err, domain.ErrResourceIdentity) {
			return out, err
		}
		if !alive {
			if !portAvailable(d.ConsolePort) || !portAvailable(d.ADBPort) {
				return out, identityError("Android process disappeared but reserved ports remain occupied")
			}
			return out, nil
		}
		out.Exists = true
		out.Diagnostics = []string{"Android Emulator console is not yet available"}
		return out, nil
	}
	defer c.close()
	if !alive {
		return out, identityError("Android console exists without the recorded process identity")
	}
	out.Exists = true
	protocol, err := a.adbProtocol(ctx, d.SDKPath)
	if err != nil {
		return out, err
	}
	available, err := a.compatibleADB(ctx, protocol)
	if err != nil {
		return out, err
	}
	if !available {
		out.Diagnostics = []string{"shared local ADB server is unavailable"}
		return out, nil
	}
	result, bootErr := a.runner().Run(ctx, execx.Command{Name: executable(d.SDKPath, "platform-tools", "adb"), Args: []string{"-H", "127.0.0.1", "-P", "5037", "-s", d.Serial, "shell", "getprop", "sys.boot_completed"}, UnsetEnv: adbRoutingEnvironment(), Timeout: 5 * time.Second})
	if observed, err := c.command("avd name"); err != nil || strings.TrimSpace(observed) != d.AVDName {
		return out, identityError("Android identity changed during boot inspection")
	}
	state := "booting"
	if bootErr == nil && strings.TrimSpace(result.Stdout) == "1" {
		out.Ready = true
		state = "ready"
	}
	out.Resources = []domain.Resource{{ID: r.Name + ":emulator", LeaseID: r.LeaseID, Runtime: r.Name, Kind: "android-emulator", ExternalID: d.AVDName, Metadata: map[string]string{"lease": r.LeaseID, "state": state, "serial": d.Serial, "avd": d.AVDName, "console_port": strconv.Itoa(d.ConsolePort), "adb_port": strconv.Itoa(d.ADBPort)}}}
	out.Endpoints = map[string]string{"adb": d.Serial}
	return out, nil
}

func (a Adapter) assertStopped(ctx context.Context, d domain.AndroidEmulator) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.ProcessID == 0 || d.ProcessStart == "" {
		return identityError("stopped Android allocation has no confirmed process identity")
	}
	alive, err := a.processes().Alive(ctx, execx.ProcessIdentity{PID: d.ProcessID, StartID: d.ProcessStart})
	if err != nil {
		return errors.Join(domain.ErrResourceIdentity, err)
	}
	if alive || !portAvailable(d.ConsolePort) || !portAvailable(d.ADBPort) {
		return identityError("stopped Android resource unexpectedly has an active process or ports")
	}
	return nil
}

func (a Adapter) Destroy(ctx context.Context, r domain.Runtime) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := loadMarker(r)
	if os.IsNotExist(err) {
		if _, e := os.Lstat(r.Android.AVDHome); os.IsNotExist(e) && r.Android.ProcessID == 0 {
			return nil
		}
		return identityError("Android ownership evidence is missing")
	}
	if err != nil {
		return err
	}
	if m.Phase == "destroyed" {
		if _, err := os.Lstat(r.Android.AVDHome); !os.IsNotExist(err) {
			return identityError("destroyed Android writable state unexpectedly reappeared")
		}
		return nil
	}
	d := m.Android
	if m.Phase == "stopped" {
		if err := a.assertStopped(ctx, d); err != nil {
			return err
		}
	}
	if m.Phase == "prepared" && (!portAvailable(d.ConsolePort) || !portAvailable(d.ADBPort)) {
		return identityError("unlaunched Android allocation has unexpected active ports")
	}
	if m.Phase != "prepared" && m.Phase != "stopped" {
		if d.ProcessID == 0 || d.ProcessStart == "" {
			return identityError("Android launch cannot be safely identified for cleanup")
		}
		alive, err := a.processes().Alive(ctx, execx.ProcessIdentity{PID: d.ProcessID, StartID: d.ProcessStart})
		if err != nil {
			return errors.Join(domain.ErrResourceIdentity, err)
		}
		c, consoleErr := connectConsole(ctx, d.ConsolePort, d.AVDName)
		if consoleErr != nil {
			if alive || !portAvailable(d.ConsolePort) || !portAvailable(d.ADBPort) {
				return errors.Join(domain.ErrResourceIdentity, consoleErr)
			}
		} else {
			if !alive {
				c.close()
				return identityError("refusing to stop a console with mismatched process identity")
			}
			if err := ctx.Err(); err != nil {
				c.close()
				return err
			}
			_, killErr := c.command("kill")
			c.close()
			if killErr != nil {
				return errors.Join(domain.ErrResourceIdentity, killErr, ctx.Err())
			}
			// Emulator 37 permits its own 20-second graceful shutdown window.
			// Observe beyond that window; never substitute an early PID-only kill.
			stopCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()
			for {
				alive, err = a.processes().Alive(stopCtx, execx.ProcessIdentity{PID: d.ProcessID, StartID: d.ProcessStart})
				// An authenticated kill may reap the leader before its descendants.
				// Wait without further effects through that uncertainty; only a later
				// successful, empty-tree observation authorizes writable cleanup.
				if err == nil && !alive && portAvailable(d.ConsolePort) && portAvailable(d.ADBPort) {
					break
				}
				timer := time.NewTimer(100 * time.Millisecond)
				select {
				case <-stopCtx.Done():
					timer.Stop()
					return errors.Join(domain.ErrResourceIdentity, stopCtx.Err(), err)
				case <-timer.C:
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		m.Phase = "stopped"
		if err := saveMarker(r, m); err != nil {
			return err
		}
	}
	// Do not follow links or remove any path outside this exact owned AVD home.
	if err := filepath.WalkDir(d.AVDHome, func(_ string, entry os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return identityError("Android writable state contains unexpected symlink")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(d.AVDHome); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.Phase = "destroyed"
	m.Android.State = "released"
	return saveMarker(r, m)
}
