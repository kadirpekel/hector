package reasoning

import (
	"github.com/kadirpekel/hector/llms"
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
// Supervisor doesn't need special preparation beyond chain-of-thought
func (s *SupervisorStrategy) PrepareIteration(iteration int, state *ReasoningState) error {
	// Delegate to base strategy
	return s.ChainOfThoughtStrategy.PrepareIteration(iteration, state)
}

// ShouldStop implements ReasoningStrategy
// Supervisor uses same stopping logic as chain-of-thought
func (s *SupervisorStrategy) ShouldStop(text string, toolCalls []llms.ToolCall, state *ReasoningState) bool {
	// Stop when no more tool calls (including agent_call)
	return s.ChainOfThoughtStrategy.ShouldStop(text, toolCalls, state)
}

// AfterIteration implements ReasoningStrategy
// Uses chain-of-thought's reflection logic
func (s *SupervisorStrategy) AfterIteration(
	iteration int,
	text string,
	toolCalls []llms.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) error {
	return s.ChainOfThoughtStrategy.AfterIteration(iteration, text, toolCalls, results, state)
}

// GetContextInjection implements ReasoningStrategy
// Uses chain-of-thought's todo injection
func (s *SupervisorStrategy) GetContextInjection(state *ReasoningState) string {
	return s.ChainOfThoughtStrategy.GetContextInjection(state)
}

// GetRequiredTools implements ReasoningStrategy
// Supervisor requires todo_write (from base) but NOT agent_call
// Note: agent_call must be registered separately after agent registry is populated
func (s *SupervisorStrategy) GetRequiredTools() []RequiredTool {
	return []RequiredTool{
		{
			Name:        "todo_write",
			Type:        "todo",
			Description: "Required for breaking down orchestration tasks",
			AutoCreate:  true,
		},
		// NOTE: agent_call is NOT listed here because:
		// - It requires AgentRegistry which doesn't exist at strategy creation time
		// - It's registered separately in cmd/hector/main.go after all agents are created
		// - User must explicitly add it to orchestrator agent's tools config
	}
}

// GetPromptSlots implements ReasoningStrategy
// Supervisor provides specialized prompts for orchestration
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

		ToolUsage: `Use agent_call to delegate tasks to specialized agents.

When you need another agent's expertise:
- Identify which agent has the needed capabilities
- Call agent_call with the agent name and a clear task description
- Provide sufficient context in the task description
- Build on previous results when making sequential calls

Common orchestration patterns:
- Sequential: Output of one agent feeds into the next
- Parallel: Multiple independent tasks executed simultaneously
- Hierarchical: Break down complex tasks, delegate parts, then synthesize

Think strategically about which agents to involve and in what order.`,

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
