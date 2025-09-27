package hector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/providers"

	// Import reasoning engines to register them
	"github.com/kadirpekel/hector/interfaces"
	"github.com/kadirpekel/hector/reasonings"

	// Import concrete provider implementations
	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"

	// Import context components
	hectorcontext "github.com/kadirpekel/hector/context"
)

// ============================================================================
// SIMPLE AGENT - SINGLE-SHOT RESPONSES WITH FULL CONTEXT
// ============================================================================

// Agent represents a simple AI agent that provides single-shot responses
type Agent struct {
	// Core Identity
	name        string
	description string

	// AI Components
	llm      providers.LLMProvider
	embedder providers.EmbedderProvider
	db       providers.DatabaseProvider

	// Context Components
	searchEngine *hectorcontext.SearchEngine
	history      *hectorcontext.ConversationHistory

	// Tool Components
	toolRegistry interfaces.ToolRegistry

	// Configuration
	config       *config.AgentConfig
	globalConfig *config.HectorConfig
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewAgent creates a new simple agent from configuration
func NewAgent(agentConfig *config.AgentConfig, globalConfig *config.HectorConfig) (*Agent, error) {
	agent := &Agent{
		name:         agentConfig.Name,
		description:  agentConfig.Description,
		config:       agentConfig,
		globalConfig: globalConfig,
	}

	// Initialize components
	if err := agent.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize agent components: %w", err)
	}

	return agent, nil
}

// initializeComponents initializes all agent components
func (a *Agent) initializeComponents() error {
	// Initialize LLM using provider reference
	if a.config.LLM != "" {
		llmProvider, err := a.createLLMProviderFromReference(a.config.LLM)
		if err != nil {
			return fmt.Errorf("failed to create LLM provider: %w", err)
		}
		a.llm = llmProvider
	}

	// Initialize Database using provider reference
	if a.config.Database != "" {
		dbProvider, err := a.createDatabaseProviderFromReference(a.config.Database)
		if err != nil {
			return fmt.Errorf("failed to create database provider: %w", err)
		}
		a.db = dbProvider
	}

	// Initialize Embedder using provider reference
	if a.config.Embedder != "" {
		embedderProvider, err := a.createEmbedderProviderFromReference(a.config.Embedder)
		if err != nil {
			return fmt.Errorf("failed to create embedder provider: %w", err)
		}
		a.embedder = embedderProvider
	}

	// Initialize Tool System - will be done by external tool package
	// if err := tools.InitializeToolsFromConfig(a, a.config); err != nil {
	// 	return fmt.Errorf("failed to initialize tools: %w", err)
	// }

	// Discover all tools from registry
	if a.toolRegistry != nil {
		ctx := context.Background()
		if err := a.toolRegistry.DiscoverAllTools(ctx); err != nil {
			// Log warning but don't fail initialization
			// TODO: Use proper logging framework
			fmt.Printf("Warning: Failed to discover tools: %v\n", err)
		}

		// List all discovered tools for debugging
		tools := a.toolRegistry.ListTools()
		if len(tools) > 0 {
			fmt.Printf("Tool registry discovered %d tools total\n", len(tools))
			for _, tool := range tools {
				source, _ := a.toolRegistry.GetToolSource(tool.Name)
				fmt.Printf("  Registered tool: %s - %s (from %s)\n", tool.Name, tool.Description, source)
			}
		}
	}

	// Initialize Search Engine
	if a.db != nil && a.embedder != nil {
		searchEngine, err := hectorcontext.NewSearchEngine(a.db, a.embedder, a.config.Search)
		if err != nil {
			fmt.Printf("Warning: Failed to create search engine: %v\n", err)
		} else {
			a.searchEngine = searchEngine
		}
	}

	// Initialize Document Stores (references to document store names)
	if len(a.config.DocumentStores) > 0 && a.searchEngine != nil {
		if err := InitializeDocumentStoresFromReferences(a.config.DocumentStores, a.searchEngine); err != nil {
			// Log warning but don't fail initialization
			// TODO: Use proper logging framework
			fmt.Printf("Warning: Failed to initialize document stores: %v\n", err)
		}
	}

	// Initialize History
	history, err := hectorcontext.NewConversationHistory(a.name)
	if err != nil {
		fmt.Printf("Warning: Failed to create conversation history: %v\n", err)
	} else {
		a.history = history
	}

	return nil
}

