package compose

import (
	"context"
	"strings"
	"testing"
)

func TestInventoryIncludesStoppedAndUnownedProjectsWithoutMutation(t *testing.T) {
	r := runtimeFixture(t)
	r.Context = "captured-context"
	container := strings.Replace(containerJSON(r, "exited"), `"com.docker.compose.project"`, `"io.agent-env.lease":"orphan-lease","com.docker.compose.project"`, 1)
	b := []string{"--context", r.Context}
	f := &fakeRunner{t: t, steps: []step{
		{args: with(b, "compose", "ls", "--all", "--format", "json"), output: `[{"Name":"ae_test_123","Status":"exited(1)","ConfigFiles":"/recorded/compose.json"},{"Name":"unrelated","Status":"running(1)"}]`},
		{args: with(b, "ps", "--all", "--quiet", "--filter", "label="+leaseLabel), output: "c123\n"},
		{args: with(b, "ps", "--all", "--quiet", "--filter", "label="+projectLabel), output: "c123\n"},
		{args: with(b, "inspect", "--type", "container", "c123"), output: container},
		{args: with(b, "network", "ls", "--quiet", "--filter", "label="+leaseLabel)},
		{args: with(b, "network", "ls", "--quiet", "--filter", "label="+projectLabel), output: "net123\n"},
		{args: with(b, "network", "inspect", "net123"), output: `[{"Id":"net123","Name":"ae_test_123_default","Labels":{"com.docker.compose.project":"ae_test_123"}}]`},
		{args: with(b, "volume", "ls", "--quiet", "--filter", "label="+leaseLabel)},
		{args: with(b, "volume", "ls", "--quiet", "--filter", "label="+projectLabel), output: "ae_test_123_data\n"},
		{args: with(b, "volume", "inspect", "ae_test_123_data"), output: `[{"Name":"ae_test_123_data","Labels":{"com.docker.compose.project":"ae_test_123"}}]`},
	}}
	items, err := testClient(f).Inventory(context.Background(), r.Context)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("got %d resources", len(items))
	}
	var found bool
	for _, item := range items {
		if item.Metadata["context"] != r.Context {
			t.Fatal("context not recorded")
		}
		if item.Kind == "container" {
			found = true
			if item.LeaseID != "orphan-lease" || item.Metadata["observed_status"] != "exited" {
				t.Fatalf("ownership/status lost: %+v", item)
			}
		}
		if item.ExternalID == "unrelated" && item.Metadata["ownership"] != "unknown" {
			t.Fatal("foreign project automatically adopted")
		}
	}
	if !found {
		t.Fatal("stopped container absent")
	}
	f.done()
}

func TestInventoryRequiresContextAndRejectsInvalidOutput(t *testing.T) {
	f := &fakeRunner{t: t}
	if _, err := testClient(f).Inventory(context.Background(), ""); err == nil {
		t.Fatal("missing context accepted")
	}
	f.done()
	f = &fakeRunner{t: t, steps: []step{{args: []string{"--context", "default", "compose", "ls", "--all", "--format", "json"}, output: "not json"}}}
	if _, err := testClient(f).Inventory(context.Background(), "default"); err == nil {
		t.Fatal("malformed inventory accepted")
	}
	f.done()
}
