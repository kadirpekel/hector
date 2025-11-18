package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
	"google.golang.org/protobuf/types/known/structpb"
)

// currentThinkingBlockID tracks the active thinking block across multiple DisplayMessage calls
// This ensures the THINKING prefix is only shown once per thinking block, even when chunks arrive separately
var currentThinkingBlockID string

// thinkingPrefixPrinted tracks whether we've already printed the prefix for the current thinking block
// This is needed because the empty start marker sets currentThinkingBlockID, but we want to print
// the prefix on the first non-empty chunk
var thinkingPrefixPrinted bool

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

func DisplayMessage(msg *pb.Message, prefix string, showThinking bool) {
	if msg == nil {
		return
	}

	if prefix != "" {
		fmt.Print(prefix)
	}

	for _, part := range msg.Parts {
		// Check metadata FIRST before displaying text
		// This ensures thinking parts and tool parts are handled correctly
		if part.Metadata != nil {
			eventType := ""
			if et, ok := part.Metadata.Fields["event_type"]; ok {
				eventType = et.GetStringValue()
			}

			// Display thinking parts (AG-UI compliant) - only if showThinking is true
			if eventType == "thinking" {
				if showThinking {
					// Get block ID to track if this is a new thinking block
					blockID := ""
					if bid, ok := part.Metadata.Fields["block_id"]; ok {
						blockID = bid.GetStringValue()
					}
					if blockID == "" {
						// Generate a fallback ID if none provided
						blockID = "unknown"
					}

					// Get text content
					text := part.GetText()
					if text == "" {
						// Try to get text from data part
						if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
							if textField, ok := dataPart.Data.Fields["text"]; ok {
								text = textField.GetStringValue()
							}
						}
					}

					// Skip empty thinking parts (these are just markers for block start)
					// But track the block ID for the first non-empty chunk
					if text == "" {
						// Empty part - track the block ID but don't display anything
						if blockID != currentThinkingBlockID {
							// New block starting
							currentThinkingBlockID = blockID
							thinkingPrefixPrinted = false
						}
						continue
					}

					// This is a non-empty thinking chunk
					// Check if this is a new thinking block (different ID)
					if blockID != currentThinkingBlockID {
						// Close previous thinking block if any
						if currentThinkingBlockID != "" {
							fmt.Print("\033[0m") // Reset styling
						}
						// Start new thinking block
						currentThinkingBlockID = blockID
						thinkingPrefixPrinted = false
					}

					// Print prefix only if we haven't printed it for this block yet
					if !thinkingPrefixPrinted {
						fmt.Print("\033[90m\033[2mTHINKING: ")
						thinkingPrefixPrinted = true
					}

					// Display the thinking content (styling already applied)
					displayThinkingPart(part)
				}
				continue
			}

			// Display tool call parts (AG-UI format)
			if eventType == "tool_call" {
				// Reset thinking block state when transitioning to tool calls
				if currentThinkingBlockID != "" {
					fmt.Print("\033[0m") // Reset styling
					currentThinkingBlockID = ""
					thinkingPrefixPrinted = false
				}
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
						// Display tool call with better formatting
						fmt.Print("\033[36m") // Cyan color for tool calls
						fmt.Printf("TOOL: %s", toolName)
						fmt.Print("\033[0m ")
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
						fmt.Print("\033[31m✗\033[0m\n") // Red for errors
						// Extract and display error message
						if toolResult := protocol.ExtractToolResult(part); toolResult != nil {
							errorMsg := toolResult.Error
							if errorMsg == "" && toolResult.Content != "" {
								// If no error field but content exists, show content (might be error message)
								errorMsg = toolResult.Content
							}
							if errorMsg != "" {
								fmt.Printf("\033[31m   Error: %s\033[0m\n", errorMsg)
							}
						}
					} else {
						fmt.Print("\033[32mOK\033[0m\n") // Green for success
					}
					os.Stdout.Sync()
					continue
				}
			}
		}

		// Display regular text parts (only if not handled by metadata above)
		if text := part.GetText(); text != "" {
			// Reset thinking block state when transitioning to regular text
			if currentThinkingBlockID != "" {
				fmt.Print("\033[0m") // Reset styling
				currentThinkingBlockID = ""
				thinkingPrefixPrinted = false
			}
			fmt.Print(text)
			os.Stdout.Sync()
			continue
		}
	}

	// Don't reset thinking block state at end of message - thinking blocks can span multiple messages
	// Only reset styling if we're not in a thinking block (already handled above)
}

