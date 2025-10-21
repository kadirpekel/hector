package context

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
)

// ============================================================================
// DOCUMENT STORE CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	// DefaultMaxFileSize is the default maximum file size for indexing (5MB)
	DefaultMaxFileSize = 5 * 1024 * 1024

	// DefaultUpdateChannelSize is the default size for the update channel
	DefaultUpdateChannelSize = 100

	// DefaultFileWatchTimeout is the default timeout for file watching operations
	DefaultFileWatchTimeout = 10 * time.Second

	// MaxConcurrentIndexing is the maximum number of concurrent indexing operations
	// Reduced to prevent overwhelming Ollama with too many embedding requests
	MaxConcurrentIndexing = 3
)

// Document types
const (
	DocumentTypeCode     = "code"
	DocumentTypeConfig   = "config"
	DocumentTypeMarkdown = "markdown"
	DocumentTypeText     = "text"
	DocumentTypeScript   = "script"
	DocumentTypeBinary   = "binary" // Binary documents (PDF, Word, etc.)
	DocumentTypeUnknown  = "unknown"
)

// File operations
const (
	OperationCreate = "create"
	OperationModify = "modify"
	OperationDelete = "delete"
)

// Default indexing timeout
const (
	DefaultIndexingTimeout = 120 * time.Second // 2 minutes per document
)

// ============================================================================
// DOCUMENT STORE ERRORS - STANDARDIZED ERROR TYPES
// ============================================================================

// DocumentStoreError represents errors in document store operations
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

// NewDocumentStoreError creates a new document store error
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

// ============================================================================
// DOCUMENT TYPES AND STRUCTURES
// ============================================================================

// Document represents document metadata during indexing
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

	// Extracted entities for code files
	Functions []string `json:"functions,omitempty"`
	Structs   []string `json:"structs,omitempty"`
	Imports   []string `json:"imports,omitempty"`
}

