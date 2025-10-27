package extraction

import (
	"context"
	"path/filepath"
	"strings"
	"time"
)

// NativeParser interface for parsing binary documents
type NativeParser interface {
	ParseDocument(ctx context.Context, filePath string, fileSize int64) (*NativeParseResult, error)
}

// NativeParseResult represents the result from a native parser
type NativeParseResult struct {
	Success          bool
	Content          string
	Title            string
	Author           string
	Metadata         map[string]string
	Error            string
	ProcessingTimeMs int64
}

// BinaryExtractor handles binary files like PDF, DOCX, XLSX using native parsers
type BinaryExtractor struct {
	nativeParsers NativeParser
}

// NewBinaryExtractor creates a new binary extractor
func NewBinaryExtractor(nativeParsers NativeParser) *BinaryExtractor {
	return &BinaryExtractor{
		nativeParsers: nativeParsers,
	}
}

// Name returns the extractor name
func (be *BinaryExtractor) Name() string {
	return "BinaryExtractor"
}

// CanExtract checks if this extractor can handle the file
func (be *BinaryExtractor) CanExtract(path string, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
	}
	return binaryExtensions[ext]
}

// Extract uses native parsers to extract content from binary files
func (be *BinaryExtractor) Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error) {
	startTime := time.Now()

	result, err := be.nativeParsers.ParseDocument(ctx, path, fileSize)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, nil
	}

	metadata := make(map[string]string)
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	return &ExtractedContent{
		Content:          result.Content,
		Title:            result.Title,
		Author:           result.Author,
		Metadata:         metadata,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// Priority returns medium priority (5)
func (be *BinaryExtractor) Priority() int {
	return 5
}
