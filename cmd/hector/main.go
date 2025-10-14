package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/cli"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/transport"
)

// ============================================================================
// VERSION
// ============================================================================

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}
	return "dev"
}

// ============================================================================
// COMMAND TYPES
// ============================================================================

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

type CLIMode string

const (
	ModeServer CLIMode = "server" // Host agents via 'serve' command
	ModeClient CLIMode = "client" // Connect to remote server (--server flag)
	ModeDirect CLIMode = "direct" // In-process execution (no --server)
)

func (m CLIMode) String() string {
	switch m {
	case ModeServer:
		return "Server"
	case ModeClient:
		return "Client (Remote)"
	case ModeDirect:
		return "Direct (Local)"
	default:
		return string(m)
	}
}

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

	// Zero-config mode options (OpenAI-based)
	APIKey        string
	BaseURL       string
	Model         string
	Tools         bool
	MCPURL        string
	DocsFolder    string
	EmbedderModel string
	VectorDB      string
}

// ============================================================================
// MULTI-AGENT SERVICE
// ============================================================================

type MultiAgentService struct {
	pb.UnimplementedA2AServiceServer
	agents   map[string]pb.A2AServiceServer
	metadata map[string]*transport.AgentMetadata
	registry *agent.AgentRegistry
}

func NewMultiAgentService(registry *agent.AgentRegistry) *MultiAgentService {
	return &MultiAgentService{
		agents:   make(map[string]pb.A2AServiceServer),
		metadata: make(map[string]*transport.AgentMetadata),
		registry: registry,
	}
}

func (s *MultiAgentService) RegisterAgent(agentID string, agentSvc pb.A2AServiceServer) {
	s.agents[agentID] = agentSvc
	log.Printf("  ✅ Registered agent: %s", agentID)
}

// RegisterAgentWithMetadata registers an agent with its metadata
func (s *MultiAgentService) RegisterAgentWithMetadata(agentID string, agentSvc pb.A2AServiceServer, meta *transport.AgentMetadata) {
	s.agents[agentID] = agentSvc
	s.metadata[agentID] = meta
	log.Printf("  ✅ Registered agent: %s (visibility: %s)", agentID, meta.Visibility)
}

// GetAgentMetadata returns metadata for a specific agent (for discovery)
func (s *MultiAgentService) GetAgentMetadata(agentID string) (*transport.AgentMetadata, error) {
	if meta, ok := s.metadata[agentID]; ok {
		return meta, nil
	}

	// Fallback: create metadata from agent card
	if agentSvc, ok := s.agents[agentID]; ok {
		card, err := agentSvc.GetAgentCard(context.Background(), &pb.GetAgentCardRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get agent card: %w", err)
		}

		return &transport.AgentMetadata{
			ID:              agentID,
			Name:            card.Name,
			Description:     card.Description,
			Version:         card.Version,
			Visibility:      "public", // Default visibility
			Capabilities:    card.Capabilities,
			SecuritySchemes: card.SecuritySchemes,
			Security:        card.Security,
		}, nil
	}

	return nil, fmt.Errorf("agent not found: %s", agentID)
}

// ListAgents returns all registered agents (discovery)
func (s *MultiAgentService) ListAgents() []string {
	agents := make([]string, 0, len(s.agents))
	for agentID := range s.agents {
		agents = append(agents, agentID)
	}
	return agents
}

// GetAgent returns a specific agent by ID
func (s *MultiAgentService) GetAgent(agentID string) (pb.A2AServiceServer, bool) {
	agent, ok := s.agents[agentID]
	return agent, ok
}

// SendMessage routes to the appropriate agent
func (s *MultiAgentService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	// First try to get agent ID from gRPC metadata (set by REST gateway)
	agentID := s.extractAgentIDFromContext(ctx)

	// If not found, try extracting from request payload
	if agentID == "" {
		agentID = s.extractAgentID(req)
	}

	// REMOVED DANGEROUS FALLBACK: Do not auto-route to single agent if agentID is empty
	// This was causing wrong agent names to work incorrectly

	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id not specified (use context_id format: agent_id:session_id)")
	}

	agentSvc, ok := s.agents[agentID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", agentID)
	}

	return agentSvc.SendMessage(ctx, req)
}

// SendStreamingMessage routes to the appropriate agent
func (s *MultiAgentService) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	// First try to get agent ID from gRPC metadata (set by REST gateway)
	agentID := s.extractAgentIDFromContext(stream.Context())

	// If not found, try extracting from request payload
	if agentID == "" {
		agentID = s.extractAgentID(req)
	}

	// REMOVED DANGEROUS FALLBACK: Do not auto-route to single agent if agentID is empty
	// This was causing wrong agent names to work incorrectly

	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id not specified")
	}

	agentSvc, ok := s.agents[agentID]
	if !ok {
		return status.Errorf(codes.NotFound, "agent '%s' not found", agentID)
	}

	return agentSvc.SendStreamingMessage(req, stream)
}

