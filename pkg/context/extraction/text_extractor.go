package extraction

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// TextExtractor handles plain text files
type TextExtractor struct{}

// NewTextExtractor creates a new text extractor
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// Name returns the extractor name
func (te *TextExtractor) Name() string {
	return "TextExtractor"
}

// CanExtract checks if this is a text file
func (te *TextExtractor) CanExtract(path string, mimeType string) bool {
	// If we already have MIME type, use it
	if mimeType != "" {
		return te.isTextMimeType(mimeType)
	}

	// Otherwise, detect from file
	return !te.isBinaryFile(path)
}

// Extract reads and cleans text content
func (te *TextExtractor) Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error) {
	startTime := time.Now()

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := te.cleanUTF8Content(string(contentBytes))
	if content == "" {
		return nil, nil
	}

	return &ExtractedContent{
		Content:          content,
		Title:            filepath.Base(path),
		Metadata:         make(map[string]string),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// Priority returns lower priority (1) so specific extractors can override
func (te *TextExtractor) Priority() int {
	return 1
}

// isBinaryFile checks if a file is binary by reading first 512 bytes
func (te *TextExtractor) isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil || n == 0 {
		return false
	}

	mimeType := http.DetectContentType(buffer[:n])
	return !te.isTextMimeType(mimeType)
}

// isTextMimeType checks if a MIME type is text-based
func (te *TextExtractor) isTextMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		strings.Contains(mimeType, "javascript")
}

// cleanUTF8Content validates and cleans UTF-8 content
func (te *TextExtractor) cleanUTF8Content(content string) string {
	if utf8.ValidString(content) {
		return content
	}

	cleaned := strings.ToValidUTF8(content, "")

	// If more than 50% was invalid, reject the file
	invalidRatio := float64(len(content)-len(cleaned)) / float64(len(content))
	if invalidRatio > 0.5 {
		return ""
	}

	return cleaned
}
