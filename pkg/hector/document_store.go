package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
)

// DocumentStoreBuilder provides a fluent API for building document store configurations
type DocumentStoreBuilder struct {
	config *config.DocumentStoreConfig
}

// NewDocumentStore creates a new document store builder
// source must be one of: "directory", "sql", "api"
// Note: The store name comes from the map key when used in config, or passed to WithDocumentStoreBuilder
func NewDocumentStore(source string) *DocumentStoreBuilder {
	if source == "" {
		panic("document store source is required")
	}
	if source != "directory" && source != "sql" && source != "api" {
		panic(fmt.Sprintf("invalid source: %s (must be 'directory', 'sql', or 'api')", source))
	}

	return &DocumentStoreBuilder{
		config: &config.DocumentStoreConfig{
			Source: source,
		},
	}
}

// Path sets the path for directory source (required for directory source)
func (b *DocumentStoreBuilder) Path(path string) *DocumentStoreBuilder {
	b.config.Path = path
	return b
}

// IncludePatterns sets the include patterns for directory source
func (b *DocumentStoreBuilder) IncludePatterns(patterns []string) *DocumentStoreBuilder {
	b.config.IncludePatterns = patterns
	return b
}

// ExcludePatterns sets the exclude patterns (replaces defaults entirely)
func (b *DocumentStoreBuilder) ExcludePatterns(patterns []string) *DocumentStoreBuilder {
	b.config.ExcludePatterns = patterns
	return b
}

// AdditionalExcludes adds additional exclude patterns (extends defaults)
func (b *DocumentStoreBuilder) AdditionalExcludes(patterns []string) *DocumentStoreBuilder {
	b.config.AdditionalExcludes = patterns
	return b
}

// WatchChanges enables file watching for directory source
func (b *DocumentStoreBuilder) WatchChanges(watch bool) *DocumentStoreBuilder {
	b.config.EnableWatchChanges = boolPtr(watch)
	return b
}

// MaxFileSize sets the maximum file size for directory source
func (b *DocumentStoreBuilder) MaxFileSize(size int64) *DocumentStoreBuilder {
	if size < 0 {
		panic("max file size must be non-negative")
	}
	b.config.MaxFileSize = size
	return b
}

// IncrementalIndexing enables incremental indexing
func (b *DocumentStoreBuilder) IncrementalIndexing(enabled bool) *DocumentStoreBuilder {
	b.config.EnableIncrementalIndexing = boolPtr(enabled)
	return b
}

// ChunkSize sets the chunk size for text splitting
func (b *DocumentStoreBuilder) ChunkSize(size int) *DocumentStoreBuilder {
	if size < 1 {
		panic("chunk size must be at least 1")
	}
	b.config.ChunkSize = size
	return b
}

// ChunkOverlap sets the chunk overlap for text splitting
func (b *DocumentStoreBuilder) ChunkOverlap(overlap int) *DocumentStoreBuilder {
	if overlap < 0 {
		panic("chunk overlap must be non-negative")
	}
	b.config.ChunkOverlap = overlap
	return b
}

// ChunkStrategy sets the chunking strategy ("simple", "overlapping", "semantic")
func (b *DocumentStoreBuilder) ChunkStrategy(strategy string) *DocumentStoreBuilder {
	b.config.ChunkStrategy = strategy
	return b
}

// ExtractMetadata enables metadata extraction
func (b *DocumentStoreBuilder) ExtractMetadata(enabled bool) *DocumentStoreBuilder {
	b.config.EnableMetadataExtraction = boolPtr(enabled)
	return b
}

// MetadataLanguages sets the languages for metadata extraction
func (b *DocumentStoreBuilder) MetadataLanguages(languages []string) *DocumentStoreBuilder {
	b.config.MetadataLanguages = languages
	return b
}

// MaxConcurrentFiles sets the maximum number of concurrent files to process
func (b *DocumentStoreBuilder) MaxConcurrentFiles(max int) *DocumentStoreBuilder {
	if max < 1 {
		panic("max concurrent files must be at least 1")
	}
	b.config.MaxConcurrentFiles = max
	return b
}

// ShowProgress enables/disables progress bar display
func (b *DocumentStoreBuilder) ShowProgress(show bool) *DocumentStoreBuilder {
	b.config.EnableProgressDisplay = boolPtr(show)
	return b
}

