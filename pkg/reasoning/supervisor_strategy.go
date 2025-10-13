package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// SUPERVISOR STRATEGY
// Optimized for multi-agent orchestration and delegation
// ============================================================================

// SupervisorStrategy implements a reasoning strategy specialized for coordinating
// multiple agents. It provides better prompting and logic for:
// - Task decomposition
// - Agent selection and delegation
// - Result synthesis
type SupervisorStrategy struct {
	// Embed ChainOfThought for base functionality
	*ChainOfThoughtStrategy
}

// NewSupervisorStrategy creates a new supervisor strategy
func NewSupervisorStrategy() *SupervisorStrategy {
	return &SupervisorStrategy{
		ChainOfThoughtStrategy: NewChainOfThoughtStrategy(),
	}
}

// PrepareIteration implements ReasoningStrategy
// Supervisor extracts goals on first iteration (if enabled), then delegates to base strategy
func (s *SupervisorStrategy) PrepareIteration(iteration int, state *ReasoningState) error {
	// On first iteration, optionally extract goals for task decomposition
	if iteration == 1 && state.Services != nil {
		cfg := state.Services.GetConfig()
		if cfg.EnableGoalExtraction {
			// Extract goals using structured output
			decomposition, err := ExtractGoals(state.Context, state.Query, []string{}, state.Services)
			if err == nil {
				// Store decomposition in custom state
				state.CustomState["task_decomposition"] = decomposition

				// Display decomposition if debug enabled
				if state.ShowDebugInfo && state.OutputChannel != nil {
					s.displayTaskDecomposition(decomposition, state.OutputChannel)
				}
			}
		}
	}

	// Delegate to base strategy
	return s.ChainOfThoughtStrategy.PrepareIteration(iteration, state)
}

// displayTaskDecomposition shows the extracted task plan to the user
func (s *SupervisorStrategy) displayTaskDecomposition(decomposition *TaskDecomposition, outputCh chan<- string) {
	outputCh <- "\033[90m\n📋 **Task Decomposition:**\n"
	outputCh <- fmt.Sprintf("  - Main Goal: %s\n", decomposition.MainGoal)
	outputCh <- fmt.Sprintf("  - Execution Order: %s\n", decomposition.ExecutionOrder)
	outputCh <- fmt.Sprintf("  - Required Agents: %v\n", decomposition.RequiredAgents)
	outputCh <- fmt.Sprintf("  - Strategy: %s\n", decomposition.Strategy)
	if len(decomposition.Subtasks) > 0 {
		outputCh <- fmt.Sprintf("  - Subtasks (%d):\n", len(decomposition.Subtasks))
		for i, task := range decomposition.Subtasks {
			deps := "none"
			if len(task.DependsOn) > 0 {
				deps = fmt.Sprintf("%v", task.DependsOn)
			}
			outputCh <- fmt.Sprintf("    %d. [P%d] %s (agent: %s, depends: %s)\n", i+1, task.Priority, task.Description, task.AgentType, deps)
		}
	}
	outputCh <- "\033[0m"
}

// ShouldStop implements ReasoningStrategy
// Supervisor uses same stopping logic as chain-of-thought
func (s *SupervisorStrategy) ShouldStop(text string, toolCalls []*protocol.ToolCall, state *ReasoningState) bool {
	// Stop when no more tool calls (including agent_call)
	return s.ChainOfThoughtStrategy.ShouldStop(text, toolCalls, state)
}

// AfterIteration implements ReasoningStrategy
// Uses chain-of-thought's reflection logic
func (s *SupervisorStrategy) AfterIteration(
	iteration int,
	text string,
	toolCalls []*protocol.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) error {
	return s.ChainOfThoughtStrategy.AfterIteration(iteration, text, toolCalls, results, state)
}

// GetContextInjection implements ReasoningStrategy
// Injects both base context (todos) and available agents list
func (s *SupervisorStrategy) GetContextInjection(state *ReasoningState) string {
	// Get base context injection (todos, etc.)
	baseContext := s.ChainOfThoughtStrategy.GetContextInjection(state)

	// Build available agents list dynamically
	agentsList := s.buildAvailableAgentsContext(state)

	// Combine both contexts
	if agentsList != "" {
		if baseContext != "" {
			return baseContext + "\n\n" + agentsList
		}
		return agentsList
	}

	return baseContext
}

// buildAvailableAgentsContext builds the agent list context injection
func (s *SupervisorStrategy) buildAvailableAgentsContext(state *ReasoningState) string {
	availableAgents := s.getAvailableAgents(state)

	if len(availableAgents) == 0 {
		return ""
	}

	context := "⚠️ AVAILABLE AGENTS (THESE ARE THE ONLY AGENTS YOU CAN CALL):\n"
	for agentID, description := range availableAgents {
		context += fmt.Sprintf("- %s: %s\n", agentID, description)
	}
	context += "\n⚠️ CRITICAL: You MUST ONLY use the agent IDs listed above.\n"
	context += "DO NOT invent or assume other agents exist.\n"
	context += "If a task needs a different type of agent, use the closest match from the list."

	return context
}

// GetRequiredTools implements ReasoningStrategy
// Supervisor requires both todo_write and agent_call for orchestration
func (s *SupervisorStrategy) GetRequiredTools() []RequiredTool {
	return []RequiredTool{
		{
			Name:        "todo_write",
			Type:        "todo",
			Description: "Required for breaking down orchestration tasks",
			AutoCreate:  true,
		},
		{
			Name:        "agent_call",
			Type:        "agent_call",
			Description: "Required for delegating tasks to other agents in multi-agent orchestration",
			AutoCreate:  true, // Can be auto-created when registry is available
		},
	}
}

