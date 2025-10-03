package reasoning

import (
	"context"
	"strings"

	"github.com/kadirpekel/hector/llms"
)

// ============================================================================
// REASONING STATE
// Shared state passed between iterations
// ============================================================================

// ReasoningState holds the state of the reasoning process
// This is passed to strategies so they can maintain and update state
type ReasoningState struct {
	// Current iteration number
	Iteration int

	// Total tokens used across all iterations
	TotalTokens int

	// Original user query (for strategies to reference)
	Query string

	// Current conversation messages (for multi-turn tool calling)
	Conversation []llms.Message

	// Accumulated assistant response text
	AssistantResponse strings.Builder

	// Tool calls made in first iteration (for history metadata)
	FirstIterationToolCalls []llms.ToolCall

	// Custom state for strategy-specific data
	// Strategies can store anything here (goals, confidence, etc.)
	CustomState map[string]interface{}

	// OutputChannel for strategies to send thinking blocks
	// (Optional - strategies can output directly for thinking mode)
	OutputChannel chan<- string

	// Configuration flags for conditional output
	ShowThinking  bool
	ShowDebugInfo bool

	// Services for strategies that need LLM calls (goal extraction, reflection)
	// Optional - only needed for advanced strategies
	Services AgentServices
	Context  context.Context
}

// NewReasoningState creates a new reasoning state
func NewReasoningState() *ReasoningState {
	return &ReasoningState{
		Iteration:               0,
		TotalTokens:             0,
		Conversation:            make([]llms.Message, 0),
		FirstIterationToolCalls: make([]llms.ToolCall, 0),
		CustomState:             make(map[string]interface{}),
	}
}
