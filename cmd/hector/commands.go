package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	clientIdentifier      = "hector-cli"
	sessionIDDisplayLen   = 8
	sessionCleanupTimeout = 5 * time.Second
	defaultServerURL      = "http://localhost:8080"
	envVarServer          = "HECTOR_SERVER"
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
		fmt.Printf("     URL: %s\n", agent.URL)
		if agent.Description != "" {
			fmt.Printf("     %s\n", agent.Description)
		}
		// Display capabilities
		capStr := "None"
		if agent.Capabilities.Streaming || agent.Capabilities.MultiTurn {
			caps := []string{}
			if agent.Capabilities.Streaming {
				caps = append(caps, "streaming")
			}
			if agent.Capabilities.MultiTurn {
				caps = append(caps, "multi-turn")
			}
			capStr = strings.Join(caps, ", ")
		}
		fmt.Printf("     Capabilities: %s\n", capStr)
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
	fmt.Printf("URL:         %s\n", card.URL)
	fmt.Printf("Description: %s\n", card.Description)
	fmt.Printf("Version:     %s\n", card.Version)
	fmt.Printf("Transport:   %s\n", card.PreferredTransport)
	fmt.Println()

	fmt.Println("Capabilities:")
	if card.Capabilities.Streaming {
		fmt.Printf("  • Streaming: Yes\n")
	}
	if card.Capabilities.MultiTurn {
		fmt.Printf("  • Multi-turn: Yes\n")
	}
	if card.Capabilities.PushNotifications {
		fmt.Printf("  • Push notifications: Yes\n")
	}
	fmt.Println()

	if len(card.Skills) > 0 {
		fmt.Println("Skills:")
		for _, skill := range card.Skills {
			fmt.Printf("  • %s: %s\n", skill.Name, skill.Description)
		}
		fmt.Println()
	}

	if len(card.SecuritySchemes) > 0 {
		fmt.Println("Authentication:")
		for _, scheme := range card.SecuritySchemes {
			fmt.Printf("  • Type: %s\n", scheme.Type)
		}
		fmt.Println()
	}

	return nil
}

// executeCallCommand executes a one-shot task on an agent
func executeCallCommand(agentURL string, input string, token string, stream bool) error {
	client := createA2AClient(token, "")

	// Get agent card
	card, err := discoverAgent(client, agentURL)
	if err != nil {
		return err
	}

	fmt.Printf("🤖 Calling %s...\n\n", card.Name)

	if stream {
		// Use A2A SSE streaming
		message := a2a.CreateTextMessage(a2a.MessageRoleUser, input)
		eventCh, err := client.SendMessageStreaming(context.Background(), card.URL, message)
		if err != nil {
			return fmt.Errorf("streaming failed: %w", err)
		}

		// Process streaming events
		for event := range eventCh {
			switch event.Type {
			case a2a.StreamEventTypeMessage:
				if event.Message != nil && event.Message.Role == a2a.MessageRoleAssistant {
					for _, part := range event.Message.Parts {
						if part.Type == a2a.PartTypeText {
							fmt.Print(part.Text)
						}
					}
				}
			case a2a.StreamEventTypeStatus:
				if event.Status != nil && event.Status.State == a2a.TaskStateFailed {
					fmt.Println("\n❌ Task failed")
					if event.Status.Reason != "" {
						return fmt.Errorf("agent error: %s", event.Status.Reason)
					}
					return fmt.Errorf("task failed")
				}
			}
		}
		fmt.Println() // New line after streaming
		return nil
	}

	// Non-streaming execution
	task, err := client.SendTextMessage(context.Background(), card.URL, input)
	if err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	// Check status
	if task.Status.State == a2a.TaskStateFailed {
		if task.Error != nil {
			return fmt.Errorf("agent error: %s - %s", task.Error.Code, task.Error.Message)
		}
		return fmt.Errorf("task failed with unknown error")
	}

	// Extract and print output from assistant messages
	output := a2a.ExtractTextFromTask(task)
	fmt.Println(output)

	// Show metadata if available
	if task.Metadata != nil {
		fmt.Println()
		if tokens, ok := task.Metadata["tokens_used"].(float64); ok {
			fmt.Printf("📊 Tokens: %.0f", tokens)
		}
		if duration, ok := task.Metadata["duration_ms"].(float64); ok {
			fmt.Printf(" | Duration: %.0fms", duration)
		}
		if len(task.Metadata) > 0 {
			fmt.Println()
		}
	}

	return nil
}

