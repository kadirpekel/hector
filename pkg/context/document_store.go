package context

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context/chunking"
	"github.com/kadirpekel/hector/pkg/context/extraction"
	"github.com/kadirpekel/hector/pkg/context/indexing"
	"github.com/kadirpekel/hector/pkg/context/metadata"
	"github.com/kadirpekel/hector/pkg/databases"
)

const (
	DefaultMaxFileSize = 5 * 1024 * 1024

	DefaultUpdateChannelSize = 100

	DefaultFileWatchTimeout = 10 * time.Second

	MaxConcurrentIndexing = 10 // Increased from 3 to 10 for better throughput
)

const (
	DocumentTypeCode     = "code"
	DocumentTypeConfig   = "config"
	DocumentTypeMarkdown = "markdown"
	DocumentTypeText     = "text"
	DocumentTypeScript   = "script"
	DocumentTypeBinary   = "binary"
	DocumentTypeUnknown  = "unknown"
)

const (
	OperationCreate = "create"
	OperationModify = "modify"
	OperationDelete = "delete"
)

const (
	DefaultIndexingTimeout = 120 * time.Second
)

type DocumentStoreError struct {
	StoreName string
	Operation string
	Message   string
	FilePath  string
	Err       error
	Timestamp time.Time
}

func (e *DocumentStoreError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s (file: %s): %v", e.StoreName, e.Operation, e.Message, e.FilePath, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s (file: %s)", e.StoreName, e.Operation, e.Message, e.FilePath)
}

func (e *DocumentStoreError) Unwrap() error {
	return e.Err
}

func NewDocumentStoreError(storeName, operation, message, filePath string, err error) *DocumentStoreError {
	return &DocumentStoreError{
		StoreName: storeName,
		Operation: operation,
		Message:   message,
		FilePath:  filePath,
		Err:       err,
		Timestamp: time.Now(),
	}
}

type Document struct {
	ID           string            `json:"id"`
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Content      string            `json:"content"`
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Language     string            `json:"language"`
	Size         int64             `json:"size"`
	Lines        int               `json:"lines"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     map[string]string `json:"metadata"`

	Functions []string `json:"functions,omitempty"`
	Structs   []string `json:"structs,omitempty"`
	Imports   []string `json:"imports,omitempty"`
}

type DocumentUpdate struct {
	FilePath  string    `json:"file_path"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}

type DocumentStoreStatus struct {
	Name          string    `json:"name"`
	SourcePath    string    `json:"source_path"`
	Storage       string    `json:"storage"`
	LastIndexed   time.Time `json:"last_indexed"`
	IsIndexing    bool      `json:"is_indexing"`
	IsWatching    bool      `json:"is_watching"`
	DocumentCount int       `json:"document_count"`
}

type DocumentStore struct {
	mu     sync.RWMutex
	name   string
	config *config.DocumentStoreConfig
	status *DocumentStoreStatus

	searchEngine *SearchEngine
	sourcePath   string // For directory sources, this is the base path

	// Data source architecture
	dataSource         indexing.DataSource
	contentExtractors  *extraction.ExtractorRegistry
	metadataExtractors *metadata.ExtractorRegistry
	chunker            chunking.Chunker

	// Progress tracking and checkpoints
	progressTracker   *ProgressTracker
	checkpointManager *CheckpointManager

	watcher       *fsnotify.Watcher
	updateChannel chan DocumentUpdate
	ctx           context.Context
	cancel        context.CancelFunc

	indexingSemaphore chan struct{}
}