// DocumentUpdate represents a file change event
type DocumentUpdate struct {
	FilePath  string    `json:"file_path"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}

// DocumentStoreStatus represents the status of a document store
type DocumentStoreStatus struct {
	Name          string    `json:"name"`
	SourcePath    string    `json:"source_path"`
	Storage       string    `json:"storage"`
	LastIndexed   time.Time `json:"last_indexed"`
	IsIndexing    bool      `json:"is_indexing"`
	IsWatching    bool      `json:"is_watching"`
	DocumentCount int       `json:"document_count"`
}

// ============================================================================
// DOCUMENT STORE - ENHANCED WITH PROPER STRUCTURE
// ============================================================================

// DocumentStore manages indexed documents for any directory using vector database
type DocumentStore struct {
	mu     sync.RWMutex
	name   string
	config *config.DocumentStoreConfig
	status *DocumentStoreStatus

	// Core components
	searchEngine *SearchEngine
	sourcePath   string

	// Native binary parsers
	nativeParsers *NativeParserRegistry

	// File watching components
	watcher       *fsnotify.Watcher
	updateChannel chan DocumentUpdate
	ctx           context.Context
	cancel        context.CancelFunc

	// Indexing state
	indexingSemaphore chan struct{}
}

// NewDocumentStore creates a new document store with enhanced validation and configuration
func NewDocumentStore(storeConfig *config.DocumentStoreConfig, searchEngine *SearchEngine) (*DocumentStore, error) {
	if storeConfig == nil {
		return nil, NewDocumentStoreError("", "NewDocumentStore", "store config is required", "", nil)
	}
	if searchEngine == nil {
		return nil, NewDocumentStoreError(storeConfig.Name, "NewDocumentStore", "search engine is required", "", nil)
	}

	// Validate and set defaults
	if err := validateAndSetDefaults(storeConfig); err != nil {
		return nil, err
	}

	// Initialize context for file watching
	ctx, cancel := context.WithCancel(context.Background())

	store := &DocumentStore{
		name:              storeConfig.Name,
		config:            storeConfig,
		searchEngine:      searchEngine,
		sourcePath:        storeConfig.Path,
		nativeParsers:     NewNativeParserRegistry(),
		updateChannel:     make(chan DocumentUpdate, DefaultUpdateChannelSize),
		ctx:               ctx,
		cancel:            cancel,
		indexingSemaphore: make(chan struct{}, MaxConcurrentIndexing),
		status: &DocumentStoreStatus{
			Name:        storeConfig.Name,
			SourcePath:  storeConfig.Path,
			Storage:     "vector_database",
			LastIndexed: time.Time{},
			IsIndexing:  false,
			IsWatching:  false,
		},
	}

	// Initialize file watcher if change tracking is enabled
	if storeConfig.WatchChanges {
		if err := store.initializeWatcher(); err != nil {
			cancel()
			return nil, NewDocumentStoreError(storeConfig.Name, "NewDocumentStore", "failed to initialize watcher", "", err)
		}
	}

	return store, nil
}

// ============================================================================
// VALIDATION AND CONFIGURATION
// ============================================================================

// validateAndSetDefaults validates and sets default values for store config
func validateAndSetDefaults(storeConfig *config.DocumentStoreConfig) error {
	if storeConfig.Name == "" {
		return NewDocumentStoreError("", "validateAndSetDefaults", "store name is required", "", nil)
	}
	if storeConfig.Path == "" {
		return NewDocumentStoreError(storeConfig.Name, "validateAndSetDefaults", "source path is required", "", nil)
	}
	if storeConfig.Source == "" {
		storeConfig.Source = "directory" // Default source type
	}

	// Set defaults
	if storeConfig.MaxFileSize == 0 {
		storeConfig.MaxFileSize = DefaultMaxFileSize
	}
	if len(storeConfig.IncludePatterns) == 0 {
		storeConfig.IncludePatterns = []string{"*"} // Include all by default
	}

	// Validate source path exists
	if _, err := os.Stat(storeConfig.Path); os.IsNotExist(err) {
		return NewDocumentStoreError(storeConfig.Name, "validateAndSetDefaults", "source path does not exist", storeConfig.Path, err)
	}

	return nil
}

// ============================================================================
// INDEXING OPERATIONS - ENHANCED
// ============================================================================

// StartIndexing begins indexing the document store with enhanced error handling
func (ds *DocumentStore) StartIndexing() error {
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

	fmt.Printf("🔍 Indexing document store '%s' from: %s\n", ds.name, ds.sourcePath)

	switch ds.config.Source {
	case "directory":
		return ds.indexDirectory()
	case "git":
		return ds.indexGitRepository()
	default:
		return NewDocumentStoreError(ds.name, "StartIndexing", "unsupported source type", ds.config.Source, nil)
	}
}

// indexDirectory indexes a local directory with enhanced error handling and progress tracking
func (ds *DocumentStore) indexDirectory() error {
	ctx := context.Background()

	// Load existing index state if incremental indexing is enabled
	var existingDocs map[string]int64
	var err error
	if ds.config.IncrementalIndexing {
		existingDocs, err = ds.loadIndexState()
		if err != nil {
			log.Printf("Warning: Failed to load index state, performing full reindex: %v", err)
			existingDocs = make(map[string]int64)
		}

		if len(existingDocs) > 0 {
			fmt.Printf("📊 Incremental indexing: Found %d existing file(s) in index\n", len(existingDocs))
		} else {
			fmt.Printf("📊 First indexing or full reindex mode\n")
		}
	}

	var indexedCount sync.WaitGroup
	var successCount int32
	var failCount int32
	var skippedCount int32

	// Track files found during walk (for cleanup and state saving)
	foundFiles := make(map[string]bool)
	indexedFiles := make(map[string]int64) // Track successfully indexed files with timestamps
	var filesMu sync.Mutex

	err = filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			atomic.AddInt32(&failCount, 1)
			log.Printf("Warning: Failed to access %s: %v", path, err)
			return nil // Continue with other files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Apply filters
		if ds.shouldExclude(path) || !ds.shouldInclude(path) {
			return nil
		}

		// Skip large files
		if info.Size() > ds.config.MaxFileSize {
			return nil
		}

		// Track this file as found (for cleanup)
		relPath, _ := filepath.Rel(ds.sourcePath, path)
		filesMu.Lock()
		foundFiles[relPath] = true
		filesMu.Unlock()

		// Check if file needs reindexing (incremental indexing)
		if !ds.shouldReindexFile(path, info.ModTime(), existingDocs) {
			atomic.AddInt32(&skippedCount, 1)
			return nil
		}

		// Index the document with semaphore control
		ds.indexingSemaphore <- struct{}{} // Acquire semaphore
		indexedCount.Add(1)
		go func(p string, i os.FileInfo) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic while indexing %s: %v", p, r)
					atomic.AddInt32(&failCount, 1)
				}
				<-ds.indexingSemaphore // Release semaphore
				indexedCount.Done()
			}()

			if err := ds.indexDocument(p, i); err != nil {
				atomic.AddInt32(&failCount, 1)
				log.Printf("Warning: Failed to index %s: %v", p, err)
			} else {
				atomic.AddInt32(&successCount, 1)
				// Track successfully indexed file with its timestamp
				rp, _ := filepath.Rel(ds.sourcePath, p)
				filesMu.Lock()
				indexedFiles[rp] = i.ModTime().Unix()
				filesMu.Unlock()
			}
		}(path, info)

		return nil
	})

	if err != nil {
		return NewDocumentStoreError(ds.name, "indexDirectory", "directory walk failed", ds.sourcePath, err)
	}

	// Wait for all indexing operations to complete
	indexedCount.Wait()

	// Cleanup deleted files if incremental indexing is enabled
	if ds.config.IncrementalIndexing {
		if err := ds.cleanupDeletedFiles(ctx, existingDocs, foundFiles); err != nil {
			log.Printf("Warning: Cleanup of deleted files failed: %v", err)
		}
	}

	ds.mu.Lock()
	ds.status.DocumentCount = int(successCount)
	ds.mu.Unlock()

	// Save index state for incremental indexing
	if ds.config.IncrementalIndexing {
		// Merge: keep timestamps for unchanged files, update for re-indexed files
		finalState := make(map[string]int64)

		// Start with existing files that weren't re-indexed (the unchanged ones)
		for path, ts := range existingDocs {
			if foundFiles[path] { // File still exists
				finalState[path] = ts
			}
		}

		// Update with newly indexed files (overwrites existing entries)
		for path, ts := range indexedFiles {
			finalState[path] = ts
		}

		// Save state (totalChunks is approximate - we track files, not chunks)
		if err := ds.saveIndexState(finalState, len(finalState)*3); err != nil {
			log.Printf("Warning: Failed to save index state: %v", err)
		}
	}

	// Enhanced status message for incremental indexing
	if ds.config.IncrementalIndexing && skippedCount > 0 {
		fmt.Printf("✅ Document store '%s' indexed: %d new/modified, %d unchanged, %d errors\n",
			ds.name, successCount, skippedCount, failCount)
	} else {
		fmt.Printf("✅ Document store '%s' indexed: %d documents (%d errors)\n",
			ds.name, successCount, failCount)
	}

	// Notify user if file watching is enabled
	if ds.config.WatchChanges {
		fmt.Printf("👁️  File watching enabled - changes will be automatically indexed\n")
	}

	return nil
}

// indexGitRepository indexes a git repository (currently delegates to directory indexing)
func (ds *DocumentStore) indexGitRepository() error {
	// Check if the path is a git repository
	if !ds.isGitRepository(ds.sourcePath) {
		return fmt.Errorf("path %s is not a git repository", ds.sourcePath)
	}

	// Use git to get all tracked files
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = ds.sourcePath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list git files: %w", err)
	}

	// Process each tracked file
	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if file == "" {
			continue
		}

		fullPath := filepath.Join(ds.sourcePath, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files that don't exist
		}

		if !info.IsDir() {
			if err := ds.indexDocument(fullPath, info); err != nil {
				// Log error but continue processing other files
				fmt.Printf("Warning: failed to index file %s: %v\n", fullPath, err)
			}
		}
	}

	return nil
}

// isGitRepository checks if the given path is a git repository
func (ds *DocumentStore) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// indexDocument indexes a single document into vector database with enhanced processing
// Now uses chunking to split large files into smaller, semantically meaningful pieces
func (ds *DocumentStore) indexDocument(path string, info os.FileInfo) error {
	relPath, _ := filepath.Rel(ds.sourcePath, path)

	// Try to parse with plugin first, fallback to text reading
	content, err := ds.extractContentWithPlugins(path, info)
	if err != nil {
		// Fallback to reading as text
		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return NewDocumentStoreError(ds.name, "indexDocument", "failed to read file", path, readErr)
		}
		content = string(contentBytes)
	}

	// Create document with enhanced metadata
	doc := ds.createDocument(relPath, info, content)

	// Extract metadata based on file type
	ds.extractMetadata(doc)

	// Prepare metadata for vector database
	metadata := ds.prepareVectorMetadata(doc)

	// Chunk the content for better semantic search
	chunks := ds.chunkContent(doc.Content, 800) // ~800 chars per chunk

	ctx, cancel := context.WithTimeout(context.Background(), DefaultIndexingTimeout)
	defer cancel()

	// Index each chunk separately with line number tracking
	for i, chunk := range chunks {
		// Generate a proper UUID for each chunk
		chunkKey := fmt.Sprintf("%s:chunk:%d", relPath, i)
		hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", ds.name, chunkKey)))
		chunkID := uuid.NewMD5(uuid.Nil, hash[:]).String()

		// Add chunk-specific metadata with line numbers
		chunkMetadata := make(map[string]interface{})
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		chunkMetadata["chunk_index"] = i
		chunkMetadata["chunk_total"] = len(chunks)
		chunkMetadata["start_line"] = chunk.StartLine
		chunkMetadata["end_line"] = chunk.EndLine
		chunkMetadata["content"] = chunk.Content // Store chunk content in metadata for retrieval

		if err := ds.searchEngine.IngestDocument(ctx, chunkID, chunk.Content, chunkMetadata); err != nil {
			return NewDocumentStoreError(ds.name, "indexDocument", "failed to ingest chunk", path, err)
		}
	}

	return nil
}

// extractContentWithPlugins attempts to extract content using native parsers or plugins
func (ds *DocumentStore) extractContentWithPlugins(path string, info os.FileInfo) (string, error) {
	// First try native parsers for binary files
	if isBinaryFileType(path) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, err := ds.nativeParsers.ParseDocument(ctx, path, info.Size())
		if err != nil {
			return "", fmt.Errorf("native parser failed: %w", err)
		}

		if !result.Success {
			return "", fmt.Errorf("native parser failed: %s", result.Error)
		}

		// Log successful parsing
		if result.ProcessingTimeMs > 0 {
			log.Printf("✅ Parsed %s with native parser (%dms)", path, result.ProcessingTimeMs)
		}

		return result.Content, nil
	}

	// For non-binary files, fall back to plain text reading
	// Plugin system integration will be implemented when needed
	return "", fmt.Errorf("no native parser available for text files - use plain text reading")
}

// ContentChunk represents a chunk of content with line number tracking
type ContentChunk struct {
	Content   string
	StartLine int
	EndLine   int
}

// chunkContent splits content into smaller chunks for better semantic search
// Now tracks line numbers for precise code references
func (ds *DocumentStore) chunkContent(content string, targetSize int) []ContentChunk {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If content is small, return as single chunk
	if len(content) <= targetSize {
		return []ContentChunk{{
			Content:   content,
			StartLine: 1,
			EndLine:   totalLines,
		}}
	}

	var chunks []ContentChunk
	var currentChunk strings.Builder
	chunkStartLine := 1
	currentLine := 1

	for _, line := range lines {
		// If adding this line would exceed target size, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(line)+1 > targetSize {
			chunks = append(chunks, ContentChunk{
				Content:   currentChunk.String(),
				StartLine: chunkStartLine,
				EndLine:   currentLine - 1,
			})
			currentChunk.Reset()
			chunkStartLine = currentLine
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		currentLine++
	}

	// Add the last chunk if not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, ContentChunk{
			Content:   currentChunk.String(),
			StartLine: chunkStartLine,
			EndLine:   totalLines,
		})
	}

	return chunks
}

// ============================================================================
// DOCUMENT PROCESSING AND METADATA EXTRACTION
// ============================================================================

// createDocument creates a document from file information
func (ds *DocumentStore) createDocument(relPath string, info os.FileInfo, content string) *Document {
	doc := &Document{
		ID:           ds.generateDocumentID(relPath),
		Path:         relPath,
		Name:         info.Name(),
		Content:      content,
		Size:         info.Size(),
		Lines:        strings.Count(content, "\n") + 1,
		LastModified: info.ModTime(),
		Metadata:     make(map[string]string),
	}

	// Detect document type and language
	doc.Type, doc.Language = ds.detectTypeAndLanguage(relPath)

	// Extract title
	doc.Title = ds.extractTitle(doc)

	return doc
}

// detectTypeAndLanguage detects the document type and language from file extension
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
		// Binary document types
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
		// Binary document languages
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

// extractTitle extracts the title from a document
func (ds *DocumentStore) extractTitle(doc *Document) string {
	switch doc.Type {
	case DocumentTypeMarkdown:
		return ds.extractMarkdownTitle(doc.Content)
	case DocumentTypeCode:
		return doc.Name // Use filename for code files
	default:
		return doc.Name
	}
}

// extractMarkdownTitle extracts title from markdown content
func (ds *DocumentStore) extractMarkdownTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}
	return ""
}

// extractMetadata extracts metadata based on document type
func (ds *DocumentStore) extractMetadata(doc *Document) {
	switch doc.Language {
	case "go":
		ds.extractGoMetadata(doc)
	case "yaml":
		ds.extractYAMLMetadata(doc)
	case "markdown":
		ds.extractMarkdownMetadata(doc)
	}
}

// extractGoMetadata extracts Go-specific metadata
func (ds *DocumentStore) extractGoMetadata(doc *Document) {
	lines := strings.Split(doc.Content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract functions
		if strings.HasPrefix(line, "func ") {
			if funcName := ds.extractGoFunctionName(line); funcName != "" {
				doc.Functions = append(doc.Functions, funcName)
			}
		}

		// Extract structs
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			if structName := ds.extractGoStructName(line); structName != "" {
				doc.Structs = append(doc.Structs, structName)
			}
		}

		// Extract imports
		if strings.HasPrefix(line, "import ") || (strings.Contains(line, `"`) && strings.Contains(line, "/")) {
			if importPath := ds.extractGoImport(line); importPath != "" {
				doc.Imports = append(doc.Imports, importPath)
			}
		}
	}
}

