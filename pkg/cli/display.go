package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
	"google.golang.org/protobuf/types/known/structpb"
)

// ANSI color codes for CLI output
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorDim     = "\033[90m"
	colorDimBold = "\033[90m\033[2m"
)

// Maximum size for JSON pretty-printing (10KB)
const maxPrettyPrintSize = 10000

// currentThinkingBlockID tracks the active thinking block across multiple DisplayMessage calls
// This ensures the THINKING prefix is only shown once per thinking block, even when chunks arrive separately
var currentThinkingBlockID string

// thinkingPrefixPrinted tracks whether we've already printed the prefix for the current thinking block
// This is needed because the empty start marker sets currentThinkingBlockID, but we want to print
// the prefix on the first non-empty chunk
var thinkingPrefixPrinted bool

// displayedToolCallIDs tracks tool call IDs that have already been displayed
// This prevents double-printing when tool calls appear both in streamed parts and final message parts
var displayedToolCallIDs = make(map[string]bool)

// lastOutputType tracks the type of the last displayed output for proper block separation
// Values: "thinking", "tool", "text", or "" (none)
var lastOutputType string

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

// ResetDisplayState resets the display state tracking variables
// Call this when starting a new conversation or message sequence
func ResetDisplayState() {
	currentThinkingBlockID = ""
	thinkingPrefixPrinted = false
	displayedToolCallIDs = make(map[string]bool)
	lastOutputType = ""
}

