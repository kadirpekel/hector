package grpc

import (
	"context"
	"io"

	"github.com/hashicorp/go-plugin"
	pb "github.com/kadirpekel/hector/pkg/plugins/grpc/proto"
	"google.golang.org/grpc"
)

// ============================================================================
// LLM PROVIDER PLUGIN
// ============================================================================

// LLMProviderPlugin is the plugin.Plugin implementation for LLM providers
type LLMProviderPlugin struct {
	plugin.Plugin
	Impl LLMProvider
}

// GRPCServer registers the LLM provider gRPC server
func (p *LLMProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterLLMProviderServer(s, &LLMProviderGRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the LLM provider gRPC client
func (p *LLMProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &LLMProviderGRPCClient{
		client: pb.NewLLMProviderClient(c),
	}, nil
}

// ============================================================================
// LLM GRPC SERVER (for plugin implementations)
// ============================================================================

// LLMProviderGRPCServer is the gRPC server that the plugin implements
type LLMProviderGRPCServer struct {
	pb.UnimplementedLLMProviderServer
	Impl LLMProvider
}

func (s *LLMProviderGRPCServer) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
	err := s.Impl.Initialize(ctx, req.Config)
	if err != nil {
		return &pb.InitializeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.InitializeResponse{Success: true}, nil
}

func (s *LLMProviderGRPCServer) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	err := s.Impl.Shutdown(ctx)
	if err != nil {
		return &pb.ShutdownResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.ShutdownResponse{Success: true}, nil
}

func (s *LLMProviderGRPCServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	err := s.Impl.Health(ctx)
	if err != nil {
		return &pb.HealthResponse{
			Healthy: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.HealthResponse{Healthy: true}, nil
}

func (s *LLMProviderGRPCServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	// Manifest is handled by the plugin registry, not the implementation
	return &pb.ManifestResponse{}, nil
}

func (s *LLMProviderGRPCServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	// Status is handled by the plugin registry, not the implementation
	return &pb.StatusResponse{Status: "ready"}, nil
}

func (s *LLMProviderGRPCServer) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
	return s.Impl.Generate(ctx, req.Messages, req.Tools)
}

func (s *LLMProviderGRPCServer) GenerateStreaming(req *pb.GenerateRequest, stream pb.LLMProvider_GenerateStreamingServer) error {
	ctx := stream.Context()
	chunks, err := s.Impl.GenerateStreaming(ctx, req.Messages, req.Tools)
	if err != nil {
		return err
	}

	for chunk := range chunks {
		if err := stream.Send(chunk); err != nil {
			return err
		}
	}
	return nil
}

func (s *LLMProviderGRPCServer) GetModelInfo(ctx context.Context, req *pb.Empty) (*pb.ModelInfo, error) {
	return s.Impl.GetModelInfo(ctx)
}

// ============================================================================
// LLM GRPC CLIENT (Hector uses this to talk to plugins)
// ============================================================================

// LLMProviderGRPCClient is the gRPC client implementation
type LLMProviderGRPCClient struct {
	client pb.LLMProviderClient
}

func (c *LLMProviderGRPCClient) Initialize(ctx context.Context, config map[string]string) error {
	resp, err := c.client.Initialize(ctx, &pb.InitializeRequest{Config: config})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *LLMProviderGRPCClient) Generate(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (*pb.GenerateResponse, error) {
	return c.client.Generate(ctx, &pb.GenerateRequest{
		Messages: messages,
		Tools:    tools,
	})
}

func (c *LLMProviderGRPCClient) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (<-chan *pb.StreamChunk, error) {
	stream, err := c.client.GenerateStreaming(ctx, &pb.GenerateRequest{
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return nil, err
	}

	chunks := make(chan *pb.StreamChunk, 100)
	go func() {
		defer close(chunks)
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				// Send error chunk
				chunks <- &pb.StreamChunk{
					Type:  pb.StreamChunk_ERROR,
					Error: err.Error(),
				}
				return
			}
			chunks <- chunk
		}
	}()

	return chunks, nil
}

func (c *LLMProviderGRPCClient) GetModelInfo(ctx context.Context) (*pb.ModelInfo, error) {
	return c.client.GetModelInfo(ctx, &pb.Empty{})
}

func (c *LLMProviderGRPCClient) Shutdown(ctx context.Context) error {
	resp, err := c.client.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *LLMProviderGRPCClient) Health(ctx context.Context) error {
	resp, err := c.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.Healthy {
		return &PluginError{Message: resp.Message}
	}
	return nil
}

// ============================================================================
// DATABASE PROVIDER PLUGIN
// ============================================================================

// DatabaseProviderPlugin is the plugin.Plugin implementation for Database providers
type DatabaseProviderPlugin struct {
	plugin.Plugin
	Impl DatabaseProvider
}

// GRPCServer registers the Database provider gRPC server
func (p *DatabaseProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterDatabaseProviderServer(s, &DatabaseProviderGRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the Database provider gRPC client
func (p *DatabaseProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &DatabaseProviderGRPCClient{
		client: pb.NewDatabaseProviderClient(c),
	}, nil
}

// ============================================================================
// DATABASE GRPC SERVER
// ============================================================================

type DatabaseProviderGRPCServer struct {
	pb.UnimplementedDatabaseProviderServer
	Impl DatabaseProvider
}

func (s *DatabaseProviderGRPCServer) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
	err := s.Impl.Initialize(ctx, req.Config)
	if err != nil {
		return &pb.InitializeResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.InitializeResponse{Success: true}, nil
}

func (s *DatabaseProviderGRPCServer) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	err := s.Impl.Shutdown(ctx)
	if err != nil {
		return &pb.ShutdownResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.ShutdownResponse{Success: true}, nil
}

func (s *DatabaseProviderGRPCServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	err := s.Impl.Health(ctx)
	if err != nil {
		return &pb.HealthResponse{Healthy: false, Message: err.Error()}, nil
	}
	return &pb.HealthResponse{Healthy: true}, nil
}

func (s *DatabaseProviderGRPCServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	return &pb.ManifestResponse{}, nil
}

func (s *DatabaseProviderGRPCServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Status: "ready"}, nil
}

func (s *DatabaseProviderGRPCServer) Upsert(ctx context.Context, req *pb.UpsertRequest) (*pb.UpsertResponse, error) {
	err := s.Impl.Upsert(ctx, req.Collection, req.Id, req.Vector, req.Metadata)
	if err != nil {
		return &pb.UpsertResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.UpsertResponse{Success: true}, nil
}

func (s *DatabaseProviderGRPCServer) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	results, err := s.Impl.Search(ctx, req.Collection, req.Vector, req.TopK)
	if err != nil {
		return &pb.SearchResponse{Error: err.Error()}, nil
	}
	return &pb.SearchResponse{Results: results}, nil
}

func (s *DatabaseProviderGRPCServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.Impl.Delete(ctx, req.Collection, req.Id)
	if err != nil {
		return &pb.DeleteResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.DeleteResponse{Success: true}, nil
}

func (s *DatabaseProviderGRPCServer) CreateCollection(ctx context.Context, req *pb.CreateCollectionRequest) (*pb.CreateCollectionResponse, error) {
	err := s.Impl.CreateCollection(ctx, req.Collection, req.VectorSize)
	if err != nil {
		return &pb.CreateCollectionResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.CreateCollectionResponse{Success: true}, nil
}

func (s *DatabaseProviderGRPCServer) DeleteCollection(ctx context.Context, req *pb.DeleteCollectionRequest) (*pb.DeleteCollectionResponse, error) {
	err := s.Impl.DeleteCollection(ctx, req.Collection)
	if err != nil {
		return &pb.DeleteCollectionResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.DeleteCollectionResponse{Success: true}, nil
}

// ============================================================================
// DATABASE GRPC CLIENT
// ============================================================================

type DatabaseProviderGRPCClient struct {
	client pb.DatabaseProviderClient
}

func (c *DatabaseProviderGRPCClient) Initialize(ctx context.Context, config map[string]string) error {
	resp, err := c.client.Initialize(ctx, &pb.InitializeRequest{Config: config})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]string) error {
	resp, err := c.client.Upsert(ctx, &pb.UpsertRequest{
		Collection: collection,
		Id:         id,
		Vector:     vector,
		Metadata:   metadata,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) Search(ctx context.Context, collection string, vector []float32, topK int32) ([]*pb.SearchResult, error) {
	resp, err := c.client.Search(ctx, &pb.SearchRequest{
		Collection: collection,
		Vector:     vector,
		TopK:       topK,
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &PluginError{Message: resp.Error}
	}
	return resp.Results, nil
}

func (c *DatabaseProviderGRPCClient) Delete(ctx context.Context, collection string, id string) error {
	resp, err := c.client.Delete(ctx, &pb.DeleteRequest{
		Collection: collection,
		Id:         id,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	resp, err := c.client.CreateCollection(ctx, &pb.CreateCollectionRequest{
		Collection: collection,
		VectorSize: vectorSize,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) DeleteCollection(ctx context.Context, collection string) error {
	resp, err := c.client.DeleteCollection(ctx, &pb.DeleteCollectionRequest{
		Collection: collection,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) Shutdown(ctx context.Context) error {
	resp, err := c.client.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *DatabaseProviderGRPCClient) Health(ctx context.Context) error {
	resp, err := c.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.Healthy {
		return &PluginError{Message: resp.Message}
	}
	return nil
}

// ============================================================================
// EMBEDDER PROVIDER PLUGIN
// ============================================================================

// EmbedderProviderPlugin is the plugin.Plugin implementation for Embedder providers
type EmbedderProviderPlugin struct {
	plugin.Plugin
	Impl EmbedderProvider
}

// GRPCServer registers the Embedder provider gRPC server
func (p *EmbedderProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterEmbedderProviderServer(s, &EmbedderProviderGRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the Embedder provider gRPC client
func (p *EmbedderProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &EmbedderProviderGRPCClient{
		client: pb.NewEmbedderProviderClient(c),
	}, nil
}

// ============================================================================
// EMBEDDER GRPC SERVER
// ============================================================================

type EmbedderProviderGRPCServer struct {
	pb.UnimplementedEmbedderProviderServer
	Impl EmbedderProvider
}

func (s *EmbedderProviderGRPCServer) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
	err := s.Impl.Initialize(ctx, req.Config)
	if err != nil {
		return &pb.InitializeResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.InitializeResponse{Success: true}, nil
}

func (s *EmbedderProviderGRPCServer) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	err := s.Impl.Shutdown(ctx)
	if err != nil {
		return &pb.ShutdownResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.ShutdownResponse{Success: true}, nil
}

func (s *EmbedderProviderGRPCServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	err := s.Impl.Health(ctx)
	if err != nil {
		return &pb.HealthResponse{Healthy: false, Message: err.Error()}, nil
	}
	return &pb.HealthResponse{Healthy: true}, nil
}

func (s *EmbedderProviderGRPCServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	return &pb.ManifestResponse{}, nil
}

func (s *EmbedderProviderGRPCServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Status: "ready"}, nil
}

func (s *EmbedderProviderGRPCServer) Embed(ctx context.Context, req *pb.EmbedRequest) (*pb.EmbedResponse, error) {
	vector, err := s.Impl.Embed(ctx, req.Text)
	if err != nil {
		return &pb.EmbedResponse{Error: err.Error()}, nil
	}
	return &pb.EmbedResponse{Vector: vector}, nil
}

func (s *EmbedderProviderGRPCServer) GetEmbedderInfo(ctx context.Context, req *pb.Empty) (*pb.EmbedderInfo, error) {
	return s.Impl.GetEmbedderInfo(ctx)
}

// ============================================================================
// EMBEDDER GRPC CLIENT
// ============================================================================

type EmbedderProviderGRPCClient struct {
	client pb.EmbedderProviderClient
}

func (c *EmbedderProviderGRPCClient) Initialize(ctx context.Context, config map[string]string) error {
	resp, err := c.client.Initialize(ctx, &pb.InitializeRequest{Config: config})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *EmbedderProviderGRPCClient) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.client.Embed(ctx, &pb.EmbedRequest{Text: text})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &PluginError{Message: resp.Error}
	}
	return resp.Vector, nil
}

func (c *EmbedderProviderGRPCClient) GetEmbedderInfo(ctx context.Context) (*pb.EmbedderInfo, error) {
	return c.client.GetEmbedderInfo(ctx, &pb.Empty{})
}

func (c *EmbedderProviderGRPCClient) Shutdown(ctx context.Context) error {
	resp, err := c.client.Shutdown(ctx, &pb.ShutdownRequest{})
	if err != nil {
		return err
	}
	if !resp.Success {
		return &PluginError{Message: resp.Error}
	}
	return nil
}

func (c *EmbedderProviderGRPCClient) Health(ctx context.Context) error {
	resp, err := c.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return err
	}
	if !resp.Healthy {
		return &PluginError{Message: resp.Message}
	}
	return nil
}

// ============================================================================
// COMMON ERROR TYPE
// ============================================================================

// PluginError represents an error from a plugin
type PluginError struct {
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}
