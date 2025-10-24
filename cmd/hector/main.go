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
		} else {
			// No config file: create zero-config based on command
			cfg = createZeroConfig(command)
		}

		// Apply defaults and validate once
		if cfg != nil {
			cfg.SetDefaults()
			if err := cfg.Validate(); err != nil {
				cli.Fatalf("Invalid configuration: %v", err)
			}
		}
	}

	// Route to appropriate handler
	if err := routeCommand(ctx, cfg); err != nil {
		cli.Fatalf("Command failed: %v", err)
	}
}

// ============================================================================
// ROUTING
// ============================================================================

// routeCommand routes Kong context to appropriate command handler
func routeCommand(ctx *kong.Context, cfg *config.Config) error {
	switch ctx.Command() {
	case "serve", "serve <agent-name>":
		return cli.ServeCommand(cli.CLI.Serve.ToCLIArgs(), cfg)
	case "list":
		return cli.ListCommand(cli.CLI.List.ToCLIArgs(), cfg)
	case "info <agent>":
		return cli.InfoCommand(cli.CLI.Info.ToCLIArgs(), cfg)
	case "call <message> <agent>", "call <message>":
		// Agent is optional in zero-config mode
		return cli.CallCommand(cli.CLI.Call.ToCLIArgs(), cfg)
	case "chat <agent>", "chat":
		// Agent is optional in zero-config mode
		return cli.ChatCommand(cli.CLI.Chat.ToCLIArgs(), cfg)
	case "task get <agent> <task-id>":
		return cli.TaskGetCommand(cli.CLI.Task.Get.ToCLIArgs(), cfg)
	case "task cancel <agent> <task-id>":
		return cli.TaskCancelCommand(cli.CLI.Task.Cancel.ToCLIArgs(), cfg)
	default:
		return nil // Kong handles help automatically
	}
}

// ============================================================================
// ZERO-CONFIG CREATION
// ============================================================================

// createZeroConfig creates config based on command and flags
func createZeroConfig(command string) *config.Config {
	var opts config.ZeroConfigOptions

	switch {
	case command == "serve" || command == "serve <agent-name>":
		opts = cli.CLI.Serve.ToCLIArgs().ToZeroConfigOptions()
	case command == "call <message> <agent>" || command == "call <message>":
		opts = cli.CLI.Call.ToCLIArgs().ToZeroConfigOptions()
	case command == "chat <agent>" || command == "chat":
		opts = cli.CLI.Chat.ToCLIArgs().ToZeroConfigOptions()
	default:
		// Other commands don't support zero-config
		return nil
	}

	return config.CreateZeroConfig(opts)
}