// NewDocumentStore creates a new document store.
// storeName is the name from the config map key (used as collection name unless overridden by Collection).
func NewDocumentStore(storeName string, storeConfig *config.DocumentStoreConfig, searchEngine *SearchEngine) (*DocumentStore, error) {
	if storeName == "" {
		return nil, NewDocumentStoreError("", "NewDocumentStore", "store name is required", "", nil)
	}
	if storeConfig == nil {
		return nil, NewDocumentStoreError(storeName, "NewDocumentStore", "store config is required", "", nil)
	}
	if searchEngine == nil {
		return nil, NewDocumentStoreError(storeName, "NewDocumentStore", "search engine is required", "", nil)
	}

	// Set defaults for config
	storeConfig.SetDefaults()

	ctx, cancel := context.WithCancel(context.Background())

	// Create data source from configuration
	factory := indexing.NewDataSourceFactory()
	dataSource, err := factory.CreateDataSource(storeConfig)
	if err != nil {
		cancel()
		return nil, NewDocumentStoreError(storeName, "NewDocumentStore", "failed to create data source", "", err)
	}

	// Determine source path for status/checkpoints
	sourcePath := storeConfig.Path
	if sourcePath == "" {
		sourcePath = dataSource.Type() // Fallback for non-directory sources
	}

	// Initialize content extractors
	contentExtractors := extraction.NewExtractorRegistry()
	contentExtractors.Register(extraction.NewTextExtractor())

	nativeParsers := NewNativeParserRegistry()
	nativeParserAdapter := newNativeParserAdapter(nativeParsers)
	contentExtractors.Register(extraction.NewBinaryExtractor(nativeParserAdapter))
	// Plugin extractors can be added here if needed

	// Initialize metadata extractors
	metadataExtractors := metadata.NewExtractorRegistry()
	if storeConfig.EnableMetadataExtraction != nil && *storeConfig.EnableMetadataExtraction {
		for _, lang := range storeConfig.MetadataLanguages {
			if lang == "go" {
				metadataExtractors.Register(metadata.NewGoExtractor())
			}
			// More languages can be added via plugins
		}
	}

	// Initialize chunker
	chunkerConfig := chunking.ChunkerConfig{
		Strategy: storeConfig.ChunkStrategy,
		Size:     storeConfig.ChunkSize,
		Overlap:  storeConfig.ChunkOverlap,
	}
	chunker, err := chunking.NewChunker(chunkerConfig)
	if err != nil {
		cancel()
		return nil, NewDocumentStoreError(storeName, "NewDocumentStore", "failed to create chunker", "", err)
	}

	// Determine concurrency limit
	maxConcurrent := storeConfig.MaxConcurrentFiles
	if maxConcurrent == 0 {
		maxConcurrent = 10
	}

	// Initialize progress tracker using pointer values (defaults already set in SetDefaults)
	showProgress := storeConfig.EnableProgressDisplay != nil && *storeConfig.EnableProgressDisplay
	verboseProgress := storeConfig.EnableVerboseProgress != nil && *storeConfig.EnableVerboseProgress
	progressTracker := NewProgressTracker(showProgress, verboseProgress)

	// Initialize checkpoint manager using pointer value (defaults already set in SetDefaults)
	enableCheckpoints := storeConfig.EnableCheckpoints != nil && *storeConfig.EnableCheckpoints
	checkpointManager := NewCheckpointManager(storeName, sourcePath, enableCheckpoints)

	store := &DocumentStore{
		name:               storeName,
		config:             storeConfig,
		searchEngine:       searchEngine,
		sourcePath:         sourcePath,
		dataSource:         dataSource,
		contentExtractors:  contentExtractors,
		metadataExtractors: metadataExtractors,
		chunker:            chunker,
		progressTracker:    progressTracker,
		checkpointManager:  checkpointManager,
		updateChannel:      make(chan DocumentUpdate, DefaultUpdateChannelSize),
		ctx:                ctx,
		cancel:             cancel,
		indexingSemaphore:  make(chan struct{}, maxConcurrent),
		status: &DocumentStoreStatus{
			Name:        storeName,
			SourcePath:  sourcePath,
			Storage:     "vector_database",
			LastIndexed: time.Time{},
			IsIndexing:  false,
			IsWatching:  false,
		},
	}

	// File watching only supported for directory sources
	if storeConfig.EnableWatchChanges != nil && *storeConfig.EnableWatchChanges && dataSource.Type() == "directory" {
		if err := store.initializeWatcher(); err != nil {
			cancel()
			return nil, NewDocumentStoreError(storeName, "NewDocumentStore", "failed to initialize watcher", "", err)
		}
	}

	return store, nil
}

func (ds *DocumentStore) StartIndexing() error {
	// Stores with collection override pointing to existing collection don't need indexing
	// (if source is not set, it means it's a read-only reference to existing collection)
	if ds.config.Collection != "" && ds.config.Source == "" {
		ds.mu.Lock()
		ds.status.IsIndexing = false
		ds.status.LastIndexed = time.Now()
		ds.mu.Unlock()
		collectionName := ds.getCollectionName()
		fmt.Printf("Collection-only store '%s' (collection: %s) - skipping indexing\n", ds.name, collectionName)
		return nil
	}

	ds.mu.Lock()
	if ds.status.IsIndexing {
		ds.mu.Unlock()
		return NewDocumentStoreError(ds.name, "StartIndexing", "indexing already in progress", "", nil)
	}
	ds.status.IsIndexing = true
	ds.mu.Unlock()

	defer func() {
		ds.mu.Lock()
		ds.status.IsIndexing = false
		ds.status.LastIndexed = time.Now()
		ds.mu.Unlock()
	}()

	fmt.Printf("Indexing document store '%s' from: %s (source: %s)\n", ds.name, ds.sourcePath, ds.dataSource.Type())

	// Validate that config source matches dataSource type
	if ds.config.Source != "" && ds.config.Source != ds.dataSource.Type() {
		return NewDocumentStoreError(ds.name, "StartIndexing",
			fmt.Sprintf("source type mismatch: config has '%s' but dataSource is '%s'", ds.config.Source, ds.dataSource.Type()),
			"", nil)
	}

	return ds.indexFromDataSource()
}

