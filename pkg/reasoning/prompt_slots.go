package reasoning

type PromptSlots struct {
	// SystemRole defines the agent's identity, purpose, and core mission.
	// This answers "WHO is the agent?"
	SystemRole string

	// Instructions contains all behavioral guidance:
	// - Execution principles (how to approach tasks)
	// - Communication style (how to present information)
	// - Tool usage patterns (how to use tools effectively)
	// - Workflow patterns (how to structure work)
	// This answers "HOW should the agent behave?"
	Instructions string

	// UserGuidance contains user-provided custom instructions.
	// Set via --instruction flag or config prompt_slots.user_guidance.
	// This answers "WHAT specific guidance has the user provided?"
	// This is applied LAST and has highest priority.
	UserGuidance string
}

func (s *PromptSlots) IsEmpty() bool {
	return s.SystemRole == "" &&
		s.Instructions == "" &&
		s.UserGuidance == ""
}

func (s *PromptSlots) Merge(other PromptSlots) PromptSlots {
	merged := *s

	if other.SystemRole != "" {
		merged.SystemRole = other.SystemRole
	}
	if other.Instructions != "" {
		merged.Instructions = other.Instructions
	}
	if other.UserGuidance != "" {
		merged.UserGuidance = other.UserGuidance
	}

	return merged
}
