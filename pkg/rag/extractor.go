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

package rag

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ContentExtractor defines the interface for extracting content from files.
//
// Direct port from legacy pkg/context/extraction/extractor.go
type ContentExtractor interface {
	// Name returns the extractor name for logging/debugging.
	Name() string

	// CanExtract determines if this extractor can handle the given file.
	CanExtract(path string, mimeType string) bool

	// Extract extracts content from the file.
	Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error)

	// Priority returns the priority (higher = preferred when multiple extractors match).
	Priority() int
}

// ExtractedContent represents extracted file content with metadata.
//
// Direct port from legacy pkg/context/extraction/extractor.go
type ExtractedContent struct {
	Content          string            // The extracted text content
	Title            string            // Document title (if available)
	Author           string            // Document author (if available)
	Metadata         map[string]string // Additional metadata
	ProcessingTimeMs int64             // Time taken to extract
	ExtractorName    string            // Name of extractor used
}

// ExtractorRegistry manages multiple content extractors.
//
// Direct port from legacy pkg/context/extraction/extractor.go
type ExtractorRegistry struct {
	extractors []ContentExtractor
}

// NewExtractorRegistry creates a new extractor registry.
func NewExtractorRegistry() *ExtractorRegistry {
	reg := &ExtractorRegistry{
		extractors: make([]ContentExtractor, 0),
	}
	// Register default text extractor
	reg.Register(NewTextExtractor())
	return reg
}

// Register adds an extractor to the registry.
func (r *ExtractorRegistry) Register(extractor ContentExtractor) {
	r.extractors = append(r.extractors, extractor)

	// Sort by priority (higher first)
	sort.Slice(r.extractors, func(i, j int) bool {
		return r.extractors[i].Priority() > r.extractors[j].Priority()
	})
}

// Extract tries to extract content using the best available extractor.
// Adapts the document-based interface for store.go compatibility.
func (r *ExtractorRegistry) Extract(ctx context.Context, doc Document) (*ExtractedContent, error) {
	mimeType := doc.MimeType
	path := doc.SourcePath

	// If we have raw content, use it directly (for SQL/API sources)
	if doc.Content != "" && !isFilePath(path) {
		return &ExtractedContent{
			Content:       doc.Content,
			Title:         doc.Title,
			Metadata:      make(map[string]string),
			ExtractorName: "direct",
		}, nil
	}

	return r.ExtractContent(ctx, path, mimeType, doc.Size)
}

// ExtractContent tries to extract content using the best available extractor.
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

// GetExtractors returns all registered extractors (for debugging).
func (r *ExtractorRegistry) GetExtractors() []ContentExtractor {
	return r.extractors
}

// HasExtractorForFile checks if any extractor can handle the given file.
// This is useful for determining if a file can be indexed before attempting extraction.
func (r *ExtractorRegistry) HasExtractorForFile(path string, mimeType string) bool {
	for _, extractor := range r.extractors {
		if extractor.CanExtract(path, mimeType) {
			return true
		}
	}
	return false
}

// isFilePath checks if the given path looks like a file path.
func isFilePath(path string) bool {
	if path == "" {
		return false
	}
	// Check if it looks like a file path (has extension or path separator)
	return strings.Contains(path, string(os.PathSeparator)) ||
		strings.Contains(path, "/") ||
		filepath.Ext(path) != ""
}

// TextExtractor handles plain text files.
//
// Direct port from legacy pkg/context/extraction/text_extractor.go
type TextExtractor struct{}

// NewTextExtractor creates a new text extractor.
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// Name returns the extractor name.
func (te *TextExtractor) Name() string {
	return "TextExtractor"
}

// CanExtract checks if this is a text file.
func (te *TextExtractor) CanExtract(path string, mimeType string) bool {
	// If we already have MIME type, use it
	if mimeType != "" {
		return te.isTextMimeType(mimeType)
	}

	// Otherwise, detect from file
	return !te.isBinaryFile(path)
}

// Extract reads and cleans text content.
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

// Priority returns lower priority (1) so specific extractors can override.
func (te *TextExtractor) Priority() int {
	return 1
}

// isBinaryFile checks if a file is binary by reading first 512 bytes.
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

// isTextMimeType checks if a MIME type is text-based.
func (te *TextExtractor) isTextMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		strings.Contains(mimeType, "javascript")
}

// cleanUTF8Content validates and cleans UTF-8 content.
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

// Ensure TextExtractor implements ContentExtractor.
var _ ContentExtractor = (*TextExtractor)(nil)
