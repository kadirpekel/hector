package context

import (
	"context"

	"github.com/kadirpekel/hector/pkg/context/extraction"
)

// nativeParserAdapter adapts NativeParserRegistry to extraction.NativeParser interface
type nativeParserAdapter struct {
	registry *NativeParserRegistry
}

// newNativeParserAdapter creates an adapter
func newNativeParserAdapter(registry *NativeParserRegistry) *nativeParserAdapter {
	return &nativeParserAdapter{registry: registry}
}

// ParseDocument adapts the call and converts the result type
func (a *nativeParserAdapter) ParseDocument(ctx context.Context, filePath string, fileSize int64) (*extraction.NativeParseResult, error) {
	result, err := a.registry.ParseDocument(ctx, filePath, fileSize)
	if err != nil {
		return nil, err
	}

	// Convert NativeParserResult to extraction.NativeParseResult
	return &extraction.NativeParseResult{
		Success:          result.Success,
		Content:          result.Content,
		Title:            result.Title,
		Author:           result.Author,
		Metadata:         result.Metadata,
		Error:            result.Error,
		ProcessingTimeMs: result.ProcessingTimeMs,
	}, nil
}
