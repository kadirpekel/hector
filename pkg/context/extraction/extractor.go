package extraction

import (
	"context"
	"fmt"
	"sort"
)

// ContentExtractor defines the interface for extracting content from files
type ContentExtractor interface {
	// Name returns the extractor name for logging/debugging
	Name() string

	// CanExtract determines if this extractor can handle the given file
	CanExtract(path string, mimeType string) bool

	// Extract extracts content from the file
	Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error)

	// Priority returns the priority (higher = preferred when multiple extractors match)
	Priority() int
}

// ExtractedContent represents extracted file content with metadata
type ExtractedContent struct {
	Content          string            // The extracted text content
	Title            string            // Document title (if available)
	Author           string            // Document author (if available)
	Metadata         map[string]string // Additional metadata
	ProcessingTimeMs int64             // Time taken to extract
	ExtractorName    string            // Name of extractor used
}

// ExtractorRegistry manages multiple content extractors
type ExtractorRegistry struct {
	extractors []ContentExtractor
}

// NewExtractorRegistry creates a new extractor registry
func NewExtractorRegistry() *ExtractorRegistry {
	return &ExtractorRegistry{
		extractors: make([]ContentExtractor, 0),
	}
}

// Register adds an extractor to the registry
func (r *ExtractorRegistry) Register(extractor ContentExtractor) {
	r.extractors = append(r.extractors, extractor)

	// Sort by priority (higher first)
	sort.Slice(r.extractors, func(i, j int) bool {
		return r.extractors[i].Priority() > r.extractors[j].Priority()
	})
}

// ExtractContent tries to extract content using the best available extractor
func (r *ExtractorRegistry) ExtractContent(ctx context.Context, path string, mimeType string, fileSize int64) (*ExtractedContent, error) {
	for _, extractor := range r.extractors {
		if extractor.CanExtract(path, mimeType) {
			content, err := extractor.Extract(ctx, path, fileSize)
			if err != nil {
				// Try next extractor
				continue
			}
			if content != nil {
				content.ExtractorName = extractor.Name()
				return content, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable extractor found for file: %s (mime: %s)", path, mimeType)
}

// GetExtractors returns all registered extractors (for debugging)
func (r *ExtractorRegistry) GetExtractors() []ContentExtractor {
	return r.extractors
}