// ============================================================================
// MAIN QUERY METHOD - SINGLE-SHOT RESPONSE
// ============================================================================

// Query processes a query and returns a single response with full context
func (a *Agent) Query(ctx context.Context, query string) (*interfaces.AgentResponse, error) {
	startTime := time.Now()

	// 1. Gather Context
	context, err := a.GatherContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to gather context: %w", err)
	}

	// 2. Execute Tools if needed
	toolResults, err := a.executeTools(ctx, query)
	if err != nil {
		// Log warning but don't fail query execution
		// TODO: Use proper logging framework
		fmt.Printf("Tool execution warning: %v\n", err)
		// Don't fail on tool errors, continue with available context
		toolResults = make(map[string]interfaces.ToolResult)
	}

	// 3. Build Final Prompt
	finalPrompt := a.buildPrompt(query, context, toolResults)

	// 4. Get LLM Response
	response, tokensUsed, err := a.llm.Generate(finalPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 5. Update History
	a.history.AddMessage("user", query, nil)
	a.history.AddMessage("assistant", response, nil)

	// 6. Build Response
	agentResponse := &interfaces.AgentResponse{
		Answer:      response,
		Context:     context,
		Sources:     a.extractSources(context),
		ToolResults: toolResults,
		TokensUsed:  tokensUsed,
		Duration:    time.Since(startTime),
	}

	return agentResponse, nil
}

// ExecuteQueryWithReasoning processes a query using the specified reasoning engine
func (a *Agent) ExecuteQueryWithReasoning(ctx context.Context, query string, reasoningConfig config.ReasoningConfig) (*interfaces.AgentResponse, error) {
	startTime := time.Now()

	// Determine reasoning engine type (default to "dynamic" if not specified)
	engineType := reasoningConfig.Engine
	if engineType == "" {
		engineType = "dynamic"
	}

	// Create reasoning engine (for now, only dynamic is supported)
	if engineType != "dynamic" {
		return nil, fmt.Errorf("unsupported reasoning engine type: %s", engineType)
	}

	// Use agent directly as it implements interfaces.Agent
	engine := reasonings.NewDynamicReasoningEngine(a, reasoningConfig)

	// Execute reasoning
	response, err := engine.ExecuteReasoning(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("reasoning failed: %w", err)
	}

	// Convert response from reasonings package to agent package
	agentResponse := &interfaces.AgentResponse{
		Answer:      response.Answer,
		Context:     response.Context,
		Sources:     response.Sources,
		ToolResults: response.ToolResults,
		TokensUsed:  response.TokensUsed,
		Duration:    time.Since(startTime),
		Confidence:  response.Confidence,
	}
	return agentResponse, nil
}

// ExecuteQueryWithReasoningStreaming processes a query using reasoning with streaming
func (a *Agent) ExecuteQueryWithReasoningStreaming(ctx context.Context, query string, reasoningConfig config.ReasoningConfig) (<-chan string, error) {
	// Determine reasoning engine type (default to "dynamic" if not specified)
	engineType := reasoningConfig.Engine
	if engineType == "" {
		engineType = "dynamic"
	}

	// Create reasoning engine (for now, only dynamic is supported)
	if engineType != "dynamic" {
		return nil, fmt.Errorf("unsupported reasoning engine type: %s", engineType)
	}

	// Use agent directly as it implements interfaces.Agent
	engine := reasonings.NewDynamicReasoningEngine(a, reasoningConfig)

	// Execute reasoning with streaming
	return engine.ExecuteReasoningStreaming(ctx, query)
}

// ============================================================================
// CONTEXT GATHERING
// ============================================================================

