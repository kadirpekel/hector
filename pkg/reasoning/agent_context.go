package reasoning

import (
	"fmt"
	"strings"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

// Built-in tool names - single source of truth
var builtInToolNames = map[string]bool{
	"execute_command": true,
	"write_file":      true,
	"read_file":       true,
	"search_replace":  true,
	"apply_patch":     true,
	"grep_search":     true,
	"search":          true,
	"todo":            true,
	"agent_call":      true,
	"web_request":     true,
}

// Constants for context building
const (
	maxDescriptionLength    = 100
	maxContextDocuments     = 5
	maxContextContentLength = 500
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

// BuildAvailableToolsContext builds context string categorizing available tools
// This provides a high-level overview of tool capabilities
func BuildAvailableToolsContext(state *ReasoningState) string {
	if state == nil || state.GetServices() == nil {
		return ""
	}

	toolService := state.GetServices().Tools()
	if toolService == nil {
		return ""
	}

	toolDefs := toolService.GetAvailableTools()
	if len(toolDefs) == 0 {
		return ""
	}

	// Categorize tools by common patterns
	builtIn := []string{}
	fileOps := []string{}
	codeOps := []string{}
	searchOps := []string{}
	otherOps := []string{}

	fileToolPatterns := []string{"write_file", "read_file", "search_replace", "apply_patch"}
	codeToolPatterns := []string{"grep_search", "search"}
	searchToolPatterns := []string{"search", "grep_search"}

	for _, toolDef := range toolDefs {
		name := toolDef.Name

		// Check if it's a known built-in
		if builtInToolNames[name] {
			builtIn = append(builtIn, name)

			// Also categorize by function
			for _, pattern := range fileToolPatterns {
				if name == pattern {
					fileOps = append(fileOps, name)
					break
				}
			}
			for _, pattern := range codeToolPatterns {
				if name == pattern {
					codeOps = append(codeOps, name)
					break
				}
			}
			for _, pattern := range searchToolPatterns {
				if name == pattern {
					searchOps = append(searchOps, name)
					break
				}
			}
		} else {
			// Likely MCP or plugin tool
			otherOps = append(otherOps, name)
		}
	}

	if len(builtIn) == 0 && len(otherOps) == 0 {
		return ""
	}

	context := "AVAILABLE TOOLS:\n"

	if len(builtIn) > 0 {
		context += "\nBuilt-in tools:\n"
		for _, tool := range builtIn {
			context += fmt.Sprintf("- %s\n", tool)
		}
	}

	if len(fileOps) > 0 {
		context += "\nFile operations: " + strings.Join(fileOps, ", ") + "\n"
	}
	if len(codeOps) > 0 {
		context += "Code search: " + strings.Join(codeOps, ", ") + "\n"
	}
	if len(searchOps) > 0 {
		context += "Search capabilities: " + strings.Join(searchOps, ", ") + "\n"
	}

	if len(otherOps) > 0 {
		context += "\nExternal integrations:\n"
		for _, tool := range otherOps {
			context += fmt.Sprintf("- %s\n", tool)
		}
		context += "(These may be MCP servers or custom plugins)\n"
	}

	context += "\nNOTE: All tools are available via function calling. Use the tool names exactly as listed above."

	return context
}

// BuildAvailableMCPIntegrationsContext builds context string listing MCP integrations
// This identifies external service integrations
func BuildAvailableMCPIntegrationsContext(state *ReasoningState) string {
	if state == nil || state.GetServices() == nil {
		return ""
	}

	toolService := state.GetServices().Tools()
	if toolService == nil {
		return ""
	}

	toolDefs := toolService.GetAvailableTools()
	if len(toolDefs) == 0 {
		return ""
	}

	// Identify likely MCP tools (tools that aren't standard built-ins)
	mcpTools := []string{}
	for _, toolDef := range toolDefs {
		if !builtInToolNames[toolDef.Name] {
			mcpTools = append(mcpTools, toolDef.Name)
		}
	}

	if len(mcpTools) == 0 {
		return ""
	}

	context := "EXTERNAL INTEGRATIONS (MCP/Plugins):\n"
	for _, tool := range mcpTools {
		// Try to get description
		var desc string
		for _, toolDef := range toolDefs {
			if toolDef.Name == tool {
				desc = toolDef.Description
				break
			}
		}

		if desc != "" && len(desc) > maxDescriptionLength {
			desc = desc[:maxDescriptionLength] + "..."
		}

		if desc != "" {
			context += fmt.Sprintf("- %s: %s\n", tool, desc)
		} else {
			context += fmt.Sprintf("- %s\n", tool)
		}
	}

	context += "\nThese tools connect to external services. Check tool descriptions for specific capabilities and requirements."

	return context
}

// BuildMemoryContext builds context string about memory capabilities
func BuildMemoryContext(state *ReasoningState) string {
	if state == nil || state.GetServices() == nil {
		return ""
	}

	sessionService := state.GetServices().Session()
	if sessionService == nil {
		return ""
	}

	// If session service exists, memory is available
	context := "MEMORY: You have access to persistent memory across sessions.\n"
	context += "Previous conversations are automatically loaded when relevant.\n"
	context += "You can reference past interactions and build on previous context."

	return context
}

// BuildCommonContext builds all common context that should be available to all strategies
// This includes: tools, document stores, memory, and other shared resources
// Strategies should call this and append their strategy-specific context
func BuildCommonContext(state *ReasoningState) string {
	if state == nil {
		return ""
	}

	var contextParts []string

	// Document stores (available to all)
	storesList := BuildAvailableDocumentStoresContext(state)
	if storesList != "" {
		contextParts = append(contextParts, storesList)
	}

	// Tool categorization (available to all)
	toolsList := BuildAvailableToolsContext(state)
	if toolsList != "" {
		contextParts = append(contextParts, toolsList)
	}

	// MCP integration details (available to all)
	mcpList := BuildAvailableMCPIntegrationsContext(state)
	if mcpList != "" {
		contextParts = append(contextParts, mcpList)
	}

	// Memory information (available to all)
	memoryInfo := BuildMemoryContext(state)
	if memoryInfo != "" {
		contextParts = append(contextParts, memoryInfo)
	}

	if len(contextParts) == 0 {
		return ""
	}

	return strings.Join(contextParts, "\n\n")
}

// BuildAvailableDocumentStoresContext builds context string listing available document stores
// This is a shared helper that can be used by any reasoning strategy
func BuildAvailableDocumentStoresContext(state *ReasoningState) string {
	if state == nil {
		return ""
	}

	// Get document stores from registry
	storeNames := hectorcontext.ListDocumentStoresFromRegistry()
	if len(storeNames) == 0 {
		return ""
	}

	context := "AVAILABLE DOCUMENT STORES (you can search these using the search tool):\n"

	var storeEntries []string
	for _, storeName := range storeNames {
		store, exists := hectorcontext.GetDocumentStoreFromRegistry(storeName)
		if !exists {
			continue
		}

		status := store.GetStatus()

		// Build description with metadata
		var descParts []string
		if status.SourcePath != "" {
			descParts = append(descParts, fmt.Sprintf("source: %s", status.SourcePath))
		}
		if status.DocumentCount > 0 {
			descParts = append(descParts, fmt.Sprintf("%d documents", status.DocumentCount))
		}

		description := storeName
		if len(descParts) > 0 {
			description += fmt.Sprintf(" (%s)", strings.Join(descParts, ", "))
		}

		storeEntries = append(storeEntries, fmt.Sprintf("- %s", description))
	}

	if len(storeEntries) == 0 {
		return ""
	}

	context += strings.Join(storeEntries, "\n")
	context += "\n\nIMPORTANT: When using the search tool, you can specify which stores to search using the 'stores' parameter.\n"
	context += "If you omit 'stores', all available stores will be searched.\n"
	context += "Use specific store names when you know which store contains the information you need."

	return context
}
