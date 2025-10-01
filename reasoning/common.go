package reasoning

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================================
// COMMON ORCHESTRATION FUNCTIONS
// Shared patterns for reasoning engines using services
// ============================================================================

// generateResponse handles LLM generation with automatic streaming support
// Returns the complete response for history management
func generateResponse(ctx context.Context, services AgentServices, prompt string, outputCh chan<- string) (*strings.Builder, error) {
	var response strings.Builder
	config := services.GetConfig()

	if config.EnableStreaming {
		// Stream each chunk as it arrives
		streamCh, err := services.LLM().GenerateLLMStreaming(prompt)
		if err != nil {
			return nil, fmt.Errorf("streaming generation failed: %w", err)
		}

		for chunk := range streamCh {
			outputCh <- chunk
			response.WriteString(chunk)
		}
	} else {
		// Single-shot generation
		resp, _, err := services.LLM().GenerateLLM(prompt)
		if err != nil {
			return nil, fmt.Errorf("generation failed: %w", err)
		}

		outputCh <- resp
		response.WriteString(resp)
	}

	return &response, nil
}

// executeDiscoveredExtensions checks for extension calls in last LLM response and executes them
// Returns nil if no extensions were called
func executeDiscoveredExtensions(ctx context.Context, services AgentServices, outputCh chan<- string) (map[string]ExtensionResult, error) {
	// Check if LLM called any extensions
	extensionCalls := services.LLM().GetExtensionCalls()
	if len(extensionCalls) == 0 {
		return nil, nil
	}

	// Execute all discovered extensions (no deduplication - trust the LLM)
	results, err := services.Extensions().ExecuteExtensions(ctx, extensionCalls)
	if err != nil {
		outputCh <- fmt.Sprintf("\nError executing extensions: %v\n", err)
		return nil, fmt.Errorf("extension execution failed: %w", err)
	}

	return results, nil
}

// recordUserQuery adds user query to conversation history
func recordUserQuery(services AgentServices, query string) {
	services.History().AddToHistory("user", query, nil)
}

// recordAssistantResponse adds complete assistant response to conversation history
func recordAssistantResponse(services AgentServices, response string) {
	services.History().AddToHistory("assistant", response, nil)
}

// handleExtensionError sends error message to output channel and returns error
func handleExtensionError(outputCh chan<- string, err error) error {
	outputCh <- fmt.Sprintf("\n❌ Extension error: %v\n", err)
	return err
}
