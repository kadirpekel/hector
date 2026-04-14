package agent

import (
	"context"
	"iter"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/google/uuid"
)

/*
InvocationContext represents the context of an agent invocation.

An invocation:
 1. Starts with a user message and ends with a final response.
 2. Can contain one or multiple agent calls.
 3. Is handled by runner.Run().

An agent call:
 1. Is handled by agent.Run().
 2. Ends when agent.Run() completes.

An agent call can contain multiple steps (LLM calls + tool executions).

	┌─────────────────────── invocation ──────────────────────────┐
	┌──────────── llm_agent_call_1 ────────────┐ ┌─ agent_call_2 ─┐
	┌──── step_1 ────────┐ ┌───── step_2 ──────┐
	[call_llm] [call_tool] [call_llm] [transfer]
*/
type InvocationContext interface {
	// Embed CallbackContext which embeds ReadonlyContext
	// This allows InvocationContext to be used wherever ReadonlyContext
	// or CallbackContext is expected.
	CallbackContext

	// Agent returns the current agent being executed.
	Agent() Agent

	// Session returns the session for this invocation.
	Session() Session

	// Memory provides access to cross-session memory.
	Memory() Memory

	// RunConfig returns the runtime configuration for this invocation.
	RunConfig() *RunConfig

	// EndInvocation signals that the invocation should stop.
	EndInvocation()

	// Ended returns whether the invocation has been ended.
	Ended() bool
}

// ReadonlyContext provides read-only access to invocation data.
// Safe to pass to tools and external code.
type ReadonlyContext interface {
	context.Context

	// InvocationID returns the unique ID for this invocation.
	InvocationID() string

	// AgentName returns the current agent's name.
	AgentName() string

	// UserContent returns the user message that started this invocation.
	UserContent() *Content

	// ReadonlyState returns read-only access to session state.
	ReadonlyState() ReadonlyState

	// UserID returns the user identifier.
	UserID() string

	// AppName returns the application name.
	AppName() string

	// SessionID returns the session identifier.
	SessionID() string

	// Branch returns the agent hierarchy path.
	Branch() string
}

// CallbackContext provides state modification for callbacks.
type CallbackContext interface {
	ReadonlyContext

	// Artifacts returns the artifact service.
	Artifacts() Artifacts

	// State returns mutable session state.
	State() State

	// SetMetadata sets custom metadata to be included in the resulting event.
	// Used by guardrails and moderation to attach intervention details.
	SetMetadata(metadata map[string]any)

	// Metadata returns the metadata set by callbacks.
	Metadata() map[string]any
}

// Session represents a conversation session.
// Defined here to avoid circular imports with session package.
type Session interface {
	ID() string
	AppName() string
	UserID() string
	State() State
	Events() Events
}

// State is a mutable key-value store for session state.
type State interface {
	Get(key string) (any, error)
	Set(key string, value any) error
	Delete(key string) error
	All() iter.Seq2[string, any]
}

// TempClearable is implemented by state stores that support clearing temp keys.
type TempClearable interface {
	// ClearTempKeys removes all keys with the "temp:" prefix.
	// Called automatically after each invocation completes.
	ClearTempKeys()
}

// ReadonlyState provides read-only access to session state.
type ReadonlyState interface {
	Get(key string) (any, error)
	All() iter.Seq2[string, any]
}

// Events provides access to session event history.
type Events interface {
	All() iter.Seq[*Event]
	Len() int
	At(i int) *Event
}

// Artifacts provides artifact storage operations.
type Artifacts interface {
	Save(ctx context.Context, name string, part a2a.Part) (*ArtifactSaveResponse, error)
	List(ctx context.Context) (*ArtifactListResponse, error)
	Load(ctx context.Context, name string) (*ArtifactLoadResponse, error)
	LoadVersion(ctx context.Context, name string, version int) (*ArtifactLoadResponse, error)
}

// ArtifactSaveResponse is returned when saving an artifact.
type ArtifactSaveResponse struct {
	Name    string
	Version int64
}

// ArtifactListResponse is returned when listing artifacts.
type ArtifactListResponse struct {
	Artifacts []ArtifactInfo
}

