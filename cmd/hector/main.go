package main

import (
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/kadirpekel/hector/pkg/cli"
	"github.com/kadirpekel/hector/pkg/config"
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
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Load environment variables
	if err := config.LoadEnvFiles(); err != nil && !os.IsNotExist(err) {
		cli.Fatalf("Failed to load environment files: %v", err)
	}

	// Parse command line with Kong
	ctx := kong.Parse(&cli.CLI,
		kong.Name("hector"),
		kong.Description("AI agent framework with A2A protocol support"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.Vars{
			"version": getVersion(),
		},
	)

	// Load config early (once) - orthogonal to mode detection
	// Client mode doesn't need config, all other modes do
	var cfg *config.Config

	// Detect mode based on command and flags
	command := ctx.Command()
	isClientMode := false

	// Check if user specified --server flag (client mode)
	switch {
	case cli.CLI.List.Server != "":
		isClientMode = true
	case cli.CLI.Info.Server != "":
		isClientMode = true
	case cli.CLI.Call.Server != "":
		isClientMode = true
	case cli.CLI.Chat.Server != "":
		isClientMode = true
	case cli.CLI.Task.Server != "":
		isClientMode = true
	}

	// Load config for non-client modes
	if !isClientMode {
		if cli.CLI.Config != "" {
			// Config file specified: load it
			loadedCfg, err := config.LoadConfig(cli.CLI.Config)
			if err != nil {
				cli.Fatalf("Failed to load config: %v", err)
			}
			cfg = loadedCfg
		}
		// No config file: zero-config mode (cfg remains nil)

		// Apply defaults and validate once
		if cfg != nil {
			cfg.SetDefaults()
			if err := cfg.Validate(); err != nil {
				cli.Fatalf("Invalid configuration: %v", err)
			}
		}
	}

	// Determine explicit mode - much better than cfg == nil
	mode := determineMode(command, isClientMode, cfg != nil)

	// Route to appropriate handler with explicit mode
	if err := routeCommand(ctx, cfg, mode); err != nil {
		cli.Fatalf("Command failed: %v", err)
	}
}

// ============================================================================
// ROUTING
// ============================================================================

// determineMode explicitly determines the CLI mode - much better than cfg == nil
func determineMode(command string, isClientMode bool, hasConfig bool) cli.CLIMode {
	if isClientMode {
		return cli.ModeClient
	}

	switch command {
	case "serve", "serve <agent-name>":
		if hasConfig {
			return cli.ModeServerConfig
		}
		return cli.ModeServerZeroConfig
	default:
		if hasConfig {
			return cli.ModeLocalConfig
		}
		return cli.ModeLocalZeroConfig
	}
}

// routeCommand routes Kong context to appropriate command handler
func routeCommand(ctx *kong.Context, cfg *config.Config, mode cli.CLIMode) error {
	// Use Kong structs directly - no conversion layer!
	switch ctx.Command() {
	case "version":
		return cli.VersionCommand(&cli.CLI.Version, cfg, mode)
	case "serve", "serve <agent-name>":
		return cli.ServeCommand(&cli.CLI.Serve, cfg, mode)
	case "list":
		return cli.ListCommand(&cli.CLI.List, cfg, mode)
	case "info <agent>":
		return cli.InfoCommand(&cli.CLI.Info, cfg, mode)
	case "call <message> <agent>", "call <message>":
		// Agent is optional in zero-config mode
		return cli.CallCommand(&cli.CLI.Call, cfg, mode)
	case "chat <agent>", "chat":
		// Agent is optional in zero-config mode
		return cli.ChatCommand(&cli.CLI.Chat, cfg, mode)
	case "task get <agent> <task-id>":
		return cli.TaskGetCommand(&cli.CLI.Task.Get, cfg, mode)
	case "task cancel <agent> <task-id>":
		return cli.TaskCancelCommand(&cli.CLI.Task.Cancel, cfg, mode)
	default:
		return nil // Kong handles help automatically
	}
}

// ============================================================================
// ZERO-CONFIG CREATION
// ============================================================================

// createZeroConfig creates config based on command and flags
func createZeroConfig(command string) *config.Config {
	switch {
	case command == "serve" || command == "serve <agent-name>":
		// Use the agent name from serve command
		cfg := config.CreateZeroConfigFromCLI(cli.CLI.Serve)
		if cfg != nil && cli.CLI.Serve.AgentName != "" {
			// Override the agent name if specified in serve command
			if agent, exists := cfg.Agents[config.DefaultAgentName]; exists {
				agent.Name = cli.CLI.Serve.AgentName
				cfg.Agents[cli.CLI.Serve.AgentName] = agent
				delete(cfg.Agents, config.DefaultAgentName)
			}
		}
		return cfg
	case command == "call <message>":
		return config.CreateZeroConfigFromCLI(cli.CLI.Call)
	case command == "chat":
		return config.CreateZeroConfigFromCLI(cli.CLI.Chat)
	default:
		// Other commands don't support zero-config
		return nil
	}
}
