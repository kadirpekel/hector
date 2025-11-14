package reasoning

import (
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

type SupervisorStrategy struct {
	*ChainOfThoughtStrategy
}

func NewSupervisorStrategy() *SupervisorStrategy {
	return &SupervisorStrategy{
		ChainOfThoughtStrategy: NewChainOfThoughtStrategy(),
	}
}

func (s *SupervisorStrategy) PrepareIteration(iteration int, state *ReasoningState) error {

	if iteration == 1 && state.GetServices() != nil && state.ShowThinking() {
		decomposition, err := ExtractGoals(state.GetContext(), state.Query(), []string{}, state.GetServices())
		if err == nil {
			state.GetCustomState()["task_decomposition"] = decomposition

			if state.GetOutputChannel() != nil {
				s.displayTaskDecomposition(decomposition, state.GetOutputChannel())
			}
		}
	}

	return s.ChainOfThoughtStrategy.PrepareIteration(iteration, state)
}

func (s *SupervisorStrategy) displayTaskDecomposition(decomposition *TaskDecomposition, outputCh chan<- *pb.Part) {
	// Build text fallback for simple clients
	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("🎯 Goal: %s\n", decomposition.MainGoal))
	textBuilder.WriteString(fmt.Sprintf("📋 Strategy: %s\n", decomposition.Strategy))
	textBuilder.WriteString(fmt.Sprintf("🔄 Execution: %s\n", decomposition.ExecutionOrder))

	if len(decomposition.Subtasks) > 0 {
		textBuilder.WriteString("📝 Subtasks:\n")
		for i, task := range decomposition.Subtasks {
			deps := "none"
			if len(task.DependsOn) > 0 {
				deps = fmt.Sprintf("%v", task.DependsOn)
			}
			textBuilder.WriteString(fmt.Sprintf("  %d. [P%d] %s → %s (deps: %s)\n",
				i+1, task.Priority, task.Description, task.AgentType, deps))
		}
	}

	// Build structured data for rich clients
	subtasksData := make([]map[string]interface{}, len(decomposition.Subtasks))
	for i, task := range decomposition.Subtasks {
		subtasksData[i] = map[string]interface{}{
			"description": task.Description,
			"priority":    task.Priority,
			"agent_type":  task.AgentType,
			"depends_on":  task.DependsOn,
		}
	}

	data := map[string]interface{}{
		"main_goal":       decomposition.MainGoal,
		"strategy":        decomposition.Strategy,
		"execution_order": decomposition.ExecutionOrder,
		"required_agents": decomposition.RequiredAgents,
		"subtasks":        subtasksData,
	}

	// Emit as AG-UI thinking part with structured data
	// thinking_type is a hint for client rendering
	outputCh <- protocol.CreateThinkingPartWithData(textBuilder.String(), "goal", data)
}

func (s *SupervisorStrategy) ShouldStop(text string, toolCalls []*protocol.ToolCall, state *ReasoningState) bool {

	return s.ChainOfThoughtStrategy.ShouldStop(text, toolCalls, state)
}

func (s *SupervisorStrategy) AfterIteration(
	iteration int,
	text string,
	toolCalls []*protocol.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) error {
	return s.ChainOfThoughtStrategy.AfterIteration(iteration, text, toolCalls, results, state)
}

func (s *SupervisorStrategy) GetContextInjection(state *ReasoningState) string {
	// Supervisor uses the same common context as all strategies
	// This includes unified multi-agent foundation (respects sub_agents config)
	// Strategy differences are in HOW agents are used (orchestration), not WHICH agents are visible
	return s.ChainOfThoughtStrategy.GetContextInjection(state)
}

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
			AutoCreate:  true,
		},
	}
}

func (s *SupervisorStrategy) GetPromptSlots() PromptSlots {
	return PromptSlots{
		SystemRole: `You are a supervisor agent that coordinates multiple specialized agents to accomplish complex tasks.

Your role is to:
1. Analyze incoming requests and break them into subtasks
2. Identify which agents have the capabilities needed for each subtask
3. Delegate work to appropriate agents using the agent_call tool
4. Synthesize results from multiple agents into coherent responses

Think strategically about task decomposition and agent coordination.`,

		Instructions: `ORCHESTRATION PROCESS:
1. ANALYZE: Understand the user's goal and identify what needs to be done
2. PLAN: Break the task into clear, independent subtasks
3. DELEGATE: Use agent_call to assign subtasks to appropriate agents
4. SYNTHESIZE: Combine agent outputs into a unified, coherent response

DELEGATION:
- Each agent_call should have a clear, focused task
- Identify which agent has the needed capabilities
- Provide sufficient context in the task description
- Build on previous results when making sequential calls
- Consider task dependencies (what info do you need first?)
- Don't just concatenate outputs - add value through synthesis

DELEGATION ERROR HANDLING:
When an agent_call fails or returns incomplete results:
1. Show the error or incomplete output to the user (don't hide delegation issues)
2. Analyze why it failed (wrong agent, insufficient context, capability gap)
3. Try alternatives (different agent, more context, break into subtasks)
4. Redelegate after adjustments
5. Only escalate to user after trying multiple delegation strategies

ORCHESTRATION PATTERNS:
- Sequential: Output of one agent feeds into the next
- Parallel: Multiple independent tasks executed simultaneously
- Hierarchical: Break down complex tasks, delegate parts, then synthesize

SYNTHESIS:
- Find connections and insights across agent outputs
- Resolve conflicts or inconsistencies
- Present a unified perspective
- Credit agents when their insights are valuable
- If agents gave partial/error results, synthesize what worked

COMMUNICATION:
- When delegating, briefly mention it naturally ("Let me consult our analyst...")
- Balance transparency with conciseness
- Focus on delivering value to the user
- Present synthesized insights clearly
- Keep explanations natural and brief
- Let the workflow feel smooth and effortless`,

		UserGuidance: "",
	}
}

func (s *SupervisorStrategy) GetName() string {
	return "supervisor"
}

func (s *SupervisorStrategy) GetDescription() string {
	return "Supervisor strategy optimized for multi-agent orchestration and delegation"
}
