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

// Defaults are handled in two layers:
// - CLI flags: Set at flag definition in main.go parseArgs()
// - Zero-config: Set in config.CreateZeroConfig()
