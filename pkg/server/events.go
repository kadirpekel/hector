package server

import (
	"context"
	"log/slog"
	"maps"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/auth"
)

// Metadata keys for A2A events
const (
	metaKeyTaskID    = "hector:task_id"
	metaKeyContextID = "hector:context_id"
	metaKeyEscalate  = "hector:escalate"
	metaKeyTransfer  = "hector:transfer_to_agent"
)

// invocationMeta contains metadata for an invocation.
type invocationMeta struct {
	userID    string
	sessionID string
	eventMeta map[string]any
}

// toInvocationMeta extracts execution metadata from the request context
func toInvocationMeta(ctx context.Context, reqCtx *a2asrv.RequestContext) invocationMeta {
	meta := invocationMeta{
		eventMeta: make(map[string]any),
	}

	// Use the context ID provided by a2asrv - this matches the Task's ContextID
	// and ensures session persistence works correctly across server restarts.
	// a2asrv either uses the client-provided context_id or generates one,
	// and stores it in the task for continuations.
	meta.sessionID = reqCtx.ContextID
	slog.Debug("Using a2asrv context as session", "session_id", meta.sessionID, "task_id", string(reqCtx.TaskID))

	// Include taskId in event metadata for frontend cancellation support
	if reqCtx.TaskID != "" {
		meta.eventMeta["taskId"] = string(reqCtx.TaskID)
	}

	// Check for auth from context (Highest Priority)
	// This ensures that if the request was authenticated (e.g. by API key),
	// the session is bound to that user ID.
	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		meta.userID = claims.Subject
	}

	// Extract user ID from message metadata (Override if explicitly provided in A2A message)
	if reqCtx.Message != nil && reqCtx.Message.Metadata != nil {
		if uid, ok := reqCtx.Message.Metadata["user_id"].(string); ok {
			meta.userID = uid
		}
	}

	// Default user ID
	if meta.userID == "" {
		meta.userID = "default"
	}

	return meta
}

// eventProcessor translates Hector events to A2A events.
type eventProcessor struct {
	reqCtx *a2asrv.RequestContext
	meta   invocationMeta

	// terminalActions accumulates actions for the terminal event
	terminalActions agent.EventActions

	// responseID is created once first artifact is sent
	responseID a2a.ArtifactID

	// terminalEvents holds potential terminal events by state
	terminalEvents map[a2a.TaskState]*a2a.TaskStatusUpdateEvent
}

func newEventProcessor(reqCtx *a2asrv.RequestContext, meta invocationMeta) *eventProcessor {
	return &eventProcessor{
		reqCtx:         reqCtx,
		meta:           meta,
		terminalEvents: make(map[a2a.TaskState]*a2a.TaskStatusUpdateEvent),
	}
}

func (p *eventProcessor) process(ctx context.Context, event *agent.Event) (*a2a.TaskArtifactUpdateEvent, error) {
	if event == nil {
		return nil, nil
	}

	p.updateTerminalActions(event)

	eventMeta := p.makeEventMeta(event)

	// Check for HITL input required - two signals:
	// 1. Event.LongRunningToolIDs (ADK-Go compatible)
	// 2. Event.Actions.RequireInput (Hector extension)
	if len(event.LongRunningToolIDs) > 0 || event.Actions.RequireInput {
		slog.Info("HITL input required",
			"longRunningToolIDs", len(event.LongRunningToolIDs),
			"requireInput", event.Actions.RequireInput)
		// Build status message with prompt if available
		var statusMsg *a2a.Message
		if event.Actions.InputPrompt != "" {
			statusMsg = a2a.NewMessageForTask(
				a2a.MessageRoleAgent,
				p.reqCtx,
				a2a.TextPart{Text: event.Actions.InputPrompt},
			)
		}

		ev := a2a.NewStatusUpdateEvent(p.reqCtx, a2a.TaskStateInputRequired, statusMsg)
		ev.Final = true

		// Add HITL metadata for UI
		if ev.Metadata == nil {
			ev.Metadata = make(map[string]any)
		}
		ev.Metadata["input_required"] = true
		if len(event.LongRunningToolIDs) > 0 {
			// Convert []string to []any for A2A Metadata compatibility
			toolIDs := make([]any, len(event.LongRunningToolIDs))
			for i, id := range event.LongRunningToolIDs {
				toolIDs[i] = id
			}
			ev.Metadata["long_running_tool_ids"] = toolIDs
		}
		if event.Actions.InputPrompt != "" {
			ev.Metadata["input_prompt"] = event.Actions.InputPrompt
		}

		p.terminalEvents[a2a.TaskStateInputRequired] = ev
	}

	// Determine if we have content to emit
	// Events can have: artifact parts, thinking, tool calls, or tool results
	hasParts := event.Artifact != nil && len(event.Artifact.Parts) > 0
	thinking := event.Thinking()
	hasThinking := thinking != nil && thinking.Content != ""
	hasToolCalls := len(event.ToolCalls()) > 0
	hasToolResults := len(event.ToolResults()) > 0

	// If no content at all, skip, unless we have custom metadata (e.g. progress events)
	if !hasParts && !hasThinking && !hasToolCalls && !hasToolResults && len(event.Metadata) == 0 {
		return nil, nil
	}

	// Get parts (may be empty for thinking-only events)
	parts := event.Parts()

	// Create or update artifact
	var result *a2a.TaskArtifactUpdateEvent
	if p.responseID == "" {
		result = a2a.NewArtifactEvent(p.reqCtx, parts...)
		p.responseID = result.Artifact.ID
	} else {
		result = a2a.NewArtifactUpdateEvent(p.reqCtx, p.responseID, parts...)
	}

	// Always include metadata for contextual blocks (thinking, tools)
	if len(eventMeta) > 0 {
		result.Metadata = eventMeta
	}

	return result, nil
}

