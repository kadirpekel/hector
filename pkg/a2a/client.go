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
	"github.com/gorilla/websocket"
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
// SESSION MANAGEMENT
// ============================================================================

// CreateSession creates a new conversation session
func (c *Client) CreateSession(ctx context.Context, sessionURL string, req *SessionRequest) (*Session, error) {
	var body []byte
	var err error
	if req != nil {
		body, err = json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, sessionURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session creation failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a conversation session
func (c *Client) DeleteSession(ctx context.Context, sessionURL string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, sessionURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("session deletion failed: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// ExecuteTaskInSession executes a task within a session context
func (c *Client) ExecuteTaskInSession(ctx context.Context, sessionURL string, input string, agentCard *AgentCard) (*TaskResponse, error) {
	taskReq := &TaskRequest{
		TaskID: uuid.New().String(),
		Input: TaskInput{
			Type:    "text/plain",
			Content: input,
		},
	}

	body, err := json.Marshal(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// sessionURL is like: http://localhost:8080/sessions/sess-123
	// we need to POST to: http://localhost:8080/sessions/sess-123/tasks
	taskURL := sessionURL + "/tasks"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, taskURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("task execution failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var taskResp TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// If task is running asynchronously, poll for completion using the status endpoint
	if taskResp.Status == TaskStatusRunning || taskResp.Status == TaskStatusPending {
		// Use the agent card's status endpoint
		// Status URL format: http://localhost:8080/agents/{agentId}/tasks/{taskId}
		statusURL := agentCard.Endpoints.Status
		statusURL = strings.ReplaceAll(statusURL, "{taskId}", taskResp.TaskID)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(60 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timeout:
				return nil, fmt.Errorf("task timed out after 60s")
			case <-ticker.C:
				// Poll for task status
				statusReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
				if err != nil {
					continue
				}

				c.setAuthHeaders(statusReq)

				statusResp, err := c.httpClient.Do(statusReq)
				if err != nil {
					continue
				}

				if statusResp.StatusCode == http.StatusOK {
					var updatedResp TaskResponse
					if err := json.NewDecoder(statusResp.Body).Decode(&updatedResp); err != nil {
						statusResp.Body.Close()
						continue
					}
					statusResp.Body.Close()

					if updatedResp.Status == TaskStatusCompleted || updatedResp.Status == TaskStatusFailed {
						return &updatedResp, nil
					}
					// Task still running, continue polling
				} else {
					statusResp.Body.Close()
				}
			}
		}
	}

	return &taskResp, nil
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

// ============================================================================
// STREAMING CLIENT
// ============================================================================

// ExecuteTaskStreamingInSession executes a task in a session with real-time streaming output
func (c *Client) ExecuteTaskStreamingInSession(ctx context.Context, sessionURL string, input string, outputCh chan<- string) (*TaskResponse, error) {
	wsURL := strings.Replace(sessionURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = wsURL + "/stream"

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	taskReq := TaskRequest{
		Input: TaskInput{
			Type:    "text/plain",
			Content: input,
		},
	}

	if err := conn.WriteJSON(taskReq); err != nil {
		return nil, fmt.Errorf("failed to send task request: %w", err)
	}

	var finalResponse *TaskResponse
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			return nil, fmt.Errorf("failed to read message: %w", err)
		}

		var chunk StreamChunk
		if err := json.Unmarshal(message, &chunk); err != nil {
			continue
		}

		if chunk.Final {
			finalResponse = &TaskResponse{
				TaskID:    chunk.TaskID,
				Status:    TaskStatusCompleted,
				StartedAt: chunk.Timestamp,
				EndedAt:   chunk.Timestamp,
			}
			break
		}

		if content, ok := chunk.Content.(string); ok && content != "" {
			outputCh <- content
		}
	}

	if finalResponse == nil {
		return nil, fmt.Errorf("no final response received")
	}

	return finalResponse, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

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
