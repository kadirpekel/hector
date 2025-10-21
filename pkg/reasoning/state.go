package reasoning

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// REASONING STATE
// Shared state passed between iterations with clear ownership boundaries
// ============================================================================

// ReasoningState holds the state of the reasoning process
//
// OWNERSHIP MODEL:
//   - Agent owns: iteration, totalTokens, history, currentTurn, assistantResponse
//     (strategies read via accessors, cannot modify)
//   - Strategy owns: CustomState, ToolState
//     (full read-write access)
//   - Shared config: Query, agentName, subAgents, OutputChannel, Services, Context
//     (read-only for both)
type ReasoningState struct {
	// ========== AGENT-OWNED FIELDS (private, read-only for strategies) ==========

	// Current iteration number (incremented atomically)
	iteration int

	// Total tokens used across all iterations
	totalTokens int

	// History messages (loaded from storage, immutable during this turn)
	history []*pb.Message

	// Current turn messages (new messages created during this execution)
	currentTurn []*pb.Message

	// Accumulated assistant response text
	assistantResponse strings.Builder

	// Tool calls made in first iteration (for history metadata)
	firstIterationToolCalls []*protocol.ToolCall

	// Flag to track if final response was added (eliminates complex inspection logic)
	finalResponseAdded bool

	// ========== SHARED IMMUTABLE CONTEXT (private, read-only for all) ==========

	// Original user query (for strategies to reference)
	query string

	// Agent context (typed, immutable)
	agentName string
	subAgents []string

	// Configuration flags for conditional output (immutable)
	showThinking  bool
	showDebugInfo bool

	// ========== STRATEGY-OWNED FIELDS (full read-write access) ==========

	// Custom state for strategy-specific data
	// Strategies can store anything here (goals, confidence, etc.)
	customState map[string]interface{}

	// Tool-specific state (for any tool to maintain state across iterations)
	// Examples: todo completion tracking, file watcher state, etc.
	toolState map[string]interface{}

	// ========== SHARED COMMUNICATION CHANNELS (read-only) ==========

	// OutputChannel for strategies to send thinking blocks
	outputChannel chan<- string

	// Services for strategies that need LLM calls (goal extraction, reflection)
	// Strategies have full access to all services for maximum flexibility
	services AgentServices
	context  context.Context
}

// ============================================================================
// CONSTRUCTOR & BUILDER PATTERN
// ============================================================================

// NewReasoningState creates a new reasoning state with defaults
// Use the builder pattern to configure it fully before use
func NewReasoningState() *ReasoningState {
	return &ReasoningState{
		iteration:               0,
		totalTokens:             0,
		history:                 make([]*pb.Message, 0),
		currentTurn:             make([]*pb.Message, 0),
		firstIterationToolCalls: make([]*protocol.ToolCall, 0),
		customState:             make(map[string]interface{}),
		toolState:               make(map[string]interface{}),
		finalResponseAdded:      false,
	}
}

// Builder returns a new StateBuilder for fluent configuration
func Builder() *StateBuilder {
	return &StateBuilder{
		state: NewReasoningState(),
	}
}

// StateBuilder provides fluent API for state initialization with validation
type StateBuilder struct {
	state *ReasoningState
	err   error
}

// WithQuery sets the original user query
func (b *StateBuilder) WithQuery(query string) *StateBuilder {
	if b.err != nil {
		return b
	}
	if query == "" {
		b.err = fmt.Errorf("query cannot be empty")
		return b
	}
	b.state.query = query
	return b
}

// WithAgentName sets the agent name
func (b *StateBuilder) WithAgentName(name string) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.agentName = name
	return b
}

// WithSubAgents sets the sub-agent IDs
func (b *StateBuilder) WithSubAgents(subAgents []string) *StateBuilder {
	if b.err != nil {
		return b
	}
	// Store a copy to prevent external mutations
	if len(subAgents) > 0 {
		b.state.subAgents = make([]string, len(subAgents))
		copy(b.state.subAgents, subAgents)
	}
	return b
}

// WithHistory sets the history messages
func (b *StateBuilder) WithHistory(history []*pb.Message) *StateBuilder {
	if b.err != nil {
		return b
	}
	// Store as-is (messages are pointers, but slice itself is owned)
	b.state.history = history
	return b
}

