package app

import (
	"context"
	"errors"

	"github.com/mahcialet/agent-env/internal/domain"
)

// AndroidProvider owns SDK discovery and native Emulator effects. Application
// orchestration owns reservations, operation fencing and durable lease state.
type AndroidProvider interface {
	Doctor(context.Context) (map[string]string, error)
	Validate(context.Context, string) (domain.AndroidEmulator, error)
	Create(context.Context, domain.Runtime) (domain.Runtime, error)
	Inspect(context.Context, domain.Runtime) (RuntimeObservation, error)
	Destroy(context.Context, domain.Runtime) error
}

func (s *Service) inspectRuntime(ctx context.Context, r domain.Runtime) (RuntimeObservation, error) {
	if r.Type == "android-emulator" {
		if s.Android == nil {
			return RuntimeObservation{}, errors.New("Android provider unavailable")
		}
		return s.Android.Inspect(ctx, r)
	}
	if s.Runtime == nil {
		return RuntimeObservation{}, errors.New("Compose provider unavailable")
	}
	return s.Runtime.Inspect(ctx, r)
}

func observeAndroid(r *domain.Runtime, o RuntimeObservation, err error) {
	if r.Android == nil {
		return
	}
	switch {
	case errors.Is(err, domain.ErrResourceIdentity):
		r.Android.State = "quarantined"
	case err != nil:
		r.Android.State = "unknown"
	case !r.Started && r.Android.State == "released":
	case o.Ready:
		r.Android.State = "ready"
	case o.Exists:
		r.Android.State = "starting"
	default:
		r.Android.State = "stopped"
	}
}
