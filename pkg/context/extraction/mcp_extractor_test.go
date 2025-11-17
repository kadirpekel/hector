package extraction

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// mockToolCaller is a mock implementation of ToolCaller for testing
type mockToolCaller struct {
	tools map[string]Tool
}

func newMockToolCaller() *mockToolCaller {
	return &mockToolCaller{
		tools: make(map[string]Tool),
	}
}

func (m *mockToolCaller) GetTool(name string) (Tool, error) {
	tool, exists := m.tools[name]
	if !exists {
		return nil, errors.New("tool not found")
	}
	return tool, nil
}

func (m *mockToolCaller) addTool(name string, tool Tool) {
	m.tools[name] = tool
}

// mockTool is a mock implementation of Tool for testing
type mockTool struct {
	name        string
	description string
	parameters  []ToolParameter
	executeFunc func(ctx context.Context, args map[string]interface{}) (ToolResult, error)
}

func (m *mockTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        m.name,
		Description: m.description,
		Parameters:  m.parameters,
	}
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return ToolResult{Success: true, Content: "mock content"}, nil
}

func TestNewMCPExtractor(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPExtractorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: MCPExtractorConfig{
				ToolCaller:      newMockToolCaller(),
				ParserToolNames: []string{"parse_document"},
			},
			wantErr: false,
		},
		{
			name: "nil tool caller",
			config: MCPExtractorConfig{
				ToolCaller:      nil,
				ParserToolNames: []string{"parse_document"},
			},
			wantErr: true,
			errMsg:  "tool caller is required",
		},
		{
			name: "empty tool names",
			config: MCPExtractorConfig{
				ToolCaller:      newMockToolCaller(),
				ParserToolNames: []string{},
			},
			wantErr: true,
			errMsg:  "at least one parser tool name is required",
		},
		{
			name: "custom priority",
			config: MCPExtractorConfig{
				ToolCaller:      newMockToolCaller(),
				ParserToolNames: []string{"parse_document"},
				Priority:        10,
			},
			wantErr: false,
		},
		{
			name: "with extensions",
			config: MCPExtractorConfig{
				ToolCaller:      newMockToolCaller(),
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{".pdf", ".docx"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewMCPExtractor(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewMCPExtractor() expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewMCPExtractor() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewMCPExtractor() unexpected error: %v", err)
				return
			}
			if extractor == nil {
				t.Errorf("NewMCPExtractor() returned nil extractor")
			}
			if tt.config.Priority != 0 && extractor.Priority() != tt.config.Priority {
				t.Errorf("NewMCPExtractor() priority = %d, want %d", extractor.Priority(), tt.config.Priority)
			}
			if tt.config.Priority == 0 && extractor.Priority() != 8 {
				t.Errorf("NewMCPExtractor() default priority = %d, want 8", extractor.Priority())
			}
		})
	}
}

func TestMCPExtractor_Name(t *testing.T) {
	caller := newMockToolCaller()
	extractor, _ := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document"},
	})

	name := extractor.Name()
	expected := "MCPExtractor:parse_document"
	if name != expected {
		t.Errorf("Name() = %v, want %v", name, expected)
	}

	// Test with multiple tools
	extractor2, _ := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document", "docling_parse"},
	})
	name2 := extractor2.Name()
	expected2 := "MCPExtractor:parse_document,docling_parse"
	if name2 != expected2 {
		t.Errorf("Name() = %v, want %v", name2, expected2)
	}
}

