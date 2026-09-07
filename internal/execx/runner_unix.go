//go:build !windows

package execx

import (
	"context"
	"os/exec"
)

func platformCommand(ctx context.Context, spec Command) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, spec.Name, spec.Args...), nil
}
