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

import "fmt"

// VectorStoreConfig configures a vector database provider.
//
// Example YAML:
//
//	vector_stores:
//	  local:
//	    type: chromem
//	    persist_path: .hector/vectors
//	  production:
//	    type: qdrant
//	    host: qdrant.example.com
//	    port: 6333
//	    api_key: ${QDRANT_API_KEY}
type VectorStoreConfig struct {
	// Type is the vector store type: "chromem", "qdrant", "pinecone", "weaviate", "milvus".
	Type string `yaml:"type"`

	// Host for external vector stores (qdrant, weaviate, milvus).
	Host string `yaml:"host,omitempty"`

	// Port for external vector stores.
	Port int `yaml:"port,omitempty"`

	// APIKey for authenticated access.
	APIKey string `yaml:"api_key,omitempty"`

	// EnableTLS enables TLS connections.
	EnableTLS *bool `yaml:"enable_tls,omitempty"`

	// PersistPath for chromem file persistence.
	PersistPath string `yaml:"persist_path,omitempty"`

	// Compress enables gzip compression for chromem persistence.
	Compress bool `yaml:"compress,omitempty"`

	// Collection is the default collection name (optional).
	Collection string `yaml:"collection,omitempty"`

	// IndexName for Pinecone.
	IndexName string `yaml:"index_name,omitempty"`

	// Environment for Pinecone.
	Environment string `yaml:"environment,omitempty"`
}

// SetDefaults applies default values.
func (c *VectorStoreConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "chromem" // Default to embedded
	}
	if c.Port == 0 {
		switch c.Type {
		case "qdrant":
			c.Port = 6333
		case "weaviate":
			c.Port = 8080
		case "milvus":
			c.Port = 19530
		}
	}
}

// Validate checks the configuration for errors.
func (c *VectorStoreConfig) Validate() error {
	validTypes := map[string]bool{
		"chromem":  true,
		"qdrant":   true,
		"pinecone": true,
		"weaviate": true,
		"milvus":   true,
		"chroma":   true,
	}

	if !validTypes[c.Type] {
		return fmt.Errorf("invalid vector store type %q (valid: chromem, qdrant, pinecone, weaviate, milvus, chroma)", c.Type)
	}

	// External stores require host
	externalStores := map[string]bool{
		"qdrant":   true,
		"weaviate": true,
		"milvus":   true,
	}
	if externalStores[c.Type] && c.Host == "" {
		return fmt.Errorf("host is required for %s vector store", c.Type)
	}

	// Pinecone requires API key
	if c.Type == "pinecone" && c.APIKey == "" {
		return fmt.Errorf("api_key is required for pinecone vector store")
	}

	return nil
}

// IsEmbedded returns true for embedded vector stores (chromem).
func (c *VectorStoreConfig) IsEmbedded() bool {
	return c.Type == "chromem"
}

// DocumentStoreConfig configures a document store for RAG.
//
// Example YAML:
//
//	document_stores:
//	  codebase:
//	    source:
//	      type: directory
//	      path: ./src
//	      include: ["*.go", "*.ts"]
//	    chunking:
//	      strategy: semantic
//	      size: 1000
//	    vector_store: local
//	    embedder: default
//	    watch: true
//	    indexing:
//	      max_concurrent: 8
//	      retry:
//	        max_retries: 3
//	        base_delay: 1s
type DocumentStoreConfig struct {
	// Source configures where documents come from.
	Source *DocumentSourceConfig `yaml:"source"`

	// Chunking configures how documents are split.
	Chunking *ChunkingConfig `yaml:"chunking,omitempty"`

	// VectorStore references a vector store from vector_stores.
	VectorStore string `yaml:"vector_store,omitempty"`

	// Embedder references an embedder from embedders.
	Embedder string `yaml:"embedder,omitempty"`

	// Collection overrides the collection name.
	Collection string `yaml:"collection,omitempty"`

	// Watch enables file watching for automatic re-indexing.
	Watch bool `yaml:"watch,omitempty"`

	// IncrementalIndexing only re-indexes changed documents.
	IncrementalIndexing bool `yaml:"incremental_indexing,omitempty"`

	// Search configures search behavior for this store.
	Search *DocumentSearchConfig `yaml:"search,omitempty"`

	// Indexing configures indexing behavior (concurrency, retry).
	Indexing *IndexingConfig `yaml:"indexing,omitempty"`

	// MCPParsers configures MCP-based document parsing (e.g., Docling).
	// When configured, MCP tools are used to parse documents instead of native parsers.
	MCPParsers *MCPParserConfig `yaml:"mcp_parsers,omitempty"`
}

