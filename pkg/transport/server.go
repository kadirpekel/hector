package transport

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

type Server struct {
	service    pb.A2AServiceServer
	grpcServer *grpc.Server
	listener   net.Listener
	config     Config
}

type Config struct {
	Address           string
	UnaryInterceptor  grpc.UnaryServerInterceptor
	StreamInterceptor grpc.StreamServerInterceptor
}

func NewServer(service pb.A2AServiceServer, config Config) *Server {
	if config.Address == "" {
		config.Address = ":8080"
	}

	return &Server{
		service: service,
		config:  config,
	}
}

func (s *Server) Start() error {

	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Address, err)
	}
	s.listener = listener

	opts := []grpc.ServerOption{}

	if s.config.UnaryInterceptor != nil {
		opts = append(opts, grpc.UnaryInterceptor(s.config.UnaryInterceptor))
	}

	if s.config.StreamInterceptor != nil {
		opts = append(opts, grpc.StreamInterceptor(s.config.StreamInterceptor))
	}

	s.grpcServer = grpc.NewServer(opts...)

	pb.RegisterA2AServiceServer(s.grpcServer, s.service)

	reflection.Register(s.grpcServer)

	slog.Info("A2A gRPC server starting", "address", s.config.Address)

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.grpcServer == nil {
		return nil
	}

	slog.Info("Shutting down A2A gRPC server")

	stopped := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		slog.Info("Server stopped gracefully")
	case <-ctx.Done():
		slog.Warn("Graceful stop timeout, forcing shutdown")
		s.grpcServer.Stop()
	}

	return nil
}

func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.Address
}

func (s *Server) StopWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.Stop(ctx)
}
