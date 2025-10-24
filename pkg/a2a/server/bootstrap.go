package server

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/runtime"
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
// Now uses Runtime as the foundation to avoid duplication
func Bootstrap(opts BootstrapOptions) (*HectorServer, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	log.Println("📋 Registering agents...")

	// Use Runtime as the CORE foundation (no duplication!)
	rt, err := runtime.NewWithConfig(opts.Config)
	if err != nil {
		return nil, fmt.Errorf("runtime initialization failed: %w", err)
	}

	// Get the registry from Runtime
	agentRegistry := rt.Registry()

	// Log registered agents with visibility info
	for _, agentID := range agentRegistry.ListAgents() {
		entry, _ := agentRegistry.Get(agentID)
		visibility := entry.Config.Visibility
		if visibility == "" {
			visibility = "public"
		}

		if entry.Config.Type == "a2a" {
			log.Printf("  ✅ External agent '%s' connected to %s (visibility: %s)", agentID, entry.Config.URL, visibility)
		} else {
			log.Printf("  ✅ Native agent '%s' created (visibility: %s)", agentID, visibility)
		}
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
	if opts.Config.Global.Auth.IsEnabled() {
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

	// Set service for direct SSE streaming (bypasses grpc-gateway for true SSE format)
	restGateway.SetService(registryService)

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

	// Create server instance with Runtime foundation
	server := &HectorServer{
		config: &ServerConfig{
			GRPCAddress:    grpcAddr,
			RESTAddress:    restAddr,
			JSONRPCAddress: jsonrpcAddr,
			EnableAuth:     opts.Config.Global.Auth.IsEnabled(),
			JWKSURL:        opts.Config.Global.Auth.JWKSURL,
			Issuer:         opts.Config.Global.Auth.Issuer,
			Audience:       opts.Config.Global.Auth.Audience,
		},
		runtime:      rt, // Store Runtime for proper lifecycle management
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
