package moderation

import (
	"context"
)

// Provider is the interface for content moderation services.
// Implementations can use external APIs (OpenAI, Lakera), internal LLMs,
// or agent-based evaluation.
type Provider interface {
	// Moderate checks content for policy violations.
	// Returns a Result indicating whether the content is safe.
	Moderate(ctx context.Context, content string) (*Result, error)

	// Name returns the provider name for logging/debugging.
	Name() string
}

// Result represents the outcome of content moderation.
type Result struct {
	// Safe indicates whether the content passed moderation.
	Safe bool

	// Category identifies the violation type (if any).
	// Examples: "hate", "violence", "self-harm", "sexual", "prompt_injection"
	Category string

	// Confidence is the moderation confidence score (0-1).
	// Higher values indicate more confidence in the classification.
	Confidence float64

	// Reason provides a human-readable explanation.
	Reason string

	// Scores contains detailed category scores (provider-specific).
	Scores map[string]float64
}

// Strategy identifies which moderation approach to use.
type Strategy string

const (
	// StrategyNone disables LLM moderation.
	StrategyNone Strategy = "none"

	// StrategyOpenAI uses OpenAI's Moderation API.
	StrategyOpenAI Strategy = "openai"

	// StrategyLakera uses Lakera Guard API.
	StrategyLakera Strategy = "lakera"

	// StrategyPrompt uses a custom prompt with any LLM.
	StrategyPrompt Strategy = "prompt"

	// StrategyAgent uses a Hector agent for moderation.
	StrategyAgent Strategy = "agent"
)
