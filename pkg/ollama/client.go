package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/internal/httpclient"
)

// ============================================================================
// SHARED OLLAMA CLIENT
// ============================================================================

// Client provides a shared HTTP client for Ollama API interactions
type Client struct {
	baseURL    string
	httpClient *httpclient.Client
}

// NewClient creates a new Ollama client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: 60 * time.Second,
			}),
			httpclient.WithMaxRetries(3),
			httpclient.WithBaseDelay(2*time.Second),
		),
	}
}

// NewClientWithTimeout creates a new Ollama client with custom timeout
func NewClientWithTimeout(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: timeout,
			}),
			httpclient.WithMaxRetries(3),
			httpclient.WithBaseDelay(2*time.Second),
		),
	}
}

// MakeRequest makes an HTTP request to the Ollama API
func (c *Client) MakeRequest(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

// MakeStreamingRequest makes a streaming HTTP request to the Ollama API
func (c *Client) MakeStreamingRequest(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make streaming request: %w", err)
	}

	return resp, nil
}

// GetBaseURL returns the base URL of the client
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
