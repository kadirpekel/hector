package session

import (
	"fmt"

	"github.com/verikod/hector/pkg/config"
)

// NewSessionService creates a session Service using the provided database configuration.
// This is the new clean interface that accepts separated configs.
func NewSessionService(pool *config.DBPool, databaseDSN string) (Service, error) {
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

	return NewSQLSessionService(db, dbCfg.Dialect())
}

// NewSessionServiceFromConfig was the old interface that used config.Config.
// It has been removed. Use NewSessionService instead.
