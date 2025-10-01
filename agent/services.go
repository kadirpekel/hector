package agent

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/reasoning"
)

const (
	streamBufferSize = 15
	fallbackLabel    = "\n🔧 Working on it..."
)

// ============================================================================
// TOOLS ARE NOW NATIVE EXTENSIONS - NO WRAPPER NEEDED
// ============================================================================

// NoOpContextService provides a no-op implementation when no document stores are configured
type NoOpContextService struct{}

// NewNoOpContextService creates a new no-op context service
func NewNoOpContextService() reasoning.ContextService {
	return &NoOpContextService{}
}

// SearchContext implements reasoning.ContextService
func (s *NoOpContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	// Return empty results when no document stores are configured
	return []databases.SearchResult{}, nil
}

// ExtractSources implements reasoning.ContextService
func (s *NoOpContextService) ExtractSources(context []databases.SearchResult) []string {
	// Return empty sources when no document stores are configured
	return []string{}
}

// ============================================================================
// CONTEXT SERVICE
// ============================================================================

// ContextOptions defines options for context gathering
type ContextOptions struct {
	MaxResults int
	MinScore   float64
}

// DefaultContextService implements reasoning.ContextService
type DefaultContextService struct {
	searchEngine *hectorcontext.SearchEngine
}

// NewContextService creates a new context service
func NewContextService(searchEngine *hectorcontext.SearchEngine) reasoning.ContextService {
	return &DefaultContextService{
		searchEngine: searchEngine,
	}
}

// SearchContext implements reasoning.ContextService
func (s *DefaultContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	if s.searchEngine == nil {
		return []databases.SearchResult{}, nil // Return empty results if no search engine
	}

	// Use search engine to find relevant context
	return s.searchEngine.Search(ctx, query, 5) // Limit to 5 results
}

// ExtractSources implements reasoning.ContextService
func (s *DefaultContextService) ExtractSources(context []databases.SearchResult) []string {
	sources := make([]string, 0, len(context))
	for _, result := range context {
		// Try to get source from metadata, fallback to ID
		if result.Metadata != nil {
			if source, ok := result.Metadata["source"].(string); ok && source != "" {
				sources = append(sources, source)
			} else if result.ID != "" {
				sources = append(sources, result.ID)
			}
		} else if result.ID != "" {
			sources = append(sources, result.ID)
		}
	}
	return sources
}

// ============================================================================
// PROMPT SERVICE
// ============================================================================

// DefaultPromptService implements reasoning.PromptService using composable parts
type DefaultPromptService struct {
	// No dependencies - uses dependency injection through method parameters
}

// NewPromptService creates a new prompt service
func NewPromptService() reasoning.PromptService {
	return &DefaultPromptService{}
}

// BuildDefaultPromptData creates standard PromptData with common fields populated (extensionResults optional)
func (s *DefaultPromptService) BuildDefaultPromptData(ctx context.Context, query string, contextService reasoning.ContextService, historyService reasoning.HistoryService, extensionService reasoning.ExtensionService, extensionResults ...map[string]reasoning.ExtensionResult) (reasoning.PromptData, error) {
	// Gather context (document search)
	context, err := contextService.SearchContext(ctx, query)
	if err != nil {
		// Log warning but continue - context search failure shouldn't stop reasoning
		fmt.Printf("Warning: Could not search context: %v\n", err)
		context = []databases.SearchResult{}
	}

	// Get conversation history
	history := historyService.GetRecentHistory(5)

	// Get available extensions
	availableExtensions := extensionService.GetAvailableExtensions()

	// Handle optional extension results
	var finalExtensionResults map[string]reasoning.ExtensionResult
	if len(extensionResults) > 0 && extensionResults[0] != nil {
		finalExtensionResults = extensionResults[0] // Use first provided extension results
	}

	// Build standard prompt data with optional extension results
	return reasoning.PromptData{
		Query:            query,
		Context:          context,
		Extensions:       availableExtensions,
		ExtensionService: extensionService, // Include extension service for formatting
		History:          history,
		ExtensionResults: finalExtensionResults, // Include extension results if provided
	}, nil
}

