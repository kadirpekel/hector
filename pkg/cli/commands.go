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
// HELPER FUNCTIONS
// ============================================================================

// ============================================================================
// COMMAND ARCHITECTURE
// ============================================================================

// VersionCommand shows Hector version information
func VersionCommand(args *VersionCmd, cfg *config.Config, mode CLIMode) error {
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
func ListCommand(args *ListCmd, cfg *config.Config, mode CLIMode) error {
	switch mode {
	case ModeClient:
		// Client mode: use HTTP client
		httpClient := runtime.NewHTTPClient(args.Server, args.Token)
		defer httpClient.Close()

		agents, err := httpClient.ListAgents(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		DisplayAgentList(agents, "Client Mode")
		return nil

	case ModeLocalConfig:
		// Local mode with config: use config directly (no runtime needed)
		agents := buildAgentCardsFromConfig(cfg)
		DisplayAgentList(agents, "Local Mode")
		return nil

	case ModeLocalZeroConfig:
		// Zero-config mode: only one agent exists
		fmt.Println("\n📋 Available Agents (Local Mode)\n")
		fmt.Printf("Found 1 agent(s):\n\n")
		fmt.Printf("• Assistant (v1.0.0)\n")
		fmt.Printf("  Description: Zero-config AI assistant\n")
		fmt.Printf("  URL: local://assistant\n")
		fmt.Printf("  Streaming: ✓\n")
		fmt.Printf("\n💡 Use 'hector call \"message\"' to interact with the agent\n")
		return nil

	default:
		return fmt.Errorf("unsupported mode for list command: %s", mode)
	}
}

// InfoCommand displays agent information
func InfoCommand(args *InfoCmd, cfg *config.Config, mode CLIMode) error {
	switch mode {
	case ModeClient:
		// Client mode: use HTTP client
		httpClient := runtime.NewHTTPClient(args.Server, args.Token)
		defer httpClient.Close()

		card, err := httpClient.GetAgentCard(context.Background(), args.Agent)
		if err != nil {
			return fmt.Errorf("failed to get agent card: %w", err)
		}

		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalConfig:
		// Local mode with config: use config directly (no runtime needed)
		// Validate agent exists
		if err := cfg.ValidateAgent(args.Agent); err != nil {
			return err
		}

		card := buildAgentCardFromConfig(cfg, args.Agent)
		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalZeroConfig:
		// Zero-config mode: only "assistant" agent exists
		if args.Agent != "assistant" && args.Agent != config.DefaultAgentName {
			return fmt.Errorf("agent '%s' not found. In zero-config mode, only 'assistant' is available", args.Agent)
		}

		// Display zero-config agent card
		fmt.Printf("\n📋 Agent Information\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Name:        assistant\n")
		fmt.Printf("Version:     v1.0.0\n")
		fmt.Printf("Description: Zero-config AI assistant\n")
		fmt.Printf("URL:         local://assistant\n")
		fmt.Printf("Streaming:   ✓\n")
		fmt.Printf("\n💡 Use 'hector call \"message\"' to interact with this agent\n")
		return nil

	default:
		return fmt.Errorf("unsupported mode for info command: %s", mode)
	}
}

// CallCommand sends a single message to an agent
func CallCommand(args *CallCmd, cfg *config.Config, mode CLIMode) error {
	// Resolve agent ID based on mode
	var agentID string
	if mode == ModeClient {
		// Client mode: agent is required
		if args.Agent == "" {
			return fmt.Errorf("agent name is required in client mode (use --agent flag)")
		}
		agentID = args.Agent
	} else if mode == ModeLocalConfig {
		// Local mode with config file: agent is required
		if args.Agent == "" {
			// List available agents
			agentNames := make([]string, 0, len(cfg.Agents))
			for name := range cfg.Agents {
				agentNames = append(agentNames, name)
			}
			return fmt.Errorf("agent name is required when using --config flag. Available agents: %v", agentNames)
		}
		agentID = args.Agent
	} else {
		// Zero-config mode: agent is forbidden
		if args.Agent != "" {
			return fmt.Errorf("agent name is not allowed in zero-config mode (remove --agent flag)")
		}
		agentID = config.DefaultAgentName
	}

	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Session management - generate session ID if not provided
	sessionID := args.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("cli-call-%d", time.Now().Unix())
		fmt.Printf("💾 Session ID: %s (resume with --session=%s)\n", sessionID, sessionID)
	} else {
		fmt.Printf("💾 Resuming session: %s\n", sessionID)
	}

	// Create message with session ID for conversation continuity
	msg := &pb.Message{
		ContextId: sessionID,
		Role:      pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: args.Message},
			},
		},
	}

	ctx := context.Background()

	if args.Stream {
		// Streaming mode
		streamChan, err := client.StreamMessage(ctx, agentID, msg)
		if err != nil {
			return fmt.Errorf("failed to start streaming: %w", err)
		}

		for chunk := range streamChan {
			if msgChunk := chunk.GetMsg(); msgChunk != nil {
				DisplayMessage(msgChunk, "")
			}
		}
		fmt.Println()
		return nil
	} else {
		// Non-streaming mode
		resp, err := client.SendMessage(ctx, agentID, msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Display response
		if respMsg := resp.GetMsg(); respMsg != nil {
			DisplayMessageLine(respMsg, "Agent: ")
		} else if task := resp.GetTask(); task != nil {
			DisplayTask(task)
		}
		return nil
	}
}

