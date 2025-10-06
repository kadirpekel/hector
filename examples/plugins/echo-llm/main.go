package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/plugins/grpc"
)

// ============================================================================
// ECHO LLM PLUGIN - SIMPLE REFERENCE IMPLEMENTATION
// ============================================================================
// This is a simple echo plugin that demonstrates how to create a Hector plugin.
// It echoes back user messages and shows the basic structure needed for any plugin.

// EchoLLMProvider is a simple LLM provider that echoes user input
type EchoLLMProvider struct {
	prefix      string
	maxTokens   int32
	temperature float64
	callCount   int
}

// Initialize is called when the plugin starts
// This is where you would initialize your LLM client, load models, etc.
func (e *EchoLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
	// Load configuration
	if prefix, ok := config["prefix"]; ok {
		e.prefix = prefix
	} else {
		e.prefix = "Echo: "
	}

	// Parse max_tokens (with default)
	e.maxTokens = 1000

	// Parse temperature (with default)
	e.temperature = 0.7

	return nil
}

// Generate produces a non-streaming response
// This is the main method where you would call your actual LLM
func (e *EchoLLMProvider) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
	e.callCount++

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get the last user message
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	// Echo back the message
	responseText := fmt.Sprintf("%s%s (call #%d)", e.prefix, lastUserMessage, e.callCount)

	// Simulate some "thinking" about tools
	if len(tools) > 0 {
		toolNames := make([]string, len(tools))
		for i, tool := range tools {
			toolNames[i] = tool.Name
		}
		responseText += fmt.Sprintf("\n\n[Available tools: %s]", strings.Join(toolNames, ", "))
	}

	// Return the response
	return &grpc.GenerateResponse{
		Text:       responseText,
		ToolCalls:  nil,                          // This echo plugin doesn't call tools
		TokensUsed: int32(len(responseText) / 4), // Rough token estimate
	}, nil
}

// GenerateStreaming produces a streaming response
// This shows how to implement streaming for real-time output
func (e *EchoLLMProvider) GenerateStreaming(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (<-chan *grpc.StreamChunk, error) {
	e.callCount++

	// Create a buffered channel for chunks
	chunks := make(chan *grpc.StreamChunk, 10)

	go func() {
		defer close(chunks)

		// Check for context cancellation
		select {
		case <-ctx.Done():
			chunks <- &grpc.StreamChunk{
				Type:  grpc.ChunkTypeError,
				Error: ctx.Err().Error(),
			}
			return
		default:
		}

		// Get the last user message
		var lastUserMessage string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserMessage = messages[i].Content
				break
			}
		}

		// Stream the response word by word
		response := fmt.Sprintf("%s%s", e.prefix, lastUserMessage)
		words := strings.Fields(response)

		var totalTokens int32
		for _, word := range words {
			// Check for cancellation between words
			select {
			case <-ctx.Done():
				chunks <- &grpc.StreamChunk{
					Type:  grpc.ChunkTypeError,
					Error: ctx.Err().Error(),
				}
				return
			default:
			}

			// Send word chunk
			chunks <- &grpc.StreamChunk{
				Type: grpc.ChunkTypeText,
				Text: word + " ",
			}
			totalTokens++
		}

		// Send done signal with final token count
		chunks <- &grpc.StreamChunk{
			Type:       grpc.ChunkTypeDone,
			TokensUsed: totalTokens,
		}
	}()

	return chunks, nil
}

// GetModelInfo returns information about this "model"
func (e *EchoLLMProvider) GetModelInfo(ctx context.Context) (*grpc.ModelInfo, error) {
	return &grpc.ModelInfo{
		ModelName:   "echo-llm-v1.0",
		MaxTokens:   e.maxTokens,
		Temperature: e.temperature,
	}, nil
}

// Shutdown is called when the plugin is being stopped
// Clean up any resources here
func (e *EchoLLMProvider) Shutdown(ctx context.Context) error {
	return nil
}

// Health checks if the plugin is functioning correctly
// This is called periodically by Hector to ensure the plugin is alive
func (e *EchoLLMProvider) Health(ctx context.Context) error {
	// In a real plugin, you might check:
	// - API connectivity
	// - Model availability
	// - Resource usage
	// For this echo plugin, we're always healthy
	return nil
}

// ============================================================================
// MAIN - PLUGIN ENTRY POINT
// ============================================================================

func main() {
	// Create the plugin implementation
	provider := &EchoLLMProvider{}

	// Serve the plugin - this blocks until Hector stops the plugin
	grpc.ServeLLMPlugin(provider)
}
