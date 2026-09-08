package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

// This native fixture deliberately uses independent built CLI invocations. It
// proves lifetime beyond create; invoking Cobra in the test process cannot.
func TestPersistentProcessNativeCLI(t *testing.T) {
	tempRoot, err := os.MkdirTemp("", "agent-env-native-process-")
	if err != nil {
		t.Fatal(err)
	}
	retain := false
	t.Cleanup(func() {
		if retain {
			t.Logf("retained uncertain native fixture evidence at %s", tempRoot)
		} else if err := os.RemoveAll(tempRoot); err != nil {
			t.Error(err)
		}
	})
	base := filepath.Join(tempRoot, "native process 日本語 space")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "state home")
	// Exercise canonical reservation paths through a symlink on native Unix,
	// including macOS's aliased temporary directory parents.
	if runtime.GOOS != "windows" {
		realHome := filepath.Join(base, "real state home")
		if err := os.MkdirAll(realHome, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realHome, home); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AGENT_ENV_HOME", home)
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	cliPath := filepath.Join(base, "agent-env"+suffix)
	helper := filepath.Join(base, "fixture browser"+suffix)
	run := func(dir, name string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		command := exec.CommandContext(ctx, name, args...)
		command.Dir = dir
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("native fixture %s %v: %v\n%s", name, args, err, output)
		}
		return string(output)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	run(root, "go", "build", "-o", cliPath, "./cmd/agent-env")
	helperSource := filepath.Join(base, "browser.go")
	source := `package main
import("encoding/json";"flag";"fmt";"net/http";"net";"os";"path/filepath";"time")
func main(){
 port:=flag.Int("port",0,"port");state:=flag.String("state","","state");hold:=flag.Bool("hold-ready",false,"hold readiness");flag.Parse()
 profile:=filepath.Join(*state,"private-profile");if err:=os.MkdirAll(profile,0700);err!=nil{panic(err)}
 if err:=os.WriteFile(filepath.Join(profile,"owner"),[]byte(fmt.Sprint(os.Getpid())),0600);err!=nil{panic(err)}
 fmt.Printf("fixture browser ready pid=%d profile=%s\n",os.Getpid(),profile)
 http.HandleFunc("/json/version",func(w http.ResponseWriter,r *http.Request){if *hold {if _,err:=os.Stat(filepath.Join(*state,"unblock"));err!=nil {http.Error(w,"fixture readiness blocked",503);return}};w.Header().Set("Content-Type","application/json");_ = json.NewEncoder(w).Encode(map[string]any{"Browser":"native-fixture/1","profile":profile,"pid":os.Getpid()})})
 http.HandleFunc("/fixture-exit",func(w http.ResponseWriter,r *http.Request){fmt.Fprintln(w,"exiting");go func(){time.Sleep(20*time.Millisecond);os.Exit(0)}()})
 go func(){time.Sleep(3*time.Minute);os.Exit(9)}()
 listener,err:=net.Listen("tcp4",fmt.Sprintf("127.0.0.1:%d",*port));if err!=nil{panic(err)}
 if err:=os.WriteFile(filepath.Join(*state,"listen-port"),[]byte(fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)),0600);err!=nil{panic(err)}
 if err:=http.Serve(listener,nil);err!=nil{panic(err)}
}`
	if err := os.WriteFile(helperSource, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	run(root, "go", "build", "-o", helper, helperSource)
	repo := filepath.Join(base, "source repository")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	sourceHelper := filepath.Join(repo, filepath.Base(helper))
	if err := os.Rename(helper, sourceHelper); err != nil {
		t.Fatal(err)
	}
	quotedHelper, _ := json.Marshal("./" + filepath.Base(helper))
	manifest := fmt.Sprintf(`version: 1
sources:
  self: {repository: ., default_ref: main}
runtimes:
  browser:
    type: process
    source: self
    working_directory: .
    command: [%s, "--port", "${port:cdp}", "--state", "${runtime_dir}"]
    ports: {cdp: {protocol: tcp}}
components:
  browser:
    runtime: browser
    endpoints: {cdp: {runtime_port: cdp}}
    readiness:
      - {type: http, url: "http://127.0.0.1:${endpoint:cdp}/json/version", timeout: 15s}
stacks:
  review: {roots: [browser]}
`, quotedHelper)
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_COUNT", "0")
	run(repo, "git", "-c", "init.templateDir=", "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.txt"), []byte("tracked fixture source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(repo, "git", "add", ".agent-env.yaml", "README.txt", filepath.Base(helper))
	run(repo, "git", "-c", "commit.gpgsign=false", "-c", "user.name=Native Fixture", "-c", "user.email=native@example.invalid", "commit", "-m", "Native process fixture")
	invoke := func(args ...string) (json.RawMessage, error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, cliPath, append([]string{"--output=json", "--owner=native-fixture"}, args...)...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		var response struct {
			Data json.RawMessage `json:"data"`
		}
		if stdout.Len() > 0 {
			if decodeErr := json.Unmarshal(stdout.Bytes(), &response); decodeErr != nil {
				return nil, fmt.Errorf("JSON %s: %w", stdout.String(), decodeErr)
			}
		}
		if err != nil {
			return response.Data, fmt.Errorf("CLI %v: %w: %s", args, err, stderr.String())
		}
		return response.Data, nil
	}
	must := func(args ...string) json.RawMessage {
		t.Helper()
		data, err := invoke(args...)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	decodeLease := func(data json.RawMessage) domain.Lease {
		t.Helper()
		var l domain.Lease
		if err := json.Unmarshal(data, &l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	leases := []domain.Lease{}
	t.Cleanup(func() {
		// Recover IDs from the isolated registry if create exited before its
		// JSON response was available. Failed allocations may still own effects.
		if data, err := invoke("list", "--cached"); err == nil {
			var registered []domain.Lease
			if json.Unmarshal(data, &registered) == nil {
				leases = registered
			}
		}
		for _, l := range leases {
			if _, err := invoke("destroy", l.ID, "--force"); err != nil {
				retain = true
				t.Errorf("native cleanup %s: %v", l.ID, err)
			}
		}
	})
	for range 2 {
		l := decodeLease(must("create", repo, "--stack", "review"))
		leases = append(leases, l)
		if l.Observed != "ready" || len(l.Runtimes) != 1 || l.Runtimes[0].Process == nil || l.Runtimes[0].Process.ProcessID <= 0 {
			t.Fatalf("native create=%+v", l)
		}
	}
	first, second := leases[0].Runtimes[0].Process, leases[1].Runtimes[0].Process
	if first.Ports["cdp"] == second.Ports["cdp"] || first.StateDirectory == second.StateDirectory || first.ProcessID == second.ProcessID {
		t.Fatal("native leases share identity, port, or private state")
	}
	client := http.Client{Timeout: 3 * time.Second}
	version := func(p *domain.PersistentProcess) map[string]any {
		t.Helper()
		response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", p.Ports["cdp"]))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var data map[string]any
		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			t.Fatal(err)
		}
		if data["Browser"] != "native-fixture/1" || data["profile"] != filepath.Join(p.StateDirectory, "private-profile") || data["pid"] != float64(p.ProcessID) {
			t.Fatalf("profile response=%v", data)
		}
		return data
	}
	// An unrelated ordinary host child is outside every lease's native group
	// or Job. Its kernel-assigned port avoids a close/rebind allocation race.
	sentinelState := filepath.Join(base, "unrelated host state")
	sentinel := exec.Command(sourceHelper, "--port", "0", "--state", sentinelState)
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() })
	sentinelPort := 0
	sentinelDeadline := time.Now().Add(10 * time.Second)
	for sentinelPort == 0 {
		data, err := os.ReadFile(filepath.Join(sentinelState, "listen-port"))
		if err == nil {
			sentinelPort, _ = strconv.Atoi(string(data))
		}
		if time.Now().After(sentinelDeadline) {
			t.Fatal("unrelated host process did not start")
		}
		if sentinelPort == 0 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	unrelated := &domain.PersistentProcess{StateDirectory: sentinelState, Ports: map[string]int{"cdp": sentinelPort}, ProcessID: sentinel.Process.Pid}
	version(unrelated)
	version(first)
	version(second)
	logs := must("logs", leases[0].ID)
	if !bytes.Contains(logs, []byte("fixture browser ready")) {
		t.Fatalf("native log absent: %s", logs)
	}
	show := func(id string) domain.Lease {
		t.Helper()
		var data struct {
			Lease domain.Lease `json:"lease"`
		}
		if err := json.Unmarshal(must("show", id), &data); err != nil {
			t.Fatal(err)
		}
		return data.Lease
	}
	if l := show(leases[0].ID); l.Observed != "ready" {
		t.Fatalf("separate CLI show=%s", l.Observed)
	}
	// Refusing dirty source cleanup must happen before any process termination.
	edited := filepath.Join(leases[0].Sources[0].WorktreePath, "README.txt")
	dirtyText := "tracked fixture source\nuser edit while process alive\n"
	if err := os.WriteFile(edited, []byte(dirtyText), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke("destroy", leases[0].ID); err == nil {
		t.Fatal("dirty live process source was silently destroyed")
	}
	if l := show(leases[0].ID); l.Observed != "quarantined" {
		t.Fatalf("dirty source state=%s", l.Observed)
	}
	if data, err := os.ReadFile(edited); err != nil || string(data) != dirtyText {
		t.Fatalf("refused cleanup changed tracked source: %v", err)
	}
	version(first)
	version(second)
	version(unrelated)
	released := decodeLease(must("destroy", leases[0].ID, "--force"))
	if released.Observed != "released" {
		t.Fatalf("forced dirty cleanup=%s", released.Observed)
	}
	if _, err := os.Stat(first.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("forced cleanup retained private state: %v", err)
	}
	artifactsDB, err := sql.Open("sqlite", filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	var patchPath string
	err = artifactsDB.QueryRow("SELECT path FROM artifacts WHERE lease_id=? AND kind='tracked-diff' ORDER BY created_at DESC LIMIT 1", leases[0].ID).Scan(&patchPath)
	_ = artifactsDB.Close()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := os.ReadFile(patchPath)
	if err != nil || !bytes.Contains(patch, []byte("+user edit while process alive")) {
		t.Fatalf("forced cleanup lacked captured tracked diff: %v", err)
	}
	version(second)
	version(unrelated)
	logs = must("logs", leases[0].ID)
	if !strings.Contains(string(logs), "fixture browser ready") {
		t.Fatal("released logs unavailable")
	}

	// The second lease retains the independent manual-death/no-restart case.
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/fixture-exit", second.Ports["cdp"]))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	deadline := time.Now().Add(10 * time.Second)
	for {
		l := show(leases[1].ID)
		if l.Observed == "degraded" {
			if l.Runtimes[0].Process.ProcessID != second.ProcessID {
				t.Fatal("dead process restarted")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("dead process not degraded: %s", l.Observed)
		}
		time.Sleep(30 * time.Millisecond)
	}
	released = decodeLease(must("destroy", leases[1].ID))
	if released.Observed != "released" {
		t.Fatalf("dead process destroy=%s", released.Observed)
	}
	if _, err := os.Stat(second.StateDirectory); !os.IsNotExist(err) {
		t.Fatalf("dead process state remains: %v", err)
	}
	version(unrelated)
	if l := show(leases[1].ID); l.Observed != "released" {
		t.Fatalf("released show=%s", l.Observed)
	}
	// Exercise a CLI crash while readiness is pending. Only the creating CLI is
	// killed; the managed native process must survive and remain recoverable.
	manifest = strings.Replace(manifest, `"--state", "${runtime_dir}"]`, `"--state", "${runtime_dir}", "--hold-ready"]`, 1)
	if err := os.WriteFile(filepath.Join(repo, ".agent-env.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	run(repo, "git", "add", ".agent-env.yaml")
	run(repo, "git", "-c", "commit.gpgsign=false", "-c", "user.name=Native Fixture", "-c", "user.email=native@example.invalid", "commit", "-m", "Hold fixture readiness for crash recovery")
	creator := exec.Command(cliPath, "--output=json", "--owner=native-fixture", "create", repo, "--stack", "review")
	var createOut, createErr bytes.Buffer
	creator.Stdout = &createOut
	creator.Stderr = &createErr
	if err := creator.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	exited := make(chan error, 1)
	go func() { exited <- creator.Wait() }()
	t.Cleanup(func() {
		if !waited {
			_ = creator.Process.Kill()
			<-exited
		}
	})
	var interrupted domain.Lease
	deadline = time.Now().Add(12 * time.Second)
	for interrupted.ID == "" {
		var rows []domain.Lease
		if err := json.Unmarshal(must("list", "--cached"), &rows); err != nil {
			t.Fatal(err)
		}
		for _, candidate := range rows {
			if candidate.ID == leases[0].ID || candidate.ID == leases[1].ID || len(candidate.Runtimes) == 0 {
				continue
			}
			if p := candidate.Runtimes[0].Process; p != nil && p.ProcessID > 0 {
				interrupted = candidate
				break
			}
		}
		if interrupted.ID != "" {
			break
		}
		select {
		case err := <-exited:
			waited = true
			t.Fatalf("creating CLI exited before interruption: %v %s %s", err, createOut.String(), createErr.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("creating CLI did not persist process identity before readiness")
		}
		time.Sleep(25 * time.Millisecond)
	}
	leases = append(leases, interrupted)
	if err := creator.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-exited
	waited = true
	// Only this isolated test DB lock is expired to avoid waiting for the real
	// crash-recovery TTL. Production acquisition and fencing remain unchanged.
	database, err := sql.Open("sqlite", filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := database.Exec("UPDATE operation_locks SET expires_at=? WHERE lease_id=?", time.Now().Add(-time.Second).UnixNano(), interrupted.ID)
	if err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		_ = database.Close()
		t.Fatalf("fixture lock expiration rows=%d err=%v", changed, err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	process := interrupted.Runtimes[0].Process
	if err := os.WriteFile(filepath.Join(process.StateDirectory, "unblock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	version(process)
	recovered := show(interrupted.ID)
	if recovered.Observed != "degraded" || recovered.Runtimes[0].Process.ProcessID != process.ProcessID {
		t.Fatalf("crash recovery changed native process: %+v", recovered)
	}
	recovered = decodeLease(must("destroy", interrupted.ID))
	if recovered.Observed != "released" {
		t.Fatalf("crash recovery cleanup=%s", recovered.Observed)
	}
	version(unrelated)
}
