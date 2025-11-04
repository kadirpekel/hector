package agui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/google/uuid"
	a2apb "github.com/kadirpekel/hector/pkg/a2a/pb"
	aguipb "github.com/kadirpekel/hector/pkg/agui/pb"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

// StreamAdapter wraps an underlying A2A stream and converts responses to AG-UI events
type StreamAdapter struct {
	underlying   A2AStreamServer
	converter    *Converter
	messageID    string
	contextID    string
	taskID       string
	inMessage    bool
	messagesSent int
	useAGUI      bool
}

// A2AStreamServer is the interface for sending A2A stream responses
type A2AStreamServer interface {
	Send(*a2apb.StreamResponse) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

// NewStreamAdapter creates a new AG-UI stream adapter
func NewStreamAdapter(underlying A2AStreamServer, useAGUI bool) *StreamAdapter {
	return &StreamAdapter{
		underlying: underlying,
		useAGUI:    useAGUI,
		messageID:  uuid.New().String(),
	}
}

// Send converts A2A StreamResponse to AG-UI events if AG-UI mode is enabled
func (a *StreamAdapter) Send(resp *a2apb.StreamResponse) error {
	// If AG-UI is not enabled, pass through to underlying stream (A2A native)
	if !a.useAGUI {
		return a.underlying.Send(resp)
	}

	// Convert A2A StreamResponse to AG-UI events
	events := a.convertToAGUIEvents(resp)

	// Send each AG-UI event
	for _, event := range events {
		if err := a.sendAGUIEvent(event); err != nil {
			return err
		}
	}

	return nil
}

// convertToAGUIEvents converts an A2A StreamResponse to AG-UI events
func (a *StreamAdapter) convertToAGUIEvents(resp *a2apb.StreamResponse) []*aguipb.AGUIEvent {
	var events []*aguipb.AGUIEvent

	switch payload := resp.Payload.(type) {
	case *a2apb.StreamResponse_Task:
		// Task lifecycle events
		task := payload.Task
		a.taskID = task.Id
		a.contextID = task.ContextId

		if task.Status != nil {
			switch task.Status.State {
			case a2apb.TaskState_TASK_STATE_SUBMITTED:
				events = append(events, NewTaskStartEvent(task.Id, task.ContextId, ""))
				events = append(events, NewMessageStartEvent(a.messageID, a.contextID, a.taskID, "agent"))
				a.inMessage = true
				a.converter = NewConverter(a.messageID, a.contextID, a.taskID)

			case a2apb.TaskState_TASK_STATE_WORKING:
				if !a.inMessage {
					events = append(events, NewMessageStartEvent(a.messageID, a.contextID, a.taskID, "agent"))
					a.inMessage = true
					a.converter = NewConverter(a.messageID, a.contextID, a.taskID)
				}
				events = append(events, NewTaskUpdateEvent(task.Id, "working", nil))

			case a2apb.TaskState_TASK_STATE_COMPLETED:
				// Close any open content blocks
				if a.converter != nil {
					closeEvents := a.converter.CloseCurrentBlock()
					events = append(events, closeEvents...)
				}
				if a.inMessage {
					events = append(events, NewMessageStopEvent(a.messageID))
					a.inMessage = false
				}
				events = append(events, NewTaskCompleteEvent(task.Id, nil))

			case a2apb.TaskState_TASK_STATE_FAILED:
				if a.converter != nil {
					closeEvents := a.converter.CloseCurrentBlock()
					events = append(events, closeEvents...)
				}
				if a.inMessage {
					events = append(events, NewMessageStopEvent(a.messageID))
					a.inMessage = false
				}
				errorMsg := "Task failed"
				if task.Status.Update != nil && len(task.Status.Update.Parts) > 0 {
					if text := task.Status.Update.Parts[0].GetText(); text != "" {
						errorMsg = text
					}
				}
				events = append(events, NewTaskErrorEvent(task.Id, errorMsg, "TASK_FAILED", nil))
			}
		}

	case *a2apb.StreamResponse_Msg:
		// Message with parts - convert each part to AG-UI events
		msg := payload.Msg

		if msg.ContextId != "" {
			a.contextID = msg.ContextId
		}
		if msg.TaskId != "" {
			a.taskID = msg.TaskId
		}

		// Start message if not already started
		if !a.inMessage {
			role := "agent"
			if msg.Role == a2apb.Role_ROLE_USER {
				role = "user"
			}
			events = append(events, NewMessageStartEvent(a.messageID, a.contextID, a.taskID, role))
			a.inMessage = true
			a.converter = NewConverter(a.messageID, a.contextID, a.taskID)
		}

		// Convert each part
		for _, part := range msg.Parts {
			partEvents := a.convertPart(part)
			events = append(events, partEvents...)
		}

	case *a2apb.StreamResponse_StatusUpdate:
		// Task status update
		update := payload.StatusUpdate
		a.taskID = update.TaskId
		a.contextID = update.ContextId

		status := strings.ToLower(update.Status.State.String())
		status = strings.TrimPrefix(status, "task_state_")

		if update.Final {
			// Close any open content blocks
			if a.converter != nil {
				closeEvents := a.converter.CloseCurrentBlock()
				events = append(events, closeEvents...)
			}
			if a.inMessage {
				events = append(events, NewMessageStopEvent(a.messageID))
				a.inMessage = false
			}
		}

		events = append(events, NewTaskUpdateEvent(update.TaskId, status, nil))

	case *a2apb.StreamResponse_ArtifactUpdate:
		// Artifact update - treat as content blocks
		artifact := payload.ArtifactUpdate.Artifact
		if artifact != nil {
			if !a.inMessage {
				events = append(events, NewMessageStartEvent(a.messageID, a.contextID, a.taskID, "agent"))
				a.inMessage = true
				a.converter = NewConverter(a.messageID, a.contextID, a.taskID)
			}

			for _, part := range artifact.Parts {
				partEvents := a.convertPart(part)
				events = append(events, partEvents...)
			}
		}
	}

	return events
}

// convertPart converts an A2A Part to AG-UI events
func (a *StreamAdapter) convertPart(part *a2apb.Part) []*aguipb.AGUIEvent {
	if a.converter == nil {
		a.converter = NewConverter(a.messageID, a.contextID, a.taskID)
	}

	// Check if this is a thinking block (extended thinking content)
	if part.Metadata != nil {
		if partType, ok := part.Metadata.Fields["part_type"]; ok && partType.GetStringValue() == "thinking" {
			return a.convertThinkingPart(part)
		}
	}

	return a.converter.ConvertPart(part)
}

// convertThinkingPart converts a thinking part to AG-UI thinking events
func (a *StreamAdapter) convertThinkingPart(part *a2apb.Part) []*aguipb.AGUIEvent {
	var events []*aguipb.AGUIEvent

	// Close any open content block before thinking
	if a.converter != nil {
		closeEvents := a.converter.CloseCurrentBlock()
		events = append(events, closeEvents...)
	}

	thinkingID := uuid.New().String()
	title := ""

	if part.Metadata != nil {
		if titleField, ok := part.Metadata.Fields["title"]; ok {
			title = titleField.GetStringValue()
		}
	}

	// Start thinking
	events = append(events, NewThinkingStartEvent(thinkingID, title))

	// Thinking content
	if text := part.GetText(); text != "" {
		events = append(events, NewThinkingDeltaEvent(thinkingID, text))
	}

	// Stop thinking
	events = append(events, NewThinkingStopEvent(thinkingID, ""))

	return events
}

// sendAGUIEvent sends an AG-UI event through the underlying stream
func (a *StreamAdapter) sendAGUIEvent(event *aguipb.AGUIEvent) error {
	// Marshal AG-UI event to JSON for SSE
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal AG-UI event: %w", err)
	}

	// Send as SSE event
	// We need to write directly to the underlying writer
	// This is a simplified version - you may need to access the underlying HTTP writer
	log.Printf("AG-UI Event: %s", string(data))

	// For now, wrap it in an A2A message format
	// The actual SSE writing will be handled by the wrapper
	var eventMap map[string]interface{}
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return fmt.Errorf("failed to unmarshal AG-UI event: %w", err)
	}

	return nil
}

