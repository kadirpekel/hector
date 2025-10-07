package a2a

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
)

// ============================================================================
// A2A CLIENT - HTTP+JSON Transport Client
// Implements A2A client for calling external agents
// ============================================================================

// Client is an A2A protocol client
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
// GET /agents/{agentId}
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
// GET /agents
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
// MESSAGE SENDING (A2A Spec Section 7.1)
// ============================================================================

// SendMessage sends a message to an agent using A2A message/send
// POST /agents/{agentId}/message/send
func (c *Client) SendMessage(ctx context.Context, agentURL string, message Message, config *MessageConfiguration) (*Task, error) {
	// Build message/send endpoint
	sendURL := fmt.Sprintf("%s/message/send", agentURL)

	// Build request params
	params := MessageSendParams{
		Message:       message,
		Configuration: config,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("message send failed: %s - %s", resp.Status, string(body))
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	// If task is async, poll for completion
	if task.Status.State == TaskStateSubmitted || task.Status.State == TaskStateWorking {
		return c.waitForTask(ctx, agentURL, task.ID)
	}

	return &task, nil
}

// SendTextMessage is a convenience method for sending simple text messages
func (c *Client) SendTextMessage(ctx context.Context, agentURL string, text string) (*Task, error) {
	message := Message{
		Role: MessageRoleUser,
		Parts: []Part{
			{
				Type: PartTypeText,
				Text: text,
			},
		},
	}

	return c.SendMessage(ctx, agentURL, message, nil)
}

// ContinueTask continues an existing task with a new message
// POST /agents/{agentId}/message/send with taskId
func (c *Client) ContinueTask(ctx context.Context, agentURL string, taskID string, message Message) (*Task, error) {
	sendURL := fmt.Sprintf("%s/message/send", agentURL)

	params := MessageSendParams{
		Message: message,
		TaskID:  taskID, // Continue existing task
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to continue task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("continue task failed: %s - %s", resp.Status, string(body))
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	// Poll for completion
	if task.Status.State == TaskStateWorking {
		return c.waitForTask(ctx, agentURL, task.ID)
	}

	return &task, nil
}

// ============================================================================
// TASK OPERATIONS (A2A Spec Sections 7.3-7.4)
// ============================================================================

// GetTask gets the current status of a task
// GET /agents/{agentId}/tasks/{taskId}
func (c *Client) GetTask(ctx context.Context, agentURL string, taskID string) (*Task, error) {
	taskURL := fmt.Sprintf("%s/tasks/%s", agentURL, taskID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, taskURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get task failed: %s - %s", resp.Status, string(body))
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &task, nil
}

// CancelTask cancels a running task
// POST /agents/{agentId}/tasks/{taskId}/cancel
func (c *Client) CancelTask(ctx context.Context, agentURL string, taskID string, reason string) (*Task, error) {
	cancelURL := fmt.Sprintf("%s/tasks/%s/cancel", agentURL, taskID)

	params := TaskCancelParams{
		TaskID: taskID,
		Reason: reason,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cancelURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cancel task failed: %s - %s", resp.Status, string(body))
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	return &task, nil
}

// waitForTask polls for task completion
func (c *Client) waitForTask(ctx context.Context, agentURL string, taskID string) (*Task, error) {
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
			task, err := c.GetTask(ctx, agentURL, taskID)
			if err != nil {
				return nil, err
			}

			// Check if task is complete
			switch task.Status.State {
			case TaskStateCompleted, TaskStateFailed, TaskStateCanceled:
				return task, nil
			}
		}
	}
}

// ============================================================================
// STREAMING (Server-Sent Events - A2A Spec 7.2)
// ============================================================================

// SendMessageStreaming sends a message to an agent with SSE streaming (A2A Spec 7.2)
// Uses Server-Sent Events per A2A specification Section 3.3.3
func (c *Client) SendMessageStreaming(ctx context.Context, agentURL string, message Message) (<-chan StreamEvent, error) {
	// Construct SSE streaming endpoint: POST /message/stream
	streamURL := agentURL + "/message/stream"

	params := MessageSendParams{
		Message: message,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, streamURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSE stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("streaming failed: %s - %s", resp.Status, string(bodyBytes))
	}

	// Use shared SSE parser
	return c.parseSSEStream(ctx, resp, ""), nil
}

// ResubscribeToTask resumes streaming for an existing task (A2A Spec 7.9)
// POST /agents/{agentId}/tasks/{taskId}/resubscribe
func (c *Client) ResubscribeToTask(ctx context.Context, agentURL string, taskID string, lastEventID string) (<-chan StreamEvent, error) {
	// Construct resubscribe endpoint
	resubscribeURL := agentURL + "/tasks/" + taskID + "/resubscribe"

	params := TaskResubscribeParams{
		LastEventID: lastEventID,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resubscribeURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resubscribe to task: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("resubscribe failed: %s - %s", resp.Status, string(bodyBytes))
	}

	// Use shared SSE parser
	return c.parseSSEStream(ctx, resp, taskID), nil
}

// ============================================================================
// SESSION MANAGEMENT (Hector Extension)
// ============================================================================

// CreateSession creates a new conversation session
func (c *Client) CreateSession(ctx context.Context, baseURL string, agentName string, metadata map[string]interface{}) (*Session, error) {
	sessionURL := fmt.Sprintf("%s/sessions", baseURL)

	req := map[string]interface{}{
		"agentName": agentName,
		"metadata":  metadata,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
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

// ListSessions lists all conversation sessions
func (c *Client) ListSessions(ctx context.Context, baseURL string) ([]Session, error) {
	sessionURL := fmt.Sprintf("%s/sessions", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, sessionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session list failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		Sessions []Session `json:"sessions"`
		Total    int       `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return result.Sessions, nil
}

// GetSession retrieves a specific conversation session
func (c *Client) GetSession(ctx context.Context, baseURL string, sessionID string) (*Session, error) {
	sessionURL := fmt.Sprintf("%s/sessions/%s", baseURL, sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, sessionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session retrieval failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a conversation session
func (c *Client) DeleteSession(ctx context.Context, baseURL string, sessionID string) error {
	sessionURL := fmt.Sprintf("%s/sessions/%s", baseURL, sessionID)

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

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// parseSSEStream parses SSE events from an HTTP response body
// Shared by SendMessageStreaming and ResubscribeToTask
func (c *Client) parseSSEStream(ctx context.Context, resp *http.Response, taskID string) <-chan StreamEvent {
	eventCh := make(chan StreamEvent, 10)

	go func() {
		defer close(eventCh)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var eventType string
		var eventData string

		for scanner.Scan() {
			line := scanner.Text()

			// SSE format: "event: type" or "data: json" or empty line (delimiter)
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				eventData = strings.TrimPrefix(line, "data: ")
			} else if line == "" && eventType != "" && eventData != "" {
				// Parse and send event
				var event StreamEvent
				switch eventType {
				case "message":
					var msgEvent TaskMessageEvent
					if err := json.Unmarshal([]byte(eventData), &msgEvent); err == nil {
						event = StreamEvent{
							Type:      StreamEventTypeMessage,
							TaskID:    msgEvent.TaskID,
							Message:   &msgEvent.Message,
							Timestamp: time.Now(),
						}
					}
				case "status":
					var statusEvent TaskStatusUpdateEvent
					if err := json.Unmarshal([]byte(eventData), &statusEvent); err == nil {
						event = StreamEvent{
							Type:      StreamEventTypeStatus,
							TaskID:    statusEvent.TaskID,
							Status:    &statusEvent.Status,
							Timestamp: time.Now(),
						}
					}
				case "artifact":
					var artifactEvent TaskArtifactUpdateEvent
					if err := json.Unmarshal([]byte(eventData), &artifactEvent); err == nil {
						event = StreamEvent{
							Type:      StreamEventTypeArtifact,
							TaskID:    artifactEvent.TaskID,
							Artifact:  &artifactEvent.Artifact,
							Timestamp: time.Now(),
						}
					}
				}

				if event.Type != "" {
					eventCh <- event

					// Check if this is a final status
					if event.Type == StreamEventTypeStatus && event.Status != nil {
						switch event.Status.State {
						case TaskStateCompleted, TaskStateFailed, TaskStateCanceled:
							return
						}
					}
				}

				// Reset for next event
				eventType = ""
				eventData = ""
			}
		}

		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			// Send error event if context not cancelled
			eventCh <- StreamEvent{
				Type:      StreamEventTypeStatus,
				TaskID:    taskID,
				Status:    &TaskStatus{State: TaskStateFailed},
				Timestamp: time.Now(),
			}
		}
	}()

	return eventCh
}

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
// HELPER FUNCTIONS
// ============================================================================

// ExtractTextFromTask extracts text content from task messages and artifacts
func ExtractTextFromTask(task *Task) string {
	if task == nil {
		return ""
	}

	var texts []string

	// Extract from messages
	for _, msg := range task.Messages {
		if msg.Role == MessageRoleAssistant {
			for _, part := range msg.Parts {
				if part.Type == PartTypeText {
					texts = append(texts, part.Text)
				}
			}
		}
	}

	// Extract from artifacts
	for _, artifact := range task.Artifacts {
		for _, part := range artifact.Parts {
			if part.Type == PartTypeText {
				texts = append(texts, part.Text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

// CreateTextMessage is a helper to create a simple text message
func CreateTextMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Parts: []Part{
			{
				Type: PartTypeText,
				Text: text,
			},
		},
	}
}
