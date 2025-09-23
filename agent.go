package hector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/providers"
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
	llm      llms.LLMProvider
	embedder embedders.EmbedderProvider
	db       databases.DatabaseProvider

	// Context Components
	searchEngine *SearchEngine
	history      *ConversationHistory

	// Tool Components
	mcp             *MCPInfrastructure
	commandExecutor *CommandExecutor

	// Configuration
	config *AgentConfig
}

// AgentResponse represents a simple response from the agent
type AgentResponse struct {
	Answer      string                   `json:"answer"`
	Context     []databases.SearchResult `json:"context,omitempty"`
	Sources     []string                 `json:"sources,omitempty"`
	ToolResults map[string]ToolResult    `json:"tool_results,omitempty"`
	TokensUsed  int                      `json:"tokens_used"`
	Duration    time.Duration            `json:"duration"`
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewAgent creates a new simple agent from configuration
func NewAgent(config *AgentConfig) (*Agent, error) {
	agent := &Agent{
		name:        config.Name,
		description: config.Description,
		config:      config,
	}

	// Initialize components
	if err := agent.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize agent components: %w", err)
	}

	return agent, nil
}

// initializeComponents initializes all agent components
func (a *Agent) initializeComponents() error {
	// Initialize LLM
	if a.config.LLM.Name != "" {
		llmProvider, err := a.createLLMProvider(a.config.LLM)
		if err != nil {
			return fmt.Errorf("failed to create LLM provider: %w", err)
		}
		a.llm = llmProvider
	}

	// Initialize Database
	if a.config.Database.Name != "" {
		dbProvider, err := a.createDatabaseProvider(a.config.Database)
		if err != nil {
			return fmt.Errorf("failed to create database provider: %w", err)
		}
		a.db = dbProvider
	}

	// Initialize Embedder
	if a.config.Embedder.Name != "" {
		embedderProvider, err := a.createEmbedderProvider(a.config.Embedder)
		if err != nil {
			return fmt.Errorf("failed to create embedder provider: %w", err)
		}
		a.embedder = embedderProvider
	}

	// Initialize Tools
	if len(a.config.MCPServers) > 0 {
		a.mcp = NewMCPInfrastructure()
		for _, serverConfig := range a.config.MCPServers {
			fmt.Printf("Discovering tools from MCP server: %s\n", serverConfig.Name)
			a.mcp.AddServer(serverConfig.Name, serverConfig.URL, serverConfig.Description)
		}

		// Discover tools from all servers
		ctx := context.Background()
		if err := a.mcp.DiscoverAllTools(ctx); err != nil {
			fmt.Printf("Warning: Failed to discover MCP tools: %v\n", err)
		}

		// List discovered tools
		tools := a.mcp.ListTools()
		fmt.Printf("Discovered %d tools total\n", len(tools))
		for _, tool := range tools {
			fmt.Printf("  Registered tool: %s - %s\n", tool.Name, tool.Description)
		}
	}

	if a.config.CommandTools != nil {
		a.commandExecutor = NewCommandExecutor(a.config.CommandTools)
	}

	// Initialize Search Engine
	if a.db != nil && a.embedder != nil {
		a.searchEngine = NewSearchEngine(a.db, a.embedder, a.config.Search)
	}

	// Initialize History
	a.history = NewConversationHistory(a.name)

	return nil
}

// ============================================================================
// MAIN QUERY METHOD - SINGLE-SHOT RESPONSE
// ============================================================================

// Query processes a query and returns a single response with full context
func (a *Agent) Query(ctx context.Context, query string) (*AgentResponse, error) {
	startTime := time.Now()

	// 1. Gather Context
	context, err := a.gatherContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to gather context: %w", err)
	}

	// 2. Execute Tools if needed
	toolResults, err := a.executeTools(ctx, query)
	if err != nil {
		fmt.Printf("Tool execution warning: %v\n", err)
		// Don't fail on tool errors, continue with available context
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
	agentResponse := &AgentResponse{
		Answer:      response,
		Context:     context,
		Sources:     a.extractSources(context),
		ToolResults: toolResults,
		TokensUsed:  tokensUsed,
		Duration:    time.Since(startTime),
	}

	return agentResponse, nil
}

// ============================================================================
// CONTEXT GATHERING
// ============================================================================

// gatherContext gathers all relevant context for the query
func (a *Agent) gatherContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	if a.searchEngine == nil || !a.config.Prompt.IncludeContext {
		return nil, nil
	}

	// Search for relevant documents
	topK := a.config.Search.TopK
	if topK == 0 {
		topK = 5 // Default
	}

	// Use SearchModels to search across all configured models
	// SearchEngine manages its own models internally
	allResults, err := a.searchEngine.SearchModels(query, topK)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Flatten results from all models
	return a.searchEngine.FlattenMultiModelResults(allResults), nil
}

// ============================================================================
// TOOL DECISION TYPES
// ============================================================================

