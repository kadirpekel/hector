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

package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// SQLTaskStore implements a2asrv.TaskStore using SQL database.
// It stores a2a.Task objects directly (as required by the interface).
type SQLTaskStore struct {
	db      *sql.DB
	dialect string
}

// taskStoreRow represents a database row for an a2a.Task.
type taskStoreRow struct {
	ID            string
	ContextID     string
	StatusJSON    string
	HistoryJSON   string
	ArtifactsJSON string
	MetadataJSON  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

const (
	// createTaskStoreTableSQL creates the a2a_tasks table.
	// Using separate statements for table and indexes to ensure SQLite compatibility.
	createTaskStoreTableSQL = `
CREATE TABLE IF NOT EXISTS a2a_tasks (
    id VARCHAR(255) PRIMARY KEY,
    context_id VARCHAR(255) NOT NULL,
    status_json TEXT NOT NULL,
    history_json TEXT,
    artifacts_json TEXT,
    metadata_json TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
)`

	createTaskStoreIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_context_id ON a2a_tasks(context_id)`

	createTaskStoreUpdatedAtIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_updated_at ON a2a_tasks(updated_at)`
)

// NewSQLTaskStore creates a new SQL-based TaskStore implementing a2asrv.TaskStore.
// The db connection should be shared with other services using the same database
// to prevent SQLite "database is locked" errors.
func NewSQLTaskStore(db *sql.DB, dialect string) (a2asrv.TaskStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Normalize dialect
	normalizedDialect := dialect
	if dialect == "sqlite3" {
		normalizedDialect = "sqlite"
	}

	switch normalizedDialect {
	case "postgres", "mysql", "sqlite":
		// Valid dialects
	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLTaskStore{
		db:      db,
		dialect: normalizedDialect,
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// initSchema creates the necessary tables and indexes.
func (s *SQLTaskStore) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create table
	if _, err := s.db.ExecContext(ctx, createTaskStoreTableSQL); err != nil {
		return fmt.Errorf("failed to create a2a_tasks table: %w", err)
	}

	// Create indexes (separate statements for SQLite compatibility)
	if _, err := s.db.ExecContext(ctx, createTaskStoreIndexesSQL); err != nil {
		return fmt.Errorf("failed to create context_id index: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, createTaskStoreUpdatedAtIndexSQL); err != nil {
		return fmt.Errorf("failed to create updated_at index: %w", err)
	}

	return nil
}

// Save stores a task (implements a2asrv.TaskStore).
// Uses UPSERT for atomic insert/update. For optimistic concurrency, we check
// if the task was modified since we loaded it by comparing timestamps stored
// in task.Metadata["_updated_at"]. If stale, we log a warning but proceed
// (a2a protocol handles task state transitions).
func (s *SQLTaskStore) Save(ctx context.Context, task *a2a.Task) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}

	// Check for stale update if task has _updated_at in metadata
	if task.Metadata != nil {
		if expectedUpdatedAt, ok := task.Metadata["_updated_at"].(string); ok {
			currentUpdatedAt, err := s.getTaskUpdatedAt(ctx, task.ID)
			if err == nil && currentUpdatedAt != "" && currentUpdatedAt != expectedUpdatedAt {
				slog.Warn("Potential stale task update",
					"taskID", task.ID,
					"expected", expectedUpdatedAt,
					"current", currentUpdatedAt)
			}
		}
	}

	row, err := s.taskToRow(task)
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	// Use UPSERT: INSERT ... ON CONFLICT UPDATE (PostgreSQL) or INSERT ... ON DUPLICATE KEY UPDATE (MySQL)
	// For SQLite, use INSERT OR REPLACE
	query := `
INSERT INTO a2a_tasks (id, context_id, status_json, history_json, artifacts_json, metadata_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    context_id = VALUES(context_id),
    status_json = VALUES(status_json),
    history_json = VALUES(history_json),
    artifacts_json = VALUES(artifacts_json),
    metadata_json = VALUES(metadata_json),
    updated_at = VALUES(updated_at)
`
	if s.dialect == "postgres" {
		query = `
INSERT INTO a2a_tasks (id, context_id, status_json, history_json, artifacts_json, metadata_json, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    context_id = EXCLUDED.context_id,
    status_json = EXCLUDED.status_json,
    history_json = EXCLUDED.history_json,
    artifacts_json = EXCLUDED.artifacts_json,
    metadata_json = EXCLUDED.metadata_json,
    updated_at = EXCLUDED.updated_at
`
	} else if s.dialect == "sqlite" {
		// SQLite 3.24+ supports ON CONFLICT (UPSERT)
		// This preserves created_at on update unlike INSERT OR REPLACE
		query = `
INSERT INTO a2a_tasks (id, context_id, status_json, history_json, artifacts_json, metadata_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    context_id = excluded.context_id,
    status_json = excluded.status_json,
    history_json = excluded.history_json,
    artifacts_json = excluded.artifacts_json,
    metadata_json = excluded.metadata_json,
    updated_at = excluded.updated_at
`
	}

	args := []interface{}{
		row.ID, row.ContextID, row.StatusJSON,
		row.HistoryJSON, row.ArtifactsJSON, row.MetadataJSON,
		row.CreatedAt, row.UpdatedAt,
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

// Get retrieves a task by ID (implements a2asrv.TaskStore).
func (s *SQLTaskStore) Get(ctx context.Context, taskID a2a.TaskID) (*a2a.Task, error) {
	slog.Debug("TaskStore.Get called", "taskID", taskID)

	query := `
SELECT id, context_id, status_json, history_json, artifacts_json, metadata_json, created_at, updated_at
FROM a2a_tasks
WHERE id = ?
`
	if s.dialect == "postgres" {
		query = `
SELECT id, context_id, status_json, history_json, artifacts_json, metadata_json, created_at, updated_at
FROM a2a_tasks
WHERE id = $1
`
	}

	var row taskStoreRow
	err := s.db.QueryRowContext(ctx, query, string(taskID)).Scan(
		&row.ID, &row.ContextID, &row.StatusJSON,
		&row.HistoryJSON, &row.ArtifactsJSON, &row.MetadataJSON,
		&row.CreatedAt, &row.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		slog.Debug("TaskStore.Get: task not found", "taskID", taskID)
		return nil, a2a.ErrTaskNotFound
	}
	if err != nil {
		slog.Error("TaskStore.Get: query error", "taskID", taskID, "error", err)
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	slog.Debug("TaskStore.Get: found task", "taskID", taskID, "contextID", row.ContextID)
	return s.rowToTask(&row)
}

// Close closes the database connection.
func (s *SQLTaskStore) Close() error {
	return s.db.Close()
}

// getTaskUpdatedAt returns the current updated_at timestamp for a task.
func (s *SQLTaskStore) getTaskUpdatedAt(ctx context.Context, taskID a2a.TaskID) (string, error) {
	query := `SELECT updated_at FROM a2a_tasks WHERE id = ?`
	if s.dialect == "postgres" {
		query = `SELECT updated_at FROM a2a_tasks WHERE id = $1`
	}

	var updatedAt time.Time
	err := s.db.QueryRowContext(ctx, query, string(taskID)).Scan(&updatedAt)
	if err != nil {
		return "", err
	}
	return updatedAt.Format(time.RFC3339Nano), nil
}

// taskToRow converts an a2a.Task to a database row.
func (s *SQLTaskStore) taskToRow(task *a2a.Task) (*taskStoreRow, error) {
	now := time.Now()

	// Serialize Status (required field)
	statusJSON, err := json.Marshal(task.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	// Serialize History
	var historyJSON []byte
	if len(task.History) > 0 {
		historyJSON, err = json.Marshal(task.History)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal history: %w", err)
		}
	} else {
		historyJSON = []byte("[]")
	}

	// Serialize Artifacts
	var artifactsJSON []byte
	if len(task.Artifacts) > 0 {
		artifactsJSON, err = json.Marshal(task.Artifacts)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal artifacts: %w", err)
		}
	} else {
		artifactsJSON = []byte("[]")
	}

	// Serialize Metadata
	var metadataJSON []byte
	if len(task.Metadata) > 0 {
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	return &taskStoreRow{
		ID:            string(task.ID),
		ContextID:     task.ContextID,
		StatusJSON:    string(statusJSON),
		HistoryJSON:   string(historyJSON),
		ArtifactsJSON: string(artifactsJSON),
		MetadataJSON:  string(metadataJSON),
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// rowToTask converts a database row to an a2a.Task.
func (s *SQLTaskStore) rowToTask(row *taskStoreRow) (*a2a.Task, error) {
	task := &a2a.Task{
		ID:        a2a.TaskID(row.ID),
		ContextID: row.ContextID,
	}

	// Deserialize Status (required)
	if row.StatusJSON == "" {
		return nil, fmt.Errorf("status_json is required but was empty")
	}
	if err := json.Unmarshal([]byte(row.StatusJSON), &task.Status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	// Deserialize History
	if row.HistoryJSON != "" && row.HistoryJSON != "[]" {
		var history []*a2a.Message
		if err := json.Unmarshal([]byte(row.HistoryJSON), &history); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history: %w", err)
		}
		task.History = history
	} else {
		task.History = make([]*a2a.Message, 0)
	}

	// Deserialize Artifacts
	if row.ArtifactsJSON != "" && row.ArtifactsJSON != "[]" {
		var artifacts []*a2a.Artifact
		if err := json.Unmarshal([]byte(row.ArtifactsJSON), &artifacts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
		}
		task.Artifacts = artifacts
	} else {
		task.Artifacts = make([]*a2a.Artifact, 0)
	}

	// Deserialize Metadata
	if row.MetadataJSON != "" && row.MetadataJSON != "{}" {
		if err := json.Unmarshal([]byte(row.MetadataJSON), &task.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		task.Metadata = make(map[string]any)
	}

	// Store updated_at in metadata for optimistic concurrency tracking
	task.Metadata["_updated_at"] = row.UpdatedAt.Format(time.RFC3339Nano)

	return task, nil
}

// Compile-time interface compliance check
var _ a2asrv.TaskStore = (*SQLTaskStore)(nil)
