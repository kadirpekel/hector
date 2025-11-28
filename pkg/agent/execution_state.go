package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"google.golang.org/protobuf/encoding/protojson"
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

// reasoningStateSnapshotJSON is the JSON representation with protojson-encoded messages
type reasoningStateSnapshotJSON struct {
	Iteration               int                  `json:"iteration"`
	TotalTokens             int                  `json:"total_tokens"`
	History                 []json.RawMessage    `json:"history"`
	CurrentTurn             []json.RawMessage    `json:"current_turn"`
	AssistantResponse       string               `json:"assistant_response"`
	FirstIterationToolCalls []*protocol.ToolCall `json:"first_iteration_tool_calls"`
	FinalResponseAdded      bool                 `json:"final_response_added"`
	Query                   string               `json:"query"`
	AgentName               string               `json:"agent_name"`
	SubAgents               []string             `json:"sub_agents"`
	ShowThinking            bool                 `json:"show_thinking"`
}

// SerializeExecutionState converts execution state to JSON
// Uses protojson for protobuf messages to properly handle oneof fields
func SerializeExecutionState(state *ExecutionState) ([]byte, error) {
	if state == nil {
		return nil, fmt.Errorf("execution state is nil")
	}

	// Create JSON-compatible version with protojson-encoded messages
	jsonState := struct {
		TaskID          string                      `json:"task_id"`
		ContextID       string                      `json:"context_id"`
		Query           string                      `json:"query"`
		ReasoningState  *reasoningStateSnapshotJSON `json:"reasoning_state"`
		PendingToolCall *protocol.ToolCall          `json:"pending_tool_call"`
		Phase           ExecutionPhase              `json:"phase,omitempty"`
		CheckpointType  CheckpointType              `json:"checkpoint_type,omitempty"`
		CheckpointTime  time.Time                   `json:"checkpoint_time,omitempty"`
	}{
		TaskID:          state.TaskID,
		ContextID:       state.ContextID,
		Query:           state.Query,
		PendingToolCall: state.PendingToolCall,
		Phase:           state.Phase,
		CheckpointType:  state.CheckpointType,
		CheckpointTime:  state.CheckpointTime,
	}

	// Convert protobuf messages to protojson-encoded JSON
	if state.ReasoningState != nil {
		jsonState.ReasoningState = &reasoningStateSnapshotJSON{
			Iteration:               state.ReasoningState.Iteration,
			TotalTokens:             state.ReasoningState.TotalTokens,
			AssistantResponse:       state.ReasoningState.AssistantResponse,
			FirstIterationToolCalls: state.ReasoningState.FirstIterationToolCalls,
			FinalResponseAdded:      state.ReasoningState.FinalResponseAdded,
			Query:                   state.ReasoningState.Query,
			AgentName:               state.ReasoningState.AgentName,
			SubAgents:               state.ReasoningState.SubAgents,
			ShowThinking:            state.ReasoningState.ShowThinking,
		}

		// Encode History messages using protojson
		jsonState.ReasoningState.History = make([]json.RawMessage, len(state.ReasoningState.History))
		for i, msg := range state.ReasoningState.History {
			msgJSON, err := protojson.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal history message %d: %w", i, err)
			}
			jsonState.ReasoningState.History[i] = json.RawMessage(msgJSON)
		}

		// Encode CurrentTurn messages using protojson
		jsonState.ReasoningState.CurrentTurn = make([]json.RawMessage, len(state.ReasoningState.CurrentTurn))
		for i, msg := range state.ReasoningState.CurrentTurn {
			msgJSON, err := protojson.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal current_turn message %d: %w", i, err)
			}
			jsonState.ReasoningState.CurrentTurn[i] = json.RawMessage(msgJSON)
		}
	}

	return json.Marshal(jsonState)
}

// DeserializeExecutionState reconstructs execution state from JSON
// Uses protojson for protobuf messages to properly handle oneof fields
func DeserializeExecutionState(data []byte) (*ExecutionState, error) {
	var jsonState struct {
		TaskID          string                      `json:"task_id"`
		ContextID       string                      `json:"context_id"`
		Query           string                      `json:"query"`
		ReasoningState  *reasoningStateSnapshotJSON `json:"reasoning_state"`
		PendingToolCall *protocol.ToolCall          `json:"pending_tool_call"`
		Phase           ExecutionPhase              `json:"phase,omitempty"`
		CheckpointType  CheckpointType              `json:"checkpoint_type,omitempty"`
		CheckpointTime  time.Time                   `json:"checkpoint_time,omitempty"`
	}

	if err := json.Unmarshal(data, &jsonState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution state: %w", err)
	}

	state := &ExecutionState{
		TaskID:          jsonState.TaskID,
		ContextID:       jsonState.ContextID,
		Query:           jsonState.Query,
		PendingToolCall: jsonState.PendingToolCall,
		Phase:           jsonState.Phase,
		CheckpointType:  jsonState.CheckpointType,
		CheckpointTime:  jsonState.CheckpointTime,
	}

	// Decode protobuf messages from protojson-encoded JSON
	if jsonState.ReasoningState != nil {
		state.ReasoningState = &ReasoningStateSnapshot{
			Iteration:               jsonState.ReasoningState.Iteration,
			TotalTokens:             jsonState.ReasoningState.TotalTokens,
			AssistantResponse:       jsonState.ReasoningState.AssistantResponse,
			FirstIterationToolCalls: jsonState.ReasoningState.FirstIterationToolCalls,
			FinalResponseAdded:      jsonState.ReasoningState.FinalResponseAdded,
			Query:                   jsonState.ReasoningState.Query,
			AgentName:               jsonState.ReasoningState.AgentName,
			SubAgents:               jsonState.ReasoningState.SubAgents,
			ShowThinking:            jsonState.ReasoningState.ShowThinking,
		}

		// Decode History messages using protojson
		state.ReasoningState.History = make([]*pb.Message, len(jsonState.ReasoningState.History))
		for i, rawMsg := range jsonState.ReasoningState.History {
			msg := &pb.Message{}
			if err := protojson.Unmarshal(rawMsg, msg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal history message %d: %w", i, err)
			}
			state.ReasoningState.History[i] = msg
		}

		// Decode CurrentTurn messages using protojson
		state.ReasoningState.CurrentTurn = make([]*pb.Message, len(jsonState.ReasoningState.CurrentTurn))
		for i, rawMsg := range jsonState.ReasoningState.CurrentTurn {
			msg := &pb.Message{}
			if err := protojson.Unmarshal(rawMsg, msg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal current_turn message %d: %w", i, err)
			}
			state.ReasoningState.CurrentTurn[i] = msg
		}
	}

	return state, nil
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
