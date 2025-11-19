package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/kadirpekel/hector/pkg/cli"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/logger"
)

const (
	// DefaultLogFile is the default log file name for client/local modes
	// Can be overridden via LOG_FILE environment variable
	DefaultLogFile = "hector.log"
	// LogFileEnvVar is the environment variable name for log file path
	LogFileEnvVar = "LOG_FILE"
	// DefaultLogFormat is the default log format
	DefaultLogFormat = "simple"
)

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}
	return "dev"
}

// printBanner prints a colored ASCII banner using hector-green (#10b981)
func printBanner() {
	// Check if stdout is a terminal
	if fileInfo, err := os.Stdout.Stat(); err == nil {
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			// Not a terminal, skip banner
			return
		}
	} else {
		return
	}

	// Green color: #10b981 = RGB(16, 185, 129)
	// Use ANSI 256-color mode: \033[38;5;Xm where X is the color code
	// For RGB(16, 185, 129), approximate with bright green: \033[38;5;42m or use RGB: \033[38;2;16;185;129m
	greenColor := "\033[38;2;16;185;129m"
	resetColor := "\033[0m"

	banner := `
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
`
	fmt.Printf("%s%s%s\n", greenColor, banner, resetColor)
}

// isClientOrLocalCommand checks if command is client/local mode by inspecting args
// This is needed before Kong parsing to determine if we should skip banner/log to file
// Returns true for: call, chat, info, task commands (all client/local modes)
func isClientOrLocalCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check for client/local commands (call, chat, info, task)
	for _, arg := range args {
		// Skip program name and flags, look for commands
		if arg == "call" || arg == "chat" || arg == "info" {
			return true
		}
		if strings.HasPrefix(arg, "task") {
			return true
		}
	}
	return false
}

// getDefaultLogFileName returns the default log file name
// Checks LOG_FILE environment variable, otherwise returns the default constant
func getDefaultLogFileName() string {
	if envLogFile := os.Getenv(LogFileEnvVar); envLogFile != "" {
		return envLogFile
	}
	return DefaultLogFile
}

// determineLogLevel determines the log level based on priority: CLI flag > config file > default
func determineLogLevel(cliLogLevel string, cfg *config.Config) (slog.Level, error) {
	if cliLogLevel != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[cliLogLevel] {
			return 0, fmt.Errorf("invalid log level '%s' (must be: debug, info, warn, error)", cliLogLevel)
		}
		return logger.ParseLevel(cliLogLevel)
	}
	if cfg != nil && cfg.Global.Observability.LogLevel != "" {
		return logger.ParseLevel(cfg.Global.Observability.LogLevel)
	}
	return slog.LevelInfo, nil
}

// determineLogFile determines the log file based on priority: CLI flag > config file > auto-enable for client/local > stderr
// Returns the file, cleanup function, and error
func determineLogFile(cliLogFile string, cfg *config.Config, mode cli.CLIMode) (*os.File, func(), error) {
	isClientOrLocalMode := mode == cli.ModeClient || mode == cli.ModeLocalConfig || mode == cli.ModeLocalZeroConfig

	if cliLogFile != "" {
		file, cleanup, err := logger.OpenLogFile(cliLogFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		return file, cleanup, nil
	}
	if cfg != nil && cfg.Global.Observability.LogFile != "" {
		file, cleanup, err := logger.OpenLogFile(cfg.Global.Observability.LogFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file from config: %w", err)
		}
		return file, cleanup, nil
	}
	if isClientOrLocalMode {
		// Auto-enable file logging for client/local modes to keep stdout clean
		file, cleanup, err := logger.OpenLogFile(getDefaultLogFileName())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create log file: %w", err)
		}
		return file, cleanup, nil
	}
	return os.Stderr, nil, nil
}

// determineLogFormat determines the log format based on priority: CLI flag > config file > default
func determineLogFormat(cliLogFormat string, cfg *config.Config) string {
	if cliLogFormat != "" {
		return cliLogFormat
	}
	if cfg != nil && cfg.Global.Observability.LogFormat != "" {
		return cfg.Global.Observability.LogFormat
	}
	return DefaultLogFormat
}