// GetAgentCard returns card for specific agent or multi-agent summary
func (s *MultiAgentService) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	if len(s.agents) == 1 {
		for _, agentSvc := range s.agents {
			return agentSvc.GetAgentCard(ctx, req)
		}
	}

	agentNames := make([]string, 0, len(s.agents))
	for id := range s.agents {
		agentNames = append(agentNames, id)
	}

	return &pb.AgentCard{
		Name:        "Hector Multi-Agent Server",
		Description: fmt.Sprintf("Multi-agent server with %d agents: %s", len(s.agents), strings.Join(agentNames, ", ")),
		Version:     getVersion(),
		Capabilities: &pb.AgentCapabilities{
			Streaming: true,
		},
	}, nil
}

// Implement other A2A methods by routing to appropriate agent
func (s *MultiAgentService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	// Try to extract agent from task name
	// For now, route to first agent if single agent
	if len(s.agents) == 1 {
		for _, agentSvc := range s.agents {
			return agentSvc.GetTask(ctx, req)
		}
	}
	return nil, status.Error(codes.Unimplemented, "GetTask requires agent specification in multi-agent mode")
}

func (s *MultiAgentService) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	if len(s.agents) == 1 {
		for _, agentSvc := range s.agents {
			return agentSvc.CancelTask(ctx, req)
		}
	}
	return nil, status.Error(codes.Unimplemented, "CancelTask requires agent specification in multi-agent mode")
}

func (s *MultiAgentService) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	if len(s.agents) == 1 {
		for _, agentSvc := range s.agents {
			return agentSvc.TaskSubscription(req, stream)
		}
	}
	return status.Error(codes.Unimplemented, "TaskSubscription requires agent specification in multi-agent mode")
}

func (s *MultiAgentService) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *MultiAgentService) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *MultiAgentService) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *MultiAgentService) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *MultiAgentService) extractAgentID(req *pb.SendMessageRequest) string {
	if req.Request == nil {
		return ""
	}

	// Try context_id format: "agent_id:session_id"
	if req.Request.ContextId != "" {
		parts := strings.SplitN(req.Request.ContextId, ":", 2)
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Try metadata
	if req.Request.Metadata != nil {
		if agentID, ok := req.Request.Metadata.Fields["agent_id"]; ok {
			return agentID.GetStringValue()
		}
	}

	return ""
}

// extractAgentIDFromContext extracts agent ID from gRPC metadata (set by REST gateway)
func (s *MultiAgentService) extractAgentIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	agentNames := md.Get("agent-name")
	if len(agentNames) == 0 {
		return ""
	}

	return agentNames[0]
}

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Load environment variables
	if err := config.LoadEnvFiles(); err != nil && !os.IsNotExist(err) {
		fatalf("Failed to load environment files: %v", err)
	}

	args := parseArgs()

	// Convert CLIArgs to cli.Args
	cliArgs := cli.Args{
		ConfigFile: args.ConfigFile,
		ServerURL:  args.ServerURL,
		Token:      args.Token,
		AgentID:    args.AgentID,
		TaskID:     args.TaskID,
		Input:      args.Input,
		Stream:     args.Stream,
		Debug:      args.Debug,
		Port:       args.Port,
		Provider:   "openai", // Default provider
		APIKey:     args.APIKey,
		BaseURL:    args.BaseURL,
		Model:      args.Model,
		Tools:      args.Tools,
		MCPURL:     args.MCPURL,
		DocsFolder: args.DocsFolder,
	}

	// Route to appropriate handler using new CLI package
	switch args.Command {
	case CommandServe:
		// Serve command still uses old implementation for server lifecycle management
		executeServeCommand(args)
	case CommandList:
		if err := cli.ListCommand(cliArgs); err != nil {
			fatalf("List command failed: %v", err)
		}
	case CommandInfo:
		if err := cli.InfoCommand(cliArgs); err != nil {
			fatalf("Info command failed: %v", err)
		}
	case CommandCall:
		if err := cli.CallCommand(cliArgs); err != nil {
			fatalf("Call command failed: %v", err)
		}
	case CommandChat:
		if err := cli.ChatCommand(cliArgs); err != nil {
			fatalf("Chat command failed: %v", err)
		}
	case CommandTask:
		// Task subcommands
		switch args.TaskAction {
		case "get":
			if err := cli.TaskGetCommand(cliArgs); err != nil {
				fatalf("Task get command failed: %v", err)
			}
		case "cancel":
			if err := cli.TaskCancelCommand(cliArgs); err != nil {
				fatalf("Task cancel command failed: %v", err)
			}
		default:
			fatalf("Unknown task action: %s (use 'get' or 'cancel')", args.TaskAction)
		}
	case CommandHelp:
		showHelp()
	default:
		showHelp()
	}
}

