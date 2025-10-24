package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/runtime"
)

// ============================================================================
// COMMAND ARCHITECTURE
// ============================================================================

// VersionCommand shows Hector version information
func VersionCommand(args *CLIArgs, cfg *config.Config) error {
	// Get version from build info (same as main.go)
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = info.Main.Version
		}
	}

	fmt.Printf("Hector version %s\n", version)
	return nil
}

//
// Unified Config-First Architecture:
//
// Config is loaded ONCE in main.go before any command execution:
//   - Client mode: cfg = nil (uses server URL + token)
//   - Local modes: cfg = loaded/created, validated, defaults set
//
// All commands receive (args, cfg):
//   - If cfg == nil: Client mode (use HTTP client)
//   - If cfg != nil: Local mode (use config directly or pass to runtime)
//
// Benefits:
//   - Single point of config loading (main.go)
//   - Fail fast on config errors
//   - No duplicate validation/defaults
//   - Consistent pattern across all commands
//
// ============================================================================

// ListCommand lists all available agents
func ListCommand(args *CLIArgs, cfg *config.Config) error {
	if cfg == nil {
		// Client mode: use HTTP client
		httpClient := runtime.NewHTTPClient(args.ServerURL, args.Token)
		defer httpClient.Close()

		agents, err := httpClient.ListAgents(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		DisplayAgentList(agents, "Client Mode")
		return nil
	}

	// Local mode: use config directly (no runtime needed)
	agents := buildAgentCardsFromConfig(cfg)
	DisplayAgentList(agents, "Local Mode")
	return nil
}

// InfoCommand displays agent information
func InfoCommand(args *CLIArgs, cfg *config.Config) error {
	if cfg == nil {
		// Client mode: use HTTP client
		httpClient := runtime.NewHTTPClient(args.ServerURL, args.Token)
		defer httpClient.Close()

		card, err := httpClient.GetAgentCard(context.Background(), args.AgentID)
		if err != nil {
			return fmt.Errorf("failed to get agent card: %w", err)
		}

		DisplayAgentCard(args.AgentID, card)
		return nil
	}

	// Local mode: use config directly (no runtime needed)
	// Validate agent exists
	if err := cfg.ValidateAgent(args.AgentID); err != nil {
		return err
	}

	card := buildAgentCardFromConfig(cfg, args.AgentID)
	DisplayAgentCard(args.AgentID, card)
	return nil
}

// CallCommand sends a single message to an agent
func CallCommand(args *CLIArgs, cfg *config.Config) error {
	client, err := createClient(args, cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	return executeCall(client, args)
}

func executeCall(a2aClient client.A2AClient, args *CLIArgs) error {
	// Create message with optional session ID for conversation continuity
	msg := &pb.Message{
		ContextId: args.SessionID, // If empty, server will generate new one
		Role:      pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: args.Input},
			},
		},
	}

	ctx := context.Background()

	if args.Stream {
		// Streaming mode
		streamChan, err := a2aClient.StreamMessage(ctx, args.AgentID, msg)
		if err != nil {
			return fmt.Errorf("failed to start streaming: %w", err)
		}

		fmt.Print("Agent: ")
		for chunk := range streamChan {
			if msgChunk := chunk.GetMsg(); msgChunk != nil {
				DisplayMessage(msgChunk, "")
			}
		}
		fmt.Println()
	} else {
		// Non-streaming mode
		resp, err := a2aClient.SendMessage(ctx, args.AgentID, msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Display response
		if respMsg := resp.GetMsg(); respMsg != nil {
			DisplayMessageLine(respMsg, "Agent: ")
		} else if task := resp.GetTask(); task != nil {
			DisplayTask(task)
		}
	}

	return nil
}

