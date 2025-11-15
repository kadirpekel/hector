package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/config"
)

// DataSourceFactory creates DataSource instances from configuration
type DataSourceFactory struct{}

// NewDataSourceFactory creates a new data source factory
func NewDataSourceFactory() *DataSourceFactory {
	return &DataSourceFactory{}
}

// CreateDataSource creates a DataSource from DocumentStoreConfig
func (f *DataSourceFactory) CreateDataSource(cfg *config.DocumentStoreConfig) (DataSource, error) {
	// If collection is set, this is a collection-only store (no indexing)
	if cfg.Collection != "" {
		return NewCollectionSource(cfg.Collection), nil
	}

	sourceType := cfg.Source
	if sourceType == "" {
		sourceType = "directory" // Default
	}

	switch sourceType {
	case "directory":
		return f.createDirectorySource(cfg)
	case "sql":
		return f.createSQLSource(cfg)
	case "api":
		return f.createAPISource(cfg)
	default:
		return nil, fmt.Errorf("unsupported source type: %s (supported: directory, sql, api)", sourceType)
	}
}

func (f *DataSourceFactory) createDirectorySource(cfg *config.DocumentStoreConfig) (DataSource, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("path is required for directory source")
	}

	// Check if path exists
	if _, err := os.Stat(cfg.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory path does not exist: %s", cfg.Path)
	}

	// Create pattern filter
	filter, err := NewPatternFilter(cfg.Path, cfg.IncludePatterns, cfg.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern filter: %w", err)
	}

	maxFileSize := cfg.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 10 * 1024 * 1024 // 10MB default
	}

	return NewDirectorySource(cfg.Path, filter, maxFileSize), nil
}

func (f *DataSourceFactory) createSQLSource(cfg *config.DocumentStoreConfig) (DataSource, error) {
	if cfg.SQL == nil {
		return nil, fmt.Errorf("SQL configuration is required for SQL source")
	}

	sqlConfig := cfg.SQL
	if sqlConfig.Driver == "" {
		return nil, fmt.Errorf("SQL driver is required")
	}
	if sqlConfig.Database == "" {
		return nil, fmt.Errorf("SQL database name is required")
	}

	// Build connection string based on driver
	var dsn string
	switch sqlConfig.Driver {
	case "postgres", "pgx":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			sqlConfig.Host, sqlConfig.Port, sqlConfig.Username, sqlConfig.Password,
			sqlConfig.Database, sqlConfig.SSLMode)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			sqlConfig.Username, sqlConfig.Password, sqlConfig.Host, sqlConfig.Port, sqlConfig.Database)
	case "sqlite3":
		dsn = sqlConfig.Database // For SQLite, database is the file path
	default:
		return nil, fmt.Errorf("unsupported SQL driver: %s", sqlConfig.Driver)
	}

	db, err := sql.Open(sqlConfig.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Convert table configs
	tableConfigs := make([]SQLTableConfig, 0, len(cfg.SQLTables))
	for _, tc := range cfg.SQLTables {
		tableConfigs = append(tableConfigs, SQLTableConfig{
			Table:           tc.Table,
			Columns:         tc.Columns,
			IDColumn:        tc.IDColumn,
			UpdatedColumn:   tc.UpdatedColumn,
			WhereClause:     tc.WhereClause,
			MetadataColumns: tc.MetadataColumns,
		})
	}

	maxRows := cfg.SQLMaxRows
	if maxRows == 0 {
		maxRows = 10000 // Default limit
	}

	return NewSQLSource(db, sqlConfig.Driver, tableConfigs, maxRows), nil
}

func (f *DataSourceFactory) createAPISource(cfg *config.DocumentStoreConfig) (DataSource, error) {
	if cfg.API == nil {
		return nil, fmt.Errorf("API configuration is required for API source")
	}

	apiConfig := cfg.API
	if apiConfig.BaseURL == "" {
		return nil, fmt.Errorf("API base URL is required")
	}
	if len(apiConfig.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one API endpoint is required")
	}

	// Convert endpoint configs
	endpoints := make([]APIEndpointConfig, 0, len(apiConfig.Endpoints))
	for _, ec := range apiConfig.Endpoints {
		var pagination *PaginationConfig
		if ec.Pagination != nil {
			pagination = &PaginationConfig{
				Type:      ec.Pagination.Type,
				PageParam: ec.Pagination.PageParam,
				SizeParam: ec.Pagination.SizeParam,
				MaxPages:  ec.Pagination.MaxPages,
				PageSize:  ec.Pagination.PageSize,
				NextField: ec.Pagination.NextField,
				DataField: ec.Pagination.DataField,
			}
		}

		endpoints = append(endpoints, APIEndpointConfig{
			Path:           ec.Path,
			Method:         ec.Method,
			Params:         ec.Params,
			Headers:        ec.Headers,
			Body:           ec.Body,
			IDField:        ec.IDField,
			ContentField:   ec.ContentField,
			MetadataFields: ec.MetadataFields,
			UpdatedField:   ec.UpdatedField,
			Pagination:     pagination,
		})
	}

	// Convert global auth config
	var globalAuth *APIAuthConfig
	if apiConfig.Auth != nil {
		globalAuth = &APIAuthConfig{
			Type:   apiConfig.Auth.Type,
			Token:  apiConfig.Auth.Token,
			User:   apiConfig.Auth.User,
			Pass:   apiConfig.Auth.Pass,
			Header: apiConfig.Auth.Header,
			Extra:  apiConfig.Auth.Extra,
		}
	}

	return NewAPISource(apiConfig.BaseURL, endpoints, globalAuth), nil
}
