package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

// CohereEmbedder implements EmbedderProvider for Cohere embeddings API
type CohereEmbedder struct {
	config    *config.EmbedderProviderConfig
	client    *http.Client
	apiKey    string
	baseURL   string
	model     string
	dimension int
	batchSize int
}

// CohereEmbedRequest represents the request payload for Cohere embeddings API
type CohereEmbedRequest struct {
	Texts     []string `json:"texts"`
	Model     string   `json:"model,omitempty"`
	InputType string   `json:"input_type,omitempty"` // "search_document", "search_query", "classification", "clustering"
	Truncate  string   `json:"truncate,omitempty"`   // "NONE", "START", "END"
}

// CohereEmbedResponse represents the response from Cohere embeddings API
type CohereEmbedResponse struct {
	ID         string      `json:"id"`
	Texts      []string    `json:"texts"`
	Embeddings [][]float32 `json:"embeddings"`
	Meta       struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
	} `json:"meta"`
}

// CohereErrorResponse represents an error response from Cohere API
type CohereErrorResponse struct {
	Message string `json:"message"`
}

func NewCohereEmbedderFromConfig(cfg *config.EmbedderProviderConfig) (*CohereEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Cohere embedder")
	}

	model := cfg.Model
	if model == "" {
		model = "embed-english-v3.0" // Default model
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Default dimensions for common models
		switch model {
		case "embed-english-v3.0":
			dimension = 1024
		case "embed-multilingual-v3.0":
			dimension = 1024
		case "embed-english-light-v3.0":
			dimension = 384
		case "embed-multilingual-light-v3.0":
			dimension = 384
		default:
			dimension = 1024
		}
	}

	baseURL := cfg.Host
	if baseURL == "" {
		baseURL = "https://api.cohere.ai/v1"
	}

	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	batchSize := 96 // Cohere's default batch size
	if cfg.BatchSize > 0 {
		batchSize = cfg.BatchSize
	}

	return &CohereEmbedder{
		config:    cfg,
		client:    &http.Client{Timeout: timeout},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		batchSize: batchSize,
	}, nil
}

func (e *CohereEmbedder) Embed(text string) ([]float32, error) {
	return e.EmbedWithContext(context.Background(), text)
}

func (e *CohereEmbedder) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	req := CohereEmbedRequest{
		Texts: []string{text},
		Model: e.model,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	httpReq.Header.Set("Accept", "application/json")

	var resp *http.Response
	maxRetries := e.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = e.client.Do(httpReq)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries-1 {
			// Exponential backoff
			backoff := time.Duration(attempt+1) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request to Cohere: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp CohereErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Cohere API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("Cohere API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response CohereEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("received empty embedding from Cohere")
	}

	return response.Embeddings[0], nil
}

func (e *CohereEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	return e.EmbedBatchWithContext(context.Background(), texts)
}

func (e *CohereEmbedder) EmbedBatchWithContext(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Process in batches
	results := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		req := CohereEmbedRequest{
			Texts: batch,
			Model: e.model,
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embed", bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
		httpReq.Header.Set("Accept", "application/json")

		resp, err := e.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResp CohereErrorResponse
			if err := json.Unmarshal(body, &errorResp); err == nil {
				return nil, fmt.Errorf("Cohere API error: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("Cohere API returned status %d: %s", resp.StatusCode, string(body))
		}

		var response CohereEmbedResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		results = append(results, response.Embeddings...)
	}

	return results, nil
}

func (e *CohereEmbedder) GetDimension() int {
	return e.dimension
}

func (e *CohereEmbedder) GetModelName() string {
	return e.model
}

func (e *CohereEmbedder) Close() error {
	return nil
}
