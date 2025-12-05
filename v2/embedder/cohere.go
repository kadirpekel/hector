// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CohereEmbedder implements Embedder using Cohere's v2 embeddings API.
//
// Ported from legacy pkg/embedders/cohere.go and updated for Cohere API v2.
// See: https://docs.cohere.com/reference/embed
type CohereEmbedder struct {
	client      *http.Client
	apiKey      string
	baseURL     string
	model       string
	dimension   int
	batchSize   int
	inputType   string // Required for v3+ models: "search_document", "search_query", "classification", "clustering"
	outputDim   *int   // Optional output dimension for v4+ models (256, 512, 1024, 1536)
	truncate    string // Optional: "NONE", "START", "END" (default: "END")
}

// CohereConfig configures the Cohere embedder.
type CohereConfig struct {
	// APIKey for Cohere API (required).
	APIKey string

	// BaseURL for the API (default: https://api.cohere.com).
	BaseURL string

	// Model name (default: embed-english-v3.0).
	// Supported: embed-english-v3.0, embed-multilingual-v3.0, embed-v4.0, etc.
	Model string

	// Dimension of embeddings (auto-detected from model if 0).
	Dimension int

	// Timeout for API requests (default: 30s).
	Timeout time.Duration

	// BatchSize for batch embedding (default: 96, Cohere's max per request).
	BatchSize int

	// InputType specifies the type of input (required for v3+ models).
	// Values: "search_document", "search_query", "classification", "clustering"
	// Default: "search_document"
	InputType string

	// OutputDimension for v4+ models (optional).
	// Values: 256, 512, 1024, 1536
	// If set, overrides model's default dimension.
	OutputDimension *int

	// Truncate specifies how to handle inputs longer than max tokens.
	// Values: "NONE", "START", "END" (default: "END")
	Truncate string
}

// cohereRequest represents the request payload for Cohere v2 embeddings API.
type cohereRequest struct {
	Texts          []string `json:"texts,omitempty"`
	Model          string   `json:"model"`
	InputType      string   `json:"input_type"`
	OutputDimension *int    `json:"output_dimension,omitempty"`
	Truncate       string   `json:"truncate,omitempty"`
	EmbeddingTypes []string `json:"embedding_types,omitempty"` // ["float"] for float embeddings
}

// cohereResponse represents the response from Cohere v2 embeddings API.
type cohereResponse struct {
	ID         string `json:"id"`
	Embeddings struct {
		Float [][]float32 `json:"float"`
	} `json:"embeddings"`
	Texts []string `json:"texts"`
	Meta  struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
	} `json:"meta"`
}

// cohereErrorResponse represents an error response from Cohere API.
type cohereErrorResponse struct {
	Message string `json:"message"`
}

// NewCohereEmbedder creates a new Cohere embedder.
func NewCohereEmbedder(cfg CohereConfig) (*CohereEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Cohere embedder")
	}

	model := cfg.Model
	if model == "" {
		model = "embed-english-v3.0"
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Default dimensions for common models
		switch model {
		case "embed-english-v3.0", "embed-multilingual-v3.0":
			dimension = 1024
		case "embed-english-light-v3.0", "embed-multilingual-light-v3.0":
			dimension = 384
		case "embed-v4.0":
			dimension = 1536 // Default for v4, can be overridden with output_dimension
		default:
			dimension = 1024
		}
	}

	// If output_dimension is set for v4+ models, use it
	if cfg.OutputDimension != nil {
		dimension = *cfg.OutputDimension
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.cohere.com"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 96 // Cohere's maximum per request
	}

	inputType := cfg.InputType
	if inputType == "" {
		inputType = "search_document" // Default for semantic search use case
	}

	truncate := cfg.Truncate
	if truncate == "" {
		truncate = "END" // Default per API spec
	}

	return &CohereEmbedder{
		client:    &http.Client{Timeout: timeout},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		batchSize: batchSize,
		inputType: inputType,
		outputDim: cfg.OutputDimension,
		truncate:  truncate,
	}, nil
}

// Embed converts text to a vector embedding.
func (e *CohereEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("received empty embedding from Cohere")
	}
	return embeddings[0], nil
}

// EmbedBatch converts multiple texts to vector embeddings.
func (e *CohereEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, 0, len(texts))

	// Process in batches (max 96 per request per Cohere API)
	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		results = append(results, embeddings...)
	}

	return results, nil
}

func (e *CohereEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	req := cohereRequest{
		Texts:          texts,
		Model:          e.model,
		InputType:      e.inputType,
		OutputDimension: e.outputDim,
		Truncate:       e.truncate,
		EmbeddingTypes: []string{"float"}, // Request float embeddings
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/v2/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Cohere: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp cohereErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Cohere API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("Cohere API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response cohereResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embeddings.Float) == 0 {
		return nil, fmt.Errorf("received empty embeddings from Cohere")
	}

	return response.Embeddings.Float, nil
}

// Dimension returns the embedding vector dimension.
func (e *CohereEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name being used.
func (e *CohereEmbedder) Model() string {
	return e.model
}

// Close releases any resources.
func (e *CohereEmbedder) Close() error {
	return nil
}

// Ensure CohereEmbedder implements Embedder.
var _ Embedder = (*CohereEmbedder)(nil)

