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
	"unicode/utf8"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
)

const (
	DefaultMaxFileSize = 5 * 1024 * 1024

	DefaultUpdateChannelSize = 100

	DefaultFileWatchTimeout = 10 * time.Second

	MaxConcurrentIndexing = 3
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
	sourcePath   string

	nativeParsers *NativeParserRegistry

	watcher       *fsnotify.Watcher
	updateChannel chan DocumentUpdate
	ctx           context.Context
	cancel        context.CancelFunc

	indexingSemaphore chan struct{}
}

func NewDocumentStore(storeConfig *config.DocumentStoreConfig, searchEngine *SearchEngine) (*DocumentStore, error) {
	if storeConfig == nil {
		return nil, NewDocumentStoreError("", "NewDocumentStore", "store config is required", "", nil)
	}
	if searchEngine == nil {
		return nil, NewDocumentStoreError(storeConfig.Name, "NewDocumentStore", "search engine is required", "", nil)
	}

	if storeConfig.MaxFileSize == 0 {
		storeConfig.MaxFileSize = DefaultMaxFileSize
	}

	if _, err := os.Stat(storeConfig.Path); os.IsNotExist(err) {
		return nil, NewDocumentStoreError(storeConfig.Name, "NewDocumentStore", "source path does not exist", storeConfig.Path, err)
	}

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

	if storeConfig.WatchChanges {
		if err := store.initializeWatcher(); err != nil {
			cancel()
			return nil, NewDocumentStoreError(storeConfig.Name, "NewDocumentStore", "failed to initialize watcher", "", err)
		}
	}

	return store, nil
}

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

	fmt.Printf("Indexing document store '%s' from: %s\n", ds.name, ds.sourcePath)

	switch ds.config.Source {
	case "directory":
		return ds.indexDirectory()
	case "git":
		return ds.indexGitRepository()
	default:
		return NewDocumentStoreError(ds.name, "StartIndexing", "unsupported source type", ds.config.Source, nil)
	}
}

func (ds *DocumentStore) indexDirectory() error {
	ctx := context.Background()

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

	foundFiles := make(map[string]bool)
	indexedFiles := make(map[string]int64)
	var filesMu sync.Mutex

	err = filepath.Walk(ds.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			atomic.AddInt32(&failCount, 1)
			log.Printf("Warning: Failed to access %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if info.Size() == 0 {
			return nil
		}

		if ds.shouldExclude(path) || !ds.shouldInclude(path) {
			return nil
		}

		if info.Size() > ds.config.MaxFileSize {
			return nil
		}

		relPath, _ := filepath.Rel(ds.sourcePath, path)
		filesMu.Lock()
		foundFiles[relPath] = true
		filesMu.Unlock()

		if !ds.shouldReindexFile(path, info.ModTime(), existingDocs) {
			atomic.AddInt32(&skippedCount, 1)
			return nil
		}

		ds.indexingSemaphore <- struct{}{}
		indexedCount.Add(1)
		go func(p string, i os.FileInfo) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic while indexing %s: %v", p, r)
					atomic.AddInt32(&failCount, 1)
				}
				<-ds.indexingSemaphore
				indexedCount.Done()
			}()

			if err := ds.indexDocument(p, i); err != nil {
				atomic.AddInt32(&failCount, 1)
				log.Printf("Warning: Failed to index %s: %v", p, err)
			} else {
				atomic.AddInt32(&successCount, 1)

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

	indexedCount.Wait()

	if ds.config.IncrementalIndexing {
		if err := ds.cleanupDeletedFiles(ctx, existingDocs, foundFiles); err != nil {
			log.Printf("Warning: Cleanup of deleted files failed: %v", err)
		}
	}

	ds.mu.Lock()
	ds.status.DocumentCount = int(successCount)
	ds.mu.Unlock()

	if ds.config.IncrementalIndexing {

		finalState := make(map[string]int64)

		for path, ts := range existingDocs {
			if foundFiles[path] {
				finalState[path] = ts
			}
		}

		for path, ts := range indexedFiles {
			finalState[path] = ts
		}

		if err := ds.saveIndexState(finalState, len(finalState)*3); err != nil {
			log.Printf("Warning: Failed to save index state: %v", err)
		}
	}

	if ds.config.IncrementalIndexing && skippedCount > 0 {
		fmt.Printf("Document store '%s' indexed: %d new/modified, %d unchanged, %d errors\n",
			ds.name, successCount, skippedCount, failCount)
	} else {
		fmt.Printf("Document store '%s' indexed: %d documents (%d errors)\n",
			ds.name, successCount, failCount)
	}

	if ds.config.WatchChanges {
		fmt.Printf("File watching enabled - changes will be automatically indexed\n")
	}

	return nil
}