// executeChatCommand starts an interactive chat session with an agent
func executeChatCommand(agentURL string, token string) error {
	client := createA2AClient(token, "")

	// Get agent card
	card, err := discoverAgent(client, agentURL)
	if err != nil {
		return err
	}

	fmt.Printf("💬 Chat with %s\n", card.Name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Type your messages below. Commands:")
	fmt.Println("  /quit or /exit - Exit chat")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println("  /info - Show agent information")
	fmt.Println()

	// Determine base server URL from agent card
	baseURL, err := extractBaseURL(card.URL)
	if err != nil {
		// Fallback: try the provided agentURL
		baseURL, err = extractBaseURL(agentURL)
		if err != nil {
			return fmt.Errorf("failed to extract base URL: %w", err)
		}
	}

	// Create session for multi-turn conversation
	metadata := map[string]interface{}{
		"client": clientIdentifier,
	}

	session, err := client.CreateSession(context.Background(), baseURL, card.Name, metadata)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	fmt.Printf("🔗 Session: %s\n\n", session.ID[:sessionIDDisplayLen]+"...")

	// Ensure session cleanup on exit
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), sessionCleanupTimeout)
		defer cancel()
		_ = client.DeleteSession(ctx, baseURL, session.ID)
	}()

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
				// Create new session (clears history)
				newSession, err := client.CreateSession(context.Background(), baseURL, card.Name, metadata)
				if err != nil {
					fmt.Printf("❌ Failed to clear history: %v\n", err)
					continue
				}
				// Delete old session
				ctx, cancel := context.WithTimeout(context.Background(), sessionCleanupTimeout)
				_ = client.DeleteSession(ctx, baseURL, session.ID)
				cancel()
				// Use new session
				session = newSession
				fmt.Printf("💭 Conversation history cleared (new session: %s)\n\n", session.ID[:sessionIDDisplayLen]+"...")
				continue
			case "/info":
				fmt.Printf("\n🤖 %s\n", card.Name)
				fmt.Printf("   %s\n", card.Description)
				fmt.Printf("   Session: %s\n\n", session.ID)
				continue
			default:
				fmt.Printf("Unknown command: %s\n", input)
				continue
			}
		}

		// Execute task with streaming using A2A SSE
		message := a2a.CreateTextMessage(a2a.MessageRoleUser, input)
		eventCh, execErr := client.SendMessageStreaming(context.Background(), card.URL, message)

		if execErr != nil {
			fmt.Printf("\n❌ Error: %v\n", execErr)
			continue
		}

		// Process streaming events
		failed := false

		for event := range eventCh {
			switch event.Type {
			case a2a.StreamEventTypeMessage:
				if event.Message != nil && event.Message.Role == a2a.MessageRoleAssistant {
					// Print each chunk as it arrives (chunks are already incremental)
					for _, part := range event.Message.Parts {
						if part.Type == a2a.PartTypeText {
							fmt.Print(part.Text)
						}
					}
				}

			case a2a.StreamEventTypeStatus:
				if event.Status != nil {
					switch event.Status.State {
					case a2a.TaskStateFailed:
						failed = true
						fmt.Println("\n❌ Task failed")
						if event.Status.Reason != "" {
							fmt.Printf("   Reason: %s\n", event.Status.Reason)
						}
					case a2a.TaskStateCompleted:
						// Task completed successfully
					}
				}

			case a2a.StreamEventTypeArtifact:
				// Artifacts are typically handled separately or could be shown inline
				// For now, we'll just note that we received them
			}
		}

		if !failed {
			fmt.Println() // New line after streaming completes
		}
		fmt.Println()
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// discoverAgent discovers an agent and returns its card
func discoverAgent(client *a2a.Client, agentURL string) (*a2a.AgentCard, error) {
	card, err := client.DiscoverAgent(context.Background(), agentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover agent: %w", err)
	}
	return card, nil
}

// extractBaseURL extracts the base URL (scheme + host) from a full URL
// Example: "http://localhost:8080/agents/my_agent" → "http://localhost:8080"
func extractBaseURL(fullURL string) (string, error) {
	parsed, err := url.Parse(fullURL)
	if err != nil {
		return "", err
	}

	// Build base URL with scheme and host only
	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	return baseURL, nil
}

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

// resolveServerURL resolves the server URL with defaults and smart fallbacks:
// 1. Use provided serverURL if not empty
// 2. Fall back to HECTOR_SERVER environment variable
// 3. Fall back to http://localhost:8080
// Also ensures the URL has a proper scheme (http:// or https://)
func resolveServerURL(serverURL string) string {
	if serverURL == "" {
		// Check environment variable
		if envServer := os.Getenv(envVarServer); envServer != "" {
			serverURL = envServer
		} else {
			// Default to localhost
			return defaultServerURL
		}
	}

	// Ensure has http:// or https://
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		return "http://" + serverURL
	}

	return serverURL
}

// resolveAgentURL resolves agent URL, supporting two formats:
// 1. Agent ID (shorthand): "my_agent" → "http://server:port/agents/my_agent"
// 2. Full URL: "http://server:port/agents/my_agent" → used as-is
// When using shorthand, the defaultServer parameter or environment variables determine the server
func resolveAgentURL(agentID string, defaultServer string) string {
	// If it's a full URL, use as-is (ignores defaultServer)
	if strings.HasPrefix(agentID, "http://") || strings.HasPrefix(agentID, "https://") {
		return agentID
	}

	// Otherwise, it's an agent ID - construct full URL with default server
	server := resolveServerURL(defaultServer)
	return fmt.Sprintf("%s/agents/%s", strings.TrimSuffix(server, "/"), agentID)
}