// ChatCommand starts an interactive chat session
// ChatCommand starts an interactive chat session with an agent
func ChatCommand(args *ChatCmd, cfg *config.Config, mode CLIMode) error {
	// Resolve agent ID based on mode
	var agentID string
	if mode == ModeClient {
		// Client mode: agent is required
		if args.Agent == "" {
			return fmt.Errorf("agent name is required in client mode (use --agent flag)")
		}
		agentID = args.Agent
	} else if mode == ModeLocalConfig {
		// Local mode with config file: agent is required
		if args.Agent == "" {
			// List available agents
			agentNames := make([]string, 0, len(cfg.Agents))
			for name := range cfg.Agents {
				agentNames = append(agentNames, name)
			}
			return fmt.Errorf("agent name is required when using --config flag. Available agents: %v", agentNames)
		}
		agentID = args.Agent
	} else {
		// Zero-config mode: agent is forbidden
		if args.Agent != "" {
			return fmt.Errorf("agent name is not allowed in zero-config mode (remove --agent flag)")
		}
		agentID = config.DefaultAgentName
	}

	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	return executeChat(client, agentID, args.SessionID, !args.NoStream)
}

func executeChat(a2aClient client.A2AClient, agentID, sessionID string, streaming bool) error {
	// Display welcome message
	mode := "Client Mode"
	// Note: We can't determine mode here anymore, but it's not critical for chat

	// Session management
	if sessionID == "" {
		// Generate new session ID if not provided
		sessionID = fmt.Sprintf("cli-chat-%d", time.Now().Unix())
		fmt.Printf("\n🤖 Chat with %s (%s)\n", agentID, mode)
		fmt.Printf("💾 Session ID: %s\n", sessionID)
		fmt.Printf("   Resume later with: --session=%s\n", sessionID)
		fmt.Println("   Type 'exit' to quit")
	} else {
		fmt.Printf("\n🤖 Chat with %s (%s)\n", agentID, mode)
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

		DisplayAgentPrompt(agentID)

		// Use the streaming preference passed as parameter

		if streaming {
			// Streaming mode
			streamChan, err := a2aClient.StreamMessage(ctx, agentID, msg)
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
			resp, err := a2aClient.SendMessage(ctx, agentID, msg)
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			if respMsg := resp.GetMsg(); respMsg != nil {
				DisplayMessageLine(respMsg, "")
			} else if task := resp.GetTask(); task != nil {
				DisplayTask(task)
			}
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

// ClientArgs interface for commands that need client creation
type ClientArgs interface {
	GetServer() string
	GetToken() string
	GetAgent() string
}

// Implement ClientArgs for CallCmd
func (c *CallCmd) GetServer() string { return c.Server }
func (c *CallCmd) GetToken() string  { return c.Token }
func (c *CallCmd) GetAgent() string  { return c.Agent }

// Implement ClientArgs for ChatCmd
func (c *ChatCmd) GetServer() string { return c.Server }
func (c *ChatCmd) GetToken() string  { return c.Token }
func (c *ChatCmd) GetAgent() string  { return c.Agent }

// Implement ClientArgs for TaskGetCmd
func (t *TaskGetCmd) GetServer() string { return CLI.Task.Server }
func (t *TaskGetCmd) GetToken() string  { return CLI.Task.Token }
func (t *TaskGetCmd) GetAgent() string  { return t.Agent }

// Implement ClientArgs for TaskCancelCmd
func (t *TaskCancelCmd) GetServer() string { return CLI.Task.Server }
func (t *TaskCancelCmd) GetToken() string  { return CLI.Task.Token }
func (t *TaskCancelCmd) GetAgent() string  { return t.Agent }

// createClient creates appropriate client based on command args and explicit mode
// Used by call/chat/task commands for execution
func createClient[T ClientArgs](args T, cfg *config.Config, mode CLIMode) (client.A2AClient, error) {
	switch mode {
	case ModeClient:
		// Client mode: use server URL and agent name
		return runtime.NewHTTPClient(args.GetServer(), args.GetToken()), nil

	case ModeLocalConfig:
		// Local mode with config: create runtime with config
		// Runtime implements A2AClient interface directly - no wrapper needed!
		return runtime.NewWithConfig(cfg)
	case ModeLocalZeroConfig:
		// Local zero-config mode: create runtime with default options
		// Runtime implements A2AClient interface directly - no wrapper needed!
		rt, err := runtime.New(runtime.Options{})
		if err != nil {
			return nil, err
		}
		return rt, nil

	default:
		return nil, fmt.Errorf("unsupported mode for client creation: %s", mode)
	}
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
