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

	// Route to appropriate handler using CLI package
	switch args.Command {
	case cli.CommandServe:
		// Serve command is in serve.go
		executeServeCommand(args)
	case cli.CommandList:
		if err := cli.ListCommand(args); err != nil {
			cli.Fatalf("List command failed: %v", err)
		}
	case cli.CommandInfo:
		if err := cli.InfoCommand(args); err != nil {
			cli.Fatalf("Info command failed: %v", err)
		}
	case cli.CommandCall:
		if err := cli.CallCommand(args); err != nil {
			cli.Fatalf("Call command failed: %v", err)
		}
	case cli.CommandChat:
		if err := cli.ChatCommand(args); err != nil {
			cli.Fatalf("Chat command failed: %v", err)
		}
	case cli.CommandTask:
		// Task subcommands
		switch args.TaskAction {
		case "get":
			if err := cli.TaskGetCommand(args); err != nil {
				cli.Fatalf("Task get command failed: %v", err)
			}
		case "cancel":
			if err := cli.TaskCancelCommand(args); err != nil {
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

// ============================================================================
// SERVE COMMAND
// ============================================================================
// Note: executeServeCommand has been moved to cmd/hector/serve.go
