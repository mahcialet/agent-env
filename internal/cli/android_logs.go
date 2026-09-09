package cli

import (
	"github.com/mahcialet/agent-env/internal/app"
	"github.com/mahcialet/agent-env/internal/domain"
)

func androidProcessLogEntries(home, leaseID string, runtime domain.Runtime) (map[string]string, error) {
	return app.AndroidProcessLogEntries(home, leaseID, runtime)
}
