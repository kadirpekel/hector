package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// A2A CLIENT - Call external A2A agents
// ============================================================================

// Client is an A2A protocol client for calling external agents
type Client struct {
	httpClient *http.Client
	auth       *AuthCredentials
}

// AuthCredentials contains authentication information
type AuthCredentials struct {
	Type         string // "bearer", "apiKey"
	Token        string
	APIKey       string
	APIKeyHeader string // Header name for API key (default: "X-API-Key")
}

// ClientConfig contains configuration for the A2A client
type ClientConfig struct {
	Timeout time.Duration
	Auth    *AuthCredentials
}

// NewClient creates a new A2A protocol client
func NewClient(cfg *ClientConfig) *Client {
	if cfg == nil {
		cfg = &ClientConfig{
			Timeout: 60 * time.Second,
		}
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		auth: cfg.Auth,
	}
}

// ============================================================================
// AGENT DISCOVERY
// ============================================================================

// DiscoverAgent fetches an agent's card
func (c *Client) DiscoverAgent(ctx context.Context, agentURL string) (*AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, agentURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get agent card: %s - %s", resp.Status, string(body))
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &card, nil
}

// ListAgents fetches available agents from a directory endpoint
func (c *Client) ListAgents(ctx context.Context, directoryURL string) ([]AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directoryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list agents: %s - %s", resp.Status, string(body))
	}

	var directory AgentDirectory
	if err := json.NewDecoder(resp.Body).Decode(&directory); err != nil {
		return nil, fmt.Errorf("failed to decode agent directory: %w", err)
	}

	return directory.Agents, nil
}

// ============================================================================
// TASK EXECUTION
// ============================================================================

// ExecuteTask executes a task on an external agent
func (c *Client) ExecuteTask(ctx context.Context, agentCard *AgentCard, input string, params map[string]interface{}) (*TaskResponse, error) {
	taskReq := TaskRequest{
		TaskID: uuid.New().String(),
		Input: TaskInput{
			Type:    "text/plain",
			Content: input,
		},
		Parameters: params,
	}

	return c.ExecuteTaskRequest(ctx, agentCard, &taskReq)
}

// ExecuteTaskRequest executes a task request on an external agent
func (c *Client) ExecuteTaskRequest(ctx context.Context, agentCard *AgentCard, taskReq *TaskRequest) (*TaskResponse, error) {
	// Ensure task has an ID
	if taskReq.TaskID == "" {
		taskReq.TaskID = uuid.New().String()
	}

	// Marshal task request
	body, err := json.Marshal(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agentCard.Endpoints.Task, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(req)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("task execution failed: %s - %s", resp.Status, string(body))
	}

	// Decode response
	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	// If task is async (status running/pending), poll for completion
	if taskResp.Status == TaskStatusRunning || taskResp.Status == TaskStatusPending {
		return c.waitForTask(ctx, agentCard, taskResp.TaskID)
	}

	return &taskResp, nil
}

// waitForTask polls for task completion
func (c *Client) waitForTask(ctx context.Context, agentCard *AgentCard, taskID string) (*TaskResponse, error) {
	if agentCard.Endpoints.Status == "" {
		return nil, fmt.Errorf("agent does not support status endpoint")
	}

	// Build status URL
	statusURL := agentCard.Endpoints.Status
	// Replace {taskId} placeholder with actual task ID
	statusURL = strings.ReplaceAll(statusURL, "{taskId}", taskID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("task timed out")
		case <-ticker.C:
			// Check task status
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create status request: %w", err)
			}

			c.setAuthHeaders(req)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to get task status: %w", err)
			}

			var taskResp TaskResponse
			if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("failed to decode task status: %w", err)
			}
			resp.Body.Close()

			// Check if task is complete
			if taskResp.Status == TaskStatusCompleted || taskResp.Status == TaskStatusFailed || taskResp.Status == TaskStatusCancelled {
				return &taskResp, nil
			}
		}
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// setAuthHeaders sets authentication headers on the request
func (c *Client) setAuthHeaders(req *http.Request) {
	if c.auth == nil {
		return
	}

	switch c.auth.Type {
	case "bearer":
		if c.auth.Token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.auth.Token))
		}
	case "apiKey":
		header := c.auth.APIKeyHeader
		if header == "" {
			header = "X-API-Key"
		}
		if c.auth.APIKey != "" {
			req.Header.Set(header, c.auth.APIKey)
		}
	}
}

// ExtractOutputText extracts text content from task output
func ExtractOutputText(output *TaskOutput) string {
	if output == nil {
		return ""
	}

	switch v := output.Content.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		// Try to JSON encode
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
	}

	return fmt.Sprintf("%v", output.Content)
}