// ============================================================================
// ARGUMENT PARSING
// ============================================================================

func parseArgs() *CLIArgs {
	args := &CLIArgs{}

	// Define subcommands
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveConfig := serveCmd.String("config", "hector.yaml", "Configuration file")
	servePort := serveCmd.Int("port", 8080, "gRPC server port (matches A2A server default)")
	serveDebug := serveCmd.Bool("debug", false, "Enable debug mode")

	// A2A Server override flags
	serveHost := serveCmd.String("host", "", "Server host (overrides config)")
	serveA2ABaseURL := serveCmd.String("a2a-base-url", "", "A2A base URL for discovery (overrides config)")

	// Zero-config mode flags (OpenAI-based)
	serveAPIKey := serveCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY)")
	serveBaseURL := serveCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	serveModel := serveCmd.String("model", "gpt-4o-mini", "OpenAI model to use in zero-config mode")
	serveTools := serveCmd.Bool("tools", false, "Enable all local tools (file, command execution)")
	serveMCP := serveCmd.String("mcp-url", "", "MCP server URL for tool integration (supports auth: https://user:pass@host)")
	serveDocs := serveCmd.String("docs", "", "Document store folder (enables RAG)")
	serveEmbedder := serveCmd.String("embedder-model", "nomic-embed-text", "Embedder model for document store")
	serveVectorDB := serveCmd.String("vectordb", "http://localhost:6333", "Vector database connection string")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listServer := listCmd.String("server", "", "A2A server URL (enables server mode)")
	listToken := listCmd.String("token", "", "Authentication token")
	listConfig := listCmd.String("config", "hector.yaml", "Configuration file (direct mode)")

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	infoServer := infoCmd.String("server", "", "A2A server URL (enables server mode)")
	infoToken := infoCmd.String("token", "", "Authentication token")
	infoConfig := infoCmd.String("config", "hector.yaml", "Configuration file (direct mode)")

	callCmd := flag.NewFlagSet("call", flag.ExitOnError)
	callServer := callCmd.String("server", "", "A2A server URL (enables server mode)")
	callToken := callCmd.String("token", "", "Authentication token")
	callStream := callCmd.Bool("stream", true, "Enable streaming (default: true)")
	callConfig := callCmd.String("config", "hector.yaml", "Configuration file (direct mode)")
	callAPIKey := callCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY)")
	callBaseURL := callCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	callModel := callCmd.String("model", "gpt-4o-mini", "OpenAI model (direct mode, zero-config)")
	callTools := callCmd.Bool("tools", false, "Enable tools (direct mode, zero-config)")
	callMCP := callCmd.String("mcp-url", "", "MCP server URL for tool integration (direct mode, zero-config)")

	chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)
	chatServer := chatCmd.String("server", "", "A2A server URL (enables server mode)")
	chatToken := chatCmd.String("token", "", "Authentication token")
	chatConfig := chatCmd.String("config", "hector.yaml", "Configuration file (direct mode)")
	chatAPIKey := chatCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY)")
	chatBaseURL := chatCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	chatModel := chatCmd.String("model", "gpt-4o-mini", "OpenAI model (direct mode, zero-config)")
	chatTools := chatCmd.Bool("tools", false, "Enable tools (direct mode, zero-config)")
	chatMCP := chatCmd.String("mcp-url", "", "MCP server URL for tool integration (direct mode, zero-config)")
	chatNoStream := chatCmd.Bool("no-stream", false, "Disable streaming (default: streaming enabled)")

	taskCmd := flag.NewFlagSet("task", flag.ExitOnError)
	taskServer := taskCmd.String("server", "", "A2A server URL (enables server mode)")
	taskToken := taskCmd.String("token", "", "Authentication token")
	taskConfig := taskCmd.String("config", "hector.yaml", "Configuration file (direct mode)")

	// Parse command
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		_ = serveCmd.Parse(os.Args[2:])
		args.Command = CommandServe
		args.ConfigFile = *serveConfig
		args.Port = *servePort
		args.Debug = *serveDebug
		args.Host = *serveHost
		args.A2ABaseURL = *serveA2ABaseURL
		args.APIKey = *serveAPIKey
		args.BaseURL = *serveBaseURL
		args.Model = *serveModel
		args.Tools = *serveTools
		args.MCPURL = *serveMCP
		args.DocsFolder = *serveDocs
		args.EmbedderModel = *serveEmbedder
		args.VectorDB = *serveVectorDB

	case "list":
		_ = listCmd.Parse(os.Args[2:])
		args.Command = CommandList
		args.ServerURL = *listServer // Don't resolve yet - let command detect mode
		args.Token = *listToken
		args.ConfigFile = *listConfig

	case "info":
		_ = infoCmd.Parse(os.Args[2:])
		if len(infoCmd.Args()) < 1 {
			fatalf("Usage: hector info <agent> [OPTIONS]")
		}
		args.Command = CommandInfo
		args.AgentID = infoCmd.Args()[0]
		args.ServerURL = *infoServer // Don't resolve yet
		args.Token = *infoToken
		args.ConfigFile = *infoConfig

	case "call":
		_ = callCmd.Parse(os.Args[2:])
		if len(callCmd.Args()) < 2 {
			fatalf("Usage: hector call <agent> \"prompt\" [OPTIONS]")
		}
		args.Command = CommandCall
		args.AgentID = callCmd.Args()[0]
		args.Input = callCmd.Args()[1]
		args.ServerURL = *callServer // Don't resolve yet
		args.Token = *callToken
		args.Stream = *callStream
		args.ConfigFile = *callConfig
		args.APIKey = *callAPIKey
		args.BaseURL = *callBaseURL
		args.Model = *callModel
		args.Tools = *callTools
		args.MCPURL = *callMCP

	case "chat":
		_ = chatCmd.Parse(os.Args[2:])
		if len(chatCmd.Args()) < 1 {
			fatalf("Usage: hector chat <agent> [OPTIONS]")
		}
		args.Command = CommandChat
		args.AgentID = chatCmd.Args()[0]
		args.ServerURL = *chatServer // Don't resolve yet
		args.Token = *chatToken
		args.ConfigFile = *chatConfig
		args.APIKey = *chatAPIKey
		args.BaseURL = *chatBaseURL
		args.Model = *chatModel
		args.Tools = *chatTools
		args.MCPURL = *chatMCP
		args.Stream = !*chatNoStream // Streaming is default, --no-stream disables it

	case "task":
		_ = taskCmd.Parse(os.Args[2:])
		if len(taskCmd.Args()) < 3 {
			fatalf("Usage: hector task <action> <agent> <task-id> [OPTIONS]\n" +
				"Actions: get, cancel")
		}
		args.Command = CommandTask
		args.TaskAction = taskCmd.Args()[0]
		args.AgentID = taskCmd.Args()[1]
		args.TaskID = taskCmd.Args()[2]
		args.ServerURL = *taskServer
		args.Token = *taskToken
		args.ConfigFile = *taskConfig

	case "help", "--help", "-h":
		args.Command = CommandHelp

	case "version", "--version", "-v":
		fmt.Printf("Hector %s\n", getVersion())
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showHelp()
		os.Exit(1)
	}

	// Validate mode and flags after parsing
	validateModeAndFlags(args)

	return args
}

