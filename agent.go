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
	ReasoningConfig ReasoningConfig // Agent-level reasoning behavior

	// Components with their own configs
	llm          llms.LLMProvider
	db           databases.VectorDB
	embedder     embedders.EmbeddingProvider
	searchEngine *SearchEngine
	mcp          *MCPInfrastructure
	history      *ConversationHistory
	memory       *AgentMemory

	// Model and ingestion management
	modelManager *ModelManager

	// Parent Agent for nested agents (fallback hierarchy)
	parent *Agent
}

// AgentResponse represents an agent query response
type AgentResponse struct {
	Answer         string                   `json:"answer"`
	ToolResults    map[string]ToolResult    `json:"tool_results,omitempty"`
	Context        []databases.SearchResult `json:"context"`
	Sources        []string                 `json:"sources"`
	Confidence     float64                  `json:"confidence"`
	TokensUsed     int                      `json:"tokens_used"`
	ReasoningSteps []ReasoningStepResult    `json:"reasoning_steps,omitempty"`
}

// ReasoningStepResult represents the result of a reasoning step
type ReasoningStepResult struct {
	StepName    string                 `json:"step_name"`
	StepType    string                 `json:"step_type"`
	Instruction string                 `json:"instruction"`
	Input       string                 `json:"input"`
	Output      string                 `json:"output"`
	LLMConfig   *YAMLProviderConfig    `json:"llm_config,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	TokensUsed  int                    `json:"tokens_used"`
	Duration    time.Duration          `json:"duration"`
}

// ReasoningContext holds context for reasoning execution
type ReasoningContext struct {
	Query            string
	Context          []string
	AvailableTools   []ToolInfo
	StepResults      []ReasoningStepResult
	ExecutionHistory []string
	ErrorHistory     []string
	CurrentStep      int
	MaxSteps         int
	RetryCount       int
	MaxRetries       int
}

// ============================================================================
// CONSTRUCTORS
// ============================================================================

// NewAgent creates a new Agent instance with zero configuration
func NewAgent() *Agent {
	// Create agent with default configurations
	agent := &Agent{
		// Initialize agent-level configurations only
		ReasoningConfig: ReasoningConfig{
			Strategy:    "single_shot",
			MaxSteps:    1,
			EnableRetry: false,
		},

		// Initialize components
		mcp:     NewMCPInfrastructure(),
		history: NewConversationHistory("default"),
		memory:  NewAgentMemory(),
		parent:  nil, // No parent by default
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

	// Configure Database with defaults
	dbConfig := YAMLProviderConfig{
		Name: "qdrant",
		Config: map[string]interface{}{
			"host":     "localhost",
			"port":     6334,
			"timeout":  30,
			"use_tls":  false,
			"insecure": false,
		},
	}
	a.WithDatabaseConfig(dbConfig)

	// Configure Embedder with defaults
	embedderConfig := map[string]interface{}{
		"provider":    "ollama",
		"model":       "nomic-embed-text",
		"host":        "http://localhost:11434",
		"dimension":   768,
		"timeout":     30,
		"max_retries": 3,
	}

	embedder, err := providers.CreateEmbedderProvider(embedderConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedder provider: %w", err)
	}
	a.embedder = embedder

	// Configure SearchEngine with defaults
	searchConfig := SearchConfig{
		MaxContextLength: 2000,
		ContextStrategy:  "relevance",
		EnableReranking:  false,
	}
	a.WithSearchConfig(searchConfig)

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
	// Update search engine with new embedder
	if a.searchEngine != nil {
		a.searchEngine.embedder = embedder
	} else {
		// Create search engine if it doesn't exist
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
// REASONING ENGINE
// ============================================================================

// ExecuteQueryWithReasoning performs an agent query using the configured reasoning strategy
func (a *Agent) ExecuteQueryWithReasoning(query string, modelNames ...string) (*AgentResponse, error) {
	a.verbosePrint("ExecuteQueryWithReasoning called with query: %s\n", query)

	if a.llm == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}

	a.verbosePrint("Starting reasoning with strategy: %s\n", a.ReasoningConfig.Strategy)
	a.verbosePrint("Reasoning config: max_steps=%d, enable_retry=%t, steps_count=%d\n",
		a.ReasoningConfig.MaxSteps, a.ReasoningConfig.EnableRetry, len(a.ReasoningConfig.Steps))

	// Initialize reasoning context
	reasoningCtx := &ReasoningContext{
		Query:            query,
		Context:          []string{},
		AvailableTools:   []ToolInfo{},
		StepResults:      []ReasoningStepResult{},
		ExecutionHistory: []string{},
		ErrorHistory:     []string{},
		CurrentStep:      0,
		MaxSteps:         a.ReasoningConfig.MaxSteps,
		RetryCount:       0,
		MaxRetries:       a.ReasoningConfig.MaxRetries,
	}

	// Execute based on reasoning strategy
	switch a.ReasoningConfig.Strategy {
	case "single_shot":
		a.verbosePrint("Using single-shot reasoning (no steps)\n")
		return a.executeSingleShot(reasoningCtx, modelNames...)
	case "iterative":
		return a.executeIterative(reasoningCtx, modelNames...)
	case "state_machine":
		return a.executeStateMachine(reasoningCtx, modelNames...)
	default:
		a.verbosePrint("Using default single-shot reasoning\n")
		return a.executeSingleShot(reasoningCtx, modelNames...)
	}
}

// ExecuteQueryWithReasoningStreaming performs an agent query using the configured reasoning strategy with streaming
func (a *Agent) ExecuteQueryWithReasoningStreaming(query string, modelNames ...string) (<-chan string, error) {
	a.verbosePrint("ExecuteQueryWithReasoningStreaming called with query: %s\n", query)

	if a.llm == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}

	a.verbosePrint("Starting reasoning with strategy: %s\n", a.ReasoningConfig.Strategy)
	a.verbosePrint("Reasoning config: max_steps=%d, enable_retry=%t, steps_count=%d\n",
		a.ReasoningConfig.MaxSteps, a.ReasoningConfig.EnableRetry, len(a.ReasoningConfig.Steps))

	// Support streaming for different reasoning strategies
	switch a.ReasoningConfig.Strategy {
	case "single_shot":
		a.verbosePrint("Using single-shot reasoning with streaming (no steps)\n")
		return a.executeSingleShotStreaming(query, modelNames...)
	case "state_machine":
		a.verbosePrint("Using state machine reasoning with streaming (%d steps)\n", len(a.ReasoningConfig.Steps))
		return a.executeStateMachineStreaming(query, modelNames...)
	case "iterative":
		a.verbosePrint("Using iterative reasoning with streaming (max %d steps)\n", a.ReasoningConfig.MaxSteps)
		return a.executeIterativeStreaming(query, modelNames...)
	default:
		// Fall back to single-shot streaming for unknown strategies
		a.verbosePrint("Unknown strategy '%s', falling back to single-shot streaming\n", a.ReasoningConfig.Strategy)
		return a.executeSingleShotStreaming(query, modelNames...)
	}
}

// executeSingleShotStreaming executes a single-shot query with streaming
func (a *Agent) executeSingleShotStreaming(query string, modelNames ...string) (<-chan string, error) {
	return a.ExecuteQueryStreaming(query, modelNames...)
}

// executeStateMachineStreaming executes state machine reasoning with streaming
func (a *Agent) executeStateMachineStreaming(query string, modelNames ...string) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Initialize reasoning context
		reasoningCtx := &ReasoningContext{
			Query:            query,
			Context:          []string{},
			AvailableTools:   []ToolInfo{},
			StepResults:      []ReasoningStepResult{},
			ExecutionHistory: []string{},
			ErrorHistory:     []string{},
			CurrentStep:      0,
			MaxSteps:         a.ReasoningConfig.MaxSteps,
			RetryCount:       0,
			MaxRetries:       a.ReasoningConfig.MaxRetries,
		}

		a.verbosePrint("Starting reasoning with %d steps...\n", len(a.ReasoningConfig.Steps))

		// Collect all step outputs for final response
		var allStepOutputs []string

		for i, step := range a.ReasoningConfig.Steps {
			if !step.Enabled {
				continue
			}

			reasoningCtx.CurrentStep++
			a.verbosePrint("\nStep %d/%d: %s (%s)\n", i+1, len(a.ReasoningConfig.Steps), step.Name, step.Type)

			// Execute step based on streaming mode
			var stepOutput string

			if a.ReasoningConfig.StreamingMode == "all_steps" {
				// Stream this step's output
				stepChan, stepErr := a.executeReasoningStepStreaming(step, reasoningCtx, modelNames...)
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
				stepResult := a.executeReasoningStep(step, reasoningCtx, modelNames...)
				stepOutput = stepResult.Output
				if !stepResult.Success {
					responseChan <- fmt.Sprintf("Step %d failed: %s\n", i+1, stepResult.Error)
					break
				}
			}

			// Store step output for final response
			allStepOutputs = append(allStepOutputs, stepOutput)

			// Create step result for context
			stepResult := ReasoningStepResult{
				StepName: step.Name,
				StepType: step.Type,
				Output:   stepOutput,
				Success:  true,
			}
			reasoningCtx.StepResults = append(reasoningCtx.StepResults, stepResult)

			a.verbosePrint("Step completed successfully\n")

			// Update context for next step
			a.updateReasoningContext(step, stepResult, reasoningCtx)
		}

		// Generate final response if we have step results
		if len(reasoningCtx.StepResults) > 0 {
			finalResponse := a.generateFinalResponseFromSteps(reasoningCtx)
			if finalResponse != "" {
				if a.ReasoningConfig.StreamingMode == "final_only" {
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

// executeIterativeStreaming executes iterative reasoning with streaming
func (a *Agent) executeIterativeStreaming(query string, modelNames ...string) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Initialize reasoning context
		reasoningCtx := &ReasoningContext{
			Query:            query,
			Context:          []string{},
			AvailableTools:   []ToolInfo{},
			StepResults:      []ReasoningStepResult{},
			ExecutionHistory: []string{},
			ErrorHistory:     []string{},
			CurrentStep:      0,
			MaxSteps:         a.ReasoningConfig.MaxSteps,
			RetryCount:       0,
			MaxRetries:       a.ReasoningConfig.MaxRetries,
		}

		a.verbosePrint("Starting iterative reasoning (max %d steps)...\n", reasoningCtx.MaxSteps)

		// Collect all iteration outputs for final response
		var allIterationOutputs []string

		for step := 0; step < reasoningCtx.MaxSteps; step++ {
			reasoningCtx.CurrentStep = step
			a.verbosePrint("\nIteration %d/%d\n", step+1, reasoningCtx.MaxSteps)

			// Execute iteration based on streaming mode
			var iterationOutput string

			if a.ReasoningConfig.StreamingMode == "all_steps" {
				// Stream this iteration's output
				iterationChan, iterErr := a.ExecuteQueryStreaming(reasoningCtx.Query, modelNames...)
				if iterErr != nil {
					responseChan <- fmt.Sprintf("Iteration %d failed: %v\n", step+1, iterErr)
					break
				}

				// Collect iteration output and forward to response channel
				var iterationOutputBuilder strings.Builder
				for chunk := range iterationChan {
					iterationOutputBuilder.WriteString(chunk)
					responseChan <- chunk
				}
				iterationOutput = iterationOutputBuilder.String()
			} else {
				// Execute iteration without streaming (final_only or none)
				response, iterErr := a.ExecuteQuery(reasoningCtx.Query, modelNames...)
				if iterErr != nil {
					responseChan <- fmt.Sprintf("Iteration %d failed: %v\n", step+1, iterErr)
					break
				}
				iterationOutput = response.Answer
			}

			// Store iteration output for final response
			allIterationOutputs = append(allIterationOutputs, iterationOutput)

			// Create step result for context
			stepResult := ReasoningStepResult{
				StepName: fmt.Sprintf("iteration_%d", step+1),
				StepType: "execute",
				Output:   iterationOutput,
				Success:  true,
			}
			reasoningCtx.StepResults = append(reasoningCtx.StepResults, stepResult)

			a.verbosePrint("Iteration completed successfully\n")

			// Check if we should continue (simplified logic)
			if step < reasoningCtx.MaxSteps-1 {
				// Update query for next iteration based on previous results
				reasoningCtx.Query = fmt.Sprintf("Based on the previous response: %s\n\nPlease continue or refine your answer.", iterationOutput)
			}
		}

		// Generate final response if we have iteration results
		if len(reasoningCtx.StepResults) > 0 {
			finalResponse := a.generateFinalResponseFromSteps(reasoningCtx)
			if finalResponse != "" {
				if a.ReasoningConfig.StreamingMode == "final_only" {
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

// executeReasoningStepStreaming executes a single reasoning step with streaming
func (a *Agent) executeReasoningStepStreaming(step ReasoningStep, ctx *ReasoningContext, modelNames ...string) (<-chan string, error) {
	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		// Build step-specific prompt
		prompt, err := a.buildStepPrompt(step, ctx)
		if err != nil {
			responseChan <- fmt.Sprintf("Failed to build step prompt: %v", err)
			return
		}

		// Use step-specific LLM config if available, otherwise use default agent
		llm := a.llm
		if step.AgentConfig != nil && step.AgentConfig.LLM.Name != "" {
			// Create a temporary LLM with step-specific config
			llm = a.createStepLLM(&step.AgentConfig.LLM)
		}

		// For execution steps, check if tools should be used
		if step.Type == "execute" && a.mcp != nil && len(a.mcp.ListTools()) > 0 {
			// Check if the step instruction mentions using tools
			instruction := ""
			if step.AgentConfig != nil {
				instruction = ""
			}
			if instruction != "" && strings.Contains(strings.ToLower(instruction), "tool") {
				// Execute tools generically based on LLM reasoning
				toolResults := a.executeToolsForQuery(ctx.Query)
				if len(toolResults) > 0 {
					// Include tool results in the prompt
					toolContext := a.buildToolContext(toolResults)
					prompt = prompt + "\n\nTool Results:\n" + toolContext
				}
			}
		}

		// Stream LLM generation
		streamChan, err := llm.GenerateStreaming(prompt)
		if err != nil {
			responseChan <- fmt.Sprintf("LLM generation failed: %v", err)
			return
		}

		// Forward streaming chunks
		for chunk := range streamChan {
			responseChan <- chunk
		}
	}()

	return responseChan, nil
}

// generateFinalResponseFromSteps generates a final response based on reasoning step results
func (a *Agent) generateFinalResponseFromSteps(ctx *ReasoningContext) string {
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

// executeSingleShot executes a single-shot query (current behavior)
func (a *Agent) executeSingleShot(ctx *ReasoningContext, modelNames ...string) (*AgentResponse, error) {
	return a.ExecuteQuery(ctx.Query, modelNames...)
}

// executeIterative executes iterative reasoning with retry logic
func (a *Agent) executeIterative(ctx *ReasoningContext, modelNames ...string) (*AgentResponse, error) {
	var lastResponse *AgentResponse
	var lastError error

	a.verbosePrint("Starting iterative reasoning (max %d steps)...\n", ctx.MaxSteps)

	for step := 0; step < ctx.MaxSteps; step++ {
		ctx.CurrentStep = step

		a.verbosePrint("\nIteration %d/%d\n", step+1, ctx.MaxSteps)

		// Execute query
		response, err := a.ExecuteQuery(ctx.Query, modelNames...)
		if err != nil {
			a.verbosePrint("Iteration failed: %v\n", err)
			ctx.ErrorHistory = append(ctx.ErrorHistory, err.Error())
			lastError = err

			// Retry logic
			if a.ReasoningConfig.EnableRetry && ctx.RetryCount < ctx.MaxRetries {
				ctx.RetryCount++
				a.verbosePrint("Retrying... (attempt %d/%d)\n", ctx.RetryCount, ctx.MaxRetries)
				continue
			}
			break
		}

		a.verbosePrint("Iteration completed (confidence: %.2f, tokens: %d)\n",
			response.Confidence, response.TokensUsed)
		a.verbosePrint("Response: %s\n", truncateString(response.Answer, 150))

		lastResponse = response

		// Check if we should continue based on response quality
		if a.shouldContinueReasoning(response, ctx) {
			a.verbosePrint("Response quality low, continuing to next iteration...\n")
			// Enhance query based on previous results
			ctx.Query = a.buildEnhancedQuery(ctx.Query, response.ToolResults)
			ctx.ExecutionHistory = append(ctx.ExecutionHistory, response.Answer)
		} else {
			a.verbosePrint("Response quality sufficient, stopping iterations\n")
			break
		}
	}

	if lastError != nil && lastResponse == nil {
		return nil, lastError
	}

	return lastResponse, nil
}

// executeStateMachine executes custom reasoning steps
func (a *Agent) executeStateMachine(ctx *ReasoningContext, modelNames ...string) (*AgentResponse, error) {
	var finalResponse *AgentResponse

	a.verbosePrint("Starting reasoning with %d steps...\n", len(a.ReasoningConfig.Steps))

	for i, step := range a.ReasoningConfig.Steps {
		if !step.Enabled {
			continue
		}

		ctx.CurrentStep++
		a.verbosePrint("\nStep %d/%d: %s (%s)\n", i+1, len(a.ReasoningConfig.Steps), step.Name, step.Type)
		a.verbosePrint("Instruction: %s\n", func() string {
			// No instruction field in AgentConfig anymore - use empty string
			return ""
		}())

		stepResult := a.executeReasoningStep(step, ctx, modelNames...)
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
		a.updateReasoningContext(step, stepResult, ctx)
	}

	// Generate final response
	a.verbosePrint("\nGenerating final response...\n")
	finalResponse = a.generateFinalResponse(ctx)
	return finalResponse, nil
}

// executeReasoningStep executes a single reasoning step
func (a *Agent) executeReasoningStep(step ReasoningStep, ctx *ReasoningContext, _ ...string) ReasoningStepResult {
	startTime := time.Now()

	result := ReasoningStepResult{
		StepName: step.Name,
		StepType: step.Type,
		Instruction: func() string {
			// No instruction field in AgentConfig anymore - use empty string
			return ""
		}(),
		Input: ctx.Query,
		LLMConfig: func() *YAMLProviderConfig {
			if step.AgentConfig != nil {
				return &step.AgentConfig.LLM
			}
			return nil
		}(),
		Config:  step.Config,
		Success: false,
	}

	// Build step-specific prompt
	prompt, err := a.buildStepPrompt(step, ctx)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to build step prompt: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute step based on type
	result.Output, result.TokensUsed, err = a.executeStep(step, prompt, ctx)

	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	result.Duration = time.Since(startTime)
	return result
}

// buildStepPrompt builds a prompt for a specific reasoning step
func (a *Agent) buildStepPrompt(step ReasoningStep, ctx *ReasoningContext) (string, error) {
	// Use step-specific instruction if available
	instruction := ""
	if step.AgentConfig != nil {
		// No instruction field in AgentConfig anymore - use empty string
		instruction = ""
	}
	if instruction == "" {
		instruction = "You are a helpful AI assistant."
	}

	// Use step-specific prompt template if available
	template := ""
	if step.AgentConfig != nil {
		// No prompt template field in AgentConfig anymore - use empty string
		template = ""
	}
	if template == "" {
		template = ""
	}

	// Build context string
	contextStr := strings.Join(ctx.Context, "\n")
	if contextStr == "" {
		contextStr = "No additional context available."
	}

	// Build previous step results
	previousResults := ""
	if len(ctx.StepResults) > 0 {
		previousResults = "Previous Step Results:\n"
		for _, result := range ctx.StepResults {
			previousResults += fmt.Sprintf("- %s: %s\n", result.StepName, result.Output)
		}
	}

	// Use step-specific template or default
	if template != "" {
		return BuildPrompt(ctx.Query, []string{contextStr}, "reasoning", instruction, template)
	}

	// Default step prompt
	prompt := fmt.Sprintf(`%s

Step: %s (%s)
%s
Query: %s
Context: %s

Please execute this step according to your role and provide a detailed response.`,
		instruction, step.Name, step.Type, previousResults, ctx.Query, contextStr)

	// Debug: Print the built prompt (commented out for cleaner output)
	// fmt.Printf("DEBUG: Built prompt for step %s (%s):\n%s\n", step.Name, step.Type, prompt)

	return prompt, nil
}

// executeStep executes any reasoning step with optional tool usage
func (a *Agent) executeStep(step ReasoningStep, prompt string, ctx *ReasoningContext) (string, int, error) {
	// Use step-specific LLM config if available, otherwise use default agent
	llm := a.llm
	if step.AgentConfig != nil && step.AgentConfig.LLM.Name != "" {
		// Create a temporary LLM with step-specific config
		llm = a.createStepLLM(&step.AgentConfig.LLM)
	}

	// For execution steps, check if tools should be used based on step instruction
	if step.Type == "execute" && a.mcp != nil && len(a.mcp.ListTools()) > 0 {
		// Check if the step instruction mentions using tools
		instruction := ""
		if step.AgentConfig != nil {
			// No instruction field in AgentConfig anymore - use empty string
			instruction = ""
		}
		if instruction != "" && strings.Contains(strings.ToLower(instruction), "tool") {
			// Execute tools generically based on LLM reasoning
			toolResults := a.executeToolsForQuery(ctx.Query)
			if len(toolResults) > 0 {
				// Include tool results in the prompt
				toolContext := a.buildToolContext(toolResults)
				enhancedPrompt := prompt + "\n\nTool Results:\n" + toolContext
				return llm.Generate(enhancedPrompt)
			}
		}
	}

	// Regular LLM generation for all step types
	return llm.Generate(prompt)
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

// shouldContinueReasoning determines if reasoning should continue
func (a *Agent) shouldContinueReasoning(response *AgentResponse, _ *ReasoningContext) bool {
	// Simple heuristic - continue if confidence is low or if there were tool errors
	if response.Confidence < 0.7 {
		return true
	}

	if len(response.ToolResults) > 0 {
		for _, result := range response.ToolResults {
			if !result.Success {
				return true
			}
		}
	}

	return false
}

// handleStepFailure handles step failure with simple retry logic
func (a *Agent) handleStepFailure(_ ReasoningStep, _ ReasoningStepResult, ctx *ReasoningContext) bool {
	// Simple retry logic - continue if we haven't exceeded max retries
	return ctx.RetryCount < ctx.MaxRetries
}

// updateReasoningContext updates context for next step
func (a *Agent) updateReasoningContext(_ ReasoningStep, result ReasoningStepResult, ctx *ReasoningContext) {
	if result.Success {
		ctx.Context = append(ctx.Context, fmt.Sprintf("%s: %s", result.StepName, result.Output))
	}
}

// generateFinalResponse generates the final response from reasoning results
func (a *Agent) generateFinalResponse(ctx *ReasoningContext) *AgentResponse {
	// Combine all step results into a final response
	var answer strings.Builder
	var totalTokens int

	a.verbosePrint("Reasoning Summary:\n")
	a.verbosePrint("   Steps completed: %d/%d\n", len(ctx.StepResults), ctx.MaxSteps)
	a.verbosePrint("   Errors encountered: %d\n", len(ctx.ErrorHistory))
	a.verbosePrint("   Retries used: %d/%d\n", ctx.RetryCount, ctx.MaxRetries)

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
		Answer:         answer.String(),
		TokensUsed:     totalTokens,
		ReasoningSteps: ctx.StepResults,
	}
}

// ExecuteQuery performs an agent query with tool usage
func (a *Agent) ExecuteQuery(query string, modelNames ...string) (*AgentResponse, error) {
	if a.llm == nil {
		return nil, fmt.Errorf("no LLM provider configured")
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
	if a.llm == nil {
		return nil, fmt.Errorf("no LLM provider configured")
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

// executeToolsForQuery uses LLM reasoning to determine which tools to execute via MCP
func (a *Agent) executeToolsForQuery(query string) map[string]ToolResult {
	toolResults := make(map[string]ToolResult)

	// Get available tools from MCP
	availableTools := a.mcp.ListTools()
	if len(availableTools) == 0 {
		return toolResults
	}

	// Create structured tool information for LLM
	toolInfo := a.createToolInfoForLLM(availableTools)

	// Ask LLM to reason about tool usage
	toolReasoningPrompt := fmt.Sprintf(`You are an AI assistant that needs to decide which tools to use for a user query.

Available tools:
%s

User query: "%s"

Based on the query, determine which tools (if any) would be helpful. Respond with a JSON object containing:
- "reasoning": Brief explanation of why these tools are needed
- "tools": Array of tool names to execute
- "parameters": Object with parameters for each tool

IMPORTANT: Use the exact parameter names from the tool schema above. For example, if a tool requires a "location" parameter, use "location" not "city" or "place".

Example response for tool usage:
{
  "reasoning": "User needs specific functionality",
  "tools": ["TOOL_NAME"],
  "parameters": {
    "TOOL_NAME": {"param1": "value1", "param2": "value2"}
  }
}

If no tools are needed, respond with:
{
  "reasoning": "Query can be answered with existing knowledge",
  "tools": [],
  "parameters": {}
}`, toolInfo, query)

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

	// Execute the recommended tools via MCP
	for _, toolName := range toolDecisions.Tools {
		params := toolDecisions.Parameters[toolName]
		if params == nil {
			// If no parameters specified, use empty parameters
			params = map[string]interface{}{}
		}

		// Execute tool via MCP

		// Type assert to map[string]interface{}
		if paramMap, ok := params.(map[string]interface{}); ok {
			result, err := a.mcp.ExecuteTool(context.Background(), toolName, paramMap)
			if err == nil {
				toolResults[toolName] = result
			} else {
				// Tool execution failed, store error result and log it
				fmt.Printf("Tool execution failed for %s: %v\n", toolName, err)
				errorResult := ToolResult{
					Content:  "",
					Success:  false,
					Error:    err.Error(),
					ToolName: toolName,
				}
				toolResults[toolName] = errorResult
			}
		}
	}

	return toolResults
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

	// Try to parse as JSON first
	var jsonResponse struct {
		Reasoning  string                 `json:"reasoning"`
		Tools      []string               `json:"tools"`
		Parameters map[string]interface{} `json:"parameters"`
	}

	if err := json.Unmarshal([]byte(response), &jsonResponse); err == nil {
		// Successfully parsed JSON
		decision.Reasoning = jsonResponse.Reasoning
		decision.Tools = jsonResponse.Tools
		decision.Parameters = jsonResponse.Parameters
		return decision
	}

	// If JSON parsing fails, return empty decision
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
	if a.llm == nil {
		return nil, fmt.Errorf("no LLM provider configured")
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
	if a.embedder == nil {
		return fmt.Errorf("no embedder configured")
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
	if !a.ReasoningConfig.Verbose {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Parse and execute the template
	tmpl, err := template.New("verbose").Parse(a.ReasoningConfig.VerboseTemplate)
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
	if a.modelManager == nil {
		return fmt.Errorf("ModelManager not initialized")
	}
	return a.modelManager.SyncModel(modelName)
}

// SyncAllModels syncs all models that have ingestion configuration
func (a *Agent) SyncAllModels() error {
	if a.modelManager == nil {
		return fmt.Errorf("ModelManager not initialized")
	}
	return a.modelManager.SyncAllModels()
}

// GetModelStatus returns the status of a model
func (a *Agent) GetModelStatus(modelName string) (map[string]interface{}, error) {
	if a.modelManager == nil {
		return nil, fmt.Errorf("ModelManager not initialized")
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