func TestMCPExtractor_CanExtract(t *testing.T) {
	caller := newMockToolCaller()

	// Add a tool to the caller
	tool := &mockTool{name: "parse_document"}
	caller.addTool("parse_document", tool)

	tests := []struct {
		name       string
		config     MCPExtractorConfig
		path       string
		mimeType   string
		wantResult bool
	}{
		{
			name: "no extensions configured - tool available",
			config: MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{}, // Empty = all files
			},
			path:       "/path/to/document.pdf",
			mimeType:   "application/pdf",
			wantResult: true,
		},
		{
			name: "extension matches - tool available",
			config: MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{".pdf", ".docx"},
			},
			path:       "/path/to/document.pdf",
			mimeType:   "application/pdf",
			wantResult: true,
		},
		{
			name: "extension doesn't match",
			config: MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{".pdf"},
			},
			path:       "/path/to/document.docx",
			mimeType:   "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			wantResult: false,
		},
		{
			name: "extension matches but tool not available",
			config: MCPExtractorConfig{
				ToolCaller:      newMockToolCaller(), // Empty caller - no tools
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{".pdf"},
			},
			path:       "/path/to/document.pdf",
			mimeType:   "application/pdf",
			wantResult: false,
		},
		{
			name: "case insensitive extension",
			config: MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document"},
				SupportedExts:   []string{".PDF", ".DOCX"},
			},
			path:       "/path/to/document.pdf",
			mimeType:   "application/pdf",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewMCPExtractor(tt.config)
			if err != nil {
				t.Fatalf("NewMCPExtractor() error = %v", err)
			}

			result := extractor.CanExtract(tt.path, tt.mimeType)
			if result != tt.wantResult {
				t.Errorf("CanExtract() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestMCPExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setupCaller func() *mockToolCaller
		path        string
		fileSize    int64
		wantContent string
		wantTitle   string
		wantAuthor  string
		wantErr     bool
	}{
		{
			name: "successful extraction",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				tool := &mockTool{
					name: "parse_document",
					parameters: []ToolParameter{
						{Name: "file_path", Type: "string", Required: true},
					},
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{
							Success: true,
							Content: "Extracted document content",
							Metadata: map[string]interface{}{
								"title":  "Test Document",
								"author": "Test Author",
							},
						}, nil
					},
				}
				caller.addTool("parse_document", tool)
				return caller
			},
			path:        "/path/to/document.pdf",
			fileSize:    1024,
			wantContent: "Extracted document content",
			wantTitle:   "Test Document",
			wantAuthor:  "Test Author",
			wantErr:     false,
		},
		{
			name: "tool not found - tries next tool",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				// First tool not available
				// Second tool available
				tool2 := &mockTool{
					name: "docling_parse",
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{
							Success: true,
							Content: "Content from docling",
						}, nil
					},
				}
				caller.addTool("docling_parse", tool2)
				return caller
			},
			path:        "/path/to/document.pdf",
			fileSize:    1024,
			wantContent: "Content from docling",
			wantTitle:   "document.pdf", // Defaults to filename
			wantErr:     false,
		},
		{
			name: "all tools fail",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				// No tools added
				return caller
			},
			path:     "/path/to/document.pdf",
			fileSize: 1024,
			wantErr:  true,
		},
		{
			name: "tool execution fails",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				tool := &mockTool{
					name: "parse_document",
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{}, errors.New("execution failed")
					},
				}
				caller.addTool("parse_document", tool)
				return caller
			},
			path:     "/path/to/document.pdf",
			fileSize: 1024,
			wantErr:  true,
		},
		{
			name: "tool returns success=false",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				tool := &mockTool{
					name: "parse_document",
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{Success: false, Error: "parsing failed"}, nil
					},
				}
				caller.addTool("parse_document", tool)
				// Add fallback tool
				tool2 := &mockTool{
					name: "fallback_parse",
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{Success: true, Content: "Fallback content"}, nil
					},
				}
				caller.addTool("fallback_parse", tool2)
				return caller
			},
			path:        "/path/to/document.pdf",
			fileSize:    1024,
			wantContent: "Fallback content",
			wantErr:     false,
		},
		{
			name: "content in metadata",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				tool := &mockTool{
					name: "parse_document",
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						return ToolResult{
							Success: true,
							Content: "", // Empty content
							Metadata: map[string]interface{}{
								"content": "Content from metadata",
								"text":    "Alternative text",
							},
						}, nil
					},
				}
				caller.addTool("parse_document", tool)
				return caller
			},
			path:        "/path/to/document.pdf",
			fileSize:    1024,
			wantContent: "Content from metadata", // Should prefer "content" over "text"
			wantErr:     false,
		},
		{
			name: "parameter name detection",
			setupCaller: func() *mockToolCaller {
				caller := newMockToolCaller()
				tool := &mockTool{
					name: "parse_document",
					parameters: []ToolParameter{
						{Name: "input", Type: "string", Required: true},
					},
					executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
						// Verify correct parameter was used
						if _, ok := args["input"]; !ok {
							return ToolResult{}, errors.New("input parameter not found")
						}
						return ToolResult{
							Success: true,
							Content: "Content parsed with input parameter",
						}, nil
					},
				}
				caller.addTool("parse_document", tool)
				return caller
			},
			path:        "/path/to/document.pdf",
			fileSize:    1024,
			wantContent: "Content parsed with input parameter",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := tt.setupCaller()
			extractor, err := NewMCPExtractor(MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document", "docling_parse", "fallback_parse"},
			})
			if err != nil {
				t.Fatalf("NewMCPExtractor() error = %v", err)
			}

			result, err := extractor.Extract(ctx, tt.path, tt.fileSize)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Extract() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Extract() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Extract() returned nil result")
				return
			}

			if result.Content != tt.wantContent {
				t.Errorf("Extract() content = %v, want %v", result.Content, tt.wantContent)
			}

			if tt.wantTitle != "" && result.Title != tt.wantTitle {
				t.Errorf("Extract() title = %v, want %v", result.Title, tt.wantTitle)
			}

			if tt.wantAuthor != "" && result.Author != tt.wantAuthor {
				t.Errorf("Extract() author = %v, want %v", result.Author, tt.wantAuthor)
			}

			// Verify metadata
			if result.Metadata["file_path"] != tt.path {
				t.Errorf("Extract() metadata file_path = %v, want %v", result.Metadata["file_path"], tt.path)
			}

			if result.Metadata["extractor"] != "mcp" {
				t.Errorf("Extract() metadata extractor = %v, want mcp", result.Metadata["extractor"])
			}
		})
	}
}

