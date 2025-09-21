package hector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/providers"
)

// Agent is the main agent system interface - holds only agent-level configs
type Agent struct {
	// Agent-level configurations only
	WorkflowConfig WorkflowConfig // Agent-level workflow behavior
	config         *AgentConfig   // Full agent configuration for access to reasoning config

	// Components with their own configs
	llm          llms.LLMProvider
	db           databases.VectorDB
	embedder     embedders.EmbeddingProvider
	searchEngine *SearchEngine
	mcp          *MCPInfrastructure
	commandTools *CommandToolRegistry // Native command-line tools
	history      *ConversationHistory
	memory       *AgentMemory

	// Model and ingestion management
	modelManager *ModelManager

	// Parent Agent for nested agents (fallback hierarchy)
	parent *Agent
}

// AgentResponse represents an agent query response
type AgentResponse struct {
	Answer        string                   `json:"answer"`
	ToolResults   map[string]ToolResult    `json:"tool_results,omitempty"`
	Context       []databases.SearchResult `json:"context"`
	Sources       []string                 `json:"sources"`
	Confidence    float64                  `json:"confidence"`
	TokensUsed    int                      `json:"tokens_used"`
	WorkflowSteps []WorkflowStepResult     `json:"workflow_steps,omitempty"`
}

// WorkflowStepResult represents the result of a workflow step
type WorkflowStepResult struct {
	StepName    string              `json:"step_name"`
	StepType    string              `json:"step_type"`
	Instruction string              `json:"instruction"`
	Input       string              `json:"input"`
	Output      string              `json:"output"`
	LLMConfig   *YAMLProviderConfig `json:"llm_config,omitempty"`
	Success     bool                `json:"success"`
	Error       string              `json:"error,omitempty"`
	TokensUsed  int                 `json:"tokens_used"`
	Duration    time.Duration       `json:"duration"`
}

// UnifiedReasoningResponse represents the AI's decision on how to handle a query
type UnifiedReasoningResponse struct {
	Approach  string   `json:"approach"`   // "direct" or "planning"
	Response  string   `json:"response"`   // Direct answer or first step description
	NextSteps []string `json:"next_steps"` // Planned steps (if planning approach)
	Continue  bool     `json:"continue"`   // Whether to continue reasoning
	Reasoning string   `json:"reasoning"`  // Why this approach was chosen
}

// WorkflowContext holds context for workflow execution
type WorkflowContext struct {
	Query            string
	Context          []string
	AvailableTools   []ToolInfo
	StepResults      []WorkflowStepResult
	ExecutionHistory []string
	ErrorHistory     []string
	CurrentStep      int
	MaxSteps         int
}

// ============================================================================
// CONSTRUCTORS
// ============================================================================

// NewAgent creates a new Agent instance with zero configuration
func NewAgent() *Agent {
	// Create agent with default configurations
	agent := &Agent{
		// Initialize agent-level configurations only
		WorkflowConfig: WorkflowConfig{
			MaxSteps: 1,
		},

		// Initialize components
		mcp:          NewMCPInfrastructure(),
		commandTools: NewCommandToolRegistry(nil), // Initialize with default security config
		history:      NewConversationHistory("default"),
		memory:       NewAgentMemory(),
		parent:       nil, // No parent by default
	}

	return agent
}

// WithLLMConfig configures LLM with its own config
func (a *Agent) WithLLMConfig(config YAMLProviderConfig) *Agent {
	// Create LLM provider from config
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name
	for key, value := range config.Config {
		configMap[key] = value
	}

	provider, err := providers.CreateLLMProvider(configMap)
	if err != nil {
		// If creation fails, we'll handle it later
		fmt.Printf("Warning: Failed to create LLM provider: %v\n", err)
		return a
	}

	a.llm = provider
	return a
}

// WithSearchConfig configures SearchEngine with its own config
func (a *Agent) WithSearchConfig(config SearchConfig) *Agent {
	if a.searchEngine != nil {
		a.searchEngine.config = config
	}
	return a
}

// WithDatabaseConfig configures database with its own config
func (a *Agent) WithDatabaseConfig(config YAMLProviderConfig) *Agent {
	// Create database provider from config
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name
	for key, value := range config.Config {
		configMap[key] = value
	}

	provider, err := providers.CreateDatabaseProvider(configMap)
	if err != nil {
		fmt.Printf("Warning: Failed to create database provider: %v\n", err)
		return a
	}

	a.db = provider

	// Update search engine with new database
	if a.searchEngine != nil {
		a.searchEngine.db = provider
	} else {
		a.searchEngine = NewSearchEngine(provider, a.embedder, make(map[string]ModelConfig))
	}

	return a
}

// SetParent sets the parent Agent for nested agents
func (a *Agent) SetParent(parent *Agent) {
	a.parent = parent
	// Update search engine to use this agent as parent
	if a.searchEngine != nil {
		a.searchEngine.SetParent(a)
	}
}

// NewAgentWithDefaults creates a new Agent instance with zero configuration
// Assumes Ollama (localhost:11434) and Qdrant (localhost:6334) are running
func NewAgentWithDefaults() (*Agent, error) {
	// Register default providers first
	if err := providers.RegisterDefaultProviders(); err != nil {
		return nil, fmt.Errorf("failed to register providers: %w", err)
	}

	agent := NewAgent()

	// Configure with sensible defaults for local services
	err := agent.configureWithDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to configure agent with defaults: %w", err)
	}

	return agent, nil
}

// configureWithDefaults configures the agent with default local services
func (a *Agent) configureWithDefaults() error {
	// Configure LLM with defaults
	llmConfig := YAMLProviderConfig{
		Name: "ollama",
		Config: map[string]interface{}{
			"model":       "llama3.2",
			"host":        "http://localhost:11434",
			"temperature": 0.7,
			"max_tokens":  1000,
			"timeout":     60,
		},
	}
	a.WithLLMConfig(llmConfig)

	// Configure minimal workflow that doesn't try to create nested agents
	a.WorkflowConfig = WorkflowConfig{
		MaxSteps:      1,
		StreamingMode: "all_steps",
		Verbose:       false,
		Steps: []WorkflowStep{
			{
				Name:        "main",
				Type:        "execute",
				Enabled:     true,
				AgentConfig: nil, // No nested agent config - use the main agent
			},
		},
	}

	// Skip database configuration by default to avoid hanging
	// Users can explicitly configure Qdrant in their config files if needed
	fmt.Println("Running in minimal mode (LLM only).")
	fmt.Println("For document search and memory features, use a config file with Qdrant:")
	fmt.Println("  docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant")
	fmt.Println("  hector --config examples/basic.yaml")

	return nil
}

// ============================================================================
// CONFIGURATION METHODS
// ============================================================================