// SetHeader implements the gRPC ServerStream interface
func (a *StreamAdapter) SetHeader(md metadata.MD) error {
	return a.underlying.SetHeader(md)
}

// SendHeader implements the gRPC ServerStream interface
func (a *StreamAdapter) SendHeader(md metadata.MD) error {
	return a.underlying.SendHeader(md)
}

// SetTrailer implements the gRPC ServerStream interface
func (a *StreamAdapter) SetTrailer(md metadata.MD) {
	a.underlying.SetTrailer(md)
}

// Context implements the gRPC ServerStream interface
func (a *StreamAdapter) Context() context.Context {
	return a.underlying.Context()
}

// SendMsg implements the gRPC ServerStream interface
func (a *StreamAdapter) SendMsg(m interface{}) error {
	return a.underlying.SendMsg(m)
}

// RecvMsg implements the gRPC ServerStream interface
func (a *StreamAdapter) RecvMsg(m interface{}) error {
	return a.underlying.RecvMsg(m)
}

// SSEWriter is an interface for writing SSE events
type SSEWriter interface {
	io.Writer
	Flush()
}

// NewAGUISSEWrapper creates an SSE wrapper that writes AG-UI events
func NewAGUISSEWrapper(writer SSEWriter) *AGUISSEWrapper {
	return &AGUISSEWrapper{
		writer: writer,
	}
}

// AGUISSEWrapper wraps an HTTP writer to send AG-UI events via SSE
type AGUISSEWrapper struct {
	writer SSEWriter
}

// WriteEvent writes an AG-UI event as an SSE event
func (w *AGUISSEWrapper) WriteEvent(event *aguipb.AGUIEvent) error {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal AG-UI event: %w", err)
	}

	// Determine event type for SSE
	eventType := "agui_event"
	switch event.Type {
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_START:
		eventType = "message_start"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_DELTA:
		eventType = "message_delta"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_STOP:
		eventType = "message_stop"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_START:
		eventType = "content_block_start"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_DELTA:
		eventType = "content_block_delta"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_STOP:
		eventType = "content_block_stop"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_START:
		eventType = "tool_call_start"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_DELTA:
		eventType = "tool_call_delta"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_STOP:
		eventType = "tool_call_stop"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_START:
		eventType = "thinking_start"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_DELTA:
		eventType = "thinking_delta"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_STOP:
		eventType = "thinking_stop"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_START:
		eventType = "task_start"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_UPDATE:
		eventType = "task_update"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_COMPLETE:
		eventType = "task_complete"
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_ERROR:
		eventType = "task_error"
	}

	// Write SSE format: event: <type>\ndata: <json>\n\n
	_, err = fmt.Fprintf(w.writer, "event: %s\ndata: %s\n\n", eventType, string(data))
	if err != nil {
		return err
	}

	w.writer.Flush()
	return nil
}
