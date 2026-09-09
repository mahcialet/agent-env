package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/oklog/ulid/v2"
)

var ErrManagementAuthority = errors.New("controller assignment authority required")

func validManagement(m *domain.Management) bool {
	if m == nil || m.AssignmentEpoch == 0 {
		return false
	}
	for _, value := range []string{m.ControllerID, m.HostID, m.HostInstanceID} {
		if len(value) == 0 || len(value) > 128 || value == "." || value == ".." {
			return false
		}
		for _, c := range value {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
				return false
			}
		}
	}
	return true
}

func copyManagement(m *domain.Management) *domain.Management {
	if m == nil {
		return nil
	}
	copy := *m
	return &copy
}

func (s *Service) validateCreateManagement(o CreateOptions) error {
	if o.Management == nil && s.Management == nil && o.LeaseID == "" {
		return nil
	}
	if !validManagement(o.Management) || !validManagement(s.Management) || *o.Management != *s.Management {
		return fmt.Errorf("%w: create requires the worker's exact controller, host, instance and epoch", ErrManagementAuthority)
	}
	id, err := ulid.ParseStrict(o.LeaseID)
	if err != nil || id.String() != o.LeaseID {
		return errors.New("controller lease ID must be a canonical ULID")
	}
	return nil
}

func (s *Service) authorizeManagement(l domain.Lease) error {
	if l.Management == nil && s.Management == nil {
		return nil
	}
	if !validManagement(l.Management) || !validManagement(s.Management) || *l.Management != *s.Management {
		return fmt.Errorf("%w: local mutation or a different controller/host/instance/epoch cannot operate on lease %s", ErrManagementAuthority, l.ID)
	}
	return nil
}

func (s *Service) checkManagement(ctx context.Context, id string) error {
	l, err := s.Store.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.authorizeManagement(l)
}

// Check before lock writes/cancellation and again under the existing local fence.
// Management is immutable; no operation can adopt or advance an assignment.
func (s *Service) acquireManagedOperation(ctx context.Context, id string) (context.Context, func() error, error) {
	if err := s.checkManagement(ctx, id); err != nil {
		return nil, nil, err
	}
	op, release, err := s.Store.AcquireContext(ctx, id, newID(), 2*time.Minute)
	if err != nil {
		return nil, nil, err
	}
	if err = s.checkManagement(op, id); err != nil {
		return nil, nil, errors.Join(err, release())
	}
	return op, release, nil
}
