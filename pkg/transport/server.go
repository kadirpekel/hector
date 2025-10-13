package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// Server manages the lifecycle of the gRPC A2A server
type Server struct {
	service    pb.A2AServiceServer
	grpcServer *grpc.Server
	listener   net.Listener
	config     Config
}

// Config holds configuration for the server
type Config struct {
	Address           string                       // e.g., ":8080" or "0.0.0.0:8080"
	UnaryInterceptor  grpc.UnaryServerInterceptor  // Optional: auth interceptor
	StreamInterceptor grpc.StreamServerInterceptor // Optional: auth interceptor
}

// NewServer creates a new gRPC server with the given A2A service implementation
func NewServer(service pb.A2AServiceServer, config Config) *Server {
	if config.Address == "" {
		config.Address = ":8080" // Default to A2A server default port
	}

	return &Server{
		service: service,
		config:  config,
		// grpcServer will be created in Start() with interceptors
	}
}

// Start starts the gRPC server (blocking call)
// This will listen for incoming connections and serve requests
func (s *Server) Start() error {
	// Create TCP listener
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Address, err)
	}
	s.listener = listener

	// Create gRPC server with optional auth interceptors
	opts := []grpc.ServerOption{}

	if s.config.UnaryInterceptor != nil {
		opts = append(opts, grpc.UnaryInterceptor(s.config.UnaryInterceptor))
	}

	if s.config.StreamInterceptor != nil {
		opts = append(opts, grpc.StreamInterceptor(s.config.StreamInterceptor))
	}

	s.grpcServer = grpc.NewServer(opts...)

	// Register the A2A service
	pb.RegisterA2AServiceServer(s.grpcServer, s.service)

	// Enable gRPC reflection for debugging (grpcurl, grpcui)
	reflection.Register(s.grpcServer)

	log.Printf("🚀 A2A gRPC server starting on %s", s.config.Address)

	// Start serving (blocking)
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.grpcServer == nil {
		return nil
	}

	log.Printf("🛑 Shutting down A2A gRPC server...")

	// Create a channel to signal when graceful stop completes
	stopped := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-stopped:
		log.Printf("✅ Server stopped gracefully")
	case <-ctx.Done():
		log.Printf("⚠️  Graceful stop timeout, forcing shutdown")
		s.grpcServer.Stop() // Force stop
	}

	return nil
}

// Address returns the server's listening address
func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.Address
}

// StopWithTimeout stops the server with a default 30-second timeout
func (s *Server) StopWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.Stop(ctx)
}
