package server

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/transport"
)

// BootstrapOptions holds options for server bootstrapping
type BootstrapOptions struct {
	Config      *config.Config
	GRPCPort    int
	RESTPort    int
	JSONRPCPort int
}

// Bootstrap creates and initializes a HectorServer from configuration
func Bootstrap(opts BootstrapOptions) (*HectorServer, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create component manager
	componentManager, err := component.NewComponentManager(opts.Config)
	if err != nil {
		return nil, fmt.Errorf("component initialization failed: %w", err)
	}

	log.Println("📋 Registering agents...")

	// Create agent registry
	agentRegistry := agent.NewAgentRegistry()

	// Register all configured agents
	agentCount := 0
	for agentID, agentCfg := range opts.Config.Agents {
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
			agentInstance, err = agent.NewAgent(&cfg, componentManager, agentRegistry)
			if err != nil {
				log.Printf("  ⚠️  Failed to create native agent '%s': %v", agentID, err)
				continue
			}
			log.Printf("  ✅ Native agent '%s' created", agentID)
		}

		// Set default visibility
		visibility := cfg.Visibility
		if visibility == "" {
			visibility = "public"
		}

		// Register agent
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &cfg, nil); err != nil {
			log.Printf("  ⚠️  Failed to register agent '%s': %v", agentID, err)
			continue
		}

		log.Printf("  ✅ Registered agent: %s (visibility: %s)", agentID, visibility)
		agentCount++
	}

	if agentCount == 0 {
		return nil, fmt.Errorf("no agents successfully registered")
	}

	// Create registry service (wraps the agent registry)
	registryService := transport.NewRegistryService(agentRegistry)

	// Determine addresses
	grpcAddr := fmt.Sprintf(":%d", opts.GRPCPort)
	restAddr := fmt.Sprintf(":%d", opts.RESTPort)
	jsonrpcAddr := fmt.Sprintf(":%d", opts.JSONRPCPort)

	// Configure authentication
	var authConfig *transport.AuthConfig
	var jwtValidator *auth.JWTValidator
	if opts.Config.Global.Auth.Enabled {
		var err error
		jwtValidator, err = auth.NewJWTValidator(
			opts.Config.Global.Auth.JWKSURL,
			opts.Config.Global.Auth.Issuer,
			opts.Config.Global.Auth.Audience,
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
	grpcServer := transport.NewServer(registryService, grpcConfig)

	// Create REST gateway with auth
	// Connect to gRPC server instead of in-process for proper streaming support
	restGateway := transport.NewRESTGateway(transport.RESTGatewayConfig{
		HTTPAddress: restAddr,
		GRPCAddress: "localhost" + grpcAddr, // Connect to actual gRPC server for streaming
	})
	if authConfig != nil {
		restGateway.SetAuth(authConfig)
	}

	// Set up agent discovery endpoint
	// RegistryService implements DiscoverableService interface
	discovery := transport.NewAgentDiscovery(registryService, authConfig)
	restGateway.SetDiscovery(discovery)

	// Create JSON-RPC handler with auth
	jsonrpcHandler := transport.NewJSONRPCHandler(
		transport.JSONRPCConfig{HTTPAddress: jsonrpcAddr},
		registryService,
	)
	if authConfig != nil {
		jsonrpcHandler.SetAuth(authConfig)
	}

	// Create server instance
	server := &HectorServer{
		config: &ServerConfig{
			GRPCAddress:    grpcAddr,
			RESTAddress:    restAddr,
			JSONRPCAddress: jsonrpcAddr,
			EnableAuth:     opts.Config.Global.Auth.Enabled,
			JWKSURL:        opts.Config.Global.Auth.JWKSURL,
			Issuer:         opts.Config.Global.Auth.Issuer,
			Audience:       opts.Config.Global.Auth.Audience,
		},
		registry:     agentRegistry,
		service:      registryService,
		grpc:         grpcServer,
		rest:         restGateway,
		jsonrpc:      jsonrpcHandler,
		authConfig:   authConfig,
		jwtValidator: jwtValidator,
	}

	return server, nil
}
