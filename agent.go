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
	Confidence  float64                  `json:"confidence,omitempty"`
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

// ExecuteQueryWithReasoning processes a query using dynamic reasoning
func (a *Agent) ExecuteQueryWithReasoning(ctx context.Context, query string, reasoningConfig ReasoningConfig) (*AgentResponse, error) {
	startTime := time.Now()

	// Create dynamic reasoning engine
	engine := &DynamicReasoningEngine{
		agent:  a,
		config: reasoningConfig,
	}

	// Execute dynamic reasoning
	response, err := engine.ExecuteDynamicReasoning(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dynamic reasoning failed: %w", err)
	}

	response.Duration = time.Since(startTime)
	return response, nil
}

// ExecuteQueryWithReasoningStreaming processes a query using dynamic reasoning with streaming
func (a *Agent) ExecuteQueryWithReasoningStreaming(ctx context.Context, query string, reasoningConfig ReasoningConfig) (<-chan string, error) {
	// Create dynamic reasoning engine
	engine := &DynamicReasoningEngine{
		agent:  a,
		config: reasoningConfig,
	}

	// Execute dynamic reasoning with streaming
	return engine.ExecuteDynamicReasoningStreaming(ctx, query)
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
// DYNAMIC REASONING TYPES
// ============================================================================

// DynamicReasoningContext holds the evolving context for dynamic reasoning
type DynamicReasoningContext struct {
	Query                string                   `json:"query"`
	OriginalGoal         string                   `json:"original_goal"`
	CurrentGoal          string                   `json:"current_goal"`
	GoalEvolutionHistory []string                 `json:"goal_evolution_history"`
	IterationResults     []DynamicIterationResult `json:"iteration_results"`
	SelfReflections      []SelfReflection         `json:"self_reflections"`
	MetaReasoning        []MetaReasoningStep      `json:"meta_reasoning"`
	AdaptationHistory    []AdaptationDecision     `json:"adaptation_history"`
	QualityMetrics       QualityMetrics           `json:"quality_metrics"`
	CurrentIteration     int                      `json:"current_iteration"`
	AvailableTools       []ToolInfo               `json:"available_tools"`
	DocumentContext      []databases.SearchResult `json:"document_context"`
	ConversationContext  string                   `json:"conversation_context"`
	ShouldStop           bool                     `json:"should_stop"`
	StopReason           string                   `json:"stop_reason"`
}

// DynamicIterationResult represents the result of a single reasoning iteration
type DynamicIterationResult struct {
	IterationNumber    int                   `json:"iteration_number"`
	StepName           string                `json:"step_name"`
	StepType           string                `json:"step_type"`
	Input              string                `json:"input"`
	Output             string                `json:"output"`
	ToolsUsed          []string              `json:"tools_used"`
	ToolResults        map[string]ToolResult `json:"tool_results"`
	QualityScore       float64               `json:"quality_score"`
	GoalProgress       float64               `json:"goal_progress"`
	Confidence         float64               `json:"confidence"`
	TokensUsed         int                   `json:"tokens_used"`
	Duration           time.Duration         `json:"duration"`
	Success            bool                  `json:"success"`
	Error              string                `json:"error,omitempty"`
	SelfReflection     *SelfReflection       `json:"self_reflection,omitempty"`
	AdaptationNeeded   bool                  `json:"adaptation_needed"`
	NextStepSuggestion string                `json:"next_step_suggestion"`
}

// SelfReflection represents AI's evaluation of its own performance
type SelfReflection struct {
	IterationNumber        int      `json:"iteration_number"`
	PerformanceScore       float64  `json:"performance_score"`
	Strengths              []string `json:"strengths"`
	Weaknesses             []string `json:"weaknesses"`
	ImprovementSuggestions []string `json:"improvement_suggestions"`
	GoalAlignment          float64  `json:"goal_alignment"`
	EfficiencyScore        float64  `json:"efficiency_score"`
	QualityAssessment      string   `json:"quality_assessment"`
	ReflectionPrompt       string   `json:"reflection_prompt"`
	ReflectionResponse     string   `json:"reflection_response"`
}

// MetaReasoningStep represents AI reasoning about its reasoning process
type MetaReasoningStep struct {
	StepNumber         int      `json:"step_number"`
	ReasoningType      string   `json:"reasoning_type"` // "strategy_selection", "step_planning", "quality_evaluation"
	Input              string   `json:"input"`
	Analysis           string   `json:"analysis"`
	Decision           string   `json:"decision"`
	Rationale          string   `json:"rationale"`
	Confidence         float64  `json:"confidence"`
	AlternativeOptions []string `json:"alternative_options"`
}

// AdaptationDecision represents AI's decision to adapt its approach
type AdaptationDecision struct {
	IterationNumber     int     `json:"iteration_number"`
	Trigger             string  `json:"trigger"` // What caused the adaptation
	PreviousApproach    string  `json:"previous_approach"`
	NewApproach         string  `json:"new_approach"`
	Reasoning           string  `json:"reasoning"`
	ExpectedImprovement float64 `json:"expected_improvement"`
	Confidence          float64 `json:"confidence"`
}

// QualityMetrics tracks various quality indicators
type QualityMetrics struct {
	OverallQuality   float64   `json:"overall_quality"`
	GoalAlignment    float64   `json:"goal_alignment"`
	Completeness     float64   `json:"completeness"`
	Accuracy         float64   `json:"accuracy"`
	Efficiency       float64   `json:"efficiency"`
	Innovation       float64   `json:"innovation"`
	Consistency      float64   `json:"consistency"`
	UserSatisfaction float64   `json:"user_satisfaction"`
	Trend            string    `json:"trend"` // "improving", "declining", "stable"
	LastUpdated      time.Time `json:"last_updated"`
}

// DynamicReasoningEngine manages the dynamic reasoning process
type DynamicReasoningEngine struct {
	agent   *Agent
	config  ReasoningConfig
	context *DynamicReasoningContext
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
// DYNAMIC REASONING ENGINE IMPLEMENTATION
// ============================================================================

// ExecuteDynamicReasoning executes the main dynamic reasoning loop
func (d *DynamicReasoningEngine) ExecuteDynamicReasoning(ctx context.Context, query string) (*AgentResponse, error) {
	// 1. Initialize dynamic reasoning context
	if err := d.initializeDynamicContext(ctx, query); err != nil {
		return nil, fmt.Errorf("failed to initialize dynamic context: %w", err)
	}

	// 2. Main dynamic reasoning loop
	for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
		d.context.CurrentIteration++

		// Step 1: Meta-reasoning about current state
		if d.config.EnableMetaReasoning {
			if err := d.performMetaReasoning(ctx); err != nil {
				fmt.Printf("Meta-reasoning warning: %v\n", err)
			}
		}

		// Step 2: Self-reflection on previous iteration
		if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
			if err := d.performSelfReflection(ctx); err != nil {
				fmt.Printf("Self-reflection warning: %v\n", err)
			}
		}

		// Step 3: Execute reasoning step
		result, err := d.executeDynamicStep(ctx)
		if err != nil {
			result = DynamicIterationResult{
				IterationNumber: d.context.CurrentIteration,
				StepName:        "reasoning_step",
				StepType:        "dynamic",
				Success:         false,
				Error:           err.Error(),
				Duration:        0,
			}
		}
		d.context.IterationResults = append(d.context.IterationResults, result)

		// Step 4: Evaluate goal achievement
		goalAchieved, progress := d.evaluateGoalAchievement(result)
		result.GoalProgress = progress

		// Step 5: Update quality metrics
		d.updateQualityMetrics(result)

		// Step 6: Check stopping conditions
		if d.shouldStopReasoning(result, goalAchieved, progress) {
			d.context.ShouldStop = true
			d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
			break
		}

		// Step 7: Adapt approach if needed
		if d.shouldAdaptApproach(result) {
			d.adaptApproach(ctx, result)
		}

		// Step 8: Evolve goals if enabled
		if d.config.EnableGoalEvolution {
			d.evolveGoals(ctx, result)
		}
	}

	// 3. Generate final response
	return d.generateFinalResponse(), nil
}