// indexDocument indexes a single document from any source
func (ds *DocumentStore) indexDocument(doc *indexing.Document) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultIndexingTimeout)
	defer cancel()

	// Determine source path for metadata
	relPath := doc.SourcePath
	if relPath == "" {
		if pathVal, ok := doc.Metadata["rel_path"].(string); ok {
			relPath = pathVal
		} else if pathVal, ok := doc.Metadata["path"].(string); ok {
			relPath = pathVal
		} else {
			relPath = doc.ID
		}
	}

	// Step 1: Extract/process content
	// For directory sources: may need binary extraction for PDFs, etc.
	// For SQL/API sources: content is already in Document.Content
	content := doc.Content
	title := ""
	author := ""
	extractedMetadata := make(map[string]string)

	if ds.dataSource.Type() == "directory" {
		// For file sources, check if we need binary extraction
		path := doc.ID
		mimeType := ds.detectMIMEType(path)

		// Try to extract using content extractors (for binary files like PDFs)
		extracted, err := ds.contentExtractors.ExtractContent(ctx, path, mimeType, doc.Size)
		if err == nil && extracted != nil && extracted.Content != "" {
			// Binary extraction succeeded, use extracted content
			content = extracted.Content
			title = extracted.Title
			author = extracted.Author
			extractedMetadata = extracted.Metadata
		}
		// If extraction failed or returned empty, use Document.Content (already read from file)
	}

	if content == "" {
		return nil // Skip empty content
	}

	// Step 2: Detect type and language
	docType, language := ds.detectDocumentType(doc)

	// Step 3: Extract metadata using metadata extractors (for code files)
	var meta *metadata.Metadata
	if ds.config.EnableMetadataExtraction != nil && *ds.config.EnableMetadataExtraction {
		var err error
		meta, err = ds.metadataExtractors.ExtractMetadata(language, content, doc.ID)
		if err != nil {
			// Non-fatal: continue without metadata
			meta = &metadata.Metadata{}
		}
	} else {
		meta = &metadata.Metadata{}
	}

	// Step 4: Chunk content using the chunker
	chunks, err := ds.chunker.Chunk(content, meta)
	if err != nil {
		return NewDocumentStoreError(ds.name, "indexDocument", "failed to chunk content", doc.ID, err)
	}

	// Step 5: Prepare base metadata for all chunks
	baseMetadata := make(map[string]interface{})

	// Copy all metadata from document, normalizing time.Time values
	for k, v := range doc.Metadata {
		baseMetadata[k] = normalizeMetadataValue(v)
	}

	// Add standard fields
	baseMetadata["path"] = relPath
	baseMetadata["source_path"] = relPath
	if nameVal, ok := doc.Metadata["name"].(string); ok {
		baseMetadata["name"] = nameVal
	} else {
		baseMetadata["name"] = doc.ID
	}

	if title != "" {
		baseMetadata["title"] = title
	} else if titleVal, ok := doc.Metadata["title"].(string); ok {
		baseMetadata["title"] = titleVal
	}

	baseMetadata["type"] = docType
	baseMetadata["language"] = language
	baseMetadata["size"] = doc.Size
	baseMetadata["last_modified"] = doc.LastModified.Unix()
	baseMetadata["store_name"] = ds.name
	// Set collection name in metadata so IngestDocument knows which collection to use
	// This uses the collection override if set, otherwise the store name
	baseMetadata["collection"] = ds.getCollectionName()
	baseMetadata["source_type"] = ds.dataSource.Type()
	baseMetadata["indexed_at"] = time.Now().Unix()

	// Add extracted metadata
	if author != "" {
		baseMetadata["author"] = author
	}
	for k, v := range extractedMetadata {
		baseMetadata["meta_"+k] = v
	}

	// Add code metadata if available
	if meta != nil && len(meta.Functions) > 0 {
		funcNames := make([]string, 0, len(meta.Functions))
		for _, f := range meta.Functions {
			funcNames = append(funcNames, f.Name)
		}
		baseMetadata["functions"] = strings.Join(funcNames, ",")
	}
	if meta != nil && len(meta.Types) > 0 {
		typeNames := make([]string, 0, len(meta.Types))
		for _, t := range meta.Types {
			typeNames = append(typeNames, t.Name)
		}
		baseMetadata["types"] = strings.Join(typeNames, ",")
	}
	if meta != nil && len(meta.Imports) > 0 {
		baseMetadata["imports"] = strings.Join(meta.Imports, ",")
	}

	// Step 6: Ingest chunks into vector database
	// Only ingest if searchEngine is properly initialized (has db and embedder)
	if ds.searchEngine != nil && ds.isSearchEngineReady() {
		for _, chunk := range chunks {
			chunkKey := fmt.Sprintf("%s:chunk:%d", relPath, chunk.Index)
			hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", ds.name, chunkKey)))
			chunkID := uuid.NewMD5(uuid.Nil, hash[:]).String()

			chunkMetadata := make(map[string]interface{})
			for k, v := range baseMetadata {
				chunkMetadata[k] = v
			}
			chunkMetadata["chunk_index"] = chunk.Index
			chunkMetadata["chunk_total"] = chunk.Total
			chunkMetadata["start_line"] = chunk.StartLine
			chunkMetadata["end_line"] = chunk.EndLine
			chunkMetadata["content"] = chunk.Content

			// Add chunk context if available
			if chunk.Context != nil {
				if chunk.Context.FunctionName != "" {
					chunkMetadata["function_context"] = chunk.Context.FunctionName
				}
				if chunk.Context.TypeName != "" {
					chunkMetadata["type_context"] = chunk.Context.TypeName
				}
			}

			if err := ds.searchEngine.IngestDocument(ctx, chunkID, chunk.Content, chunkMetadata); err != nil {
				return NewDocumentStoreError(ds.name, "indexDocument", "failed to ingest chunk", doc.ID, err)
			}
		}
	}

	return nil
}