// WithDatabase sets the vector database
func (a *Agent) WithDatabase(db databases.VectorDB) *Agent {
	a.db = db
	// Update search engine with new database
	if a.searchEngine != nil {
		a.searchEngine.db = db
	} else {
		// Create search engine if it doesn't exist
		a.searchEngine = NewSearchEngine(a.db, a.embedder, make(map[string]ModelConfig))
	}
	return a
}

// WithEmbedder sets the embedding provider
func (a *Agent) WithEmbedder(embedder embedders.EmbeddingProvider) *Agent {
	a.embedder = embedder
	// Update search engine with new embedder only if we have a database
	if a.searchEngine != nil {
		a.searchEngine.embedder = embedder
	} else if a.db != nil {
		// Create search engine only if we have both database and embedder
		a.searchEngine = NewSearchEngine(a.db, a.embedder, make(map[string]ModelConfig))
	}
	return a
}

// WithLLM sets the LLM provider
func (a *Agent) WithLLM(llm llms.LLMProvider) *Agent {
	a.llm = llm
	return a
}

// WithModelsFromConfig sets models from YAML configuration
func (a *Agent) WithModelsFromConfig(models map[string]ModelConfig) *Agent {
	a.searchEngine.models = models
	// Update search engine with new models
	a.searchEngine = NewSearchEngine(a.db, a.embedder, a.searchEngine.models)
	return a
}

// WithMCPServers adds MCP servers to the agent
func (a *Agent) WithMCPServers(servers ...MCPServerConfig) *Agent {
	for _, server := range servers {
		a.mcp.AddServer(server.Name, server.URL, server.Description)
	}
	return a
}

// WithInstruction sets the primary instruction/behavior
func (a *Agent) WithInstruction(instruction string) *Agent {
	// For now, we'll store this in a simple way
	// TODO: This should be handled by the LLM provider's config
	return a
}

// WithPromptTemplate sets a custom prompt template
func (a *Agent) WithPromptTemplate(template string) *Agent {
	// For now, we'll store this in a simple way
	// TODO: This should be handled by the LLM provider's config
	return a
}

// GetMCP returns the MCP infrastructure
func (a *Agent) GetMCP() *MCPInfrastructure {
	return a.mcp
}

// GetLLM returns the configured LLM provider
func (a *Agent) GetLLM() llms.LLMProvider {
	return a.llm
}

// GetHistory returns the conversation history
func (a *Agent) GetHistory() *ConversationHistory {
	return a.history
}

// GetMemory returns the agent memory
func (a *Agent) GetMemory() *AgentMemory {
	return a.memory
}

// SetSessionID sets the conversation session ID
func (a *Agent) SetSessionID(sessionID string) *Agent {
	a.history.SessionID = sessionID
	a.history.UserProfile.ID = sessionID
	return a
}

// buildEnhancedQuery creates an enhanced query incorporating conversation context and tool results
func (a *Agent) buildEnhancedQuery(query string, toolResults map[string]ToolResult) string {
	conversationContext := a.history.GetContextForLLM(5) // Last 5 messages

	if len(toolResults) > 0 {
		toolContext := a.buildToolContext(toolResults)
		if toolContext != "" {
			return fmt.Sprintf("Conversation Context:\n%s\n\nTool Results:\n%s\n\nOriginal question: %s",
				conversationContext, toolContext, query)
		} else {
			// Fallback to just mentioning tools were used
			toolNames := make([]string, 0, len(toolResults))
			for toolName := range toolResults {
				toolNames = append(toolNames, toolName)
			}
			return fmt.Sprintf("Conversation Context:\n%s\n\nTools used: %s\n\nOriginal question: %s",
				conversationContext, strings.Join(toolNames, ", "), query)
		}
	} else if conversationContext != "" {
		return fmt.Sprintf("Conversation Context:\n%s\n\nOriginal question: %s",
			conversationContext, query)
	}

	return query
}

// ============================================================================
// VALIDATION METHODS
// ============================================================================

// validateLLM checks if LLM provider is configured
func (a *Agent) validateLLM() error {
	if a.llm == nil {
		return fmt.Errorf("no LLM provider configured")
	}
	return nil
}

// validateEmbedder checks if embedder is configured
func (a *Agent) validateEmbedder() error {
	if a.embedder == nil {
		return fmt.Errorf("no embedder configured")
	}
	return nil
}

// validateModelManager checks if model manager is initialized
func (a *Agent) validateModelManager() error {
	if a.modelManager == nil {
		return fmt.Errorf("ModelManager not initialized")
	}
	return nil
}

// getStepAgent returns the appropriate agent for a workflow step
// Creates a new step agent if AgentConfig is provided, otherwise returns current agent
func (a *Agent) getStepAgent(step WorkflowStep) *Agent {
	if step.AgentConfig != nil {
		return a.createStepAgent(step)
	}
	return a
}

// ============================================================================
// WORKFLOW ENGINE
// ============================================================================

// ExecuteQueryWithReasoning executes the agent workflow
func (a *Agent) ExecuteQueryWithReasoning(query string, modelNames ...string) (*AgentResponse, error) {
	a.verbosePrint("ExecuteQueryWithReasoning called with query: %s\n", query)

	if err := a.validateLLM(); err != nil {
		return nil, err
	}

	a.verbosePrint("Starting agent workflow\n")
	a.verbosePrint("Workflow config: max_steps=%d, steps_count=%d\n",
		a.WorkflowConfig.MaxSteps, len(a.WorkflowConfig.Steps))

	// Initialize workflow context with document search results (like v1)
	var initialContext []string
	if a.searchEngine != nil {
		// Search for document context using the same approach as v1
		results, err := a.SearchDocuments(query, a.getDefaultModelName(), a.getDefaultTopK())
		if err == nil && len(results) > 0 {
			// Extract context from search results
			contextMap := map[string][]SearchResult{a.getDefaultModelName(): results}
			initialContext = a.searchEngine.ExtractContext(contextMap, a.searchEngine.GetMaxContextLength())
		}
	}

	workflowCtx := &WorkflowContext{
		Query:            query,
		Context:          initialContext, // Start with document search context like v1
		AvailableTools:   []ToolInfo{},
		StepResults:      []WorkflowStepResult{},
		ExecutionHistory: []string{},
		ErrorHistory:     []string{},
		CurrentStep:      0,
		MaxSteps:         a.WorkflowConfig.MaxSteps,
	}

	// Execute the agent workflow
	a.verbosePrint("Executing agent workflow\n")
	return a.executeWorkflow(workflowCtx, modelNames...)
}

// ExecuteQueryWithReasoningStreaming executes the agent workflow with streaming
func (a *Agent) ExecuteQueryWithReasoningStreaming(query string, modelNames ...string) (<-chan string, error) {
	a.verbosePrint("ExecuteQueryWithReasoningStreaming called with query: %s\n", query)

	if err := a.validateLLM(); err != nil {
		return nil, err
	}

	a.verbosePrint("Starting agent workflow\n")
	a.verbosePrint("Workflow config: max_steps=%d, steps_count=%d\n",
		a.WorkflowConfig.MaxSteps, len(a.WorkflowConfig.Steps))

	// Execute the agent workflow with streaming
	a.verbosePrint("Executing agent workflow with streaming (%d steps)\n", len(a.WorkflowConfig.Steps))
	return a.executeWorkflowStreaming(query, modelNames...)
}

