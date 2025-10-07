package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// A2A SERVER - HTTP+JSON Transport Implementation
// Spec Section 3.2.3: HTTP+JSON/REST Transport
// ============================================================================

// Server implements the A2A protocol HTTP+JSON server
type Server struct {
	host            string
	port            int
	baseURL         string
	agents          map[string]Agent // A2A-compliant agents
	agentCards      map[string]*AgentCard
	agentVisibility map[string]string // "public", "internal", "private"
	tasks           map[string]*Task  // Active tasks
	sessions        map[string]*Session
	mu              sync.RWMutex
	httpServer      *http.Server
	authValidator   AuthValidator // Optional JWT validator
}

// AuthValidator interface for authentication
type AuthValidator interface {
	HTTPMiddleware(next http.Handler) http.Handler
	ValidateToken(ctx context.Context, tokenString string) (interface{}, error)
}

// ServerConfig contains configuration for the A2A server
type ServerConfig struct {
	Host    string `yaml:"host" json:"host"`
	Port    int    `yaml:"port" json:"port"`
	BaseURL string `yaml:"base_url" json:"base_url"` // Public URL
}

// NewServer creates a new A2A protocol server
func NewServer(cfg *ServerConfig) *Server {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	}

	return &Server{
		host:            cfg.Host,
		port:            cfg.Port,
		baseURL:         baseURL,
		agents:          make(map[string]Agent),
		agentCards:      make(map[string]*AgentCard),
		agentVisibility: make(map[string]string),
		tasks:           make(map[string]*Task),
		sessions:        make(map[string]*Session),
	}
}

// SetAuthValidator sets the authentication validator
func (s *Server) SetAuthValidator(validator AuthValidator) {
	s.authValidator = validator
}

// ============================================================================
// AGENT REGISTRATION
// ============================================================================

// RegisterAgent registers an A2A-compliant agent
// Visibility: "public" (discoverable), "internal" (callable but not listed), "private" (local only)
func (s *Server) RegisterAgent(agentID string, agent Agent, visibility string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Normalize visibility
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "internal" && visibility != "private" {
		return fmt.Errorf("invalid visibility: %s", visibility)
	}

	// Get agent card
	card := agent.GetAgentCard()
	if card == nil {
		return fmt.Errorf("agent %s returned nil agent card", agentID)
	}

	// Set URL if not already set
	if card.URL == "" {
		card.URL = fmt.Sprintf("%s/agents/%s", s.baseURL, agentID)
	}

	// Set preferred transport if not set
	if card.PreferredTransport == "" {
		card.PreferredTransport = "http+json"
	}

	s.agents[agentID] = agent
	s.agentCards[agentID] = card
	s.agentVisibility[agentID] = visibility

	return nil
}

// ============================================================================
// HTTP SERVER
// ============================================================================

// Start starts the A2A HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// A2A Protocol Endpoints (Spec Section 7)
	mux.HandleFunc("/agents", s.handleListAgents) // Public discovery

	// Agent routes - conditionally protected by auth
	if s.authValidator != nil {
		mux.Handle("/agents/", s.authValidator.HTTPMiddleware(http.HandlerFunc(s.handleAgentRoutes)))
		mux.Handle("/sessions", s.authValidator.HTTPMiddleware(http.HandlerFunc(s.handleSessions)))
		mux.Handle("/sessions/", s.authValidator.HTTPMiddleware(http.HandlerFunc(s.handleSessionRoutes)))
	} else {
		mux.HandleFunc("/agents/", s.handleAgentRoutes)
		mux.HandleFunc("/sessions", s.handleSessions)
		mux.HandleFunc("/sessions/", s.handleSessionRoutes)
	}

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s.corsMiddleware(s.loggingMiddleware(mux)),
	}

	fmt.Printf("🚀 A2A Server (HTTP+JSON) starting on %s:%d\n", s.host, s.port)
	fmt.Printf("📋 Agent discovery: %s/agents\n", s.baseURL)
	if s.authValidator != nil {
		fmt.Printf("🔒 Authentication: ENABLED\n")
	} else {
		fmt.Printf("🔓 Authentication: DISABLED\n")
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// ============================================================================
// HTTP HANDLERS - Agent Discovery
// ============================================================================

// handleListAgents returns directory of public agents
// GET /agents
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Only include public agents
	agents := make([]AgentCard, 0, len(s.agentCards))
	for agentID, card := range s.agentCards {
		if s.agentVisibility[agentID] == "public" {
			agents = append(agents, *card)
		}
	}

	directory := AgentDirectory{
		Agents: agents,
		Total:  len(agents),
	}

	respondJSON(w, http.StatusOK, directory)
}

