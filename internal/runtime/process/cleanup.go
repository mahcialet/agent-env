package process

import (
	"context"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
)

// Called only after Destroy has proved the native tree absent. A Windows sharing
// violation can outlive that proof; retry just that filesystem condition without
// changing process ownership or treating an incomplete removal as success.
func removeStateDirectory(ctx context.Context, r domain.Runtime, remove func(string) error, retryable func(error) bool, budget time.Duration) error {
	deadline := time.Now().Add(budget)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := validatePaths(r, false); err != nil {
			return err
		}
		err := remove(r.Process.StateDirectory)
		if err == nil || !retryable(err) {
			return err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return err
		}
		delay := min(25*time.Millisecond, remaining)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		// Do not start another removal beyond the retry budget.
		if !time.Now().Before(deadline) {
			return err
		}
	}
}
