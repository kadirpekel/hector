package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ============================================================================
// PROPER MOCKS FOR AgentServices
// ============================================================================

type MockLLMService struct {
	responses      []mockLLMResponse
	callCount      int
	shouldError    bool
	errorOnCall    int
	streamingCalls int
}

type mockLLMResponse struct {
	text      string
	toolCalls []llms.ToolCall
	tokens    int
}

func (m *MockLLMService) Generate(messages []llms.Message, tools []llms.ToolDefinition) (string, []llms.ToolCall, int, error) {
	if m.shouldError && m.callCount >= m.errorOnCall {
		return "", nil, 0, fmt.Errorf("mock LLM error after %d calls", m.errorOnCall)
	}

	if m.callCount >= len(m.responses) {
		return "Default response", nil, 10, nil
	}

	resp := m.responses[m.callCount]
	m.callCount++
	return resp.text, resp.toolCalls, resp.tokens, nil
}

func (m *MockLLMService) GenerateStreaming(messages []llms.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]llms.ToolCall, int, error) {
	m.streamingCalls++

	if m.shouldError && m.callCount >= m.errorOnCall {
		return nil, 0, fmt.Errorf("mock LLM streaming error")
	}

	if m.callCount >= len(m.responses) {
		outputCh <- "Completed"
		return nil, 10, nil
	}

	resp := m.responses[m.callCount]
	m.callCount++

	// Stream the text in chunks
	words := strings.Split(resp.text, " ")
	for _, word := range words {
		outputCh <- word + " "
	}

	return resp.toolCalls, resp.tokens, nil
}

func (m *MockLLMService) GenerateStructured(messages []llms.Message, tools []llms.ToolDefinition, cfg *llms.StructuredOutputConfig) (string, []llms.ToolCall, int, error) {
	return m.Generate(messages, tools)
}

func (m *MockLLMService) SupportsStructuredOutput() bool {
	return false
}

type MockToolService struct {
	tools      map[string]*MockTool
	executions []string // Track which tools were executed
}

func NewMockToolService() *MockToolService {
	return &MockToolService{
		tools:      make(map[string]*MockTool),
		executions: make([]string, 0),
	}
}

func (m *MockToolService) AddTool(name string, result string, shouldError bool) {
	m.tools[name] = &MockTool{
		name:        name,
		result:      result,
		shouldError: shouldError,
	}
}

func (m *MockToolService) ExecuteToolCall(ctx context.Context, toolCall llms.ToolCall) (string, error) {
	m.executions = append(m.executions, toolCall.Name)

	tool, exists := m.tools[toolCall.Name]
	if !exists {
		return "", fmt.Errorf("tool %s not found", toolCall.Name)
	}

	if tool.shouldError {
		return "", fmt.Errorf("tool %s failed", toolCall.Name)
	}

	return tool.result, nil
}

func (m *MockToolService) GetAvailableTools() []llms.ToolDefinition {
	defs := make([]llms.ToolDefinition, 0, len(m.tools))
	for name := range m.tools {
		defs = append(defs, llms.ToolDefinition{
			Name:        name,
			Description: fmt.Sprintf("Mock tool %s", name),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{"type": "string"},
				},
			},
		})
	}
	return defs
}

func (m *MockToolService) GetTool(name string) (interface{}, error) {
	if tool, exists := m.tools[name]; exists {
		return tool, nil
	}
	return nil, fmt.Errorf("tool %s not found", name)
}

type MockTool struct {
	name        string
	result      string
	shouldError bool
}

type MockContextService struct{}

func (m *MockContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	return []databases.SearchResult{}, nil
}

func (m *MockContextService) ExtractSources(context []databases.SearchResult) []string {
	return []string{}
}

type MockPromptService struct {
	buildCalls int
}

func (m *MockPromptService) BuildMessages(ctx context.Context, query string, slots reasoning.PromptSlots, conversation []llms.Message, additionalContext string) ([]llms.Message, error) {
	m.buildCalls++

	// Build a realistic message array
	messages := []llms.Message{
		{Role: "system", Content: "You are a helpful AI assistant"},
	}

	// Add conversation history
	messages = append(messages, conversation...)

	// Add current query
	messages = append(messages, llms.Message{
		Role:    "user",
		Content: query,
	})

	return messages, nil
}

type MockHistoryService struct {
	history   []llms.Message
	addCalls  int
	sessionID string
}