// WithOutputChannel sets the output channel
func (b *StateBuilder) WithOutputChannel(ch chan<- string) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.outputChannel = ch
	return b
}

// WithShowThinking sets the thinking display flag
func (b *StateBuilder) WithShowThinking(show bool) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.showThinking = show
	return b
}

// WithShowDebugInfo sets the debug info display flag
func (b *StateBuilder) WithShowDebugInfo(show bool) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.showDebugInfo = show
	return b
}

// WithServices sets the agent services
func (b *StateBuilder) WithServices(services AgentServices) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.services = services
	return b
}

// WithContext sets the context
func (b *StateBuilder) WithContext(ctx context.Context) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.context = ctx
	return b
}

// Build returns the configured state or an error if validation fails
func (b *StateBuilder) Build() (*ReasoningState, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Validation
	if b.state.query == "" {
		return nil, fmt.Errorf("query is required")
	}

	return b.state, nil
}

// ============================================================================
// AGENT-OWNED FIELDS - READ-ONLY ACCESSORS
// ============================================================================

// Iteration returns the current iteration number (read-only)
func (s *ReasoningState) Iteration() int {
	return s.iteration
}

// TotalTokens returns the total tokens used (read-only)
func (s *ReasoningState) TotalTokens() int {
	return s.totalTokens
}

// GetHistory returns a copy of history messages (defensive copy for immutability)
func (s *ReasoningState) GetHistory() []*pb.Message {
	if s.history == nil {
		return nil
	}
	// Return a copy to prevent external mutations
	historyCopy := make([]*pb.Message, len(s.history))
	copy(historyCopy, s.history)
	return historyCopy
}

// GetCurrentTurn returns a copy of current turn messages (defensive copy)
func (s *ReasoningState) GetCurrentTurn() []*pb.Message {
	if s.currentTurn == nil {
		return nil
	}
	// Return a copy to prevent external mutations
	turnCopy := make([]*pb.Message, len(s.currentTurn))
	copy(turnCopy, s.currentTurn)
	return turnCopy
}

// GetAssistantResponse returns the accumulated response text (read-only)
func (s *ReasoningState) GetAssistantResponse() string {
	return s.assistantResponse.String()
}

// GetFirstIterationToolCalls returns a copy of tool calls from first iteration (read-only)
func (s *ReasoningState) GetFirstIterationToolCalls() []*protocol.ToolCall {
	if s.firstIterationToolCalls == nil {
		return nil
	}
	// Return a copy to prevent external mutations
	callsCopy := make([]*protocol.ToolCall, len(s.firstIterationToolCalls))
	copy(callsCopy, s.firstIterationToolCalls)
	return callsCopy
}

// IsFinalResponseAdded returns whether the final response was added
func (s *ReasoningState) IsFinalResponseAdded() bool {
	return s.finalResponseAdded
}

// ============================================================================
// AGENT-OWNED FIELDS - MUTATION METHODS (agent only)
// ============================================================================

// NextIteration increments the iteration counter atomically
// Returns the new iteration number
func (s *ReasoningState) NextIteration() int {
	s.iteration++
	return s.iteration
}

// AddTokens adds tokens to the total count
func (s *ReasoningState) AddTokens(tokens int) {
	s.totalTokens += tokens
}

// AppendResponse appends text to the assistant response
func (s *ReasoningState) AppendResponse(text string) {
	s.assistantResponse.WriteString(text)
}

// RecordFirstToolCalls records tool calls from the first iteration
// Only records if iteration is 1 and no calls are recorded yet
func (s *ReasoningState) RecordFirstToolCalls(calls []*protocol.ToolCall) {
	if s.iteration == 1 && len(s.firstIterationToolCalls) == 0 && len(calls) > 0 {
		s.firstIterationToolCalls = calls
	}
}

// AddCurrentTurnMessage adds a message to the current turn
func (s *ReasoningState) AddCurrentTurnMessage(msg *pb.Message) {
	s.currentTurn = append(s.currentTurn, msg)
}

// MarkFinalResponseAdded marks that the final response message was added
func (s *ReasoningState) MarkFinalResponseAdded() {
	s.finalResponseAdded = true
}

// SetHistory sets the history messages (agent initialization only)
func (s *ReasoningState) SetHistory(history []*pb.Message) {
	s.history = history
}