// isSearchEngineReady checks if the search engine is properly initialized
func (ds *DocumentStore) isSearchEngineReady() bool {
	if ds.searchEngine == nil {
		return false
	}
	// Use reflection to check if db and embedder are set
	// This is needed because SearchEngine{} (empty struct) is not nil but has nil fields
	seValue := reflect.ValueOf(ds.searchEngine).Elem()
	dbField := seValue.FieldByName("db")
	embedderField := seValue.FieldByName("embedder")

	return !dbField.IsNil() && !embedderField.IsNil()
}

// detectDocumentType detects document type and language from document metadata or ID
func (ds *DocumentStore) detectDocumentType(doc *indexing.Document) (string, string) {
	// Try to get from metadata first
	if langVal, ok := doc.Metadata["language"].(string); ok {
		docType := doc.Metadata["type"].(string)
		if docType == "" {
			docType = DocumentTypeUnknown
		}
		return docType, langVal
	}

	// For directory sources, detect from file extension
	if ds.dataSource.Type() == "directory" {
		return ds.detectTypeAndLanguage(doc.ID)
	}

	// For SQL sources, default to text
	if ds.dataSource.Type() == "sql" {
		return DocumentTypeText, "sql"
	}

	// For API sources, default to text
	if ds.dataSource.Type() == "api" {
		return DocumentTypeText, "json"
	}

	return DocumentTypeUnknown, "unknown"
}

// normalizeMetadataValue converts unsupported types (like time.Time) to supported types
// for vector database storage. Qdrant and other databases may not support all Go types.
func normalizeMetadataValue(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		// Convert time.Time to RFC3339 string format
		return v.Format(time.RFC3339)
	case *time.Time:
		if v != nil {
			return v.Format(time.RFC3339)
		}
		return nil
	default:
		// Return as-is for supported types (string, int, float, bool, etc.)
		return value
	}
}

// detectMIMEType detects the MIME type of a file
func (ds *DocumentStore) detectMIMEType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil || n == 0 {
		return ""
	}

	return http.DetectContentType(buffer[:n])
}

func (ds *DocumentStore) computeFileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read first 4KB for hashing (quick change detection)
	buffer := make([]byte, 4096)
	n, err := f.Read(buffer)
	if err != nil && n == 0 {
		return ""
	}

	hash := md5.Sum(buffer[:n])
	return fmt.Sprintf("%x", hash)
}

