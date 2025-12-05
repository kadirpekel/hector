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
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Global mutex to serialize Ollama embedding requests.
// Ollama's llama runner can crash with concurrent embedding requests.
var ollamaEmbedMu sync.Mutex

// OllamaEmbedder implements Embedder using Ollama's embeddings API.
//
// Ported from legacy pkg/embedders/ollama.go.
type OllamaEmbedder struct {
	client    *http.Client
	baseURL   string
	model     string
	dimension int
}

// OllamaConfig configures the Ollama embedder.
type OllamaConfig struct {
	// BaseURL for Ollama API (default: http://localhost:11434).
	BaseURL string

	// Model name (default: nomic-embed-text).
	Model string

	// Dimension of embeddings (default: 768 for nomic-embed-text).
	Dimension int

	// Timeout for API requests (default: 30s).
	Timeout time.Duration
}

// ollamaRequest represents the request payload for Ollama embeddings API.
// See: https://docs.ollama.com/api/embeddings
// Supports both single string and array of strings for batch processing.
type ollamaRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"` // string or []string
}

// ollamaResponse represents the response from Ollama embeddings API.
// Returns L2-normalized (unit-length) vectors.
type ollamaResponse struct {
	Embeddings [][]float32 `json:"embeddings"` // Array of embeddings (plural)
}

// NewOllamaEmbedder creates a new Ollama embedder.
func NewOllamaEmbedder(cfg OllamaConfig) (*OllamaEmbedder, error) {
	model := cfg.Model
	if model == "" {
		model = "nomic-embed-text"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Default dimensions for common models
		switch model {
		case "nomic-embed-text", "nomic-embed-text-v2":
			dimension = 768
		case "all-minilm:l6-v2":
			dimension = 384
		case "bge-small-en-v1.5":
			dimension = 384
		case "bge-large-en-v1.5":
			dimension = 1024
		default:
			dimension = 768
		}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OllamaEmbedder{
		client:    &http.Client{Timeout: timeout},
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
	}, nil
}

// Embed converts text to a vector embedding.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("received empty embedding from Ollama")
	}
	return embeddings[0], nil
}

// EmbedBatch converts multiple texts to vector embeddings.
// Ollama API supports batch processing via array input.
func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Serialize all Ollama embedding requests to prevent crashes
	ollamaEmbedMu.Lock()
	defer ollamaEmbedMu.Unlock()

	slog.Debug("Ollama embedding batch request", "model", e.model, "count", len(texts))

	// Use array input for batch processing (Ollama API supports this)
	var input interface{} = texts
	if len(texts) == 1 {
		// Single string for single input
		input = texts[0]
	}

	req := ollamaRequest{
		Model: e.model,
		Input: input,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		slog.Error("Ollama embedding failed", "error", err, "model", e.model)
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("received empty embeddings from Ollama")
	}

	return response.Embeddings, nil
}

// Dimension returns the embedding vector dimension.
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name being used.
func (e *OllamaEmbedder) Model() string {
	return e.model
}

// Close releases any resources.
func (e *OllamaEmbedder) Close() error {
	return nil
}

// Ensure OllamaEmbedder implements Embedder.
var _ Embedder = (*OllamaEmbedder)(nil)
