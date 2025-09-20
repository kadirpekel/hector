package llms

// ============================================================================
// LLM PROVIDER INTERFACE
// ============================================================================

// LLMProvider interface for language model generation
type LLMProvider interface {
	// Generate generates a response given a pre-built prompt
	Generate(prompt string) (string, int, error)

	// GenerateStreaming generates a streaming response given a pre-built prompt
	GenerateStreaming(prompt string) (<-chan string, error)

	// GetModelName returns the model name
	GetModelName() string

	// GetMaxTokens returns the maximum tokens for generation
	GetMaxTokens() int

	// GetTemperature returns the temperature setting
	GetTemperature() float64

	// Close closes the provider
	Close() error
}

// ============================================================================
// CONVENIENT REEXPORTS
// ============================================================================

// This file provides convenient reexports for all LLM implementations.
// All LLM types and functions are available directly from the llms package.

// Reexport LLM provider implementations
// Note: These are now direct function references since the functions are public
