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

package rag

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLSource implements DataSource for SQL databases using database/sql.
//
// Direct port from legacy pkg/context/indexing/sql_source.go
type SQLSource struct {
	db           *sql.DB
	driver       string
	tableConfigs []SQLTableConfig
	maxRows      int
}

// SQLTableConfig defines which tables and columns to index.
//
// Direct port from legacy pkg/context/indexing/sql_source.go
type SQLTableConfig struct {
	Table           string   `yaml:"table"`
	Columns         []string `yaml:"columns"`          // Columns to concatenate for content
	IDColumn        string   `yaml:"id_column"`        // Primary key or unique identifier
	UpdatedColumn   string   `yaml:"updated_column"`   // Column for tracking updates (e.g., updated_at)
	WhereClause     string   `yaml:"where_clause"`     // Optional WHERE clause for filtering
	MetadataColumns []string `yaml:"metadata_columns"` // Columns to include as metadata
}

// SQLSourceOptions configures the SQL source.
type SQLSourceOptions struct {
	DB      *sql.DB
	Driver  string
	Tables  []SQLTableConfig
	MaxRows int
}

// NewSQLSource creates a new SQL data source.
//
// Direct port from legacy pkg/context/indexing/sql_source.go
func NewSQLSource(opts SQLSourceOptions) (*SQLSource, error) {
	if opts.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if opts.Driver == "" {
		return nil, fmt.Errorf("driver is required")
	}
	if len(opts.Tables) == 0 {
		return nil, fmt.Errorf("at least one table configuration is required")
	}
	return &SQLSource{
		db:           opts.DB,
		driver:       opts.Driver,
		tableConfigs: opts.Tables,
		maxRows:      opts.MaxRows,
	}, nil
}

// Type returns the data source type.
func (s *SQLSource) Type() string {
	return "sql"
}

