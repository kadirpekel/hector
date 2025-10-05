package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kadirpekel/hector/a2a"
)

// ============================================================================
// CLIENT COMMANDS - All commands talk via A2A protocol
// ============================================================================

// executeListCommand lists available agents from an A2A server
func executeListCommand(serverURL string, token string) error {
	client := createA2AClient(token, "")

	// Ensure proper agents endpoint
	agentsURL := serverURL
	if !strings.HasSuffix(agentsURL, "/agents") {
		agentsURL = strings.TrimSuffix(agentsURL, "/") + "/agents"
	}

	agents, err := client.ListAgents(context.Background(), agentsURL)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents) == 0 {
		fmt.Println("No agents available")
		return nil
	}

	fmt.Printf("\n📋 Available agents at %s:\n\n", serverURL)
	for _, agent := range agents {
		fmt.Printf("  🤖 %s\n", agent.Name)
		fmt.Printf("     ID: %s\n", agent.AgentID)
		if agent.Description != "" {
			fmt.Printf("     %s\n", agent.Description)
		}
		fmt.Printf("     Capabilities: %s\n", strings.Join(agent.Capabilities, ", "))
		fmt.Printf("     Endpoint: %s\n", agent.Endpoints.Task)
		fmt.Println()
	}

	return nil
}

// executeInfoCommand gets detailed information about an agent
func executeInfoCommand(agentURL string, token string) error {
	client := createA2AClient(token, "")

	card, err := client.DiscoverAgent(context.Background(), agentURL)
	if err != nil {
		return fmt.Errorf("failed to get agent info: %w", err)
	}

	fmt.Printf("\n🤖 Agent: %s\n", card.Name)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	fmt.Printf("ID:          %s\n", card.AgentID)
	fmt.Printf("Description: %s\n", card.Description)
	fmt.Printf("Version:     %s\n", card.Version)
	fmt.Println()

	fmt.Println("Capabilities:")
	for _, cap := range card.Capabilities {
		fmt.Printf("  • %s\n", cap)
	}
	fmt.Println()

	fmt.Println("Endpoints:")
	fmt.Printf("  Task:   %s\n", card.Endpoints.Task)
	if card.Endpoints.Stream != "" {
		fmt.Printf("  Stream: %s\n", card.Endpoints.Stream)
	}
	if card.Endpoints.Status != "" {
		fmt.Printf("  Status: %s\n", card.Endpoints.Status)
	}
	fmt.Println()

	fmt.Println("Input Types:")
	for _, t := range card.InputTypes {
		fmt.Printf("  • %s\n", t)
	}
	fmt.Println()

	fmt.Println("Output Types:")
	for _, t := range card.OutputTypes {
		fmt.Printf("  • %s\n", t)
	}
	fmt.Println()

	if card.Auth.Type != "" {
		fmt.Printf("Authentication: %s\n", card.Auth.Type)
	}

	if len(card.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		for k, v := range card.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

// executeCallCommand executes a one-shot task on an agent
func executeCallCommand(agentURL string, input string, token string, stream bool) error {
	client := createA2AClient(token, "")

	// Get agent card
	card, err := client.DiscoverAgent(context.Background(), agentURL)
	if err != nil {
		return fmt.Errorf("failed to discover agent: %w", err)
	}

	fmt.Printf("🤖 Calling %s...\n\n", card.Name)

	if stream {
		// TODO: Implement streaming when we add WebSocket support
		fmt.Println("⚠️  Streaming not yet implemented, using standard execution")
	}

	// Execute task
	result, err := client.ExecuteTask(context.Background(), card, input, nil)
	if err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	// Check status
	if result.Status == a2a.TaskStatusFailed {
		if result.Error != nil {
			return fmt.Errorf("agent error: %s - %s", result.Error.Code, result.Error.Message)
		}
		return fmt.Errorf("task failed with unknown error")
	}

	// Extract and print output
	output := a2a.ExtractOutputText(result.Output)
	fmt.Println(output)

	// Show metadata if available
	if result.Metadata != nil {
		fmt.Println()
		if tokens, ok := result.Metadata["tokens_used"].(float64); ok {
			fmt.Printf("📊 Tokens: %.0f", tokens)
		}
		if duration, ok := result.Metadata["duration_ms"].(float64); ok {
			fmt.Printf(" | Duration: %.0fms", duration)
		}
		if len(result.Metadata) > 0 {
			fmt.Println()
		}
	}

	return nil
}

// executeChatCommand starts an interactive chat session with an agent
func executeChatCommand(agentURL string, token string) error {
	client := createA2AClient(token, "")

	// Get agent card
	card, err := client.DiscoverAgent(context.Background(), agentURL)
	if err != nil {
		return fmt.Errorf("failed to discover agent: %w", err)
	}

	fmt.Printf("💬 Chat with %s\n", card.Name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Type your messages below. Commands:")
	fmt.Println("  /quit or /exit - Exit chat")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println("  /info - Show agent information")
	fmt.Println()

	// TODO: Implement proper session management
	// For now, each message is independent

	scanner := os.Stdin
	reader := bufio.NewReader(scanner)

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println()
				break
			}
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/quit", "/exit":
				fmt.Println("Goodbye!")
				return nil
			case "/clear":
				fmt.Println("💭 Note: Session management not yet implemented")
				continue
			case "/info":
				fmt.Printf("\n🤖 %s\n", card.Name)
				fmt.Printf("   %s\n\n", card.Description)
				continue
			default:
				fmt.Printf("Unknown command: %s\n", input)
				continue
			}
		}

		// Execute task via A2A
		result, err := client.ExecuteTask(context.Background(), card, input, nil)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		if result.Status == a2a.TaskStatusFailed {
			if result.Error != nil {
				fmt.Printf("❌ Agent error: %s\n", result.Error.Message)
			} else {
				fmt.Println("❌ Task failed")
			}
			continue
		}

		// Print response
		output := a2a.ExtractOutputText(result.Output)
		fmt.Println(output)
		fmt.Println()
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// createA2AClient creates an A2A client with authentication
func createA2AClient(token string, apiKey string) *a2a.Client {
	var auth *a2a.AuthCredentials

	if token != "" {
		auth = &a2a.AuthCredentials{
			Type:  "bearer",
			Token: token,
		}
	} else if apiKey != "" {
		auth = &a2a.AuthCredentials{
			Type:   "apiKey",
			APIKey: apiKey,
		}
	}

	return a2a.NewClient(&a2a.ClientConfig{
		Auth: auth,
	})
}

// resolveServerURL resolves the server URL with defaults
func resolveServerURL(serverURL string) string {
	if serverURL == "" {
		// Check environment variable
		if envServer := os.Getenv("HECTOR_SERVER"); envServer != "" {
			return envServer
		}
		// Default to localhost
		return "http://localhost:8080"
	}

	// Ensure has http:// or https://
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		return "http://" + serverURL
	}

	return serverURL
}

// resolveAgentURL resolves agent URL, supporting shortcuts
func resolveAgentURL(agentURL string, defaultServer string) string {
	// If it's a full URL, use as-is
	if strings.HasPrefix(agentURL, "http://") || strings.HasPrefix(agentURL, "https://") {
		return agentURL
	}

	// Otherwise, it's an agent ID - prepend default server
	server := resolveServerURL(defaultServer)
	return fmt.Sprintf("%s/agents/%s", strings.TrimSuffix(server, "/"), agentURL)
}