// extractYAMLMetadata extracts YAML-specific metadata
func (ds *DocumentStore) extractYAMLMetadata(doc *Document) {
	lines := strings.Split(doc.Content, "\n")
	var keys []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				key := strings.TrimSpace(parts[0])
				if key != "" {
					keys = append(keys, key)
				}
			}
		}
	}

	if len(keys) > 0 {
		doc.Metadata["yaml_keys"] = strings.Join(keys, ",")
	}
}

// extractMarkdownMetadata extracts markdown-specific metadata
func (ds *DocumentStore) extractMarkdownMetadata(doc *Document) {
	lines := strings.Split(doc.Content, "\n")
	var headers []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			header := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if header != "" {
				headers = append(headers, header)
			}
		}
	}

	if len(headers) > 0 {
		doc.Metadata["headers"] = strings.Join(headers, ",")
	}
}

// ============================================================================
// HELPER METHODS FOR METADATA EXTRACTION
// ============================================================================

// extractGoFunctionName extracts function name from Go function declaration
func (ds *DocumentStore) extractGoFunctionName(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		funcName := parts[1]
		if parenIdx := strings.Index(funcName, "("); parenIdx > 0 {
			return funcName[:parenIdx]
		}
		return funcName
	}
	return ""
}

// extractGoStructName extracts struct name from Go struct declaration
func (ds *DocumentStore) extractGoStructName(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// extractGoImport extracts import path from Go import statement
func (ds *DocumentStore) extractGoImport(line string) string {
	if strings.Contains(line, `"`) {
		start := strings.Index(line, `"`)
		end := strings.LastIndex(line, `"`)
		if start != -1 && end != -1 && start < end {
			return line[start+1 : end]
		}
	}
	return ""
}

// ============================================================================
// SEARCH AND RETRIEVAL OPERATIONS
// ============================================================================

// Search searches documents in this store using vector database
func (ds *DocumentStore) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	// Create filter to search only this store's collection
	filter := map[string]interface{}{
		"store_name": ds.name,
	}

	// Use the search engine to perform vector similarity search with store filter
	results, err := ds.searchEngine.SearchWithFilter(ctx, query, limit, filter)
	if err != nil {
		return nil, NewDocumentStoreError(ds.name, "Search", "vector search failed", "", err)
	}

	return results, nil
}

