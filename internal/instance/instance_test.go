package instance

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestIndependentConnectionsExcludeAndRelease(t *testing.T) {
	p := filepath.Join(t.TempDir(), "owner.lock")
	release, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	if r, e := Acquire(p); e == nil {
		r()
		t.Fatal("second owner admitted")
	}
	if e = release(); e != nil {
		t.Fatal(e)
	}
	r, e := Acquire(p)
	if e != nil {
		t.Fatal(e)
	}
	defer r()
}

func TestNativeProcessCrashReleasesLock(t *testing.T) {
	if path := os.Getenv("AGENT_ENV_INSTANCE_TEST_LOCK"); path != "" {
		release, err := Acquire(path)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		fmt.Println("locked")
		time.Sleep(30 * time.Second)
		return
	}
	path := filepath.Join(t.TempDir(), "共有 state", "owner.lock")
	cmd := exec.Command(os.Args[0], "-test.run=^TestNativeProcessCrashReleasesLock$")
	cmd.Env = append(os.Environ(), "AGENT_ENV_INSTANCE_TEST_LOCK="+path)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	ready := make(chan bool, 1)
	go func() { scanner := bufio.NewScanner(pipe); ready <- scanner.Scan() && scanner.Text() == "locked" }()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("child did not acquire lock")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child lock timeout")
	}
	if release, err := Acquire(path); err == nil {
		release()
		t.Fatal("independent process admitted")
	}
	if err = cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	release, err := Acquire(path)
	if err != nil {
		t.Fatal("OS did not release crashed owner:", err)
	}
	defer release()
}