// ============================================================================
// MODE DETECTION & VALIDATION
// ============================================================================

// detectMode determines which CLI mode is active
func detectMode(args *CLIArgs) CLIMode {
	if args.Command == CommandServe {
		return ModeServer
	}

	// Check --server flag
	if args.ServerURL != "" {
		return ModeClient
	}

	return ModeDirect
}

// validateModeAndFlags checks for invalid flag combinations and fails fast
func validateModeAndFlags(args *CLIArgs) {
	mode := detectMode(args)

	// Validate based on mode
	switch mode {
	case ModeServer:
		// Server mode: all flags are valid
		return

	case ModeClient:
		// Client mode: ONLY --server, --token, --stream allowed
		// Configuration flags are NOT supported
		if args.ConfigFile != "hector.yaml" && args.ConfigFile != "" {
			fatalf(`❌ Error: --config flag is not supported in %s mode

You're connecting to a remote server which has its own configuration.

Solutions:
  • Remove --config flag to use the remote server's configuration
  • Remove --server flag (or unset HECTOR_SERVER) to use Direct mode with local config

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		// Zero-config flags not allowed in client mode
		if args.APIKey != "" {
			fatalf(`❌ Error: --api-key flag is not supported in %s mode

The remote server has its own LLM configuration.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.Model != "" && args.Model != "gpt-4o-mini" { // Check if non-default
			fatalf(`❌ Error: --model flag is not supported in %s mode

The remote server has its own model configuration.

Solutions:
  • Remove --model flag to use the remote server's models
  • Use Direct mode (remove --server) for local model selection

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.Tools {
			fatalf(`❌ Error: --tools flag is not supported in %s mode

The remote server controls which tools are enabled.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.MCPURL != "" {
			fatalf(`❌ Error: --mcp-url flag is not supported in %s mode

The remote server controls which MCP servers are configured.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.BaseURL != "" && args.BaseURL != "https://api.openai.com/v1" { // Check if non-default
			fatalf(`❌ Error: --base-url flag is not supported in %s mode

The remote server has its own API configuration.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

	case ModeDirect:
		// Direct mode: all flags valid, but check for conflicting config strategies
		hasConfigFile := args.ConfigFile != "" && args.ConfigFile != "hector.yaml"
		hasZeroConfig := args.APIKey != "" || args.Model != "gpt-4o-mini" || args.Tools

		// Check if config file exists
		configExists := false
		if args.ConfigFile != "" {
			if _, err := os.Stat(args.ConfigFile); err == nil {
				configExists = true
			}
		}

		if hasConfigFile && configExists && hasZeroConfig {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Both --config and zero-config flags provided\n")
			fmt.Fprintf(os.Stderr, "   Zero-config flags (--api-key, --model, --tools) will be ignored.\n")
			fmt.Fprintf(os.Stderr, "   Using configuration from: %s\n\n", args.ConfigFile)
		}
	}
}

// ============================================================================
// SERVE COMMAND
// ============================================================================

func executeServeCommand(args *CLIArgs) {

	// Check if config file exists
	var hectorConfig *config.Config
	if _, err := os.Stat(args.ConfigFile); os.IsNotExist(err) {
		// Zero-config mode
		if args.Debug {
			fmt.Println("🔧 No config file found, entering zero-config mode")
		}

		// Get API key from flag or environment
		// Support multiple providers: OpenAI, Anthropic, Gemini
		apiKey := args.APIKey
		if apiKey == "" {
			// Try provider-specific env vars
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				apiKey = key
			} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				apiKey = key
			} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
				apiKey = key
			}
		}
		if apiKey == "" {
			fatalf("API key required for zero-config mode\nSet one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY\nOr use --api-key flag")
		}

		// Get MCP URL from flag or environment
		mcpURL := args.MCPURL
		if mcpURL == "" {
			mcpURL = os.Getenv("MCP_SERVER_URL")
		}

		opts := config.ZeroConfigOptions{
			APIKey:      apiKey,
			BaseURL:     args.BaseURL,
			Model:       args.Model,
			EnableTools: args.Tools,
			MCPURL:      mcpURL,
			DocsFolder:  args.DocsFolder,
		}

		hectorConfig = config.CreateZeroConfig(opts)

		if args.Debug {
			fmt.Printf("  Provider: OpenAI\n")
			fmt.Printf("  Model: %s\n", args.Model)
			fmt.Printf("  Base URL: %s\n", args.BaseURL)
			if args.Tools {
				fmt.Println("  Tools: Enabled")
			}
			if args.MCPURL != "" {
				fmt.Printf("  MCP: %s\n", args.MCPURL)
			}
			if args.DocsFolder != "" {
				fmt.Printf("  Docs: %s\n", args.DocsFolder)
			}
		}
	} else {
		// Load configuration from file
		var err error
		hectorConfig, err = config.LoadConfig(args.ConfigFile)
		if err != nil {
			fatalf("Failed to load config: %v", err)
		}
	}

	// Set defaults and validate
	hectorConfig.SetDefaults()
	if err := hectorConfig.Validate(); err != nil {
		fatalf("Invalid configuration: %v", err)
	}

	// Create agent registry
	agentRegistry := agent.NewAgentRegistry()

	// Create component manager with agent registry for agent_call tool
	componentManager, err := component.NewComponentManagerWithAgentRegistry(hectorConfig, agentRegistry)
	if err != nil {
		fatalf("Component initialization failed: %v", err)
	}

	// Create multi-agent service
	multiAgentSvc := NewMultiAgentService(agentRegistry)

	// Register all configured agents
	fmt.Println("\n📋 Registering agents...")
	for agentID, agentCfg := range hectorConfig.Agents {
		cfg := agentCfg

		// Create agent based on type (native vs external)
		var agentInstance pb.A2AServiceServer
		var err error

		if cfg.Type == "a2a" {
			// External A2A agent - create client proxy
			externalAgent, extErr := agent.NewExternalA2AAgent(&cfg)
			if extErr != nil {
				log.Printf("  ⚠️  Failed to create external agent '%s': %v", agentID, extErr)
				continue
			}
			agentInstance = externalAgent
			log.Printf("  ✅ External agent '%s' connected to %s", agentID, cfg.URL)
		} else {
			// Native agent - create local instance
			agentInstance, err = agent.NewAgent(&cfg, componentManager, agentRegistry)
			if err != nil {
				log.Printf("  ⚠️  Failed to create native agent '%s': %v", agentID, err)
				continue
			}
			log.Printf("  ✅ Native agent '%s' created", agentID)
		}

		// Get agent card for metadata
		card, cardErr := agentInstance.GetAgentCard(context.Background(), &pb.GetAgentCardRequest{})
		if cardErr != nil {
			log.Printf("  ⚠️  Failed to get agent card for '%s': %v", agentID, cardErr)
			continue
		}

		// Set default visibility
		visibility := cfg.Visibility
		if visibility == "" {
			visibility = "public" // Default to public
		}

		// Register with metadata
		metadata := &transport.AgentMetadata{
			ID:              agentID,
			Name:            cfg.Name,
			Description:     cfg.Description,
			Version:         "1.0.0",
			Visibility:      visibility,
			Capabilities:    card.GetCapabilities(),
			SecuritySchemes: card.SecuritySchemes,
			Security:        card.Security,
		}
		multiAgentSvc.RegisterAgentWithMetadata(agentID, agentInstance, metadata)

		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &cfg, nil); err != nil {
			log.Printf("  ⚠️  Failed to register agent '%s' in registry: %v", agentID, err)
		}
	}

	if len(multiAgentSvc.agents) == 0 {
		log.Fatalf("❌ No agents successfully registered")
	}

	// Determine addresses - CLI flags override config (conventional pattern)
	var basePort int
	var serverHost string
	var baseURL string
	var overrides []string

	// Start with defaults
	basePort = 8080 // Match A2A server default
	serverHost = "0.0.0.0"
	baseURL = ""

	// Load config values if A2A server is configured (presence implies enabled)
	hasA2AConfig := hectorConfig.Global.A2AServer.IsEnabled()
	if hasA2AConfig {
		if hectorConfig.Global.A2AServer.Port > 0 {
			basePort = hectorConfig.Global.A2AServer.Port
		}
		if hectorConfig.Global.A2AServer.Host != "" {
			serverHost = hectorConfig.Global.A2AServer.Host
		}
		if hectorConfig.Global.A2AServer.BaseURL != "" {
			baseURL = hectorConfig.Global.A2AServer.BaseURL
		}
	}

	// CLI flags override config (conventional behavior)
	if args.Port != 8080 { // User specified --port (different from default)
		if hasA2AConfig && hectorConfig.Global.A2AServer.Port > 0 && args.Port != hectorConfig.Global.A2AServer.Port {
			overrides = append(overrides, fmt.Sprintf("port: %d (config: %d)", args.Port, hectorConfig.Global.A2AServer.Port))
		}
		basePort = args.Port
	}

	if args.Host != "" { // User specified --host
		if hasA2AConfig && hectorConfig.Global.A2AServer.Host != "" && args.Host != hectorConfig.Global.A2AServer.Host {
			overrides = append(overrides, fmt.Sprintf("host: %s (config: %s)", args.Host, hectorConfig.Global.A2AServer.Host))
		}
		serverHost = args.Host
	}

	if args.A2ABaseURL != "" { // User specified --a2a-base-url
		if hasA2AConfig && hectorConfig.Global.A2AServer.BaseURL != "" && args.A2ABaseURL != hectorConfig.Global.A2AServer.BaseURL {
			overrides = append(overrides, fmt.Sprintf("base_url: %s (config: %s)", args.A2ABaseURL, hectorConfig.Global.A2AServer.BaseURL))
		}
		baseURL = args.A2ABaseURL
	}

	if args.Debug {
		if hasA2AConfig {
			fmt.Printf("🔧 A2A server config loaded:\n")
			fmt.Printf("   Host: %s\n", serverHost)
			fmt.Printf("   Port: %d\n", basePort)
			if baseURL != "" {
				fmt.Printf("   Base URL: %s\n", baseURL)
			}
			if len(overrides) > 0 {
				fmt.Printf("🚨 CLI overrides: %s\n", strings.Join(overrides, ", "))
			}
		} else {
			fmt.Printf("🔧 Using defaults with CLI port: %s:%d\n", serverHost, basePort)
		}
	}

	grpcAddr := fmt.Sprintf("%s:%d", serverHost, basePort)
	restAddr := fmt.Sprintf("%s:%d", serverHost, basePort+1)
	jsonrpcAddr := fmt.Sprintf("%s:%d", serverHost, basePort+2)

	// Configure authentication
	var authConfig *transport.AuthConfig
	var jwtValidator *auth.JWTValidator
	if hectorConfig.Global.Auth.IsEnabled() {
		var err error
		jwtValidator, err = auth.NewJWTValidator(
			hectorConfig.Global.Auth.JWKSURL,
			hectorConfig.Global.Auth.Issuer,
			hectorConfig.Global.Auth.Audience,
		)
		if err != nil {
			log.Printf("⚠️  Failed to initialize JWT validator: %v", err)
		} else {
			authConfig = &transport.AuthConfig{
				Enabled:   true,
				Validator: jwtValidator,
			}
			log.Printf("✅ Authentication configured (JWT)")
		}
	}

	// Create gRPC server with auth interceptors
	grpcConfig := transport.Config{
		Address: grpcAddr,
	}
	if jwtValidator != nil {
		grpcConfig.UnaryInterceptor = jwtValidator.UnaryServerInterceptor()
		grpcConfig.StreamInterceptor = jwtValidator.StreamServerInterceptor()
	}
	grpcServer := transport.NewServer(multiAgentSvc, grpcConfig)

	// Create REST gateway with auth
	restGateway := transport.NewRESTGateway(transport.RESTGatewayConfig{
		HTTPAddress: restAddr,
		GRPCAddress: grpcAddr, // Point to the correct gRPC server
	})
	if authConfig != nil {
		restGateway.SetAuth(authConfig)
	}

	// Set up agent discovery endpoint
	discovery := transport.NewAgentDiscovery(multiAgentSvc, authConfig)
	restGateway.SetDiscovery(discovery)

	jsonrpcHandler := transport.NewJSONRPCHandler(
		transport.JSONRPCConfig{HTTPAddress: jsonrpcAddr},
		multiAgentSvc,
	)
	if authConfig != nil {
		jsonrpcHandler.SetAuth(authConfig)
	}

	// Start all servers
	errChan := make(chan error, 3)

	go func() {
		if err := grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		if err := restGateway.Start(context.Background()); err != nil {
			errChan <- fmt.Errorf("REST gateway error: %w", err)
		}
	}()

	go func() {
		if err := jsonrpcHandler.Start(); err != nil {
			errChan <- fmt.Errorf("JSON-RPC handler error: %w", err)
		}
	}()

	log.Printf("\n🎉 Hector v%s - All transports started!", getVersion())
	log.Printf("📡 Agents available: %d", len(multiAgentSvc.agents))
	for agentID := range multiAgentSvc.agents {
		log.Printf("   • %s", agentID)
	}
	log.Printf("\n🌐 Endpoints:")
	log.Printf("   → gRPC:     %s", grpcServer.Address())
	log.Printf("   → REST:     http://%s:%d", serverHost, basePort+1)
	log.Printf("   → JSON-RPC: http://%s:%d/rpc", serverHost, basePort+2)
	log.Printf("\n📋 Discovery & Agent Cards:")
	if baseURL != "" {
		log.Printf("   → Service Card: %s/.well-known/agent-card.json", baseURL)
		log.Printf("   → Agent List:   %s/v1/agents", baseURL)
		log.Printf("   → Agent Cards:  %s/v1/agents/{agent_id}/.well-known/agent-card.json", baseURL)
	} else {
		log.Printf("   → Service Card: http://%s:%d/.well-known/agent-card.json", serverHost, basePort+1)
		log.Printf("   → Agent List:   http://%s:%d/v1/agents", serverHost, basePort+1)
		log.Printf("   → Agent Cards:  http://%s:%d/v1/agents/{agent_id}/.well-known/agent-card.json", serverHost, basePort+1)
	}
	log.Printf("\n💡 A2A-compliant endpoints (per agent):")
	endpointBase := baseURL
	if endpointBase == "" {
		endpointBase = fmt.Sprintf("http://%s:%d", serverHost, basePort+1)
	}
	log.Printf("   POST %s/v1/agents/{agent_id}/message:send", endpointBase)
	log.Printf("   POST %s/v1/agents/{agent_id}/message:stream", endpointBase)
	log.Printf("\n💡 Test commands:")
	log.Printf("   hector list")
	log.Printf("   hector info <agent>")
	log.Printf("   hector call <agent> \"your prompt\"")
	log.Printf("   hector chat <agent>")
	log.Println("\nPress Ctrl+C to stop")

	// Wait for signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("\n🛑 Shutting down...")
	case err := <-errChan:
		log.Printf("\n❌ Server error: %v", err)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErrors []error
	if err := grpcServer.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("gRPC: %w", err))
	}
	if err := restGateway.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("REST: %w", err))
	}
	if err := jsonrpcHandler.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("JSON-RPC: %w", err))
	}

	if len(shutdownErrors) > 0 {
		log.Printf("⚠️  Errors during shutdown:")
		for _, err := range shutdownErrors {
			log.Printf("   - %v", err)
		}
		os.Exit(1)
	}

	log.Printf("👋 All servers shut down gracefully")
}

