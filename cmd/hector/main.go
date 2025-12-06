// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command hector is the CLI for Hector pkg.
//
// Usage:
//
//	hector serve --config config.yaml
//	hector serve --provider anthropic --model claude-sonnet-4-20250514
//	hector info --config config.yaml --agent assistant
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/runtime"
	"github.com/kadirpekel/hector/pkg/server"
	"github.com/kadirpekel/hector/pkg/session"
	"github.com/kadirpekel/hector/pkg/task"
)

// CLI defines the command-line interface.
type CLI struct {
	Version  VersionCmd  `cmd:"" help:"Show version information."`
	Serve    ServeCmd    `cmd:"" help:"Start the A2A server."`
	Info     InfoCmd     `cmd:"" help:"Show agent information."`
	Validate ValidateCmd `cmd:"" help:"Validate configuration file."`
	Schema   SchemaCmd   `cmd:"" help:"Generate JSON Schema for config builder."`

	Config    string `short:"c" help:"Path to config file." type:"path"`
	LogLevel  string `help:"Log level (debug, info, warn, error)." default:"info"`
	LogFile   string `help:"Log file path (empty = stderr)."`
	LogFormat string `help:"Log format (simple, verbose, or custom)." default:"simple"`
}

// VersionCmd shows version information.
type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = info.Main.Version
		}
	}
	fmt.Printf("Hector pkg version %s\n", version)
	return nil
}

// ServeCmd starts the A2A server.
type ServeCmd struct {
	// Zero-config options
	Provider       string  `help:"LLM provider (anthropic, openai, gemini, ollama)."`
	Model          string  `help:"Model name."`
	APIKey         string  `name:"api-key" help:"API key (defaults to environment variable)."`
	BaseURL        string  `name:"base-url" help:"Custom API base URL."`
	Temperature    float64 `help:"Temperature for generation." default:"0.7"`
	MaxTokens      int     `name:"max-tokens" help:"Max tokens for generation." default:"4096"`
	Instruction    string  `help:"System instruction for the agent."`
	Role           string  `help:"Agent role."`
	MCPURL         string  `name:"mcp-url" help:"MCP server URL."`
	Tools          string  `help:"Enable built-in local tools. Empty string or 'all' enables all tools. Comma-separated list enables specific tools (e.g., 'read_file,write_file')."`
	ApproveTools   string  `name:"approve-tools" help:"Enable approval for specific tools (comma-separated, e.g., execute_command,write_file). Overrides smart defaults." placeholder:"TOOL1,TOOL2"`
	NoApproveTools string  `name:"no-approve-tools" help:"Disable approval for specific tools (comma-separated, e.g., write_file). Overrides smart defaults." placeholder:"TOOL1,TOOL2"`
	Thinking       *bool   `help:"Enable thinking at API level (like --tools enables tools)." negatable:""`
	ThinkingBudget int     `name:"thinking-budget" help:"Token budget for thinking (default: 1024, must be < max-tokens)." default:"0"`
	Stream         *bool   `default:"true" negatable:"" help:"Enable streaming responses (use --no-stream to disable)"`

	// Storage options (enables task and session persistence)
	Storage   string `name:"storage" help:"Storage backend: sqlite, postgres, mysql (default: inmemory). Also enables checkpointing." placeholder:"BACKEND"`
	StorageDB string `name:"storage-db" help:"Storage database path/DSN (default: .hector/hector.db for sqlite)." placeholder:"PATH"`

	// Observability options
	Observe bool `help:"Enable observability (metrics + OTLP tracing to localhost:4317)."`

	// RAG options (zero-config document search)
	DocsFolder    string `name:"docs-folder" help:"Folder containing documents for RAG. Auto-creates document store with chromem." type:"path" placeholder:"PATH"`
	EmbedderModel string `name:"embedder-model" help:"Embedder model (auto-detected from provider, or fallback to ollama/nomic-embed-text)." placeholder:"MODEL"`
	RAGWatch      *bool  `name:"rag-watch" default:"true" negatable:"" help:"Watch docs folder for changes and auto-reindex (enabled by default)."`
	MCPParserTool string `name:"mcp-parser-tool" help:"MCP tool name(s) for document parsing (e.g., 'convert_document_into_docling_document'). Comma-separated for fallback chain." placeholder:"TOOL_NAME"`

	// Server options
	Port  int  `help:"Port to listen on." default:"8080"`
	Watch bool `help:"Watch config file for changes."`
}

