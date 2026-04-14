package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIConfig configures the OpenAI moderation provider.
type OpenAIConfig struct {
	// APIKey is the OpenAI API key.
	// Uses OPENAI_API_KEY environment variable if empty.
	APIKey string

	// Model is the moderation model to use.
	// Default: "omni-moderation-latest"
	Model string

	// Threshold is the score above which content is flagged (0-1).
	// Default: 0.8
	Threshold float64

	// Endpoint is the API endpoint URL.
	// Default: https://api.openai.com/v1/moderations
	Endpoint string

	// Timeout is the request timeout.
	// Default: 10s
	Timeout time.Duration
}

// openaiProvider implements Provider using OpenAI's Moderation API.
type openaiProvider struct {
	apiKey    string
	model     string
	threshold float64
	endpoint  string
	client    *http.Client
}

// NewOpenAI creates an OpenAI moderation provider.
// The OpenAI Moderation API is free to use for content moderation.
// Uses omni-moderation-latest by default which supports text and images.
func NewOpenAI(cfg OpenAIConfig) Provider {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	model := cfg.Model
	if model == "" {
		model = "omni-moderation-latest"
	}

	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = 0.8
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/moderations"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &openaiProvider{
		apiKey:    apiKey,
		model:     model,
		threshold: threshold,
		endpoint:  endpoint,
		client:    &http.Client{Timeout: timeout},
	}
}

func (p *openaiProvider) Name() string {
	return "openai"
}

func (p *openaiProvider) Moderate(ctx context.Context, content string) (*Result, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai moderation: API key not configured")
	}

	// Build request per OpenAI API spec
	reqBody, _ := json.Marshal(openaiModerationRequest{
		Model: p.model,
		Input: content,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("openai moderation: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai moderation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai moderation: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result openaiModerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai moderation: parse response: %w", err)
	}

	if len(result.Results) == 0 {
		return &Result{Safe: true}, nil
	}

	modResult := result.Results[0]

	// Find highest scoring category
	var maxScore float64
	var maxCategory string
	for category, score := range modResult.CategoryScores {
		if score > maxScore {
			maxScore = score
			maxCategory = category
		}
	}

	// Determine if flagged based on threshold or explicit flag
	flagged := modResult.Flagged || maxScore >= p.threshold

	return &Result{
		Safe:       !flagged,
		Category:   maxCategory,
		Confidence: maxScore,
		Reason:     fmt.Sprintf("OpenAI moderation: %s (score: %.4f)", maxCategory, maxScore),
		Scores:     modResult.CategoryScores,
	}, nil
}

// OpenAI API request/response types per official documentation
type openaiModerationRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openaiModerationResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Results []openaiModerationResult `json:"results"`
}

type openaiModerationResult struct {
	Flagged        bool               `json:"flagged"`
	Categories     map[string]bool    `json:"categories"`
	CategoryScores map[string]float64 `json:"category_scores"`
	// CategoryAppliedInputTypes is only available on omni models
	CategoryAppliedInputTypes map[string][]string `json:"category_applied_input_types,omitempty"`
}
