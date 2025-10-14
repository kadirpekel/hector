package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

// HTTPClient implements A2AClient for remote A2A servers over HTTP/REST
type HTTPClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewHTTPClient creates a new HTTP-based A2A client
func NewHTTPClient(baseURL, token string) A2AClient {
	return &HTTPClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		client: &http.Client{
			Timeout: 300 * time.Second, // Long timeout for streaming
		},
	}
}

// SendMessage sends a non-streaming message to an agent
func (c *HTTPClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/message:send", c.baseURL, agentID)

	// Build request using protojson
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

// StreamMessage sends a streaming message to an agent
func (c *HTTPClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/message:stream", c.baseURL, agentID)

	// Build request using protojson
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

	// Create channel for streaming responses
	streamChan := make(chan *pb.StreamResponse, 10)

	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// grpc-gateway wraps streaming responses in a "result" field
			var wrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal([]byte(line), &wrapper); err != nil {
				continue
			}

			// Parse the actual StreamResponse using protojson
			var streamResp pb.StreamResponse
			if err := protojson.Unmarshal(wrapper.Result, &streamResp); err != nil {
				continue
			}

			// Send the parsed response
			streamChan <- &streamResp
		}
	}()

	return streamChan, nil
}

// ListAgents lists all available agents from the server
func (c *HTTPClient) ListAgents(ctx context.Context) ([]AgentInfo, error) {
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

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse agents from response
	var agents []AgentInfo
	if agentsList, ok := response["agents"].([]interface{}); ok {
		for _, a := range agentsList {
			if agentData, ok := a.(map[string]interface{}); ok {
				agent := AgentInfo{
					ID:          getString(agentData, "id", ""),
					Name:        getString(agentData, "name", ""),
					Description: getString(agentData, "description", ""),
					Endpoint:    getString(agentData, "endpoint", ""),
				}
				agents = append(agents, agent)
			}
		}
	}

	return agents, nil
}

// GetAgentCard retrieves the agent card for a specific agent
func (c *HTTPClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
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

	// Parse generic JSON (A2A spec format)
	var cardData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&cardData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to pb.AgentCard
	card := &pb.AgentCard{
		Name:        getString(cardData, "name", ""),
		Description: getString(cardData, "description", ""),
		Version:     getString(cardData, "version", ""),
	}

	// Parse capabilities if present
	if caps, ok := cardData["capabilities"].(map[string]interface{}); ok {
		card.Capabilities = &pb.AgentCapabilities{
			Streaming: getBool(caps, "streaming", false),
		}
	}

	return card, nil
}

// GetTask retrieves a task by ID
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

// CancelTask cancels a running task
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

// Close releases resources (HTTP client doesn't need cleanup)
func (c *HTTPClient) Close() error {
	return nil
}

// Helper functions for JSON parsing
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
