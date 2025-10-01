package reasoning

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================================
// DEFAULT REASONING ENGINE - CLEAN IMPLEMENTATION
// ============================================================================

// DefaultReasoningEngine implements a clean, service-oriented reasoning approach
// using dependency injection for loose coupling and easy testing
type DefaultReasoningEngine struct {
	services AgentServices
}

// NewDefaultReasoningEngine creates a new default reasoning engine
func NewDefaultReasoningEngine(services AgentServices) *DefaultReasoningEngine {
	return &DefaultReasoningEngine{
		services: services,
	}
}

// ============================================================================
// REASONING ENGINE INTERFACE IMPLEMENTATION
// ============================================================================

func (e *DefaultReasoningEngine) Execute(ctx context.Context, query string) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		// Add user query to history
		e.services.History().AddToHistory("user", query, nil)

		// Build initial prompt
		prompt, err := e.services.Prompt().BuildDefaultPrompt(ctx, query, e.services.Context(), e.services.History(), e.services.Extensions())
		if err != nil {
			outputCh <- fmt.Sprintf("Error building prompt: %v", err)
			return
		}

		// Generate initial LLM response
		var responseBuilder strings.Builder

		config := e.services.GetConfig()
		if config.EnableStreaming {
			// LLM supports streaming - stream each chunk as it comes
			streamCh, err := e.services.LLM().GenerateLLMStreaming(prompt)
			if err != nil {
				outputCh <- fmt.Sprintf("Error generating streaming response: %v", err)
				return
			}

			for chunk := range streamCh {
				outputCh <- chunk
				responseBuilder.WriteString(chunk)
			}
		} else {
			// LLM doesn't support streaming - get masked response and output as single chunk
			maskedResponse, _, err := e.services.LLM().GenerateLLM(prompt)
			if err != nil {
				outputCh <- fmt.Sprintf("Error generating response: %v", err)
				return
			}

			outputCh <- maskedResponse
			responseBuilder.WriteString(maskedResponse)
		}

		// Execute extensions if any were found in the response
		llmService := e.services.LLM()
		if llmService != nil {
			extensionCalls := llmService.GetExtensionCalls()
			if len(extensionCalls) > 0 {
				// Execute extensions
				extensionResults, err := e.services.Extensions().ExecuteExtensions(ctx, extensionCalls)
				if err != nil {
					outputCh <- fmt.Sprintf("\nError executing extensions: %v", err)
					return
				}

				// Handle extension results based on display_direct preference
				if len(extensionResults) > 0 {
					directResults := make(map[string]ExtensionResult)
					llmResults := make(map[string]ExtensionResult)

					// Separate results based on display_direct metadata
					for name, result := range extensionResults {
						if result.Metadata != nil {
							if displayDirect, ok := result.Metadata["display_direct"].(bool); ok && displayDirect {
								directResults[name] = result
							} else {
								llmResults[name] = result
							}
						} else {
							// Default to LLM processing if no metadata
							llmResults[name] = result
						}
					}

					// Display direct results immediately
					if len(directResults) > 0 {
						outputCh <- "\n\n"
						for name, result := range directResults {
							if result.Success {
								outputCh <- fmt.Sprintf("📋 %s Results:\n%s\n\n", name, result.Content)
							} else {
								outputCh <- fmt.Sprintf("❌ %s failed: %s\n\n", name, result.Error)
							}
						}
					}

					// Only process LLM results if there are any that need analysis
					if len(llmResults) > 0 {
						// Build follow-up prompt with just the results, not the original query
						followUpPrompt, err := e.services.Prompt().BuildDefaultPrompt(ctx, "Please analyze and summarize the following results:", e.services.Context(), e.services.History(), e.services.Extensions(), llmResults)
						if err != nil {
							outputCh <- fmt.Sprintf("\nError building follow-up prompt: %v", err)
							return
						}

						// Generate final response with results that need analysis
						if config.EnableStreaming {
							followUpStreamCh, err := e.services.LLM().GenerateLLMStreaming(followUpPrompt)
							if err != nil {
								outputCh <- fmt.Sprintf("\nError generating follow-up response: %v", err)
								return
							}

							if len(directResults) == 0 {
								outputCh <- "\n\n" // Only add spacing if we haven't displayed direct results
							}
							for chunk := range followUpStreamCh {
								outputCh <- chunk
								responseBuilder.WriteString(chunk)
							}
						} else {
							followUpResponse, _, err := e.services.LLM().GenerateLLM(followUpPrompt)
							if err != nil {
								outputCh <- fmt.Sprintf("\nError generating follow-up response: %v", err)
								return
							}

							if len(directResults) == 0 {
								outputCh <- "\n\n" // Only add spacing if we haven't displayed direct results
							}
							outputCh <- followUpResponse
							responseBuilder.WriteString(followUpResponse)
						}
					}
				}
			}
		}

		// Add complete assistant response to history
		e.services.History().AddToHistory("assistant", responseBuilder.String(), nil)
	}()

	return outputCh, nil
}

// GetName returns the name of the reasoning engine
func (e *DefaultReasoningEngine) GetName() string {
	return "default"
}

// GetDescription returns a description of the reasoning engine
func (e *DefaultReasoningEngine) GetDescription() string {
	return "A clean default reasoning engine that uses agent services for all operations"
}

// Tools are now handled through extensions - no separate tool result processing needed
