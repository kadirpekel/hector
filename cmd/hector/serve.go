package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/cli"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/transport"
)

// executeServeCommand handles the 'serve' command - starts the A2A server
func executeServeCommand(args *cli.CLIArgs) {

	// Load configuration (from file or zero-config mode)
	loadResult, err := config.LoadOrCreateConfig(args.ConfigFile, args.ToZeroConfigOptions())
	if err != nil {
		cli.Fatalf("Failed to load configuration: %v", err)
	}

	hectorConfig := loadResult.Config

	// Display config mode info
	if loadResult.IsZeroConfig {
		if args.Debug {
			fmt.Print(config.FormatZeroConfigDebug(loadResult.ResolvedOpts))
		}
		// Show MCP info even without debug flag (like we do for MCP discovery)
		if loadResult.ResolvedOpts.MCPURL != "" && !args.Debug {
			fmt.Printf("🔌 MCP server: %s\n", loadResult.ResolvedOpts.MCPURL)
		}
	} else if args.Debug {
		fmt.Printf("🔧 Loaded configuration from: %s\n", args.ConfigFile)
	}

	// Set defaults and validate
	hectorConfig.SetDefaults()
	if err := hectorConfig.Validate(); err != nil {
		cli.Fatalf("Invalid configuration: %v", err)
	}

	// Create agent registry
	agentRegistry := agent.NewAgentRegistry()

	// Create component manager with agent registry for agent_call tool
	componentManager, err := component.NewComponentManagerWithAgentRegistry(hectorConfig, agentRegistry)
	if err != nil {
		cli.Fatalf("Component initialization failed: %v", err)
	}

	// Create agent router (routes A2A requests to individual agents)
	// Following LangGraph's A2A pattern: router just routes, agents have their own identity
	agentRouter := agent.NewAgentRouter(agentRegistry)

	// Register all configured agents
	fmt.Println("\n📋 Registering agents...")
	for agentID, agentCfg := range hectorConfig.Agents {
		cfg := agentCfg

		// Create agent based on type (native vs external)
		var agentInstance pb.A2AServiceServer
		var err error

		if cfg.Type == "a2a" {
			// External A2A agent - create client proxy
			externalAgent, extErr := agent.NewExternalA2AAgent(&cfg)
			if extErr != nil {
				log.Printf("  ⚠️  Failed to create external agent '%s': %v", agentID, extErr)
				continue
			}
			agentInstance = externalAgent
			log.Printf("  ✅ External agent '%s' connected to %s", agentID, cfg.URL)
		} else {
			// Native agent - create local instance
			agentInstance, err = agent.NewAgent(agentID, &cfg, componentManager, agentRegistry)
			if err != nil {
				log.Printf("  ⚠️  Failed to create native agent '%s': %v", agentID, err)
				continue
			}
			log.Printf("  ✅ Native agent '%s' created", agentID)
		}

		// Register agent in registry (single source of truth)
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &cfg, nil); err != nil {
			log.Printf("  ⚠️  Failed to register agent '%s' in registry: %v", agentID, err)
			continue
		}

		// Register with router (for logging only)
		agentRouter.RegisterAgent(agentID, agentInstance)
	}

	if agentRouter.AgentCount() == 0 {
		log.Fatalf("❌ No agents successfully registered")
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
	hasA2AConfig := hectorConfig.Global.A2AServer.IsEnabled()
	if hasA2AConfig {
		if hectorConfig.Global.A2AServer.Port > 0 {
			basePort = hectorConfig.Global.A2AServer.Port
		}
		if hectorConfig.Global.A2AServer.Host != "" {
			serverHost = hectorConfig.Global.A2AServer.Host
		}
		if hectorConfig.Global.A2AServer.BaseURL != "" {
			baseURL = hectorConfig.Global.A2AServer.BaseURL
		}
	}

	// CLI flags override config (conventional behavior)
	if args.Port != 8080 { // User specified --port (different from default)
		if hasA2AConfig && hectorConfig.Global.A2AServer.Port > 0 && args.Port != hectorConfig.Global.A2AServer.Port {
			overrides = append(overrides, fmt.Sprintf("port: %d (config: %d)", args.Port, hectorConfig.Global.A2AServer.Port))
		}
		basePort = args.Port
	}

	if args.Host != "" { // User specified --host
		if hasA2AConfig && hectorConfig.Global.A2AServer.Host != "" && args.Host != hectorConfig.Global.A2AServer.Host {
			overrides = append(overrides, fmt.Sprintf("host: %s (config: %s)", args.Host, hectorConfig.Global.A2AServer.Host))
		}
		serverHost = args.Host
	}

	if args.A2ABaseURL != "" { // User specified --a2a-base-url
		if hasA2AConfig && hectorConfig.Global.A2AServer.BaseURL != "" && args.A2ABaseURL != hectorConfig.Global.A2AServer.BaseURL {
			overrides = append(overrides, fmt.Sprintf("base_url: %s (config: %s)", args.A2ABaseURL, hectorConfig.Global.A2AServer.BaseURL))
		}
		baseURL = args.A2ABaseURL
	}

	if args.Debug {
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
	if hectorConfig.Global.Auth.IsEnabled() {
		var err error
		jwtValidator, err = auth.NewJWTValidator(
			hectorConfig.Global.Auth.JWKSURL,
			hectorConfig.Global.Auth.Issuer,
			hectorConfig.Global.Auth.Audience,
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

	log.Printf("\n🎉 Hector v%s - All transports started!", getVersion())
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
		os.Exit(1)
	}

	log.Printf("👋 All servers shut down gracefully")
}
