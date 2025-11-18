package cli

import "testing"

func TestCLIArgs_AllFields(t *testing.T) {

	args := CLIArgs{
		ConfigFile:    "test.yaml",
		LogLevel:      "debug",
		ServerURL:     "http://localhost:8080",
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
	if args.LogLevel != "debug" {
		t.Error("Failed to set LogLevel")
	}
	if args.Provider != "openai" {
		t.Error("Failed to set Provider")
	}
}