// ExecuteDynamicReasoningStreaming executes dynamic reasoning with streaming output
func (d *DynamicReasoningEngine) ExecuteDynamicReasoningStreaming(ctx context.Context, query string) (<-chan string, error) {
	ch := make(chan string, 100)

	go func() {
		defer close(ch)

		// 1. Initialize dynamic reasoning context
		if err := d.initializeDynamicContext(ctx, query); err != nil {
			ch <- fmt.Sprintf("Error: Failed to initialize dynamic context: %v\n", err)
			return
		}

		ch <- fmt.Sprintf("🧠 Starting Dynamic Reasoning for: %s\n", query)
		ch <- fmt.Sprintf("📊 Configuration: max_iterations=%d, quality_threshold=%.2f\n",
			d.config.MaxIterations, d.config.QualityThreshold)

		// 2. Main dynamic reasoning loop with streaming
		for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
			d.context.CurrentIteration++

			ch <- fmt.Sprintf("\n=== 🔄 Dynamic Iteration %d/%d ===\n",
				d.context.CurrentIteration, d.config.MaxIterations)

			// Step 1: Meta-reasoning
			if d.config.EnableMetaReasoning {
				ch <- "🧠 Performing meta-reasoning...\n"
				if err := d.performMetaReasoning(ctx); err == nil {
					if len(d.context.MetaReasoning) > 0 {
						lastMeta := d.context.MetaReasoning[len(d.context.MetaReasoning)-1]
						ch <- fmt.Sprintf("📝 Meta-reasoning: %s - %s (confidence: %.2f)\n",
							lastMeta.ReasoningType, lastMeta.Decision, lastMeta.Confidence)
					}
				}
			}

			// Step 2: Self-reflection
			if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
				ch <- "🪞 Performing self-reflection...\n"
				if err := d.performSelfReflection(ctx); err == nil {
					if len(d.context.SelfReflections) > 0 {
						lastReflection := d.context.SelfReflections[len(d.context.SelfReflections)-1]
						ch <- fmt.Sprintf("📊 Performance score: %.2f, Goal alignment: %.2f\n",
							lastReflection.PerformanceScore, lastReflection.GoalAlignment)
					}
				}
			}

			// Step 3: Execute reasoning step
			ch <- "⚡ Executing reasoning step...\n"
			result, err := d.executeDynamicStep(ctx)
			if err != nil {
				result = DynamicIterationResult{
					IterationNumber: d.context.CurrentIteration,
					StepName:        "reasoning_step",
					StepType:        "dynamic",
					Success:         false,
					Error:           err.Error(),
					Duration:        0,
				}
			}
			d.context.IterationResults = append(d.context.IterationResults, result)

			ch <- fmt.Sprintf("✅ Step completed: %s (quality: %.2f, confidence: %.2f, duration: %v)\n",
				result.StepName, result.QualityScore, result.Confidence, result.Duration)

			// Step 4: Evaluate goal achievement
			goalAchieved, progress := d.evaluateGoalAchievement(result)
			result.GoalProgress = progress
			ch <- fmt.Sprintf("🎯 Goal evaluation: achieved=%t, progress=%.2f\n", goalAchieved, progress)

			// Step 5: Update quality metrics
			d.updateQualityMetrics(result)

			// Step 6: Check stopping conditions
			if d.shouldStopReasoning(result, goalAchieved, progress) {
				d.context.ShouldStop = true
				d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
				ch <- fmt.Sprintf("🛑 Stopping: %s\n", d.context.StopReason)
				break
			}

			// Step 7: Adapt approach if needed
			if d.shouldAdaptApproach(result) {
				ch <- "🔄 Adapting approach...\n"
				d.adaptApproach(ctx, result)
			}

			// Step 8: Evolve goals if enabled
			if d.config.EnableGoalEvolution {
				ch <- "🎯 Evaluating goal evolution...\n"
				d.evolveGoals(ctx, result)
			}
		}

		// 3. Generate final response
		ch <- "\n🎉 Generating final response...\n"
		finalResponse := d.generateFinalResponse()
		ch <- fmt.Sprintf("📝 Final Answer: %s\n", finalResponse.Answer)
		ch <- fmt.Sprintf("📊 Total iterations: %d, Quality: %.2f\n",
			d.context.CurrentIteration, d.context.QualityMetrics.OverallQuality)
	}()

	return ch, nil
}