func (ds *DocumentStore) indexGitRepository() error {

	if !ds.isGitRepository(ds.sourcePath) {
		return fmt.Errorf("path %s is not a git repository", ds.sourcePath)
	}

	cmd := exec.Command("git", "ls-files")
	cmd.Dir = ds.sourcePath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list git files: %w", err)
	}

	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if file == "" {
			continue
		}

		fullPath := filepath.Join(ds.sourcePath, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			if err := ds.indexDocument(fullPath, info); err != nil {

				fmt.Printf("Warning: failed to index file %s: %v\n", fullPath, err)
			}
		}
	}

	return nil
}

func (ds *DocumentStore) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func (ds *DocumentStore) indexDocument(path string, info os.FileInfo) error {
	relPath, _ := filepath.Rel(ds.sourcePath, path)

	content, err := ds.extractContentWithPlugins(path, info)
	if err != nil {

		contentBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return NewDocumentStoreError(ds.name, "indexDocument", "failed to read file", path, readErr)
		}
		content = string(contentBytes)
	}

	content = ds.cleanUTF8Content(content)

	if content == "" {
		return nil
	}

	doc := ds.createDocument(relPath, info, content)

	ds.extractMetadata(doc)

	metadata := ds.prepareVectorMetadata(doc)

	chunks := ds.chunkContent(doc.Content, 800)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultIndexingTimeout)
	defer cancel()

	for i, chunk := range chunks {

		chunkKey := fmt.Sprintf("%s:chunk:%d", relPath, i)
		hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", ds.name, chunkKey)))
		chunkID := uuid.NewMD5(uuid.Nil, hash[:]).String()

		chunkMetadata := make(map[string]interface{})
		for k, v := range metadata {
			chunkMetadata[k] = v
		}
		chunkMetadata["chunk_index"] = i
		chunkMetadata["chunk_total"] = len(chunks)
		chunkMetadata["start_line"] = chunk.StartLine
		chunkMetadata["end_line"] = chunk.EndLine
		chunkMetadata["content"] = chunk.Content

		if err := ds.searchEngine.IngestDocument(ctx, chunkID, chunk.Content, chunkMetadata); err != nil {
			return NewDocumentStoreError(ds.name, "indexDocument", "failed to ingest chunk", path, err)
		}
	}

	return nil
}

func (ds *DocumentStore) extractContentWithPlugins(path string, info os.FileInfo) (string, error) {

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

		if result.ProcessingTimeMs > 0 {
			log.Printf("Parsed %s with native parser (%dms)", path, result.ProcessingTimeMs)
		}

		return result.Content, nil
	}

	return "", fmt.Errorf("no native parser available for text files - use plain text reading")
}

type ContentChunk struct {
	Content   string
	StartLine int
	EndLine   int
}

func (ds *DocumentStore) chunkContent(content string, targetSize int) []ContentChunk {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

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

	if currentChunk.Len() > 0 {
		chunks = append(chunks, ContentChunk{
			Content:   currentChunk.String(),
			StartLine: chunkStartLine,
			EndLine:   totalLines,
		})
	}

	return chunks
}

func (ds *DocumentStore) cleanUTF8Content(content string) string {

	if utf8.ValidString(content) {
		return content
	}

	cleaned := strings.ToValidUTF8(content, "")

	invalidRatio := float64(len(content)-len(cleaned)) / float64(len(content))
	if invalidRatio > 0.5 {
		return ""
	}

	return cleaned
}

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

	doc.Type, doc.Language = ds.detectTypeAndLanguage(relPath)

	doc.Title = ds.extractTitle(doc)

	return doc
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

