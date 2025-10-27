package extraction

import (
	"context"
	"path/filepath"
	"strings"
)

// DocumentParserPlugin interface for gRPC plugins
type DocumentParserPlugin interface {
	Name() string
	ParseDocument(ctx context.Context, filePath string, fileSize int64) (*PluginParseResult, error)
	GetSupportedExtensions() ([]string, error)
}

// PluginParseResult represents the result from a plugin parser
type PluginParseResult struct {
	Success          bool
	Content          string
	Title            string
	Author           string
	Metadata         map[string]string
	ProcessingTimeMs int64
}

// PluginExtractor handles files using external gRPC plugins
type PluginExtractor struct {
	plugin              DocumentParserPlugin
	supportedExtensions map[string]bool
}

// NewPluginExtractor creates a new plugin-based extractor
func NewPluginExtractor(plugin DocumentParserPlugin) (*PluginExtractor, error) {
	// Get supported extensions from plugin
	extensions, err := plugin.GetSupportedExtensions()
	if err != nil {
		return nil, err
	}

	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[strings.ToLower(ext)] = true
	}

	return &PluginExtractor{
		plugin:              plugin,
		supportedExtensions: extMap,
	}, nil
}

// Name returns the extractor name
func (pe *PluginExtractor) Name() string {
	return "PluginExtractor:" + pe.plugin.Name()
}

// CanExtract checks if this plugin can handle the file
func (pe *PluginExtractor) CanExtract(path string, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return pe.supportedExtensions[ext]
}

// Extract uses the plugin to extract content
func (pe *PluginExtractor) Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error) {
	result, err := pe.plugin.ParseDocument(ctx, path, fileSize)
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
		ProcessingTimeMs: result.ProcessingTimeMs,
	}, nil
}

// Priority returns high priority (10) so plugins override native extractors
func (pe *PluginExtractor) Priority() int {
	return 10
}