// initializeDynamicContext initializes the dynamic reasoning context
func (d *DynamicReasoningEngine) initializeDynamicContext(ctx context.Context, query string) error {
	// Gather document context
	documentContext, err := d.agent.gatherContext(ctx, query)
	if err != nil {
		documentContext = []databases.SearchResult{} // Continue with empty context
	}

	// Get conversation context
	conversationContext := ""
	if d.agent.history != nil {
		recentMessages := d.agent.history.GetRecentMessages(6)
		if len(recentMessages) > 0 {
			conversationData, _ := json.MarshalIndent(recentMessages, "", "  ")
			conversationContext = string(conversationData)
		}
	}

	// Get available tools
	availableTools := []ToolInfo{}
	if d.agent.mcp != nil {
		availableTools = d.agent.mcp.ListTools()
	}

	// Initialize context
	d.context = &DynamicReasoningContext{
		Query:                query,
		OriginalGoal:         query,
		CurrentGoal:          query,
		GoalEvolutionHistory: []string{query},
		IterationResults:     []DynamicIterationResult{},
		SelfReflections:      []SelfReflection{},
		MetaReasoning:        []MetaReasoningStep{},
		AdaptationHistory:    []AdaptationDecision{},
		QualityMetrics: QualityMetrics{
			OverallQuality: 0.0,
			Trend:          "stable",
			LastUpdated:    time.Now(),
		},
		CurrentIteration:    0,
		AvailableTools:      availableTools,
		DocumentContext:     documentContext,
		ConversationContext: conversationContext,
		ShouldStop:          false,
	}

	return nil
}

