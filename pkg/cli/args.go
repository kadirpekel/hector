// Package cli provides command-line interface utilities for Hector
package cli

// Args holds all CLI arguments for Hector commands
type Args struct {
	// Global flags
	ConfigFile string
	Debug      bool

	// Server mode flags
	ServerURL string
	Token     string

	// Server command flags
	Port int

	// Zero-config flags
	Provider   string // LLM provider: "openai" (default), "anthropic", "gemini"
	APIKey     string
	BaseURL    string
	Model      string
	Tools      bool
	MCPURL     string
	DocsFolder string

	// Command-specific flags
	AgentID string
	TaskID  string
	Input   string
	Stream  bool

	// Embedder/Vector DB flags (for advanced config)
	EmbedderModel string
	VectorDB      string
}

// SetDefaults sets default values for CLI arguments
func (a *Args) SetDefaults() {
	if a.ConfigFile == "" {
		a.ConfigFile = "hector.yaml"
	}
	if a.Port == 0 {
		a.Port = 8080 // Match A2A server default
	}
	if a.BaseURL == "" {
		a.BaseURL = "https://api.openai.com/v1"
	}
	if a.Model == "" {
		a.Model = "gpt-4"
	}
	// Streaming is default
	if !a.Stream {
		a.Stream = true
	}
}
