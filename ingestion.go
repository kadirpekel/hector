package hector

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	s3fs "github.com/fclairamb/afero-s3"
	miniofs "github.com/cpyun/afero-minio"
	gdrivefs "github.com/fclairamb/afero-gdrive"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/spf13/afero"
)

// ============================================================================
// SOURCE MANAGER
// ============================================================================

// SourceManager manages filesystem sources using Afero
type SourceManager struct {
	sources map[string]afero.Fs
	configs map[string]SourceConfig
}

// NewSourceManager creates a new source manager
func NewSourceManager(sources map[string]SourceConfig) *SourceManager {
	return &SourceManager{
		sources: make(map[string]afero.Fs),
		configs: sources,
	}
}

// GetSource returns a filesystem for the given source name
func (sm *SourceManager) GetSource(name string) (afero.Fs, error) {
	if fs, exists := sm.sources[name]; exists {
		return fs, nil
	}

	config, exists := sm.configs[name]
	if !exists {
		return nil, fmt.Errorf("source '%s' not found", name)
	}

	fs, err := sm.createFS(config)
	if err != nil {
		return nil, err
	}

	sm.sources[name] = fs
	return fs, nil
}

// ResolveSource resolves a source reference to a filesystem
func (sm *SourceManager) ResolveSource(sourceRef ModelIngestionSource) (afero.Fs, string, error) {
	var config SourceConfig
	var sourceName string

	if sourceRef.InlineSource != nil {
		// Use inline source
		config = *sourceRef.InlineSource
		sourceName = "inline"
	} else {
		// Use referenced source
		var exists bool
		config, exists = sm.configs[sourceRef.Source]
		if !exists {
			return nil, "", fmt.Errorf("source '%s' not found", sourceRef.Source)
		}
		sourceName = sourceRef.Source
	}

	fs, err := sm.createFS(config)
	if err != nil {
		return nil, "", err
	}

	return fs, sourceName, nil
}

// createFS creates a filesystem based on source configuration
func (sm *SourceManager) createFS(config SourceConfig) (afero.Fs, error) {
	switch config.Type {
	case "local":
		return afero.NewOsFs(), nil

	case "s3":
		// Create S3 filesystem using afero-s3
		// Note: This requires AWS credentials to be configured via environment variables
		// or AWS credential files. The bucket name is taken from the path.
		s3Fs := s3fs.NewFs(config.Path, nil) // nil uses default AWS session
		return s3Fs, nil

	case "minio":
		// Create MinIO filesystem using afero-minio
		// MinIO provides S3-compatible object storage
		ctx := context.Background()
		minioFs := miniofs.NewMinioFs(ctx, config.Path)
		return minioFs, nil

	case "gdrive":
		// Create Google Drive filesystem using afero-gdrive
		// Note: This requires Google Drive API credentials
		gdriveFs, err := gdrivefs.New(nil) // nil uses default HTTP client
		if err != nil {
			return nil, fmt.Errorf("failed to create Google Drive filesystem: %w", err)
		}
		return gdriveFs, nil

	default:
		return nil, fmt.Errorf("unsupported filesystem type: %s. Supported types: local, s3, minio, gdrive", config.Type)
	}
}

// ============================================================================
// MODEL MANAGER WITH INGESTION
// ============================================================================

// ModelManager manages models and their ingestion
type ModelManager struct {
	models  map[string]*ModelConfig
	sources *SourceManager
	agent   *Agent
}

// NewModelManager creates a new model manager
func NewModelManager(models []ModelConfig, sources *SourceManager, agent *Agent) *ModelManager {
	modelMap := make(map[string]*ModelConfig)
	for i := range models {
		modelMap[models[i].Name] = &models[i]
	}

	return &ModelManager{
		models:  modelMap,
		sources: sources,
		agent:   agent,
	}
}

// SyncModel syncs documents for a specific model
func (mm *ModelManager) SyncModel(modelName string) error {
	model, exists := mm.models[modelName]
	if !exists {
		return fmt.Errorf("model '%s' not found", modelName)
	}

	if model.Ingestion == nil {
		return fmt.Errorf("model '%s' has no ingestion configuration", modelName)
	}

	log.Printf("Syncing model '%s'...", modelName)

	for _, source := range model.Ingestion.Sources {
		err := mm.ingestFromSource(model, source)
		if err != nil {
			log.Printf("Failed to ingest from source %s: %v", source.Source, err)
		}
	}

	log.Printf("Completed syncing model '%s'", modelName)
	return nil
}

// SyncAllModels syncs all models that have ingestion configuration
func (mm *ModelManager) SyncAllModels() error {
	log.Println("Syncing all models...")

	for modelName, model := range mm.models {
		if model.Ingestion != nil {
			err := mm.SyncModel(modelName)
			if err != nil {
				log.Printf("Failed to sync model '%s': %v", modelName, err)
			}
		}
	}

	log.Println("Completed syncing all models")
	return nil
}

