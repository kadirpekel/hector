package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/tools"
)

// ============================================================================
// TYPES AND CONSTANTS
// ============================================================================

const (
	defaultConfigFile = "hector.yaml"

	// CLI flag descriptions
	serverFlagDesc = "A2A server URL (default: localhost:8080)"
	tokenFlagDesc  = "Authentication token"
)

// getVersion returns the version from build info (Git tag) or "dev" if not available
func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		// If built with go install, this will be the Git tag
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}
	return "dev"
}

// CommandType represents the type of command to execute
type CommandType string

const (
	CommandServe CommandType = "serve"
	CommandList  CommandType = "list"
	CommandInfo  CommandType = "info"
	CommandCall  CommandType = "call"
	CommandChat  CommandType = "chat"
	CommandHelp  CommandType = "help"
)

// CLIArgs holds parsed command line arguments
type CLIArgs struct {
	Command    CommandType
	ConfigFile string
	ServerURL  string
	AgentURL   string
	Input      string
	Token      string
	Stream     bool
	Debug      bool

	// Zero-config mode options (OpenAI-based)
	APIKey     string
	BaseURL    string
	Model      string
	Tools      bool
	MCPURL     string
	DocsFolder string
}

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	args := parseArgs()

	// Load environment variables from .env files (for all commands)
	if err := config.LoadEnvFiles(); err != nil {
		if !os.IsNotExist(err) {
			fatalf("Failed to load environment files: %v", err)
		}
	}

	// Route to appropriate handler
	switch args.Command {
	case CommandServe:
		executeServeCommand(args)
	case CommandList:
		if err := executeListCommand(args); err != nil {
			fatalf("List command failed: %v", err)
		}
	case CommandInfo:
		if err := executeInfoCommand(args, args.AgentURL); err != nil {
			fatalf("Info command failed: %v", err)
		}
	case CommandCall:
		if err := executeCallCommand(args, args.AgentURL, args.Input); err != nil {
			fatalf("Call command failed: %v", err)
		}
	case CommandChat:
		if err := executeChatCommand(args, args.AgentURL); err != nil {
			fatalf("Chat command failed: %v", err)
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

	// Define flags
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveConfig := serveCmd.String("config", defaultConfigFile, "Configuration file")
	serveDebug := serveCmd.Bool("debug", false, "Enable debug mode")

	// Zero-config mode flags (OpenAI-based)
	serveAPIKey := serveCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	serveBaseURL := serveCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	serveModel := serveCmd.String("model", "gpt-4o-mini", "OpenAI model to use in zero-config mode")
	serveTools := serveCmd.Bool("tools", false, "Enable all local tools (file, command execution)")
	serveMCP := serveCmd.String("mcp", "", "MCP server URL for tool integration")
	serveDocs := serveCmd.String("docs", "", "Document store folder (enables RAG)")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listServer := listCmd.String("server", "", serverFlagDesc)
	listToken := listCmd.String("token", "", tokenFlagDesc)
	listConfig := listCmd.String("config", defaultConfigFile, "Configuration file (direct mode)")

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	infoServer := infoCmd.String("server", "", serverFlagDesc)
	infoToken := infoCmd.String("token", "", tokenFlagDesc)
	infoConfig := infoCmd.String("config", defaultConfigFile, "Configuration file (direct mode)")

	callCmd := flag.NewFlagSet("call", flag.ExitOnError)
	callServer := callCmd.String("server", "", serverFlagDesc)
	callToken := callCmd.String("token", "", tokenFlagDesc)
	callStream := callCmd.Bool("stream", true, "Enable streaming (default: true, use --stream=false to disable)")
	callConfig := callCmd.String("config", defaultConfigFile, "Configuration file (direct mode)")
	callAPIKey := callCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	callBaseURL := callCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	callModel := callCmd.String("model", "gpt-4o-mini", "OpenAI model (direct mode, zero-config)")
	callTools := callCmd.Bool("tools", false, "Enable tools (direct mode, zero-config)")

	chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)
	chatServer := chatCmd.String("server", "", serverFlagDesc)
	chatToken := chatCmd.String("token", "", tokenFlagDesc)
	chatConfig := chatCmd.String("config", defaultConfigFile, "Configuration file (direct mode)")
	chatAPIKey := chatCmd.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	chatBaseURL := chatCmd.String("base-url", "https://api.openai.com/v1", "OpenAI API base URL")
	chatModel := chatCmd.String("model", "gpt-4o-mini", "OpenAI model (direct mode, zero-config)")
	chatTools := chatCmd.Bool("tools", false, "Enable tools (direct mode, zero-config)")

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
		args.Debug = *serveDebug
		args.APIKey = *serveAPIKey
		args.BaseURL = *serveBaseURL
		args.Model = *serveModel
		args.Tools = *serveTools
		args.MCPURL = *serveMCP
		args.DocsFolder = *serveDocs

	case "list":
		_ = listCmd.Parse(os.Args[2:])
		args.Command = CommandList
		args.ServerURL = *listServer // Don't resolve yet - let mode detection handle it
		args.Token = *listToken
		args.ConfigFile = *listConfig

	case "info":
		_ = infoCmd.Parse(os.Args[2:])
		if len(infoCmd.Args()) < 1 {
			fatalf("Usage: hector info <agent> [--server URL | --config FILE]")
		}
		args.Command = CommandInfo
		args.AgentURL = infoCmd.Args()[0] // Store raw agent ID
		args.ServerURL = *infoServer      // Store raw server URL
		args.Token = *infoToken
		args.ConfigFile = *infoConfig

	case "call":
		// Parse remaining args
		// Note: Go's flag package requires flags BEFORE positional arguments
		_ = callCmd.Parse(os.Args[2:])

		callArgs := callCmd.Args()
		if len(callArgs) < 2 {
			fatalf("Usage: hector call [OPTIONS] <agent> \"prompt\"\nNote: Flags must come before the agent name")
		}

		args.Command = CommandCall
		args.AgentURL = callArgs[0]  // Agent ID
		args.Input = callArgs[1]     // Prompt
		args.ServerURL = *callServer // Server URL from --server flag
		args.Token = *callToken
		args.Stream = *callStream
		args.ConfigFile = *callConfig
		args.APIKey = *callAPIKey
		args.BaseURL = *callBaseURL
		args.Model = *callModel
		args.Tools = *callTools

	case "chat":
		// Parse remaining args
		// Note: Go's flag package requires flags BEFORE positional arguments
		_ = chatCmd.Parse(os.Args[2:])

		// Get non-flag arguments (agent ID)
		chatArgs := chatCmd.Args()
		if len(chatArgs) < 1 {
			fatalf("Usage: hector chat [OPTIONS] <agent>\nNote: Flags must come before the agent name")
		}

		args.Command = CommandChat
		args.AgentURL = chatArgs[0]  // Agent ID (first non-flag arg)
		args.ServerURL = *chatServer // Server URL from --server flag
		args.Token = *chatToken
		args.ConfigFile = *chatConfig
		args.APIKey = *chatAPIKey
		args.BaseURL = *chatBaseURL
		args.Model = *chatModel
		args.Tools = *chatTools

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

	return args
}