// GetDocument retrieves a document by ID from vector database
func (ds *DocumentStore) GetDocument(id string) (databases.SearchResult, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultFileWatchTimeout)
	defer cancel()

	results, err := ds.Search(ctx, id, 1)
	if err != nil || len(results) == 0 {
		return databases.SearchResult{}, false
	}

	return results[0], true
}

// ============================================================================
// FILE FILTERING AND PATTERN MATCHING
// ============================================================================

// shouldExclude checks if a file should be excluded based on patterns
func (ds *DocumentStore) shouldExclude(path string) bool {
	for _, pattern := range ds.config.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// shouldInclude checks if a file should be included based on patterns
func (ds *DocumentStore) shouldInclude(path string) bool {
	if len(ds.config.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range ds.config.IncludePatterns {
		if pattern == "*" {
			return true
		}
		if strings.HasSuffix(path, strings.TrimPrefix(pattern, "*")) {
			return true
		}
	}
	return false
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// generateDocumentID generates a consistent UUID based on store name and file path
func (ds *DocumentStore) generateDocumentID(path string) string {
	fullPath := fmt.Sprintf("%s:%s", ds.name, path)
	hash := md5.Sum([]byte(fullPath))
	return uuid.NewMD5(uuid.Nil, hash[:]).String()
}

// prepareVectorMetadata prepares metadata for vector database ingestion
func (ds *DocumentStore) prepareVectorMetadata(doc *Document) map[string]interface{} {
	metadata := map[string]interface{}{
		"path":          doc.Path,
		"name":          doc.Name,
		"title":         doc.Title,
		"type":          doc.Type,
		"language":      doc.Language,
		"size":          doc.Size,
		"lines":         doc.Lines,
		"last_modified": doc.LastModified.Unix(),
		"functions":     strings.Join(doc.Functions, ","),
		"structs":       strings.Join(doc.Structs, ","),
		"imports":       strings.Join(doc.Imports, ","),
		"store_name":    ds.name,
		"indexed_at":    time.Now().Unix(),
	}

	// Add custom metadata
	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	return metadata
}

// ============================================================================
// STATUS AND HEALTH METHODS
// ============================================================================

// GetStatus returns detailed status information
func (ds *DocumentStore) GetStatus() *DocumentStoreStatus {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Return a copy to prevent external modification
	statusCopy := *ds.status
	return &statusCopy
}

// ============================================================================
// FILE WATCHING OPERATIONS
// ============================================================================

// initializeWatcher initializes the file system watcher
func (ds *DocumentStore) initializeWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return NewDocumentStoreError(ds.name, "initializeWatcher", "failed to create watcher", "", err)
	}
	ds.watcher = watcher
	return nil
}

// StartWatching enables automatic file change tracking
func (ds *DocumentStore) StartWatching() error {
	if !ds.config.WatchChanges || ds.watcher == nil {
		return NewDocumentStoreError(ds.name, "StartWatching", "file watching not enabled", "", nil)
	}

	ds.mu.Lock()
	if ds.status.IsWatching {
		ds.mu.Unlock()
		return NewDocumentStoreError(ds.name, "StartWatching", "already watching", "", nil)
	}
	ds.status.IsWatching = true
	ds.mu.Unlock()

	// File watching started (verbose logging removed)

	// Set up file watching
	if err := ds.setupFileWatching(); err != nil {
		ds.mu.Lock()
		ds.status.IsWatching = false
		ds.mu.Unlock()
		return NewDocumentStoreError(ds.name, "StartWatching", "failed to setup file watching", "", err)
	}

	// Start background processing
	go ds.processUpdates()
	go ds.watchFileEvents()

	// File watching enabled (verbose logging removed)
	return nil
}

// StopWatching disables automatic file change tracking
func (ds *DocumentStore) StopWatching() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.status.IsWatching {
		return nil
	}

	fmt.Printf("🛑 Stopping file watching for store '%s'...\n", ds.name)

	ds.cancel()
	if ds.watcher != nil {
		ds.watcher.Close()
	}
	close(ds.updateChannel)
	ds.status.IsWatching = false

	fmt.Printf("✅ File watching stopped for store '%s'\n", ds.name)
	return nil
}