// SetDefaults applies default values.
func (c *DocumentStoreConfig) SetDefaults() {
	if c.Source != nil {
		c.Source.SetDefaults()
	}
	if c.Chunking == nil {
		c.Chunking = &ChunkingConfig{}
	}
	c.Chunking.SetDefaults()
	if c.Search == nil {
		c.Search = &DocumentSearchConfig{}
	}
	c.Search.SetDefaults()
	if c.Indexing == nil {
		c.Indexing = &IndexingConfig{}
	}
	c.Indexing.SetDefaults()
	if c.MCPParsers != nil {
		c.MCPParsers.SetDefaults()
	}
}

// Validate checks the configuration for errors.
func (c *DocumentStoreConfig) Validate() error {
	if c.Source == nil {
		return fmt.Errorf("source is required")
	}
	if err := c.Source.Validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if c.Chunking != nil {
		if err := c.Chunking.Validate(); err != nil {
			return fmt.Errorf("chunking: %w", err)
		}
	}
	if c.Search != nil {
		if err := c.Search.Validate(); err != nil {
			return fmt.Errorf("search: %w", err)
		}
	}
	if c.Indexing != nil {
		if err := c.Indexing.Validate(); err != nil {
			return fmt.Errorf("indexing: %w", err)
		}
	}
	if c.MCPParsers != nil {
		if err := c.MCPParsers.Validate(); err != nil {
			return fmt.Errorf("mcp_parsers: %w", err)
		}
	}
	return nil
}

// DocumentSourceConfig configures a document source.
type DocumentSourceConfig struct {
	// Type is the source type: "directory", "sql", "api", "collection".
	Type string `yaml:"type"`

	// Path is the directory path (for directory sources).
	Path string `yaml:"path,omitempty"`

	// Include patterns for files (for directory sources).
	Include []string `yaml:"include,omitempty"`

	// Exclude patterns for files (for directory sources).
	Exclude []string `yaml:"exclude,omitempty"`

	// MaxFileSize limits file size in bytes (for directory sources).
	MaxFileSize int64 `yaml:"max_file_size,omitempty"`

	// SQL configuration (for sql sources).
	SQL *SQLSourceConfig `yaml:"sql,omitempty"`

	// API configuration (for api sources).
	API *APISourceConfig `yaml:"api,omitempty"`

	// Collection name (for collection sources - references existing pre-populated collection).
	Collection string `yaml:"collection,omitempty"`
}

// SetDefaults applies default values.
func (c *DocumentSourceConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "directory"
	}
	if c.MaxFileSize <= 0 {
		c.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if c.Exclude == nil {
		c.Exclude = []string{".*", "node_modules", "__pycache__", "vendor", ".git"}
	}
}

// Validate checks the configuration for errors.
func (c *DocumentSourceConfig) Validate() error {
	validTypes := map[string]bool{
		"directory":  true,
		"sql":        true,
		"api":        true,
		"collection": true,
	}
	if !validTypes[c.Type] {
		return fmt.Errorf("invalid source type %q (valid: directory, sql, api, collection)", c.Type)
	}

	switch c.Type {
	case "directory":
		if c.Path == "" {
			return fmt.Errorf("path is required for directory source")
		}
	case "sql":
		if c.SQL == nil {
			return fmt.Errorf("sql config is required for sql source")
		}
		if err := c.SQL.Validate(); err != nil {
			return fmt.Errorf("sql: %w", err)
		}
	case "api":
		if c.API == nil {
			return fmt.Errorf("api config is required for api source")
		}
		if err := c.API.Validate(); err != nil {
			return fmt.Errorf("api: %w", err)
		}
	case "collection":
		if c.Collection == "" {
			return fmt.Errorf("collection name is required for collection source")
		}
	}
	return nil
}