func (c *ServeCmd) Run(cli *CLI) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Shutting down...")
		cancel()
	}()

	// Load configuration
	cfg, loader, err := c.loadConfig(ctx, cli.Config)
	if err != nil {
		return err
	}
	if loader != nil {
		defer loader.Close()
	}

	// Override port if explicitly specified
	if c.Port != 0 && c.Port != 8080 {
		cfg.Server.Port = c.Port
	}

	// Start config watching if enabled
	if c.Watch && loader != nil {
		go func() {
			if err := loader.Watch(ctx); err != nil && ctx.Err() == nil {
				slog.Error("Config watch error", "error", err)
			}
		}()
	}

	// Create shared database pool for SQLite to prevent "database is locked" errors.
	// Both TaskStore and SessionService share the same connection pool.
	dbPool := config.NewDBPool()
	defer dbPool.Close()

	// Create session service with shared pool
	sessionSvc, err := session.NewSessionServiceFromConfig(cfg, dbPool)
	if err != nil {
		return fmt.Errorf("failed to create session service: %w", err)
	}

	// Build runtime with session service
	rt, err := runtime.New(cfg, runtime.WithSessionService(sessionSvc))
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}
	defer rt.Close()

	// Create per-agent executors
	executors := make(map[string]*server.Executor)
	for _, agentName := range cfg.ListAgents() {
		runnerCfg, err := rt.RunnerConfig(agentName)
		if err != nil {
			return fmt.Errorf("failed to create runner config for agent %s: %w", agentName, err)
		}
		executors[agentName] = server.NewExecutor(server.ExecutorConfig{
			RunnerConfig: *runnerCfg,
		})
	}

	// Create TaskStore with shared pool
	var serverOpts []server.HTTPServerOption
	taskStore, err := task.NewTaskStoreFromConfig(cfg, dbPool)
	if err != nil {
		return fmt.Errorf("failed to create task store: %w", err)
	}
	if taskStore != nil {
		serverOpts = append(serverOpts, server.WithTaskStore(taskStore))
		slog.Info("Task persistence enabled", "backend", cfg.Server.Tasks.Backend, "database", cfg.Server.Tasks.Database)
	}

	srv := server.NewHTTPServer(cfg, executors, serverOpts...)

	// Print startup info
	greenColor := "\033[38;2;16;185;129m"
	resetColor := "\033[0m"
	fmt.Printf("\n%s🚀 Hector pkg server ready!%s\n", greenColor, resetColor)
	fmt.Printf("   Web UI:      http://%s\n", srv.Address())
	fmt.Printf("   Agent Card:  http://%s/.well-known/agent-card.json\n", srv.Address())
	fmt.Printf("   Discovery:   http://%s/agents\n", srv.Address())
	fmt.Printf("   Health:      http://%s/health\n", srv.Address())
	if cfg.Server.Transport == config.TransportGRPC {
		fmt.Printf("   gRPC:        %s\n", srv.GRPCAddress())
	}

	// Show storage persistence status
	if cfg.Server.Tasks != nil && cfg.Server.Tasks.IsSQL() {
		dbName := cfg.Server.Tasks.Database
		if dbCfg, ok := cfg.Databases[dbName]; ok {
			fmt.Printf("   Storage:     %s (%s)\n", dbCfg.Driver, dbCfg.Database)
			fmt.Printf("   - Tasks:     persistent\n")
			if cfg.Server.Sessions != nil && cfg.Server.Sessions.IsSQL() {
				fmt.Printf("   - Sessions:  persistent\n")
			} else {
				fmt.Printf("   - Sessions:  in-memory\n")
			}
			if cfg.Server.Checkpoint != nil && cfg.Server.Checkpoint.IsEnabled() {
				fmt.Printf("   - Checkpoint: enabled (%s)\n", cfg.Server.Checkpoint.Strategy)
			}
		}
	} else {
		fmt.Printf("   Storage:     in-memory (not persisted)\n")
	}

	// Show observability status
	if cfg.Server.Observability != nil {
		if cfg.Server.Observability.Tracing.Enabled {
			fmt.Printf("   Tracing:     %s (%s)\n", cfg.Server.Observability.Tracing.Exporter, cfg.Server.Observability.Tracing.Endpoint)
		}
		if cfg.Server.Observability.Metrics.Enabled {
			fmt.Printf("   Metrics:     http://%s/metrics\n", srv.Address())
		}
	}

	// Initialize and start RAG document stores
	if len(cfg.DocumentStores) > 0 {
		// Index document stores asynchronously (non-blocking startup)
		go func() {
			if err := rt.IndexDocumentStores(ctx); err != nil {
				slog.Warn("Failed to index document stores", "error", err)
			}
		}()

		// Start file watching for auto re-indexing
		if err := rt.StartDocumentStoreWatching(ctx); err != nil {
			slog.Warn("Failed to start document store watching", "error", err)
		}

		// Show RAG status
		for name, store := range cfg.DocumentStores {
			if store.Source != nil {
				watchStatus := "enabled"
				if !store.Watch {
					watchStatus = "disabled"
				}
				fmt.Printf("   RAG Store:   %s (%s, watch=%s)\n", name, store.Source.Type, watchStatus)
			}
		}
	}

	fmt.Println("\n   Agents (A2A JSON-RPC endpoints):")
	for _, name := range cfg.ListAgents() {
		fmt.Printf("     - http://%s/agents/%s\n", srv.Address(), name)
	}
	fmt.Println("\nPress Ctrl+C to stop")

	// Start server (blocks until context is cancelled)
	return srv.Start(ctx)
}

