package execx

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestDetachedObservationRejectsDifferentWindowsSession(t *testing.T) {
	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		t.Fatal(err)
	}
	// Neither this root nor job exists locally. Returning false,nil would confuse
	// an invisible job in another session with a confirmed empty process tree.
	id := ProcessIdentity{PID: 2147483647, StartID: fmt.Sprintf("job|%d|Local\\agent-env-nonexistent|1:1", session^1)}
	if alive, err := (NativeDetached{}).Alive(context.Background(), id); alive || err == nil || !strings.Contains(err.Error(), "different Windows session") {
		t.Fatalf("cross-session observation authorized absence: alive=%v err=%v", alive, err)
	}
}

func TestDetachedObservationRejectsMissingWindowsSessionIdentity(t *testing.T) {
	for _, start := range []string{"job|Local\\agent-env-nonexistent|1:1", "job|invalid|Local\\agent-env-nonexistent|1:1"} {
		if alive, err := (NativeDetached{}).Alive(context.Background(), ProcessIdentity{PID: 2147483647, StartID: start}); alive || err == nil {
			t.Fatalf("incomplete session identity authorized absence: %q alive=%v err=%v", start, alive, err)
		}
	}
}
