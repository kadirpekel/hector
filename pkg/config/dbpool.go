// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DBPool manages shared database connections.
// For SQLite, it ensures only one connection is used to prevent "database is locked" errors.
type DBPool struct {
	mu    sync.Mutex
	pools map[string]*sql.DB
}

// NewDBPool creates a new database pool manager.
func NewDBPool() *DBPool {
	return &DBPool{
		pools: make(map[string]*sql.DB),
	}
}

// Get returns a database connection for the given config.
// For the same DSN, it returns the same connection pool.
func (p *DBPool) Get(cfg *DatabaseConfig) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	dsn := cfg.DSN()

	// Return existing pool if available
	if db, ok := p.pools[dsn]; ok {
		return db, nil
	}

	// Create new pool
	db, err := p.createPool(cfg)
	if err != nil {
		return nil, err
	}

	p.pools[dsn] = db
	return db, nil
}

func (p *DBPool) createPool(cfg *DatabaseConfig) (*sql.DB, error) {
	driverName := cfg.DriverName()
	dsn := cfg.DSN()

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite only supports one writer at a time. Using a single connection
	// serializes all database access and prevents "database is locked" errors.
	if driverName == "sqlite3" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		slog.Debug("SQLite: using single connection mode")
	} else {
		if cfg.MaxConns > 0 {
			db.SetMaxOpenConns(cfg.MaxConns)
		}
		if cfg.MaxIdle > 0 {
			db.SetMaxIdleConns(cfg.MaxIdle)
		}
	}
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable SQLite optimizations
	if driverName == "sqlite3" {
		if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
			slog.Warn("Failed to enable WAL mode", "error", err)
		} else {
			slog.Debug("Enabled WAL mode for SQLite")
		}
		if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout=10000"); err != nil {
			slog.Warn("Failed to set busy timeout", "error", err)
		}
	}

	return db, nil
}

// Close closes all database connections.
func (p *DBPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for dsn, db := range p.pools {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", dsn, err))
		}
	}
	p.pools = make(map[string]*sql.DB)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing pools: %v", errs)
	}
	return nil
}