// GatherContext gathers all relevant context for the query (implements interfaces.Agent)
func (a *Agent) GatherContext(ctx context.Context, query string) ([]interfaces.SearchResult, error) {
	if a.searchEngine == nil || !a.config.Prompt.IncludeContext {
		return nil, nil
	}

	// Search for relevant documents using unified search method
	topK := a.config.Search.TopK
	if topK == 0 {
		topK = 5 // Default
	}

	// Use the unified Search method and convert to interfaces.SearchResult
	results, err := a.searchEngine.Search(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert providers.SearchResult to interfaces.SearchResult
	converted := make([]interfaces.SearchResult, len(results))
	for i, r := range results {
		converted[i] = interfaces.SearchResult{
			Content:  r.Content,
			Metadata: r.Metadata,
		}
	}

	return converted, nil
}

// ============================================================================
// TOOL DECISION TYPES
// ============================================================================

// ToolDecision represents the LLM's decision about what tools to use
type ToolDecision struct {
	UseCommands bool             `json:"use_commands"`
	Commands    []CommandCall    `json:"commands,omitempty"`
	UseMCP      bool             `json:"use_mcp"`
	MCPTools    []MCPToolCall    `json:"mcp_tools,omitempty"`
	UseSearch   bool             `json:"use_search"`
	SearchCalls []SearchToolCall `json:"search_calls,omitempty"`
	Reasoning   string           `json:"reasoning,omitempty"`
}

// CommandCall represents a command with its parameters
type CommandCall struct {
	Command   string   `json:"command"`
	Arguments []string `json:"arguments,omitempty"`
}

// MCPToolCall represents an MCP tool call with parameters
type MCPToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// SearchToolCall represents a search tool call with parameters
type SearchToolCall struct {
	Query    string   `json:"query"`
	Type     string   `json:"type"`     // "content", "file", "function", "struct"
	Stores   []string `json:"stores"`   // Which document stores to search
	Language string   `json:"language"` // Filter by language
	Limit    int      `json:"limit"`    // Max results
}

// ============================================================================
// TOOL EXECUTION
// ============================================================================

// ExecuteTools executes tools for a query (implements interfaces.Agent)
func (a *Agent) ExecuteTools(ctx context.Context, query string) (map[string]interfaces.ToolResult, error) {
	return a.executeTools(ctx, query)
}

// executeTools asks the LLM to decide what tools to use, then executes them (internal implementation)
func (a *Agent) executeTools(ctx context.Context, query string) (map[string]interfaces.ToolResult, error) {
	if !a.config.Prompt.IncludeTools {
		return nil, nil
	}

	// Ask LLM to decide what tools to use based on the query
	toolDecision, err := a.getToolDecision(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool decision: %w", err)
	}

	// Execute the tool decision using the tool registry
	return a.executeToolDecision(ctx, toolDecision)
}

// executeToolDecision executes tools based on the LLM's decision
func (a *Agent) executeToolDecision(ctx context.Context, decision *ToolDecision) (map[string]interfaces.ToolResult, error) {
	toolResults := make(map[string]interfaces.ToolResult)

	// Execute commands if LLM decided to use them
	if decision.UseCommands && len(decision.Commands) > 0 {
		cmdResults, err := a.executeCommands(ctx, decision.Commands)
		if err != nil {
			return nil, fmt.Errorf("failed to execute commands: %w", err)
		}
		for k, v := range cmdResults {
			toolResults[k] = v
		}
	}

	// Execute MCP tools if LLM decided to use them
	if decision.UseMCP && len(decision.MCPTools) > 0 {
		mcpResults, err := a.executeMCPTools(ctx, decision.MCPTools)
		if err != nil {
			return nil, fmt.Errorf("failed to execute MCP tools: %w", err)
		}
		for k, v := range mcpResults {
			toolResults[k] = v
		}
	}

	// Execute search tool if LLM decided to use it
	if decision.UseSearch && len(decision.SearchCalls) > 0 {
		searchResults, err := a.executeSearchTools(ctx, decision.SearchCalls)
		if err != nil {
			return nil, fmt.Errorf("failed to execute search tools: %w", err)
		}
		for k, v := range searchResults {
			toolResults[k] = v
		}
	}

	return toolResults, nil
}

// executeCommands executes command tools
func (a *Agent) executeCommands(ctx context.Context, commands []CommandCall) (map[string]interfaces.ToolResult, error) {
	results := make(map[string]interfaces.ToolResult)

	for _, cmdCall := range commands {
		// Build arguments map for tool interface
		args := map[string]interface{}{
			"command": cmdCall.Command,
		}

		// Add arguments if provided
		if len(cmdCall.Arguments) > 0 {
			args["command"] = cmdCall.Command + " " + strings.Join(cmdCall.Arguments, " ")
		}

		// Execute via tool registry
		result, err := a.toolRegistry.ExecuteTool(ctx, "execute_command", args)
		if err != nil {
			result = interfaces.ToolResult{
				Success:  false,
				Error:    err.Error(),
				ToolName: "execute_command",
			}
		}
		results[fmt.Sprintf("command_%s", cmdCall.Command)] = result
	}

	return results, nil
}

// executeMCPTools executes MCP tools
func (a *Agent) executeMCPTools(ctx context.Context, mcpTools []MCPToolCall) (map[string]interfaces.ToolResult, error) {
	results := make(map[string]interfaces.ToolResult)

	for _, toolCall := range mcpTools {
		// Execute via tool registry
		result, err := a.toolRegistry.ExecuteTool(ctx, toolCall.Name, toolCall.Parameters)
		if err != nil {
			result = interfaces.ToolResult{
				Success:  false,
				Error:    err.Error(),
				ToolName: toolCall.Name,
			}
		}
		results[toolCall.Name] = result
	}

	return results, nil
}

// executeSearchTools executes search tools
func (a *Agent) executeSearchTools(ctx context.Context, searchCalls []SearchToolCall) (map[string]interfaces.ToolResult, error) {
	results := make(map[string]interfaces.ToolResult)

	for i, searchCall := range searchCalls {
		// Build arguments map for tool interface
		args := map[string]interface{}{
			"query": searchCall.Query,
			"type":  searchCall.Type,
			"limit": searchCall.Limit,
		}

		// Add optional parameters
		if len(searchCall.Stores) > 0 {
			args["stores"] = searchCall.Stores
		}
		if searchCall.Language != "" {
			args["language"] = searchCall.Language
		}

		// Execute via tool registry
		result, err := a.toolRegistry.ExecuteTool(ctx, "search", args)
		if err != nil {
			result = interfaces.ToolResult{
				Success:  false,
				Error:    err.Error(),
				ToolName: "search",
			}
		}
		results[fmt.Sprintf("search_%d", i)] = result
	}

	return results, nil
}

// getToolDecision asks the LLM to decide what tools to use for the given query
func (a *Agent) getToolDecision(ctx context.Context, query string) (*ToolDecision, error) {
	// Build available tools context
	availableTools := a.buildAvailableToolsContext()

	// Create tool decision prompt
	prompt := fmt.Sprintf(`You are an AI agent that needs to decide what tools to use to answer a user query.

Available Tools:
%s

User Query: %s

Based on the query, decide what tools (if any) you need to use. Respond ONLY with a JSON object in this format:
{
  "use_commands": true/false,
  "commands": [
    {"command": "ls", "arguments": ["-la", "/path"]},
    {"command": "grep", "arguments": ["pattern", "file.txt"]}
  ],
  "use_mcp": true/false,
  "mcp_tools": [
    {"name": "weather", "parameters": {"location": "New York"}},
    {"name": "search", "parameters": {"query": "AI research"}}
  ],
  "use_search": true/false,
  "search_calls": [
    {"query": "authentication", "type": "content", "language": "go", "limit": 10},
    {"query": "Login", "type": "function", "stores": ["codebase"], "limit": 5}
  ],
  "reasoning": "brief explanation of your decision"
}

Guidelines:
- For commands: provide command name and arguments separately
- For MCP tools: provide tool name and parameters as key-value pairs
- For search: specify query, type (content/file/function/struct), language filter, stores, and limit
- Only use commands if you need system information (files, processes, etc.)
- Only use MCP tools if they're relevant to the query
- Use search tool to find information in indexed documents/code
- Keep commands simple and safe
- If no tools are needed, set all use_* flags to false

JSON Response:`, availableTools, query)

	// Ask LLM for tool decision
	response, _, err := a.llm.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM tool decision: %w", err)
	}

	// Parse JSON response
	var decision ToolDecision
	if err := json.Unmarshal([]byte(response), &decision); err != nil {
		// If JSON parsing fails, return no tools
		return &ToolDecision{UseCommands: false, UseMCP: false, UseSearch: false}, nil
	}

	return &decision, nil
}

