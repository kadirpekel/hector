package grpc

import (
	"context"

	pb "github.com/kadirpekel/hector/plugins/grpc/proto"
)

// ============================================================================
// PLUGIN INTERFACES
// ============================================================================
// These interfaces are implemented by plugins and used by Hector

// LLMProvider interface for LLM plugins
type LLMProvider interface {
	Initialize(ctx context.Context, config map[string]string) error
	Generate(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (*pb.GenerateResponse, error)
	GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (<-chan *pb.StreamChunk, error)
	GetModelInfo(ctx context.Context) (*pb.ModelInfo, error)
	Shutdown(ctx context.Context) error
	Health(ctx context.Context) error
}

// DatabaseProvider interface for database plugins
type DatabaseProvider interface {
	Initialize(ctx context.Context, config map[string]string) error
	Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]string) error
	Search(ctx context.Context, collection string, vector []float32, topK int32) ([]*pb.SearchResult, error)
	Delete(ctx context.Context, collection string, id string) error
	CreateCollection(ctx context.Context, collection string, vectorSize uint64) error
	DeleteCollection(ctx context.Context, collection string) error
	Shutdown(ctx context.Context) error
	Health(ctx context.Context) error
}

// EmbedderProvider interface for embedder plugins
type EmbedderProvider interface {
	Initialize(ctx context.Context, config map[string]string) error
	Embed(ctx context.Context, text string) ([]float32, error)
	GetEmbedderInfo(ctx context.Context) (*pb.EmbedderInfo, error)
	Shutdown(ctx context.Context) error
	Health(ctx context.Context) error
}