// SQLSourceConfig configures a SQL-based document source.
type SQLSourceConfig struct {
	// Database references a database from the databases config.
	Database string `yaml:"database"`

	// Tables defines which tables to index.
	Tables []SQLTableConfig `yaml:"tables"`
}

// Validate checks the configuration for errors.
func (c *SQLSourceConfig) Validate() error {
	if c.Database == "" {
		return fmt.Errorf("database reference is required")
	}
	if len(c.Tables) == 0 {
		return fmt.Errorf("at least one table is required")
	}
	for i, table := range c.Tables {
		if err := table.Validate(); err != nil {
			return fmt.Errorf("table[%d]: %w", i, err)
		}
	}
	return nil
}

// SQLTableConfig defines which table and columns to index.
type SQLTableConfig struct {
	// Table is the table name.
	Table string `yaml:"table"`

	// Columns to concatenate for content.
	Columns []string `yaml:"columns"`

	// IDColumn is the primary key column.
	IDColumn string `yaml:"id_column"`

	// UpdatedColumn tracks document changes (e.g., updated_at).
	UpdatedColumn string `yaml:"updated_column,omitempty"`

	// WhereClause filters rows.
	WhereClause string `yaml:"where_clause,omitempty"`

	// MetadataColumns to include as metadata.
	MetadataColumns []string `yaml:"metadata_columns,omitempty"`
}

// Validate checks the configuration for errors.
func (c *SQLTableConfig) Validate() error {
	if c.Table == "" {
		return fmt.Errorf("table is required")
	}
	if len(c.Columns) == 0 {
		return fmt.Errorf("at least one column is required")
	}
	if c.IDColumn == "" {
		return fmt.Errorf("id_column is required")
	}
	return nil
}

// APISourceConfig configures an API-based document source.
type APISourceConfig struct {
	// URL is the API endpoint.
	URL string `yaml:"url"`

	// Headers are HTTP headers to include.
	Headers map[string]string `yaml:"headers,omitempty"`

	// IDField is the JSON path to document IDs.
	IDField string `yaml:"id_field"`

	// ContentField is the JSON path to document content.
	ContentField string `yaml:"content_field"`
}