// executeWorkflowStreaming executes the agent workflow with streaming
func (a *Agent) executeWorkflowStreaming(query string, modelNames ...string) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Initialize reasoning context with document search results (like v1)
		var initialContext []string
		if a.searchEngine != nil && a.db != nil && a.embedder != nil {
			// Search for document context using the same approach as v1
			results, err := a.SearchDocuments(query, a.getDefaultModelName(), a.getDefaultTopK())
			if err == nil && len(results) > 0 {
				// Extract context from search results
				contextMap := map[string][]SearchResult{a.getDefaultModelName(): results}
				initialContext = a.searchEngine.ExtractContext(contextMap, a.searchEngine.GetMaxContextLength())
			}
		}

		workflowCtx := &WorkflowContext{
			Query:            query,
			Context:          initialContext, // Start with document search context like v1
			AvailableTools:   []ToolInfo{},
			StepResults:      []WorkflowStepResult{},
			ExecutionHistory: []string{},
			ErrorHistory:     []string{},
			CurrentStep:      0,
			MaxSteps:         a.WorkflowConfig.MaxSteps,
		}

		a.verbosePrint("Starting workflow with %d steps...\n", len(a.WorkflowConfig.Steps))

		// Collect all step outputs for final response
		var allStepOutputs []string

		for i, step := range a.WorkflowConfig.Steps {
			if !step.Enabled {
				continue
			}

			workflowCtx.CurrentStep++
			a.verbosePrint("\nStep %d/%d: %s (%s)\n", i+1, len(a.WorkflowConfig.Steps), step.Name, step.Type)

			// Execute step based on streaming mode
			var stepOutput string

			if a.WorkflowConfig.StreamingMode == "all_steps" {
				// Stream this step's output
				stepChan, stepErr := a.executeAgentStepStreaming(step, workflowCtx, modelNames...)
				if stepErr != nil {
					responseChan <- fmt.Sprintf("Step %d failed: %v\n", i+1, stepErr)
					break
				}

				// Collect step output and forward to response channel
				var stepOutputBuilder strings.Builder
				for chunk := range stepChan {
					stepOutputBuilder.WriteString(chunk)
					responseChan <- chunk
				}
				stepOutput = stepOutputBuilder.String()
			} else {
				// Execute step without streaming (final_only or none)
				stepResult := a.executeAgentStep(step, workflowCtx, modelNames...)
				stepOutput = stepResult.Output
				if !stepResult.Success {
					responseChan <- fmt.Sprintf("Step %d failed: %s\n", i+1, stepResult.Error)
					break
				}
			}

			// Store step output for final response
			allStepOutputs = append(allStepOutputs, stepOutput)

			// Create step result for context
			stepResult := WorkflowStepResult{
				StepName: step.Name,
				StepType: step.Type,
				Output:   stepOutput,
				Success:  true,
			}
			workflowCtx.StepResults = append(workflowCtx.StepResults, stepResult)

			a.verbosePrint("Step completed successfully\n")

			// Update context for next step
			a.updateWorkflowContext(step, stepResult, workflowCtx)
		}

		// Generate final response if we have step results
		if len(workflowCtx.StepResults) > 0 {
			finalResponse := a.generateFinalResponseFromSteps(workflowCtx)
			if finalResponse != "" {
				if a.WorkflowConfig.StreamingMode == "final_only" {
					// Stream the final response
					responseChan <- "\n\nFinal Response: " + finalResponse
				} else {
					// For all_steps mode, just add the final response without streaming
					responseChan <- "\n\nFinal Response: " + finalResponse
				}
			}
		}
	}()

	return responseChan, nil
}

// executeAgentStepStreaming executes a step by creating a full agent instance with streaming
func (a *Agent) executeAgentStepStreaming(step WorkflowStep, ctx *WorkflowContext, modelNames ...string) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Get the appropriate agent for this step
		stepAgent := a.getStepAgent(step)
		if stepAgent == nil {
			responseChan <- fmt.Sprintf("Failed to create step agent for %s\n", step.Name)
			return
		}

		// Execute tools first if available (core workflow capability)
		var toolResults map[string]ToolResult

		// Check if we have any tools available (MCP or command tools)
		hasTools := false
		if stepAgent.mcp != nil && len(stepAgent.mcp.ListTools()) > 0 {
			hasTools = true
		}
		if stepAgent.commandTools != nil && len(stepAgent.commandTools.ListTools()) > 0 {
			hasTools = true
		}

		if hasTools {
			toolResults = stepAgent.executeToolsForQuery(ctx.Query)
		}

		// Use step agent's internal reasoning with streaming support, including tool results
		streamingChan, err := stepAgent.executeWithInternalReasoningStreaming(ctx.Query, ctx.Context, step, toolResults)
		if err != nil {
			responseChan <- fmt.Sprintf("Internal reasoning streaming failed: %v", err)
			return
		}

		// Forward streaming chunks from internal reasoning
		for chunk := range streamingChan {
			responseChan <- chunk
		}
	}()

	return responseChan, nil
}

// generateFinalResponseFromSteps generates a final response based on reasoning step results
func (a *Agent) generateFinalResponseFromSteps(ctx *WorkflowContext) string {
	if len(ctx.StepResults) == 0 {
		return ""
	}

	// Simple aggregation of step results
	var result strings.Builder
	for i, step := range ctx.StepResults {
		if step.Success {
			result.WriteString(fmt.Sprintf("Step %d (%s): %s\n", i+1, step.StepName, step.Output))
		}
	}

	return result.String()
}

// executeWorkflow executes the agent workflow reasoning
func (a *Agent) executeWorkflow(ctx *WorkflowContext, modelNames ...string) (*AgentResponse, error) {
	var finalResponse *AgentResponse

	a.verbosePrint("Starting workflow with %d steps...\n", len(a.WorkflowConfig.Steps))

	for i, step := range a.WorkflowConfig.Steps {
		if !step.Enabled {
			continue
		}

		ctx.CurrentStep++
		a.verbosePrint("\nStep %d/%d: %s (%s)\n", i+1, len(a.WorkflowConfig.Steps), step.Name, step.Type)
		a.verbosePrint("Instruction: %s\n", func() string {
			// No instruction field in AgentConfig anymore - use empty string
			return ""
		}())

		stepResult := a.executeAgentStep(step, ctx, modelNames...)
		ctx.StepResults = append(ctx.StepResults, stepResult)

		// Display step result
		if stepResult.Success {
			a.verbosePrint("Step completed successfully (%.2fs, %d tokens)\n",
				stepResult.Duration.Seconds(), stepResult.TokensUsed)
			a.verbosePrint("Output: %s\n", truncateString(stepResult.Output, 200))
		} else {
			a.verbosePrint("Step failed: %s\n", stepResult.Error)
			ctx.ErrorHistory = append(ctx.ErrorHistory, stepResult.Error)

			// Handle step failure based on error handling strategy
			if !a.handleStepFailure(step, stepResult, ctx) {
				a.verbosePrint("Stopping execution due to step failure\n")
				break
			}
		}

		// Update context for next step
		a.updateWorkflowContext(step, stepResult, ctx)
	}

	// Generate final response
	a.verbosePrint("\nGenerating final response...\n")
	finalResponse = a.generateFinalResponse(ctx)
	return finalResponse, nil
}

