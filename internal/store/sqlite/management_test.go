package sqlite

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestLeaseManagementImmutableAcrossSaveAndReopen(t *testing.T) {
	ctx := context.Background()
	for _, change := range []string{"remove", "controller", "host", "instance", "epoch", "adopt", "unchanged"} {
		t.Run(change, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "registry.sqlite")
			db, e := Open(path)
			if e != nil {
				t.Fatal(e)
			}
			original := lease("managed")
			if change != "adopt" {
				original.Management = &domain.Management{ControllerID: "controller", HostID: "host", HostInstanceID: "instance", AssignmentEpoch: 3}
			}
			if e = db.Reserve(ctx, original, 0); e != nil {
				t.Fatal(e)
			}
			candidate, e := db.Get(ctx, original.ID)
			if e != nil {
				t.Fatal(e)
			}
			switch change {
			case "remove":
				candidate.Management = nil
			case "controller":
				candidate.Management.ControllerID = "other"
			case "host":
				candidate.Management.HostID = "other"
			case "instance":
				candidate.Management.HostInstanceID = "other"
			case "epoch":
				candidate.Management.AssignmentEpoch++
			case "adopt":
				candidate.Management = &domain.Management{ControllerID: "controller", HostID: "host", HostInstanceID: "instance", AssignmentEpoch: 3}
			}
			candidate.Purpose = "updated"
			op, release, e := db.AcquireContext(ctx, original.ID, "operation", time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			e = db.Save(op, candidate)
			if change == "unchanged" && e != nil || change != "unchanged" && e == nil {
				t.Fatalf("assignment update %s: %v", change, e)
			}
			if e = release(); e != nil {
				t.Fatal(e)
			}
			if e = db.Close(); e != nil {
				t.Fatal(e)
			}
			db, e = Open(path)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			got, e := db.Get(ctx, original.ID)
			if e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(got.Management, original.Management) {
				t.Fatalf("assignment changed durably: %+v", got.Management)
			}
			if change != "unchanged" && got.Purpose != original.Purpose {
				t.Fatal("rejected assignment partially persisted")
			}
			if change == "unchanged" && got.Purpose != "updated" {
				t.Fatal("ordinary save blocked")
			}
		})
	}
}
