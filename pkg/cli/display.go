package cli

import (
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// DisplayAgentList displays a formatted list of agents
func DisplayAgentList(agents []*pb.AgentCard, mode string) {
	fmt.Printf("\n📋 Available Agents (%s)\n\n", mode)
	fmt.Printf("Found %d agent(s):\n\n", len(agents))

	for _, card := range agents {
		fmt.Printf("• %s", card.Name)
		if card.Version != "" {
			fmt.Printf(" (v%s)", card.Version)
		}
		fmt.Println()

		if card.Description != "" {
			fmt.Printf("  Description: %s\n", card.Description)
		}
		if card.Url != "" {
			fmt.Printf("  URL: %s\n", card.Url)
		}
		if card.Capabilities != nil && card.Capabilities.Streaming {
			fmt.Printf("  Streaming: ✓\n")
		}
		fmt.Println()
	}

	fmt.Println("💡 Use 'hector info <agent>' for detailed information")
	fmt.Println("💡 Use 'hector call \"prompt\" --agent <agent>' to interact with an agent")
}

// DisplayAgentCard displays a formatted agent card
func DisplayAgentCard(agentID string, card *pb.AgentCard) {
	fmt.Printf("\n📋 Agent Information: %s\n\n", agentID)
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

// DisplayMessage displays a message response
func DisplayMessage(msg *pb.Message, prefix string) {
	if msg == nil {
		return
	}

	if prefix != "" {
		fmt.Print(prefix)
	}

	for _, part := range msg.Content {
		if text := part.GetText(); text != "" {
			fmt.Print(text)
			// Flush stdout to ensure streaming output is visible immediately
			os.Stdout.Sync()
		}
	}
}

// DisplayMessageLine displays a message response with newline
func DisplayMessageLine(msg *pb.Message, prefix string) {
	DisplayMessage(msg, prefix)
	fmt.Println()
}

// DisplayTask displays a task response
func DisplayTask(task *pb.Task) {
	fmt.Printf("\n📋 Task Details\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Task ID:     %s\n", task.Id)
	fmt.Printf("Context ID:  %s\n", task.ContextId)

	if task.Status != nil {
		stateStr := task.Status.State.String()
		// Remove TASK_STATE_ prefix for cleaner display
		if len(stateStr) > 11 && stateStr[:11] == "TASK_STATE_" {
			stateStr = stateStr[11:]
		}

		// Color code based on state
		var stateDisplay string
		switch task.Status.State {
		case pb.TaskState_TASK_STATE_COMPLETED:
			stateDisplay = fmt.Sprintf("✅ %s", stateStr)
		case pb.TaskState_TASK_STATE_FAILED:
			stateDisplay = fmt.Sprintf("❌ %s", stateStr)
		case pb.TaskState_TASK_STATE_CANCELLED:
			stateDisplay = fmt.Sprintf("🚫 %s", stateStr)
		case pb.TaskState_TASK_STATE_WORKING:
			stateDisplay = fmt.Sprintf("⚙️  %s", stateStr)
		case pb.TaskState_TASK_STATE_SUBMITTED:
			stateDisplay = fmt.Sprintf("📤 %s", stateStr)
		default:
			stateDisplay = stateStr
		}

		fmt.Printf("Status:      %s\n", stateDisplay)

		if task.Status.Timestamp != nil {
			fmt.Printf("Updated:     %s\n", task.Status.Timestamp.AsTime().Format("2006-01-02 15:04:05"))
		}
	}

	// Display artifacts count
	if len(task.Artifacts) > 0 {
		fmt.Printf("Artifacts:   %d\n", len(task.Artifacts))
	}

	// Display history
	if len(task.History) > 0 {
		fmt.Printf("\n💬 History (%d messages):\n", len(task.History))
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

			// Display content
			if len(msg.Content) > 0 {
				for _, part := range msg.Content {
					if text := part.GetText(); text != "" {
						// Truncate long messages
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

// DisplayError displays an error message
func DisplayError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

// DisplayStreamingStart displays a streaming start message
func DisplayStreamingStart(agentID, mode string) {
	streamInfo := ""
	if mode != "" {
		streamInfo = fmt.Sprintf(" (%s)", mode)
	}
	fmt.Printf("\n🤖 Chat with %s%s (streaming) (type 'exit' to quit)\n\n", agentID, streamInfo)
}

// DisplayChatPrompt displays a chat input prompt
func DisplayChatPrompt() {
	fmt.Print("You: ")
}

// DisplayAgentPrompt displays an agent response prompt
func DisplayAgentPrompt(agentID string) {
	fmt.Printf("\n%s: ", agentID)
}

// DisplayGoodbye displays a goodbye message
func DisplayGoodbye() {
	fmt.Println("👋 Goodbye!")
}