func (ds *DocumentStore) detectTypeAndLanguage(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))

	typeMap := map[string]string{
		".go":   DocumentTypeCode,
		".py":   DocumentTypeCode,
		".js":   DocumentTypeCode,
		".ts":   DocumentTypeCode,
		".java": DocumentTypeCode,
		".cpp":  DocumentTypeCode,
		".c":    DocumentTypeCode,
		".rs":   DocumentTypeCode,
		".rb":   DocumentTypeCode,
		".php":  DocumentTypeCode,
		".cs":   DocumentTypeCode,
		".yaml": DocumentTypeConfig,
		".yml":  DocumentTypeConfig,
		".json": DocumentTypeConfig,
		".xml":  DocumentTypeConfig,
		".md":   DocumentTypeMarkdown,
		".txt":  DocumentTypeText,
		".sh":   DocumentTypeScript,

		".pdf":  DocumentTypeBinary,
		".docx": DocumentTypeBinary,
		".xlsx": DocumentTypeBinary,
	}

	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".rs":   "rust",
		".rb":   "ruby",
		".php":  "php",
		".cs":   "csharp",
		".yaml": "yaml",
		".yml":  "yaml",
		".json": "json",
		".xml":  "xml",
		".md":   "markdown",
		".txt":  "text",
		".sh":   "shell",

		".pdf":  "pdf",
		".docx": "word",
		".xlsx": "excel",
	}

	docType := typeMap[ext]
	if docType == "" {
		docType = DocumentTypeUnknown
	}

	language := langMap[ext]
	if language == "" {
		language = "unknown"
	}

	return docType, language
}

// GetSearchEngine returns the search engine associated with this document store
func (ds *DocumentStore) GetSearchEngine() *SearchEngine {
	return ds.searchEngine
}

// GetConfig returns the document store configuration
func (ds *DocumentStore) GetConfig() *config.DocumentStoreConfig {
	return ds.config
}

// GetCollectionName returns the collection name for this store.
// Uses Collection override if set, otherwise uses store name.
func (ds *DocumentStore) GetCollectionName() string {
	if ds.config.Collection != "" {
		return ds.config.Collection
	}
	return ds.name
}

// getCollectionName is an internal alias for GetCollectionName
func (ds *DocumentStore) getCollectionName() string {
	return ds.GetCollectionName()
}

func (ds *DocumentStore) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	if ds.searchEngine == nil {
		return nil, NewDocumentStoreError(ds.name, "Search", "search engine not available", "", nil)
	}

	// Always search by collection name (store name or override)
	collectionName := ds.getCollectionName()
	filter := map[string]interface{}{
		"collection": collectionName,
	}

	results, err := ds.searchEngine.SearchWithFilter(ctx, query, limit, filter)
	if err != nil {
		return nil, NewDocumentStoreError(ds.name, "Search", "vector search failed", "", err)
	}

	return results, nil
}

func (ds *DocumentStore) GetDocument(id string) (databases.SearchResult, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultFileWatchTimeout)
	defer cancel()

	results, err := ds.Search(ctx, id, 1)
	if err != nil || len(results) == 0 {
		return databases.SearchResult{}, false
	}

	return results[0], true
}

// documentStoreSearchTarget wraps DocumentStore to implement ParallelSearchTarget
type documentStoreSearchTarget struct {
	store *DocumentStore
}

func (t *documentStoreSearchTarget) GetID() string {
	return t.store.name
}

// SearchAllStores searches all registered document stores in parallel
// This is the unified parallel search function used by both SearchTool and SearchContext
func SearchAllStores(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	storeNames := ListDocumentStoresFromRegistry()
	if len(storeNames) == 0 {
		return []databases.SearchResult{}, nil
	}

	// Build search targets
	targets := make([]ParallelSearchTarget, 0, len(storeNames))
	for _, name := range storeNames {
		store, exists := GetDocumentStoreFromRegistry(name)
		if !exists {
			continue
		}
		targets = append(targets, &documentStoreSearchTarget{store: store})
	}

	if len(targets) == 0 {
		return []databases.SearchResult{}, nil
	}

	// Use shared parallel search function
	searchFunc := func(ctx context.Context, target ParallelSearchTarget) ([]databases.SearchResult, error) {
		storeTarget := target.(*documentStoreSearchTarget)
		return storeTarget.store.Search(ctx, query, limit)
	}

	results, err := ParallelSearch(ctx, targets, searchFunc)
	if err != nil {
		return nil, err
	}

	// Aggregate results from all stores
	var allResults []databases.SearchResult
	seenIDs := make(map[string]bool) // Deduplicate by document ID

	for _, result := range results {
		if result.Error != nil {
			// Log error but continue with other results
			fmt.Printf("Error: Search in store %s failed: %v\n", result.TargetID, result.Error)
			continue
		}

		storeResults := result.Results
		for _, res := range storeResults {
			if !seenIDs[res.ID] {
				seenIDs[res.ID] = true
				allResults = append(allResults, res)
			}
		}
	}

	// Sort by score (highest first) and limit to top results
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

func (ds *DocumentStore) shouldExclude(path string) bool {
	// Delegate to the data source filter (only for directory sources)
	if dirSource, ok := ds.dataSource.(*indexing.DirectorySource); ok {
		filter := dirSource.GetFilter()
		if filter != nil {
			return filter.ShouldExclude(path)
		}
	}
	return false
}

func (ds *DocumentStore) shouldInclude(path string) bool {
	// Delegate to the data source filter (only for directory sources)
	if dirSource, ok := ds.dataSource.(*indexing.DirectorySource); ok {
		filter := dirSource.GetFilter()
		if filter != nil {
			return filter.ShouldInclude(path)
		}
	}
	return true // Include by default if no filter
}

func (ds *DocumentStore) GetStatus() *DocumentStoreStatus {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	statusCopy := *ds.status
	return &statusCopy
}

func (ds *DocumentStore) initializeWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return NewDocumentStoreError(ds.name, "initializeWatcher", "failed to create watcher", "", err)
	}
	ds.watcher = watcher
	return nil
}