// performMetaReasoning performs AI reasoning about the reasoning process
func (d *DynamicReasoningEngine) performMetaReasoning(ctx context.Context) error {
	prompt := d.buildMetaReasoningPrompt()

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		return fmt.Errorf("meta-reasoning LLM call failed: %w", err)
	}

	// Parse meta-reasoning response
	var metaStep MetaReasoningStep
	if err := json.Unmarshal([]byte(response), &metaStep); err != nil {
		// If JSON parsing fails, create a basic meta-reasoning step
		metaStep = MetaReasoningStep{
			StepNumber:    len(d.context.MetaReasoning) + 1,
			ReasoningType: "strategy_selection",
			Analysis:      response,
			Decision:      "Continue with current approach",
			Confidence:    0.7,
		}
	}

	d.context.MetaReasoning = append(d.context.MetaReasoning, metaStep)
	return nil
}

// performSelfReflection performs AI self-reflection on performance
func (d *DynamicReasoningEngine) performSelfReflection(ctx context.Context) error {
	if len(d.context.IterationResults) == 0 {
		return nil
	}

	prompt := d.buildSelfReflectionPrompt()

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		return fmt.Errorf("self-reflection LLM call failed: %w", err)
	}

	// Parse self-reflection response
	var reflection SelfReflection
	if err := json.Unmarshal([]byte(response), &reflection); err != nil {
		// If JSON parsing fails, create a basic reflection
		reflection = SelfReflection{
			IterationNumber:   d.context.CurrentIteration,
			PerformanceScore:  0.7,
			GoalAlignment:     0.7,
			QualityAssessment: response,
		}
	}

	d.context.SelfReflections = append(d.context.SelfReflections, reflection)
	return nil
}

// executeDynamicStep executes a single reasoning step
func (d *DynamicReasoningEngine) executeDynamicStep(ctx context.Context) (DynamicIterationResult, error) {
	startTime := time.Now()

	// Build reasoning prompt
	prompt := d.buildReasoningStepPrompt()

	// Generate response
	response, tokensUsed, err := d.agent.llm.Generate(prompt)
	if err != nil {
		return DynamicIterationResult{}, fmt.Errorf("reasoning step LLM call failed: %w", err)
	}

	// Execute tools if dynamic tools are enabled
	var toolResults map[string]ToolResult
	if d.config.EnableDynamicTools {
		toolResults, _ = d.agent.executeTools(ctx, d.context.CurrentGoal)
	}

	// Calculate quality metrics
	qualityScore := d.calculateStepQuality(response, toolResults)
	confidence := d.calculateConfidence(response, toolResults)

	result := DynamicIterationResult{
		IterationNumber: d.context.CurrentIteration,
		StepName:        fmt.Sprintf("reasoning_step_%d", d.context.CurrentIteration),
		StepType:        "dynamic_reasoning",
		Input:           d.context.CurrentGoal,
		Output:          response,
		ToolsUsed:       d.extractToolsUsed(toolResults),
		ToolResults:     toolResults,
		QualityScore:    qualityScore,
		Confidence:      confidence,
		TokensUsed:      tokensUsed,
		Duration:        time.Since(startTime),
		Success:         true,
	}

	return result, nil
}

// evaluateGoalAchievement evaluates if the goal has been achieved
func (d *DynamicReasoningEngine) evaluateGoalAchievement(result DynamicIterationResult) (bool, float64) {
	// Simple heuristic based on quality and confidence
	progress := (result.QualityScore + result.Confidence) / 2.0
	achieved := progress >= d.config.QualityThreshold
	return achieved, progress
}

