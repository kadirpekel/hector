// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	// SQL schema for rate_limits table.
	createRateLimitTableSQL = `
CREATE TABLE IF NOT EXISTS rate_limits (
    scope VARCHAR(50) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    limit_type VARCHAR(50) NOT NULL,
    window VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    window_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (scope, identifier, limit_type, window)
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_window_end ON rate_limits(window_end);
CREATE INDEX IF NOT EXISTS idx_rate_limits_identifier ON rate_limits(identifier);
CREATE INDEX IF NOT EXISTS idx_rate_limits_scope_identifier ON rate_limits(scope, identifier);
`
)

// SQLStore is a SQL-based implementation of Store.
// It supports Postgres, MySQL, and SQLite.
type SQLStore struct {
	db      *sql.DB
	dialect string
}

// NewSQLStore creates a new SQL-based store.
// Supported dialects: "postgres", "mysql", "sqlite".
func NewSQLStore(db *sql.DB, dialect string) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	switch dialect {
	case "postgres", "mysql", "sqlite":
		// Valid dialects
	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLStore{
		db:      db,
		dialect: dialect,
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// initSchema creates the necessary tables.
func (s *SQLStore) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := s.db.ExecContext(ctx, createRateLimitTableSQL); err != nil {
		return fmt.Errorf("failed to create rate_limits table: %w", err)
	}

	return nil
}

// GetUsage gets current usage for a specific limit.
func (s *SQLStore) GetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow) (int64, time.Time, error) {
	query := `SELECT amount, window_end FROM rate_limits WHERE scope = ? AND identifier = ? AND limit_type = ? AND window = ?`
	if s.dialect == "postgres" {
		query = `SELECT amount, window_end FROM rate_limits WHERE scope = $1 AND identifier = $2 AND limit_type = $3 AND window = $4`
	}

	var amount int64
	var windowEnd time.Time
	err := s.db.QueryRowContext(ctx, query, string(scope), identifier, string(limitType), string(window)).Scan(&amount, &windowEnd)

	if err == sql.ErrNoRows {
		// No record exists, return 0 with future window
		return 0, time.Now().Add(window.Duration()), nil
	}

	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to query usage: %w", err)
	}

	// Check if window has expired
	now := time.Now()
	if windowEnd.Before(now) {
		// Window expired, return 0 with new window
		return 0, now.Add(window.Duration()), nil
	}

	return amount, windowEnd, nil
}

