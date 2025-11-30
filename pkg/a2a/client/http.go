package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"google.golang.org/protobuf/encoding/protojson"
)

type HTTPClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHTTPClient(baseURL, token string) A2AClient {
	return NewHTTPClientWithTLS(baseURL, token, nil)
}

// NewHTTPClientWithTLS creates an HTTP client with optional TLS configuration
func NewHTTPClientWithTLS(baseURL, token string, tlsConfig *httpclient.TLSConfig) A2AClient {
	var httpClient *http.Client

	if tlsConfig != nil && (tlsConfig.InsecureSkipVerify || tlsConfig.CACertificate != "") {
		transport, err := httpclient.ConfigureTLS(tlsConfig)
		if err != nil {
			// Log warning but use default client
			fmt.Printf("Warning: Failed to configure TLS for A2A HTTP client: %v\n", err)
			httpClient = &http.Client{
				Timeout: 300 * time.Second,
			}
		} else {
			if tlsConfig.InsecureSkipVerify {
				fmt.Printf("Warning: TLS certificate verification disabled for A2A HTTP client (insecure_skip_verify=true)\n")
			}
			httpClient = &http.Client{
				Timeout:   300 * time.Second,
				Transport: transport,
			}
		}
	} else {
		httpClient = &http.Client{
			Timeout: 300 * time.Second,
		}
	}

	return &HTTPClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		client:  httpClient,
	}
}

func (c *HTTPClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/message:send", c.baseURL, agentID)

	reqProto := &pb.SendMessageRequest{
		Request: message,
	}
	jsonData, err := protojson.Marshal(reqProto)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response pb.SendMessageResponse
	if err := protojson.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (c *HTTPClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/message:stream", c.baseURL, agentID)

	reqProto := &pb.SendMessageRequest{
		Request: message,
	}
	jsonData, err := protojson.Marshal(reqProto)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	streamChan := make(chan *pb.StreamResponse, 10)

	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		// SSE Streaming Pattern (intentionally duplicated across streaming implementations)
		// ================================================================================
		// This pattern is used in multiple files for SSE stream processing:
		//   - pkg/a2a/client/http.go (this file)
		//   - pkg/llms/anthropic.go
		//   - pkg/llms/gemini.go
		//   - pkg/llms/openai.go
		//   - pkg/llms/ollama.go
		//   - pkg/tools/mcp.go
		//
		// The duplication is intentional because:
		// 1. Each implementation has different error handling semantics (channels vs returns)
		// 2. Each has context-specific logging and recovery behavior
		// 3. Extracting to a shared utility would require complex callback patterns
		//
		// Key pattern: Use bufio.NewReader.ReadBytes('\n') instead of bufio.Scanner
		// Reason: Scanner has a default 64KB buffer limit which fails on large payloads
		// (e.g., base64-encoded images in tool results). ReadBytes has no such limit.
		// ================================================================================
		reader := bufio.NewReader(resp.Body)

		var currentEvent string
		var currentData string

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				// Log non-EOF errors for debugging, then exit stream loop
				slog.Debug("A2A SSE stream read error", "error", err)
				break
			}

			lineStr := strings.TrimSpace(string(line))

			if strings.HasPrefix(lineStr, "event: ") {
				currentEvent = strings.TrimPrefix(lineStr, "event: ")
			} else if strings.HasPrefix(lineStr, "data: ") {
				currentData = strings.TrimPrefix(lineStr, "data: ")
			} else if lineStr == "" && currentData != "" {

				if currentEvent == "message" || currentEvent == "" {

					var streamResp pb.StreamResponse
					if err := protojson.Unmarshal([]byte(currentData), &streamResp); err == nil {
						streamChan <- &streamResp
					} else {

						var rpcResp struct {
							JSONRPC string          `json:"jsonrpc"`
							ID      interface{}     `json:"id"`
							Result  json.RawMessage `json:"result"`
						}
						if err := json.Unmarshal([]byte(currentData), &rpcResp); err == nil && rpcResp.Result != nil {

							if err := protojson.Unmarshal(rpcResp.Result, &streamResp); err == nil {
								streamChan <- &streamResp
							}
						}
					}
				}

				currentEvent = ""
				currentData = ""
			}
		}
	}()

	return streamChan, nil
}

func (c *HTTPClient) ListAgents(ctx context.Context) ([]*pb.AgentCard, error) {
	url := fmt.Sprintf("%s/v1/agents", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Agents []json.RawMessage `json:"agents"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var agents []*pb.AgentCard
	for _, agentData := range response.Agents {
		var card pb.AgentCard
		if err := protojson.Unmarshal(agentData, &card); err != nil {

			continue
		}
		agents = append(agents, &card)
	}

	return agents, nil
}

func (c *HTTPClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	// Per A2A spec Section 5.3: agent cards MUST be at /.well-known/agent-card.json
	url := fmt.Sprintf("%s/v1/agents/%s/.well-known/agent-card.json", c.baseURL, agentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var cardData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&cardData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	card := &pb.AgentCard{
		Name:        getString(cardData, "name", ""),
		Description: getString(cardData, "description", ""),
		Version:     getString(cardData, "version", ""),
	}

	if caps, ok := cardData["capabilities"].(map[string]interface{}); ok {
		card.Capabilities = &pb.AgentCapabilities{
			Streaming: getBool(caps, "streaming", false),
		}
	}

	return card, nil
}

func (c *HTTPClient) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/tasks/%s", c.baseURL, agentID, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var task pb.Task
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := protojson.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &task, nil
}

func (c *HTTPClient) ListTasks(ctx context.Context, agentID string, contextID string, status pb.TaskState, pageSize int32, pageToken string) ([]*pb.Task, string, int32, error) {
	// Build query parameters
	url := fmt.Sprintf("%s/v1/tasks", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	// Add query parameters
	q := req.URL.Query()
	if contextID != "" {
		q.Set("context_id", contextID)
	}
	if status != pb.TaskState_TASK_STATE_UNSPECIFIED {
		q.Set("status", fmt.Sprintf("%d", int(status)))
	}
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", 0, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var response pb.ListTasksResponse
	if err := protojson.Unmarshal(body, &response); err != nil {
		return nil, "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Tasks, response.NextPageToken, response.TotalSize, nil
}

func (c *HTTPClient) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/tasks/%s:cancel", c.baseURL, agentID, taskID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var task pb.Task
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := protojson.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &task, nil
}

// GetAgentID returns empty string for HTTPClient (not applicable for multi-agent HTTP services)
func (c *HTTPClient) GetAgentID() string {
	return ""
}

func (c *HTTPClient) Close() error {

	c.client.CloseIdleConnections()
	return nil
}

func getString(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultVal
}