// executeAgentStep executes a step by creating a full agent instance
func (a *Agent) executeAgentStep(step WorkflowStep, ctx *WorkflowContext, _ ...string) WorkflowStepResult {
	startTime := time.Now()

	result := WorkflowStepResult{
		StepName: step.Name,
		StepType: step.Type,
		Input:    ctx.Query,
		Success:  false,
	}

	// Get the appropriate agent for this step
	stepAgent := a.getStepAgent(step)
	if stepAgent == nil {
		result.Error = "Failed to create step agent"
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute tools first if available (core workflow capability)
	var toolResults map[string]ToolResult

	// Check if we have any tools available (MCP or command tools)
	hasTools := false
	if stepAgent.mcp != nil && len(stepAgent.mcp.ListTools()) > 0 {
		hasTools = true
	}
	if stepAgent.commandTools != nil && len(stepAgent.commandTools.ListTools()) > 0 {
		hasTools = true
	}

	if hasTools {
		toolResults = stepAgent.executeToolsForQuery(ctx.Query)
	}

	// Execute with step agent using internal reasoning, including tool results
	output, tokensUsed, err := stepAgent.executeStepWithDynamicReasoning(step, "", ctx, toolResults)

	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Output = output
		result.TokensUsed = tokensUsed
	}

	result.Duration = time.Since(startTime)
	return result
}

// createStepAgent creates a full agent instance for a step
func (a *Agent) createStepAgent(step WorkflowStep) *Agent {
	if step.AgentConfig == nil {
		return a // Return current agent if no config
	}

	// Create a new agent with step-specific configuration
	stepAgent := &Agent{
		WorkflowConfig: step.AgentConfig.Workflow,
		parent:         a,              // Set parent for fallback hierarchy
		commandTools:   a.commandTools, // Inherit command tools from parent
		mcp:            a.mcp,          // Inherit MCP infrastructure from parent
		history:        a.history,      // Inherit conversation history from parent
		memory:         a.memory,       // Inherit memory from parent
	}

	// Initialize step agent components
	if err := a.initializeStepAgent(stepAgent, step.AgentConfig); err != nil {
		a.verbosePrint("Failed to initialize step agent: %v\n", err)
		return nil
	}

	return stepAgent
}

// initializeStepAgent initializes all components of a step agent
func (a *Agent) initializeStepAgent(stepAgent *Agent, config *AgentConfig) error {
	// Initialize LLM
	if config.LLM.Name != "" {
		llm := a.createStepLLM(&config.LLM)
		if llm != nil {
			stepAgent.llm = llm
		} else {
			stepAgent.llm = a.llm // Fallback to parent LLM
		}
	} else {
		stepAgent.llm = a.llm // Use parent LLM
	}

	// Initialize Memory (inherit from parent for now)
	stepAgent.memory = a.memory

	// Initialize MCP (inherit from parent for now)
	stepAgent.mcp = a.mcp

	// Initialize History (inherit from parent for now)
	stepAgent.history = a.history

	// Set reasoning config defaults
	if stepAgent.WorkflowConfig.MaxSteps == 0 {
		stepAgent.WorkflowConfig.SetDefaults()
	}

	return nil
}

// createStepLLM returns the existing LLM (step-specific config not yet implemented)
func (a *Agent) createStepLLM(config *YAMLProviderConfig) llms.LLMProvider {
	// Create a config map for the step-specific LLM
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name

	// Merge the step-specific config
	for key, value := range config.Config {
		configMap[key] = value
	}

	// Create the step-specific LLM provider
	provider, err := providers.CreateLLMProvider(configMap)
	if err != nil {
		// If creation fails, fall back to the default LLM
		fmt.Printf("Warning: Failed to create step-specific LLM: %v. Using default LLM.\n", err)
		return a.llm
	}

	// Debug: Successfully created step-specific LLM
	return provider
}

// handleStepFailure handles step failure
func (a *Agent) handleStepFailure(_ WorkflowStep, _ WorkflowStepResult, ctx *WorkflowContext) bool {
	// For now, don't continue on step failure - could be enhanced later
	return false
}

// updateWorkflowContext updates context for next step
func (a *Agent) updateWorkflowContext(_ WorkflowStep, result WorkflowStepResult, ctx *WorkflowContext) {
	if result.Success {
		ctx.Context = append(ctx.Context, fmt.Sprintf("%s: %s", result.StepName, result.Output))
	}
}

// generateFinalResponse generates the final response from reasoning results
func (a *Agent) generateFinalResponse(ctx *WorkflowContext) *AgentResponse {
	// Combine all step results into a final response
	var answer strings.Builder
	var totalTokens int

	a.verbosePrint("Reasoning Summary:\n")
	a.verbosePrint("   Steps completed: %d/%d\n", len(ctx.StepResults), ctx.MaxSteps)
	a.verbosePrint("   Errors encountered: %d\n", len(ctx.ErrorHistory))

	for _, result := range ctx.StepResults {
		if result.Success {
			answer.WriteString(fmt.Sprintf("%s: %s\n", result.StepName, result.Output))
			totalTokens += result.TokensUsed
			a.verbosePrint("   %s: %.2fs, %d tokens\n", result.StepName, result.Duration.Seconds(), result.TokensUsed)
		} else {
			a.verbosePrint("   %s: %s\n", result.StepName, result.Error)
		}
	}

	a.verbosePrint("   Total tokens used: %d\n", totalTokens)

	return &AgentResponse{
		Answer:        answer.String(),
		TokensUsed:    totalTokens,
		WorkflowSteps: ctx.StepResults,
	}
}

