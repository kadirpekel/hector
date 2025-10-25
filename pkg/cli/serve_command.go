package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/runtime"
	"github.com/kadirpekel/hector/pkg/transport"
)

// ServeCommand starts the A2A server to host agents
func ServeCommand(args *ServeCmd, cfg *config.Config, mode CLIMode) error {
	// Config is already loaded, validated, and defaults set by main.go
	// Display config mode info
	if CLI.Config == "" {
		// Zero-config mode
		if CLI.Debug {
			fmt.Print(formatZeroConfigDebug(args))
		}
		// Show MCP info even without debug flag (like we do for MCP discovery)
		if args.MCPURL != "" && !CLI.Debug {
			fmt.Printf("🔌 MCP server: %s\n", args.MCPURL)
		}
	} else if CLI.Debug {
		fmt.Printf("🔧 Loaded configuration from: %s\n", CLI.Config)
	}

	// Create runtime - this initializes ComponentManager, AgentRegistry, and all agents
	fmt.Println("\n📋 Registering agents...")
	rt, err := runtime.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("runtime initialization failed: %w", err)
	}

	// Get registry and create router for transport layer
	agentRegistry := rt.Registry()
	agentRouter := agent.NewAgentRouter(agentRegistry)

	// Register agents with router (for logging only)
	for _, agentID := range agentRegistry.ListAgents() {
		entry, _ := agentRegistry.Get(agentID)
		agentRouter.RegisterAgent(agentID, entry.Agent)

		// Log registration
		if entry.Config.Type == "a2a" {
			log.Printf("  ✅ External agent '%s' connected to %s", agentID, entry.Config.URL)
		} else {
			log.Printf("  ✅ Native agent '%s' created", agentID)
		}
	}

	if agentRouter.AgentCount() == 0 {
		return fmt.Errorf("no agents successfully registered")
	}

	// Determine addresses - CLI flags override config (conventional pattern)
	var basePort int
	var serverHost string
	var baseURL string
	var overrides []string

	// Start with defaults
	basePort = 8080 // Match A2A server default
	serverHost = "0.0.0.0"
	baseURL = ""

	// Load config values if A2A server is configured (presence implies enabled)
	hasA2AConfig := cfg.Global.A2AServer.IsEnabled()
	if hasA2AConfig {
		if cfg.Global.A2AServer.Port > 0 {
			basePort = cfg.Global.A2AServer.Port
		}
		if cfg.Global.A2AServer.Host != "" {
			serverHost = cfg.Global.A2AServer.Host
		}
		if cfg.Global.A2AServer.BaseURL != "" {
			baseURL = cfg.Global.A2AServer.BaseURL
		}
	}

	// CLI flags override config (conventional behavior)
	if args.Port != 8080 { // User specified --port (different from default)
		if hasA2AConfig && cfg.Global.A2AServer.Port > 0 && args.Port != cfg.Global.A2AServer.Port {
			overrides = append(overrides, fmt.Sprintf("port: %d (config: %d)", args.Port, cfg.Global.A2AServer.Port))
		}
		basePort = args.Port
	}

	if args.Host != "" { // User specified --host
		if hasA2AConfig && cfg.Global.A2AServer.Host != "" && args.Host != cfg.Global.A2AServer.Host {
			overrides = append(overrides, fmt.Sprintf("host: %s (config: %s)", args.Host, cfg.Global.A2AServer.Host))
		}
		serverHost = args.Host
	}

	if args.A2ABaseURL != "" { // User specified --a2a-base-url
		if hasA2AConfig && cfg.Global.A2AServer.BaseURL != "" && args.A2ABaseURL != cfg.Global.A2AServer.BaseURL {
			overrides = append(overrides, fmt.Sprintf("base_url: %s (config: %s)", args.A2ABaseURL, cfg.Global.A2AServer.BaseURL))
		}
		baseURL = args.A2ABaseURL
	}

	if CLI.Debug {
		if hasA2AConfig {
			fmt.Printf("🔧 A2A server config loaded:\n")
			fmt.Printf("   Host: %s\n", serverHost)
			fmt.Printf("   Port: %d\n", basePort)
			if baseURL != "" {
				fmt.Printf("   Base URL: %s\n", baseURL)
			}
			if len(overrides) > 0 {
				fmt.Printf("🚨 CLI overrides: %s\n", strings.Join(overrides, ", "))
			}
		} else {
			fmt.Printf("🔧 Using defaults with CLI port: %s:%d\n", serverHost, basePort)
		}
	}

	grpcAddr := fmt.Sprintf("%s:%d", serverHost, basePort)
	restAddr := fmt.Sprintf("%s:%d", serverHost, basePort+1)
	jsonrpcAddr := fmt.Sprintf("%s:%d", serverHost, basePort+2)

	// Configure authentication
	var authConfig *transport.AuthConfig
	var jwtValidator *auth.JWTValidator
	if cfg.Global.Auth.IsEnabled() {
		var err error
		jwtValidator, err = auth.NewJWTValidator(
			cfg.Global.Auth.JWKSURL,
			cfg.Global.Auth.Issuer,
			cfg.Global.Auth.Audience,
		)
		if err != nil {
			log.Printf("⚠️  Failed to initialize JWT validator: %v", err)
		} else {
			authConfig = &transport.AuthConfig{
				Enabled:   true,
				Validator: jwtValidator,
			}
			log.Printf("✅ Authentication configured (JWT)")
		}
	}

	// Create gRPC server with auth interceptors
	grpcConfig := transport.Config{
		Address: grpcAddr,
	}
	if jwtValidator != nil {
		grpcConfig.UnaryInterceptor = jwtValidator.UnaryServerInterceptor()
		grpcConfig.StreamInterceptor = jwtValidator.StreamServerInterceptor()
	}
	grpcServer := transport.NewServer(agentRouter, grpcConfig)

	// Create REST gateway with auth
	restGateway := transport.NewRESTGateway(transport.RESTGatewayConfig{
		HTTPAddress: restAddr,
		GRPCAddress: grpcAddr, // Point to the correct gRPC server
	})
	if authConfig != nil {
		restGateway.SetAuth(authConfig)
	}

	// Set up agent discovery endpoint
	discovery := transport.NewAgentDiscovery(agentRouter, authConfig)
	restGateway.SetDiscovery(discovery)

	jsonrpcHandler := transport.NewJSONRPCHandler(
		transport.JSONRPCConfig{HTTPAddress: jsonrpcAddr},
		agentRouter,
	)
	if authConfig != nil {
		jsonrpcHandler.SetAuth(authConfig)
	}

	// Start all servers
	errChan := make(chan error, 3)

	go func() {
		if err := grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		if err := restGateway.Start(context.Background()); err != nil {
			errChan <- fmt.Errorf("REST gateway error: %w", err)
		}
	}()

	go func() {
		if err := jsonrpcHandler.Start(); err != nil {
			errChan <- fmt.Errorf("JSON-RPC handler error: %w", err)
		}
	}()

	log.Printf("\n🎉 Hector - All transports started!")
	log.Printf("📡 Agents available: %d", agentRouter.AgentCount())
	for _, agentID := range agentRouter.ListAgents() {
		log.Printf("   • %s", agentID)
	}
	log.Printf("\n🌐 Endpoints:")
	log.Printf("   → gRPC:     %s", grpcServer.Address())
	log.Printf("   → REST:     http://%s:%d", serverHost, basePort+1)
	log.Printf("   → JSON-RPC: http://%s:%d/rpc", serverHost, basePort+2)
	log.Printf("\n📋 Discovery & Agent Cards:")
	if baseURL != "" {
		log.Printf("   → Service Card: %s/.well-known/agent-card.json", baseURL)
		log.Printf("   → Agent List:   %s/v1/agents", baseURL)
		log.Printf("   → Agent Cards:  %s/v1/agents/{name}/.well-known/agent-card.json", baseURL)
	} else {
		log.Printf("   → Service Card: http://%s:%d/.well-known/agent-card.json", serverHost, basePort+1)
		log.Printf("   → Agent List:   http://%s:%d/v1/agents", serverHost, basePort+1)
		log.Printf("   → Agent Cards:  http://%s:%d/v1/agents/{name}/.well-known/agent-card.json", serverHost, basePort+1)
	}
	log.Printf("\n💡 A2A-compliant endpoints (per agent):")
	endpointBase := baseURL
	if endpointBase == "" {
		endpointBase = fmt.Sprintf("http://%s:%d", serverHost, basePort+1)
	}
	log.Printf("   POST %s/v1/agents/{name}/message:send", endpointBase)
	log.Printf("   POST %s/v1/agents/{name}/message:stream", endpointBase)
	log.Printf("\n💡 Test commands:")
	log.Printf("   hector list")
	log.Printf("   hector info <agent>")
	log.Printf("   hector call <agent> \"your prompt\"")
	log.Printf("   hector chat <agent>")
	log.Println("\nPress Ctrl+C to stop")

	// Wait for signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("\n🛑 Shutting down...")
	case err := <-errChan:
		log.Printf("\n❌ Server error: %v", err)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErrors []error
	if err := grpcServer.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("gRPC: %w", err))
	}
	if err := restGateway.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("REST: %w", err))
	}
	if err := jsonrpcHandler.Stop(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("JSON-RPC: %w", err))
	}

	if len(shutdownErrors) > 0 {
		log.Printf("⚠️  Errors during shutdown:")
		for _, err := range shutdownErrors {
			log.Printf("   - %v", err)
		}
		return fmt.Errorf("shutdown errors occurred")
	}

	log.Printf("👋 All servers shut down gracefully")
	return nil
}

// formatZeroConfigDebug returns formatted debug output for zero-config mode
// Reconstructed from CLI args
func formatZeroConfigDebug(args *ServeCmd) string {
	output := "🔧 Zero-config mode:\n"
	output += fmt.Sprintf("  Provider: %s\n", args.Provider)
	output += fmt.Sprintf("  Model: %s\n", args.Model)
	if args.BaseURL != "" {
		output += fmt.Sprintf("  Base URL: %s\n", args.BaseURL)
	}
	if args.MCPURL != "" {
		output += fmt.Sprintf("  MCP URL: %s\n", args.MCPURL)
	}
	if args.DocsFolder != "" {
		output += fmt.Sprintf("  Docs Folder: %s\n", args.DocsFolder)
	}
	output += fmt.Sprintf("  Tools: %t\n", args.Tools)
	output += fmt.Sprintf("  Agent Name: %s\n", config.DefaultAgentName)
	return output
}