// VerboseProgress enables/disables verbose progress (shows current file)
func (b *DocumentStoreBuilder) VerboseProgress(verbose bool) *DocumentStoreBuilder {
	b.config.EnableVerboseProgress = boolPtr(verbose)
	return b
}

// EnableCheckpoints enables/disables checkpointing for resume capability
func (b *DocumentStoreBuilder) EnableCheckpoints(enabled bool) *DocumentStoreBuilder {
	b.config.EnableCheckpoints = boolPtr(enabled)
	return b
}

// QuietMode enables/disables quiet mode (suppresses per-file warnings)
func (b *DocumentStoreBuilder) QuietMode(quiet bool) *DocumentStoreBuilder {
	b.config.EnableQuietMode = boolPtr(quiet)
	return b
}

// WithSQLConfig sets the SQL configuration for SQL source
func (b *DocumentStoreBuilder) WithSQLConfig(sqlConfig *config.DocumentStoreSQLConfig) *DocumentStoreBuilder {
	b.config.SQL = sqlConfig
	return b
}

// WithSQLBuilder sets the SQL configuration using a builder
func (b *DocumentStoreBuilder) WithSQLBuilder(sqlBuilder *DocumentStoreSQLBuilder) *DocumentStoreBuilder {
	sqlConfig, err := sqlBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build SQL config: %v", err))
	}
	b.config.SQL = sqlConfig
	return b
}

// SQLMaxRows sets the maximum rows to index per table for SQL source
func (b *DocumentStoreBuilder) SQLMaxRows(maxRows int) *DocumentStoreBuilder {
	if maxRows < 0 {
		panic("SQL max rows must be non-negative")
	}
	b.config.SQLMaxRows = maxRows
	return b
}

// WithSQLTable adds a SQL table configuration
func (b *DocumentStoreBuilder) WithSQLTable(tableConfig *config.DocumentStoreSQLTableConfig) *DocumentStoreBuilder {
	if b.config.SQLTables == nil {
		b.config.SQLTables = make([]config.DocumentStoreSQLTableConfig, 0)
	}
	b.config.SQLTables = append(b.config.SQLTables, *tableConfig)
	return b
}

// WithSQLTableBuilder adds a SQL table configuration using a builder
func (b *DocumentStoreBuilder) WithSQLTableBuilder(tableBuilder *DocumentStoreSQLTableBuilder) *DocumentStoreBuilder {
	tableConfig, err := tableBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build SQL table config: %v", err))
	}
	return b.WithSQLTable(tableConfig)
}

// WithAPIConfig sets the API configuration for API source
func (b *DocumentStoreBuilder) WithAPIConfig(apiConfig *config.DocumentStoreAPIConfig) *DocumentStoreBuilder {
	b.config.API = apiConfig
	return b
}

// WithAPIBuilder sets the API configuration using a builder
func (b *DocumentStoreBuilder) WithAPIBuilder(apiBuilder *DocumentStoreAPIBuilder) *DocumentStoreBuilder {
	apiConfig, err := apiBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build API config: %v", err))
	}
	b.config.API = apiConfig
	return b
}

// Build creates the document store configuration
func (b *DocumentStoreBuilder) Build() (*config.DocumentStoreConfig, error) {
	// Set defaults
	b.config.SetDefaults()

	// Validate
	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("document store validation failed: %w", err)
	}

	return b.config, nil
}

// DocumentStoreSQLBuilder provides a fluent API for building SQL document store configurations
type DocumentStoreSQLBuilder struct {
	config *config.DocumentStoreSQLConfig
}

// NewDocumentStoreSQL creates a new SQL document store builder
func NewDocumentStoreSQL(database string) *DocumentStoreSQLBuilder {
	if database == "" {
		panic("database name is required")
	}
	return &DocumentStoreSQLBuilder{
		config: &config.DocumentStoreSQLConfig{
			Database: database,
		},
	}
}

// Driver sets the database driver ("postgres", "mysql", "sqlite3")
func (b *DocumentStoreSQLBuilder) Driver(driver string) *DocumentStoreSQLBuilder {
	b.config.Driver = driver
	return b
}

// Host sets the database host
func (b *DocumentStoreSQLBuilder) Host(host string) *DocumentStoreSQLBuilder {
	b.config.Host = host
	return b
}

// Port sets the database port
func (b *DocumentStoreSQLBuilder) Port(port int) *DocumentStoreSQLBuilder {
	if port < 0 || port > 65535 {
		panic("port must be between 0 and 65535")
	}
	b.config.Port = port
	return b
}

