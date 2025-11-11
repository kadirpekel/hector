package llms

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

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type OllamaProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
	baseURL    string
}

type OllamaRequest struct {
	Model      string          `json:"model"`
	Messages   []OllamaMessage `json:"messages"`
	Stream     bool            `json:"stream"`
	Format     interface{}     `json:"format,omitempty"` // "json" string or schema object
	Options    *OllamaOptions  `json:"options,omitempty"`
	Tools      []OllamaTool    `json:"tools,omitempty"`
	ToolChoice string          `json:"tool_choice,omitempty"`
	Think      interface{}     `json:"think,omitempty"` // true/false or "low"/"medium"/"high" for GPT-OSS
}

type OllamaMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	Thinking   string           `json:"thinking,omitempty"` // Thinking/reasoning trace
	ToolCalls  []OllamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolName   string           `json:"tool_name,omitempty"` // For tool result messages
}

type OllamaTool struct {
	Type     string             `json:"type"`
	Function OllamaToolFunction `json:"function"`
}

type OllamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OllamaToolCall struct {
	Type     string                 `json:"type"` // Should be "function"
	Function OllamaToolCallFunction `json:"function"`
}

type OllamaToolCallFunction struct {
	Index     int                    `json:"index,omitempty"` // Index for parallel tool calls
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // Max tokens
}

type OllamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration"`
	LoadDuration       int64         `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration int64         `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       int64         `json:"eval_duration"`
	Error              string        `json:"error,omitempty"`
}

type OllamaStreamChunk struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration"`
	LoadDuration       int64         `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration int64         `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       int64         `json:"eval_duration"`
	Error              string        `json:"error,omitempty"`
}

func NewOllamaProviderFromConfig(cfg *config.LLMProviderConfig) (*OllamaProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	baseURL := cfg.Host
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &OllamaProvider{
		config:     cfg,
		httpClient: createHTTPClient(cfg),
		baseURL:    baseURL,
	}, nil
}

func (p *OllamaProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	// Create span for LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "ollama"),
			attribute.Bool("streaming", false),
		),
	)
	defer span.End()

	request := p.buildRequest(messages, false, tools, nil)

	response, err := p.makeRequest(ctx, request)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}

		return "", nil, 0, err
	}

	if response.Error != "" {
		apiErr := fmt.Errorf("Ollama API error: %s", response.Error)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error)

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, apiErr)
		}

		return "", nil, 0, apiErr
	}

	text := response.Message.Content
	tokensUsed := response.PromptEvalCount + response.EvalCount

	var toolCalls []*protocol.ToolCall
	if len(response.Message.ToolCalls) > 0 {
		toolCalls = p.parseToolCalls(response.Message.ToolCalls)
	}

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.PromptEvalCount),
		attribute.Int(observability.AttrLLMTokensOutput, response.EvalCount),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.PromptEvalCount, response.EvalCount, nil)
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *OllamaProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	request := p.buildRequest(messages, true, tools, nil)

	outputCh := make(chan StreamChunk, 100)

	go func() {
		defer close(outputCh)

		if err := p.makeStreamingRequest(ctx, request, outputCh); err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
		}
	}()

	return outputCh, nil
}

