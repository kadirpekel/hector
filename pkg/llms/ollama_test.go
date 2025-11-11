package llms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
)

func TestNewOllamaProviderFromConfig(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:        "ollama",
		Model:       "llama3.2",
		Host:        "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v, want nil", err)
	}

	if provider == nil {
		t.Fatal("NewOllamaProviderFromConfig() returned nil provider")
	}

	if provider.GetModelName() != "llama3.2" {
		t.Errorf("GetModelName() = %v, want llama3.2", provider.GetModelName())
	}

	if provider.GetMaxTokens() != 2000 {
		t.Errorf("GetMaxTokens() = %v, want 2000", provider.GetMaxTokens())
	}

	if provider.GetTemperature() != 0.7 {
		t.Errorf("GetTemperature() = %v, want 0.7", provider.GetTemperature())
	}
}

func TestOllamaProvider_GetModelName(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)
	if provider.GetModelName() != "llama3.2" {
		t.Errorf("GetModelName() = %v, want llama3.2", provider.GetModelName())
	}
}

func TestOllamaProvider_GetMaxTokens(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:      "ollama",
		Model:     "llama3.2",
		Host:      "http://localhost:11434",
		MaxTokens: 4000,
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)
	if provider.GetMaxTokens() != 4000 {
		t.Errorf("GetMaxTokens() = %v, want 4000", provider.GetMaxTokens())
	}
}

func TestOllamaProvider_GetTemperature(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:        "ollama",
		Model:       "llama3.2",
		Host:        "http://localhost:11434",
		Temperature: 0.9,
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)
	if provider.GetTemperature() != 0.9 {
		t.Errorf("GetTemperature() = %v, want 0.9", provider.GetTemperature())
	}
}

func TestOllamaProvider_Close(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)
	if err := provider.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestOllamaProvider_SupportsStructuredOutput(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)
	if !provider.SupportsStructuredOutput() {
		t.Error("SupportsStructuredOutput() = false, want true")
	}
}

func TestOllamaProvider_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat, got %s", r.URL.Path)
		}

		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Model != "llama3.2" {
			t.Errorf("Expected model llama3.2, got %s", req.Model)
		}
		if req.Stream {
			t.Error("Expected stream=false for non-streaming request")
		}
		// Note: think field is only set for thinking-capable models when ShowThinking is enabled
		// llama3.2 is not a thinking-capable model, so think field should not be set
		if len(req.Messages) == 0 {
			t.Error("Expected at least one message")
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you today?",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       15,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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

func TestOllamaProvider_Generate_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if len(req.Tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "test_tool" {
			t.Errorf("Expected tool name test_tool, got %s", req.Tools[0].Function.Name)
		}
		if req.ToolChoice != "auto" {
			t.Errorf("Expected tool_choice=auto, got %s", req.ToolChoice)
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "",
				ToolCalls: []OllamaToolCall{
					{
						Type: "function",
						Function: OllamaToolCallFunction{
							Index: 0,
							Name:  "test_tool",
							Arguments: map[string]interface{}{
								"param1": "value1",
							},
						},
					},
				},
			},
			Done:            true,
			PromptEvalCount: 20,
			EvalCount:       10,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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
	if toolCalls[0].Name != "test_tool" {
		t.Errorf("Generate() toolCall Name = %v, want test_tool", toolCalls[0].Name)
	}
	if tokens != 30 {
		t.Errorf("Generate() tokens = %v, want 30", tokens)
	}
}

func TestOllamaProvider_Generate_WithToolResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Check that tool result is properly formatted
		foundToolResult := false
		for _, msg := range req.Messages {
			if msg.Role == "tool" && msg.ToolName != "" {
				foundToolResult = true
				if msg.Content == "" {
					t.Error("Tool result message should have content")
				}
			}
		}

		if !foundToolResult {
			t.Error("Expected tool result message in request")
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "The result is value1",
			},
			Done:            true,
			PromptEvalCount: 30,
			EvalCount:       10,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	// Create messages with tool call and tool result
	toolCall := &protocol.ToolCall{
		ID:   "call_0_test_tool",
		Name: "test_tool",
		Args: map[string]interface{}{"param1": "value1"},
	}
	toolResult := &protocol.ToolResult{
		ToolCallID: "call_0_test_tool",
		Content:    "result_value",
		Error:      "",
	}
	messages := []*pb.Message{
		protocol.CreateUserMessage("Use the test tool"),
		{
			Role:  pb.Role_ROLE_AGENT,
			Parts: []*pb.Part{protocol.CreateToolCallPart(toolCall)},
		},
		{
			Role:  pb.Role_ROLE_USER,
			Parts: []*pb.Part{protocol.CreateToolResultPart(toolResult)},
		},
	}

	text, _, _, err := provider.Generate(context.Background(), messages, []ToolDefinition{})

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if !strings.Contains(text, "result") {
		t.Errorf("Generate() text should contain 'result', got %v", text)
	}
}

