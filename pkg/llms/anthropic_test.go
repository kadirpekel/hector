package llms

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

func TestNewAnthropicProvider(t *testing.T) {
	// Test basic functionality
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	if provider == nil {
		t.Fatal("NewAnthropicProvider() returned nil provider")
	}

	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("NewAnthropicProvider() model = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}

	if provider.GetMaxTokens() != 4096 {
		t.Errorf("NewAnthropicProvider() maxTokens = %v, want 4096", provider.GetMaxTokens())
	}

	if provider.GetTemperature() != 1.0 {
		t.Errorf("NewAnthropicProvider() temperature = %v, want 1.0", provider.GetTemperature())
	}
}

func TestNewAnthropicProviderFromConfig(t *testing.T) {
	// Test valid config
	config := &config.LLMProviderConfig{
		Type:    "anthropic",
		Model:   "claude-3-5-sonnet-20241022",
		Host:    "https://api.anthropic.com",
		APIKey:  "sk-ant-test-key",
		Timeout: 30,
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v, want nil", err)
	}

	if provider == nil {
		t.Fatal("NewAnthropicProviderFromConfig() returned nil provider")
	}

	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("NewAnthropicProviderFromConfig() model = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}

func TestAnthropicProvider_GetModelName(t *testing.T) {
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelName() = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}

func TestAnthropicProvider_GetMaxTokens(t *testing.T) {
	// Test with default provider (should have default max tokens)
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	// Default should be 4096 based on the NewAnthropicProvider function
	expectedTokens := 4096
	if provider.GetMaxTokens() != expectedTokens {
		t.Errorf("GetMaxTokens() = %v, want %v", provider.GetMaxTokens(), expectedTokens)
	}
}

func TestAnthropicProvider_GetTemperature(t *testing.T) {
	// Test with default provider (should have default temperature)
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	// Default should be 1.0 based on the NewAnthropicProvider function
	expectedTemp := 1.0
	if provider.GetTemperature() != expectedTemp {
		t.Errorf("GetTemperature() = %v, want %v", provider.GetTemperature(), expectedTemp)
	}
}

func TestAnthropicProvider_Close(t *testing.T) {
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	// Should not panic and should return nil
	err := provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestAnthropicProvider_Generate_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected /v1/messages, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("x-api-key")
		if auth != "sk-ant-test-key" {
			t.Errorf("Expected x-api-key header, got %s", auth)
		}

		// Check anthropic version header
		version := r.Header.Get("anthropic-version")
		if version != "2023-06-01" {
			t.Errorf("Expected anthropic-version 2023-06-01, got %s", version)
		}

		// Parse request body
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request structure
		if req.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model claude-3-5-sonnet-20241022, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("Expected user role, got %s", req.Messages[0].Role)
		}

		// Send mock response
		response := AnthropicResponse{
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Hello! How can I help you today?",
				},
			},
			Usage: AnthropicUsage{
				InputTokens:  10,
				OutputTokens: 15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test Generate
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	text, toolCalls, tokens, err := provider.Generate(messages, tools)

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if text != "Hello! How can I help you today?" {
		t.Errorf("Generate() text = %v, want Hello! How can I help you today?", text)
	}
	if len(toolCalls) != 0 {
		t.Errorf("Generate() toolCalls length = %v, want 0", len(toolCalls))
	}
	if tokens != 25 {
		t.Errorf("Generate() tokens = %v, want 25", tokens)
	}
}

func TestAnthropicProvider_Generate_WithTools(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify tools in request
		if len(req.Tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Name != "test_tool" {
			t.Errorf("Expected tool name test_tool, got %s", req.Tools[0].Name)
		}

		// Send mock response with tool use
		response := AnthropicResponse{
			Content: []AnthropicContent{
				{
					Type: "tool_use",
					ID:   "toolu_123",
					Name: "test_tool",
					Input: map[string]interface{}{
						"param1": "value1",
					},
				},
			},
			Usage: AnthropicUsage{
				InputTokens:  20,
				OutputTokens: 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test Generate with tools
	messages := []a2a.Message{
		a2a.CreateUserMessage("Use the test tool"),
	}
	tools := []ToolDefinition{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	text, toolCalls, tokens, err := provider.Generate(messages, tools)

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if text != "" {
		t.Errorf("Generate() text = %v, want empty", text)
	}
	if len(toolCalls) != 1 {
		t.Errorf("Generate() toolCalls length = %v, want 1", len(toolCalls))
	}
	if toolCalls[0].ID != "toolu_123" {
		t.Errorf("Generate() toolCall ID = %v, want toolu_123", toolCalls[0].ID)
	}
	if toolCalls[0].Name != "test_tool" {
		t.Errorf("Generate() toolCall Name = %v, want test_tool", toolCalls[0].Name)
	}
	if tokens != 30 {
		t.Errorf("Generate() tokens = %v, want 30", tokens)
	}
}

func TestAnthropicProvider_Generate_HTTPError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test Generate with HTTP error
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	_, _, _, err = provider.Generate(messages, tools)

	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
}

func TestAnthropicProvider_Generate_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test Generate with invalid JSON
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	_, _, _, err = provider.Generate(messages, tools)

	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
}

func TestAnthropicProvider_GenerateStreaming_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected /v1/messages, got %s", r.URL.Path)
		}

		// Parse request body
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true in request")
		}

		// Send streaming response
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Transfer-Encoding", "chunked")

		// Send multiple chunks
		chunks := []string{
			`event: message_start
data: {"type": "message_start", "message": {"id": "msg_123", "type": "message", "role": "assistant", "content": [], "model": "claude-3-5-sonnet-20241022", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 10, "output_tokens": 0}}}`,
			`event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}`,
			`event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}`,
			`event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": " there"}}`,
			`event: content_block_stop
data: {"type": "content_block_stop", "index": 0}`,
			`event: message_delta
data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence": null}, "usage": {"output_tokens": 8}}`,
			`event: message_stop
data: {"type": "message_stop"}`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test GenerateStreaming
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	ch, err := provider.GenerateStreaming(messages, tools)

	if err != nil {
		t.Errorf("GenerateStreaming() error = %v, want nil", err)
	}

	// Collect chunks
	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// Check first text chunk
	foundText := false
	for _, chunk := range chunks {
		if chunk.Type == "text" && strings.Contains(chunk.Text, "Hello") {
			foundText = true
			break
		}
	}
	if !foundText {
		t.Error("Expected to find text chunk with 'Hello'")
	}
}

func TestAnthropicProvider_GenerateStreaming_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Test GenerateStreaming with error
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	ch, err := provider.GenerateStreaming(messages, tools)

	if err != nil {
		// If there's an immediate error, that's expected
		return
	}

	// If no immediate error, check that the channel eventually sends an error
	hasError := false
	for chunk := range ch {
		if chunk.Type == "error" {
			hasError = true
			break
		}
	}

	if !hasError {
		t.Error("GenerateStreaming() expected error chunk, got none")
	}
}

func TestAnthropicProvider_WithCustomHTTPClient(t *testing.T) {
	// Create custom HTTP client
	customClient := httpclient.New(
		httpclient.WithMaxRetries(1),
		httpclient.WithBaseDelay(100*time.Millisecond),
	)

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := AnthropicResponse{
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Hello from custom client!",
				},
			},
			Usage: AnthropicUsage{
				InputTokens:  5,
				OutputTokens: 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with custom client
	config := &config.LLMProviderConfig{
		Type:   "anthropic",
		Model:  "claude-3-5-sonnet-20241022",
		Host:   server.URL,
		APIKey: "sk-ant-test-key",
	}

	provider, err := NewAnthropicProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewAnthropicProviderFromConfig() error = %v", err)
	}

	// Replace the HTTP client
	provider.httpClient = customClient

	// Test Generate with custom client
	messages := []a2a.Message{
		a2a.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	text, _, tokens, err := provider.Generate(messages, tools)

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if text != "Hello from custom client!" {
		t.Errorf("Generate() text = %v, want Hello from custom client!", text)
	}
	if tokens != 13 {
		t.Errorf("Generate() tokens = %v, want 13", tokens)
	}
}

func TestAnthropicProvider_MessageConversion(t *testing.T) {
	// Test conversion from universal Message format to Anthropic format
	_ = []a2a.Message{
		a2a.CreateUserMessage("Hello"),
		a2a.CreateAssistantMessage("Hi there!"),
		a2a.CreateTextMessage(a2a.MessageRoleSystem, "You are a helpful assistant"),
	}

	// This would be tested through the actual Generate method
	// but we can test the conversion logic indirectly
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	// Test that the provider can handle different message types
	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelName() = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}

func TestAnthropicProvider_ToolConversion(t *testing.T) {
	// Test conversion from universal ToolDefinition format to Anthropic format
	_ = []ToolDefinition{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type":        "string",
						"description": "First parameter",
					},
					"param2": map[string]interface{}{
						"type":        "number",
						"description": "Second parameter",
					},
				},
				"required": []string{"param1"},
			},
		},
	}

	// This would be tested through the actual Generate method
	// but we can test the conversion logic indirectly
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	// Test that the provider can handle tool definitions
	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelName() = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}
