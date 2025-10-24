package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/runtime"
	"github.com/kadirpekel/hector/pkg/transport"
)

type Server struct {
	config       *config.Config
	configLoader *config.Loader
	opts         Options

	runtime       *runtime.Runtime
	observability *observability.Manager

	grpcServer    *transport.Server
	restGateway   *transport.RESTGateway
	jsonrpcServer *transport.JSONRPCHandler

	stopChan   chan struct{}
	reloadChan chan struct{}
	doneChan   chan struct{}
}

type Options struct {
	Config *config.Config

	ConfigLoader *config.Loader

	Host    string
	Port    int
	BaseURL string
	Debug   bool
}

func New(opts Options) (*Server, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	s := &Server{
		config:       opts.Config,
		configLoader: opts.ConfigLoader,
		opts:         opts,
		stopChan:     make(chan struct{}),
		reloadChan:   make(chan struct{}, 1),
		doneChan:     make(chan struct{}),
	}

	if s.configLoader != nil {
		s.configLoader.SetOnChange(func(newCfg *config.Config) error {
			log.Printf("📝 Configuration change detected, triggering server reload...")
			s.config = newCfg
			select {
			case s.reloadChan <- struct{}{}:

			default:

			}
			return nil
		})
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.initialize(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if err := s.startTransport(); err != nil {
		s.cleanup(context.Background())
		return fmt.Errorf("failed to start transport: %w", err)
	}

	s.logStartup()

	go s.runLifecycle()

	return nil
}

func (s *Server) Wait() {
	<-s.doneChan

	if s.configLoader != nil {
		s.configLoader.Stop()
	}
}

func (s *Server) Stop(ctx context.Context) error {
	close(s.stopChan)

	select {
	case <-s.doneChan:

		if s.configLoader != nil {
			s.configLoader.Stop()
		}
		return nil
	case <-ctx.Done():

		if s.configLoader != nil {
			s.configLoader.Stop()
		}
		return ctx.Err()
	}
}

func (s *Server) initialize() error {

	if s.opts.Debug {
		log.Println("Initializing runtime and agents...")
	}

	rt, err := runtime.NewWithConfig(s.config)
	if err != nil {
		return fmt.Errorf("runtime initialization failed: %w", err)
	}
	s.runtime = rt

	if s.config.Global.Observability.Metrics.Enabled || s.config.Global.Observability.Tracing.Enabled {
		log.Println("🔭 Initializing observability...")

		obsConfig := observability.Config{
			Tracing: observability.TracerConfig{
				Enabled:      s.config.Global.Observability.Tracing.Enabled,
				ExporterType: s.config.Global.Observability.Tracing.ExporterType,
				EndpointURL:  s.config.Global.Observability.Tracing.EndpointURL,
				SamplingRate: s.config.Global.Observability.Tracing.SamplingRate,
				ServiceName:  s.config.Global.Observability.Tracing.ServiceName,
			},
			Metrics: observability.MetricsConfig{
				Enabled: s.config.Global.Observability.Metrics.Enabled,
				Port:    s.config.Global.Observability.Metrics.Port,
			},
		}

		obsMgr := observability.NewManager(obsConfig)
		if err := obsMgr.Initialize(context.Background()); err != nil {
			log.Printf("⚠️  Failed to initialize observability: %v", err)
		} else {
			s.observability = obsMgr
			if obsConfig.Tracing.Enabled {
				log.Printf("  ✅ Tracing enabled (endpoint: %s, sampling: %.2f)",
					obsConfig.Tracing.EndpointURL, obsConfig.Tracing.SamplingRate)
			}
			if obsConfig.Metrics.Enabled {
				log.Printf("  ✅ Metrics enabled")
			}
		}
	}

	agentRegistry := rt.Registry()
	for _, agentID := range agentRegistry.ListAgents() {
		entry, _ := agentRegistry.Get(agentID)

		if entry.Config.Type == "a2a" {
			log.Printf("External agent '%s' connected to %s", agentID, entry.Config.URL)
		} else {
			log.Printf("Native agent '%s' created", agentID)
		}
	}

	if agentRegistry.ListAgents() == nil || len(agentRegistry.ListAgents()) == 0 {
		return fmt.Errorf("no agents successfully registered")
	}

	return nil
}

func (s *Server) startTransport() error {
	agentRegistry := s.runtime.Registry()
	agentRouter := agent.NewAgentRouter(agentRegistry)

	addresses := s.resolveAddresses()

	authConfig, err := s.setupAuth()
	if err != nil {
		return fmt.Errorf("auth setup failed: %w", err)
	}

	grpcConfig := transport.Config{
		Address: addresses.GRPC,
	}
	if authConfig != nil && authConfig.Validator != nil {
		grpcConfig.UnaryInterceptor = authConfig.Validator.UnaryServerInterceptor()
		grpcConfig.StreamInterceptor = authConfig.Validator.StreamServerInterceptor()
	}

	s.grpcServer = transport.NewRegistryServer(agentRegistry, grpcConfig)

	s.restGateway = transport.NewRESTGateway(transport.RESTGatewayConfig{
		HTTPAddress: addresses.REST,
		GRPCAddress: addresses.GRPC,
	})
	if authConfig != nil {
		s.restGateway.SetAuth(authConfig)
	}

	discovery := transport.NewAgentDiscovery(agentRouter, authConfig)
	s.restGateway.SetDiscovery(discovery)

	s.jsonrpcServer = transport.NewJSONRPCHandler(
		transport.JSONRPCConfig{HTTPAddress: addresses.JSONRPC},
		agentRouter,
	)
	if authConfig != nil {
		s.jsonrpcServer.SetAuth(authConfig)
	}

	errChan := make(chan error, 3)

	go func() {
		if err := s.grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		if err := s.restGateway.Start(context.Background()); err != nil {
			errChan <- fmt.Errorf("REST gateway error: %w", err)
		}
	}()

	go func() {
		if err := s.jsonrpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("JSON-RPC handler error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(500 * time.Millisecond):

		return nil
	}
}

func (s *Server) runLifecycle() {
	defer close(s.doneChan)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			s.cleanup(context.Background())
			return

		case <-s.stopChan:
			log.Println("Stop requested...")
			s.cleanup(context.Background())
			return

		case <-s.reloadChan:
			log.Println("🔄 Configuration reload requested...")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			s.cleanup(shutdownCtx)
			cancel()

			if err := s.initialize(); err != nil {
				log.Printf("❌ Failed to reinitialize after reload: %v", err)
				return
			}

			if err := s.startTransport(); err != nil {
				log.Printf("❌ Failed to start transport after reload: %v", err)
				return
			}

			s.logStartup()
			log.Println("✅ Server reloaded successfully")
		}
	}
}

func (s *Server) cleanup(ctx context.Context) {
	var shutdownErrors []error

	if s.grpcServer != nil {
		if err := s.grpcServer.Stop(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("gRPC: %w", err))
		}
	}
	if s.restGateway != nil {
		if err := s.restGateway.Stop(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("REST: %w", err))
		}
	}
	if s.jsonrpcServer != nil {
		if err := s.jsonrpcServer.Stop(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("JSON-RPC: %w", err))
		}
	}

	if s.observability != nil {
		if err := s.observability.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("observability: %w", err))
		}
	}

	if s.runtime != nil {
		if err := s.runtime.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("runtime: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		for _, err := range shutdownErrors {
			log.Printf("Shutdown error: %v", err)
		}
	}
}

