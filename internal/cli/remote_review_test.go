package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/policy"
)

type ownerReviewResult struct {
	Request    protocol.CreateRequest `json:"request"`
	LocalOwner string                 `json:"local_owner"`
}

func TestRemoteCreateOwnerProcess(t *testing.T) {
	if os.Getenv("AGENT_ENV_OWNER_REVIEW_CHILD") != "1" {
		return
	}
	root := New(io.Discard, io.Discard)
	cmd, _, err := root.Find([]string{"create"})
	if err != nil {
		t.Fatal(err)
	}
	var flags []string
	if err = json.Unmarshal([]byte(os.Getenv("AGENT_ENV_OWNER_REVIEW_FLAGS")), &flags); err != nil {
		t.Fatal(err)
	}
	if err = cmd.ParseFlags(flags); err != nil {
		t.Fatal(err)
	}
	id, _ := cmd.Flags().GetString("operation-id")
	owner, _ := cmd.Flags().GetString("owner")
	if err = json.NewEncoder(os.Stdout).Encode(ownerReviewResult{Request: protocol.CreateRequest{OperationID: id, Options: remoteCreateOptions(cmd)}, LocalOwner: owner}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestRemoteCreateImplicitOwnerIsStableAcrossProcesses(t *testing.T) {
	invoke := func(envOwner string, extra ...string) ownerReviewResult {
		t.Helper()
		flags, _ := json.Marshal(append([]string{"--operation-id=retry-create", "--stack=app", "--ttl=1h", "--purpose=review"}, extra...))
		cmd := exec.Command(os.Args[0], "-test.run=^TestRemoteCreateOwnerProcess$")
		for _, entry := range os.Environ() {
			name, _, _ := strings.Cut(entry, "=")
			if strings.EqualFold(name, "AGENT_ENV_OWNER") || strings.HasPrefix(name, "AGENT_ENV_OWNER_REVIEW_") {
				continue
			}
			cmd.Env = append(cmd.Env, entry)
		}
		cmd.Env = append(cmd.Env, "AGENT_ENV_OWNER="+envOwner, "AGENT_ENV_OWNER_REVIEW_CHILD=1", "AGENT_ENV_OWNER_REVIEW_FLAGS="+string(flags))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child: %v: %s", err, output)
		}
		var result ownerReviewResult
		if err = json.Unmarshal(output, &result); err != nil {
			t.Fatalf("decode %q: %v", output, err)
		}
		return result
	}
	a, b := invoke(""), invoke("")
	if a.LocalOwner == b.LocalOwner {
		t.Fatal("local per-invocation owner defaults unexpectedly changed")
	}
	if a.Request.OperationID != "retry-create" || b.Request.OperationID != a.Request.OperationID {
		t.Fatal("actual operation-id flag not preserved")
	}
	araw, _ := json.Marshal(a.Request)
	braw, _ := json.Marshal(b.Request)
	if !bytes.Equal(araw, braw) {
		t.Fatalf("same remote create retry has conflicting payloads:\n%s\n%s", araw, braw)
	}
	for _, test := range []struct {
		env   string
		flags []string
		want  string
	}{
		{"", []string{"--owner=explicit/owner"}, "explicit/owner"},
		{"environment/owner", nil, "environment/owner"},
		{"environment/owner", []string{"--owner=flag/owner"}, "flag/owner"},
	} {
		result := invoke(test.env, test.flags...)
		var options struct {
			Owner string `json:"owner"`
		}
		if err := json.Unmarshal(result.Request.Options, &options); err != nil {
			t.Fatal(err)
		}
		if options.Owner != test.want {
			t.Fatalf("owner=%q want %q", options.Owner, test.want)
		}
	}
}

func TestWorkerCapacityBoundedBeforeResources(t *testing.T) {
	limit := policy.Defaults().MaxActive
	for _, n := range []int{-1, 0, 1, limit, limit + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "uncreated-home")
			t.Setenv("AGENT_ENV_HOME", home)
			cmd := New(io.Discard, io.Discard)
			cmd.SetArgs([]string{"worker", "serve", "--host-id=fixture", "--max-leases=" + fmt.Sprint(n)})
			err := cmd.Execute()
			if err == nil {
				t.Fatal("worker unexpectedly started")
			}
			if n < 1 || n > limit {
				if !strings.Contains(err.Error(), "max-leases") {
					t.Fatalf("capacity check did not precede TLS: %v", err)
				}
			} else if !strings.Contains(err.Error(), "remote mode requires") {
				t.Fatalf("supported capacity rejected before TLS: %v", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("worker created resources during preflight: %v", err)
			}
		})
	}
}

func TestWorkerAndroidCapacityBoundedBeforeResources(t *testing.T) {
	for _, n := range []int{-1, 0, 1, 65, 66, 1000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "uncreated-home")
			t.Setenv("AGENT_ENV_HOME", home)
			cmd := New(io.Discard, io.Discard)
			cmd.SetArgs([]string{"worker", "serve", "--host-id=fixture", "--android-slots=" + fmt.Sprint(n)})
			err := cmd.Execute()
			if err == nil {
				t.Fatal("worker unexpectedly started")
			}
			if n < 0 || n > 65 {
				if !strings.Contains(err.Error(), "android-slots") {
					t.Fatalf("capacity check did not precede TLS: %v", err)
				}
			} else if !strings.Contains(err.Error(), "remote mode requires") {
				t.Fatalf("supported capacity rejected before TLS: %v", err)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("worker created resources during preflight: %v", err)
			}
		})
	}
}