// Username sets the database username
func (b *DocumentStoreSQLBuilder) Username(username string) *DocumentStoreSQLBuilder {
	b.config.Username = username
	return b
}

// Password sets the database password
func (b *DocumentStoreSQLBuilder) Password(password string) *DocumentStoreSQLBuilder {
	b.config.Password = password
	return b
}

// SSLMode sets the SSL mode
func (b *DocumentStoreSQLBuilder) SSLMode(mode string) *DocumentStoreSQLBuilder {
	b.config.SSLMode = mode
	return b
}

// Build creates the SQL document store configuration
func (b *DocumentStoreSQLBuilder) Build() (*config.DocumentStoreSQLConfig, error) {
	if b.config.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}
	return b.config, nil
}

// DocumentStoreSQLTableBuilder provides a fluent API for building SQL table configurations
type DocumentStoreSQLTableBuilder struct {
	config *config.DocumentStoreSQLTableConfig
}

// NewDocumentStoreSQLTable creates a new SQL table builder
func NewDocumentStoreSQLTable(table string, columns []string, idColumn string) *DocumentStoreSQLTableBuilder {
	if table == "" {
		panic("table name is required")
	}
	if len(columns) == 0 {
		panic("at least one column is required")
	}
	if idColumn == "" {
		panic("ID column is required")
	}
	return &DocumentStoreSQLTableBuilder{
		config: &config.DocumentStoreSQLTableConfig{
			Table:    table,
			Columns:  columns,
			IDColumn: idColumn,
		},
	}
}

// UpdatedColumn sets the column for tracking updates
func (b *DocumentStoreSQLTableBuilder) UpdatedColumn(column string) *DocumentStoreSQLTableBuilder {
	b.config.UpdatedColumn = column
	return b
}

// WhereClause sets the WHERE clause for filtering
func (b *DocumentStoreSQLTableBuilder) WhereClause(clause string) *DocumentStoreSQLTableBuilder {
	b.config.WhereClause = clause
	return b
}

// MetadataColumns sets the columns to include as metadata
func (b *DocumentStoreSQLTableBuilder) MetadataColumns(columns []string) *DocumentStoreSQLTableBuilder {
	b.config.MetadataColumns = columns
	return b
}

// Build creates the SQL table configuration
func (b *DocumentStoreSQLTableBuilder) Build() (*config.DocumentStoreSQLTableConfig, error) {
	return b.config, nil
}

// DocumentStoreAPIBuilder provides a fluent API for building API document store configurations
type DocumentStoreAPIBuilder struct {
	config *config.DocumentStoreAPIConfig
}

// NewDocumentStoreAPI creates a new API document store builder
func NewDocumentStoreAPI(baseURL string) *DocumentStoreAPIBuilder {
	if baseURL == "" {
		panic("base URL is required")
	}
	return &DocumentStoreAPIBuilder{
		config: &config.DocumentStoreAPIConfig{
			BaseURL:   baseURL,
			Endpoints: make([]config.DocumentStoreAPIEndpointConfig, 0),
		},
	}
}

// WithAuth sets the authentication configuration
func (b *DocumentStoreAPIBuilder) WithAuth(authConfig *config.DocumentStoreAPIAuthConfig) *DocumentStoreAPIBuilder {
	b.config.Auth = authConfig
	return b
}

// WithAuthBuilder sets the authentication configuration using a builder
func (b *DocumentStoreAPIBuilder) WithAuthBuilder(authBuilder *DocumentStoreAPIAuthBuilder) *DocumentStoreAPIBuilder {
	authConfig, err := authBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build auth config: %v", err))
	}
	b.config.Auth = authConfig
	return b
}

// WithEndpoint adds an API endpoint configuration
func (b *DocumentStoreAPIBuilder) WithEndpoint(endpointConfig *config.DocumentStoreAPIEndpointConfig) *DocumentStoreAPIBuilder {
	b.config.Endpoints = append(b.config.Endpoints, *endpointConfig)
	return b
}

// WithEndpointBuilder adds an API endpoint configuration using a builder
func (b *DocumentStoreAPIBuilder) WithEndpointBuilder(endpointBuilder *DocumentStoreAPIEndpointBuilder) *DocumentStoreAPIBuilder {
	endpointConfig, err := endpointBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build endpoint config: %v", err))
	}
	return b.WithEndpoint(endpointConfig)
}

