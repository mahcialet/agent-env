package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
	"github.com/mahcialet/agent-env/internal/paths"
	"github.com/mahcialet/agent-env/internal/store/sqlite"
)

// Create's first store operation is Reserve, after application preflight. Delay
// even opening SQLite until then: opening a registry mutates its directory and
// must not write inside an image or AVD template rejected by that preflight.
func openCreateService(out, errOut io.Writer) (*app.Service, func() error, error) {
	home, err := paths.Resolve()
	if err != nil {
		return nil, nil, err
	}
	store := &createStore{home: home}
	return serviceForStore(home, store, out, errOut), store.Close, nil
}

type createStore struct {
	app.Store
	home  string
	close func() error
}

func (s *createStore) Reserve(ctx context.Context, lease domain.Lease, maxActive int) error {
	if s.Store == nil {
		if err := os.MkdirAll(s.home, 0700); err != nil {
			return err
		}
		store, err := sqlite.Open(filepath.Join(s.home, "state.db"))
		if err != nil {
			return err
		}
		s.Store, s.close = store, store.Close
	}
	return s.Store.Reserve(ctx, lease, maxActive)
}

func (s *createStore) Close() error {
	if s.close == nil {
		return nil
	}
	return s.close()
}