// ExecuteQuery performs an agent query with tool usage
func (a *Agent) ExecuteQuery(query string, modelNames ...string) (*AgentResponse, error) {
	if err := a.validateLLM(); err != nil {
		return nil, err
	}

	// Add user message to conversation history
	userMessage := a.history.AddMessage("user", query, map[string]interface{}{
		"timestamp":   time.Now(),
		"model_names": modelNames,
	})

	// First, try to answer with agent query (existing functionality)
	agentResponse, err := a.Query(query, modelNames...)
	if err != nil {
		// If agent query fails, return the error instead of creating fake response
		return nil, fmt.Errorf("agent query failed: %w", err)
	}

	// Determine which tools to use based on query analysis
	toolResults := a.executeToolsForQuery(query)

	// Add tool calls to conversation history
	for toolName, result := range toolResults {
		a.history.AddToolCall(userMessage.ID, toolName, map[string]interface{}{
			"query": query,
		})
		a.history.AddToolResult(userMessage.ID, result)
	}

	// Generate final response incorporating tool results
	finalAnswer := agentResponse.Answer

	// Build enhanced query with conversation context and tool results
	enhancedQuery := a.buildEnhancedQuery(query, toolResults)

	// Always generate enhanced response if we have tool results or conversation context
	if enhancedQuery != query {
		prompt, err := BuildPrompt(enhancedQuery, []string{}, a.getDefaultModelName(), "You are a helpful AI assistant.", "")
		if err == nil {
			enhancedResponse, _, err := a.llm.Generate(prompt)
			if err == nil && enhancedResponse != "" {
				finalAnswer = enhancedResponse
			}
		}
	}

	// Add assistant response to conversation history
	a.history.AddMessage("assistant", finalAnswer, map[string]interface{}{
		"confidence":         agentResponse.Confidence,
		"tokens_used":        agentResponse.TokensUsed,
		"tool_results_count": len(toolResults),
	})

	return &AgentResponse{
		Answer:      finalAnswer,
		ToolResults: toolResults,
		Sources:     agentResponse.Sources,
		Confidence:  agentResponse.Confidence,
		TokensUsed:  agentResponse.TokensUsed,
	}, nil
}

// ExecuteQueryStreaming performs an agent query with tool usage and streaming response
func (a *Agent) ExecuteQueryStreaming(query string, modelNames ...string) (<-chan string, error) {
	if err := a.validateLLM(); err != nil {
		return nil, err
	}

	ch := make(chan string)

	go func() {
		defer close(ch)

		// Add user message to conversation history
		userMessage := a.history.AddMessage("user", query, map[string]interface{}{
			"timestamp":   time.Now(),
			"model_names": modelNames,
		})

		// First, try to get context from agent query (only if search engine is configured)
		var context []string
		if a.searchEngine != nil {
			results, err := a.SearchDocuments(query, a.getDefaultModelName(), a.getDefaultTopK())
			if err == nil && len(results) > 0 {
				context = a.searchEngine.ExtractContext(map[string][]SearchResult{a.getDefaultModelName(): results}, a.searchEngine.GetMaxContextLength())
			}
		}

		// Determine which tools to use based on query analysis
		toolResults := a.executeToolsForQuery(query)

		// Show tool call notifications with LLM-generated descriptions
		if len(toolResults) > 0 {
			for toolName, result := range toolResults {
				// Get the tool info from MCP
				if toolInfo, exists := a.mcp.GetTool(toolName); exists {
					// Generate humanized description using LLM
					description, err := a.generateToolDescription(toolInfo, result)
					if err != nil {
						// If description generation fails, use simple fallback
						ch <- fmt.Sprintf("Using tool: %s\n", toolName)
					} else {
						ch <- fmt.Sprintf("%s\n", description)
					}
				} else {
					ch <- fmt.Sprintf("Using tool: %s\n", toolName)
				}
			}
		}

		// Add tool calls to conversation history
		for toolName, result := range toolResults {
			a.history.AddToolCall(userMessage.ID, toolName, map[string]interface{}{
				"query": query,
			})
			a.history.AddToolResult(userMessage.ID, result)
		}

		// Build enhanced query with conversation context and tool results
		enhancedQuery := a.buildEnhancedQuery(query, toolResults)

		// Build prompt using the prompt builder
		prompt, err := BuildPrompt(enhancedQuery, context, a.getDefaultModelName(), "You are a helpful AI assistant.", "")
		if err != nil {
			ch <- "Error: " + err.Error()
			return
		}

		// Start streaming
		streamCh, err := a.llm.GenerateStreaming(prompt)
		if err != nil {
			ch <- "Error: " + err.Error()
			return
		}

		// Stream the response
		for chunk := range streamCh {
			ch <- chunk
		}
	}()

	return ch, nil
}

// executeToolsForQuery uses direct mapping for simple commands, LLM reasoning for complex ones
func (a *Agent) executeToolsForQuery(query string) map[string]ToolResult {
	toolResults := make(map[string]ToolResult)

	// Get available tools from MCP and command tools
	availableTools := a.mcp.ListTools()
	if a.commandTools != nil {
		commandTools := a.commandTools.ListTools()
		availableTools = append(availableTools, commandTools...)
	}

	if len(availableTools) == 0 {
		return toolResults
	}

	// Create structured tool information for LLM
	toolInfo := a.createToolInfoForLLM(availableTools)

	// Ask LLM to reason about tool usage naturally
	toolReasoningPrompt := fmt.Sprintf(`You have access to these tools and need to decide which ones to use for the user's request.

USER REQUEST: "%s"

AVAILABLE TOOLS:
%s

Think about what the user wants and choose the most appropriate tools. If you need to make assumptions about file names or locations, make reasonable guesses - you can always recover if wrong.

If the request can be answered with general knowledge alone, use no tools.

Respond with JSON only:
{
  "tools": ["tool_name"],
  "parameters": {"tool_name": {"param": "value"}}
}`, query, toolInfo)

	// Get LLM reasoning about tool usage
	prompt, err := BuildPrompt(toolReasoningPrompt, []string{}, "tool-reasoning", "You are a helpful AI assistant.", "")
	if err != nil {
		// If prompt building fails, return empty results (no tools executed)
		return toolResults
	}

	reasoningResponse, _, err := a.llm.Generate(prompt)
	if err != nil {
		// If LLM reasoning fails, return empty results (no tools executed)
		return toolResults
	}

	// Parse LLM response (simplified - in production you'd want proper JSON parsing)
	toolDecisions := a.parseToolDecisions(reasoningResponse)

	// Execute the recommended tools with intelligent retry logic
	for _, toolName := range toolDecisions.Tools {
		params := toolDecisions.Parameters[toolName]
		if params == nil {
			// If no parameters specified, use empty parameters
			params = map[string]interface{}{}
		}

		// Type assert to map[string]interface{}
		if paramMap, ok := params.(map[string]interface{}); ok {
			result := a.executeToolWithRetry(query, toolName, paramMap)
			toolResults[toolName] = result
		}
	}

	return toolResults
}