func (ds *DocumentStore) extractTitle(doc *Document) string {
	switch doc.Type {
	case DocumentTypeMarkdown:
		return ds.extractMarkdownTitle(doc.Content)
	case DocumentTypeCode:
		return doc.Name
	default:
		return doc.Name
	}
}

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

func (ds *DocumentStore) extractGoMetadata(doc *Document) {
	lines := strings.Split(doc.Content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "func ") {
			if funcName := ds.extractGoFunctionName(line); funcName != "" {
				doc.Functions = append(doc.Functions, funcName)
			}
		}

		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			if structName := ds.extractGoStructName(line); structName != "" {
				doc.Structs = append(doc.Structs, structName)
			}
		}

		if strings.HasPrefix(line, "import ") || (strings.Contains(line, `"`) && strings.Contains(line, "/")) {
			if importPath := ds.extractGoImport(line); importPath != "" {
				doc.Imports = append(doc.Imports, importPath)
			}
		}
	}
}

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

func (ds *DocumentStore) extractGoStructName(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

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

func (ds *DocumentStore) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {

	filter := map[string]interface{}{
		"store_name": ds.name,
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

func (ds *DocumentStore) shouldExclude(path string) bool {

	relPath, err := filepath.Rel(ds.sourcePath, path)
	if err != nil {
		relPath = path
	}

	normalizedPath := filepath.ToSlash(relPath)

	for _, pattern := range ds.config.ExcludePatterns {

		normalizedPattern := filepath.ToSlash(pattern)

		if matched, err := filepath.Match(normalizedPattern, normalizedPath); err == nil && matched {
			return true
		}

		if strings.HasPrefix(normalizedPattern, "**/") && strings.HasSuffix(normalizedPattern, "/**") {
			dirName := strings.Trim(normalizedPattern, "*/")

			if strings.Contains("/"+normalizedPath+"/", "/"+dirName+"/") {
				return true
			}
		}

		if strings.HasPrefix(normalizedPattern, "*.") {
			ext := strings.TrimPrefix(normalizedPattern, "*")
			if strings.HasSuffix(normalizedPath, ext) {
				return true
			}
		}

		if strings.Contains(normalizedPattern, "**") {

			if strings.HasPrefix(normalizedPattern, "**/") {
				simplePattern := strings.TrimPrefix(normalizedPattern, "**/")
				if matched, err := filepath.Match(simplePattern, filepath.Base(normalizedPath)); err == nil && matched {
					return true
				}
			}
		}

		if !strings.Contains(normalizedPattern, "*") {

			if strings.Contains(normalizedPath, normalizedPattern) {
				return true
			}
		}
	}

	return false
}

func (ds *DocumentStore) shouldInclude(path string) bool {
	if len(ds.config.IncludePatterns) == 0 {
		return true
	}

	relPath, err := filepath.Rel(ds.sourcePath, path)
	if err != nil {
		relPath = path
	}

	normalizedPath := filepath.ToSlash(relPath)

	for _, pattern := range ds.config.IncludePatterns {

		normalizedPattern := filepath.ToSlash(pattern)

		if pattern == "*" {
			return true
		}

		if matched, err := filepath.Match(normalizedPattern, normalizedPath); err == nil && matched {
			return true
		}

		if strings.HasPrefix(normalizedPattern, "**/") && strings.HasSuffix(normalizedPattern, "/**") {
			dirName := strings.Trim(normalizedPattern, "*/")

			if strings.Contains("/"+normalizedPath+"/", "/"+dirName+"/") {
				return true
			}
		}

		if strings.HasPrefix(normalizedPattern, "*.") {
			ext := strings.TrimPrefix(normalizedPattern, "*")
			if strings.HasSuffix(normalizedPath, ext) {
				return true
			}
		}

		if strings.Contains(normalizedPattern, "**") {

			if strings.HasPrefix(normalizedPattern, "**/") {
				simplePattern := strings.TrimPrefix(normalizedPattern, "**/")
				if matched, err := filepath.Match(simplePattern, filepath.Base(normalizedPath)); err == nil && matched {
					return true
				}
			}
		}

		if !strings.Contains(normalizedPattern, "*") {

			if strings.Contains(normalizedPath, normalizedPattern) {
				return true
			}
		}
	}
	return false
}

func (ds *DocumentStore) generateDocumentID(path string) string {
	fullPath := fmt.Sprintf("%s:%s", ds.name, path)
	hash := md5.Sum([]byte(fullPath))
	return uuid.NewMD5(uuid.Nil, hash[:]).String()
}

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

	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	return metadata
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
				log.Printf("Warning: Failed to watch directory %s: %v", path, err)
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
			log.Printf("File watcher error in store %s: %v", ds.name, err)
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
		log.Printf("Update channel full for store %s, dropping update for %s", ds.name, event.Name)
	}
}

func (ds *DocumentStore) processUpdates() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case _, ok := <-ds.updateChannel:
			if !ok {
				return
			}

		}
	}
}

