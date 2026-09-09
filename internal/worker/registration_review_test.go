package worker

import (
	"context"
	"errors"
	"github.com/mahcialet/agent-env/internal/controlplane/client"
	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"testing"
	"time"
)

type registrationReviewTransport struct {
	fakeTransport
	first error
	calls int
	stop  error
}

func (t *registrationReviewTransport) Info(context.Context) (protocol.Info, error) {
	return protocol.Info{ControllerID: "controller", ProtocolVersion: protocol.Version, ProductVersion: "fixture"}, nil
}
func (t *registrationReviewTransport) Register(context.Context, protocol.RegisterRequest) (protocol.RegisterResult, error) {
	t.calls++
	if t.calls == 1 && t.first != nil {
		return protocol.RegisterResult{}, t.first
	}
	return protocol.RegisterResult{}, t.stop
}
func TestWorkerRegistrationRejectsPermanentStatusWithoutRetry(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first error
		calls int
	}{
		{"forbidden", nil, 1}, {"conflict", nil, 1}, {"server-retry", &client.StatusError{StatusCode: 503}, 2}, {"rate-retry", &client.StatusError{StatusCode: 429}, 2}, {"transport-retry", errors.New("connection reset"), 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j, err := OpenJournal(t.TempDir(), "host")
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			code := 403
			if tc.name == "conflict" {
				code = 409
			}
			stop := &client.StatusError{StatusCode: code, Code: "rejected", Message: "registration refused"}
			transport := &registrationReviewTransport{first: tc.first, stop: stop}
			r := Runner{Journal: j, Transport: transport, Executor: &fakeExecutor{}, Registration: protocol.RegisterRequest{ProductVersion: "fixture"}, RetryDelay: time.Millisecond}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err = r.Run(ctx)
			if !errors.Is(err, stop) || transport.calls != tc.calls {
				t.Fatalf("registration error=%v calls=%d want=%d", err, transport.calls, tc.calls)
			}
		})
	}
}
