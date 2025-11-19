package config

import (
	"testing"
)

func TestAgentShortcuts_DocsFolder(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:       "Test Agent",
				LLM:        "test-llm",
				DocsFolder: "./test-folder",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Fatal("DocumentStores should be auto-populated from docs_folder")
	}
	if len(agent.DocumentStores) != 1 {
		t.Fatalf("Should have exactly 1 document store, got %d", len(agent.DocumentStores))
	}

	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}
	if store.Path != "./test-folder" {
		t.Errorf("Document store path should be './test-folder', got '%s'", store.Path)
	}

	if agent.VectorStore != "default-vector-store" {
		t.Errorf("VectorStore should be 'default-vector-store', got '%s'", agent.VectorStore)
	}
	if agent.Embedder != "default-embedder" {
		t.Errorf("Embedder should be 'default-embedder', got '%s'", agent.Embedder)
	}

	searchTool, exists := cfg.Tools["search"]
	if !exists {
		t.Fatal("Search tool should be auto-created")
	}
	if searchTool.Type != "search" {
		t.Errorf("Search tool type should be 'search', got '%s'", searchTool.Type)
	}
	// Note: Document stores are now assigned at the agent level, not tool level
	// The search tool will automatically use the agent's assigned document stores
	// when the agent is created, so we verify the agent has the store assigned above
}

func TestAgentShortcuts_DocsFolder_WithMCPTools_NoAutoConfig(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Tools: map[string]*ToolConfig{
			"composio": {
				Type:      "mcp",
				Enabled:   BoolPtr(true),
				ServerURL: "https://apollo.composio.dev/v3/mcp/...", // Composio (GitHub, Slack, etc.)
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:       "Test Agent",
				LLM:        "test-llm",
				DocsFolder: "./test-folder",
				// No MCPParserTool specified - should NOT auto-configure
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Fatal("DocumentStores should be auto-populated from docs_folder")
	}

	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}

	// Verify MCP parsers are NOT auto-configured when MCP tools exist but no parser tool is specified
	// This is correct - Composio MCP provides GitHub/Slack tools, not document parsers
	if store.MCPParsers != nil {
		t.Fatal("MCP parsers should NOT be auto-configured when MCP tools exist but --mcp-parser-tool is not specified")
	}
}

func TestAgentShortcuts_DocsFolder_WithoutMCPTools(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:       "Test Agent",
				LLM:        "test-llm",
				DocsFolder: "./test-folder",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}

	// Verify MCP parsers are NOT auto-configured when no MCP tools are present
	if store.MCPParsers != nil {
		t.Fatal("MCP parsers should NOT be auto-configured when no MCP tools are present")
	}
}

func TestAgentShortcuts_DocsFolder_WithMCPParserToolExplicit(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Tools: map[string]*ToolConfig{
			"docling": {
				Type:      "mcp",
				Enabled:   BoolPtr(true),
				ServerURL: "http://localhost:3000/mcp",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				DocsFolder:    "./test-folder",
				MCPParserTool: "custom_parse_tool", // Explicitly specify single parser tool name
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}

	// Verify MCP parsers ARE auto-configured when user explicitly specifies a tool name
	// We trust the user's input - validation happens at runtime when tools are actually used
	if store.MCPParsers == nil {
		t.Fatal("MCP parsers should be auto-configured when user explicitly specifies --mcp-parser-tool")
	}

	// Verify custom tool name is used
	if len(store.MCPParsers.ToolNames) != 1 {
		t.Fatalf("Expected 1 tool name when single tool is specified, got %d", len(store.MCPParsers.ToolNames))
	}
	if store.MCPParsers.ToolNames[0] != "custom_parse_tool" {
		t.Errorf("Expected tool name 'custom_parse_tool', got '%s'", store.MCPParsers.ToolNames[0])
	}

	// Verify MCPParserTool field is cleared after expansion
	if agent.MCPParserTool != "" {
		t.Error("MCPParserTool should be cleared after expansion")
	}
}

func TestAgentShortcuts_DocsFolder_WithMCPParserToolMultiple(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Tools: map[string]*ToolConfig{
			"docling": {
				Type:      "mcp",
				Enabled:   BoolPtr(true),
				ServerURL: "http://localhost:3000/mcp",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				DocsFolder:    "./test-folder",
				MCPParserTool: "parse_document,docling_parse,convert_document", // Multiple tools as fallback chain
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}

	// Verify MCP parsers were auto-configured with multiple tool names
	if store.MCPParsers == nil {
		t.Fatal("MCP parsers should be auto-configured when user explicitly specifies multiple tools")
	}

	// Verify all tool names are included
	expectedTools := []string{"parse_document", "docling_parse", "convert_document"}
	if len(store.MCPParsers.ToolNames) != len(expectedTools) {
		t.Fatalf("Expected %d tool names, got %d", len(expectedTools), len(store.MCPParsers.ToolNames))
	}
	for i, expected := range expectedTools {
		if store.MCPParsers.ToolNames[i] != expected {
			t.Errorf("Expected tool name '%s' at index %d, got '%s'", expected, i, store.MCPParsers.ToolNames[i])
		}
	}
}

func TestAgentShortcuts_DocsFolder_WithDisabledMCPTool(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Tools: map[string]*ToolConfig{
			"docling": {
				Type:      "mcp",
				Enabled:   BoolPtr(false), // Disabled
				ServerURL: "http://localhost:3000/mcp",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:       "Test Agent",
				LLM:        "test-llm",
				DocsFolder: "./test-folder",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}

	// Verify MCP parsers are NOT auto-configured when MCP tool is disabled
	if store.MCPParsers != nil {
		t.Fatal("MCP parsers should NOT be auto-configured when MCP tool is disabled")
	}
}

func TestAgentShortcuts_EnableTools(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				EnableTools: BoolPtr(true),
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	if agent.Tools != nil {
		t.Error("Tools should be nil (meaning all tools available)")
	}

	if _, exists := cfg.Tools["execute_command"]; !exists {
		t.Error("execute_command should be auto-configured")
	}
	if _, exists := cfg.Tools["write_file"]; !exists {
		t.Error("write_file should be auto-configured")
	}
	if _, exists := cfg.Tools["search_replace"]; !exists {
		t.Error("search_replace should be auto-configured")
	}
	if _, exists := cfg.Tools["todo_write"]; !exists {
		t.Error("todo_write should be auto-configured")
	}
}

func TestAgentShortcuts_MutuallyExclusive_DocsFolder(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:           "Test Agent",
				LLM:            "test-llm",
				DocsFolder:     "./test-folder",
				DocumentStores: []string{"explicit-store"},
			},
		},
	}

	_, err := ProcessConfigPipeline(cfg)
	if err == nil {
		t.Fatal("Should error when both docs_folder and document_stores are specified")
	}
	if !contains(err.Error(), "mutually exclusive") {
		t.Errorf("Error should mention 'mutually exclusive', got: %s", err.Error())
	}
}