// Build creates the API document store configuration
func (b *DocumentStoreAPIBuilder) Build() (*config.DocumentStoreAPIConfig, error) {
	if b.config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if len(b.config.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one endpoint is required")
	}
	return b.config, nil
}

// DocumentStoreAPIAuthBuilder provides a fluent API for building API authentication configurations
type DocumentStoreAPIAuthBuilder struct {
	config *config.DocumentStoreAPIAuthConfig
}

// NewDocumentStoreAPIAuth creates a new API auth builder
// authType must be one of: "bearer", "basic", "apikey"
func NewDocumentStoreAPIAuth(authType string) *DocumentStoreAPIAuthBuilder {
	if authType == "" {
		panic("auth type is required")
	}
	if authType != "bearer" && authType != "basic" && authType != "apikey" {
		panic(fmt.Sprintf("invalid auth type: %s (must be 'bearer', 'basic', or 'apikey')", authType))
	}
	return &DocumentStoreAPIAuthBuilder{
		config: &config.DocumentStoreAPIAuthConfig{
			Type: authType,
		},
	}
}

// Token sets the bearer token (for bearer auth)
func (b *DocumentStoreAPIAuthBuilder) Token(token string) *DocumentStoreAPIAuthBuilder {
	b.config.Token = token
	return b
}

// Username sets the username (for basic auth)
func (b *DocumentStoreAPIAuthBuilder) Username(username string) *DocumentStoreAPIAuthBuilder {
	b.config.User = username
	return b
}

// Password sets the password (for basic auth)
func (b *DocumentStoreAPIAuthBuilder) Password(password string) *DocumentStoreAPIAuthBuilder {
	b.config.Pass = password
	return b
}

// Header sets the header name for API key auth
func (b *DocumentStoreAPIAuthBuilder) Header(header string) *DocumentStoreAPIAuthBuilder {
	b.config.Header = header
	return b
}

// Extra sets additional authentication parameters
func (b *DocumentStoreAPIAuthBuilder) Extra(extra map[string]string) *DocumentStoreAPIAuthBuilder {
	b.config.Extra = extra
	return b
}

// Build creates the API authentication configuration
func (b *DocumentStoreAPIAuthBuilder) Build() (*config.DocumentStoreAPIAuthConfig, error) {
	return b.config, nil
}

// DocumentStoreAPIEndpointBuilder provides a fluent API for building API endpoint configurations
type DocumentStoreAPIEndpointBuilder struct {
	config *config.DocumentStoreAPIEndpointConfig
}

// NewDocumentStoreAPIEndpoint creates a new API endpoint builder
func NewDocumentStoreAPIEndpoint(path string) *DocumentStoreAPIEndpointBuilder {
	if path == "" {
		panic("endpoint path is required")
	}
	return &DocumentStoreAPIEndpointBuilder{
		config: &config.DocumentStoreAPIEndpointConfig{
			Path:   path,
			Method: "GET", // Default method
		},
	}
}

// Method sets the HTTP method
func (b *DocumentStoreAPIEndpointBuilder) Method(method string) *DocumentStoreAPIEndpointBuilder {
	b.config.Method = method
	return b
}

// Params sets the query parameters
func (b *DocumentStoreAPIEndpointBuilder) Params(params map[string]string) *DocumentStoreAPIEndpointBuilder {
	b.config.Params = params
	return b
}

// Headers sets the HTTP headers
func (b *DocumentStoreAPIEndpointBuilder) Headers(headers map[string]string) *DocumentStoreAPIEndpointBuilder {
	b.config.Headers = headers
	return b
}

// Body sets the request body
func (b *DocumentStoreAPIEndpointBuilder) Body(body string) *DocumentStoreAPIEndpointBuilder {
	b.config.Body = body
	return b
}

// WithAuth sets the endpoint-specific authentication
func (b *DocumentStoreAPIEndpointBuilder) WithAuth(authConfig *config.DocumentStoreAPIAuthConfig) *DocumentStoreAPIEndpointBuilder {
	b.config.Auth = authConfig
	return b
}

// WithAuthBuilder sets the endpoint-specific authentication using a builder
func (b *DocumentStoreAPIEndpointBuilder) WithAuthBuilder(authBuilder *DocumentStoreAPIAuthBuilder) *DocumentStoreAPIEndpointBuilder {
	authConfig, err := authBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build auth config: %v", err))
	}
	b.config.Auth = authConfig
	return b
}

