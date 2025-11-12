package reasoning

import (
	"fmt"
)

// AgentContextOptions controls how agent context is built
type AgentContextOptions struct {
	// OnlySubAgents: if true, only list sub-agents from config; if false, list all available agents
	OnlySubAgents bool
	// ExcludeCurrentAgent: if true, exclude the current agent from the list
	ExcludeCurrentAgent bool
	// IncludeInternal: if true, include agents with "internal" visibility
	IncludeInternal bool
	// MessageStyle: "supervisor" (strict) or "assistant" (helpful)
	MessageStyle string
}

// DefaultAgentContextOptions returns default options for chain-of-thought strategy
func DefaultAgentContextOptions() AgentContextOptions {
	return AgentContextOptions{
		OnlySubAgents:       true,
		ExcludeCurrentAgent: false,
		IncludeInternal:     false,
		MessageStyle:        "assistant",
	}
}

// SupervisorAgentContextOptions returns options for supervisor strategy
func SupervisorAgentContextOptions() AgentContextOptions {
	return AgentContextOptions{
		OnlySubAgents:       false, // Supervisor can see all agents
		ExcludeCurrentAgent: true,  // Don't include self
		IncludeInternal:     false, // Exclude internal agents
		MessageStyle:        "supervisor",
	}
}

// BuildAvailableAgentsContext builds context string listing available agents
// This is a shared helper that can be used by any reasoning strategy
func BuildAvailableAgentsContext(state *ReasoningState, opts AgentContextOptions) string {
	if state == nil || state.GetServices() == nil || state.GetServices().Registry() == nil {
		return ""
	}

	registry := state.GetServices().Registry()
	var agentEntries []AgentRegistryEntry

	if opts.OnlySubAgents {
		// Only list sub-agents from config
		subAgents := state.SubAgents()
		if len(subAgents) == 0 {
			return ""
		}
		agentEntries = registry.FilterAgents(subAgents)
	} else {
		// List all available agents (supervisor mode)
		allAgents := registry.ListAgents()
		agentEntries = make([]AgentRegistryEntry, 0, len(allAgents))

		for _, entry := range allAgents {
			// Filter by visibility
			if !opts.IncludeInternal && entry.Visibility == "internal" {
				continue
			}
			// Exclude current agent if requested
			if opts.ExcludeCurrentAgent && entry.ID == state.AgentName() {
				continue
			}
			agentEntries = append(agentEntries, entry)
		}
	}

	if len(agentEntries) == 0 {
		return ""
	}

	// Build unified context message (same content for both styles, with optional tone adjustment)
	var header string
	var criticalPrefix string
	if opts.MessageStyle == "supervisor" {
		header = "AVAILABLE AGENTS (THESE ARE THE ONLY AGENTS YOU CAN CALL):\n"
		criticalPrefix = "CRITICAL"
	} else {
		header = "AVAILABLE AGENTS (you can call these using the agent_call tool):\n"
		criticalPrefix = "IMPORTANT"
	}

	context := header
	for _, entry := range agentEntries {
		description := entry.Card.Description
		if description == "" {
			description = entry.Card.Name
		}
		context += fmt.Sprintf("- %s: %s\n", entry.ID, description)
	}

	// Unified instructions (same for both styles - all important information)
	context += fmt.Sprintf("\n%s: You MUST use the exact agent IDs listed above (e.g., agent='weather_assistant').\n", criticalPrefix)
	context += "DO NOT invent or assume other agent names exist.\n"
	context += "DO NOT abbreviate agent names (e.g., use 'weather_assistant' not 'weather').\n"
	context += "\nTo call an agent, use the agent_call tool with:\n"
	context += "  - agent: the exact agent ID from the list above\n"
	context += "  - task: your request or question for that agent\n"
	context += "\nWhen the user's request relates to information or capabilities that an available agent provides, you MUST call that agent first before responding. For example, if asked about weather, activities, or plans, call the weather_assistant agent to get current weather conditions."

	if opts.MessageStyle == "supervisor" {
		context += "\n\nIf a task needs a different type of agent, use the closest match from the list."
	}

	return context
}