// setupFileWatching sets up file system watching
func (ds *DocumentStore) setupFileWatching() error {
	// Add the source directory
	err := ds.watcher.Add(ds.sourcePath)
	if err != nil {
		return NewDocumentStoreError(ds.name, "setupFileWatching", "failed to watch source directory", ds.sourcePath, err)
	}

	// Walk and add subdirectories
	return filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if ds.shouldExclude(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add directories to watcher
		if info.IsDir() {
			err := ds.watcher.Add(path)
			if err != nil {
				log.Printf("Warning: Failed to watch directory %s: %v", path, err)
			}
		}

		return nil
	})
}

// watchFileEvents processes file system events
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
			log.Printf("File watcher error in store %s: %v", ds.name, err)
		}
	}
}

// handleFileEvent processes individual file system events
func (ds *DocumentStore) handleFileEvent(event fsnotify.Event) {
	// Skip if file doesn't match our patterns
	if ds.shouldExclude(event.Name) || !ds.shouldInclude(event.Name) {
		return
	}

	var operation string

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = OperationCreate
		if info, err := os.Stat(event.Name); err == nil {
			_ = ds.indexDocument(event.Name, info)
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = OperationModify
		if info, err := os.Stat(event.Name); err == nil {
			_ = ds.indexDocument(event.Name, info)
		}
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = OperationDelete
	default:
		return // Ignore other operations
	}

	// Send update to processing channel
	select {
	case ds.updateChannel <- DocumentUpdate{
		FilePath:  event.Name,
		Operation: operation,
		Timestamp: time.Now(),
	}:
	case <-ds.ctx.Done():
		return
	default:
		log.Printf("Update channel full for store %s, dropping update for %s", ds.name, event.Name)
	}
}