// Validate checks the configuration for errors.
func (c *APISourceConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

// ChunkingConfig configures document chunking.
type ChunkingConfig struct {
	// Strategy is the chunking strategy: "simple", "overlapping", "semantic".
	Strategy string `yaml:"strategy,omitempty"`

	// Size is the target chunk size in characters.
	Size int `yaml:"size,omitempty"`

	// Overlap is the overlap size (for overlapping strategy).
	Overlap int `yaml:"overlap,omitempty"`

	// MinSize is the minimum chunk size.
	MinSize int `yaml:"min_size,omitempty"`

	// MaxSize is the maximum chunk size.
	MaxSize int `yaml:"max_size,omitempty"`

	// PreserveWords avoids splitting mid-word.
	PreserveWords *bool `yaml:"preserve_words,omitempty"`
}

// SetDefaults applies default values.
func (c *ChunkingConfig) SetDefaults() {
	if c.Strategy == "" {
		c.Strategy = "simple"
	}
	if c.Size <= 0 {
		c.Size = 1000
	}
	if c.Overlap < 0 {
		c.Overlap = 0
	}
	if c.MinSize <= 0 {
		c.MinSize = 100
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 2000
	}
	if c.PreserveWords == nil {
		c.PreserveWords = BoolPtr(true)
	}
}

// Validate checks the configuration for errors.
func (c *ChunkingConfig) Validate() error {
	validStrategies := map[string]bool{
		"simple":      true,
		"overlapping": true,
		"semantic":    true,
	}
	if !validStrategies[c.Strategy] {
		return fmt.Errorf("invalid chunking strategy %q (valid: simple, overlapping, semantic)", c.Strategy)
	}
	if c.Size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	if c.Overlap < 0 {
		return fmt.Errorf("overlap must be non-negative")
	}
	if c.Overlap >= c.Size {
		return fmt.Errorf("overlap must be less than size")
	}
	return nil
}

// DocumentSearchConfig configures search behavior for a document store.
type DocumentSearchConfig struct {
	// TopK is the default number of results.
	TopK int `yaml:"top_k,omitempty"`

	// Threshold filters results below this score.
	Threshold float32 `yaml:"threshold,omitempty"`

	// EnableHyDE enables hypothetical document embeddings.
	EnableHyDE bool `yaml:"enable_hyde,omitempty"`

	// HyDELLM references an LLM for HyDE generation.
	HyDELLM string `yaml:"hyde_llm,omitempty"`

	// EnableRerank enables LLM-based reranking.
	EnableRerank bool `yaml:"enable_rerank,omitempty"`

	// RerankLLM references an LLM for reranking.
	RerankLLM string `yaml:"rerank_llm,omitempty"`

	// RerankMaxResults limits reranking candidates.
	RerankMaxResults int `yaml:"rerank_max_results,omitempty"`

	// EnableMultiQuery enables query expansion.
	EnableMultiQuery bool `yaml:"enable_multi_query,omitempty"`

	// MultiQueryLLM references an LLM for query expansion.
	MultiQueryLLM string `yaml:"multi_query_llm,omitempty"`

	// MultiQueryCount is the number of query variants.
	MultiQueryCount int `yaml:"multi_query_count,omitempty"`
}

// SetDefaults applies default values.
func (c *DocumentSearchConfig) SetDefaults() {
	if c.TopK <= 0 {
		c.TopK = 10
	}
	if c.RerankMaxResults <= 0 {
		c.RerankMaxResults = 20
	}
	if c.MultiQueryCount <= 0 {
		c.MultiQueryCount = 3
	}
}

// Validate checks the configuration for errors.
func (c *DocumentSearchConfig) Validate() error {
	if c.TopK < 0 {
		return fmt.Errorf("top_k must be non-negative")
	}
	if c.Threshold < 0 || c.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	if c.EnableHyDE && c.HyDELLM == "" {
		return fmt.Errorf("hyde_llm is required when enable_hyde is true")
	}
	if c.EnableRerank && c.RerankLLM == "" {
		return fmt.Errorf("rerank_llm is required when enable_rerank is true")
	}
	if c.EnableMultiQuery && c.MultiQueryLLM == "" {
		return fmt.Errorf("multi_query_llm is required when enable_multi_query is true")
	}
	return nil
}

// IndexingConfig configures document indexing behavior.
//
// Example YAML:
//
//	indexing:
//	  max_concurrent: 8      # Number of parallel workers
//	  retry:
//	    max_retries: 5       # Retry failed operations
//	    base_delay: 2s       # Initial delay between retries
//	    max_delay: 60s       # Maximum delay between retries
type IndexingConfig struct {
	// MaxConcurrent limits parallel document processing.
	// Default: runtime.NumCPU() (typically 4-16 workers)
	// Set to 1 for sequential indexing.
	//
	// Guidelines:
	//   - CPU-bound (local embedding): use NumCPU
	//   - IO-bound (API embedding): use 2x-4x NumCPU
	//   - Rate-limited APIs: use lower values (2-4)
	MaxConcurrent int `yaml:"max_concurrent,omitempty"`

	// Retry configures retry behavior for transient failures.
	Retry *RetryConfig `yaml:"retry,omitempty"`
}

// SetDefaults applies default values.
func (c *IndexingConfig) SetDefaults() {
	// MaxConcurrent defaults are handled in rag package (runtime.NumCPU)
	if c.Retry == nil {
		c.Retry = &RetryConfig{}
	}
	c.Retry.SetDefaults()
}

// Validate checks the configuration for errors.
func (c *IndexingConfig) Validate() error {
	if c.MaxConcurrent < 0 {
		return fmt.Errorf("max_concurrent must be non-negative")
	}
	if c.Retry != nil {
		if err := c.Retry.Validate(); err != nil {
			return fmt.Errorf("retry: %w", err)
		}
	}
	return nil
}

// RetryConfig configures retry behavior for transient failures.
//
// Uses exponential backoff with jitter (pattern from v2/httpclient).
//
// Example YAML:
//
//	retry:
//	  max_retries: 3      # Number of retry attempts
//	  base_delay: 1s      # Initial delay (doubles each retry)
//	  max_delay: 30s      # Maximum delay between retries
//	  jitter: 0.1         # Randomness factor (0.0-1.0)
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries int `yaml:"max_retries,omitempty"`

	// BaseDelay is the initial delay between retries.
	// Each subsequent retry doubles this value.
	// Default: 1s
	BaseDelay Duration `yaml:"base_delay,omitempty"`

	// MaxDelay is the maximum delay between retries.
	// Default: 30s
	MaxDelay Duration `yaml:"max_delay,omitempty"`

	// Jitter adds randomness to delays to prevent thundering herd.
	// Value between 0.0 and 1.0.
	// Default: 0.1 (±10% variation)
	Jitter float64 `yaml:"jitter,omitempty"`
}