func TestOllamaProvider_GenerateStreaming_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true in request")
		}

		w.Header().Set("Content-Type", "application/json")

		chunks := []string{
			`{"model":"llama3.2","message":{"role":"assistant","content":"Hello"},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant","content":" there"},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant","content":"!"},"done":true,"prompt_eval_count":10,"eval_count":8}`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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

	if len(chunks) < 3 {
		t.Errorf("Expected at least 3 chunks, got %d", len(chunks))
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

	// Check for done chunk with tokens
	foundDone := false
	for _, chunk := range chunks {
		if chunk.Type == "done" && chunk.Tokens > 0 {
			foundDone = true
			break
		}
	}
	if !foundDone {
		t.Error("Expected to find done chunk with tokens")
	}
}

func TestOllamaProvider_GenerateStreaming_WithThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Think != true {
			t.Error("Expected think=true for thinking-capable model")
		}

		w.Header().Set("Content-Type", "application/json")

		chunks := []string{
			`{"model":"qwen3","message":{"role":"assistant","thinking":"Let me think about this..."},"done":false}`,
			`{"model":"qwen3","message":{"role":"assistant","thinking":" I need to calculate..."},"done":false}`,
			`{"model":"qwen3","message":{"role":"assistant","content":"The answer is 42"},"done":true,"prompt_eval_count":10,"eval_count":5}`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "qwen3",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("What is 6 * 7?"),
	}
	tools := []ToolDefinition{}

	// Set ShowThinking in context to enable thinking for thinking-capable models
	ctx := context.WithValue(context.Background(), protocol.ShowThinkingKey, true)
	ch, err := provider.GenerateStreaming(ctx, messages, tools)

	if err != nil {
		t.Errorf("GenerateStreaming() error = %v, want nil", err)
	}

	var thinkingText string
	var contentText string

	for chunk := range ch {
		if chunk.Type == "thinking" {
			thinkingText += chunk.Text
		} else if chunk.Type == "text" {
			contentText += chunk.Text
		}
	}

	if thinkingText == "" {
		t.Error("Expected to find thinking chunks")
	}
	if !strings.Contains(contentText, "42") {
		t.Errorf("Expected to find answer in content, got %v", contentText)
	}
}

func TestOllamaProvider_GenerateStreaming_WithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		chunks := []string{
			`{"model":"llama3.2","message":{"role":"assistant","tool_calls":[{"type":"function","function":{"index":0,"name":"test_tool","arguments":{"param1":"value1"}}}]},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant","tool_calls":[{"type":"function","function":{"index":0,"name":"test_tool","arguments":{"param2":"value2"}}}]},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant"},"done":true,"prompt_eval_count":20,"eval_count":10}`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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
					"param1": map[string]interface{}{"type": "string"},
					"param2": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	ch, err := provider.GenerateStreaming(context.Background(), messages, tools)

	if err != nil {
		t.Errorf("GenerateStreaming() error = %v, want nil", err)
	}

	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	foundToolCall := false
	for _, chunk := range chunks {
		if chunk.Type == "tool_call" {
			foundToolCall = true
			if chunk.ToolCall.Name != "test_tool" {
				t.Errorf("Expected tool call name test_tool, got %s", chunk.ToolCall.Name)
			}
			// Check that arguments were merged (both param1 and param2)
			if chunk.ToolCall.Args["param1"] == nil || chunk.ToolCall.Args["param2"] == nil {
				t.Error("Expected merged arguments with both param1 and param2")
			}
		}
	}
	if !foundToolCall {
		t.Error("Expected to find tool_call chunk")
	}
}

func TestOllamaProvider_GenerateStructured_JSONString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.Format != "json" {
			t.Errorf("Expected format=json, got %v", req.Format)
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: `{"sentiment":"positive","score":0.95}`,
			},
			Done:            true,
			PromptEvalCount: 15,
			EvalCount:       20,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("Analyze sentiment"),
	}

	structConfig := &StructuredOutputConfig{
		Format: "json",
	}

	text, _, tokens, err := provider.GenerateStructured(context.Background(), messages, []ToolDefinition{}, structConfig)

	if err != nil {
		t.Errorf("GenerateStructured() error = %v, want nil", err)
	}
	if !strings.Contains(text, "sentiment") {
		t.Errorf("GenerateStructured() text should contain 'sentiment', got %v", text)
	}
	if tokens != 35 {
		t.Errorf("GenerateStructured() tokens = %v, want 35", tokens)
	}
}

func TestOllamaProvider_GenerateStructured_WithSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Format should be a schema object, not a string
		formatMap, ok := req.Format.(map[string]interface{})
		if !ok {
			t.Errorf("Expected format to be a map, got %T", req.Format)
		}
		if formatMap["type"] != "object" {
			t.Errorf("Expected schema type=object, got %v", formatMap["type"])
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: `{"name":"John","age":30}`,
			},
			Done:            true,
			PromptEvalCount: 20,
			EvalCount:       15,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("Create a person object"),
	}

	structConfig := &StructuredOutputConfig{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"age": map[string]interface{}{
					"type": "number",
				},
			},
			"required": []string{"name", "age"},
		},
	}

	text, _, tokens, err := provider.GenerateStructured(context.Background(), messages, []ToolDefinition{}, structConfig)

	if err != nil {
		t.Errorf("GenerateStructured() error = %v, want nil", err)
	}
	if !strings.Contains(text, "name") || !strings.Contains(text, "age") {
		t.Errorf("GenerateStructured() text should contain 'name' and 'age', got %v", text)
	}
	if tokens != 35 {
		t.Errorf("GenerateStructured() tokens = %v, want 35", tokens)
	}
}