// executeToolWithRetry executes tools naturally - just try once and let AI handle failures
func (a *Agent) executeToolWithRetry(originalQuery string, toolName string, params map[string]interface{}) ToolResult {
	// Execute the tool
	var result ToolResult
	var err error

	// Check if it's a command tool first
	if a.commandTools != nil {
		if _, exists := a.commandTools.GetTool(toolName); exists {
			result, err = a.commandTools.ExecuteTool(context.Background(), toolName, params)
		} else {
			// Execute via MCP
			result, err = a.mcp.ExecuteTool(context.Background(), toolName, params)
		}
	} else {
		// Execute via MCP only
		result, err = a.mcp.ExecuteTool(context.Background(), toolName, params)
	}

	// Return the result as-is, let the AI handle any failures naturally
	if err != nil {
		return ToolResult{
			Content:  "",
			Success:  false,
			Error:    err.Error(),
			ToolName: toolName,
		}
	}

	return result
}

// createToolInfoForLLM creates a JSON-formatted string of tool information for LLM consumption
func (a *Agent) createToolInfoForLLM(tools []ToolInfo) string {
	jsonBytes, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		// Fallback to simple format if JSON marshaling fails
		return fmt.Sprintf("Error formatting tools: %v", err)
	}

	return string(jsonBytes)
}

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Description string `yaml:"description,omitempty"`
}

// ToolDecision represents LLM's decision about tool usage
type ToolDecision struct {
	Reasoning  string                 `json:"reasoning"`
	Tools      []string               `json:"tools"`
	Parameters map[string]interface{} `json:"parameters"`
}

// parseToolDecisions parses LLM response for tool decisions using proper JSON parsing
func (a *Agent) parseToolDecisions(response string) ToolDecision {
	decision := ToolDecision{
		Reasoning:  "LLM reasoning",
		Tools:      []string{},
		Parameters: make(map[string]interface{}),
	}

	// Clean the response by removing markdown code blocks if present
	cleanedResponse := response

	// Remove ```json and ``` markers
	if strings.Contains(response, "```json") {
		// Extract JSON from markdown code block
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // Skip "```json"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				cleanedResponse = strings.TrimSpace(response[start : start+end])
			}
		}
	} else if strings.Contains(response, "```") {
		// Handle generic code blocks
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3 // Skip "```"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				cleanedResponse = strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Try to parse as JSON
	var jsonResponse struct {
		Reasoning  string                 `json:"reasoning"`
		Tools      []string               `json:"tools"`
		Parameters map[string]interface{} `json:"parameters"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &jsonResponse); err == nil {
		// Successfully parsed JSON
		decision.Reasoning = jsonResponse.Reasoning
		decision.Tools = jsonResponse.Tools
		decision.Parameters = jsonResponse.Parameters
		return decision
	}

	// If JSON parsing still fails, return empty decision
	return decision
}

// generateToolDescription uses LLM to generate a human-friendly description of what the tool is doing
func (a *Agent) generateToolDescription(toolInfo ToolInfo, result ToolResult) (string, error) {
	// Handle failed tools differently
	if !result.Success {
		return fmt.Sprintf("failed to %s", strings.ToLower(toolInfo.Description)), nil
	}

	// Extract arguments from result metadata (look for common argument field names)
	args := make(map[string]interface{})
	if result.Metadata != nil {
		// Look for common argument field names
		argFields := []string{"args", "arguments", "params", "parameters"}
		for _, fieldName := range argFields {
			if toolArgs, ok := result.Metadata[fieldName].(map[string]interface{}); ok {
				args = toolArgs
				break
			}
		}
	}

	// Create a prompt for the LLM to generate a description
	prompt := fmt.Sprintf(`Based on the tool name and arguments, generate a brief, human-friendly description of what the tool is doing.

Tool: %s
Description: %s
Arguments: %v

Generate a short, natural description (1-3 words) that explains what the tool is doing.
Examples:
- "checking data"
- "searching information"
- "reading file"
- "fetching data"

Description:`, toolInfo.Name, toolInfo.Description, args)

	// Get LLM response
	builtPrompt, err := BuildPrompt(prompt, []string{}, "tool-description", "You are a helpful AI assistant.", "")
	if err != nil {
		// Template failure - return error instead of fallback
		return "", fmt.Errorf("failed to build tool description prompt: %w", err)
	}

	response, _, err := a.llm.Generate(builtPrompt)
	if err != nil {
		// LLM failure - return error instead of fallback
		return "", fmt.Errorf("failed to generate tool description: %w", err)
	}

	if response == "" {
		return "", fmt.Errorf("empty response from LLM for tool description")
	}

	// Clean up the response (remove quotes, extra whitespace)
	response = strings.TrimSpace(response)
	response = strings.Trim(response, "\"'")

	return response, nil
}

// buildToolContext creates a context string from tool results
func (a *Agent) buildToolContext(toolResults map[string]ToolResult) string {
	var contextParts []string

	for toolName, result := range toolResults {
		if result.Success && result.Content != "" {
			contextParts = append(contextParts, fmt.Sprintf("%s: %s", toolName, result.Content))
		} else if !result.Success && result.Error != "" {
			contextParts = append(contextParts, fmt.Sprintf("%s: ERROR - %s", toolName, result.Error))
		}
	}

	return strings.Join(contextParts, "\n\n")
}

// getDefaultModelName returns the first available model name, or "document" as fallback
func (a *Agent) getDefaultModelName() string {
	if a.searchEngine == nil {
		return "document" // Fallback when no search engine is configured
	}
	modelNames := GetAllModelNames(a.searchEngine.models)
	if len(modelNames) > 0 {
		return modelNames[0]
	}
	return "document" // Fallback for backward compatibility
}

// getDefaultTopK returns the default top K value
func (a *Agent) getDefaultTopK() int {
	// For now, return a hardcoded default
	// In the future, this could be stored in SearchConfig or as a separate field
	return 5
}

// ============================================================================
// CORE AGENT OPERATIONS
// ============================================================================

// Query performs a complete agent query
func (a *Agent) Query(query string, modelNames ...string) (*AgentResponse, error) {
	if err := a.validateLLM(); err != nil {
		return nil, err
	}

	// Handle different model name patterns
	var targetModels []string

	if a.searchEngine == nil {
		// No search engine configured - use empty model list
		targetModels = []string{}
	} else if len(modelNames) == 0 {
		// No models specified - search all models (default behavior)
		targetModels = GetAllModelNames(a.searchEngine.models)
	} else if len(modelNames) == 1 {
		modelName := modelNames[0]
		if modelName == "*" || modelName == "all" || modelName == "" {
			// Wildcard - search all models
			targetModels = GetAllModelNames(a.searchEngine.models)
		} else {
			// Single model
			targetModels = []string{modelName}
		}
	} else {
		// Multiple specific models
		targetModels = modelNames
	}

	// Search specified models (only if search engine is configured)
	var allResults map[string][]SearchResult
	var context []string

	if a.searchEngine != nil {
		var err error
		allResults, err = a.searchEngine.SearchModels(query, a.getDefaultTopK(), targetModels...)
		if err != nil {
			// If search fails, warn user and continue with empty context
			fmt.Printf("Warning: Search failed: %v. Continuing with empty context.\n", err)
			allResults = map[string][]SearchResult{}
		}

		// Extract context from all models with proper sectioning (empty if no results)
		context = a.searchEngine.ExtractContext(allResults, a.searchEngine.GetMaxContextLength())
	} else {
		// No search engine configured - use empty context
		allResults = map[string][]SearchResult{}
		context = []string{}
	}

	// Generate response using LLM with context (empty if no documents found)
	var answer string
	var tokensUsed int
	var genErr error

	modelNameStr := strings.Join(targetModels, ",")

	// Build prompt using the prompt builder
	prompt, err := BuildPrompt(query, context, modelNameStr, "You are a helpful AI assistant.", "")
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Generate response using the pre-built prompt
	answer, tokensUsed, genErr = a.llm.Generate(prompt)

	if genErr != nil {
		return nil, fmt.Errorf("failed to generate response: %w", genErr)
	}

	// Extract sources from all models (empty if no results)
	sources := a.searchEngine.ExtractSources(allResults)

	return &AgentResponse{
		Answer:     answer,
		Context:    a.searchEngine.FlattenMultiModelResults(allResults),
		Sources:    sources,
		Confidence: a.searchEngine.CalculateConfidence(allResults),
		TokensUsed: tokensUsed,
	}, nil
}

// SearchDocuments performs vector similarity search
func (a *Agent) SearchDocuments(query string, modelName string, topK int) ([]SearchResult, error) {
	return a.searchEngine.SearchDocuments(query, modelName, topK)
}

// UpsertDocument adds or updates a document in the vector database
func (a *Agent) UpsertDocument(modelName string, id string, data interface{}) error {
	if err := a.validateEmbedder(); err != nil {
		return err
	}

	config, exists := a.searchEngine.models[modelName]
	if !exists {
		return fmt.Errorf("model %s not found", modelName)
	}

	// Extract embedding text and metadata
	embeddingText, metadata, err := ExtractDocumentData(data)
	if err != nil {
		return fmt.Errorf("failed to extract document data: %w", err)
	}

	// Generate embedding
	vector, err := a.embedder.Embed(embeddingText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Upsert to database
	err = a.db.Upsert(context.Background(), config.Collection, id, vector, metadata)
	if err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}

	return nil
}

// truncateString truncates a string to the specified length and adds ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// VerboseTemplateData represents the data available in verbose templates
type VerboseTemplateData struct {
	Message   string
	Timestamp time.Time
	Level     string
	Source    string
}

// verbosePrint prints a message only if verbose mode is enabled, using the configured template
func (a *Agent) verbosePrint(format string, args ...interface{}) {
	if !a.WorkflowConfig.Verbose {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Parse and execute the template
	tmpl, err := template.New("verbose").Parse(a.WorkflowConfig.VerboseTemplate)
	if err != nil {
		// If template parsing fails, fall back to plain message
		fmt.Printf("%s", message)
		return
	}

	// Create template data
	data := VerboseTemplateData{
		Message:   message,
		Timestamp: time.Now(),
		Level:     "info",
		Source:    "reasoning",
	}

	// Execute template
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		// If template execution fails, fall back to plain message
		fmt.Printf("%s", message)
	}
}

// ============================================================================
// MODEL MANAGER METHODS
// ============================================================================

// SyncModel syncs documents for a specific model
func (a *Agent) SyncModel(modelName string) error {
	if err := a.validateModelManager(); err != nil {
		return err
	}
	return a.modelManager.SyncModel(modelName)
}

// SyncAllModels syncs all models that have ingestion configuration
func (a *Agent) SyncAllModels() error {
	if err := a.validateModelManager(); err != nil {
		return err
	}
	return a.modelManager.SyncAllModels()
}

// GetModelStatus returns the status of a model
func (a *Agent) GetModelStatus(modelName string) (map[string]interface{}, error) {
	if err := a.validateModelManager(); err != nil {
		return nil, err
	}
	return a.modelManager.GetModelStatus(modelName)
}

// ListModels returns a list of all models
func (a *Agent) ListModels() []string {
	if a.modelManager == nil {
		return []string{}
	}
	return a.modelManager.ListModels()
}

// ============================================================================
// AGENT WORKFLOW EXECUTION
// ============================================================================

// executeStepWithDynamicReasoning executes a step with integrated dynamic reasoning decision
func (a *Agent) executeStepWithDynamicReasoning(step WorkflowStep, prompt string, ctx *WorkflowContext, toolResults map[string]ToolResult) (string, int, error) {
	// Use agent's internal reasoning to decide approach and execute, with tool results
	return a.executeWithInternalReasoning(ctx.Query, ctx.Context, step, toolResults)
}

// cleanJSONResponse extracts JSON from responses that may contain backticks or other formatting
func (a *Agent) cleanJSONResponse(response string) string {
	// Remove markdown code fences if present
	cleanedResponse := response
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // Skip "```json"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				cleanedResponse = strings.TrimSpace(response[start : start+end])
			}
		}
	} else if strings.Contains(response, "```") {
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3 // Skip "```"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				cleanedResponse = strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Remove backticks if present (for non-fenced JSON)
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "`", "")

	// Find JSON object boundaries
	start := strings.Index(cleanedResponse, "{")
	end := strings.LastIndex(cleanedResponse, "}")

	if start != -1 && end != -1 && end > start {
		return cleanedResponse[start : end+1]
	}

	return cleanedResponse
}