// DiscoverDocuments returns channels of discovered documents and errors.
//
// Direct port from legacy pkg/context/indexing/sql_source.go
func (s *SQLSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(docChan)
		defer close(errChan)

		for _, tableConfig := range s.tableConfigs {
			if err := s.indexTable(ctx, tableConfig, docChan, errChan); err != nil {
				select {
				case errChan <- fmt.Errorf("failed to index table %s: %w", tableConfig.Table, err):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return docChan, errChan
}

// indexTable indexes a single table.
func (s *SQLSource) indexTable(ctx context.Context, config SQLTableConfig, docChan chan<- Document, errChan chan<- error) error {
	// Build SELECT query
	columns := append(config.Columns, config.IDColumn)
	if config.UpdatedColumn != "" {
		columns = append(columns, config.UpdatedColumn)
	}
	columns = append(columns, config.MetadataColumns...)

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), config.Table)
	if config.WhereClause != "" {
		query += " WHERE " + config.WhereClause
	}
	if s.maxRows > 0 {
		// Add LIMIT clause (syntax varies by database)
		switch s.driver {
		case "postgres", "pgx":
			query += fmt.Sprintf(" LIMIT %d", s.maxRows)
		case "mysql":
			query += fmt.Sprintf(" LIMIT %d", s.maxRows)
		case "sqlite3", "sqlite":
			query += fmt.Sprintf(" LIMIT %d", s.maxRows)
		}
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Map column indices
	idIdx := -1
	updatedIdx := -1
	for i, col := range columnNames {
		if col == config.IDColumn {
			idIdx = i
		}
		if config.UpdatedColumn != "" && col == config.UpdatedColumn {
			updatedIdx = i
		}
	}

	if idIdx == -1 {
		return fmt.Errorf("ID column %s not found", config.IDColumn)
	}

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create slice to hold column values
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			select {
			case errChan <- fmt.Errorf("scan failed: %w", err):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// Extract ID
		id, ok := values[idIdx].(string)
		if !ok {
			// Try converting to string
			id = fmt.Sprintf("%v", values[idIdx])
		}

		// Build content from content columns
		contentParts := make([]string, 0, len(config.Columns))
		for i := 0; i < len(config.Columns); i++ {
			if val := values[i]; val != nil {
				contentParts = append(contentParts, fmt.Sprintf("%v", val))
			}
		}
		content := strings.Join(contentParts, "\n\n")

		// Extract last modified time
		var lastModified time.Time
		if updatedIdx >= 0 && values[updatedIdx] != nil {
			switch v := values[updatedIdx].(type) {
			case time.Time:
				lastModified = v
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					lastModified = t
				}
			}
		}

		// Build metadata
		metadata := make(map[string]any)
		metadata["table"] = config.Table
		metadata["id"] = id
		metadata["last_modified"] = lastModified.Unix()

		// Get metadata start index (after content columns, ID, and optionally UpdatedColumn)
		metadataStartIdx := len(config.Columns) + 1 // After content columns and ID
		if config.UpdatedColumn != "" {
			metadataStartIdx++ // After updated column
		}
		for i, col := range config.MetadataColumns {
			idx := metadataStartIdx + i
			if idx < len(values) && values[idx] != nil {
				metadata[col] = values[idx]
			}
		}

		// Calculate approximate size
		size := int64(len(content))

		// Document ID format: driver:table:id
		docID := fmt.Sprintf("%s:%s:%s", s.driver, config.Table, id)

		doc := Document{
			ID:         docID,
			Content:    content,
			SourcePath: fmt.Sprintf("%s/%s", config.Table, id),
			Size:       size,
			Metadata:   metadata,
		}

		select {
		case docChan <- doc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return rows.Err()
}

// ReadDocument retrieves a specific document by its ID.
//
// Direct port from legacy pkg/context/indexing/sql_source.go
func (s *SQLSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// Parse ID format: driver:table:id
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid document ID format: %s", id)
	}

	_, table, docID := parts[0], parts[1], parts[2]

	// Find table config
	var tableConfig *SQLTableConfig
	for _, cfg := range s.tableConfigs {
		if cfg.Table == table {
			tableConfig = &cfg
			break
		}
	}
	if tableConfig == nil {
		return nil, fmt.Errorf("table %s not found in configuration", table)
	}

	// Build query
	columns := append(tableConfig.Columns, tableConfig.IDColumn)
	if tableConfig.UpdatedColumn != "" {
		columns = append(columns, tableConfig.UpdatedColumn)
	}
	columns = append(columns, tableConfig.MetadataColumns...)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?",
		strings.Join(columns, ", "), tableConfig.Table, tableConfig.IDColumn)

	row := s.db.QueryRowContext(ctx, query, docID)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Build content
	contentParts := make([]string, 0, len(tableConfig.Columns))
	for i := 0; i < len(tableConfig.Columns); i++ {
		if val := values[i]; val != nil {
			contentParts = append(contentParts, fmt.Sprintf("%v", val))
		}
	}
	content := strings.Join(contentParts, "\n\n")

	// Extract last modified time
	var lastModified time.Time
	updatedIdx := len(tableConfig.Columns) + 1 // ID column is after content columns
	if tableConfig.UpdatedColumn != "" {
		if updatedIdx < len(values) && values[updatedIdx] != nil {
			switch v := values[updatedIdx].(type) {
			case time.Time:
				lastModified = v
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					lastModified = t
				}
			}
		}
	}

	// Build metadata
	metadata := make(map[string]any)
	metadata["table"] = tableConfig.Table
	metadata["id"] = docID
	metadata["last_modified"] = lastModified.Unix()

	// Extract metadata columns
	metadataStartIdx := len(tableConfig.Columns) + 1 // After ID column
	if tableConfig.UpdatedColumn != "" {
		metadataStartIdx++ // After updated column
	}
	for i, col := range tableConfig.MetadataColumns {
		idx := metadataStartIdx + i
		if idx < len(values) && values[idx] != nil {
			metadata[col] = values[idx]
		}
	}

	return &Document{
		ID:         id,
		Content:    content,
		SourcePath: fmt.Sprintf("%s/%s", tableConfig.Table, docID),
		Size:       int64(len(content)),
		Metadata:   metadata,
	}, nil
}

// SupportsIncrementalIndexing returns true if UpdatedColumn is configured.
func (s *SQLSource) SupportsIncrementalIndexing() bool {
	// SQL sources support incremental indexing if UpdatedColumn is configured
	for _, cfg := range s.tableConfigs {
		if cfg.UpdatedColumn != "" {
			return true
		}
	}
	return false
}

// GetLastModified returns the last modification time for a document.
func (s *SQLSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	doc, err := s.ReadDocument(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	if lastMod, ok := doc.Metadata["last_modified"].(int64); ok {
		return time.Unix(lastMod, 0), nil
	}
	return time.Time{}, nil
}

// Close closes the underlying database connection.
// Note: In most cases, the connection is managed externally (e.g., by DBPool),
// so this is a no-op. The caller should manage the lifecycle.
func (s *SQLSource) Close() error {
	// Don't close the DB here - it's managed by the caller/DBPool
	return nil
}

// Ensure SQLSource implements DataSource.
var _ DataSource = (*SQLSource)(nil)
