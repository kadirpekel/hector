package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

func DisplayAgentList(agents []*pb.AgentCard, mode string) {
	fmt.Printf("\nAvailable Agents (%s)\n\n", mode)
	fmt.Printf("Found %d agent(s):\n\n", len(agents))

	for _, card := range agents {
		// Extract agent ID from URL
		agentID := extractAgentIDFromURL(card.Url)

		fmt.Printf("• %s", card.Name)
		if card.Version != "" {
			fmt.Printf(" (v%s)", card.Version)
		}
		fmt.Println()

		if agentID != "" && agentID != card.Name {
			fmt.Printf("  ID: %s\n", agentID)
		}
		if card.Description != "" {
			fmt.Printf("  Description: %s\n", card.Description)
		}
		if card.Url != "" {
			fmt.Printf("  URL: %s\n", card.Url)
		}
		if card.Capabilities != nil && card.Capabilities.Streaming {
			fmt.Printf("  Streaming: yes\n")
		}
		fmt.Println()
	}

	fmt.Println("Tip: Use 'hector info <agent_id>' for detailed information")
	fmt.Println("Tip: Use 'hector call \"prompt\" --agent <agent_id>' to interact with an agent")
}

// extractAgentIDFromURL extracts the agent ID from the URL query parameter
func extractAgentIDFromURL(url string) string {
	if url == "" {
		return ""
	}
	// Parse URL to extract ?agent=<id>
	parts := strings.Split(url, "?")
	if len(parts) < 2 {
		return ""
	}
	params := strings.Split(parts[1], "&")
	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) == 2 && kv[0] == "agent" {
			return kv[1]
		}
	}
	return ""
}

func DisplayAgentCard(agentID string, card *pb.AgentCard) {
	fmt.Printf("\nAgent Information: %s\n\n", agentID)
	fmt.Printf("Name: %s\n", card.Name)
	if card.Description != "" {
		fmt.Printf("Description: %s\n", card.Description)
	}
	if card.Version != "" {
		fmt.Printf("Version: %s\n", card.Version)
	}
	if card.Capabilities != nil {
		fmt.Printf("Streaming: %v\n", card.Capabilities.Streaming)
	}
}

func DisplayMessage(msg *pb.Message, prefix string) {
	if msg == nil {
		return
	}

	if prefix != "" {
		fmt.Print(prefix)
	}

	for _, part := range msg.Parts {
		// Display text parts
		if text := part.GetText(); text != "" {
			fmt.Print(text)
			os.Stdout.Sync()
			continue
		}

		// Display tool call parts (AGUI format)
		if part.Metadata != nil {
			// Check for AGUI tool call (event_type = "tool_call" without is_error)
			if eventType, ok := part.Metadata.Fields["event_type"]; ok && eventType.GetStringValue() == "tool_call" {
				// Check if it's a tool call (no is_error) or tool result (has is_error)
				_, hasIsError := part.Metadata.Fields["is_error"]

				if !hasIsError {
					// This is a tool call
					// Extract tool name from AGUI metadata or data
					toolName := ""
					if name, ok := part.Metadata.Fields["tool_name"]; ok {
						toolName = name.GetStringValue()
					} else if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
						if name, ok := dataPart.Data.Fields["name"]; ok {
							toolName = name.GetStringValue()
						}
					}
					if toolName != "" {
						fmt.Printf("🔧 %s ", toolName)
						os.Stdout.Sync()
					}
					continue
				} else {
					// This is a tool result
					isError := false
					if isErrorField, ok := part.Metadata.Fields["is_error"]; ok {
						isError = isErrorField.GetBoolValue()
					}
					if isError {
						fmt.Print("✗\n")
					} else {
						fmt.Print("✓\n")
					}
					os.Stdout.Sync()
					continue
				}
			}
		}
	}
}

func DisplayMessageLine(msg *pb.Message, prefix string) {
	DisplayMessage(msg, prefix)
	fmt.Println()
}

func DisplayTask(task *pb.Task) {
	fmt.Printf("\nTask Details\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Task ID:     %s\n", task.Id)
	fmt.Printf("Context ID:  %s\n", task.ContextId)

	if task.Status != nil {
		stateStr := task.Status.State.String()

		if len(stateStr) > 11 && stateStr[:11] == "TASK_STATE_" {
			stateStr = stateStr[11:]
		}

		var stateDisplay string
		switch task.Status.State {
		case pb.TaskState_TASK_STATE_COMPLETED:
			stateDisplay = fmt.Sprintf("[SUCCESS] %s", stateStr)
		case pb.TaskState_TASK_STATE_FAILED:
			stateDisplay = fmt.Sprintf("[FAILED] %s", stateStr)
		case pb.TaskState_TASK_STATE_CANCELLED:
			stateDisplay = fmt.Sprintf("[CANCELLED] %s", stateStr)
		case pb.TaskState_TASK_STATE_WORKING:
			stateDisplay = fmt.Sprintf("[IN PROGRESS] %s", stateStr)
		case pb.TaskState_TASK_STATE_SUBMITTED:
			stateDisplay = fmt.Sprintf("[SUBMITTED] %s", stateStr)
		default:
			stateDisplay = stateStr
		}

		fmt.Printf("Status:      %s\n", stateDisplay)

		if task.Status.Timestamp != nil {
			fmt.Printf("Updated:     %s\n", task.Status.Timestamp.AsTime().Format("2006-01-02 15:04:05"))
		}
	}

	if len(task.Artifacts) > 0 {
		fmt.Printf("Artifacts:   %d\n", len(task.Artifacts))
	}

	if len(task.History) > 0 {
		fmt.Printf("\nHistory (%d messages):\n", len(task.History))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, msg := range task.History {
			roleStr := "Unknown"
			switch msg.Role {
			case pb.Role_ROLE_USER:
				roleStr = "User"
			case pb.Role_ROLE_AGENT:
				roleStr = "Agent"
			}

			fmt.Printf("%d. [%s] ", i+1, roleStr)

			if len(msg.Parts) > 0 {
				for _, part := range msg.Parts {
					if text := part.GetText(); text != "" {

						if len(text) > 200 {
							fmt.Printf("%s...\n", text[:200])
						} else {
							fmt.Printf("%s\n", text)
						}
					}
				}
			}
		}
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func DisplayError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

func DisplayStreamingStart(agentID, mode string) {
	streamInfo := ""
	if mode != "" {
		streamInfo = fmt.Sprintf(" (%s)", mode)
	}
	fmt.Printf("\nChat with %s%s (streaming) (type 'exit' to quit)\n\n", agentID, streamInfo)
}

func DisplayChatPrompt() {
	fmt.Print("You: ")
}

func DisplayAgentPrompt(agentID string) {
	fmt.Printf("\n%s: ", agentID)
}

func DisplayGoodbye() {
	fmt.Println("Goodbye!")
}
