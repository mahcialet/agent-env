package store

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
)

func TestTransientInputNeverEntersControllerJournal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	w := register(t, s, "a", 2, 2)
	op, err := s.Create(ctx, request("create"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Poll(ctx, w); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Complete(ctx, protocol.Result{WorkerIdentity: w, OperationID: op.ID, LeaseID: op.LeaseID, Epoch: op.Epoch, State: "completed", LocalState: "ready"}); err != nil {
		t.Fatal(err)
	}
	const canary = "controller-review-transient-canary"
	for _, kind := range []string{"ui", "browser"} {
		r := protocol.SubmitRequest{OperationID: kind + "-input", LeaseID: op.LeaseID, Kind: kind, Payload: json.RawMessage(`{"ui":{"Text":"` + canary + `"},"ui":{}}`)}
		_, err = s.Submit(ctx, r)
		if !isFault(err, "invalid") || strings.Contains(err.Error(), canary) {
			t.Fatalf("transient input not refused safely: %v", err)
		}
		if _, err = s.GetOperation(ctx, r.OperationID); !isFault(err, "not_found") {
			t.Fatalf("transient operation persisted: %v", err)
		}
	}
	files, err := filepath.Glob(s.Path + "*")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(canary)) {
			t.Fatalf("transient input reached database/WAL: %s", filepath.Base(path))
		}
	}
}