// handleAgentRoutes routes agent-specific requests
func (s *Server) handleAgentRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /agents/{agentId}[/...]
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	agentID := parts[0]

	// Route based on path
	switch {
	case len(parts) == 1:
		// GET /agents/{agentId} - Get agent card
		s.handleGetAgentCard(w, r, agentID)

	case len(parts) == 3 && parts[1] == "message" && parts[2] == "send":
		// POST /agents/{agentId}/message/send - A2A message/send (Spec 7.1)
		s.handleMessageSend(w, r, agentID)

	case len(parts) == 3 && parts[1] == "message" && parts[2] == "stream":
		// POST /agents/{agentId}/message/stream - A2A SSE streaming (Spec 7.2)
		s.handleMessageStream(w, r, agentID)

	case len(parts) == 3 && parts[1] == "tasks":
		taskID := parts[2]
		// GET /agents/{agentId}/tasks/{taskId} - A2A tasks/get (Spec 7.3)
		s.handleTaskGet(w, r, agentID, taskID)

	case len(parts) == 4 && parts[1] == "tasks" && parts[3] == "cancel":
		taskID := parts[2]
		// POST /agents/{agentId}/tasks/{taskId}/cancel - A2A tasks/cancel (Spec 7.4)
		s.handleTaskCancel(w, r, agentID, taskID)

	case len(parts) == 4 && parts[1] == "tasks" && parts[3] == "resubscribe":
		taskID := parts[2]
		// POST /agents/{agentId}/tasks/{taskId}/resubscribe - A2A tasks/resubscribe (Spec 7.9)
		s.handleTaskResubscribe(w, r, agentID, taskID)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleGetAgentCard returns an agent's card
// GET /agents/{agentId}
func (s *Server) handleGetAgentCard(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	card, exists := s.agentCards[agentID]
	visibility := s.agentVisibility[agentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Private agents not accessible via API
	if visibility == "private" {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, card)
}

// ============================================================================
// HTTP HANDLERS - A2A RPC Methods (Spec Section 7)
// ============================================================================

// handleMessageSend implements message/send (Spec Section 7.1)
// POST /agents/{agentId}/message/send
func (s *Server) handleMessageSend(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	visibility := s.agentVisibility[agentID]
	s.mu.RUnlock()

	if !exists || visibility == "private" {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var params MessageSendParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create or continue task
	var task *Task
	if params.TaskID != "" {
		// Continue existing task
		s.mu.RLock()
		task, exists = s.tasks[params.TaskID]
		s.mu.RUnlock()

		if !exists {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		// Add new message to task
		task.Messages = append(task.Messages, params.Message)
	} else {
		// Create new task
		task = &Task{
			ID:       uuid.New().String(),
			Messages: []Message{params.Message},
			Status: TaskStatus{
				State:     TaskStateSubmitted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
	}

	// Store task
	s.mu.Lock()
	s.tasks[task.ID] = task
	task.Status.State = TaskStateWorking
	task.Status.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Execute task asynchronously
	go s.executeTask(agent, task)

	// Return task immediately
	respondJSON(w, http.StatusAccepted, task)
}

// handleTaskGet implements tasks/get (Spec Section 7.3)
// GET /agents/{agentId}/tasks/{taskId}
func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request, agentID, taskID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// handleTaskCancel implements tasks/cancel (Spec Section 7.4)
// POST /agents/{agentId}/tasks/{taskId}/cancel
func (s *Server) handleTaskCancel(w http.ResponseWriter, r *http.Request, agentID, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional cancel reason
	var params TaskCancelParams
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&params)
	}

	s.mu.Lock()
	task, exists := s.tasks[taskID]
	if exists {
		task.Status.State = TaskStateCanceled
		task.Status.UpdatedAt = time.Now()
		if params.Reason != "" {
			task.Status.Reason = params.Reason
		}
	}
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// handleMessageStream implements message/stream with SSE (Spec Section 7.2)
// POST /agents/{agentId}/message/stream
func (s *Server) handleMessageStream(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	visibility := s.agentVisibility[agentID]
	s.mu.RUnlock()

	if !exists || visibility == "private" {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var params MessageSendParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create or continue task
	var task *Task
	if params.TaskID != "" {
		// Continue existing task
		s.mu.RLock()
		task, exists = s.tasks[params.TaskID]
		s.mu.RUnlock()

		if !exists {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		// Add new message to task
		task.Messages = append(task.Messages, params.Message)
	} else {
		// Create new task
		task = &Task{
			ID:       uuid.New().String(),
			Messages: []Message{params.Message},
			Status: TaskStatus{
				State:     TaskStateSubmitted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
	}

	// Store task
	s.mu.Lock()
	s.tasks[task.ID] = task
	task.Status.State = TaskStateWorking
	task.Status.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Set SSE headers (Spec Section 3.3.3)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial status event
	s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
		TaskID: task.ID,
		Status: task.Status,
	})

	// Execute task with streaming
	eventCh, err := agent.ExecuteTaskStreaming(r.Context(), task)
	if err != nil {
		// Send error status
		task.Status.State = TaskStateFailed
		task.Status.UpdatedAt = time.Now()
		task.Error = &TaskError{
			Code:    "execution_error",
			Message: err.Error(),
		}
		s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
			TaskID: task.ID,
			Status: task.Status,
		})
		return
	}

	// Stream events to client
	for event := range eventCh {
		switch event.Type {
		case StreamEventTypeMessage:
			if event.Message != nil {
				s.sendSSEEvent(w, flusher, "message", TaskMessageEvent{
					TaskID:  event.TaskID,
					Message: *event.Message,
				})
			}
		case StreamEventTypeArtifact:
			if event.Artifact != nil {
				s.sendSSEEvent(w, flusher, "artifact", TaskArtifactUpdateEvent{
					TaskID:   event.TaskID,
					Artifact: *event.Artifact,
				})
			}
		case StreamEventTypeStatus:
			if event.Status != nil {
				s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
					TaskID: event.TaskID,
					Status: *event.Status,
				})
				// Update task status
				s.mu.Lock()
				if t, exists := s.tasks[event.TaskID]; exists {
					t.Status = *event.Status
				}
				s.mu.Unlock()
			}
		}
	}

	// Send final completion status
	s.mu.Lock()
	if task.Status.State != TaskStateCompleted && task.Status.State != TaskStateFailed {
		task.Status.State = TaskStateCompleted
		task.Status.UpdatedAt = time.Now()
	}
	finalStatus := task.Status
	s.mu.Unlock()

	s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
		TaskID: task.ID,
		Status: finalStatus,
	})
}

// handleTaskResubscribe implements tasks/resubscribe (Spec Section 7.9)
// POST /agents/{agentId}/tasks/{taskId}/resubscribe
func (s *Server) handleTaskResubscribe(w http.ResponseWriter, r *http.Request, agentID, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get task
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	agent, agentExists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if !agentExists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send current status
	s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
		TaskID: task.ID,
		Status: task.Status,
	})

	// If task is still running, resubscribe to events
	if task.Status.State == TaskStateWorking {
		// Execute streaming again
		eventCh, err := agent.ExecuteTaskStreaming(r.Context(), task)
		if err != nil {
			task.Status.State = TaskStateFailed
			task.Status.UpdatedAt = time.Now()
			task.Error = &TaskError{
				Code:    "resubscribe_error",
				Message: err.Error(),
			}
			s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
				TaskID: task.ID,
				Status: task.Status,
			})
			return
		}

		// Stream events
		for event := range eventCh {
			switch event.Type {
			case StreamEventTypeMessage:
				if event.Message != nil {
					s.sendSSEEvent(w, flusher, "message", TaskMessageEvent{
						TaskID:  event.TaskID,
						Message: *event.Message,
					})
				}
			case StreamEventTypeArtifact:
				if event.Artifact != nil {
					s.sendSSEEvent(w, flusher, "artifact", TaskArtifactUpdateEvent{
						TaskID:   event.TaskID,
						Artifact: *event.Artifact,
					})
				}
			case StreamEventTypeStatus:
				if event.Status != nil {
					s.sendSSEEvent(w, flusher, "status", TaskStatusUpdateEvent{
						TaskID: event.TaskID,
						Status: *event.Status,
					})
				}
			}
		}
	}
}

