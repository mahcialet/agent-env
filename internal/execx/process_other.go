//go:build !unix && !windows

package execx

import (
	"context"
	"errors"
	"os/exec"
)

func runProcessTree(context.Context, *exec.Cmd) error {
	return errors.New("process tree containment is unavailable on this operating system")
}