// ============================================================================
// SERVE COMMAND - Start A2A Server
// ============================================================================

func executeServeCommand(args *CLIArgs) {
	// Load environment variables
	if err := config.LoadEnvFiles(); err != nil {
		if !os.IsNotExist(err) {
			fatalf("Failed to load environment files: %v", err)
		}
	}

	// Load or create configuration using unified function
	hectorConfig, err := loadConfigFromArgsOrFile(args, true)
	if err != nil {
		fatalf("Failed to load configuration: %v", err)
	}

	// Print debug info if requested
	if args.Debug {
		isZeroConfig := !fileExists(args.ConfigFile) && args.ConfigFile == defaultConfigFile
		if isZeroConfig {
			fmt.Println("🔧 Zero-config mode")
			fmt.Printf("  Model: %s\n", args.Model)
			if args.Tools {
				fmt.Println("  Tools: Enabled (all local tools)")
			}
			if args.MCPURL != "" {
				fmt.Printf("  MCP: %s\n", args.MCPURL)
			}
			if args.DocsFolder != "" {
				fmt.Printf("  Docs: %s\n", args.DocsFolder)
			}
		} else {
			fmt.Printf("📄 Loaded config from: %s\n", args.ConfigFile)
		}
	}

	// Create component manager
	componentManager, err := component.NewComponentManager(hectorConfig)
	if err != nil {
		fatalf("Component initialization failed: %v", err)
	}

	// Initialize document stores if configured
	if err := initializeDocumentStores(hectorConfig, componentManager); err != nil {
		fatalf("Document store initialization failed: %v", err)
	}

	fmt.Println("🚀 Starting Hector A2A Server...")
	if hectorConfig.Name != "" {
		fmt.Printf("📋 Configuration: %s\n", hectorConfig.Name)
	}

	// Get server config
	serverCfg := &hectorConfig.Global.A2AServer
	serverCfg.Enabled = true
	serverCfg.SetDefaults()

	if args.Debug {
		fmt.Printf("🐛 Debug Mode: Enabled\n")
		fmt.Printf("🌐 Host: %s\n", serverCfg.Host)
		fmt.Printf("🔌 Port: %d\n", serverCfg.Port)
		fmt.Printf("📊 Agents to register: %d\n", len(hectorConfig.Agents))
	}

	// Create A2A server
	a2aServerCfg := &a2a.ServerConfig{
		Host:    serverCfg.Host,
		Port:    serverCfg.Port,
		BaseURL: serverCfg.BaseURL,
	}

	server := a2a.NewServer(a2aServerCfg)

	// Initialize authentication if enabled
	if hectorConfig.Global.Auth.Enabled {
		fmt.Println("\n🔒 Initializing authentication...")

		authValidator, err := auth.NewJWTValidator(
			hectorConfig.Global.Auth.JWKSURL,
			hectorConfig.Global.Auth.Issuer,
			hectorConfig.Global.Auth.Audience,
		)
		if err != nil {
			fmt.Printf("❌ Failed to initialize JWT validator: %v\n", err)
			fmt.Println("Please check your auth configuration:")
			fmt.Printf("   - JWKS URL: %s\n", hectorConfig.Global.Auth.JWKSURL)
			fmt.Printf("   - Issuer: %s\n", hectorConfig.Global.Auth.Issuer)
			fmt.Printf("   - Audience: %s\n", hectorConfig.Global.Auth.Audience)
			return
		}

		server.SetAuthValidator(authValidator)
		fmt.Println("  ✅ JWT validator initialized")
		fmt.Printf("     Provider: %s\n", hectorConfig.Global.Auth.Issuer)
		if args.Debug {
			fmt.Printf("     JWKS URL: %s\n", hectorConfig.Global.Auth.JWKSURL)
			fmt.Printf("     Audience: %s\n", hectorConfig.Global.Auth.Audience)
		}
	}

	// Create agent registry for orchestration
	agentRegistry := agent.NewAgentRegistry()

	// Register all configured agents
	fmt.Println("\n📋 Registering agents...")

	// Create A2A client for external agents
	a2aClient := a2a.NewClient(&a2a.ClientConfig{})

	for agentID, agentConfig := range hectorConfig.Agents {
		var agentInstance a2a.Agent
		var err error

		// Create agent based on type
		switch agentConfig.Type {
		case "native", "":
			// Native Hector agent - directly implements a2a.Agent
			agentInstance, err = agent.NewAgent(&agentConfig, componentManager, agentRegistry)
			if err != nil {
				fmt.Printf("❌ Failed to create native agent '%s': %v\n", agentID, err)
				continue
			}
			if args.Debug {
				fmt.Printf("  🔧 Created native agent: %s\n", agentID)
			}

		case "a2a":
			// External A2A agent - discover immediately
			agentInstance, err = agent.NewA2AAgentFromURL(context.Background(), agentConfig.URL, a2aClient)
			if err != nil {
				fmt.Printf("⚠️  Failed to discover external agent '%s' at %s: %v\n", agentID, agentConfig.URL, err)
				fmt.Printf("    Make sure the external A2A server is running and accessible.\n")
				fmt.Printf("    Skipping registration for '%s'.\n", agentID)
				continue
			}
			if args.Debug {
				fmt.Printf("  🌐 Discovered external agent: %s → %s\n", agentID, agentConfig.URL)
			}

		default:
			fmt.Printf("❌ Invalid agent type '%s' for agent '%s'\n", agentConfig.Type, agentID)
			continue
		}

		// Register in A2A server (both native and external agents)
		// Pass visibility to control public exposure
		if err := server.RegisterAgent(agentID, agentInstance, agentConfig.Visibility); err != nil {
			fmt.Printf("❌ Failed to register agent '%s': %v\n", agentID, err)
			continue
		}

		// Register in agent registry for orchestration (both types)
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &agentConfig, nil); err != nil {
			fmt.Printf("⚠️  Failed to register agent '%s' in orchestration registry: %v\n", agentID, err)
		}

		// Show registration confirmation with visibility
		agentTypeLabel := "native"
		if agentConfig.Type == "a2a" {
			agentTypeLabel = "external"
		}
		visibilityLabel := agentConfig.Visibility
		if visibilityLabel == "" {
			visibilityLabel = "public"
		}
		fmt.Printf("  ✅ %s (%s) [%s, %s]\n", agentConfig.Name, agentID, agentTypeLabel, visibilityLabel)

		if args.Debug {
			if agentConfig.Type == "a2a" {
				fmt.Printf("      → Source: %s\n", agentConfig.URL)
			}
			fmt.Printf("      → Endpoint: %s/agents/%s\n", serverCfg.BaseURL, agentID)
		}
	}

	// ✅ Agent registry configured during agent creation
	if args.Debug && len(agentRegistry.List()) > 0 {
		supervisorCount := 0
		for _, entry := range agentRegistry.List() {
			if entry.Config.Reasoning.Engine == "supervisor" {
				supervisorCount++
				subAgentInfo := "all agents"
				if len(entry.Config.SubAgents) > 0 {
					subAgentInfo = fmt.Sprintf("%v", entry.Config.SubAgents)
				}
				fmt.Printf("  🧠 Supervisor '%s' can orchestrate: %s\n", entry.Name, subAgentInfo)
			}
		}
		if supervisorCount > 0 {
			fmt.Printf("\n🔗 Agent orchestration ready: %d supervisor(s) configured\n", supervisorCount)
		}
	}

	// Register agent_call tool for orchestration
	if err := registerOrchestrationTools(componentManager, agentRegistry, args.Debug); err != nil {
		fmt.Printf("⚠️  Warning: Failed to register orchestration tools: %v\n", err)
	}

	fmt.Println("\n🌐 A2A Server ready!")
	fmt.Printf("📡 Agent directory: %s/agents\n", serverCfg.BaseURL)
	fmt.Println("\n💡 Test with Hector CLI:")
	fmt.Printf("   hector list\n")
	fmt.Printf("   hector call <agent-id> \"your prompt\"\n")
	fmt.Println("\nPress Ctrl+C to stop")

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigCh:
		fmt.Println("\n\n🛑 Shutting down gracefully...")
		if err := server.Stop(ctx); err != nil {
			fmt.Printf("Error during shutdown: %v\n", err)
		}
		fmt.Println("✅ Server stopped")
	case err := <-errCh:
		fatalf("Server error: %v", err)
	}
}