// SetDefaults applies default values.
func (c *RetryConfig) SetDefaults() {
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.BaseDelay <= 0 {
		c.BaseDelay = Duration(1000000000) // 1s in nanoseconds
	}
	if c.MaxDelay <= 0 {
		c.MaxDelay = Duration(30000000000) // 30s in nanoseconds
	}
	if c.Jitter <= 0 {
		c.Jitter = 0.1
	}
}

// Validate checks the configuration for errors.
func (c *RetryConfig) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if c.BaseDelay < 0 {
		return fmt.Errorf("base_delay must be non-negative")
	}
	if c.MaxDelay < 0 {
		return fmt.Errorf("max_delay must be non-negative")
	}
	if c.Jitter < 0 || c.Jitter > 1 {
		return fmt.Errorf("jitter must be between 0 and 1")
	}
	return nil
}

// MCPParserConfig configures MCP-based document parsing.
//
// MCP (Model Context Protocol) tools like Docling can be used for advanced
// document parsing with better quality than native parsers.
//
// Example YAML:
//
//	mcp_parsers:
//	  tool_names: ["convert_document_into_docling_document"]
//	  extensions: [".pdf", ".docx", ".pptx"]
//	  priority: 8
//	  path_prefix: "/docs"  # For containerized MCP services
type MCPParserConfig struct {
	// ToolNames are the MCP tool names to try for parsing (in order).
	// Example: ["parse_document", "docling_parse", "convert_document"]
	ToolNames []string `yaml:"tool_names"`

	// Extensions limits which file types use MCP parsing.
	// Empty means all binary files.
	// Example: [".pdf", ".docx", ".pptx", ".xlsx"]
	Extensions []string `yaml:"extensions,omitempty"`

	// Priority sets the extractor priority (higher = preferred).
	// Default: 8 (higher than native parsers at 5)
	Priority *int `yaml:"priority,omitempty"`

	// PreferNative uses MCP only when native parsers fail.
	// Default: false
	PreferNative *bool `yaml:"prefer_native,omitempty"`

	// PathPrefix remaps local paths for containerized MCP services.
	// Example: "/docs" when mounting ./my-docs:/docs in Docker
	PathPrefix string `yaml:"path_prefix,omitempty"`
}

// SetDefaults applies default values.
func (c *MCPParserConfig) SetDefaults() {
	if c.Priority == nil {
		priority := 8 // Higher than native (5)
		c.Priority = &priority
	}
	if c.PreferNative == nil {
		c.PreferNative = BoolPtr(false)
	}
	if c.Extensions == nil {
		// Default to common binary document formats
		c.Extensions = []string{".pdf", ".docx", ".pptx", ".xlsx"}
	}
}

// Validate checks the configuration for errors.
func (c *MCPParserConfig) Validate() error {
	if len(c.ToolNames) == 0 {
		return fmt.Errorf("tool_names is required")
	}
	return nil
}
