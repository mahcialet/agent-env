package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mahcialet/agent-env/internal/controlplane/protocol"
	"github.com/mahcialet/agent-env/internal/remotesource"
)

func TestCompactCreatePackageHydratesAndRejectsConflicts(t *testing.T) {
	for _, mode := range []string{"compact", "legacy", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			e, op, process, _ := executorFixture(t)
			var envelope protocol.CreateRequest
			if err := json.Unmarshal(op.Payload, &envelope); err != nil {
				t.Fatal(err)
			}
			var pkg remotesource.Package
			if err := json.Unmarshal(envelope.Package, &pkg); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "compact":
				pkg.Manifest = nil
			case "conflict":
				envelope.Manifest = json.RawMessage(`{}`)
			}
			envelope.Package, _ = json.Marshal(pkg)
			op.Payload, _ = json.Marshal(envelope)
			err := e.Prepare(context.Background(), op)
			if mode == "conflict" {
				if err == nil {
					t.Fatal("mismatched duplicate manifest accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			result := e.Execute(context.Background(), op)
			if result.State != "completed" || process.starts != 1 {
				t.Fatalf("prepared package did not execute: %+v starts=%d", result, process.starts)
			}
		})
	}
}
