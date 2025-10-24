package grpc

import (
	"github.com/hashicorp/go-plugin"
	pb "github.com/kadirpekel/hector/pkg/plugins/grpc/proto"
)

func ServeLLMPlugin(impl LLMProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeLLM): &LLMProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func ServeDatabasePlugin(impl DatabaseProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeDatabase): &DatabaseProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func ServeEmbedderPlugin(impl EmbedderProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeEmbedder): &EmbedderProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

type (
	Message        = pb.Message
	ToolDefinition = pb.ToolDefinition
	ToolCall       = pb.ToolCall

	GenerateResponse = pb.GenerateResponse
	StreamChunk      = pb.StreamChunk
	ModelInfo        = pb.ModelInfo

	SearchResult = pb.SearchResult

	EmbedderInfo = pb.EmbedderInfo

	Empty              = pb.Empty
	InitializeRequest  = pb.InitializeRequest
	InitializeResponse = pb.InitializeResponse
	ShutdownRequest    = pb.ShutdownRequest
	ShutdownResponse   = pb.ShutdownResponse
	HealthRequest      = pb.HealthRequest
	HealthResponse     = pb.HealthResponse
	ManifestRequest    = pb.ManifestRequest
	ManifestResponse   = pb.ManifestResponse
	StatusRequest      = pb.StatusRequest
	StatusResponse     = pb.StatusResponse
)

const (
	PluginTypeLLM            = "llm_provider"
	PluginTypeDatabase       = "database_provider"
	PluginTypeEmbedder       = "embedder_provider"
	PluginTypeTool           = "tool_provider"
	PluginTypeReasoning      = "reasoning_strategy"
	PluginTypeDocumentParser = "document_parser"
)

const (
	ChunkTypeText     = pb.StreamChunk_TEXT
	ChunkTypeToolCall = pb.StreamChunk_TOOL_CALL
	ChunkTypeDone     = pb.StreamChunk_DONE
	ChunkTypeError    = pb.StreamChunk_ERROR
)