// ChatCommand starts an interactive chat session
func ChatCommand(args *CLIArgs, cfg *config.Config) error {
	client, err := createClient(args, cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	return executeChat(client, args, cfg)
}

func executeChat(a2aClient client.A2AClient, args *CLIArgs, cfg *config.Config) error {
	// Display welcome message
	mode := "Client Mode"
	if cfg != nil {
		mode = "Local Mode"
	}

	streamInfo := ""
	if args.Stream {
		streamInfo = " (streaming)"
	}

	// Session management
	sessionID := args.SessionID
	if sessionID == "" {
		// Generate new session ID if not provided
		sessionID = fmt.Sprintf("cli-chat-%d", time.Now().Unix())
		fmt.Printf("\n🤖 Chat with %s (%s)%s\n", args.AgentID, mode, streamInfo)
		fmt.Printf("💾 Session ID: %s\n", sessionID)
		fmt.Printf("   Resume later with: --session=%s\n", sessionID)
		fmt.Println("   Type 'exit' to quit")
	} else {
		fmt.Printf("\n🤖 Chat with %s (%s)%s\n", args.AgentID, mode, streamInfo)
		fmt.Printf("💾 Resuming session: %s\n", sessionID)
		fmt.Println("   Type 'exit' to quit")
	}

	// Create reader for user input
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		DisplayChatPrompt()
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "/exit" || input == "/quit" {
			DisplayGoodbye()
			break
		}

		// Create message with consistent session ID for conversation continuity
		msg := &pb.Message{
			ContextId: sessionID, // Use consistent session ID throughout conversation
			Role:      pb.Role_ROLE_USER,
			Content: []*pb.Part{
				{
					Part: &pb.Part_Text{Text: input},
				},
			},
		}

		DisplayAgentPrompt(args.AgentID)

		if args.Stream {
			// Streaming mode
			streamChan, err := a2aClient.StreamMessage(ctx, args.AgentID, msg)
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			for chunk := range streamChan {
				if msgChunk := chunk.GetMsg(); msgChunk != nil {
					DisplayMessage(msgChunk, "")
				}
			}
			fmt.Println()
		} else {
			// Non-streaming mode
			resp, err := a2aClient.SendMessage(ctx, args.AgentID, msg)
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			if respMsg := resp.GetMsg(); respMsg != nil {
				DisplayMessage(respMsg, "")
			} else if task := resp.GetTask(); task != nil {
				fmt.Printf("Task created: %s (status: %s)", task.Id, task.Status)
			}
			fmt.Println()
		}
	}

	return nil
}

// ============================================================================
// CLIENT CREATION HELPER
// ============================================================================
//
// createClient creates appropriate client based on whether cfg is nil:
//   - cfg == nil: Client mode → HTTPClient
//   - cfg != nil: Local mode → Runtime (which implements A2AClient)
//
// ============================================================================

// createClient creates appropriate client based on config presence
// Used by call/chat/task commands for execution
func createClient(args *CLIArgs, cfg *config.Config) (client.A2AClient, error) {
	if cfg == nil {
		// Client mode: HTTP client for remote server
		return runtime.NewHTTPClient(args.ServerURL, args.Token), nil
	}

	// Local mode: validate agent exists before expensive initialization
	if args.AgentID != "" {
		if err := cfg.ValidateAgent(args.AgentID); err != nil {
			return nil, err
		}
	}

	// Create runtime with validated config
	// Runtime implements A2AClient interface directly - no wrapper needed!
	return runtime.NewWithConfig(cfg)
}

// ============================================================================
// AGENT CARD BUILDING
// ============================================================================
//
// These functions build AgentCard protobuf objects from config metadata
// without instantiating actual agent instances. This allows fast metadata
// queries for list/info commands.
//
// ============================================================================

// buildAgentCardsFromConfig builds AgentCard objects from config metadata
// without creating actual agent instances (lightweight operation)
func buildAgentCardsFromConfig(cfg *config.Config) []*pb.AgentCard {
	cards := make([]*pb.AgentCard, 0, len(cfg.Agents))

	for agentID := range cfg.Agents {
		card := buildAgentCardFromConfig(cfg, agentID)
		cards = append(cards, card)
	}

	return cards
}

// buildAgentCardFromConfig builds a single AgentCard from config metadata
func buildAgentCardFromConfig(cfg *config.Config, agentID string) *pb.AgentCard {
	agentCfg := cfg.Agents[agentID]

	// Build description from agent config and LLM info
	description := agentCfg.Description
	if description == "" {
		// Auto-generate description from LLM info
		if llmCfg, ok := cfg.LLMs[agentCfg.LLM]; ok {
			description = fmt.Sprintf("AI assistant powered by %s (%s)", llmCfg.Type, llmCfg.Model)
		} else {
			description = "AI assistant"
		}
	}

	// Determine if streaming is supported (native agents always support streaming)
	supportsStreaming := agentCfg.Type != "a2a" // Native agents support streaming, external might not

	return &pb.AgentCard{
		Name:        agentCfg.Name,
		Description: description,
		Version:     "1.0.0", // Default version for config-based agents
		Url:         fmt.Sprintf("local://%s", agentID),
		Capabilities: &pb.AgentCapabilities{
			Streaming: supportsStreaming,
		},
	}
}