// IDField sets the JSON field containing the document ID
func (b *DocumentStoreAPIEndpointBuilder) IDField(field string) *DocumentStoreAPIEndpointBuilder {
	b.config.IDField = field
	return b
}

// ContentField sets the JSON field(s) containing the document content
func (b *DocumentStoreAPIEndpointBuilder) ContentField(field string) *DocumentStoreAPIEndpointBuilder {
	b.config.ContentField = field
	return b
}

// MetadataFields sets the JSON fields to include as metadata
func (b *DocumentStoreAPIEndpointBuilder) MetadataFields(fields []string) *DocumentStoreAPIEndpointBuilder {
	b.config.MetadataFields = fields
	return b
}

// UpdatedField sets the JSON field containing the update timestamp
func (b *DocumentStoreAPIEndpointBuilder) UpdatedField(field string) *DocumentStoreAPIEndpointBuilder {
	b.config.UpdatedField = field
	return b
}

// WithPagination sets the pagination configuration
func (b *DocumentStoreAPIEndpointBuilder) WithPagination(paginationConfig *config.DocumentStoreAPIPaginationConfig) *DocumentStoreAPIEndpointBuilder {
	b.config.Pagination = paginationConfig
	return b
}

// WithPaginationBuilder sets the pagination configuration using a builder
func (b *DocumentStoreAPIEndpointBuilder) WithPaginationBuilder(paginationBuilder *DocumentStoreAPIPaginationBuilder) *DocumentStoreAPIEndpointBuilder {
	paginationConfig, err := paginationBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build pagination config: %v", err))
	}
	b.config.Pagination = paginationConfig
	return b
}

// Build creates the API endpoint configuration
func (b *DocumentStoreAPIEndpointBuilder) Build() (*config.DocumentStoreAPIEndpointConfig, error) {
	if b.config.Path == "" {
		return nil, fmt.Errorf("endpoint path is required")
	}
	return b.config, nil
}

// DocumentStoreAPIPaginationBuilder provides a fluent API for building pagination configurations
type DocumentStoreAPIPaginationBuilder struct {
	config *config.DocumentStoreAPIPaginationConfig
}

// NewDocumentStoreAPIPagination creates a new pagination builder
// paginationType must be one of: "offset", "cursor", "page", "link"
func NewDocumentStoreAPIPagination(paginationType string) *DocumentStoreAPIPaginationBuilder {
	if paginationType == "" {
		panic("pagination type is required")
	}
	return &DocumentStoreAPIPaginationBuilder{
		config: &config.DocumentStoreAPIPaginationConfig{
			Type: paginationType,
		},
	}
}

// PageParam sets the query parameter name for page/offset
func (b *DocumentStoreAPIPaginationBuilder) PageParam(param string) *DocumentStoreAPIPaginationBuilder {
	b.config.PageParam = param
	return b
}

// SizeParam sets the query parameter name for page size
func (b *DocumentStoreAPIPaginationBuilder) SizeParam(param string) *DocumentStoreAPIPaginationBuilder {
	b.config.SizeParam = param
	return b
}

// MaxPages sets the maximum number of pages to fetch (0 = unlimited)
func (b *DocumentStoreAPIPaginationBuilder) MaxPages(maxPages int) *DocumentStoreAPIPaginationBuilder {
	if maxPages < 0 {
		panic("max pages must be non-negative")
	}
	b.config.MaxPages = maxPages
	return b
}

// PageSize sets the number of items per page
func (b *DocumentStoreAPIPaginationBuilder) PageSize(size int) *DocumentStoreAPIPaginationBuilder {
	if size < 1 {
		panic("page size must be at least 1")
	}
	b.config.PageSize = size
	return b
}

// NextField sets the JSON field containing the next page URL/cursor
func (b *DocumentStoreAPIPaginationBuilder) NextField(field string) *DocumentStoreAPIPaginationBuilder {
	b.config.NextField = field
	return b
}

// DataField sets the JSON field containing the array of items (if nested)
func (b *DocumentStoreAPIPaginationBuilder) DataField(field string) *DocumentStoreAPIPaginationBuilder {
	b.config.DataField = field
	return b
}

// Build creates the pagination configuration
func (b *DocumentStoreAPIPaginationBuilder) Build() (*config.DocumentStoreAPIPaginationConfig, error) {
	return b.config, nil
}
