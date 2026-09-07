//go:build unix

package execx

import (
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestTerminateProcessGroupWaitsForZombieReaping(t *testing.T) {
	for _, result := range []struct {
		name string
		err  error
	}{{"group gone", syscall.ESRCH}, {"signal delivered", nil}} {
		t.Run(result.name, func(t *testing.T) {
			calls := 0
			pid := syscall.Getpgrp() + 1
			err := terminateProcessGroup(pid, time.Second, func(target int, signal syscall.Signal) error {
				if target != -pid || signal != syscall.SIGKILL {
					t.Fatalf("unexpected signal %v to %d", signal, target)
				}
				calls++
				if calls == 1 {
					return syscall.EPERM
				}
				return result.err
			})
			if err != nil || calls != 2 {
				t.Fatalf("termination: %v, calls: %d", err, calls)
			}
		})
	}
}

func TestTerminateProcessGroupPreservesFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		retryFor time.Duration
	}{{"permission denied", syscall.EPERM, 20 * time.Millisecond},
		{"other failure", syscall.EINVAL, time.Second},
		{"no retry off Darwin", syscall.EPERM, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := terminateProcessGroup(syscall.Getpgrp()+1, tc.retryFor, func(int, syscall.Signal) error {
				calls++
				return tc.err
			})
			if !errors.Is(err, tc.err) {
				t.Fatalf("termination: %v, want %v", err, tc.err)
			}
			if (tc.err != syscall.EPERM || tc.retryFor == 0) && calls != 1 {
				t.Fatalf("unexpected retry: %d calls", calls)
			}
		})
	}
}

func TestTerminateProcessGroupRefusesUnsafeTargets(t *testing.T) {
	for _, pid := range []int{-1, 0, syscall.Getpgrp()} {
		err := terminateProcessGroup(pid, time.Second, func(int, syscall.Signal) error {
			t.Fatal("attempted to signal unsafe target")
			return nil
		})
		if err == nil {
			t.Fatalf("accepted unsafe process group %d", pid)
		}
	}
}
