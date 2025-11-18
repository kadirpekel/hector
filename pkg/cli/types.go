package cli

type CommandType string

const (
	CommandVersion CommandType = "version"
	CommandServe   CommandType = "serve"
	CommandInfo    CommandType = "info"
	CommandCall    CommandType = "call"
	CommandChat    CommandType = "chat"
	CommandTask    CommandType = "task"
	CommandHelp    CommandType = "help"
)

type CLIMode string

const (
	ModeServerZeroConfig CLIMode = "server-zero-config"
	ModeServerConfig     CLIMode = "server-config"
	ModeClient           CLIMode = "client"
	ModeLocalZeroConfig  CLIMode = "local-zero-config"
	ModeLocalConfig      CLIMode = "local-config"
)

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

type CLIArgs struct {
	Command    CommandType
	ConfigFile string
	ServerURL  string
	AgentID    string
	TaskID     string
	TaskAction string
	Input      string
	Token      string
	Stream     bool
	LogLevel   string
	Port       int
	SessionID  string

	Host       string
	A2ABaseURL string

	Provider      string
	APIKey        string
	BaseURL       string
	Model         string
	Tools         bool
	MCPURL        string
	DocsFolder    string
	EmbedderModel string
	VectorDB      string

	Mode CLIMode
}
