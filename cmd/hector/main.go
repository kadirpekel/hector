package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/kadirpekel/hector/pkg/cli"
	"github.com/kadirpekel/hector/pkg/config"
)

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}
	return "dev"
}

func main() {

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
