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
	APIKey     string
	Stream     bool
	Debug      bool
}

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	args := parseArgs()

	// Route to appropriate handler
	switch args.Command {
	case CommandServe:
		executeServeCommand(args)
	case CommandList:
		if err := executeListCommand(args.ServerURL, args.Token); err != nil {
			fatalf("List command failed: %v", err)
		}
	case CommandInfo:
		if err := executeInfoCommand(args.AgentURL, args.Token); err != nil {
			fatalf("Info command failed: %v", err)
		}
	case CommandCall:
		if err := executeCallCommand(args.AgentURL, args.Input, args.Token, args.Stream); err != nil {
			fatalf("Call command failed: %v", err)
		}
	case CommandChat:
		if err := executeChatCommand(args.AgentURL, args.Token); err != nil {
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

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listServer := listCmd.String("server", "", serverFlagDesc)
	listToken := listCmd.String("token", "", tokenFlagDesc)

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	infoServer := infoCmd.String("server", "", serverFlagDesc)
	infoToken := infoCmd.String("token", "", tokenFlagDesc)

	callCmd := flag.NewFlagSet("call", flag.ExitOnError)
	callServer := callCmd.String("server", "", serverFlagDesc)
	callToken := callCmd.String("token", "", tokenFlagDesc)
	callStream := callCmd.Bool("stream", true, "Enable streaming (default: true, use --stream=false to disable)")

	chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)
	chatServer := chatCmd.String("server", "", serverFlagDesc)
	chatToken := chatCmd.String("token", "", tokenFlagDesc)

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

	case "list":
		_ = listCmd.Parse(os.Args[2:])
		args.Command = CommandList
		args.ServerURL = resolveServerURL(*listServer)
		args.Token = *listToken

	case "info":
		_ = infoCmd.Parse(os.Args[2:])
		if len(infoCmd.Args()) < 1 {
			fatalf("Usage: hector info <agent> [--server URL]")
		}
		args.Command = CommandInfo
		args.AgentURL = resolveAgentURL(infoCmd.Args()[0], *infoServer)
		args.Token = *infoToken

	case "call":
		_ = callCmd.Parse(os.Args[2:])
		if len(callCmd.Args()) < 2 {
			fatalf("Usage: hector call <agent> \"prompt\" [--server URL]")
		}
		args.Command = CommandCall
		args.AgentURL = resolveAgentURL(callCmd.Args()[0], *callServer)
		args.Input = callCmd.Args()[1]
		args.Token = *callToken
		args.Stream = *callStream

	case "chat":
		_ = chatCmd.Parse(os.Args[2:])
		if len(chatCmd.Args()) < 1 {
			fatalf("Usage: hector chat <agent> [--server URL]")
		}
		args.Command = CommandChat
		args.AgentURL = resolveAgentURL(chatCmd.Args()[0], *chatServer)
		args.Token = *chatToken

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

	// Load configuration
	hectorConfig, err := config.LoadConfig(args.ConfigFile)
	if err != nil {
		fatalf("Failed to load config: %v", err)
	}

	// Set defaults
	hectorConfig.SetDefaults()

	// Validate configuration
	if err := hectorConfig.Validate(); err != nil {
		fatalf("Invalid configuration: %v", err)
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
			agentInstance, err = agent.NewAgent(&agentConfig, componentManager)
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
  list               List available agents from A2A server
  info <agent>       Get detailed agent information
  call <agent> "..."  Execute a task on an agent
  chat <agent>       Start interactive chat with an agent
  help               Show this help message
  version            Show version information

SERVER MODE:
  hector serve [options]
    --config FILE    Configuration file (default: hector.yaml)
    --debug          Enable debug output

CLIENT MODE:
  hector list [options]
    --server URL     A2A server URL (default: localhost:8080)
    --token TOKEN    Authentication token

  hector info <agent> [options]
    --server URL     A2A server URL (default: localhost:8080)
    --token TOKEN    Authentication token

  hector call <agent> "prompt" [options]
    --server URL     A2A server URL (default: localhost:8080)
    --token TOKEN    Authentication token
    --stream BOOL    Enable streaming (default: true, use --stream=false to disable)

  hector chat <agent> [options]
    --server URL     A2A server URL (default: localhost:8080)
    --token TOKEN    Authentication token

AGENT SHORTCUTS:
  You can specify agents in two ways:

  1. Agent ID (shorthand):
     $ hector call my_agent "prompt"
     Constructs: http://localhost:8080/agents/my_agent

  2. Full URL:
     $ hector call http://example.com:8080/agents/my_agent "prompt"
     Uses the URL as-is

  Use --server to change the default server for shorthand notation:
     $ hector call --server http://localhost:8081 my_agent "prompt"
     Constructs: http://localhost:8081/agents/my_agent

EXAMPLES:
  # Start server
  $ hector serve --config hector.yaml

  # List agents from local server (default)
  $ hector list

  # List agents from remote server
  $ hector list --server https://agents.example.com

  # Get agent info (shorthand)
  $ hector info my_agent

  # Get agent info (full URL)
  $ hector info http://localhost:8080/agents/my_agent

  # Call agent on default server (localhost:8080)
  $ hector call my_agent "Analyze competitors"

  # Call agent on custom server
  $ hector call --server http://localhost:8081 my_agent "Analyze competitors"

  # Call agent with full URL (ignores --server flag)
  $ hector call http://example.com/agents/my_agent "Analyze competitors"

  # Interactive chat with shorthand
  $ hector chat my_agent

  # Interactive chat on custom server
  $ hector chat --server https://agents.example.com my_agent

  # With authentication
  $ hector call my_agent "prompt" --token "your-bearer-token"

  # Disable streaming
  $ hector call my_agent "prompt" --stream=false

ENVIRONMENT VARIABLES:
  HECTOR_SERVER    Default A2A server URL (overrides localhost:8080)
  HECTOR_TOKEN     Default authentication token

For more information: https://github.com/kadirpekel/hector
`)
}

// ============================================================================
// ORCHESTRATION TOOLS REGISTRATION
// ============================================================================

// registerOrchestrationTools registers agent_call and other orchestration tools
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