func TestMCPExtractor_Priority(t *testing.T) {
	caller := newMockToolCaller()

	tests := []struct {
		name     string
		priority int
		want     int
	}{
		{
			name:     "default priority",
			priority: 0,
			want:     8,
		},
		{
			name:     "custom priority",
			priority: 10,
			want:     10,
		},
		{
			name:     "low priority",
			priority: 3,
			want:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewMCPExtractor(MCPExtractorConfig{
				ToolCaller:      caller,
				ParserToolNames: []string{"parse_document"},
				Priority:        tt.priority,
			})
			if err != nil {
				t.Fatalf("NewMCPExtractor() error = %v", err)
			}

			if extractor.Priority() != tt.want {
				t.Errorf("Priority() = %v, want %v", extractor.Priority(), tt.want)
			}
		})
	}
}

func TestMCPExtractor_hasParserTool(t *testing.T) {
	caller := newMockToolCaller()
	tool := &mockTool{name: "parse_document"}
	caller.addTool("parse_document", tool)

	extractor, _ := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document"},
	})

	if !extractor.hasParserTool() {
		t.Error("hasParserTool() = false, want true")
	}

	// Test with tool not available
	emptyCaller := newMockToolCaller()
	extractor2, _ := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      emptyCaller,
		ParserToolNames: []string{"parse_document"},
	})

	if extractor2.hasParserTool() {
		t.Error("hasParserTool() = true, want false (tool not available)")
	}
}

func TestMCPExtractor_Extract_FileMetadata(t *testing.T) {
	ctx := context.Background()
	caller := newMockToolCaller()

	tool := &mockTool{
		name: "parse_document",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			return ToolResult{
				Success: true,
				Content: "Test content",
			}, nil
		},
	}
	caller.addTool("parse_document", tool)

	extractor, _ := NewMCPExtractor(MCPExtractorConfig{
		ToolCaller:      caller,
		ParserToolNames: []string{"parse_document"},
	})

	path := "/path/to/test/document.pdf"
	fileSize := int64(2048)

	result, err := extractor.Extract(ctx, path, fileSize)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Verify file metadata
	if result.Metadata["file_path"] != path {
		t.Errorf("Metadata file_path = %v, want %v", result.Metadata["file_path"], path)
	}

	if result.Metadata["file_size"] != "2048" {
		t.Errorf("Metadata file_size = %v, want 2048", result.Metadata["file_size"])
	}

	if result.Metadata["extractor"] != "mcp" {
		t.Errorf("Metadata extractor = %v, want mcp", result.Metadata["extractor"])
	}

	if result.Metadata["tool"] != "parse_document" {
		t.Errorf("Metadata tool = %v, want parse_document", result.Metadata["tool"])
	}

	// Verify title defaults to filename
	expectedTitle := filepath.Base(path)
	if result.Title != expectedTitle {
		t.Errorf("Title = %v, want %v", result.Title, expectedTitle)
	}
}