func DisplayMessage(msg *pb.Message, prefix string, showThinking bool, showTools bool) bool {
	if msg == nil {
		return false
	}

	hasOutput := false

	if prefix != "" {
		fmt.Print(prefix)
		hasOutput = true
	}

	// First, check if this is an approval message by looking for approval DataPart
	isApprovalMessage := false
	for _, part := range msg.Parts {
		if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
			if interactionType, ok := dataPart.Data.Fields["interaction_type"]; ok {
				it := interactionType.GetStringValue()
				if it == "tool_approval" || it == "approval" {
					isApprovalMessage = true
					break
				}
			}
		}
	}

	for _, part := range msg.Parts {
		// If this is an approval message, handle TextPart specially
		if isApprovalMessage {
			if text := part.GetText(); text != "" {
				// This is the TextPart of an approval message
				// Reset thinking block state when transitioning to approval
				if currentThinkingBlockID != "" {
					fmt.Print(colorReset + "\n") // Reset styling and add newline
					currentThinkingBlockID = ""
					thinkingPrefixPrinted = false
				}

				// Add newline if transitioning from a different block type
				if lastOutputType != "" && lastOutputType != "approval" {
					fmt.Print("\n")
				}

				// Display the approval prompt text
				fmt.Print(text)
				os.Stdout.Sync() // Ensure text is flushed
				lastOutputType = "approval"
				hasOutput = true
				continue
			}

			// Skip DataPart of approval message (we already detected it's an approval)
			if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
				if interactionType, ok := dataPart.Data.Fields["interaction_type"]; ok {
					it := interactionType.GetStringValue()
					if it == "tool_approval" || it == "approval" {
						// Already handled above, just mark as approval type
						lastOutputType = "approval"
						hasOutput = true
						continue
					}
				}
			}
		}

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
					// Add newline if transitioning from a different block type
					if lastOutputType != "" && lastOutputType != "thinking" {
						fmt.Print("\n")
					}

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
							fmt.Print(colorReset + "\n") // Reset styling and add newline
						}
						// Start new thinking block
						currentThinkingBlockID = blockID
						thinkingPrefixPrinted = false
					}

					// Print prefix only if we haven't printed it for this block yet
					if !thinkingPrefixPrinted {
						fmt.Print(colorDimBold + "THINKING: ")
						thinkingPrefixPrinted = true
					}

					// Display the thinking content (styling already applied)
					displayThinkingPart(part)
					lastOutputType = "thinking"
					hasOutput = true
				}
				continue
			}

			// Display approval request parts
			if eventType == "approval" || eventType == "tool_approval" {
				// Reset thinking block state when transitioning to approval
				if currentThinkingBlockID != "" {
					fmt.Print(colorReset + "\n") // Reset styling and add newline
					currentThinkingBlockID = ""
					thinkingPrefixPrinted = false
				}

				// Add newline if transitioning from a different block type
				if lastOutputType != "" && lastOutputType != "approval" {
					fmt.Print("\n")
				}

				// Display text content if this part has text (approval messages have text in TextPart)
				if text := part.GetText(); text != "" {
					fmt.Print(text)
					os.Stdout.Sync() // Ensure text is flushed before prompt
				}

				// Mark this as approval type
				lastOutputType = "approval"
				hasOutput = true
				continue
			}

			// Display tool call parts (AG-UI format) - only if showTools is true
			if eventType == "tool_call" {
				// Reset thinking block state when transitioning to tool calls
				if currentThinkingBlockID != "" {
					fmt.Print(colorReset + "\n") // Reset styling and add newline
					currentThinkingBlockID = ""
					thinkingPrefixPrinted = false
				}

				// Skip tool calls/results if showTools is false (clean output mode)
				if !showTools {
					continue
				}

				// Check if it's a tool call (no is_error) or tool result (has is_error)
				_, hasIsError := part.Metadata.Fields["is_error"]

				if !hasIsError {
					// This is a tool call - add newline if transitioning from a different block type
					if lastOutputType != "" && lastOutputType != "tool" {
						fmt.Print("\n")
					}
					// This is a tool call
					// Extract tool call ID to track duplicates
					toolCallID := ""
					if id, ok := part.Metadata.Fields["tool_call_id"]; ok {
						toolCallID = id.GetStringValue()
					} else if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
						if id, ok := dataPart.Data.Fields["id"]; ok {
							toolCallID = id.GetStringValue()
						}
					}

					// Skip if we've already displayed this tool call (prevents duplication)
					if toolCallID != "" && displayedToolCallIDs[toolCallID] {
						continue
					}

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
						// Mark as displayed
						if toolCallID != "" {
							displayedToolCallIDs[toolCallID] = true
						}
						// Display tool call with better formatting
						fmt.Print(colorCyan) // Cyan color for tool calls
						fmt.Printf("TOOL: %s", toolName)
						fmt.Print(colorReset)
						// Set lastOutputType to "tool" so result stays on same line
						lastOutputType = "tool"
						os.Stdout.Sync()
						hasOutput = true
					}
					continue
				} else {
					// This is a tool result
					// Extract tool call ID to match with the call
					toolCallID := ""
					if id, ok := part.Metadata.Fields["tool_call_id"]; ok {
						toolCallID = id.GetStringValue()
					} else if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
						if id, ok := dataPart.Data.Fields["tool_call_id"]; ok {
							toolCallID = id.GetStringValue()
						}
					}

					// Skip if we've already displayed this result (prevents duplication)
					if toolCallID != "" && displayedToolCallIDs[toolCallID+"_result"] {
						continue
					}

					isError := false
					if isErrorField, ok := part.Metadata.Fields["is_error"]; ok {
						isError = isErrorField.GetBoolValue()
					}

					// Mark result as displayed
					if toolCallID != "" {
						displayedToolCallIDs[toolCallID+"_result"] = true
					}

					// Don't add newline if we're continuing a tool call (lastOutputType is already "tool")
					// The tool call and result should be on the same line
					if isError {
						fmt.Print(" " + colorRed + "✗" + colorReset + "\n") // Red for errors, with space before
						// Extract and display error message
						if toolResult := protocol.ExtractToolResult(part); toolResult != nil {
							errorMsg := toolResult.Error
							if errorMsg == "" && toolResult.Content != "" {
								// If no error field but content exists, show content (might be error message)
								errorMsg = toolResult.Content
							}
							if errorMsg != "" {
								fmt.Printf(colorRed+"   Error: %s"+colorReset+"\n", errorMsg)
							}
						}
						lastOutputType = "tool"
						hasOutput = true
					} else {
						// Display tool result status
						fmt.Print(" " + colorGreen + "OK" + colorReset) // Green for success, with space before

						// If showTools is true, also display the full result content
						if showTools {
							if toolResult := protocol.ExtractToolResult(part); toolResult != nil {
								if toolResult.Content != "" {
									// Display result content in a formatted way
									fmt.Print("\n")
									fmt.Print(colorDim) // Dim color for result content
									fmt.Print("   Result: ")
									fmt.Print(colorReset)
									// Try to format as JSON if possible, otherwise display as-is
									content := toolResult.Content
									if len(content) > 0 && len(content) < maxPrettyPrintSize && (content[0] == '{' || content[0] == '[') {
										// Try to pretty-print JSON (only for small content)
										var jsonData interface{}
										if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
											if jsonBytes, err := json.MarshalIndent(jsonData, "   ", "  "); err == nil {
												content = string(jsonBytes)
											}
										}
									}
									// Display content with indentation
									lines := strings.Split(content, "\n")
									for _, line := range lines {
										fmt.Printf("   %s\n", line)
									}
									fmt.Print(colorReset)
								}
							}
						} else {
							fmt.Print("\n") // Just OK status, no content
						}
						lastOutputType = "tool"
						hasOutput = true
					}
					os.Stdout.Sync()
					continue
				}
			}
		}

		// Display regular text parts (only if not handled by metadata above)
		// For approval messages, display the text part normally (it will be followed by DataPart)
		if text := part.GetText(); text != "" {

			// Reset thinking block state when transitioning to regular text
			if currentThinkingBlockID != "" {
				fmt.Print(colorReset + "\n") // Reset styling and add newline
				currentThinkingBlockID = ""
				thinkingPrefixPrinted = false
			}

			// Add newline if transitioning from a different block type
			if lastOutputType != "" && lastOutputType != "text" {
				fmt.Print("\n")
			}

			fmt.Print(text)
			lastOutputType = "text"
			os.Stdout.Sync()
			hasOutput = true
			continue
		}
	}

	// Don't reset thinking block state at end of message - thinking blocks can span multiple messages
	// Only reset styling if we're not in a thinking block (already handled above)
	return hasOutput
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

	fmt.Print(colorDimBold + "THINKING: ")
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
			color = colorGreen // green
		case "in_progress":
			checkbox = "⧗"
			color = colorYellow // yellow
		case "pending":
			checkbox = "☐"
			color = colorWhite // white
		case "canceled":
			checkbox = "☒"
			color = colorRed // red
		default:
			checkbox = "☐"
			color = colorWhite
		}

		fmt.Printf("  %s%s%d. %s%s%s\n", color, checkbox, i+1, content, colorReset, colorDimBold)
	}
	fmt.Print(colorReset)
}

