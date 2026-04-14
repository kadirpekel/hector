package server

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"

	"github.com/verikod/hector/pkg/execution"
)

// QueuedRequestHandler wraps a a2asrv.RequestHandler and intercepts async
// execution requests to enqueue them for durable processing.
// It implements a2asrv.RequestHandler.
type QueuedRequestHandler struct {
	delegate  a2asrv.RequestHandler
	provider  execution.Provider
	appID     string
	agentName string
}

// NewQueuedRequestHandler creates a new QueuedRequestHandler.
func NewQueuedRequestHandler(delegate a2asrv.RequestHandler, provider execution.Provider, appID, agentName string) *QueuedRequestHandler {
	return &QueuedRequestHandler{
		delegate:  delegate,
		provider:  provider,
		appID:     appID,
		agentName: agentName,
	}
}

// OnSendMessage handles message sending requests.
// Intercepts requests with blocking=false and enqueues them.
func (h *QueuedRequestHandler) OnSendMessage(ctx context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	// Check if this is an async request (blocking=false or nil defaults to async in A2A)
	isAsync := params.Config == nil || params.Config.Blocking == nil || !*params.Config.Blocking

	// If sync or no queue available, delegate
	if !isAsync || h.provider == nil {
		return h.delegate.OnSendMessage(ctx, params)
	}

	// === ASYNC REQUEST: Enqueue for durable execution ===

	// Extract input text from message parts
	input := extractTextFromMessage(params.Message)
	if input == "" {
		// If no text, delegate (might be special message type)
		return h.delegate.OnSendMessage(ctx, params)
	}

	// Generate context ID if not provided
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("api-%s-%d", h.agentName, time.Now().UnixNano())
	}

	// Create queue item
	// Note: We create a unique task ID different from context ID
	// a2a-go usually generates UUIDs for task IDs
	taskID := contextID // Simple default

	// Create execution request
	req := &execution.Request{
		ID:        taskID, // This is the TaskID. Provider might generate a separate ExecutionID.
		AppID:     h.appID,
		AgentName: h.agentName,
		Input:     input,
		ContextID: contextID,
		Priority:  0,
		Metadata:  map[string]any{},
	}

	// Enqueue via provider
	if err := h.provider.Submit(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to submit task execution: %w", err)
	}

	// Return a task response matching A2A spec
	now := time.Now()
	taskResp := &a2a.Task{
		ID:        a2a.TaskID(taskID),
		ContextID: contextID,
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateSubmitted,
			Timestamp: &now,
		},
	}

	return taskResp, nil
}

// OnCancelTask delegates to the underlying handler.
func (h *QueuedRequestHandler) OnCancelTask(ctx context.Context, params *a2a.TaskIDParams) (*a2a.Task, error) {
	return h.delegate.OnCancelTask(ctx, params)
}

// OnDeleteTaskPushConfig delegates to the underlying handler.
func (h *QueuedRequestHandler) OnDeleteTaskPushConfig(ctx context.Context, params *a2a.DeleteTaskPushConfigParams) error {
	return h.delegate.OnDeleteTaskPushConfig(ctx, params)
}

// OnGetTask delegates to the underlying handler.
func (h *QueuedRequestHandler) OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error) {
	return h.delegate.OnGetTask(ctx, params)
}

// OnGetExtendedAgentCard delegates to the underlying handler.
func (h *QueuedRequestHandler) OnGetExtendedAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	return h.delegate.OnGetExtendedAgentCard(ctx)
}

// OnListTaskPushConfig delegates to the underlying handler.
func (h *QueuedRequestHandler) OnListTaskPushConfig(ctx context.Context, params *a2a.ListTaskPushConfigParams) ([]*a2a.TaskPushConfig, error) {
	return h.delegate.OnListTaskPushConfig(ctx, params)
}

// OnResubscribeToTask delegates to the underlying handler.
func (h *QueuedRequestHandler) OnResubscribeToTask(ctx context.Context, params *a2a.TaskIDParams) iter.Seq2[a2a.Event, error] {
	return h.delegate.OnResubscribeToTask(ctx, params)
}

// OnSendMessageStream delegates to the underlying handler.
func (h *QueuedRequestHandler) OnSendMessageStream(ctx context.Context, params *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	return h.delegate.OnSendMessageStream(ctx, params)
}

// OnGetTaskPushConfig delegates to the underlying handler.
func (h *QueuedRequestHandler) OnGetTaskPushConfig(ctx context.Context, params *a2a.GetTaskPushConfigParams) (*a2a.TaskPushConfig, error) {
	return h.delegate.OnGetTaskPushConfig(ctx, params)
}

// OnSetTaskPushConfig delegates to the underlying handler.
func (h *QueuedRequestHandler) OnSetTaskPushConfig(ctx context.Context, params *a2a.TaskPushConfig) (*a2a.TaskPushConfig, error) {
	return h.delegate.OnSetTaskPushConfig(ctx, params)
}

// extractTextFromMessage extracts text content from an A2A message.
func extractTextFromMessage(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	var texts []string
	for _, part := range msg.Parts {
		if txt, ok := part.(a2a.TextPart); ok {
			texts = append(texts, txt.Text)
		}
	}
	if len(texts) > 0 {
		return texts[0] // Just take first part for now, or join if needed
	}
	return ""
}
