package reasoning

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

type ReasoningState struct {
	iteration int

	totalTokens int

	history []*pb.Message

	currentTurn []*pb.Message

	assistantResponse strings.Builder

	firstIterationToolCalls []*protocol.ToolCall

	finalResponseAdded bool

	query string

	agentName string
	subAgents []string

	showThinking  bool
	showDebugInfo bool

	customState map[string]interface{}

	toolState map[string]interface{}

	outputChannel chan<- *pb.Part

	services AgentServices
	context  context.Context
}

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

func Builder() *StateBuilder {
	return &StateBuilder{
		state: NewReasoningState(),
	}
}

type StateBuilder struct {
	state *ReasoningState
	err   error
}

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

func (b *StateBuilder) WithAgentName(name string) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.agentName = name
	return b
}

func (b *StateBuilder) WithSubAgents(subAgents []string) *StateBuilder {
	if b.err != nil {
		return b
	}

	if len(subAgents) > 0 {
		b.state.subAgents = make([]string, len(subAgents))
		copy(b.state.subAgents, subAgents)
	}
	return b
}

func (b *StateBuilder) WithHistory(history []*pb.Message) *StateBuilder {
	if b.err != nil {
		return b
	}

	b.state.history = history
	return b
}

func (b *StateBuilder) WithOutputChannel(ch chan<- *pb.Part) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.outputChannel = ch
	return b
}

func (b *StateBuilder) WithShowThinking(show bool) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.showThinking = show
	return b
}

func (b *StateBuilder) WithShowDebugInfo(show bool) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.showDebugInfo = show
	return b
}

func (b *StateBuilder) WithServices(services AgentServices) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.services = services
	return b
}

func (b *StateBuilder) WithContext(ctx context.Context) *StateBuilder {
	if b.err != nil {
		return b
	}
	b.state.context = ctx
	return b
}

func (b *StateBuilder) Build() (*ReasoningState, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.state.query == "" {
		return nil, fmt.Errorf("query is required")
	}

	return b.state, nil
}

func (s *ReasoningState) Iteration() int {
	return s.iteration
}

func (s *ReasoningState) TotalTokens() int {
	return s.totalTokens
}

func (s *ReasoningState) GetHistory() []*pb.Message {
	if s.history == nil {
		return nil
	}

	historyCopy := make([]*pb.Message, len(s.history))
	copy(historyCopy, s.history)
	return historyCopy
}

func (s *ReasoningState) GetCurrentTurn() []*pb.Message {
	if s.currentTurn == nil {
		return nil
	}

	turnCopy := make([]*pb.Message, len(s.currentTurn))
	copy(turnCopy, s.currentTurn)
	return turnCopy
}

func (s *ReasoningState) GetAssistantResponse() string {
	return s.assistantResponse.String()
}

func (s *ReasoningState) GetFirstIterationToolCalls() []*protocol.ToolCall {
	if s.firstIterationToolCalls == nil {
		return nil
	}

	callsCopy := make([]*protocol.ToolCall, len(s.firstIterationToolCalls))
	copy(callsCopy, s.firstIterationToolCalls)
	return callsCopy
}

func (s *ReasoningState) IsFinalResponseAdded() bool {
	return s.finalResponseAdded
}

func (s *ReasoningState) NextIteration() int {
	s.iteration++
	return s.iteration
}

func (s *ReasoningState) AddTokens(tokens int) {
	s.totalTokens += tokens
}

func (s *ReasoningState) AppendResponse(text string) {
	s.assistantResponse.WriteString(text)
}

func (s *ReasoningState) RecordFirstToolCalls(calls []*protocol.ToolCall) {
	if s.iteration == 1 && len(s.firstIterationToolCalls) == 0 && len(calls) > 0 {
		s.firstIterationToolCalls = calls
	}
}

func (s *ReasoningState) AddCurrentTurnMessage(msg *pb.Message) {
	s.currentTurn = append(s.currentTurn, msg)
}

func (s *ReasoningState) MarkFinalResponseAdded() {
	s.finalResponseAdded = true
}

func (s *ReasoningState) SetHistory(history []*pb.Message) {
	s.history = history
}

func (s *ReasoningState) Query() string {
	return s.query
}

func (s *ReasoningState) AgentName() string {
	return s.agentName
}

func (s *ReasoningState) SubAgents() []string {
	if s.subAgents == nil {
		return nil
	}

	subAgentsCopy := make([]string, len(s.subAgents))
	copy(subAgentsCopy, s.subAgents)
	return subAgentsCopy
}

func (s *ReasoningState) ShowThinking() bool {
	return s.showThinking
}

func (s *ReasoningState) ShowDebugInfo() bool {
	return s.showDebugInfo
}

func (s *ReasoningState) AllMessages() []*pb.Message {
	all := make([]*pb.Message, 0, len(s.history)+len(s.currentTurn))
	all = append(all, s.history...)
	all = append(all, s.currentTurn...)
	return all
}

func (s *ReasoningState) GetOutputChannel() chan<- *pb.Part {
	return s.outputChannel
}

func (s *ReasoningState) GetServices() AgentServices {
	return s.services
}

func (s *ReasoningState) GetContext() context.Context {
	return s.context
}

func (s *ReasoningState) GetCustomState() map[string]interface{} {
	return s.customState
}

func (s *ReasoningState) GetToolState() map[string]interface{} {
	return s.toolState
}

func (s *ReasoningState) SetAgentName(name string) {
	s.agentName = name
}

func (s *ReasoningState) SetSubAgents(subAgents []string) {
	if len(subAgents) > 0 {
		s.subAgents = make([]string, len(subAgents))
		copy(s.subAgents, subAgents)
	}
}

func (s *ReasoningState) Conversation() []*pb.Message {
	return s.AllMessages()
}

func (s *ReasoningState) SetConversation(messages []*pb.Message, historyCount int) {
	if historyCount > len(messages) {
		historyCount = len(messages)
	}
	s.history = messages[:historyCount]
	s.currentTurn = messages[historyCount:]
}

var _ = (*ReasoningState).legacyHistoryAccess

func (s *ReasoningState) legacyHistoryAccess() []*pb.Message {
	return s.history
}

var _ = (*ReasoningState).legacyCurrentTurnAccess

func (s *ReasoningState) legacyCurrentTurnAccess() []*pb.Message {
	return s.currentTurn
}
