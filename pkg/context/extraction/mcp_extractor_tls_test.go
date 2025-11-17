package extraction

import (
	"context"
	"testing"
)

// TestMCPExtractor_WithTLSTool tests MCP extractor with TLS-enabled MCP tools
// This test verifies that MCP extractors can work with MCP tools that use TLS
func TestMCPExtractor_WithTLSTool(t *testing.T) {
	caller := newMockToolCaller()

	// Create a mock tool that simulates TLS-enabled MCP tool
	tool := &mockTool{
		name: "parse_document",
		parameters: []ToolParameter{
			{Name: "file_path", Type: "string", Required: true},
		},
		executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			// Simulate successful extraction via TLS-enabled MCP server
			return ToolResult{
				Success: true,
				Content: "Document content extracted via TLS-enabled MCP server",
				Metadata: map[string]interface{}{
					"title":       "Test Document",
					"author":      "Test Author",
					"source":      "mcp",
					"tls_enabled": true,
					"server_url":  "https://mcp-server.example.com",
				},
			}, nil
		},
	}
	caller.addTool("parse_document", tool)

	extractor, err := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document"},
		SupportedExts:   []string{".pdf", ".docx"},
		Priority:        8,
	})
	if err != nil {
		t.Fatalf("NewMCPExtractor() error = %v", err)
	}

	// Test CanExtract
	if !extractor.CanExtract("/test/document.pdf", "application/pdf") {
		t.Error("Expected CanExtract to return true for .pdf file")
	}

	// Test Extract
	result, err := extractor.Extract(context.Background(), "/test/document.pdf", 1024)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if result == nil {
		t.Fatal("Extract() returned nil result")
	}

	if result.Content == "" {
		t.Error("Expected non-empty content")
	}

	if result.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", result.Title)
	}

	if result.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", result.Author)
	}

	// Verify metadata includes TLS-related info
	if result.Metadata["extractor"] != "mcp" {
		t.Errorf("Expected extractor 'mcp', got '%s'", result.Metadata["extractor"])
	}

	if result.Metadata["tool"] != "parse_document" {
		t.Errorf("Expected tool 'parse_document', got '%s'", result.Metadata["tool"])
	}
}

// TestMCPExtractor_TLS_MultipleTools tests MCP extractor with multiple TLS-enabled tools
func TestMCPExtractor_TLS_MultipleTools(t *testing.T) {
	caller := newMockToolCaller()

	// Add multiple parser tools (some may use TLS, some may not)
	tools := []struct {
		name   string
		params []ToolParameter
		result string
	}{
		{
			name: "parse_document",
			params: []ToolParameter{
				{Name: "file_path", Type: "string", Required: true},
			},
			result: "Content from parse_document",
		},
		{
			name: "docling_parse",
			params: []ToolParameter{
				{Name: "file_path", Type: "string", Required: true},
			},
			result: "Content from docling_parse",
		},
	}

	for _, toolDef := range tools {
		tool := &mockTool{
			name:       toolDef.name,
			parameters: toolDef.params,
			executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
				// Return different content based on tool name
				content := toolDef.result
				return ToolResult{
					Success: true,
					Content: content,
					Metadata: map[string]interface{}{
						"tool": toolDef.name,
					},
				}, nil
			},
		}
		caller.addTool(toolDef.name, tool)
	}

	extractor, err := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document", "docling_parse"},
		SupportedExts:   []string{".pdf"},
	})
	if err != nil {
		t.Fatalf("NewMCPExtractor() error = %v", err)
	}

	// Extract should use the first available tool
	result, err := extractor.Extract(context.Background(), "/test/document.pdf", 1024)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if result == nil {
		t.Fatal("Extract() returned nil result")
	}

	// Should use the first tool (parse_document)
	if result.Content != "Content from parse_document" {
		t.Errorf("Expected content from parse_document, got '%s'", result.Content)
	}

	if result.Metadata["tool"] != "parse_document" {
		t.Errorf("Expected tool 'parse_document', got '%s'", result.Metadata["tool"])
	}
}

// TestMCPExtractor_TLS_ToolFailure tests fallback when TLS-enabled tool fails
func TestMCPExtractor_TLS_ToolFailure(t *testing.T) {
	caller := newMockToolCaller()

	// First tool fails (e.g., TLS connection issue)
	failingTool := &mockTool{
		name: "parse_document",
		parameters: []ToolParameter{
			{Name: "file_path", Type: "string", Required: true},
		},
		executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			return ToolResult{
				Success: false,
				Error:   "TLS connection failed",
			}, nil
		},
	}
	caller.addTool("parse_document", failingTool)

	// Second tool succeeds
	workingTool := &mockTool{
		name: "docling_parse",
		parameters: []ToolParameter{
			{Name: "file_path", Type: "string", Required: true},
		},
		executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			return ToolResult{
				Success: true,
				Content: "Content from fallback tool",
			}, nil
		},
	}
	caller.addTool("docling_parse", workingTool)

	extractor, err := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document", "docling_parse"},
		SupportedExts:   []string{".pdf"},
	})
	if err != nil {
		t.Fatalf("NewMCPExtractor() error = %v", err)
	}

	// Extract should fallback to second tool
	result, err := extractor.Extract(context.Background(), "/test/document.pdf", 1024)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if result == nil {
		t.Fatal("Extract() returned nil result")
	}

	// Should use the fallback tool
	if result.Content != "Content from fallback tool" {
		t.Errorf("Expected content from fallback tool, got '%s'", result.Content)
	}

	if result.Metadata["tool"] != "docling_parse" {
		t.Errorf("Expected tool 'docling_parse', got '%s'", result.Metadata["tool"])
	}
}