// formatPromptData formats PromptData into template-ready map with all standard fields
func (s *DefaultPromptService) formatPromptData(data reasoning.PromptData) map[string]interface{} {
	templateData := map[string]interface{}{
		"Query": data.Query,
	}

	// Format context if present
	if len(data.Context) > 0 {
		var contextFormatted strings.Builder
		contextFormatted.WriteString("Relevant context from documents:\n")
		for i, doc := range data.Context {
			if i >= 5 { // Limit to 5 docs for readability
				break
			}
			content := doc.Content
			if len(content) > 500 { // Reasonable content limit
				content = content[:500] + "..."
			}
			contextFormatted.WriteString(fmt.Sprintf("- Document %d: %s\n", i+1, content))
		}
		templateData["ContextFormatted"] = contextFormatted.String()
	} else {
		templateData["ContextFormatted"] = ""
	}

	// Format extensions if present
	if len(data.Extensions) > 0 {
		// Use the extension service's built-in formatting
		extensionsFormatted := data.ExtensionService.FormatForPrompt()
		templateData["Extensions"] = extensionsFormatted
	} else {
		templateData["Extensions"] = ""
	}

	// Format extension results if present
	if len(data.ExtensionResults) > 0 {
		var extensionResultsFormatted strings.Builder
		extensionResultsFormatted.WriteString("IMPORTANT: The following extensions have ALREADY been executed for this query:\n")

		for name, result := range data.ExtensionResults {
			if result.Success {
				extensionResultsFormatted.WriteString(fmt.Sprintf("✅ %s executed successfully. Output:\n%s\n\n", name, result.Content))
			} else {
				extensionResultsFormatted.WriteString(fmt.Sprintf("❌ %s failed with error: %s\n\n", name, result.Error))
			}
		}

		extensionResultsFormatted.WriteString("Based on these extension execution results, provide a direct response to the user. DO NOT re-execute the same extensions.\n")
		templateData["ExtensionResults"] = extensionResultsFormatted.String()
	} else {
		templateData["ExtensionResults"] = ""
	}

	// Format history if present
	if len(data.History) > 0 {
		var historyFormatted strings.Builder
		historyFormatted.WriteString("Recent conversation history:\n")
		for _, msg := range data.History {
			historyFormatted.WriteString(fmt.Sprintf("- %s: %s\n", msg.Role, msg.Content))
		}
		templateData["History"] = historyFormatted.String()
	} else {
		templateData["History"] = ""
	}

	return templateData
}

// BuildPromptFromParts builds a prompt using composable parts with map (RECOMMENDED)
func (s *DefaultPromptService) BuildPromptFromParts(templateParts map[string]string, data reasoning.PromptData) (string, error) {
	// Extract template parts with sensible defaults
	systemPrompt := templateParts["system"]
	if systemPrompt == "" {
		systemPrompt = "You are a helpful AI assistant."
	}

	instructions := templateParts["instructions"]
	if instructions == "" {
		instructions = "Provide a helpful and accurate response based on the available context and extensions."
	}

	outputFormat := templateParts["output"]
	if outputFormat == "" {
		outputFormat = "Response:"
	}

	// Build the backbone template with customizable parts
	backboneTemplate := `{{.SystemPrompt}}

{{.History}}

{{.ContextFormatted}}

{{.Extensions}}

{{.ExtensionResults}}

{{.Instructions}}

User query: {{.Query}}

{{.OutputFormat}}`

	// Get formatted data using helper method
	templateData := s.formatPromptData(data)

	// Add the custom parts
	templateData["SystemPrompt"] = systemPrompt
	templateData["Instructions"] = instructions
	templateData["OutputFormat"] = outputFormat

	// Parse and execute the backbone template
	tmpl, err := template.New("backbone").Parse(backboneTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse backbone template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("backbone template execution failed: %w", err)
	}

	return buf.String(), nil
}

// BuildPromptWithServices builds a prompt with template parts and auto-populated data (CONVENIENCE)
func (s *DefaultPromptService) BuildPromptWithServices(ctx context.Context, query string, templateParts map[string]string, contextService reasoning.ContextService, historyService reasoning.HistoryService, extensionService reasoning.ExtensionService) (string, error) {
	// Step 1: Build default prompt data automatically
	data, err := s.BuildDefaultPromptData(ctx, query, contextService, historyService, extensionService)
	if err != nil {
		return "", fmt.Errorf("failed to build default prompt data: %w", err)
	}

	// Step 2: Use the consolidated BuildPromptFromParts method
	return s.BuildPromptFromParts(templateParts, data)
}

// BuildDefaultPrompt builds a prompt with default template parts and auto-populated data (SHORTCUT)
func (s *DefaultPromptService) BuildDefaultPrompt(ctx context.Context, query string, contextService reasoning.ContextService, historyService reasoning.HistoryService, extensionService reasoning.ExtensionService, extensionResults ...map[string]reasoning.ExtensionResult) (string, error) {
	// Step 1: Build default prompt data with optional extension results
	data, err := s.BuildDefaultPromptData(ctx, query, contextService, historyService, extensionService, extensionResults...)
	if err != nil {
		return "", fmt.Errorf("failed to build default prompt data: %w", err)
	}

	// Step 2: Use default template parts
	defaultTemplateParts := map[string]string{
		"system":       "You are a helpful AI assistant.",
		"instructions": "Provide a helpful and accurate response. First give a natural response to the user, then use available extensions if you need to gather additional information or perform tasks.",
		"output":       "Response:",
	}

	// Step 3: Build prompt with default parts
	return s.BuildPromptFromParts(defaultTemplateParts, data)
}

