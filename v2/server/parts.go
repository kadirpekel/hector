// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"

	"github.com/a2aproject/a2a-go/a2a"

	"github.com/kadirpekel/hector/v2/agent"
)

// toHectorContent converts an A2A message to Hector content.
func toHectorContent(msg *a2a.Message) (*agent.Content, error) {
	if msg == nil {
		return nil, nil
	}

	content := &agent.Content{
		Parts: msg.Parts,
		Role:  toHectorRole(msg.Role),
	}

	return content, nil
}

// toHectorRole converts A2A message role to Hector role.
func toHectorRole(role a2a.MessageRole) a2a.MessageRole {
	// A2A roles map directly
	return role
}

// toA2AMessage converts Hector content to an A2A message.
func toA2AMessage(content *agent.Content) *a2a.Message {
	if content == nil {
		return nil
	}
	return a2a.NewMessage(content.Role, content.Parts...)
}

// toA2AParts converts Hector event to A2A parts.
func toA2AParts(event *agent.Event) ([]a2a.Part, error) {
	if event == nil || event.Message == nil {
		return nil, nil
	}
	return event.Message.Parts, nil
}

// toHectorEvent converts an A2A event to a Hector event.
func toHectorEvent(ctx agent.InvocationContext, a2aEvent a2a.Event) (*agent.Event, error) {
	switch v := a2aEvent.(type) {
	case *a2a.Task:
		return taskToEvent(ctx, v)

	case *a2a.Message:
		return messageToEvent(ctx, v)

	case *a2a.TaskArtifactUpdateEvent:
		if len(v.Artifact.Parts) == 0 {
			return nil, nil
		}
		return artifactUpdateToEvent(ctx, v)

	case *a2a.TaskStatusUpdateEvent:
		if v.Final {
			return statusUpdateToEvent(ctx, v)
		}
		if v.Status.Message == nil {
			return nil, nil
		}
		return statusMessageToEvent(ctx, v)

	default:
		return nil, fmt.Errorf("unknown A2A event type: %T", v)
	}
}

func taskToEvent(ctx agent.InvocationContext, task *a2a.Task) (*agent.Event, error) {
	event := agent.NewEvent(ctx.InvocationID())
	event.Author = ctx.Agent().Name()
	event.Branch = ctx.Branch()

	// Collect parts from artifacts and status message
	var parts []a2a.Part
	for _, artifact := range task.Artifacts {
		parts = append(parts, artifact.Parts...)
	}
	if task.Status.Message != nil {
		parts = append(parts, task.Status.Message.Parts...)
	}

	if len(parts) > 0 {
		event.Message = a2a.NewMessage(a2a.MessageRoleAgent, parts...)
	}

	event.CustomMetadata = map[string]any{
		metaKeyTaskID:    string(task.ID),
		metaKeyContextID: task.ContextID,
	}

	// Check for input required state
	if task.Status.State == a2a.TaskStateInputRequired {
		// Mark long-running tools from artifact metadata
		event.LongRunningToolIDs = extractLongRunningToolIDs(task)
	}

	event.Partial = !task.Status.State.Terminal() && task.Status.State != a2a.TaskStateInputRequired
	event.TurnComplete = task.Status.State.Terminal()

	return event, nil
}

func messageToEvent(ctx agent.InvocationContext, msg *a2a.Message) (*agent.Event, error) {
	event := agent.NewEvent(ctx.InvocationID())
	event.Author = ctx.Agent().Name()
	event.Branch = ctx.Branch()
	event.Message = msg

	if msg.TaskID != "" || msg.ContextID != "" {
		event.CustomMetadata = map[string]any{
			metaKeyTaskID:    string(msg.TaskID),
			metaKeyContextID: msg.ContextID,
		}
	}

	event.Actions = extractEventActions(msg.Metadata)

	return event, nil
}

func artifactUpdateToEvent(ctx agent.InvocationContext, update *a2a.TaskArtifactUpdateEvent) (*agent.Event, error) {
	event := agent.NewEvent(ctx.InvocationID())
	event.Author = ctx.Agent().Name()
	event.Branch = ctx.Branch()

	if len(update.Artifact.Parts) > 0 {
		event.Message = a2a.NewMessage(a2a.MessageRoleAgent, update.Artifact.Parts...)
	}

	event.CustomMetadata = map[string]any{
		metaKeyTaskID:    string(update.TaskID),
		metaKeyContextID: update.ContextID,
	}
	event.Partial = !update.LastChunk

	return event, nil
}