// processUpdates handles incremental document updates
func (ds *DocumentStore) processUpdates() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case _, ok := <-ds.updateChannel:
			if !ok {
				return
			}

			// Silent processing - files are automatically re-indexed by file watcher
			// Users will see the updated files when they search/use them
			// Verbose per-file logging is too noisy during active development
		}
	}
}

// RefreshDocument re-indexes a specific document
func (ds *DocumentStore) RefreshDocument(relativePath string) error {
	fullPath := filepath.Join(ds.sourcePath, relativePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		fmt.Printf("Document %s might have been deleted\n", relativePath)
		return nil
	}

	return ds.indexDocument(fullPath, info)
}

// ============================================================================
// INCREMENTAL INDEXING SUPPORT
// ============================================================================

// IndexedFileInfo stores metadata about an indexed file for incremental indexing
type IndexedFileInfo struct {
	Path         string
	LastModified int64 // Unix timestamp
}

// IndexState stores the complete index state for incremental indexing
type IndexState struct {
	StoreName   string           `json:"store_name"`
	SourcePath  string           `json:"source_path"`
	LastIndexed time.Time        `json:"last_indexed"`
	Files       map[string]int64 `json:"files"` // path -> last_modified timestamp
	TotalFiles  int              `json:"total_files"`
	TotalChunks int              `json:"total_chunks"`
}