// executeWithInternalReasoning uses internal dynamic reasoning to process queries
func (a *Agent) executeWithInternalReasoning(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (string, int, error) {
	a.verbosePrint("Agent starting internal reasoning for: %s\n", query)

	// First, decide if this needs simple or dynamic reasoning internally
	needsDynamicReasoning := a.shouldUseDynamicReasoningInternal(query)

	if needsDynamicReasoning {
		a.verbosePrint("Using advanced internal reasoning\n")
		return a.executeDynamicInternalReasoning(query, context, step, toolResults)
	} else {
		a.verbosePrint("Using simple internal reasoning\n")
		return a.executeSimpleInternalReasoning(query, context, step, toolResults)
	}
}

// shouldUseDynamicReasoningInternal decides if query needs dynamic reasoning
func (a *Agent) shouldUseDynamicReasoningInternal(query string) bool {
	// Use AI to make this decision, but with a much better prompt
	prompt := fmt.Sprintf(`You are deciding whether a user query needs simple direct execution or complex dynamic reasoning.

USER QUERY: "%s"

SIMPLE EXECUTION is for:
- Direct commands (show file, list directory, run command)
- Factual lookups (what is X, when did Y happen)
- Basic operations (calculate, convert, format)
- Single-step tasks

DYNAMIC REASONING is for:
- Multi-step analysis requiring planning
- Creative problem solving
- Research requiring multiple sources
- Complex decision making with trade-offs
- Open-ended exploration

Respond with ONLY one word: "simple" or "dynamic"`, query)

	response, _, err := a.llm.Generate(prompt)
	if err != nil {
		// If LLM fails, default to simple (safer)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "dynamic"
}

// executeSimpleInternalReasoning handles simple queries directly
func (a *Agent) executeSimpleInternalReasoning(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (string, int, error) {
	// If we have tool results, use them directly for simple queries
	if len(toolResults) > 0 {
		// For simple file operations, return the tool output directly
		for _, result := range toolResults {
			if result.Success && strings.Contains(result.Content, "\n") {
				// This looks like file content, return it directly
				return strings.TrimSpace(result.Content), 0, nil
			} else if result.Success && result.Content != "" {
				// Return the tool result with minimal formatting
				return result.Content, 0, nil
			}
		}
	}

	// Build simple prompt for cases without useful tool results
	prompt := fmt.Sprintf(`Query: %s

Instructions: Provide a direct, concise answer. If tools were executed, use their results. Do not ask for clarification.`, query)

	// Add context if available
	if len(context) > 0 {
		prompt = fmt.Sprintf("Context: %s\n\n%s", strings.Join(context, "; "), prompt)
	}

	// Add tool results if available
	if len(toolResults) > 0 {
		toolContext := a.buildToolContext(toolResults)
		prompt = prompt + "\n\nTool Results:\n" + toolContext
	}

	// Generate response
	return a.llm.Generate(prompt)
}

// executeDynamicInternalReasoning uses the full dynamic reasoning engine
func (a *Agent) executeDynamicInternalReasoning(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (string, int, error) {
	// Create a dynamic reasoning engine for this agent with full configuration
	dynamicEngine := a.createDynamicEngineForStep(step)
	if dynamicEngine == nil {
		a.verbosePrint("Failed to create dynamic engine, falling back to simple reasoning\n")
		return a.executeSimpleInternalReasoning(query, context, step, toolResults)
	}

	// Build enhanced query with document context and tool results
	enhancedQuery := query
	if len(toolResults) > 0 {
		toolContext := a.buildToolContext(toolResults)
		enhancedQuery = query + "\n\nTool Results:\n" + toolContext
	}

	// Execute dynamic reasoning with full capabilities
	response, err := dynamicEngine.ExecuteDynamicReasoning(enhancedQuery, context)
	if err != nil {
		a.verbosePrint("Dynamic reasoning failed: %v, falling back to simple\n", err)
		return a.executeSimpleInternalReasoning(query, context, step, toolResults)
	}

	return response.Answer, response.TokensUsed, nil
}

// createDynamicEngineForStep creates a dynamic reasoning engine with step-specific configuration
func (a *Agent) createDynamicEngineForStep(step WorkflowStep) *DynamicReasoningEngine {
	// Use step-specific dynamic config if available, otherwise create optimized default
	var dynamicConfig *DynamicReasoningConfig

	if step.AgentConfig != nil && step.AgentConfig.Reasoning != nil {
		// Use user-configured reasoning settings
		dynamicConfig = step.AgentConfig.Reasoning
		a.verbosePrint("Using step-specific reasoning config\n")
	} else if a.config != nil && a.config.Reasoning != nil {
		// Use main agent's reasoning configuration
		dynamicConfig = a.config.Reasoning
		a.verbosePrint("Using main agent reasoning config\n")
	} else {
		// Create optimized config for internal reasoning
		dynamicConfig = &DynamicReasoningConfig{
			MaxIterations:        5,                        // Reasonable default for internal reasoning
			EnableMetaReasoning:  true,                     // Enable meta-reasoning for better decisions
			EnableSelfReflection: true,                     // Enable self-reflection for quality
			EnableGoalEvolution:  false,                    // Keep goals stable for individual steps
			EnableDynamicTools:   true,                     // Enable dynamic tool selection
			QualityThreshold:     0.7,                      // Quality threshold for iterations
			AdaptationThreshold:  0.5,                      // When to adapt approach
			GoalThreshold:        0.8,                      // Goal achievement threshold
			Verbose:              a.WorkflowConfig.Verbose, // Inherit verbose setting
			StreamingMode:        "all_steps",              // Enable streaming for better UX
		}
		a.verbosePrint("Using default optimized reasoning config\n")
	}

	// Create and return dynamic reasoning engine
	return NewDynamicReasoningEngine(a, dynamicConfig)
}

// executeWithInternalReasoningStreaming uses internal dynamic reasoning with streaming support
func (a *Agent) executeWithInternalReasoningStreaming(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (<-chan string, error) {
	a.verbosePrint("Agent starting internal reasoning with streaming for: %s\n", query)

	// First, decide if this needs simple or dynamic reasoning internally
	needsDynamicReasoning := a.shouldUseDynamicReasoningInternal(query)

	if needsDynamicReasoning {
		a.verbosePrint("Using advanced internal reasoning with streaming\n")
		return a.executeDynamicInternalReasoningStreaming(query, context, step, toolResults)
	} else {
		a.verbosePrint("Using simple internal reasoning with streaming\n")
		return a.executeSimpleInternalReasoningStreaming(query, context, step, toolResults)
	}
}

// executeDynamicInternalReasoningStreaming uses the full dynamic reasoning engine with streaming
func (a *Agent) executeDynamicInternalReasoningStreaming(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (<-chan string, error) {
	// Create a dynamic reasoning engine for this agent with full configuration
	dynamicEngine := a.createDynamicEngineForStep(step)
	if dynamicEngine == nil {
		a.verbosePrint("Failed to create dynamic engine, falling back to simple reasoning\n")
		return a.executeSimpleInternalReasoningStreaming(query, context, step, toolResults)
	}

	// Build enhanced query with tool results
	enhancedQuery := query
	if len(toolResults) > 0 {
		toolContext := a.buildToolContext(toolResults)
		enhancedQuery = query + "\n\nTool Results:\n" + toolContext
	}

	// Execute dynamic reasoning with streaming and full capabilities
	return dynamicEngine.ExecuteDynamicReasoningStreaming(enhancedQuery, context)
}

// executeSimpleInternalReasoningStreaming handles simple queries with streaming
func (a *Agent) executeSimpleInternalReasoningStreaming(query string, context []string, step WorkflowStep, toolResults map[string]ToolResult) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Execute simple reasoning (non-streaming) and stream the result
		response, _, err := a.executeSimpleInternalReasoning(query, context, step, toolResults)
		if err != nil {
			responseChan <- fmt.Sprintf("Simple reasoning failed: %v", err)
			return
		}

		// Stream the response
		responseChan <- response
	}()

	return responseChan, nil
}

// buildEnhancedQueryForStep builds enhanced query with document context for step
func (a *Agent) buildEnhancedQueryForStep(query string, step WorkflowStep) string {
	// Search for document context if search engine is available
	// Note: Tool execution is now handled at the workflow level, not here
	return query
}
