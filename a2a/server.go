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
	"github.com/gorilla/websocket"
)

// ============================================================================
// A2A SERVER - Exposes Hector agents via A2A protocol
// ============================================================================

// Server implements the A2A protocol server
type Server struct {
	host       string
	port       int
	baseURL    string
	agents     map[string]Agent // Pure A2A Agent interface
	agentCards map[string]*AgentCard
	sessions   map[string]*Session
	tasks      map[string]*TaskResponse
	mu         sync.RWMutex
	httpServer *http.Server
}

// ServerConfig contains configuration for the A2A server
type ServerConfig struct {
	Host    string `yaml:"host" json:"host"`
	Port    int    `yaml:"port" json:"port"`
	BaseURL string `yaml:"base_url" json:"base_url"` // Public URL for agent cards
}

// NewServer creates a new A2A protocol server
func NewServer(cfg *ServerConfig) *Server {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	}

	return &Server{
		host:       cfg.Host,
		port:       cfg.Port,
		baseURL:    baseURL,
		agents:     make(map[string]Agent),
		agentCards: make(map[string]*AgentCard),
		sessions:   make(map[string]*Session),
		tasks:      make(map[string]*TaskResponse),
	}
}

// ============================================================================
// AGENT REGISTRATION
// ============================================================================

// RegisterAgent registers an A2A-compliant agent
// The agent must implement the pure A2A Agent interface
func (s *Server) RegisterAgent(agentID string, agent Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the agent's card directly (pure A2A protocol)
	card := agent.GetAgentCard()

	// Ensure card has the correct agentID and endpoints
	card.AgentID = agentID
	card.Endpoints = AgentEndpoints{
		Task:   fmt.Sprintf("%s/agents/%s/tasks", s.baseURL, agentID),
		Stream: fmt.Sprintf("%s/agents/%s/stream", s.baseURL, agentID),
		Status: fmt.Sprintf("%s/agents/%s/tasks/{taskId}", s.baseURL, agentID),
	}

	s.agents[agentID] = agent
	s.agentCards[agentID] = card

	return nil
}

// Note: createAgentCard removed - agents provide their own cards via GetAgentCard()

// ============================================================================
// HTTP SERVER
// ============================================================================

// Start starts the A2A HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// A2A Protocol Endpoints
	mux.HandleFunc("/agents", s.handleListAgents)       // GET - List all agents
	mux.HandleFunc("/agents/", s.handleAgentRoutes)     // Agent-specific routes
	mux.HandleFunc("/sessions", s.handleSessions)       // Session management
	mux.HandleFunc("/sessions/", s.handleSessionRoutes) // Session-specific routes

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s.corsMiddleware(s.loggingMiddleware(mux)),
	}

	fmt.Printf("🚀 A2A Server starting on %s:%d\n", s.host, s.port)
	fmt.Printf("📋 Agent Cards available at: %s/agents\n", s.baseURL)
	fmt.Printf("💬 Sessions available at: %s/sessions\n", s.baseURL)

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
// HTTP HANDLERS
// ============================================================================

// handleListAgents returns the directory of all available agents
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]AgentCard, 0, len(s.agentCards))
	for _, card := range s.agentCards {
		agents = append(agents, *card)
	}

	directory := AgentDirectory{
		Agents: agents,
		Total:  len(agents),
	}

	respondJSON(w, http.StatusOK, directory)
}

// handleAgentRoutes routes agent-specific requests
func (s *Server) handleAgentRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /agents/{agentId}[/tasks[/{taskId}]]
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
	case len(parts) == 2 && parts[1] == "tasks":
		// POST /agents/{agentId}/tasks - Execute task
		s.handleExecuteTask(w, r, agentID)
	case len(parts) == 3 && parts[1] == "tasks":
		// GET /agents/{agentId}/tasks/{taskId} - Get task status
		s.handleGetTaskStatus(w, r, agentID, parts[2])
	case len(parts) == 2 && parts[1] == "stream":
		// WebSocket /agents/{agentId}/stream - Streaming execution
		s.handleStreamTask(w, r, agentID)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleGetAgentCard returns an agent's card
func (s *Server) handleGetAgentCard(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	card, exists := s.agentCards[agentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, card)
}

// handleExecuteTask executes a task on an agent
func (s *Server) handleExecuteTask(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Parse task request
	var taskReq TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&taskReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate task ID if not provided
	if taskReq.TaskID == "" {
		taskReq.TaskID = uuid.New().String()
	}

	// Initialize task response
	taskResp := &TaskResponse{
		TaskID:    taskReq.TaskID,
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
	}

	s.mu.Lock()
	s.tasks[taskReq.TaskID] = taskResp
	s.mu.Unlock()

	// Execute agent in background
	go s.executeTask(agent, &taskReq, taskResp)

	// Return immediate response with task ID
	respondJSON(w, http.StatusAccepted, taskResp)
}

