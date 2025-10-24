package reasoning

type PromptSlots struct {
	SystemRole string

	ReasoningInstructions string

	ToolUsage string

	OutputFormat string

	CommunicationStyle string

	Additional string
}

func (s *PromptSlots) IsEmpty() bool {
	return s.SystemRole == "" &&
		s.ReasoningInstructions == "" &&
		s.ToolUsage == "" &&
		s.OutputFormat == "" &&
		s.CommunicationStyle == "" &&
		s.Additional == ""
}

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