// displayGoalCLI renders goal decomposition from structured data
func displayGoalCLI(data *structpb.Struct) {
	fmt.Print(colorDimBold + "THINKING: ")

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

	fmt.Print(colorReset)
}

func DisplayMessageLine(msg *pb.Message, prefix string, showThinking bool, showTools bool) {
	if DisplayMessage(msg, prefix, showThinking, showTools) {
		fmt.Println()
	}
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
	if err != nil {
		slog.Error(err.Error())
	}
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

// PromptForApproval prompts the user for approval and returns their decision
// Returns "approve" or "deny"
func PromptForApproval() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(colorYellow + "[APPROVAL] " + colorReset)
		fmt.Print("Approve or deny? (approve/deny/a/d): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "deny" // Safe default on error
		}

		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "approve", "a":
			return "approve"
		case "deny", "d":
			return "deny"
		default:
			fmt.Println("Please enter 'approve' or 'deny' (or 'a'/'d')")
			continue
		}
	}
}

// IsApprovalRequest checks if a message is an approval request
func IsApprovalRequest(msg *pb.Message) bool {
	if msg == nil {
		return false
	}

	for _, part := range msg.Parts {
		if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
			fields := dataPart.Data.Fields
			if interactionType, ok := fields["interaction_type"]; ok {
				it := interactionType.GetStringValue()
				if it == "tool_approval" || it == "approval" {
					return true
				}
			}
		}
	}
	return false
}

// CreateApprovalResponse creates a message with approval decision
func CreateApprovalResponse(contextID, taskID, decision string) *pb.Message {
	// Validate that IDs are not empty
	if contextID == "" || taskID == "" {
		slog.Warn("Creating approval response with empty IDs", "contextID", contextID, "taskID", taskID)
	}

	// Create structured data part
	decisionValue := structpb.NewStringValue(decision)
	fields := map[string]*structpb.Value{
		"decision": decisionValue,
	}

	return &pb.Message{
		ContextId: contextID,
		TaskId:    taskID,
		Role:      pb.Role_ROLE_USER,
		Parts: []*pb.Part{
			// Text part with decision
			{
				Part: &pb.Part_Text{
					Text: decision,
				},
			},
			// Data part with structured decision
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: fields,
						},
					},
				},
			},
		},
	}
}
