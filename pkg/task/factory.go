package task

import (
	"fmt"

	"github.com/a2aproject/a2a-go/a2asrv"

	"github.com/verikod/hector/pkg/config"
)

// NewTaskStore creates a TaskStore using the provided database configuration.
// This is the new clean interface that accepts separated configs.
func NewTaskStore(pool *config.DBPool, databaseDSN string) (a2asrv.TaskStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("DBPool is required")
	}

	if databaseDSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	// Parse database config
	dbCfg, err := config.ParseDSN(databaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	// Get database connection from pool
	db, err := pool.Get(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	return NewSQLTaskStore(db, dbCfg.Dialect())
}

// NewTaskStoreFromConfig was the old interface that used config.Config.
// It has been removed. Use NewTaskStore instead.

// NewService creates a new persistent task service.
func NewService(store a2asrv.TaskStore) (Service, error) {
	persistentStore, ok := store.(PersistentTaskStore)
	if !ok {
		return nil, fmt.Errorf("store must implement PersistentTaskStore interface (List method required)")
	}
	return NewPersistentService(persistentStore), nil
}
