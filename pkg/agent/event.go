package agent

import (
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
)

// Event author constants
const (
	// AuthorUser represents events authored by the user (human input).
	// Used when tool results or other events come from user interactions.
	AuthorUser = "user"

	// AuthorSystem represents events authored by the system.
	// Used for system-generated events, errors, notifications, or conversation summaries.
	AuthorSystem = "system"
)

// Event metadata type constants
const (
	// MetadataTypeProgress indicates a progress/lifecycle event (transient).
	MetadataTypeProgress = "progress"

	// MetadataTypeSummary indicates a conversation summary event (persisted).
	MetadataTypeSummary = "summary"

	// MetadataTypeGuardrail indicates an input/output/tool guardrail intervention.
	MetadataTypeGuardrail = "guardrail"

	// MetadataTypeModeration indicates a moderation provider intervention.
	MetadataTypeModeration = "moderation"
)

// Intervention action constants for guardrails and moderation.
const (
	// InterventionActionBlock means the request/response was blocked.
	InterventionActionBlock = "block"

	// InterventionActionModify means the content was modified.
	InterventionActionModify = "modify"

	// InterventionActionWarn means a warning was logged but content passed.
	InterventionActionWarn = "warn"
)

// Intervention source constants identify where the intervention originated.
const (
	InterventionSourceInputGuardrail  = "input_guardrail"
	InterventionSourceOutputGuardrail = "output_guardrail"
	InterventionSourceToolGuardrail   = "tool_guardrail"
	InterventionSourceModeration      = "moderation"
)

// Progress event constants for CustomMetadata
const (
	// ProgressStatusSummarizing indicates summarization has started.
	ProgressStatusSummarizing = "summarizing"

	// ProgressStatusSummarized indicates summarization completed successfully.
	ProgressStatusSummarized = "summarized"

	// ProgressStatusError indicates a lifecycle operation failed.
	ProgressStatusError = "error"

	// Generic Progress Statuses (for any core loop activity)
	ProgressStatusActive    = "active"
	ProgressStatusCompleted = "completed"
)

// Metadata keys for A2A-compatible event serialization.
// These keys are used in the Metadata map for consistent wire format.
const (
	MetaKeyThinking           = "thinking"
	MetaKeyToolCalls          = "tool_calls"
	MetaKeyToolResults        = "tool_results"
	MetaKeyActions            = "actions"
	MetaKeyLongRunningToolIDs = "long_running_tool_ids"
	MetaKeyErrorCode          = "error_code"
	MetaKeyErrorMessage       = "error_message"
	MetaKeyEventID            = "event_id"
	MetaKeyAuthor             = "author"
	MetaKeyAgentID            = "agent_id"
	MetaKeyBranch             = "branch"
	MetaKeyInvocationID       = "invocation_id"
	MetaKeyPartial            = "partial"
	MetaKeyTurnComplete       = "turn_complete"
	MetaKeyInterrupted        = "interrupted"
)