func (ds *DocumentStore) StartWatching() error {
	if ds.config.EnableWatchChanges == nil || !*ds.config.EnableWatchChanges || ds.watcher == nil {
		return NewDocumentStoreError(ds.name, "StartWatching", "file watching not enabled", "", nil)
	}

	ds.mu.Lock()
	if ds.status.IsWatching {
		ds.mu.Unlock()
		return NewDocumentStoreError(ds.name, "StartWatching", "already watching", "", nil)
	}
	ds.status.IsWatching = true
	ds.mu.Unlock()

	if err := ds.setupFileWatching(); err != nil {
		ds.mu.Lock()
		ds.status.IsWatching = false
		ds.mu.Unlock()
		return NewDocumentStoreError(ds.name, "StartWatching", "failed to setup file watching", "", err)
	}

	go ds.processUpdates()
	go ds.watchFileEvents()

	return nil
}

func (ds *DocumentStore) StopWatching() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.status.IsWatching {
		return nil
	}

	fmt.Printf("Stopping file watching for store '%s'...\n", ds.name)

	ds.cancel()
	if ds.watcher != nil {
		ds.watcher.Close()
	}
	close(ds.updateChannel)
	ds.status.IsWatching = false

	fmt.Printf("File watching stopped for store '%s'\n", ds.name)
	return nil
}

func (ds *DocumentStore) setupFileWatching() error {

	err := ds.watcher.Add(ds.sourcePath)
	if err != nil {
		return NewDocumentStoreError(ds.name, "setupFileWatching", "failed to watch source directory", ds.sourcePath, err)
	}

	return filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ds.shouldExclude(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			err := ds.watcher.Add(path)
			if err != nil {
				slog.Warn("Failed to watch directory", "path", path, "error", err)
			}
		}

		return nil
	})
}

func (ds *DocumentStore) watchFileEvents() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case event, ok := <-ds.watcher.Events:
			if !ok {
				return
			}
			ds.handleFileEvent(event)
		case err, ok := <-ds.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error in store", "store", ds.name, "error", err)
		}
	}
}

func (ds *DocumentStore) handleFileEvent(event fsnotify.Event) {

	if ds.shouldExclude(event.Name) || !ds.shouldInclude(event.Name) {
		return
	}

	var operation string

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = OperationCreate
		if _, err := os.Stat(event.Name); err == nil {
			// Read document from data source
			if doc, err := ds.dataSource.ReadDocument(ds.ctx, event.Name); err == nil {
				_ = ds.indexDocument(doc)
			}
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = OperationModify
		if _, err := os.Stat(event.Name); err == nil {
			// Read document from data source
			if doc, err := ds.dataSource.ReadDocument(ds.ctx, event.Name); err == nil {
				_ = ds.indexDocument(doc)
			}
		}
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = OperationDelete
	default:
		return
	}

	select {
	case ds.updateChannel <- DocumentUpdate{
		FilePath:  event.Name,
		Operation: operation,
		Timestamp: time.Now(),
	}:
	case <-ds.ctx.Done():
		return
	default:
		slog.Warn("Update channel full for store, dropping update", "store", ds.name, "file", event.Name)
	}
}

