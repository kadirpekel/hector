package cli

import "github.com/kadirpekel/hector/pkg/config"

// ============================================================================
// COMMAND TYPES
// ============================================================================

// CommandType represents a CLI command
type CommandType string

const (
	CommandServe CommandType = "serve"
	CommandList  CommandType = "list"
	CommandInfo  CommandType = "info"
	CommandCall  CommandType = "call"
	CommandChat  CommandType = "chat"
	CommandTask  CommandType = "task"
	CommandHelp  CommandType = "help"
)

// ============================================================================
// CLI MODES
// ============================================================================

// CLIMode represents the operational mode of the CLI
type CLIMode string

const (
	// Deployment modes
	ModeServerZeroConfig CLIMode = "server-zero-config" // 'serve' without config file
	ModeServerConfig     CLIMode = "server-config"      // 'serve' with config file
	ModeClient           CLIMode = "client"             // Connect to remote server (--server)
	ModeLocalZeroConfig  CLIMode = "local-zero-config"  // In-process, no config file
	ModeLocalConfig      CLIMode = "local-config"       // In-process with config file
)

// String returns a human-readable string for the CLI mode
func (m CLIMode) String() string {
	switch m {
	case ModeServerZeroConfig:
		return "Server (Zero-Config)"
	case ModeServerConfig:
		return "Server (Config)"
	case ModeClient:
		return "Client (Remote)"
	case ModeLocalZeroConfig:
		return "Local (Zero-Config)"
	case ModeLocalConfig:
		return "Local (Config)"
	default:
		return string(m)
	}
}

// ============================================================================
// CLI ARGUMENTS
// ============================================================================

// CLIArgs holds parsed command line arguments
type CLIArgs struct {
	Command    CommandType
	ConfigFile string
	ServerURL  string
	AgentID    string
	TaskID     string
	TaskAction string // For task command: "get" or "cancel"
	Input      string
	Token      string
	Stream     bool
	Debug      bool
	Port       int

	// A2A Server options (override config)
	Host       string
	A2ABaseURL string

	// Zero-config mode options
	Provider                string // Detected provider: "openai", "anthropic", "gemini"
	APIKey                  string
	BaseURL                 string
	Model                   string
	Tools                   bool
	MCPURL                  string
	DocsFolder              string
	EmbedderModel           string
	VectorDB                string
	ExplicitZeroConfigFlags bool // Tracks if user explicitly provided zero-config flags
}

// ToZeroConfigOptions converts CLIArgs to config.ZeroConfigOptions
// This consolidates the mapping logic in one place to avoid duplication
func (a *CLIArgs) ToZeroConfigOptions() config.ZeroConfigOptions {
	return config.ZeroConfigOptions{
		Provider:    a.Provider,
		APIKey:      a.APIKey,
		BaseURL:     a.BaseURL,
		Model:       a.Model,
		EnableTools: a.Tools,
		MCPURL:      a.MCPURL,
		DocsFolder:  a.DocsFolder,
		AgentName:   a.AgentID,
	}
}