// getAvailableAgents returns map of available agents from reasoning state
// Uses agent registry service from state.Services and filters by:
// - sub_agents config (explicit whitelist)
// - visibility rules (for auto-discovery)
// - current agent (prevent self-calls)
// Returns map[agentID]description for LLM prompt injection
func (s *SupervisorStrategy) getAvailableAgents(state *ReasoningState) map[string]string {
	agents := make(map[string]string)

	// Get registry service from services
	if state == nil || state.Services == nil || state.Services.Registry() == nil {
		return agents
	}

	registry := state.Services.Registry()

	// Get current agent name to exclude it from available agents
	currentAgentName := s.getCurrentAgentName(state)

	// Check if we should filter to specific sub-agents
	subAgents := s.getSubAgentsFromConfig(state)

	// Get agent entries from registry service (returns A2A AgentCards)
	var agentEntries []AgentRegistryEntry
	if len(subAgents) > 0 {
		// EXPLICIT MODE: sub_agents specified - show all listed agents regardless of visibility
		// This allows orchestrators to explicitly control which agents (including internal) they can call
		agentEntries = registry.FilterAgents(subAgents)
	} else {
		// AUTO-DISCOVERY MODE: No sub_agents - apply visibility rules
		// Show: public + private (NOT internal, which requires explicit knowledge)
		allAgents := registry.ListAgents()
		agentEntries = make([]AgentRegistryEntry, 0, len(allAgents))

		for _, entry := range allAgents {
			// Include public and private agents for auto-discovery
			// Exclude internal agents (they require explicit listing in sub_agents)
			if entry.Visibility != "internal" {
				agentEntries = append(agentEntries, entry)
			}
		}
	}

	// Build map for prompt injection, excluding current agent
	for _, entry := range agentEntries {
		// Skip the current agent to prevent self-calls
		if entry.ID != currentAgentName {
			// Use A2A AgentCard description
			agents[entry.ID] = entry.Card.Description
		}
	}

	return agents
}

// getSubAgentsFromConfig extracts sub_agents list from state's custom data
func (s *SupervisorStrategy) getSubAgentsFromConfig(state *ReasoningState) []string {
	if subAgents, ok := state.CustomState["sub_agents"].([]string); ok {
		return subAgents
	}
	return nil
}

// getCurrentAgentName extracts the current agent's name from state
func (s *SupervisorStrategy) getCurrentAgentName(state *ReasoningState) string {
	if agentName, ok := state.CustomState["agent_name"].(string); ok {
		return agentName
	}
	return ""
}

// buildToolUsageGuidance creates agent_call usage guidance
// Note: Specific agent examples are injected via GetContextInjection()
func (s *SupervisorStrategy) buildToolUsageGuidance() string {
	return `Use agent_call to delegate tasks to specialized agents.

When you need another agent's expertise:
- Identify which agent has the needed capabilities
- Call agent_call with the agent ID and a clear task description
- Provide sufficient context in the task description
- Build on previous results when making sequential calls

Common orchestration patterns:
- Sequential: Output of one agent feeds into the next
- Parallel: Multiple independent tasks executed simultaneously
- Hierarchical: Break down complex tasks, delegate parts, then synthesize

Think strategically about which agents to involve and in what order.`
}

// GetPromptSlots implements ReasoningStrategy
// Supervisor provides specialized prompts for orchestration
// Note: Available agents are injected dynamically via GetContextInjection(), not here
func (s *SupervisorStrategy) GetPromptSlots() PromptSlots {
	return PromptSlots{
		SystemRole: `You are a supervisor agent that coordinates multiple specialized agents to accomplish complex tasks.

Your role is to:
1. Analyze incoming requests and break them into subtasks
2. Identify which agents have the capabilities needed for each subtask
3. Delegate work to appropriate agents using the agent_call tool
4. Synthesize results from multiple agents into coherent responses

Think strategically about task decomposition and agent coordination.`,

		ReasoningInstructions: `ORCHESTRATION PROCESS:

1. ANALYZE: Understand the user's goal and identify what needs to be done
2. PLAN: Break the task into clear, independent subtasks
3. DELEGATE: Use agent_call to assign subtasks to appropriate agents
4. SYNTHESIZE: Combine agent outputs into a unified, coherent response

DELEGATION GUIDELINES:
- Each agent_call should have a clear, focused task
- Build on previous results (reference them in subsequent calls)
- Don't just concatenate outputs - add value through synthesis
- Consider task dependencies (what info do you need first?)

SYNTHESIS GUIDELINES:
- Find connections and insights across agent outputs
- Resolve conflicts or inconsistencies
- Present a unified perspective
- Credit agents when their insights are valuable`,

		ToolUsage: s.buildToolUsageGuidance(),

		OutputFormat: `Present results in a clear, organized way:
- Explain your orchestration strategy
- Show which agents you consulted
- Synthesize their insights
- Provide a unified, actionable response`,

		CommunicationStyle: `Be clear and systematic:
- Explain your orchestration approach
- Show your reasoning about agent selection
- Acknowledge when building on previous results
- Present synthesized insights, not just concatenated outputs`,
	}
}

// GetName implements ReasoningStrategy
func (s *SupervisorStrategy) GetName() string {
	return "supervisor"
}

// GetDescription implements ReasoningStrategy
func (s *SupervisorStrategy) GetDescription() string {
	return "Supervisor strategy optimized for multi-agent orchestration and delegation"
}
