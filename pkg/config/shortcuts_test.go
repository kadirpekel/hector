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

	if agent.Database != "default-database" {
		t.Errorf("Database should be 'default-database', got '%s'", agent.Database)
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
	found := false
	for _, ds := range searchTool.DocumentStores {
		if ds == storeName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Search tool should reference document store '%s'", storeName)
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