// displayThinkingPart renders thinking parts based on structured data
// Client decides rendering based on backend's structured data (AG-UI principle)
func displayThinkingPart(part *pb.Part) {
	// Get thinking type hint from metadata
	thinkingType := ""
	if tt, ok := part.Metadata.Fields["thinking_type"]; ok {
		thinkingType = tt.GetStringValue()
	}

	// Check for structured data
	dataPart := part.GetData()
	if dataPart != nil && dataPart.Data != nil {
		// Rich client: render structured data
		switch thinkingType {
		case "todo":
			displayTodosCLI(dataPart.Data)
			return
		case "goal":
			displayGoalCLI(dataPart.Data)
			return
		}
	}

	// Fallback: display text (for simple thinking parts or backwards compatibility)
	// Try to get text from data first, then from text field
	text := ""
	if dataPart != nil && dataPart.Data != nil {
		if textField, ok := dataPart.Data.Fields["text"]; ok {
			text = textField.GetStringValue()
		}
	}
	if text == "" {
		text = part.GetText()
	}

	if text != "" {
		// Display with dimmed styling (prefix and styling already applied if new block)
		// Don't reset styling here - let it continue for the entire thinking block
		fmt.Print(text)
		os.Stdout.Sync()
	}
}

// displayTodosCLI renders todo list from structured data
func displayTodosCLI(data *structpb.Struct) {
	todosField, ok := data.Fields["todos"]
	if !ok {
		return
	}

	todosList := todosField.GetListValue()
	if todosList == nil {
		return
	}

	fmt.Print("\033[90m\033[2mTHINKING: ")
	fmt.Println("TASKS: Current Tasks:")

	for i, todoValue := range todosList.Values {
		todoStruct := todoValue.GetStructValue()
		if todoStruct == nil {
			continue
		}

		content := ""
		if c, ok := todoStruct.Fields["content"]; ok {
			content = c.GetStringValue()
		}

		status := ""
		if s, ok := todoStruct.Fields["status"]; ok {
			status = s.GetStringValue()
		}

		var checkbox, color string
		switch status {
		case "completed":
			checkbox = "☑"
			color = "\033[32m" // green
		case "in_progress":
			checkbox = "⧗"
			color = "\033[33m" // yellow
		case "pending":
			checkbox = "☐"
			color = "\033[37m" // white
		case "canceled":
			checkbox = "☒"
			color = "\033[31m" // red
		default:
			checkbox = "☐"
			color = "\033[37m"
		}

		fmt.Printf("  %s%s%d. %s\033[0m\033[90m\033[2m\n", color, checkbox, i+1, content)
	}
	fmt.Print("\033[0m")
}

// displayGoalCLI renders goal decomposition from structured data
func displayGoalCLI(data *structpb.Struct) {
	fmt.Print("\033[90m\033[2mTHINKING: ")

	// Display main goal
	if mainGoal, ok := data.Fields["main_goal"]; ok {
		fmt.Printf("GOAL: %s\n", mainGoal.GetStringValue())
	}

	// Display strategy
	if strategy, ok := data.Fields["strategy"]; ok {
		fmt.Printf("STRATEGY: %s\n", strategy.GetStringValue())
	}

	// Display subtasks if present
	if subtasksField, ok := data.Fields["subtasks"]; ok {
		subtasksList := subtasksField.GetListValue()
		if subtasksList != nil && len(subtasksList.Values) > 0 {
			fmt.Println("SUBTASKS:")

			for i, subtaskValue := range subtasksList.Values {
				subtaskStruct := subtaskValue.GetStructValue()
				if subtaskStruct == nil {
					continue
				}

				desc := ""
				if d, ok := subtaskStruct.Fields["description"]; ok {
					desc = d.GetStringValue()
				}

				priority := int64(0)
				if p, ok := subtaskStruct.Fields["priority"]; ok {
					priority = int64(p.GetNumberValue())
				}

				agentType := ""
				if a, ok := subtaskStruct.Fields["agent_type"]; ok {
					agentType = a.GetStringValue()
				}

				fmt.Printf("  %d. [P%d] %s → %s\n", i+1, priority, desc, agentType)
			}
		}
	}

	fmt.Print("\033[0m")
}

func DisplayMessageLine(msg *pb.Message, prefix string, showThinking bool) {
	DisplayMessage(msg, prefix, showThinking)
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
	fmt.Printf("ERROR: %v\n", err)
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
