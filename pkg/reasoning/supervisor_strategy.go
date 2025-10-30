package reasoning

import (
	"fmt"

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

	if iteration == 1 && state.GetServices() != nil {
		cfg := state.GetServices().GetConfig()
		if cfg.EnableGoalExtraction {

			decomposition, err := ExtractGoals(state.GetContext(), state.Query(), []string{}, state.GetServices())
			if err == nil {

				state.GetCustomState()["task_decomposition"] = decomposition

				if state.ShowThinking() && state.GetOutputChannel() != nil {
					s.displayTaskDecomposition(decomposition, state.GetOutputChannel())
				}
			}
		}
	}

	return s.ChainOfThoughtStrategy.PrepareIteration(iteration, state)
}

func (s *SupervisorStrategy) displayTaskDecomposition(decomposition *TaskDecomposition, outputCh chan<- *pb.Part) {
	output := ThinkingBlock(fmt.Sprintf("Task Decomposition: %s", decomposition.MainGoal))
	output += ThinkingBlock(fmt.Sprintf("Execution Order: %s", decomposition.ExecutionOrder))
	output += ThinkingBlock(fmt.Sprintf("Required Agents: %v", decomposition.RequiredAgents))
	output += ThinkingBlock(fmt.Sprintf("Strategy: %s", decomposition.Strategy))

	if len(decomposition.Subtasks) > 0 {
		for i, task := range decomposition.Subtasks {
			deps := "none"
			if len(task.DependsOn) > 0 {
				deps = fmt.Sprintf("%v", task.DependsOn)
			}
			output += ThinkingBlock(fmt.Sprintf("Subtask %d: [P%d] %s (agent: %s, depends: %s)", i+1, task.Priority, task.Description, task.AgentType, deps))
		}
	}

	outputCh <- createTextPart(output)
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

	baseContext := s.ChainOfThoughtStrategy.GetContextInjection(state)

	agentsList := s.buildAvailableAgentsContext(state)

	if agentsList != "" {
		if baseContext != "" {
			return baseContext + "\n\n" + agentsList
		}
		return agentsList
	}

	return baseContext
}

func (s *SupervisorStrategy) buildAvailableAgentsContext(state *ReasoningState) string {
	availableAgents := s.getAvailableAgents(state)

	if len(availableAgents) == 0 {
		return ""
	}

	context := "AVAILABLE AGENTS (THESE ARE THE ONLY AGENTS YOU CAN CALL):\n"
	for agentID, description := range availableAgents {
		context += fmt.Sprintf("- %s: %s\n", agentID, description)
	}
	context += "\nCRITICAL: You MUST ONLY use the agent IDs listed above.\n"
	context += "DO NOT invent or assume other agents exist.\n"
	context += "If a task needs a different type of agent, use the closest match from the list."

	return context
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

func (s *SupervisorStrategy) getAvailableAgents(state *ReasoningState) map[string]string {
	agents := make(map[string]string)

	if state == nil || state.GetServices() == nil || state.GetServices().Registry() == nil {
		return agents
	}

	registry := state.GetServices().Registry()

	currentAgentName := s.getCurrentAgentName(state)

	subAgents := s.getSubAgentsFromConfig(state)

	var agentEntries []AgentRegistryEntry
	if len(subAgents) > 0 {

		agentEntries = registry.FilterAgents(subAgents)
	} else {

		allAgents := registry.ListAgents()
		agentEntries = make([]AgentRegistryEntry, 0, len(allAgents))

		for _, entry := range allAgents {

			if entry.Visibility != "internal" {
				agentEntries = append(agentEntries, entry)
			}
		}
	}

	for _, entry := range agentEntries {

		if entry.ID != currentAgentName {

			agents[entry.ID] = entry.Card.Description
		}
	}

	return agents
}

func (s *SupervisorStrategy) getSubAgentsFromConfig(state *ReasoningState) []string {
	subAgents := state.SubAgents()
	if len(subAgents) == 0 {
		return nil
	}
	return subAgents
}

func (s *SupervisorStrategy) getCurrentAgentName(state *ReasoningState) string {
	return state.AgentName()
}

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
- When delegating to agents, briefly mention it ("Let me consult our analyst...")
- Balance transparency with conciseness
- Focus on delivering value to the user
- Provide unified, actionable responses`,

		CommunicationStyle: `Be natural and conversational:
- Briefly mention what you're doing when coordinating agents
- Keep explanations natural and brief
- Let the workflow feel smooth and effortless
- Present synthesized insights clearly`,
	}
}

func (s *SupervisorStrategy) GetName() string {
	return "supervisor"
}

func (s *SupervisorStrategy) GetDescription() string {
	return "Supervisor strategy optimized for multi-agent orchestration and delegation"
}