// initializeDocumentStores initializes document stores from configuration
func initializeDocumentStores(hectorConfig *config.Config, componentManager *component.ComponentManager) error {
	if len(hectorConfig.DocumentStores) == 0 {
		return nil
	}

	var dbName, embedderName string
	for _, agentConfig := range hectorConfig.Agents {
		if len(agentConfig.DocumentStores) > 0 {
			dbName = agentConfig.Database
			embedderName = agentConfig.Embedder
			break
		}
	}

	if dbName == "" {
		for name := range hectorConfig.Databases {
			dbName = name
			break
		}
	}
	if embedderName == "" {
		for name := range hectorConfig.Embedders {
			embedderName = name
			break
		}
	}

	if dbName == "" || embedderName == "" {
		return fmt.Errorf("document stores require a database and embedder to be configured")
	}

	db, err := componentManager.GetDatabase(dbName)
	if err != nil {
		return fmt.Errorf("failed to get database '%s': %w", dbName, err)
	}

	embedder, err := componentManager.GetEmbedder(embedderName)
	if err != nil {
		return fmt.Errorf("failed to get embedder '%s': %w", embedderName, err)
	}

	searchConfig := config.SearchConfig{}
	searchConfig.SetDefaults()
	searchEngine, err := hectorcontext.NewSearchEngine(db, embedder, searchConfig)
	if err != nil {
		return fmt.Errorf("failed to create search engine: %w", err)
	}

	docStores := make([]config.DocumentStoreConfig, 0, len(hectorConfig.DocumentStores))
	for _, storeConfig := range hectorConfig.DocumentStores {
		docStores = append(docStores, storeConfig)
	}

	err = hectorcontext.InitializeDocumentStoresFromConfig(docStores, searchEngine)
	if err != nil {
		return fmt.Errorf("failed to initialize document stores: %w", err)
	}

	return nil
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
  info <agent>       Get detailed agent information
  call <agent> "..."  Execute a task on an agent
  chat <agent>       Start interactive chat with an agent
  help               Show this help message
  version            Show version information

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

EXECUTION MODES:

  Direct Mode (default - no server needed)
    Commands run in-process with local agent execution.
    Fast, simple, perfect for development and experimentation.

  Server Mode (with --server flag)
    Commands communicate via A2A protocol with a remote server.
    Use for production, multi-agent systems, or shared deployments.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SERVER HOSTING:
  hector serve [options]
    --config FILE            Configuration file (default: hector.yaml)
    --debug                  Enable debug output
    
  Zero-Config Mode (when hector.yaml doesn't exist):
    --api-key KEY            OpenAI API key (or set OPENAI_API_KEY env var) [REQUIRED]
    --base-url URL           OpenAI API base URL (default: https://api.openai.com/v1)
    --model MODEL            OpenAI model (default: gpt-4o-mini)
    --tools                  Enable all local tools (file, command execution)
    --mcp URL                MCP server URL for tool integration
    --docs FOLDER            Document store folder (requires additional config)

DIRECT MODE (In-Process Execution):
  hector call <agent> "prompt" [options]
    --config FILE            Configuration file (default: hector.yaml)
    --api-key KEY            OpenAI API key for zero-config [REQUIRED if no config]
    --base-url URL           OpenAI API base URL (default: https://api.openai.com/v1)
    --model MODEL            OpenAI model for zero-config (default: gpt-4o-mini)
    --tools                  Enable tools for zero-config
    --stream BOOL            Enable streaming (default: true)

  hector chat <agent> [options]
    --config FILE            Configuration file
    --api-key KEY            OpenAI API key for zero-config [REQUIRED if no config]
    --base-url URL           OpenAI API base URL
    --model MODEL            OpenAI model for zero-config
    --tools                  Enable tools for zero-config

  hector list [options]
    --config FILE            Configuration file

  hector info <agent> [options]
    --config FILE            Configuration file

SERVER MODE (A2A Protocol):
  hector call <agent> "prompt" --server URL [options]
    --server URL             A2A server URL (enables server mode)
    --token TOKEN            Authentication token
    --stream BOOL            Enable streaming (default: true)

  hector chat <agent> --server URL [options]
    --server URL             A2A server URL (enables server mode)
    --token TOKEN            Authentication token

  hector list --server URL [options]
    --server URL             A2A server URL (enables server mode)
    --token TOKEN            Authentication token

  hector info <agent> --server URL [options]
    --server URL             A2A server URL (enables server mode)
    --token TOKEN            Authentication token

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

EXAMPLES:

  Direct Mode (Quick & Simple):
  ─────────────────────────────
  # Quick test with zero-config (API key from environment)
  $ export OPENAI_API_KEY=sk-...
  $ hector call assistant "what is 2+2?"

  # With API key as flag
  $ hector call assistant "hello" --api-key sk-...

  # With tools enabled
  $ hector call --api-key sk-... --tools assistant "list files"

  # With custom model
  $ hector call --api-key sk-... --model gpt-4 assistant "complex task"

  # With config file
  $ hector call researcher "analyze data" --config agents.yaml

  # Interactive chat
  $ export OPENAI_API_KEY=sk-...
  $ hector chat assistant

  # List agents from config
  $ hector list --config agents.yaml

  Server Mode (Production):
  ──────────────────────────
  # Start server
  $ hector serve --config agents.yaml

  # Connect to server
  $ hector call assistant "hello" --server localhost:8080

  # Connect to remote server
  $ hector call assistant "hello" --server https://agents.example.com

  # Interactive chat with server
  $ hector chat assistant --server localhost:8080

  # List server agents
  $ hector list --server localhost:8080

  # With authentication
  $ hector call assistant "prompt" --server https://prod.example.com --token abc123

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ENVIRONMENT VARIABLES:
  HECTOR_SERVER    Default A2A server URL
  HECTOR_TOKEN     Default authentication token

For more information: https://github.com/kadirpekel/hector
`)
}

// ============================================================================
// ORCHESTRATION TOOLS REGISTRATION
// ============================================================================

func registerOrchestrationTools(componentManager *component.ComponentManager, agentRegistry *agent.AgentRegistry, debug bool) error {
	toolRegistry := componentManager.GetToolRegistry()

	// Create orchestration tool source
	orchestrationSource := tools.NewLocalToolSource("orchestration")

	// Register agent_call tool
	agentCallTool := agent.NewAgentCallTool(agentRegistry)
	if err := orchestrationSource.RegisterTool(agentCallTool); err != nil {
		return fmt.Errorf("failed to register agent_call tool: %w", err)
	}

	// Register the orchestration source in the tool registry
	if err := toolRegistry.RegisterSource(orchestrationSource); err != nil {
		return fmt.Errorf("failed to register orchestration tool source: %w", err)
	}

	if debug {
		fmt.Println("\n🔧 Orchestration tools registered:")
		fmt.Println("  ✅ agent_call - Enables multi-agent orchestration")
	}

	return nil
}

// ============================================================================
// UTILITIES
// ============================================================================

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