func statusUpdateToEvent(ctx agent.InvocationContext, update *a2a.TaskStatusUpdateEvent) (*agent.Event, error) {
	event := agent.NewEvent(ctx.InvocationID())
	event.Author = ctx.Agent().Name()
	event.Branch = ctx.Branch()

	if update.Status.Message != nil {
		event.Message = update.Status.Message
	}

	event.CustomMetadata = map[string]any{
		metaKeyTaskID:    string(update.TaskID),
		metaKeyContextID: update.ContextID,
	}
	event.Actions = extractEventActions(update.Metadata)
	event.TurnComplete = true

	return event, nil
}

func statusMessageToEvent(ctx agent.InvocationContext, update *a2a.TaskStatusUpdateEvent) (*agent.Event, error) {
	event := agent.NewEvent(ctx.InvocationID())
	event.Author = ctx.Agent().Name()
	event.Branch = ctx.Branch()

	if update.Status.Message != nil {
		event.Message = update.Status.Message
		// Mark as thought content for intermediate status messages
		// (similar to ADK's handling of working state messages)
	}

	event.CustomMetadata = map[string]any{
		metaKeyTaskID:    string(update.TaskID),
		metaKeyContextID: update.ContextID,
	}
	event.Partial = true

	return event, nil
}

func extractEventActions(meta map[string]any) agent.EventActions {
	var actions agent.EventActions
	if meta == nil {
		return actions
	}

	if v, ok := meta[metaKeyEscalate].(bool); ok {
		actions.Escalate = v
	}
	if v, ok := meta[metaKeyTransfer].(string); ok {
		actions.TransferToAgent = v
	}

	return actions
}

func extractLongRunningToolIDs(task *a2a.Task) []string {
	// Extract long-running tool IDs from task artifacts/metadata
	// This is implementation-specific based on how tools mark themselves
	var ids []string
	for _, artifact := range task.Artifacts {
		for _, part := range artifact.Parts {
			if dp, ok := part.(a2a.DataPart); ok {
				if meta := dp.Metadata; meta != nil {
					if longRunning, ok := meta["long_running"].(bool); ok && longRunning {
						if id, ok := meta["tool_call_id"].(string); ok {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}
	return ids
}

// ApprovalResponse represents an approval decision from the user.
type ApprovalResponse struct {
	// Decision is "approve" or "deny"
	Decision string
	// ToolCallID is the ID of the tool call being approved/denied
	ToolCallID string
	// TaskID is the task this approval is for
	TaskID string
}

// ExtractApprovalResponse checks if a message contains an approval response.
// Returns nil if the message is not an approval response.
//
// Approval responses can be:
// 1. A DataPart with type: "tool_approval"
// 2. A TextPart with "approve" or "deny" (for simple approvals)
func ExtractApprovalResponse(msg *a2a.Message) *ApprovalResponse {
	if msg == nil || len(msg.Parts) == 0 {
		return nil
	}

	for _, part := range msg.Parts {
		// Check for structured approval (DataPart)
		if dp, ok := part.(a2a.DataPart); ok {
			if partType, ok := dp.Data["type"].(string); ok && partType == "tool_approval" {
				decision, _ := dp.Data["decision"].(string)
				toolCallID, _ := dp.Data["tool_call_id"].(string)
				taskID, _ := dp.Data["task_id"].(string)
				if decision != "" {
					return &ApprovalResponse{
						Decision:   decision,
						ToolCallID: toolCallID,
						TaskID:     taskID,
					}
				}
			}
		}

		// Check for simple text approval
		if tp, ok := part.(a2a.TextPart); ok {
			text := tp.Text
			if text == "approve" || text == "approved" {
				return &ApprovalResponse{Decision: "approve"}
			}
			if text == "deny" || text == "denied" || text == "reject" {
				return &ApprovalResponse{Decision: "deny"}
			}
		}
	}

	return nil
}