// ============================================================================
// SHARED CONTEXT - READ-ONLY ACCESSORS
// ============================================================================

// Query returns the original user query (immutable)
func (s *ReasoningState) Query() string {
	return s.query
}

// AgentName returns the current agent's name (immutable)
func (s *ReasoningState) AgentName() string {
	return s.agentName
}

// SubAgents returns a copy of sub-agent IDs (defensive copy for immutability)
func (s *ReasoningState) SubAgents() []string {
	if s.subAgents == nil {
		return nil
	}
	// Return a copy to prevent external mutations
	subAgentsCopy := make([]string, len(s.subAgents))
	copy(subAgentsCopy, s.subAgents)
	return subAgentsCopy
}

// ShowThinking returns the thinking display flag (immutable)
func (s *ReasoningState) ShowThinking() bool {
	return s.showThinking
}

// ShowDebugInfo returns the debug info display flag (immutable)
func (s *ReasoningState) ShowDebugInfo() bool {
	return s.showDebugInfo
}

// ============================================================================
// CONVERSATION HELPERS
// ============================================================================

// AllMessages returns all messages (history + current turn) as a single slice
// This is useful for building LLM prompts that need the full conversation
func (s *ReasoningState) AllMessages() []*pb.Message {
	all := make([]*pb.Message, 0, len(s.history)+len(s.currentTurn))
	all = append(all, s.history...)
	all = append(all, s.currentTurn...)
	return all
}

// ============================================================================
// ACCESSORS FOR SHARED RESOURCES & STRATEGY STATE
// ============================================================================

// GetOutputChannel returns the output channel for sending messages
func (s *ReasoningState) GetOutputChannel() chan<- string {
	return s.outputChannel
}

// GetServices returns the agent services for LLM calls
func (s *ReasoningState) GetServices() AgentServices {
	return s.services
}

// GetContext returns the context for cancellation and timeouts
func (s *ReasoningState) GetContext() context.Context {
	return s.context
}

// GetCustomState returns the custom state map for strategy-specific data
// The returned map can be modified but should not be replaced
func (s *ReasoningState) GetCustomState() map[string]interface{} {
	return s.customState
}

// GetToolState returns the tool state map for tool-specific data
// The returned map can be modified but should not be replaced
func (s *ReasoningState) GetToolState() map[string]interface{} {
	return s.toolState
}

// ============================================================================
// BACKWARDS COMPATIBILITY & LEGACY SETTERS
// ============================================================================

// SetAgentName sets the agent name (legacy - use builder instead)
func (s *ReasoningState) SetAgentName(name string) {
	s.agentName = name
}

// SetSubAgents sets sub-agent IDs (legacy - use builder instead)
func (s *ReasoningState) SetSubAgents(subAgents []string) {
	if len(subAgents) > 0 {
		s.subAgents = make([]string, len(subAgents))
		copy(s.subAgents, subAgents)
	}
}

// Conversation returns all messages for backwards compatibility
// Deprecated: Use AllMessages() instead
func (s *ReasoningState) Conversation() []*pb.Message {
	return s.AllMessages()
}

// SetConversation is a compatibility helper
// Deprecated: Use SetHistory and AddCurrentTurnMessage instead
func (s *ReasoningState) SetConversation(messages []*pb.Message, historyCount int) {
	if historyCount > len(messages) {
		historyCount = len(messages)
	}
	s.history = messages[:historyCount]
	s.currentTurn = messages[historyCount:]
}

// ============================================================================
// LEGACY FIELD ACCESS (for gradual migration)
// ============================================================================

// These are kept for backwards compatibility but should be migrated to use
// the new accessor methods. They will be removed in a future version.

// History field accessor (legacy - use GetHistory() instead)
// Deprecated: Direct field access will be removed
var _ = (*ReasoningState).legacyHistoryAccess

func (s *ReasoningState) legacyHistoryAccess() []*pb.Message {
	return s.history
}

// CurrentTurn field accessor (legacy - use GetCurrentTurn() instead)
// Deprecated: Direct field access will be removed
var _ = (*ReasoningState).legacyCurrentTurnAccess

func (s *ReasoningState) legacyCurrentTurnAccess() []*pb.Message {
	return s.currentTurn
}
