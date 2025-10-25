package cli

// ============================================================================
// CLI STRUCTURE
// ============================================================================
//
// This defines the command-line interface using declarative struct tags.
// Benefits:
// - Flags can appear anywhere (before/after positional args)
// - No duplication - global flags defined once
// - Auto-generated help
// - Type safety with validation
// - Environment variable support built-in
//
// ============================================================================

// CLI defines the complete Hector command-line interface
var CLI struct {
	// Global flags (inherited by all commands)
	Config string `short:"c" help:"Configuration file" type:"path" placeholder:"PATH"`
	Debug  bool   `short:"d" help:"Enable debug mode"`

	// Commands
	Version VersionCmd `cmd:"" help:"Show Hector version information"`
	Serve   ServeCmd   `cmd:"" help:"Start the A2A server to host agents"`
	List    ListCmd    `cmd:"" help:"List all available agents"`
	Info    InfoCmd    `cmd:"" help:"Display detailed agent information"`
	Call    CallCmd    `cmd:"" help:"Send a single message to an agent"`
	Chat    ChatCmd    `cmd:"" help:"Start an interactive chat session with an agent"`
	Task    TaskCmd    `cmd:"" help:"Task operations (get or cancel)"`
}

// ============================================================================
// SHARED FLAG GROUPS
// ============================================================================

// ZeroConfigFlags are the common LLM configuration flags used across multiple commands
// These are embedded to avoid duplication (DRY principle)
type ZeroConfigFlags struct {
	Provider      string `help:"LLM provider" enum:"openai,anthropic,gemini" default:"openai" env:"HECTOR_PROVIDER"`
	Model         string `help:"LLM model name" env:"HECTOR_MODEL"`
	APIKey        string `name:"api-key" help:"API key for LLM provider" env:"OPENAI_API_KEY,ANTHROPIC_API_KEY,GEMINI_API_KEY"`
	BaseURL       string `name:"base-url" help:"Custom API base URL" env:"HECTOR_BASE_URL" placeholder:"URL"`
	Tools         bool   `help:"Enable built-in tools"`
	MCPURL        string `name:"mcp-url" help:"MCP server URL for external tools" env:"MCP_URL" placeholder:"URL"`
	DocsFolder    string `name:"docs-folder" help:"Folder containing documents for RAG" type:"path" placeholder:"PATH"`
	EmbedderModel string `name:"embedder-model" help:"Embedder model for document store" default:"nomic-embed-text"`
	VectorDB      string `name:"vectordb" help:"Vector database connection string" default:"http://localhost:6334" placeholder:"URL"`
}

// ClientModeFlags are the common flags for connecting to a remote server
type ClientModeFlags struct {
	Server string `help:"A2A server URL (enables client mode)" env:"HECTOR_SERVER" placeholder:"URL"`
	Token  string `help:"Authentication token for server access" env:"HECTOR_TOKEN"`
	Agent  string `help:"Agent name (required when using --server or --config)" placeholder:"AGENT"`
}

// ============================================================================
// VERSION COMMAND
// ============================================================================

// VersionCmd shows Hector version information
type VersionCmd struct {
	// No flags needed for version command
}

// ============================================================================
// SERVE COMMAND
// ============================================================================

// ServeCmd starts the A2A server
type ServeCmd struct {
	// Server options
	Port       int    `default:"8080" help:"gRPC server port (default: ${default})"`
	Host       string `help:"Server host address (overrides config)" placeholder:"HOST"`
	A2ABaseURL string `name:"a2a-base-url" help:"A2A base URL for agent discovery" placeholder:"URL"`

	// Zero-config options (embedded - no duplication!)
	ZeroConfigFlags `embed:"" prefix:""`

	// Positional argument (optional)
	AgentName string `arg:"" optional:"" help:"Agent name for zero-config mode (default: assistant)" placeholder:"AGENT"`
}

// ============================================================================
// LIST COMMAND
// ============================================================================

// ListCmd lists all available agents
type ListCmd struct {
	ClientModeFlags `embed:"" prefix:""`
}

// ============================================================================
// INFO COMMAND
// ============================================================================

// InfoCmd displays detailed agent information
type InfoCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	// Positional argument
	Agent string `arg:"" help:"Agent name to get information about" placeholder:"AGENT"`
}

// ============================================================================
// CALL COMMAND
// ============================================================================

// CallCmd sends a single message to an agent
type CallCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	// Execution options
	Stream    bool   `default:"true" negatable:"" help:"Enable streaming responses (use --no-stream to disable)"`
	SessionID string `name:"session" help:"Session ID for conversation continuity" env:"HECTOR_SESSION" placeholder:"SESSION_ID"`

	// Zero-config options (embedded - no duplication!)
	ZeroConfigFlags `embed:"" prefix:""`

	// Positional argument (message only)
	Message string `arg:"" help:"Message to send to the agent" placeholder:"MESSAGE"`
}

// ============================================================================
// CHAT COMMAND
// ============================================================================

// ChatCmd starts an interactive chat session
type ChatCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	// Execution options
	NoStream  bool   `name:"no-stream" help:"Disable streaming responses (streaming enabled by default)"`
	SessionID string `name:"session" help:"Session ID to resume previous conversation" env:"HECTOR_SESSION" placeholder:"SESSION_ID"`

	// Zero-config options (embedded - no duplication!)
	ZeroConfigFlags `embed:"" prefix:""`
}

// ============================================================================
// TASK COMMAND
// ============================================================================

// TaskCmd handles task operations
type TaskCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	// Subcommands
	Get    TaskGetCmd    `cmd:"" help:"Get task details by ID"`
	Cancel TaskCancelCmd `cmd:"" help:"Cancel a running task by ID"`
}

// TaskGetCmd retrieves task details
type TaskGetCmd struct {
	Agent  string `arg:"" help:"Agent name that owns the task" placeholder:"AGENT"`
	TaskID string `arg:"" help:"Task ID to retrieve" placeholder:"TASK_ID"`
}

// TaskCancelCmd cancels a running task
type TaskCancelCmd struct {
	Agent  string `arg:"" help:"Agent name that owns the task" placeholder:"AGENT"`
	TaskID string `arg:"" help:"Task ID to cancel" placeholder:"TASK_ID"`
}