func (ds *DocumentStore) processUpdates() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case update, ok := <-ds.updateChannel:
			if !ok {
				return
			}
			// Process file updates from file watcher
			switch update.Operation {
			case OperationCreate, OperationModify:
				// File already re-indexed in handleFileEvent
				slog.Debug("Document updated", "file", update.FilePath, "operation", update.Operation)
			case OperationDelete:
				// Clean up deleted file from index
				if ds.searchEngine != nil {
					relPath, _ := filepath.Rel(ds.sourcePath, update.FilePath)
					filter := map[string]interface{}{
						"store_name": ds.name,
						"path":       relPath,
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					if err := ds.searchEngine.DeleteByFilter(ctx, filter); err != nil {
						slog.Error("Failed to delete document from index", "path", relPath, "error", err)
					}
					cancel()
				}
			}
		}
	}
}

func (ds *DocumentStore) RefreshDocument(relativePath string) error {
	fullPath := filepath.Join(ds.sourcePath, relativePath)
	_, err := os.Stat(fullPath)
	if err != nil {
		fmt.Printf("Document %s might have been deleted\n", relativePath)
		return nil
	}

	// Read document from data source
	doc, err := ds.dataSource.ReadDocument(ds.ctx, fullPath)
	if err != nil {
		return err
	}
	return ds.indexDocument(doc)
}

type IndexedFileInfo struct {
	Path         string
	LastModified int64
}

type IndexState struct {
	StoreName   string                   `json:"store_name"`
	SourcePath  string                   `json:"source_path"`
	LastIndexed time.Time                `json:"last_indexed"`
	Files       map[string]FileIndexInfo `json:"files"`
	TotalFiles  int                      `json:"total_files"`
	TotalChunks int                      `json:"total_chunks"`
}

type FileIndexInfo struct {
	ModTime int64  `json:"mod_time"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash,omitempty"` // MD5 hash of first 4KB for quick change detection
}

func (ds *DocumentStore) getIndexStatePath() string {

	stateDir := filepath.Join(ds.sourcePath, ".hector")

	return filepath.Join(stateDir, fmt.Sprintf("index_state_%s.json", ds.name))
}

func (ds *DocumentStore) loadIndexState() (map[string]FileIndexInfo, error) {
	statePath := ds.getIndexStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]FileIndexInfo), nil
		}
		return nil, fmt.Errorf("failed to read index state: %w", err)
	}

	var state IndexState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse index state: %w", err)
	}

	if state.StoreName != ds.name || state.SourcePath != ds.sourcePath {
		slog.Warn("Index state mismatch, rebuilding index", "state_store", state.StoreName, "current_store", ds.name, "state_path", state.SourcePath, "current_path", ds.sourcePath)
		return make(map[string]FileIndexInfo), nil
	}

	return state.Files, nil
}

func (ds *DocumentStore) saveIndexState(files map[string]FileIndexInfo, totalChunks int) error {
	statePath := ds.getIndexStatePath()
	stateDir := filepath.Dir(statePath)

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := IndexState{
		StoreName:   ds.name,
		SourcePath:  ds.sourcePath,
		LastIndexed: time.Now(),
		Files:       files,
		TotalFiles:  len(files),
		TotalChunks: totalChunks,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index state: %w", err)
	}

	tempPath := statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index state: %w", err)
	}

	if err := os.Rename(tempPath, statePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to save index state: %w", err)
	}

	return nil
}

func (ds *DocumentStore) shouldReindexFile(path string, currentModTime time.Time, existingDocs map[string]FileIndexInfo) bool {
	if ds.config.EnableIncrementalIndexing == nil || !*ds.config.EnableIncrementalIndexing {
		return true
	}

	if len(existingDocs) == 0 {
		return true
	}

	relPath, err := filepath.Rel(ds.sourcePath, path)
	if err != nil {
		return true
	}

	storedInfo, exists := existingDocs[relPath]
	if !exists {
		// New file
		return true
	}

	// Check file size first (fastest)
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	if info.Size() != storedInfo.Size {
		return true
	}

	// Check modification time
	currentUnix := currentModTime.Unix()
	if currentUnix > storedInfo.ModTime {
		// File modified - verify with hash if available
		if storedInfo.Hash != "" {
			currentHash := ds.computeFileHash(path)
			return currentHash != storedInfo.Hash
		}
		return true
	}

	return false
}

