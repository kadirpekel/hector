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

// OpenAIEmbedder implements Embedder using OpenAI's embeddings API.
//
// Ported from legacy pkg/embedders/openai.go.
type OpenAIEmbedder struct {
	client         *http.Client
	apiKey         string
	baseURL        string
	model          string
	dimension      int
	batchSize      int
	encodingFormat string
	user           string
}

// OpenAIConfig configures the OpenAI embedder.
type OpenAIConfig struct {
	// APIKey for OpenAI API (required).
	APIKey string

	// BaseURL for the API (default: https://api.openai.com/v1).
	BaseURL string

	// Model name (default: text-embedding-3-small).
	Model string

	// Dimension of embeddings (auto-detected from model if 0).
	// For text-embedding-3 models, this maps to the 'dimensions' API parameter.
	Dimension int

	// Timeout for API requests (default: 30s).
	Timeout time.Duration

	// BatchSize for batch embedding (default: 100).
	// Note: OpenAI supports up to 2048 inputs per request, but we use 100 as default
	// to stay within token limits (300,000 tokens total per request).
	BatchSize int

	// EncodingFormat specifies the format to return embeddings in.
	// Values: "float" (default), "base64"
	EncodingFormat string

	// User is a unique identifier representing your end-user.
	// Can help OpenAI monitor and detect abuse.
	User string
}

// openaiRequest represents the request payload for OpenAI embeddings API.
// See: https://platform.openai.com/docs/api-reference/embeddings/create
type openaiRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	Dimensions     *int     `json:"dimensions,omitempty"`     // Optional for text-embedding-3+
	EncodingFormat string   `json:"encoding_format,omitempty"` // "float" (default) or "base64"
	User           string   `json:"user,omitempty"`            // Optional user identifier
}

// openaiResponse represents the response from OpenAI embeddings API.
type openaiResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// openaiErrorResponse represents an error response from OpenAI API.
type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIEmbedder creates a new OpenAI embedder.
func NewOpenAIEmbedder(cfg OpenAIConfig) (*OpenAIEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI embedder")
	}

	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Default dimensions for common models
		switch model {
		case "text-embedding-3-small":
			dimension = 1536
		case "text-embedding-3-large":
			dimension = 3072
		default:
			dimension = 1536
		}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}

	encodingFormat := cfg.EncodingFormat
	if encodingFormat == "" {
		encodingFormat = "float" // Default per API spec
	}

	return &OpenAIEmbedder{
		client:    &http.Client{Timeout: timeout},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		model:     model,
		dimension:      dimension,
		batchSize:      batchSize,
		encodingFormat: encodingFormat,
		user:           cfg.User,
	}, nil
}

// Embed converts text to a vector embedding.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("received empty embedding from OpenAI")
	}
	return embeddings[0], nil
}

// EmbedBatch converts multiple texts to vector embeddings.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	results := make([][]float32, 0, len(texts))

	// Process in batches
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

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Note: Currently only "float" encoding_format is supported.
	// Base64 encoding would require different response parsing.
	if e.encodingFormat != "" && e.encodingFormat != "float" {
		return nil, fmt.Errorf("encoding_format %q not yet supported (only 'float' is supported)", e.encodingFormat)
	}

	req := openaiRequest{
		Model:          e.model,
		Input:          texts,
		EncodingFormat: e.encodingFormat,
		User:           e.user,
	}

	// Add dimensions parameter for text-embedding-3+ models if specified.
	// This allows customizing the output dimension (e.g., 256, 512, 1024, 1536 for text-embedding-3-small).
	if e.dimension > 0 && (e.model == "text-embedding-3-small" || e.model == "text-embedding-3-large") {
		req.Dimensions = &e.dimension
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp openaiErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
				errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response openaiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Sort embeddings by index to match input order
	embeddings := make([][]float32, len(response.Data))
	for _, item := range response.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

// Dimension returns the embedding vector dimension.
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name being used.
func (e *OpenAIEmbedder) Model() string {
	return e.model
}

// Close releases any resources.
func (e *OpenAIEmbedder) Close() error {
	return nil
}

// Ensure OpenAIEmbedder implements Embedder.
var _ Embedder = (*OpenAIEmbedder)(nil)