func (p *OllamaProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "ollama"),
			attribute.Bool("streaming", false),
			attribute.Bool("structured", true),
		),
	)
	defer span.End()

	// Build system prompt with schema if provided
	systemPrompt := p.buildSystemPromptWithSchema(structConfig)
	if systemPrompt != "" {
		// Add system prompt as first message
		systemMsg := &pb.Message{
			Role: pb.Role_ROLE_UNSPECIFIED,
			Parts: []*pb.Part{
				{Part: &pb.Part_Text{Text: systemPrompt}},
			},
		}
		messages = append([]*pb.Message{systemMsg}, messages...)
	}

	request := p.buildRequest(messages, false, tools, structConfig)

	response, err := p.makeRequest(ctx, request)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}

		return "", nil, 0, err
	}

	if response.Error != "" {
		apiErr := fmt.Errorf("Ollama API error: %s", response.Error)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error)

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, apiErr)
		}

		return "", nil, 0, apiErr
	}

	text := response.Message.Content
	tokensUsed := response.PromptEvalCount + response.EvalCount

	var toolCalls []*protocol.ToolCall
	if len(response.Message.ToolCalls) > 0 {
		toolCalls = p.parseToolCalls(response.Message.ToolCalls)
	}

	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.PromptEvalCount),
		attribute.Int(observability.AttrLLMTokensOutput, response.EvalCount),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.PromptEvalCount, response.EvalCount, nil)
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *OllamaProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	// Build system prompt with schema if provided
	systemPrompt := p.buildSystemPromptWithSchema(structConfig)
	if systemPrompt != "" {
		systemMsg := &pb.Message{
			Role: pb.Role_ROLE_UNSPECIFIED,
			Parts: []*pb.Part{
				{Part: &pb.Part_Text{Text: systemPrompt}},
			},
		}
		messages = append([]*pb.Message{systemMsg}, messages...)
	}

	request := p.buildRequest(messages, true, tools, structConfig)

	outputCh := make(chan StreamChunk, 100)

	go func() {
		defer close(outputCh)

		if err := p.makeStreamingRequest(ctx, request, outputCh); err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
		}
	}()

	return outputCh, nil
}

func (p *OllamaProvider) SupportsStructuredOutput() bool {
	return true
}

func (p *OllamaProvider) GetModelName() string {
	return p.config.Model
}

func (p *OllamaProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

func (p *OllamaProvider) GetTemperature() float64 {
	return p.config.Temperature
}

func (p *OllamaProvider) Close() error {
	return nil
}

func (p *OllamaProvider) buildRequest(messages []*pb.Message, stream bool, tools []ToolDefinition, structConfig *StructuredOutputConfig) OllamaRequest {
	ollamaMessages := make([]OllamaMessage, 0, len(messages))
	// Track tool call IDs to tool names for mapping tool results
	toolCallIDToName := make(map[string]string)

	for _, msg := range messages {
		// Skip system messages (they'll be handled separately or converted)
		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				// Convert system message to user message with system prefix
				ollamaMessages = append(ollamaMessages, OllamaMessage{
					Role:    "user",
					Content: fmt.Sprintf("System: %s", textContent),
				})
			}
			continue
		}

		// Handle tool results - Ollama uses tool_name instead of tool_call_id
		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {
			for _, tr := range toolResults {
				content := tr.Content
				if tr.Error != "" {
					content = fmt.Sprintf("Error: %s", tr.Error)
				}
				// Look up tool name from tool call ID mapping
				toolName := toolCallIDToName[tr.ToolCallID]
				if toolName == "" {
					// Fallback: use tool_call_id as tool_name if mapping not found
					toolName = tr.ToolCallID
				}
				ollamaMessages = append(ollamaMessages, OllamaMessage{
					Role:     "tool",
					Content:  content,
					ToolName: toolName, // Ollama uses tool_name field
				})
			}
			continue
		}

		// Handle regular messages
		textContent := protocol.ExtractTextFromMessage(msg)
		role := roleToOllama(msg.Role)

		ollamaMsg := OllamaMessage{
			Role:    role,
			Content: textContent,
		}

		// Handle tool calls from assistant
		toolCalls := protocol.GetToolCallsFromMessage(msg)
		if len(toolCalls) > 0 {
			ollamaMsg.ToolCalls = make([]OllamaToolCall, len(toolCalls))
			for i, tc := range toolCalls {
				args := tc.Args
				if args == nil {
					args = make(map[string]interface{})
				}
				// Track tool call ID to name mapping for tool results
				toolCallIDToName[tc.ID] = tc.Name
				ollamaMsg.ToolCalls[i] = OllamaToolCall{
					Type: "function",
					Function: OllamaToolCallFunction{
						Index:     i, // Ollama uses index for parallel tool calls
						Name:      tc.Name,
						Arguments: args,
					},
				}
			}
		}

		ollamaMessages = append(ollamaMessages, ollamaMsg)
	}

	request := OllamaRequest{
		Model:    p.config.Model,
		Messages: ollamaMessages,
		Stream:   stream,
	}

	// Add Options only if we have meaningful values
	// SetDefaults ensures Temperature and MaxTokens have defaults, so we'll always have values
	// But we should still check to avoid sending empty Options
	if p.config.Temperature > 0 || p.config.MaxTokens > 0 {
		opts := &OllamaOptions{}
		if p.config.Temperature > 0 {
			opts.Temperature = p.config.Temperature
		}
		if p.config.MaxTokens > 0 {
			opts.NumPredict = p.config.MaxTokens
		}
		// Only add Options if at least one field is set
		if opts.Temperature > 0 || opts.NumPredict > 0 {
			request.Options = opts
		}
	}

	// Enable thinking for known thinking-capable models
	// Models that don't support it will ignore this field or return an error
	if p.isThinkingCapableModel(p.config.Model) {
		request.Think = true
	}

	// Set format for structured output
	if structConfig != nil && structConfig.Format == "json" {
		if structConfig.Schema != nil {
			// Use schema object format (Ollama supports both string and object)
			request.Format = structConfig.Schema
		} else {
			// Use simple "json" string format
			request.Format = "json"
		}
	}

	// Add tools if provided
	// Note: Some models (like deepseek-r1:8b) don't support tools
	// We'll send them anyway and let Ollama return an error if unsupported
	if len(tools) > 0 {
		request.Tools = p.convertToOllamaTools(tools)
		request.ToolChoice = "auto"
	}

	return request
}

