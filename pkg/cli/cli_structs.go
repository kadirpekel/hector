package cli

var CLI struct {
	Config          string `short:"c" help:"Configuration path (file path or key in backend store)" placeholder:"PATH"`
	ConfigType      string `help:"Configuration backend type" enum:"file,consul,etcd,zookeeper" default:"file"`
	ConfigWatch     bool   `help:"Watch for configuration changes and auto-reload" default:"false"`
	ConfigEndpoints string `help:"Comma-separated backend endpoints (e.g., localhost:8500 for consul)" placeholder:"ENDPOINTS"`
	LogLevel        string `name:"log-level" help:"Set log level (debug, info, warn, error)" env:"LOG_LEVEL"`
	LogFile         string `name:"log-file" help:"Path to log file (empty = stderr)" env:"LOG_FILE"`
	LogFormat       string `name:"log-format" help:"Set log format (simple, verbose, or custom)" env:"LOG_FORMAT"`

	Version  VersionCmd  `cmd:"" help:"Show Hector version information"`
	Validate ValidateCmd `cmd:"" help:"Validate configuration file syntax and semantics"`
	Serve    ServeCmd    `cmd:"" help:"Start the A2A server to host agents"`
	Info     InfoCmd     `cmd:"" help:"Display detailed agent information"`
	Call     CallCmd     `cmd:"" help:"Send a single message to an agent"`
	Chat     ChatCmd     `cmd:"" help:"Start an interactive chat session with an agent"`
	Task     TaskCmd     `cmd:"" help:"Task operations (get or cancel)"`
}

type ZeroConfigFlags struct {
	Provider       string  `help:"LLM provider" enum:"openai,anthropic,gemini,ollama" default:"openai" env:"HECTOR_PROVIDER"`
	Model          string  `help:"LLM model name" env:"HECTOR_MODEL"`
	APIKey         string  `name:"api-key" help:"API key for LLM provider (overrides env vars)"`
	BaseURL        string  `name:"base-url" help:"Custom API base URL" env:"HECTOR_BASE_URL" placeholder:"URL"`
	Temperature    float64 `help:"LLM temperature (0.0-2.0, controls randomness)" default:"0.7"`
	MaxTokens      int     `name:"max-tokens" help:"Maximum tokens in LLM response" default:"8000"`
	Role           string  `help:"Override agent's system role/persona (e.g., 'You are a security expert')"`
	Instruction    string  `help:"Additional guidance appended to system prompt (e.g., 'Focus on performance')"`
	Tools          bool    `help:"Enable built-in tools"`
	Thinking       bool    `help:"Enable thinking at API level (like --tools enables tools)" negatable:""`
	ThinkingBudget int     `name:"thinking-budget" help:"Token budget for thinking (default: 1024, must be < max-tokens)" default:"0"`
	ShowTools      bool    `name:"show-tools" help:"Show tool calls and results in output (disabled by default, use --show-tools to enable)" negatable:""`
	ShowThinking   bool    `name:"show-thinking" help:"Show thinking blocks in output (disabled by default, use --show-thinking to enable)" negatable:""`
	MCPURL         string  `name:"mcp-url" help:"MCP server URL for external tools" env:"MCP_URL" placeholder:"URL"`
	MCPParserTool  string  `name:"mcp-parser-tool" help:"MCP parser tool name(s) to use for document parsing (comma-separated for fallback chain, e.g., parse_document,docling_parse,convert_document)" placeholder:"TOOL_NAME"`
	DocsFolder     string  `name:"docs-folder" help:"Folder containing documents for RAG" type:"path" placeholder:"PATH"`
	EmbedderModel  string  `name:"embedder-model" help:"Embedder model for document store" default:"nomic-embed-text"`
	VectorDB       string  `name:"vectordb" help:"Vector database connection string" default:"http://localhost:6334" placeholder:"URL"`
	Observe        bool    `help:"Enable observability (metrics + tracing to localhost:4317)"`
}

type ClientModeFlags struct {
	URL   string `help:"Agent card URL or service base URL (enables client mode)" env:"HECTOR_URL" placeholder:"URL"`
	Token string `help:"Authentication token for agent access" env:"HECTOR_TOKEN"`
}

type VersionCmd struct {
}

type ServeCmd struct {
	Port       int    `default:"8080" help:"HTTP server port (default: ${default})"`
	Host       string `help:"Server host address (overrides config)" placeholder:"HOST"`
	A2ABaseURL string `name:"a2a-base-url" help:"A2A base URL for agent discovery" placeholder:"URL"`

	ZeroConfigFlags `embed:"" prefix:""`

	AgentName string `arg:"" optional:"" help:"Agent name for zero-config mode (default: assistant)" placeholder:"AGENT"`
}

type InfoCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	Agent string `arg:"" help:"Agent name to get information about" placeholder:"AGENT"`
}

type CallCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	Stream    bool   `default:"true" negatable:"" help:"Enable streaming responses (use --no-stream to disable)"`
	SessionID string `name:"session" help:"Session ID for conversation continuity" env:"HECTOR_SESSION" placeholder:"SESSION_ID"`
	Agent     string `help:"Agent name (not used in client mode with --url)" placeholder:"AGENT"`

	ZeroConfigFlags `embed:"" prefix:""`

	Message string `arg:"" help:"Message to send to the agent" placeholder:"MESSAGE"`
}

type ChatCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	NoStream  bool   `name:"no-stream" help:"Disable streaming responses (streaming enabled by default)"`
	SessionID string `name:"session" help:"Session ID to resume previous conversation" env:"HECTOR_SESSION" placeholder:"SESSION_ID"`
	Agent     string `help:"Agent name (not used in client mode with --url)" placeholder:"AGENT"`

	ZeroConfigFlags `embed:"" prefix:""`
}

type TaskCmd struct {
	ClientModeFlags `embed:"" prefix:""`

	Get    TaskGetCmd    `cmd:"" help:"Get task details by ID"`
	Cancel TaskCancelCmd `cmd:"" help:"Cancel a running task by ID"`
}

type TaskGetCmd struct {
	Agent  string `arg:"" help:"Agent name that owns the task" placeholder:"AGENT"`
	TaskID string `arg:"" help:"Task ID to retrieve" placeholder:"TASK_ID"`
}

type TaskCancelCmd struct {
	Agent  string `arg:"" help:"Agent name that owns the task" placeholder:"AGENT"`
	TaskID string `arg:"" help:"Task ID to cancel" placeholder:"TASK_ID"`
}