// IncrementUsage increments usage for a specific limit.
func (s *SQLStore) IncrementUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, incrementAmount int64) (int64, time.Time, error) {
	now := time.Now()

	// Try to update existing record
	var updateQuery string
	if s.dialect == "postgres" {
		updateQuery = `
			UPDATE rate_limits 
			SET amount = amount + $1, updated_at = $2
			WHERE scope = $3 AND identifier = $4 AND limit_type = $5 AND window = $6 AND window_end > $7
			RETURNING amount, window_end
		`
	} else {
		updateQuery = `
			UPDATE rate_limits 
			SET amount = amount + ?, updated_at = ?
			WHERE scope = ? AND identifier = ? AND limit_type = ? AND window = ? AND window_end > ?
		`
	}

	var amount int64
	var windowEnd time.Time

	if s.dialect == "postgres" {
		// Postgres supports RETURNING
		err := s.db.QueryRowContext(ctx, updateQuery, incrementAmount, now, string(scope), identifier, string(limitType), string(window), now).Scan(&amount, &windowEnd)
		if err == nil {
			return amount, windowEnd, nil
		}
		if err != sql.ErrNoRows {
			return 0, time.Time{}, fmt.Errorf("failed to update usage: %w", err)
		}
	} else {
		// MySQL and SQLite don't support RETURNING, so we update then select
		result, err := s.db.ExecContext(ctx, updateQuery, incrementAmount, now, string(scope), identifier, string(limitType), string(window), now)
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("failed to update usage: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected > 0 {
			// Record was updated, fetch the current value
			amount, windowEnd, err = s.GetUsage(ctx, scope, identifier, limitType, window)
			if err != nil {
				return 0, time.Time{}, fmt.Errorf("failed to get usage after update: %w", err)
			}
			return amount, windowEnd, nil
		}
	}

	// No record exists or window expired, insert new record
	newWindowEnd := now.Add(window.Duration())
	insertQuery := `
		INSERT INTO rate_limits (scope, identifier, limit_type, window, amount, window_end, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	if s.dialect == "postgres" {
		insertQuery = `
			INSERT INTO rate_limits (scope, identifier, limit_type, window, amount, window_end, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (scope, identifier, limit_type, window) 
			DO UPDATE SET amount = EXCLUDED.amount, window_end = EXCLUDED.window_end, updated_at = EXCLUDED.updated_at
		`
	}

	_, err := s.db.ExecContext(ctx, insertQuery, string(scope), identifier, string(limitType), string(window), incrementAmount, newWindowEnd, now, now)
	if err != nil {
		// Handle potential race condition with another insert
		if s.dialect == "mysql" || s.dialect == "sqlite" {
			// Retry the update
			return s.IncrementUsage(ctx, scope, identifier, limitType, window, incrementAmount)
		}
		return 0, time.Time{}, fmt.Errorf("failed to insert usage: %w", err)
	}

	return incrementAmount, newWindowEnd, nil
}

// SetUsage sets usage for a specific limit.
func (s *SQLStore) SetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64, windowEnd time.Time) error {
	now := time.Now()

	var query string
	if s.dialect == "postgres" {
		query = `
			INSERT INTO rate_limits (scope, identifier, limit_type, window, amount, window_end, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (scope, identifier, limit_type, window) 
			DO UPDATE SET amount = EXCLUDED.amount, window_end = EXCLUDED.window_end, updated_at = EXCLUDED.updated_at
		`
	} else if s.dialect == "mysql" {
		query = `
			INSERT INTO rate_limits (scope, identifier, limit_type, window, amount, window_end, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE amount = VALUES(amount), window_end = VALUES(window_end), updated_at = VALUES(updated_at)
		`
	} else {
		// SQLite
		query = `
			INSERT OR REPLACE INTO rate_limits (scope, identifier, limit_type, window, amount, window_end, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
	}

	_, err := s.db.ExecContext(ctx, query, string(scope), identifier, string(limitType), string(window), amount, windowEnd, now, now)
	if err != nil {
		return fmt.Errorf("failed to set usage: %w", err)
	}

	return nil
}

// DeleteUsage deletes usage records for an identifier.
func (s *SQLStore) DeleteUsage(ctx context.Context, scope Scope, identifier string) error {
	query := `DELETE FROM rate_limits WHERE scope = ? AND identifier = ?`
	if s.dialect == "postgres" {
		query = `DELETE FROM rate_limits WHERE scope = $1 AND identifier = $2`
	}

	_, err := s.db.ExecContext(ctx, query, string(scope), identifier)
	if err != nil {
		return fmt.Errorf("failed to delete usage: %w", err)
	}

	return nil
}

// DeleteExpired deletes expired usage records.
func (s *SQLStore) DeleteExpired(ctx context.Context, before time.Time) error {
	query := `DELETE FROM rate_limits WHERE window_end < ?`
	if s.dialect == "postgres" {
		query = `DELETE FROM rate_limits WHERE window_end < $1`
	}

	_, err := s.db.ExecContext(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to delete expired records: %w", err)
	}

	return nil
}

// Close closes the store.
// Note: This does NOT close the underlying database connection,
// as that connection may be shared with other components.
func (s *SQLStore) Close() error {
	// Don't close the database connection as it may be shared
	return nil
}

// Dialect returns the SQL dialect (for testing).
func (s *SQLStore) Dialect() string {
	return s.dialect
}
