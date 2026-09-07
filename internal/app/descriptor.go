package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/evidence"
)

// The database remains authoritative; the descriptor is portable diagnostic evidence.
func (s *Service) persist(ctx context.Context, l domain.Lease) error {
	if err := s.Store.Save(ctx, l); err != nil {
		return err
	}
	directory := filepath.Join(s.Home, "leases", l.ID)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		SchemaVersion int          `json:"schema_version"`
		Lease         domain.Lease `json:"lease"`
	}{1, l}, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "environment.json")
	if err := evidence.AtomicWrite(path, append(data, '\n'), 0600); err != nil {
		return err
	}
	return nil
}