// loadConfig loads configuration from file or creates zero-config.
func (c *ServeCmd) loadConfig(ctx context.Context, configPath string) (*config.Config, *config.Loader, error) {
	if configPath != "" {
		_ = config.LoadDotEnvForConfig(configPath)
		cfg, loader, err := config.LoadConfigFile(ctx, configPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load config: %w", err)
		}
		slog.Info("Loaded configuration", "path", configPath)
		return cfg, loader, nil
	}

	// Zero-config mode
	// Handle streaming default: Kong may not set pointer bool defaults properly
	// So we explicitly set it to true if nil (default behavior)
	streaming := c.Stream
	if streaming == nil {
		streaming = config.BoolPtr(true) // Default to true (streaming enabled)
	}
	cfg := config.CreateZeroConfig(config.ZeroConfig{
		Provider:       c.Provider,
		Model:          c.Model,
		APIKey:         c.APIKey,
		BaseURL:        c.BaseURL,
		Temperature:    c.Temperature,
		MaxTokens:      c.MaxTokens,
		Instruction:    c.Instruction,
		Role:           c.Role,
		MCPURL:         c.MCPURL,
		Tools:          c.Tools,
		ApproveTools:   c.ApproveTools,
		NoApproveTools: c.NoApproveTools,
		Thinking:       c.Thinking,
		ThinkingBudget: c.ThinkingBudget,
		Streaming:      streaming,
		Storage:        c.Storage,
		StorageDB:      c.StorageDB,
		Observe:        c.Observe,
		Port:           c.Port,
		DocsFolder:     c.DocsFolder,
		EmbedderModel:  c.EmbedderModel,
		RAGWatch:       c.RAGWatch,
		MCPParserTool:  c.MCPParserTool,
	})
	slog.Info("Using zero-config mode")
	if c.Tools != "" {
		if c.Tools == "all" || strings.TrimSpace(c.Tools) == "" {
			slog.Info("All built-in local tools enabled")
		} else {
			slog.Info("Selected built-in local tools enabled", "tools", c.Tools)
		}
	}
	if c.Storage != "" && c.Storage != "inmemory" {
		dbInfo := c.StorageDB
		if dbInfo == "" {
			// Show default database info based on backend
			switch c.Storage {
			case "sqlite", "sqlite3":
				dbInfo = "./.hector/hector.db"
			case "postgres":
				dbInfo = "localhost:5432/hector"
			case "mysql":
				dbInfo = "localhost:3306/hector"
			}
		}
		slog.Info("Persistent storage enabled", "backend", c.Storage, "database", dbInfo)
		slog.Info("Checkpointing auto-enabled", "strategy", "hybrid")
	}
	if c.Observe {
		slog.Info("Observability enabled", "tracing", "otlp://localhost:4317", "metrics", "prometheus")
	}
	if c.DocsFolder != "" {
		embedderModel := c.EmbedderModel
		if embedderModel == "" {
			embedderModel = "(auto-detected)"
		}
		watchEnabled := c.RAGWatch == nil || *c.RAGWatch
		slog.Info("RAG enabled", "docs_folder", c.DocsFolder, "embedder", embedderModel, "watch", watchEnabled)
		if c.MCPParserTool != "" {
			slog.Info("MCP document parsing enabled", "tools", c.MCPParserTool)
		}
	}
	if cfg.Agents != nil {
		for name, agent := range cfg.Agents {
			if agent != nil {
				streamingEnabled := config.BoolValue(agent.Streaming, false)
				slog.Info("Agent streaming configuration", "agent", name, "streaming", streamingEnabled)
			}
		}
	}
	return cfg, nil, nil
}