func (ds *DocumentStore) cleanupDeletedFiles(ctx context.Context, existingDocs map[string]FileIndexInfo, foundFiles map[string]bool) ([]string, map[string]bool, error) {
	// Skip cleanup if no existing docs
	if len(existingDocs) == 0 {
		return nil, nil, nil
	}

	deletedFiles := make([]string, 0)
	cleanedUpFiles := make(map[string]bool)

	// Identify all deleted files
	for path := range existingDocs {
		if !foundFiles[path] {
			deletedFiles = append(deletedFiles, path)
			ds.progressTracker.IncrementDeleted()
		}
	}

	if len(deletedFiles) == 0 {
		return deletedFiles, cleanedUpFiles, nil
	}

	// If no search engine or search engine not ready, just remove from index state
	if ds.searchEngine == nil || !ds.isSearchEngineReady() {
		for _, path := range deletedFiles {
			cleanedUpFiles[path] = true
		}
		fmt.Printf("🗑️  Removed %d deleted file(s) from index state\n", len(deletedFiles))
		return deletedFiles, cleanedUpFiles, nil
	}

	// Clean up from vector database
	successCount := 0
	for _, path := range deletedFiles {
		filter := map[string]interface{}{
			"store_name": ds.name,
			"path":       path,
		}

		if err := ds.searchEngine.DeleteByFilter(ctx, filter); err != nil {
			slog.Warn("Failed to delete from vector DB", "path", path, "error", err)
		} else {
			cleanedUpFiles[path] = true
			successCount++
		}
	}

	if successCount > 0 {
		fmt.Printf("🗑️  Cleaned up %d deleted file(s) from index '%s'\n", successCount, ds.name)
	}
	if successCount < len(deletedFiles) {
		fmt.Printf("⚠️  %d file(s) pending cleanup (will retry)\n", len(deletedFiles)-successCount)
	}

	return deletedFiles, cleanedUpFiles, nil
}

func (ds *DocumentStore) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.status.IsWatching {
		ds.cancel()
		if ds.watcher != nil {
			ds.watcher.Close()
		}
		close(ds.updateChannel)
		ds.status.IsWatching = false
	}

	return nil
}

type DocumentStoreRegistry struct {
	mu     sync.RWMutex
	stores map[string]*DocumentStore
}

var globalDocumentStoreRegistry *DocumentStoreRegistry

func init() {
	globalDocumentStoreRegistry = &DocumentStoreRegistry{
		stores: make(map[string]*DocumentStore),
	}
}

func RegisterDocumentStore(store *DocumentStore) {
	globalDocumentStoreRegistry.mu.Lock()
	defer globalDocumentStoreRegistry.mu.Unlock()

	globalDocumentStoreRegistry.stores[store.name] = store
}

func GetDocumentStoreFromRegistry(name string) (*DocumentStore, bool) {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	store, exists := globalDocumentStoreRegistry.stores[name]
	return store, exists
}

func ListDocumentStoresFromRegistry() []string {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	var names []string
	for name := range globalDocumentStoreRegistry.stores {
		names = append(names, name)
	}
	return names
}

func GetDocumentStoreStats() map[string]interface{} {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, store := range globalDocumentStoreRegistry.stores {
		stats[name] = store.GetStatus()
	}

	return stats
}

func UnregisterDocumentStore(name string) {
	globalDocumentStoreRegistry.mu.Lock()
	defer globalDocumentStoreRegistry.mu.Unlock()

	if store, exists := globalDocumentStoreRegistry.stores[name]; exists {

		if store.status.IsWatching {
			_ = store.StopWatching()
		}

		store.Close()
		delete(globalDocumentStoreRegistry.stores, name)
	}
}

// InitializeDocumentStoresFromConfig initializes document stores from configs.
// storeNames is a slice of store names (map keys) corresponding to configs.
// This function is deprecated - use ContextServiceBuilder instead.
func InitializeDocumentStoresFromConfig(storeNames []string, configs []*config.DocumentStoreConfig, searchEngine *SearchEngine) error {
	if len(storeNames) != len(configs) {
		return fmt.Errorf("storeNames and configs must have the same length")
	}
	if len(configs) == 0 {
		return nil
	}

	for i, config := range configs {
		storeName := storeNames[i]
		store, err := NewDocumentStore(storeName, config, searchEngine)
		if err != nil {
			fmt.Printf("Warning: Failed to create document store %s: %v\n", storeName, err)
			continue
		}

		RegisterDocumentStore(store)

		if err := store.StartIndexing(); err != nil {
			fmt.Printf("Warning: Failed to index document store %s: %v\n", storeName, err)
			continue
		}

		if store.config.EnableWatchChanges != nil && *store.config.EnableWatchChanges {
			go func(s *DocumentStore, name string) {
				if err := s.StartWatching(); err != nil {
					fmt.Printf("Warning: Failed to start file watching for %s: %v\n", name, err)
				}
			}(store, storeName)
		}
	}

	return nil
}
