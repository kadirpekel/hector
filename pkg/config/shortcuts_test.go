package config

import (
	"testing"
)

func TestAgentShortcuts_DocsFolder(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]LLMProviderConfig{
			"test-llm": {
				Type: "openai",
			},
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Name:       "Test Agent",
				LLM:        "test-llm",
				DocsFolder: "./test-folder",
			},
		},
	}

	// Apply defaults (this should expand shortcuts)
	cfg.SetDefaults()

	// Check that document store was auto-created
	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Fatal("DocumentStores should be auto-populated from docs_folder")
	}
	if len(agent.DocumentStores) != 1 {
		t.Fatalf("Should have exactly 1 document store, got %d", len(agent.DocumentStores))
	}

	// Check that the document store exists in config
	storeName := agent.DocumentStores[0]
	store, exists := cfg.DocumentStores[storeName]
	if !exists {
		t.Fatal("Document store should exist in config")
	}
	if store.Path != "./test-folder" {
		t.Errorf("Document store path should be './test-folder', got '%s'", store.Path)
	}

	// Check that database and embedder were auto-set
	if agent.Database != "default-database" {
		t.Errorf("Database should be 'default-database', got '%s'", agent.Database)
	}
	if agent.Embedder != "default-embedder" {
		t.Errorf("Embedder should be 'default-embedder', got '%s'", agent.Embedder)
	}

	// Check that search tool was auto-created
	searchTool, exists := cfg.Tools.Tools["search"]
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
		LLMs: map[string]LLMProviderConfig{
			"test-llm": {
				Type: "openai",
			},
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				EnableTools: true,
			},
		},
	}

	// Apply defaults (this should expand shortcuts)
	cfg.SetDefaults()

	// Check that tools were auto-enabled (nil = all tools)
	agent := cfg.Agents["test-agent"]
	if agent.Tools != nil {
		t.Error("Tools should be nil (meaning all tools available)")
	}

	// Check that core tools were auto-configured
	if _, exists := cfg.Tools.Tools["execute_command"]; !exists {
		t.Error("execute_command should be auto-configured")
	}
	if _, exists := cfg.Tools.Tools["write_file"]; !exists {
		t.Error("write_file should be auto-configured")
	}
	if _, exists := cfg.Tools.Tools["search_replace"]; !exists {
		t.Error("search_replace should be auto-configured")
	}
	if _, exists := cfg.Tools.Tools["todo_write"]; !exists {
		t.Error("todo_write should be auto-configured")
	}
}

func TestAgentShortcuts_MutuallyExclusive_DocsFolder(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Name:           "Test Agent",
				LLM:            "test-llm",
				DocsFolder:     "./test-folder",
				DocumentStores: []string{"explicit-store"}, // Both shortcut and explicit
			},
		},
	}

	// Apply defaults first
	cfg.SetDefaults()

	// Validation should fail (mutually exclusive)
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Should error when both docs_folder and document_stores are specified")
	}
	if !contains(err.Error(), "mutually exclusive") {
		t.Errorf("Error should mention 'mutually exclusive', got: %s", err.Error())
	}
}

func TestAgentShortcuts_MutuallyExclusive_EnableTools(t *testing.T) {
	cfg := &Config{
		LLMs: map[string]LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				EnableTools: true,
				Tools:       []string{"write_file"}, // Both shortcut and explicit
			},
		},
	}

	// Validation should fail BEFORE SetDefaults (during Validate on the raw config)
	// The validation happens in AgentConfig.Validate() which is called before expansion
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
		LLMs: map[string]LLMProviderConfig{
			"test-llm": {
				Type:   "openai",
				APIKey: "test-api-key",
			},
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Name:        "Test Agent",
				LLM:         "test-llm",
				DocsFolder:  "./test-folder",
				EnableTools: true, // Both shortcuts
			},
		},
	}

	// Apply defaults (this should expand both shortcuts)
	cfg.SetDefaults()

	// Check document store expansion
	agent := cfg.Agents["test-agent"]
	if len(agent.DocumentStores) == 0 {
		t.Error("DocumentStores should be auto-populated")
	}

	// Check tools expansion
	if agent.Tools != nil {
		t.Error("Tools should be nil (all tools available)")
	}

	// Check that both search tool and other tools were created
	if _, exists := cfg.Tools.Tools["search"]; !exists {
		t.Error("Search tool should be created")
	}
	if _, exists := cfg.Tools.Tools["execute_command"]; !exists {
		t.Error("execute_command should be created")
	}
	if _, exists := cfg.Tools.Tools["write_file"]; !exists {
		t.Error("write_file should be created")
	}

	// Check that execute_command has sandboxing enabled (which allows empty allowed_commands)
	execCmd := cfg.Tools.Tools["execute_command"]
	if !execCmd.EnableSandboxing {
		t.Error("execute_command should have sandboxing enabled")
	}

	// Note: Full validation requires all LLM/DB/Embedder configs to be valid
	// The key test is that shortcuts expanded correctly (which we verified above)
}
