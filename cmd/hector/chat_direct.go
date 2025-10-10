package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/agent"
)

// ============================================================================
// DIRECT CHAT - INTERACTIVE CHAT WITHOUT SERVER
// ============================================================================

// startDirectChat starts an interactive chat session with a direct agent
func startDirectChat(ctx context.Context, agentInstance *agent.Agent, agentName string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n💬 Starting chat with %s (Direct Mode)\n", agentName)
	fmt.Println("Type your messages below. Commands:")
	fmt.Println("  /quit or /exit - End chat session")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println()

	for {
		// Show prompt
		fmt.Print("You: ")

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Handle empty input
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/quit", "/exit":
				fmt.Println("\n👋 Chat session ended")
				return nil
			case "/clear":
				fmt.Println("🧹 Conversation history cleared")
				// TODO: Implement history clearing if needed
				continue
			default:
				fmt.Printf("Unknown command: %s\n", input)
				continue
			}
		}

		// Execute agent call
		fmt.Printf("\n%s: ", agentName)

		// Create task from input
		task := &a2a.Task{
			Messages: []a2a.Message{
				{
					Role: a2a.MessageRoleUser,
					Parts: []a2a.Part{
						{
							Type: a2a.PartTypeText,
							Text: input,
						},
					},
				},
			},
		}

		// Execute with streaming
		streamCh, err := agentInstance.ExecuteTaskStreaming(ctx, task)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		// Stream response
		for event := range streamCh {
			if event.Message != nil {
				// Extract text from message parts
				for _, part := range event.Message.Parts {
					if part.Type == a2a.PartTypeText {
						fmt.Print(part.Text)
					}
				}
			}
			// Check for errors in status
			if event.Status != nil && event.Status.State == a2a.TaskStateFailed {
				fmt.Printf("\n❌ Error: task failed")
				break
			}
		}
		fmt.Println()
	}
}