// buildAvailableToolsContext builds a description of available tools using the tool registry
func (a *Agent) buildAvailableToolsContext() string {
	var tools []string

	// Use tool registry to list all available tools
	if a.toolRegistry != nil {
		availableTools := a.toolRegistry.ListTools()
		toolsByRepo := a.toolRegistry.ListToolsByRepository()

		// Add tools grouped by repository
		for repoName, repoTools := range toolsByRepo {
			if len(repoTools) > 0 {
				tools = append(tools, fmt.Sprintf("\n=== %s Tools ===", strings.ToUpper(repoName[:1])+repoName[1:]))
				for _, tool := range repoTools {
					tools = append(tools, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))

					// Add parameter information for better LLM understanding
					if len(tool.Parameters) > 0 {
						var params []string
						for _, param := range tool.Parameters {
							paramDesc := param.Name
							if param.Required {
								paramDesc += " (required)"
							}
							if param.Default != nil {
								paramDesc += fmt.Sprintf(" [default: %v]", param.Default)
							}
							params = append(params, paramDesc)
						}
						tools = append(tools, fmt.Sprintf("  Parameters: %s", strings.Join(params, ", ")))
					}
				}
			}
		}

		// Add summary
		if len(availableTools) > 0 {
			tools = append(tools, fmt.Sprintf("\nTotal available tools: %d", len(availableTools)))
		}
	}

	if len(tools) == 0 {
		return "No tools available"
	}

	return strings.Join(tools, "\n")
}