func (ds *DocumentStore) RefreshDocument(relativePath string) error {
	fullPath := filepath.Join(ds.sourcePath, relativePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		fmt.Printf("Document %s might have been deleted\n", relativePath)
		return nil
	}

	return ds.indexDocument(fullPath, info)
}

type IndexedFileInfo struct {
	Path         string
	LastModified int64
}

type IndexState struct {
	StoreName   string           `json:"store_name"`
	SourcePath  string           `json:"source_path"`
	LastIndexed time.Time        `json:"last_indexed"`
	Files       map[string]int64 `json:"files"`
	TotalFiles  int              `json:"total_files"`
	TotalChunks int              `json:"total_chunks"`
}

func (ds *DocumentStore) getIndexStatePath() string {

	stateDir := filepath.Join(ds.sourcePath, ".hector")

	return filepath.Join(stateDir, fmt.Sprintf("index_state_%s.json", ds.name))
}

func (ds *DocumentStore) loadIndexState() (map[string]int64, error) {
	statePath := ds.getIndexStatePath()

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {

			return make(map[string]int64), nil
		}
		return nil, fmt.Errorf("failed to read index state: %w", err)
	}

	var state IndexState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse index state: %w", err)
	}

	if state.StoreName != ds.name || state.SourcePath != ds.sourcePath {
		log.Printf("Index state mismatch (store: %s vs %s, path: %s vs %s), rebuilding index",
			state.StoreName, ds.name, state.SourcePath, ds.sourcePath)
		return make(map[string]int64), nil
	}

	return state.Files, nil
}

func (ds *DocumentStore) saveIndexState(files map[string]int64, totalChunks int) error {
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

func (ds *DocumentStore) shouldReindexFile(path string, currentModTime time.Time, existingDocs map[string]int64) bool {

	if !ds.config.IncrementalIndexing {
		return true
	}

	if len(existingDocs) == 0 {
		return true
	}

	relPath, err := filepath.Rel(ds.sourcePath, path)
	if err != nil {
		return true
	}

	storedModTime, exists := existingDocs[relPath]
	if !exists {

		return true
	}

	currentUnix := currentModTime.Unix()
	return currentUnix > storedModTime
}

func (ds *DocumentStore) cleanupDeletedFiles(ctx context.Context, existingDocs map[string]int64, foundFiles map[string]bool) error {
	if !ds.config.IncrementalIndexing || len(existingDocs) == 0 {

		return nil
	}

	deletedCount := 0
	for path := range existingDocs {
		if !foundFiles[path] {

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
		fmt.Printf("Cleaned up %d deleted file(s) from index '%s'\n", deletedCount, ds.name)
	}

	return nil
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

func InitializeDocumentStoresFromConfig(configs []*config.DocumentStoreConfig, searchEngine *SearchEngine) error {
	if len(configs) == 0 {
		return nil
	}

	for _, config := range configs {
		store, err := NewDocumentStore(config, searchEngine)
		if err != nil {
			fmt.Printf("Warning: Failed to create document store %s: %v\n", config.Name, err)
			continue
		}

		RegisterDocumentStore(store)

		if err := store.StartIndexing(); err != nil {
			fmt.Printf("Warning: Failed to index document store %s: %v\n", config.Name, err)
			continue
		}

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
