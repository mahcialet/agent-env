//go:build linux || darwin

package execx

import (
	"context"
	"errors"
	"strings"
	"syscall"
	"time"
)

func observeManaged(id ProcessIdentity) (ProcessObservation, error) {
	want, ok := strings.CutPrefix(id.StartID, "group|")
	if !ok || want == "" {
		return ProcessObservation{}, errors.New("managed process group identity missing")
	}
	actual, root, err := detachedIdentity(id.PID)
	if err != nil {
		return ProcessObservation{}, err
	}
	if root {
		if actual != want {
			return ProcessObservation{}, errors.Join(ErrProcessTreeUnconfirmed, errors.New("managed root identity mismatch"))
		}
		group, err := syscall.Getpgid(id.PID)
		if err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return ProcessObservation{}, errors.Join(errProcessCensusUnstable, err)
			}
			return ProcessObservation{}, err
		}
		if group != id.PID {
			return ProcessObservation{}, errors.Join(ErrProcessTreeUnconfirmed, errors.New("managed root left its process group"))
		}
		// Do not combine a birth observation from the old PID with a group
		// observation from a replacement process during the census window.
		confirmed, stillRoot, err := detachedIdentity(id.PID)
		if err != nil {
			return ProcessObservation{}, err
		}
		if !stillRoot {
			return ProcessObservation{}, errProcessCensusUnstable
		}
		if confirmed != want {
			return ProcessObservation{}, ErrProcessTreeUnconfirmed
		}
		return ProcessObservation{Alive: true, RootAlive: true}, nil
	}
	alive, err := detachedGroupAlive(id.PID)
	observed := ProcessObservation{Alive: alive}
	if err != nil {
		return observed, err
	}
	if alive {
		return observed, errors.Join(ErrProcessTreeUnconfirmed, errors.New("managed leader absent; live group ownership is uncertain"))
	}
	return observed, nil
}

func signalManaged(ctx context.Context, id ProcessIdentity, signal syscall.Signal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	observed, err := (NativeDetached{}).Observe(ctx, id)
	if err != nil {
		return err
	}
	if !observed.Alive {
		return nil
	}
	if !observed.RootAlive {
		return ErrProcessTreeUnconfirmed
	}
	// Both native birth and PGID were checked immediately before the group signal.
	// Unix offers no atomic birth-conditioned group signal; never retry a signal
	// after the leader has disappeared or a later observation becomes uncertain.
	if err := syscall.Kill(-id.PID, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

func terminateManaged(ctx context.Context, id ProcessIdentity, grace time.Duration) error {
	if err := signalManaged(ctx, id, syscall.SIGTERM); err != nil {
		return err
	}
	if err := waitManagedGone(ctx, id, grace); err == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := signalManaged(ctx, id, syscall.SIGKILL); err != nil {
		return err
	}
	return waitManagedGone(ctx, id, 2*time.Second)
}