func TestAgentShortcuts_MCPParserTool_RequiresDocsFolder(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				MCPParserTool: "parse_document",
				// No DocsFolder - should error
			},
		},
	}

	_, err := ProcessConfigPipeline(cfg)
	if err == nil {
		t.Fatal("Should error when mcp_parser_tool is specified without docs_folder")
	}
	if !contains(err.Error(), "mcp_parser_tool shortcut requires docs_folder") {
		t.Errorf("Error should mention 'mcp_parser_tool shortcut requires docs_folder', got: %s", err.Error())
	}
}

func TestAgentShortcuts_MutuallyExclusive_EnableTools(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				EnableTools: BoolPtr(true),
				Tools:       []string{"write_file"},
			},
		},
	}

	agent := cfg.Agents["test-agent"]
	err := agent.Validate()
	if err == nil {
		t.Fatal("Should error when both enable_tools and explicit tools list are specified")
	}
	if !contains(err.Error(), "mutually exclusive") {
		t.Errorf("Error should mention 'mutually exclusive', got: %s", err.Error())
	}
}

func TestAgentShortcuts_Combined(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				DocsFolder:  "./test-folder",
				EnableTools: BoolPtr(true),
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Error("DocumentStores should be auto-populated")
	}

	if agent.Tools != nil {
		t.Error("Tools should be nil (all tools available)")
	}

	if _, exists := cfg.Tools["search"]; !exists {
		t.Error("Search tool should be created")
	}
	if _, exists := cfg.Tools["execute_command"]; !exists {
		t.Error("execute_command should be created")
	}
	if _, exists := cfg.Tools["write_file"]; !exists {
		t.Error("write_file should be created")
	}

	execCmd := cfg.Tools["execute_command"]
	if execCmd.EnableSandboxing == nil || !*execCmd.EnableSandboxing {
		t.Error("execute_command should have sandboxing enabled")
	}

}

func TestAgentShortcuts_DocsFolder_WithPathPrefix(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				DocsFolder:    "./test-folder:/docs", // local:remote syntax
				MCPParserTool: "convert_document",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Fatal("DocumentStores should be auto-populated from docs_folder")
	}

	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}
	if store.Path != "./test-folder" {
		t.Errorf("Document store path should be './test-folder', got '%s'", store.Path)
	}

	// Verify MCP parser config has path_prefix
	if store.MCPParsers == nil {
		t.Fatal("MCPParsers should be configured")
	}
	if store.MCPParsers.PathPrefix != "/docs" {
		t.Errorf("MCPParsers.PathPrefix should be '/docs', got '%s'", store.MCPParsers.PathPrefix)
	}
}

func TestAgentShortcuts_DocsFolder_WithPathPrefix_NestedRemotePath(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				DocsFolder:    "/path/to/local/docs:/data/documents", // absolute paths
				MCPParserTool: "parse_document",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store := cfg.DocumentStores[storeName]

	if store.Path != "/path/to/local/docs" {
		t.Errorf("Document store path should be '/path/to/local/docs', got '%s'", store.Path)
	}
	if store.MCPParsers.PathPrefix != "/data/documents" {
		t.Errorf("MCPParsers.PathPrefix should be '/data/documents', got '%s'", store.MCPParsers.PathPrefix)
	}
}

func TestAgentShortcuts_DocsFolder_NoPathPrefix(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]*LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-key",
			},
		},
		Agents: map[string]*AgentConfig{
			"test-agent": {
				Name:          "Test Agent",
				LLM:           "test-llm",
				DocsFolder:    "./test-folder", // No path prefix
				MCPParserTool: "convert_document",
			},
		},
	}

	processedCfg, err := ProcessConfigPipeline(cfg)
	if err != nil {
		t.Fatalf("ProcessConfigPipeline failed: %v", err)
	}
	cfg = processedCfg

	agent := cfg.Agents["test-agent"]
	storeName := agent.DocumentStores[0]
	store := cfg.DocumentStores[storeName]

	if store.Path != "./test-folder" {
		t.Errorf("Document store path should be './test-folder', got '%s'", store.Path)
	}
	// PathPrefix should be empty when not specified
	if store.MCPParsers.PathPrefix != "" {
		t.Errorf("MCPParsers.PathPrefix should be empty, got '%s'", store.MCPParsers.PathPrefix)
	}
}