// sendSSEEvent sends a Server-Sent Event per A2A spec
func (s *Server) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	// Write SSE format: event: type\ndata: json\n\n
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// ============================================================================
// TASK EXECUTION
// ============================================================================

// executeTask runs the agent and updates task status
func (s *Server) executeTask(agent Agent, task *Task) {
	ctx := context.Background()

	// Execute agent
	resultTask, err := agent.ExecuteTask(ctx, task)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		task.Status.State = TaskStateFailed
		task.Status.UpdatedAt = time.Now()
		task.Error = &TaskError{
			Code:    "execution_error",
			Message: err.Error(),
		}
	} else if resultTask != nil {
		// Copy result into our task
		task.Status = resultTask.Status
		task.Messages = resultTask.Messages
		task.Artifacts = resultTask.Artifacts
		task.Error = resultTask.Error
		task.Metadata = resultTask.Metadata
	}

	// Ensure completion state
	if task.Status.State != TaskStateFailed && task.Status.State != TaskStateCanceled {
		task.Status.State = TaskStateCompleted
	}
	task.Status.UpdatedAt = time.Now()
}

// ============================================================================
// SESSION MANAGEMENT (Hector Extension)
// ============================================================================

// handleSessions handles session operations
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateSession(w, r)
	case http.MethodGet:
		s.handleListSessions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionRoutes routes session-specific requests
func (s *Server) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/sessions/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sessionID := parts[0]

	switch r.Method {
	case http.MethodGet:
		s.handleGetSession(w, r, sessionID)
	case http.MethodDelete:
		s.handleDeleteSession(w, r, sessionID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentName string                 `json:"agentName"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find agent by ID or name
	s.mu.RLock()
	agentID := req.AgentName
	// First try direct lookup by ID
	_, exists := s.agents[agentID]
	if !exists {
		// Try finding by agent name (card.Name)
		for id, card := range s.agentCards {
			if card.Name == req.AgentName {
				agentID = id
				exists = true
				break
			}
		}
	}
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Create session
	session := &Session{
		ID:             uuid.New().String(),
		AgentName:      agentID, // Store the agent ID for lookups
		Tasks:          []string{},
		CreatedAt:      time.Now(),
		LastActivityAt: time.Now(),
		Metadata:       req.Metadata,
	}

	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	respondJSON(w, http.StatusCreated, session)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, session)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	s.mu.Lock()
	_, exists := s.sessions[sessionID]
	if exists {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fmt.Printf("[A2A] %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		fmt.Printf("[A2A] %s %s - %v\n", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