// isThinkingCapableModel checks if a model name indicates it supports thinking
func (p *OllamaProvider) isThinkingCapableModel(modelName string) bool {
	modelLower := strings.ToLower(modelName)
	// Check for known thinking-capable model patterns
	// Note: Not all variants support thinking (e.g., qwen3-coder:30b doesn't support thinking)
	thinkingModels := []string{
		"qwen3",       // Qwen3 base models support thinking
		"deepseek-r1", // DeepSeek R1 models support thinking
		"deepseek-v3", // DeepSeek V3 models support thinking
		"gpt-oss",     // GPT-OSS supports thinking
	}
	// Exclude models that don't support thinking despite matching patterns
	excludedModels := []string{
		"qwen3-coder", // Qwen3-coder variants don't support thinking
		"qwen2-coder", // Qwen2-coder variants don't support thinking
	}

	// Check exclusions first
	for _, excluded := range excludedModels {
		if strings.Contains(modelLower, excluded) {
			return false
		}
	}

	// Check if model matches thinking-capable patterns
	for _, pattern := range thinkingModels {
		if strings.Contains(modelLower, pattern) {
			return true
		}
	}
	return false
}

func (p *OllamaProvider) convertToOllamaTools(tools []ToolDefinition) []OllamaTool {
	result := make([]OllamaTool, len(tools))
	for i, tool := range tools {
		result[i] = OllamaTool{
			Type: "function",
			Function: OllamaToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

func (p *OllamaProvider) parseToolCalls(ollamaToolCalls []OllamaToolCall) []*protocol.ToolCall {
	toolCalls := make([]*protocol.ToolCall, 0, len(ollamaToolCalls))
	for i, tc := range ollamaToolCalls {
		args := tc.Function.Arguments
		if args == nil {
			args = make(map[string]interface{})
		}
		// Generate a unique ID for the tool call
		// Use index if available, otherwise generate timestamp-based ID
		var toolCallID string
		if tc.Function.Index >= 0 {
			toolCallID = fmt.Sprintf("call_%d_%s", tc.Function.Index, tc.Function.Name)
		} else {
			toolCallID = fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), i)
		}
		toolCalls = append(toolCalls, &protocol.ToolCall{
			ID:   toolCallID,
			Name: tc.Function.Name,
			Args: args,
		})
	}
	return toolCalls
}

func (p *OllamaProvider) makeRequest(ctx context.Context, request OllamaRequest) (*OllamaResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (p *OllamaProvider) makeStreamingRequest(ctx context.Context, request OllamaRequest, outputCh chan<- StreamChunk) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	// HTTP client may return both response and error for non-2xx status codes
	// We need to check the response body even if there's an error
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			errorBody := string(bodyBytes)
			if readErr != nil {
				errorBody = fmt.Sprintf("(failed to read error body: %v)", readErr)
			}
			// Try to extract error message from JSON if present
			var errorJSON struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(bodyBytes, &errorJSON) == nil && errorJSON.Error != "" {
				return fmt.Errorf("Ollama API error: %s", errorJSON.Error)
			}
			return fmt.Errorf("Ollama API request failed with status %d: %s", resp.StatusCode, errorBody)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to make streaming request: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("failed to make streaming request: no response received")
	}

	reader := bufio.NewReader(resp.Body)
	// Track tool calls by index for accumulation
	toolCallsMap := make(map[int]*OllamaToolCall)
	var totalTokens int

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var chunk OllamaStreamChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Error != "" {
			return fmt.Errorf("Ollama API error: %s", chunk.Error)
		}

		// Accumulate text content
		if chunk.Message.Content != "" {
			outputCh <- StreamChunk{
				Type: "text",
				Text: chunk.Message.Content,
			}
		}

		// Handle thinking/reasoning trace (Ollama thinking capability)
		if chunk.Message.Thinking != "" {
			// Emit thinking as thinking chunks - will be converted to thinking parts
			outputCh <- StreamChunk{
				Type: "thinking",
				Text: chunk.Message.Thinking,
			}
		}

		// Handle tool calls - accumulate by index
		if len(chunk.Message.ToolCalls) > 0 {
			for _, tc := range chunk.Message.ToolCalls {
				idx := tc.Function.Index
				if idx < 0 {
					// If no index, use the map size as index
					idx = len(toolCallsMap)
				}
				// Update or create tool call entry
				if existing, exists := toolCallsMap[idx]; exists {
					// Merge: update arguments if provided, keep name
					if len(tc.Function.Arguments) > 0 {
						// Merge arguments (Ollama might send partial updates)
						for k, v := range tc.Function.Arguments {
							existing.Function.Arguments[k] = v
						}
					}
				} else {
					// Create new entry
					toolCallsMap[idx] = &tc
				}
			}
		}

		// Update token count and check if done
		if chunk.Done {
			totalTokens = chunk.PromptEvalCount + chunk.EvalCount

			// Send accumulated tool calls when done
			if len(toolCallsMap) > 0 {
				// Convert map to slice in index order
				var accumulatedToolCalls []OllamaToolCall
				for i := 0; i < len(toolCallsMap); i++ {
					if tc, exists := toolCallsMap[i]; exists {
						accumulatedToolCalls = append(accumulatedToolCalls, *tc)
					}
				}
				if len(accumulatedToolCalls) > 0 {
					toolCalls := p.parseToolCalls(accumulatedToolCalls)
					for _, tc := range toolCalls {
						outputCh <- StreamChunk{
							Type:     "tool_call",
							ToolCall: tc,
						}
					}
				}
			}

			outputCh <- StreamChunk{
				Type:   "done",
				Tokens: totalTokens,
			}
			break
		}
	}

	return nil
}

func (p *OllamaProvider) buildSystemPromptWithSchema(structConfig *StructuredOutputConfig) string {
	if structConfig == nil || structConfig.Schema == nil {
		return ""
	}

	schemaJSON, err := json.MarshalIndent(structConfig.Schema, "", "  ")
	if err != nil {
		return ""
	}

	return fmt.Sprintf(`You must respond with valid JSON matching this exact schema:

%s

Important:
- Output ONLY valid JSON, no other text
- All required fields must be present
- Follow the exact structure specified
- Use correct data types for each field`, string(schemaJSON))
}

func roleToOllama(role pb.Role) string {
	switch role {
	case pb.Role_ROLE_USER:
		return "user"
	case pb.Role_ROLE_AGENT:
		return "assistant"
	case pb.Role_ROLE_UNSPECIFIED:
		return "system"
	default:
		return "user"
	}
}