// ArtifactInfo describes a stored artifact.
type ArtifactInfo struct {
	Name    string
	Version int64
}

// ArtifactLoadResponse is returned when loading an artifact.
type ArtifactLoadResponse struct {
	Name    string
	Version int64
	Part    a2a.Part
}

// Memory provides cross-session memory operations.
type Memory interface {
	AddSession(ctx context.Context, session Session) error
	Search(ctx context.Context, query string) (*MemorySearchResponse, error)
}

// MemorySearchResponse contains memory search results.
type MemorySearchResponse struct {
	Results []MemoryResult
}

// MemoryResult is a single memory search result.
type MemoryResult struct {
	Content  string
	Score    float64
	Metadata map[string]any
}

// CancellableTask provides task-level cancellation support.
// Defined in agent package to avoid circular imports with pkg/task.
// The task.Task type implements this interface.
type CancellableTask interface {
	// RegisterExecution registers a child execution for cascade cancellation.
	RegisterExecution(exec *ChildExecution)

	// UnregisterExecution removes a child execution from tracking.
	UnregisterExecution(callID string)

	// CancelExecution cancels a specific child execution.
	CancelExecution(callID string) bool

	// GetID returns the task ID.
	GetID() string
}

// ChildExecution represents an active tool or sub-agent execution.
// Used for cascade cancellation when a task is cancelled.
type ChildExecution struct {
	// CallID is the unique identifier for this execution.
	CallID string

	// Name is the tool or agent name.
	Name string

	// Type is either "tool" or "agent".
	Type string

	// Cancel is called to cancel this execution.
	Cancel func() bool
}

// RunConfig contains runtime configuration for an invocation.
type RunConfig struct {
	// StreamingMode controls event streaming behavior.
	StreamingMode StreamingMode

	// SaveInputBlobsAsArtifacts saves file inputs as artifacts.
	SaveInputBlobsAsArtifacts bool

	// Task provides access to the parent task for cascade cancellation.
	// This is set by the executor when a task is associated with the invocation.
	Task CancellableTask

	// SkipAppendUserMessage skips appending the user message to session.
	// Used for checkpoint recovery where the message already exists in history.
	SkipAppendUserMessage bool
}

// StreamingMode controls how events are streamed.
type StreamingMode string

const (
	StreamingModeNone StreamingMode = "none"
	StreamingModeSSE  StreamingMode = "sse"
	StreamingModeFull StreamingMode = "full"
)

// invocationContext is the concrete implementation of InvocationContext.
type invocationContext struct {
	context.Context

	agent        Agent
	session      Session
	artifacts    Artifacts
	memory       Memory
	invocationID string
	branch       string
	userContent  *Content
	runConfig    *RunConfig
	ended        bool
	metadata     map[string]any
}

// InvocationContextParams contains parameters for creating an InvocationContext.
type InvocationContextParams struct {
	Artifacts   Artifacts
	Memory      Memory
	Session     Session
	Agent       Agent
	Branch      string
	UserContent *Content
	RunConfig   *RunConfig
}

// NewInvocationContext creates a new InvocationContext.
func NewInvocationContext(ctx context.Context, params InvocationContextParams) InvocationContext {
	invocationID := uuid.NewString()
	return &invocationContext{
		Context:      ctx,
		agent:        params.Agent,
		session:      params.Session,
		artifacts:    params.Artifacts,
		memory:       params.Memory,
		invocationID: invocationID,
		branch:       params.Branch,
		userContent:  params.UserContent,
		runConfig:    params.RunConfig,
	}
}

func (c *invocationContext) Agent() Agent          { return c.agent }
func (c *invocationContext) Session() Session      { return c.session }
func (c *invocationContext) Artifacts() Artifacts  { return c.artifacts }
func (c *invocationContext) Memory() Memory        { return c.memory }
func (c *invocationContext) InvocationID() string  { return c.invocationID }
func (c *invocationContext) Branch() string        { return c.branch }
func (c *invocationContext) UserContent() *Content { return c.userContent }
func (c *invocationContext) RunConfig() *RunConfig { return c.runConfig }
func (c *invocationContext) EndInvocation()        { c.ended = true }
func (c *invocationContext) Ended() bool           { return c.ended }

