package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
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
//
// Commands are organized into two categories based on initialization requirements:
//
// 1. METADATA COMMANDS (list, info)
//    Client Mode:  createRuntimeClient() → runtime.NewHTTPClient()
//    Local Modes:  loadConfigLightweight() → config only, no runtime
//    Result:       Fast metadata queries without expensive initialization
//
// 2. EXECUTION COMMANDS (call, chat, task)
//    Client Mode:  createRuntimeClient() → runtime.NewHTTPClient()
//    Local Modes:  createRuntimeClient() → runtime.New() with full stack
//    Result:       Complete execution environment with all components
//
// Key Design Principles:
// - Client mode: ALL commands use createRuntimeClient() for consistency
// - Local metadata: Skip runtime, use loadConfigLightweight() for speed
// - Local execution: Full runtime via createRuntimeClient() for agent operations
// - Single factory pattern eliminates duplication across all commands
//
// ============================================================================

// ListCommand lists all available agents
func ListCommand(args *CLIArgs) error {
	mode := DetectMode(args)

	switch mode {
	case ModeClient:
		// Client mode: use runtime client factory
		a2aClient, err := createRuntimeClient(args)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer a2aClient.Close()

		agents, err := a2aClient.ListAgents(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		DisplayAgentList(agents, mode.String())
		return nil

	case ModeLocalConfig, ModeLocalZeroConfig:
		// Local modes: lightweight config read (no runtime initialization)
		cfg, err := loadConfigLightweight(args)
		if err != nil {
			return err
		}

		agents := buildAgentCardsFromConfig(cfg)
		DisplayAgentList(agents, mode.String())
		return nil

	default:
		return fmt.Errorf("list command not supported in %s mode", mode.String())
	}
}

// InfoCommand displays agent information
func InfoCommand(args *CLIArgs) error {
	mode := DetectMode(args)

	switch mode {
	case ModeClient:
		// Client mode: use runtime client factory
		a2aClient, err := createRuntimeClient(args)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer a2aClient.Close()

		card, err := a2aClient.GetAgentCard(context.Background(), args.AgentID)
		if err != nil {
			return fmt.Errorf("failed to get agent card: %w", err)
		}

		DisplayAgentCard(args.AgentID, card)
		return nil

	case ModeLocalConfig, ModeLocalZeroConfig:
		// Local modes: lightweight config read (no runtime initialization)
		cfg, err := loadConfigLightweight(args)
		if err != nil {
			return err
		}

		// Validate agent exists
		if err := cfg.ValidateAgent(args.AgentID); err != nil {
			return err
		}

		card := buildAgentCardFromConfig(cfg, args.AgentID)
		DisplayAgentCard(args.AgentID, card)
		return nil

	default:
		return fmt.Errorf("info command not supported in %s mode", mode.String())
	}
}

// CallCommand sends a single message to an agent
func CallCommand(args *CLIArgs) error {
	a2aClient, err := createRuntimeClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	return executeCall(a2aClient, args)
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
func ChatCommand(args *CLIArgs) error {
	a2aClient, err := createRuntimeClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	return executeChat(a2aClient, args)
}

func executeChat(a2aClient client.A2AClient, args *CLIArgs) error {
	// Display welcome message
	mode := DetectMode(args)

	streamInfo := ""
	if args.Stream {
		streamInfo = " (streaming)"
	}

	// Session management
	sessionID := args.SessionID
	if sessionID == "" {
		// Generate new session ID if not provided
		sessionID = fmt.Sprintf("cli-chat-%d", time.Now().Unix())
		fmt.Printf("\n🤖 Chat with %s (%s)%s\n", args.AgentID, mode.String(), streamInfo)
		fmt.Printf("💾 Session ID: %s\n", sessionID)
		fmt.Printf("   Resume later with: --session=%s\n", sessionID)
		fmt.Println("   Type 'exit' to quit")
	} else {
		fmt.Printf("\n🤖 Chat with %s (%s)%s\n", args.AgentID, mode.String(), streamInfo)
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
// CLIENT CREATION HELPERS
// ============================================================================
//
// Two helper functions provide the core initialization logic:
//
// - loadConfigLightweight(): Config-only read (no agent/component creation)
//   Used by list/info commands in local modes only
//
// - createRuntimeClient(): Creates appropriate client based on mode
//   Used by ALL commands in client mode
//   Used by call/chat/task commands in local modes
//
// ============================================================================

// loadConfigLightweight loads config without runtime/component initialization
// Used by list/info commands for fast metadata queries in local modes
func loadConfigLightweight(args *CLIArgs) (*config.Config, error) {
	cfg, err := runtime.LoadConfigForValidation(args.ConfigFile, args.ToRuntimeOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

// createRuntimeClient creates appropriate client based on mode
// Used by ALL commands in client mode (returns HTTP client)
// Used by call/chat/task in local modes (returns full runtime client)
func createRuntimeClient(args *CLIArgs) (client.A2AClient, error) {
	mode := DetectMode(args)

	switch mode {
	case ModeClient:
		// Client mode: HTTP client for remote server
		return runtime.NewHTTPClient(args.ServerURL, args.Token), nil

	case ModeLocalConfig:
		// Local config mode: validate agent before expensive initialization
		cfg, err := loadConfigLightweight(args)
		if err != nil {
			return nil, err
		}

		// Validate agent exists before expensive initialization
		if args.AgentID != "" {
			if err := cfg.ValidateAgent(args.AgentID); err != nil {
				return nil, err
			}
		}

		// Create runtime with validated config
		rt, err := runtime.NewWithConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize runtime: %w", err)
		}
		return rt.Client(), nil

	case ModeLocalZeroConfig:
		// Local zero-config mode: create runtime with auto-config
		rt, err := runtime.New(args.ToRuntimeOptions())
		if err != nil {
			return nil, fmt.Errorf("failed to initialize runtime: %w", err)
		}
		return rt.Client(), nil

	default:
		return nil, fmt.Errorf("unsupported mode: %s", mode.String())
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