// InfoCmd shows agent information.
type InfoCmd struct {
	Agent string `arg:"" optional:"" help:"Agent name to show info for."`
}

func (c *InfoCmd) Run(cli *CLI) error {
	ctx := context.Background()

	if cli.Config == "" {
		return fmt.Errorf("--config is required for info command")
	}

	_ = config.LoadDotEnvForConfig(cli.Config)
	cfg, loader, err := config.LoadConfigFile(ctx, cli.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	defer loader.Close()

	if c.Agent == "" {
		fmt.Println("Available agents:")
		for _, name := range cfg.ListAgents() {
			agent, _ := cfg.GetAgent(name)
			desc := agent.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Printf("  - %s: %s\n", name, desc)
		}
		return nil
	}

	agent, ok := cfg.GetAgent(c.Agent)
	if !ok {
		return fmt.Errorf("agent %q not found", c.Agent)
	}

	fmt.Printf("\nAgent: %s\n", c.Agent)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Name:        %s\n", agent.GetDisplayName())
	if agent.Description != "" {
		fmt.Printf("Description: %s\n", agent.Description)
	}
	fmt.Printf("LLM:         %s\n", agent.LLM)
	if len(agent.Tools) > 0 {
		fmt.Printf("Tools:       %v\n", agent.Tools)
	}
	if len(agent.InputModes) > 0 {
		fmt.Printf("Input:       %v\n", agent.InputModes)
	}
	if len(agent.OutputModes) > 0 {
		fmt.Printf("Output:      %v\n", agent.OutputModes)
	}

	return nil
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
	// Use ANSI RGB color mode: \033[38;2;R;G;Bm
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

// shouldSkipBanner checks if command should skip banner
// In pkg, "info", "validate", and "schema" commands skip banner (they're informational, not server)
func shouldSkipBanner(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check for informational commands
	for _, arg := range args {
		// Skip program name and flags, look for commands
		if arg == "info" || arg == "validate" || arg == "schema" {
			return true
		}
	}
	return false
}

func main() {
	// Skip banner for informational commands (info, validate)
	if !shouldSkipBanner(os.Args) {
		printBanner()
	}

	// Validate mutual exclusivity of --config and zero-config flags BEFORE kong parsing.
	// This provides clear error messages when users mix config file and zero-config modes.
	// Ported from legacy pkg/cli/mutual_exclusivity.go
	if !ShouldSkipValidation(os.Args) {
		if err := ValidateConfigMutualExclusivity(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	_ = config.LoadDotEnv()

	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("hector"),
		kong.Description("Hector pkg - Config-first AI Agent Platform"),
		kong.UsageOnError(),
	)

	// Initialize logger with CLI flags/env vars (before config loading)
	// Config file logger settings will be applied later if no CLI/env overrides
	_, _, _, cleanup, err := initLoggerFromCLI(cli.LogLevel, cli.LogFile, cli.LogFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	err = ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