// ReadonlyContext implementation for InvocationContext
func (c *invocationContext) AgentName() string {
	if c.agent != nil {
		return c.agent.Name()
	}
	return ""
}

func (c *invocationContext) ReadonlyState() ReadonlyState {
	if c.session != nil {
		return c.session.State()
	}
	return nil
}

func (c *invocationContext) UserID() string {
	if c.session != nil {
		return c.session.UserID()
	}
	return ""
}

func (c *invocationContext) AppName() string {
	if c.session != nil {
		return c.session.AppName()
	}
	return ""
}

func (c *invocationContext) SessionID() string {
	if c.session != nil {
		return c.session.ID()
	}
	return ""
}

// CallbackContext implementation for InvocationContext
func (c *invocationContext) State() State {
	if c.session != nil {
		return c.session.State()
	}
	return nil
}

func (c *invocationContext) SetMetadata(metadata map[string]any) {
	if c.metadata == nil {
		c.metadata = make(map[string]any)
	}
	for k, v := range metadata {
		c.metadata[k] = v
	}
}

func (c *invocationContext) Metadata() map[string]any {
	return c.metadata
}

// callbackContext implements CallbackContext.
type callbackContext struct {
	context.Context
	invCtx   InvocationContext
	actions  *EventActions
	metadata map[string]any
}

func newCallbackContext(invCtx InvocationContext) *callbackContext {
	return &callbackContext{
		Context: invCtx,
		invCtx:  invCtx,
		actions: &EventActions{StateDelta: make(map[string]any)},
	}
}

func (c *callbackContext) InvocationID() string  { return c.invCtx.InvocationID() }
func (c *callbackContext) AgentName() string     { return c.invCtx.Agent().Name() }
func (c *callbackContext) UserContent() *Content { return c.invCtx.UserContent() }
func (c *callbackContext) Branch() string        { return c.invCtx.Branch() }
func (c *callbackContext) Artifacts() Artifacts  { return c.invCtx.Artifacts() }

func (c *callbackContext) UserID() string {
	if c.invCtx.Session() != nil {
		return c.invCtx.Session().UserID()
	}
	return ""
}

func (c *callbackContext) AppName() string {
	if c.invCtx.Session() != nil {
		return c.invCtx.Session().AppName()
	}
	return ""
}

func (c *callbackContext) SessionID() string {
	if c.invCtx.Session() != nil {
		return c.invCtx.Session().ID()
	}
	return ""
}

func (c *callbackContext) ReadonlyState() ReadonlyState {
	if c.invCtx.Session() != nil {
		return c.invCtx.Session().State()
	}
	return nil
}

func (c *callbackContext) State() State {
	return &callbackState{
		ctx:   c,
		state: c.invCtx.Session().State(),
	}
}

func (c *callbackContext) SetMetadata(metadata map[string]any) {
	if c.metadata == nil {
		c.metadata = make(map[string]any)
	}
	for k, v := range metadata {
		c.metadata[k] = v
	}
}

func (c *callbackContext) Metadata() map[string]any {
	return c.metadata
}

// callbackState wraps State to track modifications in actions.
type callbackState struct {
	ctx   *callbackContext
	state State
}

func (s *callbackState) Get(key string) (any, error) {
	// Check delta first
	if val, ok := s.ctx.actions.StateDelta[key]; ok {
		return val, nil
	}
	return s.state.Get(key)
}

func (s *callbackState) Set(key string, val any) error {
	s.ctx.actions.StateDelta[key] = val
	return s.state.Set(key, val)
}

func (s *callbackState) Delete(key string) error {
	// Track deletion in delta (set to nil)
	s.ctx.actions.StateDelta[key] = nil
	return s.state.Delete(key)
}

func (s *callbackState) All() iter.Seq2[string, any] {
	return s.state.All()
}

var (
	_ InvocationContext = (*invocationContext)(nil)
	_ ReadonlyContext   = (*invocationContext)(nil)
	_ CallbackContext   = (*invocationContext)(nil)
	_ CallbackContext   = (*callbackContext)(nil)
	_ State             = (*callbackState)(nil)
)
