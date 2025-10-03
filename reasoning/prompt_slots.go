package reasoning

// ============================================================================
// PROMPT SLOTS - PREDEFINED CONTRACT FOR PROMPT COMPOSITION
// ============================================================================

// PromptSlots defines the standard slots for composing system prompts
// This provides a fixed contract that all strategies can populate
// Agent merges strategy slots with user config, PromptService renders
type PromptSlots struct {
	// SystemRole defines the assistant's identity and role
	// Example: "You are an AI coding assistant" or "You are a debugging expert"
	SystemRole string

	// ReasoningInstructions defines how the assistant should approach reasoning
	// Example: "Use step-by-step reasoning" or "Use Thought/Action/Observation format"
	ReasoningInstructions string

	// ToolUsage defines how the assistant should use available tools
	// Example: "Use tools naturally to solve problems" or "Action: [tool_name]"
	ToolUsage string

	// OutputFormat defines the expected output structure
	// Example: "Be concise and clear" or "End with Final Answer:"
	OutputFormat string

	// CommunicationStyle defines the communication approach
	// Example: "Use markdown formatting" or "Be professional and direct"
	CommunicationStyle string

	// Additional provides a slot for user-specific customizations
	// This is where users can add domain-specific instructions
	Additional string
}

// IsEmpty returns true if all slots are empty
func (s *PromptSlots) IsEmpty() bool {
	return s.SystemRole == "" &&
		s.ReasoningInstructions == "" &&
		s.ToolUsage == "" &&
		s.OutputFormat == "" &&
		s.CommunicationStyle == "" &&
		s.Additional == ""
}

// Merge merges other slots into this one (non-empty values override)
func (s *PromptSlots) Merge(other PromptSlots) PromptSlots {
	merged := *s

	if other.SystemRole != "" {
		merged.SystemRole = other.SystemRole
	}
	if other.ReasoningInstructions != "" {
		merged.ReasoningInstructions = other.ReasoningInstructions
	}
	if other.ToolUsage != "" {
		merged.ToolUsage = other.ToolUsage
	}
	if other.OutputFormat != "" {
		merged.OutputFormat = other.OutputFormat
	}
	if other.CommunicationStyle != "" {
		merged.CommunicationStyle = other.CommunicationStyle
	}
	if other.Additional != "" {
		merged.Additional = other.Additional
	}

	return merged
}
