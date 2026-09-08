package compose

import (
	"context"
	"testing"

	"github.com/mahcialet/agent-env/internal/domain"
)

func TestUnknownProviderNeverReachesRunner(t *testing.T) {
	c := Client{}
	r := domain.Runtime{Provider: "unexpected"}
	ctx := context.Background()
	if _, err := c.DoctorFor(ctx, r.Provider); err == nil {
		t.Fatal("doctor accepted unknown provider")
	}
	if _, err := c.Render(ctx, r, nil); err == nil {
		t.Fatal("render accepted unknown provider")
	}
	if err := c.Up(ctx, r); err == nil {
		t.Fatal("up accepted unknown provider")
	}
	if err := c.Down(ctx, r); err == nil {
		t.Fatal("down accepted unknown provider")
	}
	if _, err := c.Logs(ctx, r); err == nil {
		t.Fatal("logs accepted unknown provider")
	}
	if _, err := c.Inspect(ctx, r); err == nil {
		t.Fatal("inspect accepted unknown provider")
	}
	if _, err := c.InventoryFor(ctx, r.Provider, ""); err == nil {
		t.Fatal("inventory accepted unknown provider")
	}
}
