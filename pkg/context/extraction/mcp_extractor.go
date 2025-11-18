package extraction

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

// ToolCaller is a minimal interface for calling tools without creating import cycles
// This allows MCP extractors to work with any tool registry implementation
type ToolCaller interface {
	GetTool(name string) (Tool, error)
}

// Tool is a minimal interface for executing tools
type Tool interface {
	GetInfo() ToolInfo
	Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)
}

// ToolInfo contains information about a tool
type ToolInfo struct {
	Name        string
	Description string
	Parameters  []ToolParameter
}

// ToolParameter describes a tool parameter
type ToolParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// ToolResult contains the result of tool execution
type ToolResult struct {
	Success  bool
	Content  string
	Error    string
	Metadata interface{}
}

// MCPExtractor handles document parsing via MCP tools
// This allows using any MCP service (Docling, etc.) for document parsing
type MCPExtractor struct {
	toolCaller      ToolCaller
	parserToolNames []string // List of MCP tool names that can parse documents
	supportedExts   map[string]bool
	priority        int
}

// MCPExtractorConfig configures an MCP extractor
type MCPExtractorConfig struct {
	ToolCaller      ToolCaller
	ParserToolNames []string // Tool names to try (e.g., ["parse_document", "docling_parse"])
	SupportedExts   []string // File extensions this extractor handles (empty = all)
	Priority        int      // Priority (higher = preferred)
}

// NewMCPExtractor creates a new MCP-based extractor
func NewMCPExtractor(config MCPExtractorConfig) (*MCPExtractor, error) {
	if config.ToolCaller == nil {
		return nil, fmt.Errorf("tool caller is required")
	}

	if len(config.ParserToolNames) == 0 {
		return nil, fmt.Errorf("at least one parser tool name is required")
	}

	priority := config.Priority
	if priority == 0 {
		priority = 8 // Higher than BinaryExtractor (5) but lower than PluginExtractor (10)
	}

	extMap := make(map[string]bool)
	for _, ext := range config.SupportedExts {
		extMap[strings.ToLower(ext)] = true
	}

	return &MCPExtractor{
		toolCaller:      config.ToolCaller,
		parserToolNames: config.ParserToolNames,
		supportedExts:   extMap,
		priority:        priority,
	}, nil
}

// Name returns the extractor name
func (e *MCPExtractor) Name() string {
	return fmt.Sprintf("MCPExtractor:%s", strings.Join(e.parserToolNames, ","))
}

