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

func main() {
	// Print colored ASCII banner
	printBanner()

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

	// Initialize logger EARLY with defaults so config processing logs are formatted correctly
	// We'll re-initialize with proper settings after config is loaded
	earlyLogLevel := slog.LevelInfo
	if cli.CLI.LogLevel != "" {
		if level, err := logger.ParseLevel(cli.CLI.LogLevel); err == nil {
			earlyLogLevel = level
		}
	} else if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		if level, err := logger.ParseLevel(envLogLevel); err == nil {
			earlyLogLevel = level
		}
	}

	earlyLogFile := os.Stderr
	if cli.CLI.LogFile != "" {
		if file, _, err := logger.OpenLogFile(cli.CLI.LogFile); err == nil {
			earlyLogFile = file
		}
	} else if envLogFile := os.Getenv("LOG_FILE"); envLogFile != "" {
		if file, _, err := logger.OpenLogFile(envLogFile); err == nil {
			earlyLogFile = file
		}
	}

	earlyLogFormat := "simple"
	if cli.CLI.LogFormat != "" {
		earlyLogFormat = cli.CLI.LogFormat
	} else if envLogFormat := os.Getenv("LOG_FORMAT"); envLogFormat != "" {
		earlyLogFormat = envLogFormat
	}

	logger.Init(earlyLogLevel, earlyLogFile, earlyLogFormat)

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

	// Re-configure logging with final settings (config file may override early defaults)
	// Priority: CLI flag > config file > mode-based defaults
	var logLevel slog.Level
	var err error

	// 1. Check CLI flag first (highest priority)
	if cli.CLI.LogLevel != "" {
		// Validate enum values manually since Kong enum requires default for optional fields
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[cli.CLI.LogLevel] {
			cli.Fatalf("Invalid log level '%s' (must be: debug, info, warn, error)", cli.CLI.LogLevel)
		}
		logLevel, err = logger.ParseLevel(cli.CLI.LogLevel)
		if err != nil {
			cli.Fatalf("Invalid log level: %v", err)
		}
	} else if cfg != nil && cfg.Global.Observability.LogLevel != "" {
		// 2. Check config file
		logLevel, err = logger.ParseLevel(cfg.Global.Observability.LogLevel)
		if err != nil {
			cli.Fatalf("Invalid log_level in config: %v", err)
		}
	} else {
		// 3. Universal default: INFO level (since logs go to file in client/local modes)
		// Important startup messages are printed directly via fmt.Printf
		logLevel = slog.LevelInfo
	}

	// Determine log file path
	// Priority: CLI flag > config file > environment variable > auto-enable for client/local modes > stderr
	var logFile *os.File
	var logFileCleanup func()
	isClientOrLocalMode := mode == cli.ModeClient || mode == cli.ModeLocalConfig || mode == cli.ModeLocalZeroConfig

	if cli.CLI.LogFile != "" {
		// CLI flag (highest priority)
		logFile, logFileCleanup, err = logger.OpenLogFile(cli.CLI.LogFile)
		if err != nil {
			cli.Fatalf("Failed to open log file: %v", err)
		}
		defer logFileCleanup()
	} else if cfg != nil && cfg.Global.Observability.LogFile != "" {
		// Config file
		logFile, logFileCleanup, err = logger.OpenLogFile(cfg.Global.Observability.LogFile)
		if err != nil {
			cli.Fatalf("Failed to open log file from config: %v", err)
		}
		defer logFileCleanup()
	} else if envLogFile := os.Getenv("LOG_FILE"); envLogFile != "" {
		// Environment variable
		logFile, logFileCleanup, err = logger.OpenLogFile(envLogFile)
		if err != nil {
			cli.Fatalf("Failed to open log file from LOG_FILE env: %v", err)
		}
		defer logFileCleanup()
	} else if isClientOrLocalMode {
		// Auto-enable file logging for client/local modes to keep stdout clean
		logFile, logFileCleanup, err = logger.OpenLogFile("hector.log")
		if err != nil {
			cli.Fatalf("Failed to create log file: %v", err)
		}
		defer logFileCleanup()
	} else {
		// Server modes: use stderr (default)
		logFile = os.Stderr
	}

	// Determine log format
	// Priority: CLI flag > config file > default (simple)
	var logFormat string
	if cli.CLI.LogFormat != "" {
		logFormat = cli.CLI.LogFormat
	} else if cfg != nil && cfg.Global.Observability.LogFormat != "" {
		logFormat = cfg.Global.Observability.LogFormat
	} else {
		logFormat = "simple" // default
	}

	// Re-initialize logger with final settings (may differ from early initialization)
	// Only re-initialize if settings changed from early initialization
	if logLevel != earlyLogLevel || logFile != earlyLogFile || logFormat != earlyLogFormat {
		logger.Init(logLevel, logFile, logFormat)
	}

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