// getIndexStatePath returns the path to the index state file
func (ds *DocumentStore) getIndexStatePath() string {
	// Store in .hector directory to keep it out of version control
	stateDir := filepath.Join(ds.sourcePath, ".hector")
	// Use store name in filename to support multiple stores
	return filepath.Join(stateDir, fmt.Sprintf("index_state_%s.json", ds.name))
}

// loadIndexState loads the index state from local file
func (ds *DocumentStore) loadIndexState() (map[string]int64, error) {
	statePath := ds.getIndexStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file yet - first run
			return make(map[string]int64), nil
		}
		return nil, fmt.Errorf("failed to read index state: %w", err)
	}

	var state IndexState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse index state: %w", err)
	}

	// Validate state matches current configuration
	if state.StoreName != ds.name || state.SourcePath != ds.sourcePath {
		log.Printf("Index state mismatch (store: %s vs %s, path: %s vs %s), rebuilding index",
			state.StoreName, ds.name, state.SourcePath, ds.sourcePath)
		return make(map[string]int64), nil
	}

	return state.Files, nil
}

// saveIndexState saves the index state to local file
func (ds *DocumentStore) saveIndexState(files map[string]int64, totalChunks int) error {
	statePath := ds.getIndexStatePath()
	stateDir := filepath.Dir(statePath)

	// Create .hector directory if it doesn't exist
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

	// Write atomically using temp file + rename
	tempPath := statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index state: %w", err)
	}

	if err := os.Rename(tempPath, statePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to save index state: %w", err)
	}

	return nil
}

// shouldReindexFile determines if a file needs to be re-indexed based on modification time
func (ds *DocumentStore) shouldReindexFile(path string, currentModTime time.Time, existingDocs map[string]int64) bool {
	// If incremental indexing is disabled, always reindex
	if !ds.config.IncrementalIndexing {
		return true
	}

	// If existingDocs is empty, reindex everything (first run)
	if len(existingDocs) == 0 {
		return true
	}

	// Get relative path
	relPath, err := filepath.Rel(ds.sourcePath, path)
	if err != nil {
		return true // If we can't get relative path, reindex to be safe
	}

	// Check if file exists in our index
	storedModTime, exists := existingDocs[relPath]
	if !exists {
		// New file, needs indexing
		return true
	}

	// Compare modification times (using Unix timestamps for consistency)
	currentUnix := currentModTime.Unix()
	if currentUnix > storedModTime {
		// File has been modified since last indexing
		return true
	}

	// File is unchanged, skip reindexing
	return false
}