// CanExtract checks if this extractor can handle the file
func (e *MCPExtractor) CanExtract(path string, mimeType string) bool {
	// If no specific extensions configured, try all files
	if len(e.supportedExts) == 0 {
		// Check if any parser tool is available
		return e.hasParserTool()
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !e.supportedExts[ext] {
		return false
	}

	// Also check if parser tool is available
	return e.hasParserTool()
}

// hasParserTool checks if at least one parser tool is available
func (e *MCPExtractor) hasParserTool() bool {
	for _, toolName := range e.parserToolNames {
		if _, err := e.toolCaller.GetTool(toolName); err == nil {
			return true
		}
	}
	return false
}

// Extract uses MCP tools to extract content from files
func (e *MCPExtractor) Extract(ctx context.Context, path string, fileSize int64) (*ExtractedContent, error) {
	startTime := time.Now()

	// Try each parser tool in order
	for _, toolName := range e.parserToolNames {
		tool, err := e.toolCaller.GetTool(toolName)
		if err != nil {
			// Tool not available, try next
			continue
		}

		// Prepare arguments for the MCP tool
		// Common MCP document parser tools expect: file_path, file_path, or path
		args := make(map[string]interface{})

		// Try common parameter names
		if toolInfo := tool.GetInfo(); len(toolInfo.Parameters) > 0 {
			// Use the first required parameter name, or common names
			for _, param := range toolInfo.Parameters {
				if param.Required {
					args[param.Name] = path
					break
				}
			}
			// If no required param found, try common names
			if len(args) == 0 {
				commonNames := []string{"file_path", "path", "input", "document"}
				for _, name := range commonNames {
					for _, param := range toolInfo.Parameters {
						if param.Name == name {
							args[name] = path
							break
						}
					}
					if len(args) > 0 {
						break
					}
				}
			}
		} else {
			// Fallback: try common parameter names
			args["file_path"] = path
		}

		// Execute the MCP tool
		result, err := tool.Execute(ctx, args)
		if err != nil {
			slog.Debug("MCP tool execution error",
				"tool", toolName,
				"path", path,
				"error", err.Error())
			// Tool execution failed, try next tool
			continue
		}

		// Debug log: show what the MCP tool returned
		contentLength := len(result.Content)
		contentPreview := result.Content
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:100] + "..."
		}
		slog.Debug(fmt.Sprintf("MCP tool %s result for %s: success=%v, error=%q, content_length=%d, content_preview=%q, has_metadata=%v",
			toolName, path, result.Success, result.Error, contentLength, contentPreview, result.Metadata != nil),
			"tool", toolName,
			"path", path,
			"success", result.Success,
			"error", result.Error,
			"content_length", contentLength,
			"content_preview", contentPreview,
			"has_metadata", result.Metadata != nil)

		if !result.Success {
			// Tool returned failure, try next tool
			// MCP tool layer already detected and reported the error
			slog.Debug("MCP tool returned failure, trying next tool",
				"tool", toolName,
				"path", path,
				"error", result.Error,
				"content_length", len(result.Content))
			continue
		}

		// Extract content from tool result
		content := result.Content
		if content == "" {
			// Try to extract from metadata or other fields
			if metadata, ok := result.Metadata.(map[string]interface{}); ok {
				if text, ok := metadata["content"].(string); ok {
					content = text
				} else if text, ok := metadata["text"].(string); ok {
					content = text
				}
			}
		}

		// Trim whitespace
		content = strings.TrimSpace(content)

		// Defense in depth: Check if content itself is an error message
		// This handles edge cases where MCP tool layer might have missed an error pattern
		// (though it should be rare now that MCP tool properly detects errors)
		if content != "" {
			contentLower := strings.ToLower(content)
			// Check for MCP tool error message patterns
			if strings.HasPrefix(contentLower, "error executing tool") ||
				strings.HasPrefix(contentLower, "error:") ||
				strings.HasPrefix(contentLower, "tool error:") {
				slog.Debug("MCP tool returned error message in content (defense in depth check), failing extraction",
					"tool", toolName,
					"path", path,
					"error_content", content)
				return nil, fmt.Errorf("MCP tool %s failed: %s", toolName, content)
			}
		}

		// If content is empty, try next tool
		if content == "" {
			slog.Debug("MCP tool returned empty content, trying next tool",
				"tool", toolName,
				"path", path,
				"result_success", result.Success,
				"result_error", result.Error)
			continue
		}

		// Extract metadata from tool result
		metadata := make(map[string]string)
		title := ""
		author := ""

		if result.Metadata != nil {
			if metaMap, ok := result.Metadata.(map[string]interface{}); ok {
				for k, v := range metaMap {
					if strVal, ok := v.(string); ok {
						metadata[k] = strVal
						if k == "title" || k == "document_title" {
							title = strVal
						}
						if k == "author" || k == "document_author" {
							author = strVal
						}
					}
				}
			}
		}

		// If title not found, use filename
		if title == "" {
			title = filepath.Base(path)
		}

		// Add file metadata
		metadata["file_path"] = path
		metadata["file_size"] = fmt.Sprintf("%d", fileSize)
		metadata["extractor"] = "mcp"
		metadata["tool"] = toolName

		processingTime := time.Since(startTime).Milliseconds()

		return &ExtractedContent{
			Content:          content,
			Title:            title,
			Author:           author,
			Metadata:         metadata,
			ProcessingTimeMs: processingTime,
		}, nil
	}

	// All tools failed
	return nil, fmt.Errorf("all MCP parser tools failed for file %s (tried tools: %v)", path, e.parserToolNames)
}

// Priority returns the extractor priority
func (e *MCPExtractor) Priority() int {
	return e.priority
}
