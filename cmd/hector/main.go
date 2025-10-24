package main

import (
	"os"
	"runtime/debug"

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

// Note: CLI types, parsing, and validation have been moved to pkg/cli/

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Load environment variables
	if err := config.LoadEnvFiles(); err != nil && !os.IsNotExist(err) {
		cli.Fatalf("Failed to load environment files: %v", err)
	}

	args := cli.ParseArgs(getVersion())

	// Load config early (once) - orthogonal to mode detection
	// Client mode doesn't need config, all other modes do
	var cfg *config.Config
	mode := cli.DetectMode(args)

	if mode != cli.ModeClient {
		// Load config based on file existence, not mode
		if args.ConfigFile != "" {
			// Config file specified: load it
			loadedCfg, err := config.LoadConfig(args.ConfigFile)
			if err != nil {
				cli.Fatalf("Failed to load config: %v", err)
			}
			cfg = loadedCfg
		} else {
			// No config file: create zero-config
			cfg = config.CreateZeroConfig(args.ToZeroConfigOptions())
		}

		// Apply defaults and validate once
		cfg.SetDefaults()
		if err := cfg.Validate(); err != nil {
			cli.Fatalf("Invalid configuration: %v", err)
		}
	}

	// Route to appropriate handler using CLI package
	// All commands receive cfg (nil for client mode)
	switch args.Command {
	case cli.CommandServe:
		if err := cli.ServeCommand(args, cfg); err != nil {
			cli.Fatalf("Serve command failed: %v", err)
		}
	case cli.CommandList:
		if err := cli.ListCommand(args, cfg); err != nil {
			cli.Fatalf("List command failed: %v", err)
		}
	case cli.CommandInfo:
		if err := cli.InfoCommand(args, cfg); err != nil {
			cli.Fatalf("Info command failed: %v", err)
		}
	case cli.CommandCall:
		if err := cli.CallCommand(args, cfg); err != nil {
			cli.Fatalf("Call command failed: %v", err)
		}
	case cli.CommandChat:
		if err := cli.ChatCommand(args, cfg); err != nil {
			cli.Fatalf("Chat command failed: %v", err)
		}
	case cli.CommandTask:
		// Task subcommands
		switch args.TaskAction {
		case "get":
			if err := cli.TaskGetCommand(args, cfg); err != nil {
				cli.Fatalf("Task get command failed: %v", err)
			}
		case "cancel":
			if err := cli.TaskCancelCommand(args, cfg); err != nil {
				cli.Fatalf("Task cancel command failed: %v", err)
			}
		default:
			cli.Fatalf("Unknown task action: %s (use 'get' or 'cancel')", args.TaskAction)
		}
	case cli.CommandHelp:
		cli.ShowHelp()
	default:
		cli.ShowHelp()
	}
}