// executeTask runs the agent and updates task status
// Uses pure A2A protocol: TaskRequest → Agent.ExecuteTask() → TaskResponse
func (s *Server) executeTask(agent Agent, req *TaskRequest, resp *TaskResponse) {
	ctx := context.Background()

	// Execute agent using pure A2A protocol
	result, err := agent.ExecuteTask(ctx, req)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		resp.Status = TaskStatusFailed
		resp.Error = &TaskError{
			Code:    "execution_error",
			Message: err.Error(),
		}
	} else {
		// Copy the result into our response
		resp.Status = result.Status
		resp.Output = result.Output
		resp.Error = result.Error
		resp.Metadata = result.Metadata
		resp.EndedAt = result.EndedAt
	}

	if resp.EndedAt.IsZero() {
		resp.EndedAt = time.Now()
	}
}

// handleGetTaskStatus returns the status of a task
func (s *Server) handleGetTaskStatus(w http.ResponseWriter, r *http.Request, agentID, taskID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	taskResp, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, taskResp)
}

// handleStreamTask handles streaming task execution via WebSocket
func (s *Server) handleStreamTask(w http.ResponseWriter, r *http.Request, agentID string) {
	// Upgrade to WebSocket
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins (configure for production)
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Get agent
	s.mu.RLock()
	agent, exists := s.agents[agentID]
	s.mu.RUnlock()

	if !exists {
		conn.WriteJSON(map[string]string{"error": "Agent not found"})
		return
	}

	// Read task request from WebSocket
	var taskReq TaskRequest
	if err := conn.ReadJSON(&taskReq); err != nil {
		conn.WriteJSON(map[string]string{"error": "Invalid task request"})
		return
	}

	// Generate task ID if not provided
	if taskReq.TaskID == "" {
		taskReq.TaskID = uuid.New().String()
	}

	// Execute task with streaming
	ctx := context.Background()
	streamCh, err := agent.ExecuteTaskStreaming(ctx, &taskReq)
	if err != nil {
		conn.WriteJSON(&StreamChunk{
			TaskID:    taskReq.TaskID,
			ChunkType: ChunkTypeError,
			Content:   fmt.Sprintf("Execution failed: %v", err),
			Timestamp: time.Now(),
			Final:     true,
		})
		return
	}

	// Stream chunks to client
	for chunk := range streamCh {
		// Add timestamp if not set
		if chunk.Timestamp.IsZero() {
			chunk.Timestamp = time.Now()
		}

		if err := conn.WriteJSON(chunk); err != nil {
			// Client disconnected
			return
		}

		// If this is the final chunk, close connection
		if chunk.Final {
			break
		}
	}
}

// ============================================================================
// SESSION MANAGEMENT
// ============================================================================

// handleSessions handles session creation and listing
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Create new session
		s.handleCreateSession(w, r)
	case http.MethodGet:
		// List sessions (optionally filtered by agent)
		s.handleListSessions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSessionRoutes routes session-specific requests
func (s *Server) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path: /sessions/{sessionId}
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
	case http.MethodPost:
		// POST /sessions/{sessionId}/tasks - Execute task in session context
		if len(parts) == 2 && parts[1] == "tasks" {
			s.handleSessionTask(w, r, sessionID)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateSession creates a new session
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify agent exists
	s.mu.RLock()
	_, exists := s.agents[req.AgentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Create session
	session := &Session{
		SessionID:      uuid.New().String(),
		AgentID:        req.AgentID,
		CreatedAt:      time.Now(),
		LastActivityAt: time.Now(),
		State:          make(map[string]interface{}),
		Metadata:       req.Metadata,
	}

	s.mu.Lock()
	s.sessions[session.SessionID] = session
	s.mu.Unlock()

	respondJSON(w, http.StatusCreated, session)
}

// handleListSessions lists all sessions, optionally filtered by agent
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range s.sessions {
		if agentID == "" || session.AgentID == agentID {
			sessions = append(sessions, session)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// handleGetSession retrieves a session
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

// handleDeleteSession deletes a session
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

// handleSessionTask executes a task in a session context
func (s *Server) handleSessionTask(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Get session
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Update session activity
	s.mu.Lock()
	session.LastActivityAt = time.Now()
	s.mu.Unlock()

	// Parse task request
	var taskReq TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&taskReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Inject session context
	if taskReq.Context == nil {
		taskReq.Context = &TaskContext{}
	}
	taskReq.Context.SessionID = sessionID

	// Generate task ID if not provided
	if taskReq.TaskID == "" {
		taskReq.TaskID = uuid.New().String()
	}

	// Get agent
	s.mu.RLock()
	agent, exists := s.agents[session.AgentID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Initialize task response
	taskResp := &TaskResponse{
		TaskID:    taskReq.TaskID,
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
	}

	s.mu.Lock()
	s.tasks[taskReq.TaskID] = taskResp
	s.mu.Unlock()

	// Execute agent in background
	go s.executeTask(agent, &taskReq, taskResp)

	// Return immediate response with task ID
	respondJSON(w, http.StatusAccepted, taskResp)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// extractInput extracts the input string from TaskInput
func (s *Server) extractInput(input TaskInput) string {
	switch v := input.Content.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		// Try to JSON encode if it's structured data
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
	}
	return fmt.Sprintf("%v", input.Content)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ============================================================================
// MIDDLEWARE
// ============================================================================

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
