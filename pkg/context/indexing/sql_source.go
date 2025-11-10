package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLSource implements DataSource for SQL databases using database/sql
type SQLSource struct {
	db           *sql.DB
	driver       string
	tableConfigs []SQLTableConfig
	maxRows      int
}

// SQLTableConfig defines which tables and columns to index
type SQLTableConfig struct {
	Table           string   `yaml:"table"`
	Columns         []string `yaml:"columns"`          // Columns to concatenate for content
	IDColumn        string   `yaml:"id_column"`        // Primary key or unique identifier
	UpdatedColumn   string   `yaml:"updated_column"`   // Column for tracking updates (e.g., updated_at)
	WhereClause     string   `yaml:"where_clause"`     // Optional WHERE clause for filtering
	MetadataColumns []string `yaml:"metadata_columns"` // Columns to include as metadata
}

// NewSQLSource creates a new SQL data source
func NewSQLSource(db *sql.DB, driver string, tableConfigs []SQLTableConfig, maxRows int) *SQLSource {
	return &SQLSource{
		db:           db,
		driver:       driver,
		tableConfigs: tableConfigs,
		maxRows:      maxRows,
	}
}

func (s *SQLSource) Type() string {
	return "sql"
}

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
		case "sqlite3":
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
	contentStartIdx := len(config.Columns)
	metadataStartIdx := contentStartIdx + 1
	if config.UpdatedColumn != "" {
		metadataStartIdx++
	}

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
		metadata := make(map[string]interface{})
		metadata["table"] = config.Table
		metadata["id"] = id
		for i, col := range config.MetadataColumns {
			idx := metadataStartIdx + i
			if idx < len(values) && values[idx] != nil {
				metadata[col] = values[idx]
			}
		}

		// Calculate approximate size
		size := int64(len(content))

		doc := Document{
			ID:           fmt.Sprintf("%s:%s:%s", s.driver, config.Table, id),
			Content:      content,
			Metadata:     metadata,
			LastModified: lastModified,
			Size:         size,
			ShouldIndex:  true,
		}

		select {
		case docChan <- doc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return rows.Err()
}

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

	// QueryRow doesn't have Columns() method, we need to use the table config
	columnNames := append(tableConfig.Columns, tableConfig.IDColumn)
	if tableConfig.UpdatedColumn != "" {
		columnNames = append(columnNames, tableConfig.UpdatedColumn)
	}
	columnNames = append(columnNames, tableConfig.MetadataColumns...)

	values := make([]interface{}, len(columnNames))
	valuePtrs := make([]interface{}, len(columnNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Build document (similar to DiscoverDocuments)
	idIdx := 0
	for i, col := range columnNames {
		if col == tableConfig.IDColumn {
			idIdx = i
			break
		}
	}

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
	metadata := make(map[string]interface{})
	metadata["table"] = tableConfig.Table
	metadata["id"] = values[idIdx]
	
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
		ID:           id,
		Content:      content,
		Metadata:     metadata,
		LastModified: lastModified,
		Size:         int64(len(content)),
		ShouldIndex:  true,
	}, nil
}

func (s *SQLSource) SupportsIncrementalIndexing() bool {
	// SQL sources support incremental indexing if UpdatedColumn is configured
	for _, cfg := range s.tableConfigs {
		if cfg.UpdatedColumn != "" {
			return true
		}
	}
	return false
}

func (s *SQLSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	doc, err := s.ReadDocument(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	return doc.LastModified, nil
}

func (s *SQLSource) Close() error {
	return s.db.Close()
}