// updateQualityMetrics updates the overall quality metrics
func (d *DynamicReasoningEngine) updateQualityMetrics(result DynamicIterationResult) {
	// Update metrics based on latest result
	d.context.QualityMetrics.OverallQuality = result.QualityScore
	d.context.QualityMetrics.GoalAlignment = result.GoalProgress
	d.context.QualityMetrics.Completeness = result.GoalProgress
	d.context.QualityMetrics.Accuracy = result.Confidence
	d.context.QualityMetrics.LastUpdated = time.Now()

	// Determine trend
	if len(d.context.IterationResults) > 1 {
		prevResult := d.context.IterationResults[len(d.context.IterationResults)-2]
		if result.QualityScore > prevResult.QualityScore {
			d.context.QualityMetrics.Trend = "improving"
		} else if result.QualityScore < prevResult.QualityScore {
			d.context.QualityMetrics.Trend = "declining"
		} else {
			d.context.QualityMetrics.Trend = "stable"
		}
	}
}

// shouldStopReasoning determines if reasoning should stop
func (d *DynamicReasoningEngine) shouldStopReasoning(result DynamicIterationResult, goalAchieved bool, progress float64) bool {
	// Stop if goal achieved
	if goalAchieved {
		return true
	}

	// Stop if quality threshold reached
	if result.QualityScore >= d.config.QualityThreshold {
		return true
	}

	// Stop if max iterations reached
	if d.context.CurrentIteration >= d.config.MaxIterations {
		return true
	}

	// Stop if quality declining for multiple iterations
	if len(d.context.IterationResults) >= 3 {
		recentResults := d.context.IterationResults[len(d.context.IterationResults)-3:]
		declining := true
		for i := 1; i < len(recentResults); i++ {
			if recentResults[i].QualityScore >= recentResults[i-1].QualityScore {
				declining = false
				break
			}
		}
		if declining {
			return true
		}
	}

	return false
}

// determineStopReason determines why reasoning stopped
func (d *DynamicReasoningEngine) determineStopReason(result DynamicIterationResult, goalAchieved bool, progress float64) string {
	if goalAchieved {
		return "Goal achieved"
	}
	if result.QualityScore >= d.config.QualityThreshold {
		return "Quality threshold reached"
	}
	if d.context.CurrentIteration >= d.config.MaxIterations {
		return "Maximum iterations reached"
	}
	if d.context.QualityMetrics.Trend == "declining" {
		return "Quality declining"
	}
	return "Unknown"
}

// shouldAdaptApproach determines if approach should be adapted
func (d *DynamicReasoningEngine) shouldAdaptApproach(result DynamicIterationResult) bool {
	// Adapt if quality is low
	if result.QualityScore < 0.5 {
		return true
	}

	// Adapt if no progress for multiple iterations
	if len(d.context.IterationResults) >= 2 {
		prevResult := d.context.IterationResults[len(d.context.IterationResults)-2]
		if result.GoalProgress <= prevResult.GoalProgress {
			return true
		}
	}

	return false
}

// adaptApproach adapts the reasoning approach
func (d *DynamicReasoningEngine) adaptApproach(ctx context.Context, result DynamicIterationResult) {
	adaptation := AdaptationDecision{
		IterationNumber:  d.context.CurrentIteration,
		Trigger:          "Low quality or stagnant progress",
		PreviousApproach: "Current reasoning strategy",
		NewApproach:      "Adjusted reasoning strategy",
		Reasoning:        "Adapting to improve performance",
		Confidence:       0.7,
	}

	d.context.AdaptationHistory = append(d.context.AdaptationHistory, adaptation)
}

// evolveGoals evolves goals based on discoveries
func (d *DynamicReasoningEngine) evolveGoals(ctx context.Context, result DynamicIterationResult) {
	// Simple goal evolution - could be enhanced with LLM-driven evolution
	if result.QualityScore > 0.8 && len(result.Output) > 100 {
		// Goal might need refinement based on new insights
		d.context.GoalEvolutionHistory = append(d.context.GoalEvolutionHistory,
			fmt.Sprintf("Iteration %d: Refined based on insights", d.context.CurrentIteration))
	}
}

// generateFinalResponse generates the final agent response
func (d *DynamicReasoningEngine) generateFinalResponse() *AgentResponse {
	// Combine all iteration outputs
	var finalAnswer strings.Builder
	var allToolResults = make(map[string]ToolResult)
	var totalTokens int

	for _, result := range d.context.IterationResults {
		if result.Success {
			finalAnswer.WriteString(result.Output)
			finalAnswer.WriteString("\n\n")

			// Merge tool results
			for name, toolResult := range result.ToolResults {
				allToolResults[name] = toolResult
			}

			totalTokens += result.TokensUsed
		}
	}

	return &AgentResponse{
		Answer:      strings.TrimSpace(finalAnswer.String()),
		Context:     d.context.DocumentContext,
		Sources:     d.agent.extractSources(d.context.DocumentContext),
		ToolResults: allToolResults,
		TokensUsed:  totalTokens,
		Confidence:  d.context.QualityMetrics.OverallQuality,
	}
}

