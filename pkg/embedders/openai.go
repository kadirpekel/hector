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

// OpenAIEmbedder implements EmbedderProvider for OpenAI embeddings API
type OpenAIEmbedder struct {
	config    *config.EmbedderProviderConfig
	client    *http.Client
	apiKey    string
	baseURL   string
	model     string
	dimension int
	batchSize int
}

// OpenAIEmbedRequest represents the request payload for OpenAI embeddings API
type OpenAIEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	User  string   `json:"user,omitempty"`
}

// OpenAIEmbedResponse represents the response from OpenAI embeddings API
type OpenAIEmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIErrorResponse represents an error response from OpenAI API
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewOpenAIEmbedderFromConfig(cfg *config.EmbedderProviderConfig) (*OpenAIEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI embedder")
	}

	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small" // Default model
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Default dimensions for common models
		switch model {
		case "text-embedding-3-small":
			dimension = 1536
		case "text-embedding-3-large":
			dimension = 3072
		case "text-embedding-ada-002":
			dimension = 1536
		default:
			dimension = 1536
		}
	}

	baseURL := cfg.Host
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	batchSize := 100
	if cfg.BatchSize > 0 {
		batchSize = cfg.BatchSize
	}

	return &OpenAIEmbedder{
		config:    cfg,
		client:    &http.Client{Timeout: timeout},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		batchSize: batchSize,
	}, nil
}

func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	return e.EmbedWithContext(context.Background(), text)
}

func (e *OpenAIEmbedder) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	req := OpenAIEmbedRequest{
		Model: e.model,
		Input: []string{text},
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
		return nil, fmt.Errorf("failed to send request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp OpenAIErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)", errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("received empty embedding from OpenAI")
	}

	return response.Data[0].Embedding, nil
}

func (e *OpenAIEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	return e.EmbedBatchWithContext(context.Background(), texts)
}

func (e *OpenAIEmbedder) EmbedBatchWithContext(ctx context.Context, texts []string) ([][]float32, error) {
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
		req := OpenAIEmbedRequest{
			Model: e.model,
			Input: batch,
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
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var errorResp OpenAIErrorResponse
			if err := json.Unmarshal(body, &errorResp); err == nil {
				return nil, fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message)
			}
			return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
		}

		var response OpenAIEmbedResponse
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

		results = append(results, embeddings...)
	}

	return results, nil
}

func (e *OpenAIEmbedder) GetDimension() int {
	return e.dimension
}

func (e *OpenAIEmbedder) GetModelName() string {
	return e.model
}

func (e *OpenAIEmbedder) Close() error {
	return nil
}
