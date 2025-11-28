package embedders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/ollama"
)

// Global mutex to serialize Ollama embedding requests
// Ollama's llama runner crashes when receiving concurrent embedding requests
var ollamaEmbedMu sync.Mutex

type OllamaEmbedder struct {
	config *config.EmbedderProviderConfig
	client *ollama.Client
}

type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewOllamaEmbedder() *OllamaEmbedder {
	config := &config.EmbedderProviderConfig{
		Type:       "ollama",
		Model:      "nomic-embed-text",
		Host:       "http://localhost:11434",
		Dimension:  768,
		Timeout:    30,
		MaxRetries: 3,
	}

	embedder, _ := NewOllamaEmbedderFromConfig(config)
	return embedder
}

func NewOllamaEmbedderFromConfig(config *config.EmbedderProviderConfig) (*OllamaEmbedder, error) {
	return &OllamaEmbedder{
		config: config,
		client: ollama.NewClientWithTimeout(config.Host, time.Duration(config.Timeout)*time.Second),
	}, nil
}

func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	return e.EmbedWithContext(context.Background(), text)
}

func (e *OllamaEmbedder) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	// Serialize all Ollama embedding requests to prevent crashes
	// Ollama's llama runner crashes with SIGABRT when receiving concurrent embedding requests
	// See: https://github.com/ollama/ollama/issues - "decode: cannot decode batches with this context"
	ollamaEmbedMu.Lock()
	defer ollamaEmbedMu.Unlock()

	textLen := len(text)
	slog.Debug("Ollama embedding request", "model", e.config.Model, "text_length", textLen)

	request := OllamaEmbedRequest{
		Model:  e.config.Model,
		Prompt: text,
	}

	var resp *http.Response
	var err error
	for attempt := 0; attempt < e.config.MaxRetries; attempt++ {
		resp, err = e.client.MakeRequest(ctx, "/api/embeddings", request)
		if err == nil {
			break
		}

		slog.Debug("Ollama embedding retry", "attempt", attempt+1, "error", err, "text_length", textLen)
		if attempt < e.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		slog.Error("Ollama embedding failed", "error", err, "text_length", textLen, "model", e.config.Model)
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OllamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embedding) == 0 {
		return nil, fmt.Errorf("received empty embedding from Ollama")
	}

	return response.Embedding, nil
}

func (e *OllamaEmbedder) GetModel() string {
	return e.config.Model
}

func (e *OllamaEmbedder) SetModel(model string) {
	e.config.Model = model
}

func (e *OllamaEmbedder) GetBaseURL() string {
	return e.config.Host
}

func (e *OllamaEmbedder) SetBaseURL(baseURL string) {
	e.config.Host = baseURL
}

func (e *OllamaEmbedder) GetDimension() int {
	return e.config.Dimension
}

func (e *OllamaEmbedder) GetModelName() string {
	return e.config.Model
}

var (
	OllamaNomicEmbedText   = "nomic-embed-text"
	OllamaNomicEmbedTextV2 = "nomic-embed-text-v2"

	OllamaAllMiniLML6V2  = "all-minilm:l6-v2"
	OllamaAllMpnetBaseV2 = "all-mpnet-base-v2"

	OllamaBGESmallEnV15 = "bge-small-en-v1.5"
	OllamaBGELargeEnV15 = "bge-large-en-v1.5"

	OllamaE5SmallV2 = "e5-small-v2"
	OllamaE5BaseV2  = "e5-base-v2"
	OllamaE5LargeV2 = "e5-large-v2"
)

func (e *OllamaEmbedder) Close() error {

	return nil
}
