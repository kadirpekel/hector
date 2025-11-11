package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ExecutionPhase represents the phase of execution when checkpoint was created
type ExecutionPhase string

const (
	PhaseInitialized   ExecutionPhase = "initialized"
	PhasePreLLM        ExecutionPhase = "pre_llm"
	PhasePostLLM       ExecutionPhase = "post_llm"
	PhaseToolExecution ExecutionPhase = "tool_execution"
	PhasePostTool      ExecutionPhase = "post_tool"
	PhaseIterationEnd  ExecutionPhase = "iteration_end"
	PhaseToolApproval  ExecutionPhase = "tool_approval" // Used by async HITL
	PhaseError         ExecutionPhase = "error"
)

// CheckpointType represents why the checkpoint was created
type CheckpointType string

const (
	CheckpointTypeEvent    CheckpointType = "event"    // Event-driven (tool approval, error)
	CheckpointTypeInterval CheckpointType = "interval" // Interval-based (periodic)
	CheckpointTypeManual   CheckpointType = "manual"   // Manual pause
	CheckpointTypeError    CheckpointType = "error"    // Error recovery
)

// ExecutionState represents the state needed to resume task execution
type ExecutionState struct {
	TaskID          string                  `json:"task_id"`
	ContextID       string                  `json:"context_id"`
	Query           string                  `json:"query"`
	ReasoningState  *ReasoningStateSnapshot `json:"reasoning_state"`
	PendingToolCall *protocol.ToolCall      `json:"pending_tool_call"`

	// Checkpoint metadata (optional, backward compatible)
	Phase          ExecutionPhase `json:"phase,omitempty"`
	CheckpointType CheckpointType `json:"checkpoint_type,omitempty"`
	CheckpointTime time.Time      `json:"checkpoint_time,omitempty"`
}

// ReasoningStateSnapshot is a serializable version of reasoning state
type ReasoningStateSnapshot struct {
	Iteration               int                  `json:"iteration"`
	TotalTokens             int                  `json:"total_tokens"`
	History                 []*pb.Message        `json:"history"`
	CurrentTurn             []*pb.Message        `json:"current_turn"`
	AssistantResponse       string               `json:"assistant_response"`
	FirstIterationToolCalls []*protocol.ToolCall `json:"first_iteration_tool_calls"`
	FinalResponseAdded      bool                 `json:"final_response_added"`
	Query                   string               `json:"query"`
	AgentName               string               `json:"agent_name"`
	SubAgents               []string             `json:"sub_agents"`
	ShowThinking            bool                 `json:"show_thinking"`
}

// SerializeExecutionState converts execution state to JSON
func SerializeExecutionState(state *ExecutionState) ([]byte, error) {
	return json.Marshal(state)
}

// DeserializeExecutionState reconstructs execution state from JSON
func DeserializeExecutionState(data []byte) (*ExecutionState, error) {
	var state ExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// CaptureExecutionState creates a snapshot of current execution state
func CaptureExecutionState(
	taskID string,
	contextID string,
	query string,
	reasoningState *reasoning.ReasoningState,
	pendingToolCall *protocol.ToolCall,
) *ExecutionState {
	return &ExecutionState{
		TaskID:    taskID,
		ContextID: contextID,
		Query:     query,
		ReasoningState: &ReasoningStateSnapshot{
			Iteration:               reasoningState.Iteration(),
			TotalTokens:             reasoningState.TotalTokens(),
			History:                 reasoningState.GetHistory(),
			CurrentTurn:             reasoningState.GetCurrentTurn(),
			AssistantResponse:       reasoningState.GetAssistantResponse(),
			FirstIterationToolCalls: reasoningState.GetFirstIterationToolCalls(),
			FinalResponseAdded:      reasoningState.IsFinalResponseAdded(),
			Query:                   reasoningState.Query(),
			AgentName:               reasoningState.AgentName(),
			SubAgents:               reasoningState.SubAgents(),
			ShowThinking:            reasoningState.ShowThinking(),
		},
		PendingToolCall: pendingToolCall,
	}
}

// RestoreReasoningState reconstructs a ReasoningState from snapshot
func (s *ExecutionState) RestoreReasoningState(
	outputCh chan<- *pb.Part,
	services reasoning.AgentServices,
	ctx context.Context,
) (*reasoning.ReasoningState, error) {
	if s.ReasoningState == nil {
		return nil, fmt.Errorf("reasoning state snapshot is nil")
	}

	state, err := reasoning.Builder().
		WithQuery(s.Query).
		WithAgentName(s.ReasoningState.AgentName).
		WithSubAgents(s.ReasoningState.SubAgents).
		WithOutputChannel(outputCh).
		WithShowThinking(s.ReasoningState.ShowThinking).
		WithServices(services).
		WithContext(ctx).
		WithHistory(s.ReasoningState.History).
		Build()

	if err != nil {
		return nil, err
	}

	// Restore iteration count
	for i := 0; i < s.ReasoningState.Iteration; i++ {
		state.NextIteration()
	}

	// Restore current turn messages
	for _, msg := range s.ReasoningState.CurrentTurn {
		state.AddCurrentTurnMessage(msg)
	}

	// Restore assistant response
	state.AppendResponse(s.ReasoningState.AssistantResponse)

	// Restore final response flag
	if s.ReasoningState.FinalResponseAdded {
		state.MarkFinalResponseAdded()
	}

	return state, nil
}