// ingestFromSource ingests documents from a specific source
func (mm *ModelManager) ingestFromSource(model *ModelConfig, source ModelIngestionSource) error {
	fs, sourceName, err := mm.sources.ResolveSource(source)
	if err != nil {
		return err
	}

	log.Printf("Ingesting from source '%s' with pattern '%s'", sourceName, source.Pattern)

	files, err := mm.discoverFiles(fs, source.Pattern, source.ExcludePatterns, sourceName)
	if err != nil {
		return err
	}

	log.Printf("Found %d files matching pattern", len(files))

	for _, filePath := range files {
		content, metadata, err := mm.processFile(fs, filePath, sourceName)
		if err != nil {
			log.Printf("Failed to process file %s: %v", filePath, err)
			continue // Skip problematic files
		}

		docID := uuid.New().String()
		err = mm.agent.UpsertDocument(model.Name, docID, map[string]interface{}{
			"content":  content,
			"metadata": metadata,
		})

		if err != nil {
			log.Printf("Failed to store document %s: %v", filePath, err)
		} else {
			log.Printf("Successfully ingested: %s", filepath.Base(filePath))
		}
	}

	return nil
}

// discoverFiles discovers files matching the pattern
func (mm *ModelManager) discoverFiles(fs afero.Fs, pattern string, excludePatterns []string, sourceName string) ([]string, error) {
	var files []string

	// For now, we'll use a simple approach with local filesystem
	// In the future, we'll need to implement proper pattern matching for different filesystems
	if osFs, ok := fs.(*afero.OsFs); ok {
		// Get the source path from the source manager
		sourceConfig, exists := mm.sources.configs[sourceName]
		if !exists {
			return files, fmt.Errorf("source '%s' not found", sourceName)
		}

		sourcePath := sourceConfig.Path
		log.Printf("Walking directory: %s", sourcePath)

		// Use the underlying OS filesystem for walking
		err := afero.Walk(osFs, sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				// Check if file matches pattern
				// Convert **/*.txt to *.txt for filepath.Match
				simplePattern := pattern
				if strings.HasPrefix(pattern, "**/") {
					simplePattern = pattern[3:] // Remove "**/" prefix
				}

				if matched, _ := filepath.Match(simplePattern, filepath.Base(path)); matched {
					// Check if file should be excluded
					excluded := false
					for _, excludePattern := range excludePatterns {
						simpleExcludePattern := excludePattern
						if strings.HasPrefix(excludePattern, "**/") {
							simpleExcludePattern = excludePattern[3:] // Remove "**/" prefix
						}
						if matched, _ := filepath.Match(simpleExcludePattern, filepath.Base(path)); matched {
							excluded = true
							break
						}
					}

					if !excluded {
						files = append(files, path)
					}
				}
			}
			return nil
		})

		return files, err
	}

	// For other filesystems, we'll implement pattern matching later
	return files, fmt.Errorf("pattern matching not yet implemented for filesystem type")
}

// processFile processes a file and extracts content and metadata
func (mm *ModelManager) processFile(fs afero.Fs, filePath string, sourceName string) (string, map[string]interface{}, error) {
	// Read file content
	content, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return "", nil, err
	}

	// Get file info
	info, err := fs.Stat(filePath)
	if err != nil {
		return "", nil, err
	}

	// Extract file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	// Parse content based on file type
	var parsedContent string
	var parseErr error

	switch ext {
	case ".pdf":
		parsedContent, parseErr = mm.parsePDF(content)
	case ".docx":
		parsedContent, parseErr = mm.parseWord(content)
	case ".txt", ".md":
		parsedContent = string(content)
	default:
		parsedContent = string(content)
	}

	// If parsing failed, fall back to raw content
	if parseErr != nil {
		log.Printf("Failed to parse %s file %s: %v, using raw content", ext, filePath, parseErr)
		parsedContent = string(content)
	}

	// Extract metadata
	metadata := map[string]interface{}{
		"source":      sourceName,
		"path":        filePath,
		"filename":    filepath.Base(filePath),
		"size":        info.Size(),
		"modified":    info.ModTime().Format(time.RFC3339),
		"ingested_at": time.Now().Format(time.RFC3339),
		"extension":   ext[1:], // Remove the dot
		"content":     parsedContent,
	}

	return parsedContent, metadata, nil
}

// parsePDF extracts text content from PDF files
func (mm *ModelManager) parsePDF(content []byte) (string, error) {
	// Create a temporary file for PDF parsing
	tmpFile, err := os.CreateTemp("", "hector_pdf_*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		return "", err
	}
	tmpFile.Close()

	// Open PDF file
	f, r, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Extract text from all pages
	var textContent strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			log.Printf("Failed to extract text from page %d: %v", i, err)
			continue
		}

		textContent.WriteString(content)
		textContent.WriteString("\n")
	}

	return textContent.String(), nil
}

// parseWord extracts text content from Word (.docx) files
func (mm *ModelManager) parseWord(content []byte) (string, error) {
	// Create a temporary file for Word parsing
	tmpFile, err := os.CreateTemp("", "hector_word_*.docx")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		return "", err
	}
	tmpFile.Close()

	// Open Word file
	doc, err := docx.ReadDocxFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer doc.Close()

	// Extract text content
	textContent := doc.Editable().GetContent()
	return textContent, nil
}

// GetModelStatus returns the status of a model
func (mm *ModelManager) GetModelStatus(modelName string) (map[string]interface{}, error) {
	model, exists := mm.models[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}

	status := map[string]interface{}{
		"name":          model.Name,
		"collection":    model.Collection,
		"has_ingestion": model.Ingestion != nil,
	}

	if model.Ingestion != nil {
		status["auto_sync"] = model.Ingestion.AutoSync
		status["sync_interval"] = model.Ingestion.SyncInterval
		status["source_count"] = len(model.Ingestion.Sources)
	}

	return status, nil
}

// ListModels returns a list of all models
func (mm *ModelManager) ListModels() []string {
	var models []string
	for name := range mm.models {
		models = append(models, name)
	}
	return models
}
