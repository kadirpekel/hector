package llms

import (
	"encoding/json"
	"github.com/kadirpekel/hector/pkg/a2a"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

func TestNewOpenAIProvider(t *testing.T) {
	// Test basic functionality
	provider := NewOpenAIProvider("sk-test-key", "gpt-4o")

	if provider == nil {
		t.Fatal("NewOpenAIProvider() returned nil provider")
	}

	if provider.GetModelName() != "gpt-4o" {
		t.Errorf("NewOpenAIProvider() model = %v, want gpt-4o", provider.GetModelName())
	}

	if provider.GetMaxTokens() != 1000 {
		t.Errorf("NewOpenAIProvider() maxTokens = %v, want 1000", provider.GetMaxTokens())
	}

	if provider.GetTemperature() != 0.7 {
		t.Errorf("NewOpenAIProvider() temperature = %v, want 0.7", provider.GetTemperature())
	}
}

func TestNewOpenAIProviderFromConfig(t *testing.T) {
	// Test valid config
	config := &config.LLMProviderConfig{
		Type:    "openai",
		Model:   "gpt-4o",
		Host:    "https://api.openai.com/v1",
		APIKey:  "sk-test-key",
		Timeout: 30,
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v, want nil", err)
	}

	if provider == nil {
		t.Fatal("NewOpenAIProviderFromConfig() returned nil provider")
	}

	if provider.GetModelName() != "gpt-4o" {
		t.Errorf("NewOpenAIProviderFromConfig() model = %v, want gpt-4o", provider.GetModelName())
	}
}

func TestOpenAIProvider_GetModelName(t *testing.T) {
	provider := NewOpenAIProvider("sk-test-key", "gpt-4o")

	if provider.GetModelName() != "gpt-4o" {
		t.Errorf("GetModelName() = %v, want gpt-4o", provider.GetModelName())
	}
}

func TestOpenAIProvider_GetMaxTokens(t *testing.T) {
	// Test with default provider (should have default max tokens)
	provider := NewOpenAIProvider("sk-test-key", "gpt-4o")

	// Default should be 1000 based on the NewOpenAIProvider function
	expectedTokens := 1000
	if provider.GetMaxTokens() != expectedTokens {
		t.Errorf("GetMaxTokens() = %v, want %v", provider.GetMaxTokens(), expectedTokens)
	}
}

func TestOpenAIProvider_GetTemperature(t *testing.T) {
	// Test with default provider (should have default temperature)
	provider := NewOpenAIProvider("sk-test-key", "gpt-4o")

	// Default should be 0.7 based on the NewOpenAIProvider function
	expectedTemp := 0.7
	if provider.GetTemperature() != expectedTemp {
		t.Errorf("GetTemperature() = %v, want %v", provider.GetTemperature(), expectedTemp)
	}
}

func TestOpenAIProvider_Close(t *testing.T) {
	provider := NewOpenAIProvider("sk-test-key", "gpt-4o")

	// Should not panic and should return nil
	err := provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestOpenAIProvider_Generate_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer sk-test-key") {
			t.Errorf("Expected Bearer token, got %s", auth)
		}

		// Parse request body
		var req OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request structure
		if req.Model != "gpt-4o" {
			t.Errorf("Expected model gpt-4o, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("Expected user role, got %s", req.Messages[0].Role)
		}

		// Send mock response
		content := "Hello! How can I help you today?"
		response := OpenAIResponse{
			Choices: []Choice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: &content,
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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

func TestOpenAIProvider_Generate_WithTools(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify tools in request
		if len(req.Tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "test_tool" {
			t.Errorf("Expected tool name test_tool, got %s", req.Tools[0].Function.Name)
		}

		// Send mock response with tool call
		emptyContent := ""
		response := OpenAIResponse{
			Choices: []Choice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: &emptyContent,
						ToolCalls: []OpenAIToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: OpenAIFunctionCall{
									Name:      "test_tool",
									Arguments: `{"param1": "value1"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: Usage{
				PromptTokens:     20,
				CompletionTokens: 10,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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
	if toolCalls[0].ID != "call_123" {
		t.Errorf("Generate() toolCall ID = %v, want call_123", toolCalls[0].ID)
	}
	if toolCalls[0].Name != "test_tool" {
		t.Errorf("Generate() toolCall Name = %v, want test_tool", toolCalls[0].Name)
	}
	if tokens != 30 {
		t.Errorf("Generate() tokens = %v, want 30", tokens)
	}
}

func TestOpenAIProvider_Generate_HTTPError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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

func TestOpenAIProvider_Generate_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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

func TestOpenAIProvider_GenerateStreaming_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
		}

		// Parse request body
		var req OpenAIRequest
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
			`data: {"choices":[{"delta":{"role":"assistant"}}]}`,
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" there"}}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`data: {"usage":{"prompt_tokens":10,"completion_tokens":8,"total_tokens":18}}`,
			"data: [DONE]",
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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

func TestOpenAIProvider_GenerateStreaming_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	// Create provider with test server
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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

func TestOpenAIProvider_WithCustomHTTPClient(t *testing.T) {
	// Create custom HTTP client
	customClient := httpclient.New(
		httpclient.WithMaxRetries(1),
		httpclient.WithBaseDelay(100*time.Millisecond),
	)

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := "Hello from custom client!"
		response := OpenAIResponse{
			Choices: []Choice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: &content,
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     5,
				CompletionTokens: 8,
				TotalTokens:      13,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with custom client
	config := &config.LLMProviderConfig{
		Type:   "openai",
		Model:  "gpt-4o",
		Host:   server.URL,
		APIKey: "sk-test-key",
	}

	provider, err := NewOpenAIProviderFromConfig(config)
	if err != nil {
		t.Fatalf("NewOpenAIProviderFromConfig() error = %v", err)
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
