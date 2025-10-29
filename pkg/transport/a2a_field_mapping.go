package transport

import (
	"encoding/json"
)

func applyA2AFieldMapping(data []byte) []byte {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(data, &bodyMap); err != nil {
		return data
	}

	if message, ok := bodyMap["message"].(map[string]interface{}); ok && bodyMap["request"] == nil {
		bodyMap["request"] = message
		delete(bodyMap, "message")
	}

	if request, ok := bodyMap["request"].(map[string]interface{}); ok {
		// Convert legacy "content" field to official A2A "parts" field for backward compatibility
		if content, ok := request["content"]; ok {
			request["parts"] = content
			delete(request, "content")
		}

		if role, ok := request["role"].(string); ok {
			switch role {
			case "user":
				request["role"] = "ROLE_USER"
			case "agent", "assistant":
				request["role"] = "ROLE_AGENT"
			}
		}
	}

	if task, ok := bodyMap["task"].(map[string]interface{}); ok {
		if status, ok := task["status"].(map[string]interface{}); ok {
			if state, ok := status["state"].(string); ok {
				status["state"] = normalizeTaskState(state)
			}
		}
	}

	result, err := json.Marshal(bodyMap)
	if err != nil {
		return data
	}
	return result
}

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