// Event represents an interaction in an agent conversation.
// Events are A2A-native: they use Artifact for content and Metadata for contextual blocks.
// This ensures consistent format across storage and wire transmission.
type Event struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`

	// Timestamp when the event was created.
	Timestamp time.Time `json:"timestamp"`

	// InvocationID links this event to its invocation.
	InvocationID string `json:"invocation_id"`

	// Branch isolates conversation history for parallel agents.
	// Format: "agent_1.agent_2.agent_3" (parent chain).
	Branch string `json:"branch,omitempty"`

	// Author is the display name of the agent (or "user"/"system").
	// This is used for UI rendering and attribution.
	Author string `json:"author"`

	// AgentID is the unique identifier of the agent that produced this event.
	// Used for internal routing, logic, and state tracking.
	AgentID string `json:"agent_id,omitempty"`

	// Artifact contains the A2A artifact with message parts.
	// This is the primary content carrier for A2A-native events.
	Artifact *a2a.Artifact `json:"artifact,omitempty"`

	// Metadata contains all contextual data in A2A-compatible format.
	// Keys: thinking, tool_calls, tool_results, actions, etc.
	// This is the canonical source for contextual blocks.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Actions captures side effects (state changes, transfers, etc).
	// Also stored in Metadata["actions"] for A2A wire format.
	Actions EventActions `json:"-"`

	// LongRunningToolIDs identifies tools awaiting external completion.
	LongRunningToolIDs []string `json:"-"`

	// Partial indicates this is a streaming chunk, not a complete event.
	Partial bool `json:"partial,omitempty"`

	// TurnComplete indicates this is the final event of a turn.
	TurnComplete bool `json:"turn_complete,omitempty"`

	// Interrupted indicates this event was cut off due to cancellation or timeout.
	Interrupted bool `json:"interrupted,omitempty"`

	// ErrorCode is a machine-readable error identifier (e.g., "rate_limit", "timeout").
	ErrorCode string `json:"-"`

	// ErrorMessage is a human-readable error description.
	ErrorMessage string `json:"-"`

	// OnPersisted is a callback invoked after the event is persisted to the session.
	// Used by async agents (like ParallelAgent) to synchronize with the persistence layer.
	// This field is not serialized.
	OnPersisted func() `json:"-"`
}

// ThinkingState represents the model's reasoning process.
// Maps to ThinkingWidget in the UI with proper lifecycle animations.
type ThinkingState struct {
	// ID uniquely identifies this thinking block within a conversation.
	ID string `json:"id"`

	// Status indicates the lifecycle state: "active" | "completed".
	Status string `json:"status"`

	// Content is the thinking text (may stream incrementally).
	Content string `json:"content"`

	// Signature is used for multi-turn verification (e.g., Anthropic).
	Signature string `json:"signature,omitempty"`

	// Type categorizes the thinking: "default" | "planning" | "reflection" | "goal".
	Type string `json:"type,omitempty"`
}

// ToolCallState represents a tool invocation.
// Maps to ToolWidget in the UI with "working" status.
type ToolCallState struct {
	// ID uniquely identifies this tool call (matches LLM's tool_use ID).
	ID string `json:"id"`

	// Name is the tool being called.
	Name string `json:"name"`

	// Args are the arguments passed to the tool.
	Args map[string]any `json:"args"`

	// Status indicates lifecycle: "pending" | "working".
	Status string `json:"status"`
}

// ToolResultState represents a tool execution result.
// Updates the corresponding ToolWidget to "success" | "failed".
type ToolResultState struct {
	// ToolCallID links this result to its ToolCallState.
	ToolCallID string `json:"tool_call_id"`

	// Content is the tool's output.
	Content string `json:"content"`

	// Status indicates outcome: "success" | "failed".
	Status string `json:"status"`

	// IsError indicates if Content represents an error message.
	IsError bool `json:"is_error,omitempty"`

	// Metadata contains custom metadata (e.g., intervention details).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewEvent creates a new event with generated ID and current timestamp.
func NewEvent(invocationID string) *Event {
	return &Event{
		ID:           uuid.NewString(),
		Timestamp:    time.Now(),
		InvocationID: invocationID,
		Metadata:     make(map[string]any),
		Actions:      EventActions{StateDelta: make(map[string]any)},
	}
}

// ensureMetadata initializes the Metadata map if nil.
func (e *Event) ensureMetadata() {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
}

// Thinking returns the thinking state from metadata.
func (e *Event) Thinking() *ThinkingState {
	if e.Metadata == nil {
		return nil
	}
	v, ok := e.Metadata[MetaKeyThinking]
	if !ok {
		return nil
	}
	// Handle both direct ThinkingState and map[string]any from JSON
	switch t := v.(type) {
	case *ThinkingState:
		return t
	case ThinkingState:
		return &t
	case map[string]any:
		return &ThinkingState{
			ID:        getString(t, "id"),
			Status:    getString(t, "status"),
			Content:   getString(t, "content"),
			Signature: getString(t, "signature"),
			Type:      getString(t, "type"),
		}
	}
	return nil
}

// SetThinking stores the thinking state in metadata.
func (e *Event) SetThinking(t *ThinkingState) {
	e.ensureMetadata()
	if t == nil {
		delete(e.Metadata, MetaKeyThinking)
		return
	}
	e.Metadata[MetaKeyThinking] = map[string]any{
		"id":      t.ID,
		"status":  t.Status,
		"content": t.Content,
		"type":    t.Type,
	}
	if t.Signature != "" {
		e.Metadata[MetaKeyThinking].(map[string]any)["signature"] = t.Signature
	}
}

// ToolCalls returns the tool calls from metadata.
func (e *Event) ToolCalls() []ToolCallState {
	if e.Metadata == nil {
		return nil
	}
	v, ok := e.Metadata[MetaKeyToolCalls]
	if !ok {
		return nil
	}
	// Handle various storage formats
	switch t := v.(type) {
	case []ToolCallState:
		return t
	case []map[string]any:
		// Stored by SetToolCalls
		result := make([]ToolCallState, 0, len(t))
		for _, m := range t {
			tc := ToolCallState{
				ID:     getString(m, "id"),
				Name:   getString(m, "name"),
				Status: getString(m, "status"),
			}
			if args, ok := m["args"].(map[string]any); ok {
				tc.Args = args
			}
			result = append(result, tc)
		}
		return result
	case []any:
		// From JSON deserialization
		result := make([]ToolCallState, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				tc := ToolCallState{
					ID:     getString(m, "id"),
					Name:   getString(m, "name"),
					Status: getString(m, "status"),
				}
				if args, ok := m["args"].(map[string]any); ok {
					tc.Args = args
				}
				result = append(result, tc)
			}
		}
		return result
	}
	return nil
}

// SetToolCalls stores the tool calls in metadata.
func (e *Event) SetToolCalls(tc []ToolCallState) {
	e.ensureMetadata()
	if len(tc) == 0 {
		delete(e.Metadata, MetaKeyToolCalls)
		return
	}
	calls := make([]map[string]any, len(tc))
	for i, c := range tc {
		calls[i] = map[string]any{
			"id":     c.ID,
			"name":   c.Name,
			"args":   c.Args,
			"status": c.Status,
		}
	}
	e.Metadata[MetaKeyToolCalls] = calls
}

// ToolResults returns the tool results from metadata.
func (e *Event) ToolResults() []ToolResultState {
	if e.Metadata == nil {
		return nil
	}
	v, ok := e.Metadata[MetaKeyToolResults]
	if !ok {
		return nil
	}
	// Handle various storage formats
	switch t := v.(type) {
	case []ToolResultState:
		return t
	case []map[string]any:
		// Stored by SetToolResults
		result := make([]ToolResultState, 0, len(t))
		for _, m := range t {
			tr := ToolResultState{
				ToolCallID: getString(m, "tool_call_id"),
				Content:    getString(m, "content"),
				Status:     getString(m, "status"),
			}
			if isErr, ok := m["is_error"].(bool); ok {
				tr.IsError = isErr
			}
			if meta, ok := m["metadata"].(map[string]any); ok {
				tr.Metadata = meta
			}
			result = append(result, tr)
		}
		return result
	case []any:
		// From JSON deserialization
		result := make([]ToolResultState, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				tr := ToolResultState{
					ToolCallID: getString(m, "tool_call_id"),
					Content:    getString(m, "content"),
					Status:     getString(m, "status"),
				}
				if isErr, ok := m["is_error"].(bool); ok {
					tr.IsError = isErr
				}
				if meta, ok := m["metadata"].(map[string]any); ok {
					tr.Metadata = meta
				}
				result = append(result, tr)
			}
		}
		return result
	}
	return nil
}

// SetToolResults stores the tool results in metadata.
func (e *Event) SetToolResults(tr []ToolResultState) {
	e.ensureMetadata()
	if len(tr) == 0 {
		delete(e.Metadata, MetaKeyToolResults)
		return
	}
	results := make([]map[string]any, len(tr))
	for i, r := range tr {
		results[i] = map[string]any{
			"tool_call_id": r.ToolCallID,
			"content":      r.Content,
			"status":       r.Status,
			"is_error":     r.IsError,
		}
		if r.Metadata != nil {
			results[i]["metadata"] = r.Metadata
		}
	}
	e.Metadata[MetaKeyToolResults] = results
}

// getString safely extracts a string from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetCustomMetadata merges custom metadata into the event metadata.
func (e *Event) SetCustomMetadata(custom map[string]any) {
	e.ensureMetadata()
	for k, v := range custom {
		e.Metadata[k] = v
	}
}

// CustomMetadata returns the metadata map.
// This returns the full Metadata map which includes both custom and system keys.
func (e *Event) CustomMetadata() map[string]any {
	return e.Metadata
}

// SetErrorCode sets the error code in metadata.
func (e *Event) SetErrorCode(code string) {
	e.ensureMetadata()
	e.ErrorCode = code
	if code != "" {
		e.Metadata[MetaKeyErrorCode] = code
	} else {
		delete(e.Metadata, MetaKeyErrorCode)
	}
}

// SetErrorMessage sets the error message in metadata.
func (e *Event) SetErrorMessage(msg string) {
	e.ensureMetadata()
	e.ErrorMessage = msg
	if msg != "" {
		e.Metadata[MetaKeyErrorMessage] = msg
	} else {
		delete(e.Metadata, MetaKeyErrorMessage)
	}
}

// SetLongRunningToolIDs sets the long running tool IDs in metadata.
func (e *Event) SetLongRunningToolIDs(ids []string) {
	e.ensureMetadata()
	e.LongRunningToolIDs = ids
	if len(ids) > 0 {
		e.Metadata[MetaKeyLongRunningToolIDs] = ids
	} else {
		delete(e.Metadata, MetaKeyLongRunningToolIDs)
	}
}

// Parts returns the artifact parts, or nil if no artifact.
func (e *Event) Parts() []a2a.Part {
	if e.Artifact == nil {
		return nil
	}
	return e.Artifact.Parts
}

// SetParts sets the artifact parts.
func (e *Event) SetParts(parts []a2a.Part) {
	if e.Artifact == nil {
		e.Artifact = &a2a.Artifact{}
	}
	e.Artifact.Parts = parts
}

// EventActions represents side effects attached to an event.
type EventActions struct {
	// StateDelta contains key-value changes to session state.
	StateDelta map[string]any

	// ArtifactDelta tracks artifact updates (filename -> version).
	ArtifactDelta map[string]int64

	// SkipSummarization prevents LLM summarization of tool responses.
	SkipSummarization bool

	// TransferToAgent requests control transfer to another agent.
	TransferToAgent string

	// Escalate requests escalation to a higher-level agent.
	Escalate bool

	// RequireInput signals that human input is required (HITL pattern).
	// When true, the task transitions to `input_required` state.
	// This is used by long-running tools that need approval or additional info.
	RequireInput bool

	// InputPrompt is the message shown to the human when RequireInput is true.
	// Should explain what input is needed and why.
	InputPrompt string
}

// IsFinalResponse returns whether this event is a final response.
// Multiple events can be final when multiple agents participate in one invocation.
//
// An event is NOT final if it:
// - Contains function/tool calls (awaiting execution)
// - Contains function/tool responses (awaiting LLM summarization)
// - Is a partial/streaming event
func (e *Event) IsFinalResponse() bool {
	// SkipSummarization or long-running tools are explicitly final
	if e.Actions.SkipSummarization || len(e.LongRunningToolIDs) > 0 {
		return true
	}

	// Partial events are never final
	if e.Partial {
		return false
	}

	// Events with tool calls or results are not final
	if e.HasToolCalls() || e.HasToolResults() {
		return false
	}

	return true
}

// HasToolCalls returns true if this event contains tool call requests.
// Tool calls indicate the LLM wants to execute tools before providing a final response.
func (e *Event) HasToolCalls() bool {
	// Check ToolCalls from metadata first (preferred)
	if len(e.ToolCalls()) > 0 {
		return true
	}

	// Fall back to checking artifact/message parts (A2A protocol)
	return hasPartOfType(e.Parts(), "tool_use")
}

// HasToolResults returns true if this event contains tool execution results.
// Tool results indicate tools have been executed and the LLM needs to process them.
func (e *Event) HasToolResults() bool {
	// Check ToolResults from metadata first (preferred)
	if len(e.ToolResults()) > 0 {
		return true
	}

	// Fall back to checking artifact/message parts (A2A protocol)
	return hasPartOfType(e.Parts(), "tool_result")
}

// hasPartOfType checks if parts contain a DataPart with the specified type.
func hasPartOfType(parts []a2a.Part, partType string) bool {
	if parts == nil {
		return false
	}

	for _, part := range parts {
		if dp, ok := part.(a2a.DataPart); ok {
			if typeVal, hasType := dp.Data["type"].(string); hasType && typeVal == partType {
				return true
			}
		}
	}
	return false
}

// TextContent extracts text content from the event's artifact/message.
func (e *Event) TextContent() string {
	parts := e.Parts()
	if parts == nil {
		return ""
	}

	var text string
	for _, part := range parts {
		if tp, ok := part.(a2a.TextPart); ok {
			text += tp.Text
		}
	}
	return text
}

// Content is a convenience type for building message content.
type Content struct {
	Parts []a2a.Part
	Role  a2a.MessageRole
}

// NewTextContent creates content with a text part.
func NewTextContent(text string, role a2a.MessageRole) *Content {
	return &Content{
		Parts: []a2a.Part{a2a.TextPart{Text: text}},
		Role:  role,
	}
}

// ToMessage converts Content to an a2a.Message.
func (c *Content) ToMessage() *a2a.Message {
	if c == nil {
		return nil
	}
	return a2a.NewMessage(c.Role, c.Parts...)
}

// AddPart appends a part to the content.
func (c *Content) AddPart(part a2a.Part) {
	c.Parts = append(c.Parts, part)
}

// AddText appends a text part to the content.
func (c *Content) AddText(text string) {
	c.Parts = append(c.Parts, a2a.TextPart{Text: text})
}

// BuildInterventionMetadata creates standardized metadata for guardrail/moderation events.
// This ensures consistent structure across all intervention events for traceability.
//
// Parameters:
//   - metadataType: MetadataTypeGuardrail or MetadataTypeModeration
//   - action: InterventionActionBlock, InterventionActionModify, or InterventionActionWarn
//   - source: InterventionSourceInputGuardrail, InterventionSourceModeration, etc.
//   - reason: Human-readable explanation of the intervention
//   - extras: Additional context (category, confidence, provider, severity, etc.)
//
// Example:
//
//	metadata := agent.BuildInterventionMetadata(
//	    agent.MetadataTypeGuardrail,
//	    agent.InterventionActionBlock,
//	    agent.InterventionSourceInputGuardrail,
//	    "Injection detected: 'ignore previous instructions'",
//	    map[string]any{"category": "injection", "severity": "high"},
//	)
func BuildInterventionMetadata(metadataType, action, source, reason string, extras map[string]any) map[string]any {
	intervention := map[string]any{
		"action": action,
		"source": source,
		"reason": reason,
	}
	for k, v := range extras {
		intervention[k] = v
	}
	return map[string]any{
		"type":         metadataType,
		"intervention": intervention,
	}
}
