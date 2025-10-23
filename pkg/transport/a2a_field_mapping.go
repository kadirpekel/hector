package transport

import (
	"encoding/json"
)

// applyA2AFieldMapping transforms A2A JSON fields to protobuf format.
// This handles the mapping between A2A specification's camelCase field names
// and protobuf's internal representation:
//   - parts → content
//   - lowercase roles (user/agent) → uppercase enum format (ROLE_USER/ROLE_AGENT)
//   - lowercase states (submitted/working/etc) → uppercase enum format (TASK_STATE_*)
func applyA2AFieldMapping(data []byte) []byte {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(data, &bodyMap); err != nil {
		return data // Return unchanged if parsing fails
	}

	// Look for message object and apply transformations
	if message, ok := bodyMap["message"].(map[string]interface{}); ok {
		// Map "parts" → "content" for A2A compatibility
		if parts, ok := message["parts"]; ok {
			message["content"] = parts
			delete(message, "parts")
		}

		// Map lowercase "role" values to protobuf enum format
		if role, ok := message["role"].(string); ok {
			switch role {
			case "user":
				message["role"] = "ROLE_USER"
			case "agent", "assistant":
				message["role"] = "ROLE_AGENT"
			}
		}
	}

	// Look for task object and apply state transformations
	if task, ok := bodyMap["task"].(map[string]interface{}); ok {
		if status, ok := task["status"].(map[string]interface{}); ok {
			if state, ok := status["state"].(string); ok {
				status["state"] = normalizeTaskState(state)
			}
		}
	}

	// Re-marshal
	result, err := json.Marshal(bodyMap)
	if err != nil {
		return data // Return original if marshaling fails
	}
	return result
}

// normalizeTaskState converts lowercase task states to uppercase enum format
func normalizeTaskState(state string) string {
	switch state {
	case "submitted":
		return "TASK_STATE_SUBMITTED"
	case "working":
		return "TASK_STATE_WORKING"
	case "completed":
		return "TASK_STATE_COMPLETED"
	case "failed":
		return "TASK_STATE_FAILED"
	case "cancelled", "canceled":
		return "TASK_STATE_CANCELLED"
	case "input-required":
		return "TASK_STATE_INPUT_REQUIRED"
	case "auth-required":
		return "TASK_STATE_AUTH_REQUIRED"
	case "rejected":
		return "TASK_STATE_REJECTED"
	default:
		return state
	}
}