func (p *eventProcessor) makeTerminalEvents() []a2a.Event {
	result := make([]a2a.Event, 0, 2)

	// Close artifact stream if we sent any artifacts
	if p.responseID != "" {
		ev := a2a.NewArtifactUpdateEvent(p.reqCtx, p.responseID)
		ev.LastChunk = true
		result = append(result, ev)
	}

	// Check for failure or input required (in priority order)
	for _, state := range []a2a.TaskState{a2a.TaskStateFailed, a2a.TaskStateInputRequired} {
		if ev, ok := p.terminalEvents[state]; ok {
			ev.Metadata = p.setActionsMeta(ev.Metadata)
			result = append(result, ev)
			return result
		}
	}

	// Default: completed
	ev := a2a.NewStatusUpdateEvent(p.reqCtx, a2a.TaskStateCompleted, nil)
	ev.Final = true
	ev.Metadata = p.setActionsMeta(maps.Clone(p.meta.eventMeta))
	result = append(result, ev)

	return result
}

func (p *eventProcessor) makeFailedEvent(cause error, event *agent.Event) *a2a.TaskStatusUpdateEvent {
	meta := p.meta.eventMeta
	if event != nil {
		meta = p.makeEventMeta(event)
	}
	return toFailedStatusEvent(p.reqCtx, cause, meta)
}

func (p *eventProcessor) updateTerminalActions(event *agent.Event) {
	p.terminalActions.Escalate = p.terminalActions.Escalate || event.Actions.Escalate
	if event.Actions.TransferToAgent != "" {
		p.terminalActions.TransferToAgent = event.Actions.TransferToAgent
	}
}

func (p *eventProcessor) makeEventMeta(event *agent.Event) map[string]any {
	meta := maps.Clone(p.meta.eventMeta)
	if meta == nil {
		meta = make(map[string]any)
	}

	meta["event_id"] = event.ID
	meta["author"] = event.Author
	if event.Branch != "" {
		meta["branch"] = event.Branch
	}
	// ADK-Go aligned: Pass partial flag to UI for duplicate detection
	// UI should track streamed content and skip final if it matches
	meta["partial"] = event.Partial

	// Add invocation ID for stable widget identification
	if event.InvocationID != "" {
		meta["invocation_id"] = event.InvocationID
	}

	// Add AgentID for proper highlighting (matches Canvas Node ID)
	if event.AgentID != "" {
		meta["agent_id"] = event.AgentID
	}

	// Propagate Metadata (generic extension point)
	if len(event.Metadata) > 0 {
		for k, v := range event.Metadata {
			meta[k] = v
		}
	}

	// Contextual Blocks - These enable rich UI rendering with proper lifecycle
	// Each block type maps to a specific widget in the UI

	// Thinking block - renders as ThinkingWidget with auto-expand/collapse
	if thinking := event.Thinking(); thinking != nil {
		meta["thinking"] = map[string]any{
			"id":      thinking.ID,
			"status":  thinking.Status,
			"content": thinking.Content,
			"type":    thinking.Type,
		}
		// Include signature for multi-turn verification (Anthropic requirement)
		if thinking.Signature != "" {
			meta["thinking"].(map[string]any)["signature"] = thinking.Signature
		}
	}

	// Tool calls - render as ToolWidget with "working" status
	if toolCalls := event.ToolCalls(); len(toolCalls) > 0 {
		toolCallsMeta := make([]map[string]any, len(toolCalls))
		for i, tc := range toolCalls {
			toolCallsMeta[i] = map[string]any{
				"id":     tc.ID,
				"name":   tc.Name,
				"args":   tc.Args,
				"status": tc.Status,
			}
		}
		meta["tool_calls"] = toolCallsMeta
	}

	// Tool results - update ToolWidget to "success"/"failed"
	if toolResults := event.ToolResults(); len(toolResults) > 0 {
		toolResultsMeta := make([]map[string]any, len(toolResults))
		for i, tr := range toolResults {
			toolResultsMeta[i] = map[string]any{
				"tool_call_id": tr.ToolCallID,
				"content":      tr.Content,
				"status":       tr.Status,
				"is_error":     tr.IsError,
				"metadata":     tr.Metadata,
			}
		}
		meta["tool_results"] = toolResultsMeta
	}

	return meta
}

func (p *eventProcessor) setActionsMeta(meta map[string]any) map[string]any {
	if meta == nil {
		meta = make(map[string]any)
	}

	if p.terminalActions.Escalate {
		meta[metaKeyEscalate] = true
	}
	if p.terminalActions.TransferToAgent != "" {
		meta[metaKeyTransfer] = p.terminalActions.TransferToAgent
	}

	return meta
}

func toFailedStatusEvent(reqCtx *a2asrv.RequestContext, cause error, meta map[string]any) *a2a.TaskStatusUpdateEvent {
	msg := a2a.NewMessageForTask(a2a.MessageRoleAgent, reqCtx, a2a.TextPart{Text: cause.Error()})
	ev := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateFailed, msg)
	ev.Metadata = meta
	ev.Final = true
	return ev
}