// Helper methods for building prompts and calculating metrics

func (d *DynamicReasoningEngine) buildMetaReasoningPrompt() string {
	return fmt.Sprintf(`You are an AI reasoning about your own reasoning process. Analyze the current state and decide what to do next.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Previous Results: %d iterations completed
- Quality Trend: %s

Available Tools: %d tools
Document Context: %d documents
Conversation Context: %s

Respond with a JSON object:
{
  "reasoning_type": "strategy_selection|step_planning|quality_evaluation",
  "analysis": "Your analysis of the current state",
  "decision": "What should be done next",
  "rationale": "Why this decision makes sense",
  "confidence": 0.8,
  "alternative_options": ["option1", "option2"]
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		len(d.context.IterationResults),
		d.context.QualityMetrics.Trend,
		len(d.context.AvailableTools),
		len(d.context.DocumentContext),
		d.context.ConversationContext[:min(len(d.context.ConversationContext), 200)])
}

func (d *DynamicReasoningEngine) buildSelfReflectionPrompt() string {
	lastResult := d.context.IterationResults[len(d.context.IterationResults)-1]

	return fmt.Sprintf(`You are an AI reflecting on your own performance. Analyze your recent reasoning step.

Recent Step Analysis:
- Step: %s (%s)
- Output: %s
- Quality Score: %.2f
- Confidence: %.2f
- Duration: %v
- Success: %t

Overall Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d

Respond with a JSON object:
{
  "performance_score": 0.8,
  "strengths": ["strength1", "strength2"],
  "weaknesses": ["weakness1"],
  "improvement_suggestions": ["suggestion1"],
  "goal_alignment": 0.9,
  "efficiency_score": 0.7,
  "quality_assessment": "Detailed assessment"
}`,
		lastResult.StepName,
		lastResult.StepType,
		lastResult.Output[:min(len(lastResult.Output), 200)],
		lastResult.QualityScore,
		lastResult.Confidence,
		lastResult.Duration,
		lastResult.Success,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations)
}

func (d *DynamicReasoningEngine) buildReasoningStepPrompt() string {
	prompt := d.agent.buildPrompt(d.context.CurrentGoal, d.context.DocumentContext, make(map[string]ToolResult))

	// Add dynamic reasoning context
	prompt += fmt.Sprintf(`

Dynamic Reasoning Context:
- Iteration: %d/%d
- Original Goal: %s
- Current Goal: %s
- Previous Iterations: %d completed

Please provide a thoughtful response that addresses the current goal while considering the context above.`,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		d.context.OriginalGoal,
		d.context.CurrentGoal,
		len(d.context.IterationResults))

	return prompt
}

func (d *DynamicReasoningEngine) calculateStepQuality(response string, toolResults map[string]ToolResult) float64 {
	// Simple quality calculation based on response length and tool success
	quality := 0.5 // Base quality

	// Add quality for response length (longer = potentially more detailed)
	if len(response) > 100 {
		quality += 0.2
	}
	if len(response) > 500 {
		quality += 0.1
	}

	// Add quality for successful tool usage
	for _, result := range toolResults {
		if result.Success {
			quality += 0.1
		}
	}

	// Cap at 1.0
	if quality > 1.0 {
		quality = 1.0
	}

	return quality
}

func (d *DynamicReasoningEngine) calculateConfidence(response string, toolResults map[string]ToolResult) float64 {
	// Simple confidence calculation
	confidence := 0.7 // Base confidence

	// Increase confidence if tools were used successfully
	successfulTools := 0
	totalTools := len(toolResults)
	for _, result := range toolResults {
		if result.Success {
			successfulTools++
		}
	}

	if totalTools > 0 {
		toolSuccessRate := float64(successfulTools) / float64(totalTools)
		confidence = (confidence + toolSuccessRate) / 2.0
	}

	return confidence
}

func (d *DynamicReasoningEngine) extractToolsUsed(toolResults map[string]ToolResult) []string {
	tools := make([]string, 0, len(toolResults))
	for name := range toolResults {
		tools = append(tools, name)
	}
	return tools
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