// cleanupDeletedFiles removes documents from the index for files that no longer exist
func (ds *DocumentStore) cleanupDeletedFiles(ctx context.Context, existingDocs map[string]int64, foundFiles map[string]bool) error {
	if !ds.config.IncrementalIndexing || len(existingDocs) == 0 {
		// Only cleanup during incremental indexing
		return nil
	}

	deletedCount := 0
	for path := range existingDocs {
		if !foundFiles[path] {
			// File no longer exists in the directory, remove from index
			// Delete all chunks for this file
			filter := map[string]interface{}{
				"store_name": ds.name,
				"path":       path,
			}

			if err := ds.searchEngine.DeleteByFilter(ctx, filter); err != nil {
				log.Printf("Warning: Failed to delete indexed file %s: %v", path, err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("🗑️  Cleaned up %d deleted file(s) from index '%s'\n", deletedCount, ds.name)
	}

	return nil
}

// ============================================================================
// CLEANUP AND RESOURCE MANAGEMENT
// ============================================================================

// Close closes the document store and cleans up resources
func (ds *DocumentStore) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Stop file watching
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

// ============================================================================
// DOCUMENT STORE REGISTRY - ENHANCED MANAGEMENT
// ============================================================================

// DocumentStoreRegistry manages a global registry of document stores
type DocumentStoreRegistry struct {
	mu     sync.RWMutex
	stores map[string]*DocumentStore
}

// Global registry instance
var globalDocumentStoreRegistry *DocumentStoreRegistry

func init() {
	globalDocumentStoreRegistry = &DocumentStoreRegistry{
		stores: make(map[string]*DocumentStore),
	}
}

// RegisterDocumentStore registers a document store in the global registry
func RegisterDocumentStore(store *DocumentStore) {
	globalDocumentStoreRegistry.mu.Lock()
	defer globalDocumentStoreRegistry.mu.Unlock()

	globalDocumentStoreRegistry.stores[store.name] = store
}

// GetDocumentStoreFromRegistry retrieves a document store by name from registry
func GetDocumentStoreFromRegistry(name string) (*DocumentStore, bool) {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	store, exists := globalDocumentStoreRegistry.stores[name]
	return store, exists
}

// ListDocumentStoresFromRegistry returns all available document store names
func ListDocumentStoresFromRegistry() []string {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	var names []string
	for name := range globalDocumentStoreRegistry.stores {
		names = append(names, name)
	}
	return names
}

// GetDocumentStoreStats returns statistics for all registered stores
func GetDocumentStoreStats() map[string]interface{} {
	globalDocumentStoreRegistry.mu.RLock()
	defer globalDocumentStoreRegistry.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, store := range globalDocumentStoreRegistry.stores {
		stats[name] = store.GetStatus()
	}

	return stats
}

// UnregisterDocumentStore removes a document store from registry
func UnregisterDocumentStore(name string) {
	globalDocumentStoreRegistry.mu.Lock()
	defer globalDocumentStoreRegistry.mu.Unlock()

	if store, exists := globalDocumentStoreRegistry.stores[name]; exists {
		// Stop file watching if active
		if store.status.IsWatching {
			_ = store.StopWatching()
		}
		// Close the store
		store.Close()
		delete(globalDocumentStoreRegistry.stores, name)
	}
}

// InitializeDocumentStoresFromConfig creates and registers document stores from config
func InitializeDocumentStoresFromConfig(configs []config.DocumentStoreConfig, searchEngine *SearchEngine) error {
	if len(configs) == 0 {
		return nil
	}

	for _, config := range configs {
		store, err := NewDocumentStore(&config, searchEngine)
		if err != nil {
			fmt.Printf("Warning: Failed to create document store %s: %v\n", config.Name, err)
			continue
		}

		// Register the store
		RegisterDocumentStore(store)

		// Start indexing SYNCHRONOUSLY (wait for completion)
		if err := store.StartIndexing(); err != nil {
			fmt.Printf("Warning: Failed to index document store %s: %v\n", config.Name, err)
			continue
		}

		// Start file watching in background (after indexing completes)
		if store.config.WatchChanges {
			go func(s *DocumentStore, name string) {
				if err := s.StartWatching(); err != nil {
					fmt.Printf("Warning: Failed to start file watching for %s: %v\n", name, err)
				}
			}(store, config.Name)
		}
	}

	return nil
}
