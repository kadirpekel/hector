package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/config"
)

// ParseArgs parses command line arguments and returns a CLIArgs structure
func ParseArgs(version string) *CLIArgs {
	args := &CLIArgs{}

	// Define subcommands
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveConfig := serveCmd.String("config", "", "Configuration file (required)")
	servePort := serveCmd.Int("port", 8080, "gRPC server port (matches A2A server default)")
	serveDebug := serveCmd.Bool("debug", false, "Enable debug mode")

	// A2A Server override flags
	serveHost := serveCmd.String("host", "", "Server host (overrides config)")
	serveA2ABaseURL := serveCmd.String("a2a-base-url", "", "A2A base URL for discovery (overrides config)")

	// Zero-config mode flags (consolidated)
	serveZeroConfig := addZeroConfigFlags(serveCmd)
	serveEmbedder := serveCmd.String("embedder-model", "nomic-embed-text", "Embedder model for document store")
	serveVectorDB := serveCmd.String("vectordb", "http://localhost:6334", "Vector database connection string")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listServer := listCmd.String("server", "", "A2A server URL (enables server mode)")
	listToken := listCmd.String("token", "", "Authentication token")
	listConfig := listCmd.String("config", "", "Configuration file (local mode)")

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	infoServer := infoCmd.String("server", "", "A2A server URL (enables server mode)")
	infoToken := infoCmd.String("token", "", "Authentication token")
	infoConfig := infoCmd.String("config", "", "Configuration file (local mode)")

	callCmd := flag.NewFlagSet("call", flag.ExitOnError)
	callServer := callCmd.String("server", "", "A2A server URL (enables server mode)")
	callToken := callCmd.String("token", "", "Authentication token")
	callStream := callCmd.Bool("stream", true, "Enable streaming (default: true)")
	callSession := callCmd.String("session", "", "Session ID for resuming conversations")
	callConfig := callCmd.String("config", "", "Configuration file (local mode)")
	callZeroConfig := addZeroConfigFlags(callCmd)

	chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)
	chatServer := chatCmd.String("server", "", "A2A server URL (enables server mode)")
	chatToken := chatCmd.String("token", "", "Authentication token")
	chatSession := chatCmd.String("session", "", "Session ID for resuming conversations")
	chatConfig := chatCmd.String("config", "", "Configuration file (local mode)")
	chatZeroConfig := addZeroConfigFlags(chatCmd)
	chatNoStream := chatCmd.Bool("no-stream", false, "Disable streaming (default: streaming enabled)")

	taskCmd := flag.NewFlagSet("task", flag.ExitOnError)
	taskServer := taskCmd.String("server", "", "A2A server URL (enables server mode)")
	taskToken := taskCmd.String("token", "", "Authentication token")
	taskSession := taskCmd.String("session", "", "Session ID for task context")
	taskConfig := taskCmd.String("config", "", "Configuration file (local mode)")

	// Parse command
	if len(os.Args) < 2 {
		ShowHelp()
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
		serveZeroConfig.populateArgs(args)
		args.EmbedderModel = *serveEmbedder
		args.VectorDB = *serveVectorDB

		// Detect flags in wrong position (after positional args)
		CheckForMisplacedFlags(serveCmd.Args(), "serve")

		// Optional positional argument: agent name for zero-config mode
		// If not provided, CreateZeroConfig uses config.DefaultAgentName
		if len(serveCmd.Args()) > 0 {
			args.AgentID = serveCmd.Args()[0]
		}

		if len(serveCmd.Args()) > 1 {
			Fatalf("Usage: hector serve [AGENT] [OPTIONS]\n\nError: Too many positional arguments.\n\nExamples:\n  hector serve                    # Creates agent named '%s' (default)\n  hector serve myagent            # Creates agent named 'myagent'\n  hector serve --tools gopher     # Creates agent named 'gopher' with tools\n  hector serve --config file.yaml # Uses agents from config file", config.DefaultAgentName)
		}

	case "list":
		_ = listCmd.Parse(os.Args[2:])
		args.Command = CommandList
		args.ServerURL = *listServer // Don't resolve yet - let command detect mode
		args.Token = *listToken
		args.ConfigFile = *listConfig

	case "info":
		_ = infoCmd.Parse(os.Args[2:])
		if len(infoCmd.Args()) < 1 {
			Fatalf("Usage: hector info <agent> [OPTIONS]")
		}
		args.Command = CommandInfo
		args.AgentID = infoCmd.Args()[0]
		args.ServerURL = *infoServer // Don't resolve yet
		args.Token = *infoToken
		args.ConfigFile = *infoConfig

	case "call":
		_ = callCmd.Parse(os.Args[2:])
		args.Command = CommandCall
		args.ServerURL = *callServer // Don't resolve yet
		args.Token = *callToken
		args.Stream = *callStream
		args.SessionID = *callSession
		args.ConfigFile = *callConfig
		callZeroConfig.populateArgs(args)

		// Handle agent name and input based on mode
		mode := DetectMode(args)
		switch mode {
		case ModeLocalZeroConfig:
			// Zero-config mode: no agent name, use default
			if len(callCmd.Args()) < 1 {
				Fatalf("Usage: hector call [OPTIONS] \"prompt\"")
			}
			if len(callCmd.Args()) > 1 {
				Fatalf("Usage: hector call [OPTIONS] \"prompt\"\n\nError: Agent name not supported in zero-config mode.\n\nFor zero-config mode:\n  hector call \"your prompt\"\n\nTo use named agents, create a config file:\n  hector call --config myconfig.yaml <agent-name> \"your prompt\"")
			}
			args.AgentID = config.DefaultAgentName
			args.Input = callCmd.Args()[0]

		case ModeLocalConfig:
			// Local config mode: agent name is REQUIRED
			if len(callCmd.Args()) < 2 {
				Fatalf("Usage: hector call --config FILE <agent> \"prompt\"\n\nError: Agent name required when using configuration file.\n\nAvailable agents are defined in your config file.\nExample: hector call --config %s <agent-name> \"your prompt\"", args.ConfigFile)
			}
			args.AgentID = callCmd.Args()[0]
			args.Input = callCmd.Args()[1]

		case ModeClient:
			// Client mode: agent name is required
			if len(callCmd.Args()) < 2 {
				Fatalf("Usage: hector call --server URL <agent> \"prompt\"")
			}
			args.AgentID = callCmd.Args()[0]
			args.Input = callCmd.Args()[1]

		default:
			Fatalf("Error: Invalid mode for 'call' command: %s", mode)
		}

		// Detect flags in wrong position (after positional args)
		expectedArgs := 2
		if mode == ModeLocalZeroConfig {
			expectedArgs = 1
		}
		if len(callCmd.Args()) > expectedArgs {
			CheckForMisplacedFlags(callCmd.Args()[expectedArgs:], "call")
		}

	case "chat":
		_ = chatCmd.Parse(os.Args[2:])
		args.Command = CommandChat
		args.ServerURL = *chatServer // Don't resolve yet
		args.Token = *chatToken
		args.SessionID = *chatSession
		args.ConfigFile = *chatConfig
		chatZeroConfig.populateArgs(args)
		args.Stream = !*chatNoStream // Streaming is default, --no-stream disables it

		// Handle agent name based on mode
		mode := DetectMode(args)
		switch mode {
		case ModeLocalZeroConfig:
			// Zero-config mode: no agent name, use default
			if len(chatCmd.Args()) > 0 {
				Fatalf("Usage: hector chat [OPTIONS]\n\nError: Agent name not supported in zero-config mode.\n\nFor zero-config mode, omit agent name:\n  hector chat\n\nTo use named agents, create a config file:\n  hector chat --config myconfig.yaml <agent-name>")
			}
			args.AgentID = config.DefaultAgentName

		case ModeLocalConfig:
			// Local config mode: agent name is REQUIRED
			if len(chatCmd.Args()) < 1 {
				Fatalf("Usage: hector chat --config FILE <agent>\n\nError: Agent name required when using configuration file.\n\nAvailable agents are defined in your config file.\nExample: hector chat --config %s <agent-name>", args.ConfigFile)
			}
			args.AgentID = chatCmd.Args()[0]

		case ModeClient:
			// Client mode: agent name is required
			if len(chatCmd.Args()) < 1 {
				Fatalf("Usage: hector chat --server URL <agent>")
			}
			args.AgentID = chatCmd.Args()[0]

		default:
			Fatalf("Error: Invalid mode for 'chat' command: %s", mode)
		}

		// Detect flags in wrong position (after positional args)
		expectedArgs := 1
		if mode == ModeLocalZeroConfig {
			expectedArgs = 0
		}
		if len(chatCmd.Args()) > expectedArgs {
			CheckForMisplacedFlags(chatCmd.Args()[expectedArgs:], "chat")
		}

	case "task":
		_ = taskCmd.Parse(os.Args[2:])
		if len(taskCmd.Args()) < 3 {
			Fatalf("Usage: hector task <action> <agent> <task-id> [OPTIONS]\n" +
				"Actions: get, cancel")
		}
		args.Command = CommandTask
		args.TaskAction = taskCmd.Args()[0]
		args.AgentID = taskCmd.Args()[1]
		args.TaskID = taskCmd.Args()[2]
		args.ServerURL = *taskServer
		args.Token = *taskToken
		args.SessionID = *taskSession
		args.ConfigFile = *taskConfig

	case "help", "--help", "-h":
		args.Command = CommandHelp

	case "version", "--version", "-v":
		fmt.Printf("Hector %s\n", version)
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		ShowHelp()
		os.Exit(1)
	}

	// Track which zero-config flags were explicitly provided by user
	explicitAPIKey := args.APIKey != ""
	explicitModel := args.Model != ""
	explicitBaseURL := args.BaseURL != ""
	explicitTools := args.Tools

	// Validate mode and flags after parsing
	// Pass explicit flag info for better validation
	args.ExplicitZeroConfigFlags = explicitAPIKey || explicitModel || explicitBaseURL || explicitTools
	ValidateModeAndFlags(args)

	return args
}

