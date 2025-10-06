package grpc

import (
	"github.com/hashicorp/go-plugin"
	pb "github.com/kadirpekel/hector/pkg/plugins/grpc/proto"
)

// ============================================================================
// PLUGIN AUTHOR HELPERS
// ============================================================================
// These functions make it easier for plugin authors to create plugins

// ServeLLMPlugin starts serving an LLM provider plugin
// This is the main entry point for LLM plugin executables
//
// Example usage in your plugin's main.go:
//
//	func main() {
//	    grpc.ServeLLMPlugin(&MyLLMProvider{})
//	}
func ServeLLMPlugin(impl LLMProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeLLM): &LLMProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// ServeDatabasePlugin starts serving a Database provider plugin
// This is the main entry point for Database plugin executables
//
// Example usage in your plugin's main.go:
//
//	func main() {
//	    grpc.ServeDatabasePlugin(&MyDatabaseProvider{})
//	}
func ServeDatabasePlugin(impl DatabaseProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeDatabase): &DatabaseProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// ServeEmbedderPlugin starts serving an Embedder provider plugin
// This is the main entry point for Embedder plugin executables
//
// Example usage in your plugin's main.go:
//
//	func main() {
//	    grpc.ServeEmbedderPlugin(&MyEmbedderProvider{})
//	}
func ServeEmbedderPlugin(impl EmbedderProvider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: GetHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			string(PluginTypeEmbedder): &EmbedderProviderPlugin{Impl: impl},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// ============================================================================
// TYPE ALIASES FOR CONVENIENCE
// ============================================================================
// Export proto types for plugin authors

type (
	// Message types
	Message        = pb.Message
	ToolDefinition = pb.ToolDefinition
	ToolCall       = pb.ToolCall

	// Response types
	GenerateResponse = pb.GenerateResponse
	StreamChunk      = pb.StreamChunk
	ModelInfo        = pb.ModelInfo

	// Database types
	SearchResult = pb.SearchResult

	// Embedder types
	EmbedderInfo = pb.EmbedderInfo

	// Common types
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

// ============================================================================
// CONSTANTS
// ============================================================================

// Plugin types
const (
	PluginTypeLLM       = "llm_provider"
	PluginTypeDatabase  = "database_provider"
	PluginTypeEmbedder  = "embedder_provider"
	PluginTypeTool      = "tool_provider"
	PluginTypeReasoning = "reasoning_strategy"
)

// Stream chunk types
const (
	ChunkTypeText     = pb.StreamChunk_TEXT
	ChunkTypeToolCall = pb.StreamChunk_TOOL_CALL
	ChunkTypeDone     = pb.StreamChunk_DONE
	ChunkTypeError    = pb.StreamChunk_ERROR
)