// ============================================================================
// LLM SERVICE
// ============================================================================

// DefaultLLMService implements reasoning.LLMService
type DefaultLLMService struct {
	llmProvider          llms.LLMProvider
	lastRawResponse      string // Store last raw response for extension parsing
	extensionService     reasoning.ExtensionService
	lastExtensionCalls   []reasoning.ExtensionCall
	lastExtensionResults map[string]reasoning.ExtensionResult
}

// NewLLMService creates a new LLM service
func NewLLMService(llmProvider llms.LLMProvider) reasoning.LLMService {
	return &DefaultLLMService{
		llmProvider:          llmProvider,
		extensionService:     reasoning.NewExtensionService(), // Will be set later via SetExtensionService
		lastExtensionResults: make(map[string]reasoning.ExtensionResult),
	}
}

// NewLLMServiceWithExtensions creates a new LLM service with extension service
func NewLLMServiceWithExtensions(llmProvider llms.LLMProvider, extensionService reasoning.ExtensionService) reasoning.LLMService {
	return &DefaultLLMService{
		llmProvider:          llmProvider,
		extensionService:     extensionService,
		lastExtensionResults: make(map[string]reasoning.ExtensionResult),
	}
}

// GenerateLLM implements reasoning.LLMService with extension processing
func (s *DefaultLLMService) GenerateLLM(prompt string) (string, int, error) {
	if s.llmProvider == nil {
		return "", 0, fmt.Errorf("LLM provider not available")
	}

	// Get raw response from LLM provider
	rawResponse, tokens, err := s.llmProvider.Generate(prompt)
	if err != nil {
		return "", tokens, err
	}

	// Store raw response for extension parsing
	s.lastRawResponse = rawResponse

	// Process extensions through extension service
	maskedResponse, extensionCalls := s.extensionService.ProcessResponse(rawResponse)
	s.lastExtensionCalls = extensionCalls

	return maskedResponse, tokens, nil
}

// StreamingBuffers manages streaming with extension call masking
type StreamingBuffers struct {
	inputBuffer     strings.Builder // Raw LLM stream
	streamedLength  int             // Length of content already streamed (avoids String() calls)
	inExtensionCall bool            // Currently inside an extension call
	maxMarkerLength int             // Cached max marker length (computed once)
}

// GenerateLLMStreaming implements reasoning.LLMService with extension call masking
func (s *DefaultLLMService) GenerateLLMStreaming(prompt string) (<-chan string, error) {
	if s.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not available")
	}

	rawStreamCh, err := s.llmProvider.GenerateStreaming(prompt)
	if err != nil {
		return nil, err
	}

	outputCh := make(chan string, 10)

	go func() {
		defer close(outputCh)

		// Initialize buffers with cached max marker length
		buffers := &StreamingBuffers{
			maxMarkerLength: s.getMaxMarkerLength(),
		}

		// Process raw LLM stream preserving original chunks
		for chunk := range rawStreamCh {
			buffers.inputBuffer.WriteString(chunk)
			s.processChunk(buffers, outputCh)
		}

		// Final processing
		s.finalizeStreaming(buffers, outputCh)
	}()

	return outputCh, nil
}

// processChunk processes streaming content with extension marker detection
func (s *DefaultLLMService) processChunk(buffers *StreamingBuffers, outputCh chan<- string) {
	// If we're already in an extension call, don't stream anything
	if buffers.inExtensionCall {
		return
	}

	// Get accumulated content length without string allocation
	totalLength := buffers.inputBuffer.Len()

	// Delegate marker detection to extension service (decoupled)
	if s.extensionService != nil {
		// Only convert to string for marker detection
		fullContent := buffers.inputBuffer.String()
		found, markerPos, _ := s.extensionService.ContainsMarker(fullContent)

		if found {
			// Found marker - stream everything before it
			if markerPos > buffers.streamedLength {
				newContent := fullContent[buffers.streamedLength:markerPos]
				outputCh <- newContent
				buffers.streamedLength = markerPos
			}
			buffers.inExtensionCall = true
			return
		}
	}

	// No marker found yet - use cached buffer size to prevent streaming partial markers
	bufferSize := buffers.maxMarkerLength
	if totalLength <= bufferSize {
		// Content is smaller than buffer, don't stream anything yet
		return
	}

	// Stream everything except the last bufferSize characters
	safeLength := totalLength - bufferSize
	if safeLength > buffers.streamedLength {
		// Only convert to string for the slice we need
		fullContent := buffers.inputBuffer.String()
		newContent := fullContent[buffers.streamedLength:safeLength]
		outputCh <- newContent
		buffers.streamedLength = safeLength
	}
}

