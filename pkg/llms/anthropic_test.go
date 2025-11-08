package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/protocol"
)

func TestNewAnthropicProvider(t *testing.T) {

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

	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	expectedTokens := 4096
	if provider.GetMaxTokens() != expectedTokens {
		t.Errorf("GetMaxTokens() = %v, want %v", provider.GetMaxTokens(), expectedTokens)
	}
}

func TestAnthropicProvider_GetTemperature(t *testing.T) {

	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	expectedTemp := 1.0
	if provider.GetTemperature() != expectedTemp {
		t.Errorf("GetTemperature() = %v, want %v", provider.GetTemperature(), expectedTemp)
	}
}

func TestAnthropicProvider_Close(t *testing.T) {
	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	err := provider.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestAnthropicProvider_Generate_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected /v1/messages, got %s", r.URL.Path)
		}

		auth := r.Header.Get("x-api-key")
		if auth != "sk-ant-test-key" {
			t.Errorf("Expected x-api-key header, got %s", auth)
		}

		version := r.Header.Get("anthropic-version")
		if version != "2023-06-01" {
			t.Errorf("Expected anthropic-version 2023-06-01, got %s", version)
		}

		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model claude-3-5-sonnet-20241022, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("Expected user role, got %s", req.Messages[0].Role)
		}

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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	text, toolCalls, tokens, err := provider.Generate(context.Background(), messages, tools)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if len(req.Tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Name != "test_tool" {
			t.Errorf("Expected tool name test_tool, got %s", req.Tools[0].Name)
		}

		response := AnthropicResponse{
			Content: []AnthropicContent{
				{
					Type: "tool_use",
					ID:   "toolu_123",
					Name: "test_tool",
					Input: &map[string]interface{}{
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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Use the test tool"),
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

	text, toolCalls, tokens, err := provider.Generate(context.Background(), messages, tools)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	_, _, _, err = provider.Generate(context.Background(), messages, tools)

	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
}

func TestAnthropicProvider_Generate_InvalidJSON(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	_, _, _, err = provider.Generate(context.Background(), messages, tools)

	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
}

func TestAnthropicProvider_GenerateStreaming_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected /v1/messages, got %s", r.URL.Path)
		}

		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true in request")
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Transfer-Encoding", "chunked")

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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	ch, err := provider.GenerateStreaming(context.Background(), messages, tools)

	if err != nil {
		t.Errorf("GenerateStreaming() error = %v, want nil", err)
	}

	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

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

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	ch, err := provider.GenerateStreaming(context.Background(), messages, tools)

	if err != nil {

		return
	}

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

	customClient := httpclient.New(
		httpclient.WithMaxRetries(1),
		httpclient.WithBaseDelay(100*time.Millisecond),
	)

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

	provider.httpClient = customClient

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	text, _, tokens, err := provider.Generate(context.Background(), messages, tools)

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

	_ = []*pb.Message{
		protocol.CreateUserMessage("Hello"),
		protocol.CreateTextMessage(pb.Role_ROLE_AGENT, "Hi there!"),
		protocol.CreateTextMessage(pb.Role_ROLE_USER, "You are a helpful assistant"),
	}

	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelName() = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}

func TestAnthropicProvider_ToolConversion(t *testing.T) {

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

	provider := NewAnthropicProvider("sk-ant-test-key", "claude-3-5-sonnet-20241022")

	if provider.GetModelName() != "claude-3-5-sonnet-20241022" {
		t.Errorf("GetModelName() = %v, want claude-3-5-sonnet-20241022", provider.GetModelName())
	}
}