// ============================================================================
// PROMPT BUILDING
// ============================================================================

// BuildPrompt builds the final prompt with all context (implements interfaces.Agent)
func (a *Agent) BuildPrompt(query string, context []interfaces.SearchResult, toolResults map[string]interfaces.ToolResult) string {
	return a.buildPrompt(query, context, toolResults)
}

// buildPrompt builds the final prompt with all context using smart template substitution (internal implementation)
func (a *Agent) buildPrompt(query string, context []interfaces.SearchResult, toolResults map[string]interfaces.ToolResult) string {
	// Determine which template to use
	var template string

	if a.config.Prompt.FullTemplate != "" {
		// User provided a complete template
		template = a.config.Prompt.FullTemplate
	} else {
		// Build template from components
		template = a.buildComponentTemplate()
	}

	// Apply universal variable substitution
	return a.substituteVariables(template, query, context, toolResults)
}

// buildComponentTemplate builds a template from individual prompt components
func (a *Agent) buildComponentTemplate() string {
	var template strings.Builder

	// 1. System Prompt
	if a.config.Prompt.SystemPrompt != "" {
		template.WriteString(a.config.Prompt.SystemPrompt + "\n\n")
	} else {
		template.WriteString("You are {name}, {description}\n\n")
	}

	// 2. Instructions
	if a.config.Prompt.Instructions != "" {
		template.WriteString("Instructions: " + a.config.Prompt.Instructions + "\n\n")
	}

	// 3. Context from documents (conditional)
	if a.config.Prompt.IncludeContext {
		template.WriteString("Relevant Context:\n{context}\n")
	}

	// 4. Tool Results (conditional)
	if a.config.Prompt.IncludeTools {
		template.WriteString("Tool Results:\n{tools}\n")
	}

	// 5. Conversation History (conditional)
	if a.config.Prompt.IncludeHistory {
		template.WriteString("Recent Conversation:\n{history}\n")
	}

	// 6. Current Query
	if a.config.Prompt.Template != "" {
		// Use custom template
		template.WriteString(a.config.Prompt.Template)
	} else {
		// Default template
		template.WriteString("User Query: {query}\n\nProvide a helpful, accurate response based on the above context and your knowledge.")
	}

	return template.String()
}