func (s *Server) resolveAddresses() ServerAddresses {
	basePort := 8080
	serverHost := "0.0.0.0"
	baseURL := ""

	if s.config.Global.A2AServer.IsEnabled() {
		if s.config.Global.A2AServer.Port > 0 {
			basePort = s.config.Global.A2AServer.Port
		}
		if s.config.Global.A2AServer.Host != "" {
			serverHost = s.config.Global.A2AServer.Host
		}
		if s.config.Global.A2AServer.BaseURL != "" {
			baseURL = s.config.Global.A2AServer.BaseURL
		}
	}

	if s.opts.Port != 8080 && s.opts.Port != 0 {
		basePort = s.opts.Port
	}
	if s.opts.Host != "" {
		serverHost = s.opts.Host
	}
	if s.opts.BaseURL != "" {
		baseURL = s.opts.BaseURL
	}

	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", serverHost, basePort+1)
	}

	return ServerAddresses{
		GRPC:    fmt.Sprintf("%s:%d", serverHost, basePort),
		REST:    fmt.Sprintf("%s:%d", serverHost, basePort+1),
		JSONRPC: fmt.Sprintf("%s:%d", serverHost, basePort+2),
		BaseURL: baseURL,
		Host:    serverHost,
		Port:    basePort,
	}
}

func (s *Server) setupAuth() (*transport.AuthConfig, error) {
	if !s.config.Global.Auth.IsEnabled() {
		return nil, nil
	}

	jwtValidator, err := auth.NewJWTValidator(
		s.config.Global.Auth.JWKSURL,
		s.config.Global.Auth.Issuer,
		s.config.Global.Auth.Audience,
	)
	if err != nil {
		return nil, fmt.Errorf("JWT validator creation failed: %w", err)
	}

	log.Printf("✅ Authentication configured (JWT)")

	return &transport.AuthConfig{
		Enabled:   true,
		Validator: jwtValidator,
	}, nil
}

func (s *Server) logStartup() {
	addresses := s.resolveAddresses()
	agentCount := len(s.runtime.Registry().ListAgents())

	log.Printf("Server started with %d agents", agentCount)
	log.Printf("gRPC: %s", addresses.GRPC)
	log.Printf("REST: http://%s", addresses.REST)
	log.Printf("JSON-RPC: http://%s/rpc", addresses.JSONRPC)

	if s.opts.Debug {
		log.Printf("API endpoints: %s/v1/agents/{name}/message:send", addresses.BaseURL)
	}

	log.Println("Press Ctrl+C to stop")
}

type ServerAddresses struct {
	GRPC    string
	REST    string
	JSONRPC string
	BaseURL string
	Host    string
	Port    int
}