// ============================================================================
// HELP
// ============================================================================

func showHelp() {
	fmt.Print(`
Hector - AI Agent Platform

USAGE:
  hector <command> [options]

COMMANDS:
  serve              Start A2A server to host agents
  list               List available agents
  info <agent>       Get agent information
  call <agent> "..."  Execute a task on an agent
  chat <agent>       Start interactive chat
  task <action> <agent> <task-id>  Manage tasks (actions: get, cancel)
  help               Show this help message
  version            Show version information

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
THREE MODES OF OPERATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Hector operates in three distinct modes based on your command and flags:

1️⃣  SERVER MODE - Host agents for multiple clients
   Trigger: 'serve' command
   Use when: Production deployments, multi-agent systems, team access
   Supports: --config AND zero-config flags

2️⃣  CLIENT MODE - Connect to remote Hector server
   Trigger: --server flag
   Use when: Accessing remote/production servers, team collaboration
   Supports: ONLY --server, --token, --stream
   ⚠️  --config and zero-config flags NOT supported (server has its own config)

3️⃣  DIRECT MODE - Run agents in-process (no server)
   Trigger: No --server flag
   Use when: Quick tasks, local development, scripts, experimentation
   Supports: --config AND zero-config flags

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔧 SERVER MODE - Start persistent server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  hector serve [options]
    --config FILE            Configuration file (default: hector.yaml)
    --port PORT              gRPC server port (default: 8080, overrides config)
    --host HOST              Server host (overrides config)
    --a2a-base-url URL       A2A base URL for discovery (overrides config)
    --debug                  Enable debug output
    
  Zero-Config Options (when hector.yaml doesn't exist):
    --model MODEL            Model name (default: gpt-4o-mini)
    --api-key KEY            API key (or set env var, see below)
    --tools                  Enable all local tools
    --mcp-url URL            MCP server URL (supports auth: https://user:pass@host)
    --docs FOLDER            Document store folder (RAG)
    --embedder-model MODEL   Embedder model (default: nomic-embed-text)
    --vectordb URL           Vector DB (default: http://localhost:6333)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🌐 CLIENT MODE - Connect to remote server
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  hector list [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  hector info <agent> [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  hector call <agent> "prompt" [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token
    --stream BOOL    Enable streaming (default: true)

  hector chat <agent> [options]
    --server URL     Server URL (triggers client mode)
    --token TOKEN    Authentication token

  ⚠️  Important: --config, --model, --tools, --api-key flags are NOT supported
      in client mode. The remote server uses its own configuration.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

💻 DIRECT MODE - In-process execution (no server)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Same commands as Client mode, but WITHOUT --server flag:

  hector list [--config FILE]
  hector info <agent> [--config FILE]
  hector call <agent> "prompt" [--config FILE] [zero-config options]
  hector chat <agent> [--config FILE] [zero-config options]

  With Config File:
    --config FILE    Configuration file (default: hector.yaml)

  Zero-Config Options (for call and chat):
    --api-key KEY    API key (or set env var, see below)
    --base-url URL   API base URL
    --model MODEL    Model name (default: gpt-4o-mini)
    --tools          Enable local tools
    --mcp-url URL    MCP server URL (supports auth: https://user:pass@host)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

EXAMPLES:
  
  Server Mode - Host agents:
    $ hector serve                                    # Use config file
    $ hector serve --model gpt-4o --tools             # Zero-config mode
    $ hector serve --config prod.yaml --port 9090     # Custom config & port
  
  Client Mode - Connect to remote server:
    $ hector list --server http://remote:8080         # List remote agents
    $ hector call assistant "task" --server URL       # Execute on remote
    $ hector chat coder --server URL --token abc123   # Chat with auth
  
  Direct Mode - In-process execution:
    $ hector list                                     # List from local config
    $ hector call assistant "task"                    # Zero-config (fastest!)
    $ hector call assistant "task" --config my.yaml  # Use specific config
    $ hector call assistant "task" --model gpt-4o    # Override model
    $ hector chat assistant --tools                   # Enable tools

  Mode Selection Examples:
    # Same command, different modes:
    $ hector call agent "task"                # Direct mode (local)
    $ hector call agent "task" --server URL   # Client mode (remote)

ENVIRONMENT VARIABLES:
  API Keys (for zero-config mode):
    OPENAI_API_KEY       OpenAI API key
    ANTHROPIC_API_KEY    Anthropic (Claude) API key
    GEMINI_API_KEY       Google Gemini API key
  
  MCP Configuration:
    MCP_SERVER_URL       MCP server URL (supports auth: https://user:pass@host)

MODE DETECTION:
  • If you use 'serve' command → Server mode
  • If you use --server flag → Client mode
  • Otherwise → Direct mode

For more information: https://github.com/kadirpekel/hector
`)
}

// ============================================================================
// UTILITIES
// ============================================================================

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