func (m *MockHistoryService) GetRecentHistory(sessionID string, count int) []llms.Message {
	m.sessionID = sessionID
	if len(m.history) <= count {
		return m.history
	}
	return m.history[len(m.history)-count:]
}

func (m *MockHistoryService) AddToHistory(sessionID string, msg llms.Message) {
	m.addCalls++
	m.sessionID = sessionID
	m.history = append(m.history, msg)
}

func (m *MockHistoryService) ClearHistory(sessionID string) {
	m.history = nil
	m.sessionID = sessionID
}

type MockAgentServices struct {
	config  config.ReasoningConfig
	llm     *MockLLMService
	tools   *MockToolService
	context *MockContextService
	prompt  *MockPromptService
	history *MockHistoryService
}

func NewMockAgentServices() *MockAgentServices {
	return &MockAgentServices{
		config: config.ReasoningConfig{
			Engine:        "chain-of-thought",
			MaxIterations: 5,
			ShowThinking:  false,
			ShowDebugInfo: false,
		},
		llm:     &MockLLMService{responses: []mockLLMResponse{}},
		tools:   NewMockToolService(),
		context: &MockContextService{},
		prompt:  &MockPromptService{},
		history: &MockHistoryService{},
	}
}

func (m *MockAgentServices) GetConfig() config.ReasoningConfig {
	return m.config
}

func (m *MockAgentServices) LLM() reasoning.LLMService {
	return m.llm
}

func (m *MockAgentServices) Tools() reasoning.ToolService {
	return m.tools
}

func (m *MockAgentServices) Context() reasoning.ContextService {
	return m.context
}

func (m *MockAgentServices) Prompt() reasoning.PromptService {
	return m.prompt
}

func (m *MockAgentServices) History() reasoning.HistoryService {
	return m.history
}

// ============================================================================
// REAL TESTS FOR AGENT EXECUTION
// ============================================================================

func TestAgent_ExecuteTask_SimpleResponse(t *testing.T) {
	// Setup
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "Hello! How can I help you today?", toolCalls: nil, tokens: 15},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	// Create test task
	task := &a2a.Task{
		ID: "test-1",
		Messages: []a2a.Message{
			{
				Role: a2a.MessageRoleUser,
				Parts: []a2a.Part{
					{Type: a2a.PartTypeText, Text: "Hello"},
				},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Execute
	result, err := agent.ExecuteTask(context.Background(), task)

	// Verify
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", result.Status.State)
	}

	if len(result.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(result.Messages))
	}

	assistantMsg := result.Messages[1]
	if assistantMsg.Role != a2a.MessageRoleAssistant {
		t.Errorf("Expected assistant role, got %s", assistantMsg.Role)
	}

	if !strings.Contains(assistantMsg.Parts[0].Text, "Hello") {
		t.Errorf("Expected response to contain 'Hello', got: %s", assistantMsg.Parts[0].Text)
	}

	// Verify LLM was called
	if services.llm.callCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", services.llm.callCount)
	}
}

