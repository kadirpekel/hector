// Package server provides the HectorServer which coordinates multiple A2A transports
package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/transport"
)

// HectorServer coordinates multiple A2A protocol transports (gRPC, REST, JSON-RPC)
type HectorServer struct {
	config       *ServerConfig
	registry     *agent.AgentRegistry
	service      pb.A2AServiceServer
	grpc         *transport.Server
	rest         *transport.RESTGateway
	jsonrpc      *transport.JSONRPCHandler
	authConfig   *transport.AuthConfig
	jwtValidator *auth.JWTValidator
}

// ServerConfig holds configuration for the HectorServer
type ServerConfig struct {
	GRPCAddress    string
	RESTAddress    string
	JSONRPCAddress string

	// Auth configuration
	EnableAuth bool
	JWKSURL    string
	Issuer     string
	Audience   string
}

// Start starts all transports and blocks until shutdown signal
func (s *HectorServer) Start(ctx context.Context) error {
	log.Printf("🚀 Starting Hector A2A Server v%s...", getVersion())

	// Start all transports in goroutines
	errChan := make(chan error, 3)

	go func() {
		if err := s.grpc.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		if err := s.rest.Start(ctx); err != nil {
			errChan <- fmt.Errorf("REST gateway error: %w", err)
		}
	}()

	go func() {
		if err := s.jsonrpc.Start(); err != nil {
			errChan <- fmt.Errorf("JSON-RPC handler error: %w", err)
		}
	}()

	// Log startup info
	agentList := s.service.(*transport.RegistryService).ListAgents()
	log.Printf("\n🎉 Hector - All transports started!")
	log.Printf("📡 Agents available: %d", len(agentList))
	for _, agentID := range agentList {
		log.Printf("   • %s", agentID)
	}
	log.Printf("\n🌐 Endpoints:")
	log.Printf("   → gRPC:     %s", s.grpc.Address())
	log.Printf("   → REST:     http://0.0.0.0%s (via gRPC proxy, streaming enabled)", s.config.RESTAddress)
	log.Printf("   → JSON-RPC: http://0.0.0.0%s/rpc", s.config.JSONRPCAddress)
	log.Printf("\n📋 Discovery & Agent Cards:")
	log.Printf("   → Service Card: http://0.0.0.0%s/.well-known/agent-card.json", s.config.RESTAddress)
	log.Printf("   → Agent List:   http://0.0.0.0%s/v1/agents", s.config.RESTAddress)
	log.Printf("   → Agent Cards:  http://0.0.0.0%s/v1/agents/{agent_id}/.well-known/agent-card.json", s.config.RESTAddress)
	log.Printf("\n💡 A2A-compliant endpoints (per agent):")
	for _, agentID := range agentList {
		log.Printf("   → http://0.0.0.0%s/v1/agents/%s/message:send", s.config.RESTAddress, agentID)
		log.Printf("   → http://0.0.0.0%s/v1/agents/%s/message:stream", s.config.RESTAddress, agentID)
	}

	if s.authConfig != nil && s.authConfig.Enabled {
		log.Printf("\n🔐 Authentication: Enabled (JWT)")
	}

	log.Printf("\n✅ Server ready! Press Ctrl+C to stop.\n")

	// Wait for shutdown signal or error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		log.Printf("\n⚠️  Received signal %v, shutting down gracefully...", sig)
		return s.Stop(ctx)
	case <-ctx.Done():
		log.Printf("\n⚠️  Context cancelled, shutting down gracefully...")
		return s.Stop(ctx)
	}
}

// Stop gracefully stops all transports
func (s *HectorServer) Stop(ctx context.Context) error {
	log.Println("🛑 Stopping all transports...")

	// Stop gRPC server
	if s.grpc != nil {
		if err := s.grpc.Stop(ctx); err != nil {
			log.Printf("   ⚠️  gRPC server stop error: %v", err)
		} else {
			log.Println("   ✅ gRPC server stopped")
		}
	}

	// Stop REST gateway
	if s.rest != nil {
		if err := s.rest.Stop(ctx); err != nil {
			log.Printf("   ⚠️  REST gateway stop error: %v", err)
		} else {
			log.Println("   ✅ REST gateway stopped")
		}
	}

	// Stop JSON-RPC handler
	if s.jsonrpc != nil {
		if err := s.jsonrpc.Stop(ctx); err != nil {
			log.Printf("   ⚠️  JSON-RPC handler stop error: %v", err)
		} else {
			log.Println("   ✅ JSON-RPC handler stopped")
		}
	}

	// Close JWT validator
	if s.jwtValidator != nil {
		s.jwtValidator.Close()
		log.Println("   ✅ JWT validator closed")
	}

	log.Println("👋 Server shutdown complete")
	return nil
}

// Registry returns the agent registry
func (s *HectorServer) Registry() *agent.AgentRegistry {
	return s.registry
}

// Service returns the underlying A2A service
func (s *HectorServer) Service() pb.A2AServiceServer {
	return s.service
}

// getVersion returns the build version
func getVersion() string {
	return "dev" // This will be set by build flags
}