// substituteVariables performs universal variable substitution on any template
func (a *Agent) substituteVariables(template, query string, context []interfaces.SearchResult, toolResults map[string]interfaces.ToolResult) string {
	// Built-in variables (always available)
	template = strings.ReplaceAll(template, "{query}", query)
	template = strings.ReplaceAll(template, "{name}", a.name)
	template = strings.ReplaceAll(template, "{description}", a.description)

	// Contextual variables (only substitute if content exists)
	if len(context) > 0 {
		template = strings.ReplaceAll(template, "{context}", a.formatContext(context))
	} else {
		template = strings.ReplaceAll(template, "{context}", "")
	}

	if len(toolResults) > 0 {
		template = strings.ReplaceAll(template, "{tools}", a.formatToolResults(toolResults))
	} else {
		template = strings.ReplaceAll(template, "{tools}", "")
	}

	historyStr := a.formatHistory()
	template = strings.ReplaceAll(template, "{history}", historyStr)

	// Custom user variables
	for key, value := range a.config.Prompt.Variables {
		template = strings.ReplaceAll(template, "{"+key+"}", value)
	}

	// Clean up any empty sections (remove extra newlines)
	template = strings.ReplaceAll(template, "\n\n\n", "\n\n")
	template = strings.TrimSpace(template)

	return template
}

// ============================================================================
// PROMPT FORMATTING HELPERS
// ============================================================================

// formatAsJSON formats any data structure as indented JSON
func formatAsJSON(data interface{}, errorPrefix string) string {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting %s: %v", errorPrefix, err)
	}
	return string(jsonData)
}

// formatContext formats document context as structured JSON for better AI reasoning
func (a *Agent) formatContext(context []interfaces.SearchResult) string {
	if len(context) == 0 {
		return ""
	}

	// Trim content in place for cleaner output
	for i := range context {
		context[i].Content = strings.TrimSpace(context[i].Content)
	}

	return formatAsJSON(context, "context")
}

// formatToolResults formats tool execution results as structured JSON
func (a *Agent) formatToolResults(toolResults map[string]interfaces.ToolResult) string {
	if len(toolResults) == 0 {
		return ""
	}

	// Convert map to slice for consistent JSON output
	results := make([]interfaces.ToolResult, 0, len(toolResults))
	for _, result := range toolResults {
		results = append(results, result)
	}

	return formatAsJSON(results, "tool results")
}