// DetectMode determines which CLI mode is active
func DetectMode(args *CLIArgs) CLIMode {
	// Server mode (serve command)
	if args.Command == CommandServe {
		if args.ConfigFile != "" {
			return ModeServerConfig
		}
		return ModeServerZeroConfig
	}

	// Client mode (--server flag)
	if args.ServerURL != "" {
		return ModeClient
	}

	// Local mode - distinguish between zero-config and config
	if args.ConfigFile != "" {
		return ModeLocalConfig
	}
	return ModeLocalZeroConfig
}

// ValidateModeAndFlags checks for invalid flag combinations and fails fast
func ValidateModeAndFlags(args *CLIArgs) {
	mode := DetectMode(args)

	// Validate based on mode
	switch mode {
	case ModeServerZeroConfig, ModeServerConfig:
		// Server modes: all flags are valid
		return

	case ModeClient:
		// Client mode: ONLY --server, --token, --stream allowed
		// Configuration flags are NOT supported
		if args.ConfigFile != "" {
			Fatalf(`❌ Error: --config flag is not supported in %s mode

You're connecting to a remote server which has its own configuration.

Solutions:
  • Remove --config flag to use the remote server's configuration
  • Remove --server flag (or unset HECTOR_SERVER) to use Local mode with local config

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		// Zero-config flags not allowed in client mode
		if args.APIKey != "" {
			Fatalf(`❌ Error: --api-key flag is not supported in %s mode

The remote server has its own LLM configuration.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.Model != "" {
			Fatalf(`❌ Error: --model flag is not supported in %s mode

The remote server has its own model configuration.

Solutions:
  • Remove --model flag to use the remote server's models
  • Use Local mode (remove --server) for local model selection

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.Tools {
			Fatalf(`❌ Error: --tools flag is not supported in %s mode

The remote server controls which tools are enabled.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.MCPURL != "" {
			Fatalf(`❌ Error: --mcp-url flag is not supported in %s mode

The remote server controls which MCP servers are configured.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

		if args.BaseURL != "" {
			Fatalf(`❌ Error: --base-url flag is not supported in %s mode

The remote server has its own API configuration.

Current mode: %s
Server: %s`, mode, mode, args.ServerURL)
		}

	case ModeLocalZeroConfig:
		// Local zero-config mode: API key validation happens in config.CreateZeroConfig()
		// which checks both flags and environment variables

	case ModeLocalConfig:
		// Local config mode: Validate config file exists
		if _, err := os.Stat(args.ConfigFile); err != nil {
			Fatalf("Error: Configuration file not found: %s", args.ConfigFile)
		}

		// Warn if zero-config flags were explicitly provided
		if args.ExplicitZeroConfigFlags {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Zero-config flags provided with --config\n")
			fmt.Fprintf(os.Stderr, "   Zero-config flags will be ignored.\n")
			fmt.Fprintf(os.Stderr, "   Using configuration from: %s\n\n", args.ConfigFile)
		}
	}
}