// ToolDecision represents the LLM's decision about what tools to use
type ToolDecision struct {
	UseCommands bool          `json:"use_commands"`
	Commands    []CommandCall `json:"commands,omitempty"`
	UseMCP      bool          `json:"use_mcp"`
	MCPTools    []MCPToolCall `json:"mcp_tools,omitempty"`
	Reasoning   string        `json:"reasoning,omitempty"`
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

// ============================================================================
// TOOL EXECUTION
// ============================================================================

// executeTools asks the LLM to decide what tools to use, then executes them
func (a *Agent) executeTools(ctx context.Context, query string) (map[string]ToolResult, error) {
	if !a.config.Prompt.IncludeTools {
		return nil, nil
	}

	// Ask LLM to decide what tools to use based on the query
	toolDecision, err := a.getToolDecision(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool decision: %w", err)
	}

	toolResults := make(map[string]ToolResult)

	// Execute commands if LLM decided to use them
	if toolDecision.UseCommands && len(toolDecision.Commands) > 0 && a.commandExecutor != nil {
		for _, cmdCall := range toolDecision.Commands {
			// Build full command string with arguments
			fullCommand := cmdCall.Command
			if len(cmdCall.Arguments) > 0 {
				fullCommand += " " + strings.Join(cmdCall.Arguments, " ")
			}

			result, err := a.commandExecutor.Execute(ctx, fullCommand, "")
			if err != nil {
				toolResults[fmt.Sprintf("command_%s", cmdCall.Command)] = ToolResult{
					Success:  false,
					Error:    err.Error(),
					ToolName: "command_executor",
				}
			} else {
				toolResults[fmt.Sprintf("command_%s", cmdCall.Command)] = result
			}
		}
	}

	// Execute MCP tools if LLM decided to use them
	if toolDecision.UseMCP && len(toolDecision.MCPTools) > 0 && a.mcp != nil {
		for _, toolCall := range toolDecision.MCPTools {
			result, err := a.mcp.ExecuteTool(ctx, toolCall.Name, toolCall.Parameters)
			if err != nil {
				toolResults[toolCall.Name] = ToolResult{
					Success:  false,
					Error:    err.Error(),
					ToolName: toolCall.Name,
				}
			} else {
				toolResults[toolCall.Name] = result
			}
		}
	}

	return toolResults, nil
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
  "reasoning": "brief explanation of your decision"
}

Guidelines:
- For commands: provide command name and arguments separately
- For MCP tools: provide tool name and parameters as key-value pairs
- Only use commands if you need system information (files, processes, etc.)
- Only use MCP tools if they're relevant to the query
- Keep commands simple and safe
- If no tools are needed, set both use_commands and use_mcp to false

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
		return &ToolDecision{UseCommands: false, UseMCP: false}, nil
	}

	return &decision, nil
}

// buildAvailableToolsContext builds a description of available tools
func (a *Agent) buildAvailableToolsContext() string {
	var tools []string

	if a.commandExecutor != nil {
		tools = append(tools, "- Command Execution: Run shell commands (ls, cat, grep, find, etc.) to get system information")
		tools = append(tools, "  Example: ls -la, grep 'pattern' file.txt, find . -name '*.go'")
	}

	if a.mcp != nil {
		mcpTools := a.mcp.ListTools()
		for _, tool := range mcpTools {
			tools = append(tools, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
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

// buildPrompt builds the final prompt with all context using smart template substitution
func (a *Agent) buildPrompt(query string, context []databases.SearchResult, toolResults map[string]ToolResult) string {
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
func (a *Agent) substituteVariables(template, query string, context []databases.SearchResult, toolResults map[string]ToolResult) string {
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
func (a *Agent) formatContext(context []databases.SearchResult) string {
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
func (a *Agent) formatToolResults(toolResults map[string]ToolResult) string {
	if len(toolResults) == 0 {
		return ""
	}

	// Convert map to slice for consistent JSON output
	results := make([]ToolResult, 0, len(toolResults))
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

// extractSources extracts source information from context
func (a *Agent) extractSources(context []databases.SearchResult) []string {
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

// createLLMProvider creates an LLM provider from config
func (a *Agent) createLLMProvider(config LLMConfig) (llms.LLMProvider, error) {
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name

	if config.Model != "" {
		configMap["model"] = config.Model
	}
	if config.APIKey != "" {
		configMap["api_key"] = config.APIKey
	}
	if config.BaseURL != "" {
		configMap["base_url"] = config.BaseURL
	}
	if config.Temperature > 0 {
		configMap["temperature"] = config.Temperature
	}
	if config.MaxTokens > 0 {
		configMap["max_tokens"] = config.MaxTokens
	}

	for key, value := range config.Extra {
		configMap[key] = value
	}

	return providers.CreateLLMProvider(configMap)
}

// createEmbedderProvider creates an embedder provider from config
func (a *Agent) createEmbedderProvider(config LLMConfig) (embedders.EmbedderProvider, error) {
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name

	if config.Model != "" {
		configMap["model"] = config.Model
	}
	if config.BaseURL != "" {
		configMap["base_url"] = config.BaseURL
	}

	for key, value := range config.Extra {
		configMap[key] = value
	}

	return providers.CreateEmbedderProvider(configMap)
}

// createDatabaseProvider creates a database provider from config
func (a *Agent) createDatabaseProvider(config LLMConfig) (databases.DatabaseProvider, error) {
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name

	if config.BaseURL != "" {
		configMap["base_url"] = config.BaseURL
	}

	for key, value := range config.Extra {
		configMap[key] = value
	}

	return providers.CreateDatabaseProvider(configMap)
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

// GetMCP returns the MCP infrastructure for tool access
func (a *Agent) GetMCP() *MCPInfrastructure {
	return a.mcp
}

// GetSearchEngine returns the search engine for document operations
func (a *Agent) GetSearchEngine() *SearchEngine {
	return a.searchEngine
}

// ============================================================================
// DEFAULT AGENT FACTORY
// ============================================================================

// NewAgentWithDefaults creates a new agent with default configuration
func NewAgentWithDefaults() (*Agent, error) {
	config := &AgentConfig{
		Name:        "Default Agent",
		Description: "Default AI agent with basic capabilities",
		LLM: LLMConfig{
			Name:  "ollama",
			Model: "llama3.2",
		},
		Prompt: PromptConfig{
			IncludeContext: true,
			IncludeHistory: true,
			IncludeTools:   true,
		},
	}

	return NewAgent(config)
}