func TestOllamaProvider_GenerateStructuredStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if !req.Stream {
			t.Error("Expected stream=true")
		}
		if req.Format == nil {
			t.Error("Expected format to be set")
		}

		w.Header().Set("Content-Type", "application/json")

		chunks := []string{
			`{"model":"llama3.2","message":{"role":"assistant","content":"{\"sentiment\":"},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant","content":"\"positive\""},"done":false}`,
			`{"model":"llama3.2","message":{"role":"assistant","content":"}"},"done":true,"prompt_eval_count":15,"eval_count":10}`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n"))
		}
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("Analyze sentiment"),
	}

	structConfig := &StructuredOutputConfig{
		Format: "json",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sentiment": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	ch, err := provider.GenerateStructuredStreaming(context.Background(), messages, []ToolDefinition{}, structConfig)

	if err != nil {
		t.Errorf("GenerateStructuredStreaming() error = %v, want nil", err)
	}

	var fullText string
	for chunk := range ch {
		if chunk.Type == "text" {
			fullText += chunk.Text
		}
	}

	if !strings.Contains(fullText, "sentiment") {
		t.Errorf("Expected to find 'sentiment' in streamed text, got %v", fullText)
	}
}

func TestOllamaProvider_Generate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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

func TestOllamaProvider_Generate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OllamaResponse{
			Error: "Model not found",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("Hello"),
	}
	tools := []ToolDefinition{}

	_, _, _, err = provider.Generate(context.Background(), messages, tools)

	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Model not found") {
		t.Errorf("Generate() error should contain 'Model not found', got %v", err)
	}
}

func TestOllamaProvider_GenerateStreaming_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
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

func TestOllamaProvider_InterfaceCompliance(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  "http://localhost:11434",
	}

	provider, _ := NewOllamaProviderFromConfig(cfg)

	// Test LLMProvider interface
	var _ LLMProvider = provider

	// Test StructuredOutputProvider interface
	var _ StructuredOutputProvider = provider
}

func TestOllamaProvider_ParallelToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "",
				ToolCalls: []OllamaToolCall{
					{
						Type: "function",
						Function: OllamaToolCallFunction{
							Index: 0,
							Name:  "tool1",
							Arguments: map[string]interface{}{
								"param1": "value1",
							},
						},
					},
					{
						Type: "function",
						Function: OllamaToolCallFunction{
							Index: 1,
							Name:  "tool2",
							Arguments: map[string]interface{}{
								"param2": "value2",
							},
						},
					},
				},
			},
			Done:            true,
			PromptEvalCount: 25,
			EvalCount:       15,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateUserMessage("Use both tools"),
	}
	tools := []ToolDefinition{
		{Name: "tool1", Description: "Tool 1", Parameters: map[string]interface{}{"type": "object"}},
		{Name: "tool2", Description: "Tool 2", Parameters: map[string]interface{}{"type": "object"}},
	}

	_, toolCalls, _, err := provider.Generate(context.Background(), messages, tools)

	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if len(toolCalls) != 2 {
		t.Errorf("Generate() toolCalls length = %v, want 2", len(toolCalls))
	}
	if toolCalls[0].Name != "tool1" {
		t.Errorf("Generate() first toolCall Name = %v, want tool1", toolCalls[0].Name)
	}
	if toolCalls[1].Name != "tool2" {
		t.Errorf("Generate() second toolCall Name = %v, want tool2", toolCalls[1].Name)
	}
}

func TestOllamaProvider_DefaultHost(t *testing.T) {
	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		// Host not set - should use default
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	if provider.baseURL != "http://localhost:11434" {
		t.Errorf("Expected default host http://localhost:11434, got %v", provider.baseURL)
	}
}

func TestOllamaProvider_SystemMessageHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// System messages should be converted to user messages with "System: " prefix
		foundSystemPrefix := false
		for _, msg := range req.Messages {
			if strings.HasPrefix(msg.Content, "System: ") {
				foundSystemPrefix = true
				break
			}
		}

		if !foundSystemPrefix {
			t.Error("Expected system message to be converted to user message with 'System: ' prefix")
		}

		response := OllamaResponse{
			Model: "llama3.2",
			Message: OllamaMessage{
				Role:    "assistant",
				Content: "OK",
			},
			Done:            true,
			PromptEvalCount: 5,
			EvalCount:       2,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.LLMProviderConfig{
		Type:  "ollama",
		Model: "llama3.2",
		Host:  server.URL,
	}

	provider, err := NewOllamaProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewOllamaProviderFromConfig() error = %v", err)
	}

	messages := []*pb.Message{
		protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, "You are a helpful assistant"),
		protocol.CreateUserMessage("Hello"),
	}

	_, _, _, err = provider.Generate(context.Background(), messages, []ToolDefinition{})
	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
}