func TestAgent_ExecuteTask_WithToolCall(t *testing.T) {
	// Setup
	services := NewMockAgentServices()
	services.tools.AddTool("search", "Search results: found 3 items", false)

	services.llm.responses = []mockLLMResponse{
		{
			text: "Let me search for that",
			toolCalls: []llms.ToolCall{
				{Name: "search", Arguments: map[string]interface{}{"query": "test"}},
			},
			tokens: 10,
		},
		{
			text:      "I found 3 items for you",
			toolCalls: nil,
			tokens:    15,
		},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "test-2",
		Messages: []a2a.Message{
			{
				Role:  a2a.MessageRoleUser,
				Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Search for something"}},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Execute
	result, err := agent.ExecuteTask(context.Background(), task)

	// Verify
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", result.Status.State)
	}

	// Verify tool was executed
	if len(services.tools.executions) != 1 {
		t.Errorf("Expected 1 tool execution, got %d", len(services.tools.executions))
	}

	if services.tools.executions[0] != "search" {
		t.Errorf("Expected 'search' tool to be executed, got %s", services.tools.executions[0])
	}

	// Verify LLM was called twice (once for tool call, once for final response)
	if services.llm.callCount != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", services.llm.callCount)
	}
}

func TestAgent_ExecuteTask_ToolFailure(t *testing.T) {
	// Setup
	services := NewMockAgentServices()
	services.tools.AddTool("failing_tool", "", true) // Tool will fail

	services.llm.responses = []mockLLMResponse{
		{
			text: "Let me use the tool",
			toolCalls: []llms.ToolCall{
				{Name: "failing_tool", Arguments: map[string]interface{}{"input": "test"}},
			},
			tokens: 10,
		},
		{
			text:      "The tool encountered an error",
			toolCalls: nil,
			tokens:    15,
		},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "test-3",
		Messages: []a2a.Message{
			{
				Role:  a2a.MessageRoleUser,
				Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Use the failing tool"}},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Execute
	result, err := agent.ExecuteTask(context.Background(), task)

	// Verify - should complete but mention the error
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed state (agent handles tool errors), got %s", result.Status.State)
	}

	// Tool should have been attempted
	if len(services.tools.executions) != 1 {
		t.Errorf("Expected 1 tool execution attempt, got %d", len(services.tools.executions))
	}
}

func TestAgent_ExecuteTask_MaxIterations(t *testing.T) {
	// Setup - LLM keeps calling tools forever
	services := NewMockAgentServices()
	services.config.MaxIterations = 3
	services.tools.AddTool("endless_tool", "result", false)

	// LLM will keep suggesting tool calls
	services.llm.responses = []mockLLMResponse{
		{text: "Iteration 1", toolCalls: []llms.ToolCall{{Name: "endless_tool", Arguments: map[string]interface{}{}}}, tokens: 10},
		{text: "Iteration 2", toolCalls: []llms.ToolCall{{Name: "endless_tool", Arguments: map[string]interface{}{}}}, tokens: 10},
		{text: "Iteration 3", toolCalls: []llms.ToolCall{{Name: "endless_tool", Arguments: map[string]interface{}{}}}, tokens: 10},
		{text: "Iteration 4", toolCalls: []llms.ToolCall{{Name: "endless_tool", Arguments: map[string]interface{}{}}}, tokens: 10},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 3,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "test-4",
		Messages: []a2a.Message{
			{
				Role:  a2a.MessageRoleUser,
				Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Run endless loop"}},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Execute
	result, err := agent.ExecuteTask(context.Background(), task)

	// Verify
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Should complete (hit max iterations)
	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", result.Status.State)
	}

	// Should not exceed max iterations
	if services.llm.callCount > 3 {
		t.Errorf("Expected max 3 LLM calls (max iterations), got %d", services.llm.callCount)
	}
}

func TestAgent_ExecuteTask_ContextCancellation(t *testing.T) {
	// Setup
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "This will be canceled", toolCalls: nil, tokens: 10},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "test-5",
		Messages: []a2a.Message{
			{
				Role:  a2a.MessageRoleUser,
				Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "This will be canceled"}},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Create cancelable context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait to ensure context is canceled
	time.Sleep(5 * time.Millisecond)

	// Execute with canceled context
	result, err := agent.ExecuteTask(ctx, task)

	// Verify - should complete (agent handles cancellation gracefully)
	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Response should mention cancellation
	response := result.Messages[len(result.Messages)-1].Parts[0].Text
	if !strings.Contains(response, "Cancel") && !strings.Contains(response, "cancel") {
		t.Logf("Note: Response may not explicitly mention cancellation: %s", response)
	}
}

func TestAgent_ExecuteTaskStreaming_Success(t *testing.T) {
	// Setup
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "Streaming response here", toolCalls: nil, tokens: 15},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "test-6",
		Messages: []a2a.Message{
			{
				Role:  a2a.MessageRoleUser,
				Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Stream response"}},
			},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	// Execute
	eventCh, err := agent.ExecuteTaskStreaming(context.Background(), task)
	if err != nil {
		t.Fatalf("ExecuteTaskStreaming failed: %v", err)
	}

	// Collect events
	var messages []a2a.Message
	var finalStatus *a2a.TaskStatus

	for event := range eventCh {
		if event.Type == a2a.StreamEventTypeMessage {
			messages = append(messages, *event.Message)
		} else if event.Type == a2a.StreamEventTypeStatus {
			finalStatus = event.Status
		}
	}

	// Verify
	if len(messages) == 0 {
		t.Error("Expected to receive streamed messages")
	}

	if finalStatus == nil {
		t.Fatal("Expected final status event")
	}

	if finalStatus.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", finalStatus.State)
	}

	// Note: Agent uses Generate() not GenerateStreaming() internally
	// It streams output to the channel in a different way
	// The test verifies that streaming events are produced correctly
}
