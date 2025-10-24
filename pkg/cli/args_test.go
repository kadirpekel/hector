package cli

import "testing"

func TestCLIArgs_AllFields(t *testing.T) {

	args := CLIArgs{
		ConfigFile:    "test.yaml",
		Debug:         true,
		ServerURL:     "http://localhost:8081",
		Token:         "token",
		Port:          8080,
		Provider:      "openai",
		APIKey:        "key",
		BaseURL:       "url",
		Model:         "model",
		Tools:         true,
		MCPURL:        "mcp",
		DocsFolder:    "docs",
		AgentID:       "agent",
		Input:         "input",
		Stream:        false,
		EmbedderModel: "embedder",
		VectorDB:      "db",
	}

	if args.ConfigFile != "test.yaml" {
		t.Error("Failed to set ConfigFile")
	}
	if !args.Debug {
		t.Error("Failed to set Debug")
	}
	if args.Provider != "openai" {
		t.Error("Failed to set Provider")
	}
}