// formatHistory formats conversation history as structured JSON
func (a *Agent) formatHistory() string {
	if a.history == nil {
		return ""
	}

	recentMessages := a.history.GetRecentMessages(6) // Last 6 messages (3 exchanges)
	if len(recentMessages) == 0 {
		return ""
	}

	return formatAsJSON(recentMessages, "history")
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// ExtractSources extracts source information from context (implements interfaces.Agent)
func (a *Agent) ExtractSources(context []interfaces.SearchResult) []string {
	return a.extractSources(context)
}

// extractSources extracts source information from context (internal implementation)
func (a *Agent) extractSources(context []interfaces.SearchResult) []string {
	sources := make([]string, 0, len(context))
	for _, doc := range context {
		if source, ok := doc.Metadata["source"].(string); ok && source != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

// ============================================================================
// PROVIDER CREATION HELPERS
// ============================================================================

// createLLMProviderFromReference creates an LLM provider from a provider reference
func (a *Agent) createLLMProviderFromReference(providerName string) (providers.LLMProvider, error) {
	// Look up provider config from HectorConfig
	if a.globalConfig == nil || a.globalConfig.Providers.LLMs == nil {
		return nil, fmt.Errorf("no global configuration available for provider lookup")
	}

	providerConfig, exists := a.globalConfig.Providers.LLMs[providerName]
	if !exists {
		return nil, fmt.Errorf("LLM provider '%s' not found in configuration", providerName)
	}

	// Create provider from configuration
	switch providerConfig.Type {
	case "ollama":
		return llms.NewOllamaProviderFromConfig(&providerConfig)
	case "openai":
		return llms.NewOpenAIProviderFromConfig(&providerConfig)
	default:
		return nil, fmt.Errorf("unsupported LLM provider type: %s", providerConfig.Type)
	}
}

// createEmbedderProviderFromReference creates an embedder provider from a provider reference
func (a *Agent) createEmbedderProviderFromReference(providerName string) (providers.EmbedderProvider, error) {
	// Look up provider config from HectorConfig
	if a.globalConfig == nil || a.globalConfig.Providers.Embedders == nil {
		return nil, fmt.Errorf("no global configuration available for embedder provider lookup")
	}

	providerConfig, exists := a.globalConfig.Providers.Embedders[providerName]
	if !exists {
		return nil, fmt.Errorf("embedder provider '%s' not found in configuration", providerName)
	}

	// Create provider from configuration
	switch providerConfig.Type {
	case "ollama":
		return embedders.NewOllamaEmbedderFromConfig(&providerConfig)
	default:
		return nil, fmt.Errorf("unsupported embedder provider type: %s", providerConfig.Type)
	}
}

// createDatabaseProviderFromReference creates a database provider from a provider reference
func (a *Agent) createDatabaseProviderFromReference(providerName string) (providers.DatabaseProvider, error) {
	// Look up provider config from HectorConfig
	if a.globalConfig == nil || a.globalConfig.Providers.Databases == nil {
		return nil, fmt.Errorf("no global configuration available for database provider lookup")
	}

	providerConfig, exists := a.globalConfig.Providers.Databases[providerName]
	if !exists {
		return nil, fmt.Errorf("database provider '%s' not found in configuration", providerName)
	}

	// Create provider from configuration
	switch providerConfig.Type {
	case "qdrant":
		return databases.NewQdrantDatabaseProviderFromConfig(&providerConfig)
	default:
		return nil, fmt.Errorf("unsupported database provider type: %s", providerConfig.Type)
	}
}

// InitializeDocumentStoresFromReferences initializes document stores from references
func InitializeDocumentStoresFromReferences(storeRefs []string, searchEngine *hectorcontext.SearchEngine) error {
	// For now, just a stub - in a full implementation, this would look up
	// document store configurations by reference and initialize them
	for _, storeRef := range storeRefs {
		fmt.Printf("Would initialize document store: %s\n", storeRef)
	}
	return nil
}

// ============================================================================
// ACCESSOR METHODS
// ============================================================================

// GetName returns the agent's name
func (a *Agent) GetName() string {
	return a.name
}

// GetDescription returns the agent's description
func (a *Agent) GetDescription() string {
	return a.description
}

// GetToolRegistryInstance returns the tool registry for tool access
func (a *Agent) GetToolRegistryInstance() interfaces.ToolRegistry {
	return a.toolRegistry
}

// SetToolRegistry sets the tool registry for the agent
func (a *Agent) SetToolRegistry(registry interfaces.ToolRegistry) {
	a.toolRegistry = registry
}

// GetSearchEngine returns the search engine for document operations
func (a *Agent) GetSearchEngine() *hectorcontext.SearchEngine {
	return a.searchEngine
}

// ============================================================================
// INTERFACES.AGENT IMPLEMENTATION
// ============================================================================

// GetHistory returns the conversation history interface
func (a *Agent) GetHistory() interfaces.ConversationHistoryInterface {
	return &ConversationHistoryAdapter{history: a.history}
}

// GetToolRegistry returns the tool registry interface for reasoning
func (a *Agent) GetToolRegistry() interfaces.ToolRegistryInterface {
	return &ToolRegistryAdapter{registry: a.toolRegistry}
}

// GetLLM returns the LLM interface
func (a *Agent) GetLLM() interfaces.LLMInterface {
	return &LLMAdapter{llm: a.llm}
}

// GetConfig returns the agent configuration
func (a *Agent) GetConfig() *config.AgentConfig {
	return a.config
}

// ============================================================================
// INTERFACE ADAPTERS (minimal implementations)
// ============================================================================

// ConversationHistoryAdapter adapts ConversationHistory for reasonings package
type ConversationHistoryAdapter struct {
	history *hectorcontext.ConversationHistory
}

func (c *ConversationHistoryAdapter) GetRecentConversationMessages(count int) []interfaces.ConversationMessage {
	if c.history == nil {
		return []interfaces.ConversationMessage{}
	}

	messages := c.history.GetRecentMessages(count)
	result := make([]interfaces.ConversationMessage, len(messages))
	for i, msg := range messages {
		result[i] = interfaces.ConversationMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
	}
	return result
}

func (c *ConversationHistoryAdapter) AddMessage(role, content string, metadata map[string]interface{}) {
	if c.history != nil {
		c.history.AddMessage(role, content, metadata)
	}
}

// ToolRegistryAdapter adapts ToolRegistry for reasonings package
type ToolRegistryAdapter struct {
	registry interfaces.ToolRegistry
}

func (t *ToolRegistryAdapter) ListTools() []interfaces.ToolInfo {
	if t.registry == nil {
		return []interfaces.ToolInfo{}
	}

	tools := t.registry.ListTools()
	result := make([]interfaces.ToolInfo, len(tools))
	for i, tool := range tools {
		result[i] = interfaces.ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
		}
	}
	return result
}

func (t *ToolRegistryAdapter) ExecuteTool(ctx context.Context, toolName string, parameters map[string]interface{}) (interfaces.ToolResult, error) {
	if t.registry == nil {
		return interfaces.ToolResult{}, fmt.Errorf("no tool registry available")
	}

	result, err := t.registry.ExecuteTool(ctx, toolName, parameters)
	return interfaces.ToolResult{
		Success:  result.Success,
		Output:   result.Content,
		Error:    result.Error,
		ToolName: result.ToolName,
	}, err
}

func (t *ToolRegistryAdapter) GetToolSource(toolName string) (string, bool) {
	if t.registry == nil {
		return "", false
	}
	return t.registry.GetToolSource(toolName)
}

// LLMAdapter adapts LLM for reasonings package
type LLMAdapter struct {
	llm providers.LLMProvider
}

func (l *LLMAdapter) Generate(prompt string) (string, int, error) {
	if l.llm == nil {
		return "", 0, fmt.Errorf("no LLM provider available")
	}
	return l.llm.Generate(prompt)
}

func (l *LLMAdapter) GenerateStreaming(prompt string) (<-chan string, error) {
	if l.llm == nil {
		return nil, fmt.Errorf("no LLM provider available")
	}
	return l.llm.GenerateStreaming(prompt)
}

// ============================================================================
// DEFAULT AGENT FACTORY
// ============================================================================

// NewAgentWithDefaults creates a new agent with default configuration
func NewAgentWithDefaults() (*Agent, error) {
	agentConfig := &config.AgentConfig{
		Name:        "Default Agent",
		Description: "Default AI agent with basic capabilities",
		LLM:         "ollama", // Reference to default provider
		Prompt: config.PromptConfig{
			IncludeContext: true,
			IncludeHistory: true,
			IncludeTools:   true,
		},
	}

	return NewAgent(agentConfig, nil)
}

// LoadAgentFromFile loads an Agent from a YAML file
func LoadAgentFromFile(filename string) (*Agent, error) {
	// Load the unified config
	hectorConfig, err := config.LoadHectorConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Extract the first agent from the config
	var agentConfig *config.AgentConfig
	for _, agent := range hectorConfig.Agents {
		agentConfig = &agent
		break
	}

	if agentConfig == nil {
		return nil, fmt.Errorf("no agents found in config file")
	}

	// Create Agent instance from config
	return NewAgent(agentConfig, hectorConfig)
}
