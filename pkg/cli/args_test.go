package cli

import "testing"

func TestArgs_SetDefaults(t *testing.T) {
	args := &Args{}
	args.SetDefaults()

	// Test default values
	if args.ConfigFile != "hector.yaml" {
		t.Errorf("Expected ConfigFile 'hector.yaml', got '%s'", args.ConfigFile)
	}
	if args.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", args.Port)
	}
	if args.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected BaseURL 'https://api.openai.com/v1', got '%s'", args.BaseURL)
	}
	if args.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got '%s'", args.Model)
	}
	if !args.Stream {
		t.Error("Expected Stream to be true by default")
	}
}

func TestArgs_SetDefaults_PreservesExisting(t *testing.T) {
	args := &Args{
		ConfigFile: "custom.yaml",
		Port:       9000,
		BaseURL:    "https://custom.com",
		Model:      "custom-model",
	}
	args.SetDefaults()

	// Test that existing values are preserved
	if args.ConfigFile != "custom.yaml" {
		t.Errorf("Expected ConfigFile 'custom.yaml', got '%s'", args.ConfigFile)
	}
	if args.Port != 9000 {
		t.Errorf("Expected Port 9000, got %d", args.Port)
	}
	if args.BaseURL != "https://custom.com" {
		t.Errorf("Expected BaseURL 'https://custom.com', got '%s'", args.BaseURL)
	}
	if args.Model != "custom-model" {
		t.Errorf("Expected Model 'custom-model', got '%s'", args.Model)
	}
}

func TestArgs_AllFields(t *testing.T) {
	// Test that all expected fields exist
	args := Args{
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

	// Just verify we can set all fields
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