func main() {
	// Detect client/local mode early to skip banner and route logs to file
	isClientOrLocal := isClientOrLocalCommand(os.Args)

	// Skip banner for client/local mode commands (clean output)
	if !isClientOrLocal {
		printBanner()
	}

	if err := config.LoadEnvFiles(); err != nil && !os.IsNotExist(err) {
		cli.Fatalf("Failed to load environment files: %v", err)
	}

	// Validate mutual exclusivity EARLY - before Kong processes arguments
	// This checks raw command-line args, so we don't need to worry about defaults
	if !cli.ShouldSkipValidation(os.Args) {
		if err := cli.ValidateConfigMutualExclusivity(os.Args); err != nil {
			cli.Fatalf("%v", err)
		}
	}

	ctx := kong.Parse(&cli.CLI,
		kong.Name("hector"),
		kong.Description("Declarative A2A-Native AI agent framework"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.Vars{
			"version": getVersion(),
		},
	)

	var cfg *config.Config
	var configLoader *config.Loader

	command := ctx.Command()

	isClientMode := isClientModeCommand(command)

	hasConfigFile := cli.CLI.Config != ""

	// Load and process configuration
	// Note: Config processing logs (e.g., "Auto-created tools") will use slog's default logger (stderr)
	// until we initialize our custom logger below. This is fine - stderr is separate from stdout.
	if !isClientMode && command != "validate <config>" {
		if hasConfigFile {
			configType, err := config.ParseConfigType(cli.CLI.ConfigType)
			if err != nil {
				cli.Fatalf("Invalid config type: %v", err)
			}

			var endpoints []string
			if cli.CLI.ConfigEndpoints != "" {
				endpoints = strings.Split(cli.CLI.ConfigEndpoints, ",")
				for i := range endpoints {
					endpoints[i] = strings.TrimSpace(endpoints[i])
				}
			}

			loaderOpts := config.LoaderOptions{
				Type:      configType,
				Path:      cli.CLI.Config,
				Endpoints: endpoints,
				Watch:     cli.CLI.ConfigWatch,
			}

			cfg, configLoader, err = config.LoadConfigWithLoader(loaderOpts)
			if err != nil {
				cli.Fatalf("%s", cli.FormatConfigError(cli.CLI.Config, err))
			}
		} else {
			cfg = createZeroConfig(command)
		}

		if cfg != nil {
			var err error
			cfg, err = config.ProcessConfigPipeline(cfg)
			if err != nil {
				cli.Fatalf("%s", cli.FormatConfigError(cli.CLI.Config, err))
			}
		}
	}

	if command == "validate <config>" {
		err := cli.ValidateCommand(&cli.CLI.Validate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	mode := determineMode(command, isClientMode, hasConfigFile)

	// Initialize logger with final settings
	logLevel, err := determineLogLevel(cli.CLI.LogLevel, cfg)
	if err != nil {
		cli.Fatalf("Invalid log level: %v", err)
	}

	logFile, cleanup, err := determineLogFile(cli.CLI.LogFile, cfg, mode)
	if err != nil {
		cli.Fatalf("Failed to open log file: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	logFormat := determineLogFormat(cli.CLI.LogFormat, cfg)

	logger.Init(logLevel, logFile, logFormat)

	if err := routeCommand(ctx, cfg, configLoader, mode); err != nil {
		cli.Fatalf("Command failed: %v", err)
	}
}

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

func routeCommand(ctx *kong.Context, cfg *config.Config, configLoader *config.Loader, mode cli.CLIMode) error {

	switch ctx.Command() {
	case "version":
		return cli.VersionCommand(&cli.CLI.Version, cfg, mode)
	case "serve", "serve <agent-name>":
		return cli.ServeCommand(&cli.CLI.Serve, cfg, configLoader, mode)
	case "info <agent>":
		return cli.InfoCommand(&cli.CLI.Info, cfg, mode)
	case "call <message>":
		return cli.CallCommand(&cli.CLI.Call, cfg, mode)
	case "chat":
		return cli.ChatCommand(&cli.CLI.Chat, cfg, mode)
	case "task get <agent> <task-id>":
		return cli.TaskGetCommand(&cli.CLI.Task.Get, cfg, mode)
	case "task cancel <agent> <task-id>":
		return cli.TaskCancelCommand(&cli.CLI.Task.Cancel, cfg, mode)
	default:
		return nil
	}
}

func createZeroConfig(command string) *config.Config {

	switch {
	case command == "serve" || command == "serve <agent-name>":

		cfg := config.CreateZeroConfig(cli.CLI.Serve)
		if cfg != nil && cli.CLI.Serve.AgentName != "" {

			if agent, exists := cfg.Agents[config.DefaultAgentName]; exists {
				agent.Name = cli.CLI.Serve.AgentName
				cfg.Agents[cli.CLI.Serve.AgentName] = agent
				delete(cfg.Agents, config.DefaultAgentName)
			}
		}
		return cfg
	case command == "call <message>":
		return config.CreateZeroConfig(cli.CLI.Call)
	case command == "chat":
		return config.CreateZeroConfig(cli.CLI.Chat)
	default:
		return nil
	}
}

func isClientModeCommand(command string) bool {
	switch command {
	case "info <agent>":
		return cli.CLI.Info.URL != ""
	case "call <message>":
		return cli.CLI.Call.URL != ""
	case "chat":
		return cli.CLI.Chat.URL != ""
	case "task get <agent> <task-id>", "task cancel <agent> <task-id>":
		return cli.CLI.Task.URL != ""
	default:
		return false
	}
}