// getMaxMarkerLength returns the length of the longest registered marker
func (s *DefaultLLMService) getMaxMarkerLength() int {
	if s.extensionService == nil {
		return streamBufferSize
	}

	maxLen := streamBufferSize
	for _, ext := range s.extensionService.GetAvailableExtensions() {
		if len(ext.OpenTag) > maxLen {
			maxLen = len(ext.OpenTag)
		}
	}
	return maxLen
}

// extractLabelFromExtensions extracts the label from already-parsed extension calls
func (s *DefaultLLMService) extractLabelFromExtensions(extensionCalls []reasoning.ExtensionCall) string {
	if len(extensionCalls) > 0 {
		return extensionCalls[0].UserDisplay
	}
	return fallbackLabel
}

// finalizeStreaming handles final processing
func (s *DefaultLLMService) finalizeStreaming(buffers *StreamingBuffers, outputCh chan<- string) {
	// Get complete response once
	fullResponse := buffers.inputBuffer.String()

	// Store raw response and process extensions (single call)
	s.lastRawResponse = fullResponse
	_, extensionCalls := s.extensionService.ProcessResponse(fullResponse)
	s.lastExtensionCalls = extensionCalls

	// If we're not in an extension call, stream any remaining buffered content
	if !buffers.inExtensionCall {
		totalLength := len(fullResponse)
		if totalLength > buffers.streamedLength {
			remainingContent := fullResponse[buffers.streamedLength:]
			outputCh <- remainingContent
		}
		return
	}

	// Extract and display label from already-parsed extension calls
	label := s.extractLabelFromExtensions(extensionCalls)
	if label != "" {
		outputCh <- label
	}
}

// GetLastRawResponse returns the last raw response from streaming
func (s *DefaultLLMService) GetLastRawResponse() string {
	return s.lastRawResponse
}

// SetExtensionService sets the extension service for processing
func (s *DefaultLLMService) SetExtensionService(service reasoning.ExtensionService) {
	s.extensionService = service
}

// GetExtensionCalls returns the extension calls from the last response
func (s *DefaultLLMService) GetExtensionCalls() []reasoning.ExtensionCall {
	return s.lastExtensionCalls
}

// GetExtensionResults returns the extension results from the last execution
func (s *DefaultLLMService) GetExtensionResults() map[string]reasoning.ExtensionResult {
	return s.lastExtensionResults
}

// ExecuteExtensions executes extension calls and stores results
// NOTE: This method is deprecated - extensions should be executed by the reasoning engine
func (s *DefaultLLMService) ExecuteExtensions(ctx context.Context) error {
	// Extensions are now executed by the reasoning engine, not the LLM service
	// This method is kept for interface compatibility but does nothing
	return nil
}

// ============================================================================
// HISTORY SERVICE
// ============================================================================

// DefaultHistoryService implements reasoning.HistoryService
type DefaultHistoryService struct {
	history []hectorcontext.ConversationMessage
	maxSize int
}

// NewHistoryService creates a new history service
func NewHistoryService(maxSize int) reasoning.HistoryService {
	if maxSize <= 0 {
		maxSize = 10 // Default max size
	}
	return &DefaultHistoryService{
		history: make([]hectorcontext.ConversationMessage, 0),
		maxSize: maxSize,
	}
}

// AddToHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) AddToHistory(role, content string, metadata map[string]interface{}) {
	message := hectorcontext.ConversationMessage{
		Role:     role,
		Content:  content,
		Metadata: metadata,
	}

	s.history = append(s.history, message)

	// Trim history if it exceeds max size
	if len(s.history) > s.maxSize {
		s.history = s.history[len(s.history)-s.maxSize:]
	}
}

// GetRecentHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) GetRecentHistory(count int) []hectorcontext.ConversationMessage {
	if count <= 0 || len(s.history) == 0 {
		return []hectorcontext.ConversationMessage{}
	}

	start := len(s.history) - count
	if start < 0 {
		start = 0
	}

	// Return a copy to prevent external modification
	result := make([]hectorcontext.ConversationMessage, len(s.history[start:]))
	copy(result, s.history[start:])
	return result
}

// ClearHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) ClearHistory() {
	s.history = make([]hectorcontext.ConversationMessage, 0)
}
