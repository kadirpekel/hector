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

func VersionCommand(args *VersionCmd, cfg *config.Config, mode CLIMode) error {

	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = info.Main.Version
		}
	}

	fmt.Printf("Hector version %s\n", version)
	return nil
}

func InfoCommand(args *InfoCmd, cfg *config.Config, mode CLIMode) error {
	switch mode {
	case ModeClient:
		// Use UniversalA2AClient to get agent card from any A2A service
		url := args.URL
		if url == "" {
			return fmt.Errorf("--url is required in client mode (agent card URL or service base URL)")
		}

		a2aClient, err := client.NewUniversalA2AClient(url, args.Agent, args.Token, nil)
		if err != nil {
			return fmt.Errorf("failed to create A2A client: %w", err)
		}
		defer a2aClient.Close()

		card, err := a2aClient.GetAgentCard(context.Background(), args.Agent)
		if err != nil {
			return fmt.Errorf("failed to get agent card: %w", err)
		}

		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalConfig:
		// Local mode - build from config
		if err := cfg.ValidateAgent(args.Agent); err != nil {
			return err
		}

		card := buildAgentCardFromConfig(cfg, args.Agent)
		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalZeroConfig:
		// Zero-config mode
		if args.Agent != config.DefaultAgentName {
			return fmt.Errorf("agent '%s' not found. In zero-config mode, only 'assistant' is available", args.Agent)
		}

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

func CallCommand(args *CallCmd, cfg *config.Config, mode CLIMode) error {

	// Validate agent requirements per mode
	if mode == ModeLocalConfig && args.Agent == "" {
		agentIDs := make([]string, 0, len(cfg.Agents))
		for id := range cfg.Agents {
			agentIDs = append(agentIDs, id)
		}
		return fmt.Errorf("agent ID is required when using --config flag. Available agents: %v", agentIDs)
	} else if mode == ModeLocalZeroConfig && args.Agent != "" {
		return fmt.Errorf("agent ID is not allowed in zero-config mode (remove --agent flag)")
	}

	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Determine agent ID
	var agentID string
	if mode == ModeLocalZeroConfig {
		agentID = config.DefaultAgentName
	} else if args.Agent != "" {
		agentID = args.Agent
	} else {
		// Discover agent ID from client (for client mode with URL-only)
		agentID = client.GetAgentID()
		if agentID == "" {
			return fmt.Errorf("could not determine agent ID")
		}
	}

	sessionID := args.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("cli-call-%d", time.Now().Unix())
		fmt.Printf("💾 Session ID: %s (resume with --session=%s)\n", sessionID, sessionID)
	} else {
		fmt.Printf("💾 Resuming session: %s\n", sessionID)
	}

	msg := &pb.Message{
		ContextId: sessionID,
		Role:      pb.Role_ROLE_USER,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: args.Message},
			},
		},
	}

	ctx := context.Background()

	if args.Stream {

		streamChan, err := client.StreamMessage(ctx, agentID, msg)
		if err != nil {
			return fmt.Errorf("failed to start streaming: %w", err)
		}

		for chunk := range streamChan {
			if msgChunk := chunk.GetMsg(); msgChunk != nil {
				DisplayMessage(msgChunk, "", args.Thinking)
			}
		}
		fmt.Println()
		return nil
	} else {

		resp, err := client.SendMessage(ctx, agentID, msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		if respMsg := resp.GetMsg(); respMsg != nil {
			DisplayMessageLine(respMsg, "Agent: ", args.Thinking)
		} else if task := resp.GetTask(); task != nil {
			DisplayTask(task)
		}
		return nil
	}
}

func ChatCommand(args *ChatCmd, cfg *config.Config, mode CLIMode) error {

	// Validate agent requirements per mode
	if mode == ModeLocalConfig && args.Agent == "" {
		agentIDs := make([]string, 0, len(cfg.Agents))
		for id := range cfg.Agents {
			agentIDs = append(agentIDs, id)
		}
		return fmt.Errorf("agent ID is required when using --config flag. Available agents: %v", agentIDs)
	} else if mode == ModeLocalZeroConfig && args.Agent != "" {
		return fmt.Errorf("agent ID is not allowed in zero-config mode (remove --agent flag)")
	}

	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Determine agent ID
	var agentID string
	if mode == ModeLocalZeroConfig {
		agentID = config.DefaultAgentName
	} else if args.Agent != "" {
		agentID = args.Agent
	} else {
		// Discover agent ID from client (for client mode with URL-only)
		agentID = client.GetAgentID()
		if agentID == "" {
			return fmt.Errorf("could not determine agent ID")
		}
	}

	return executeChat(client, agentID, args.SessionID, !args.NoStream, args.Thinking)
}

func executeChat(a2aClient client.A2AClient, agentID, sessionID string, streaming bool, showThinking bool) error {

	mode := "Client Mode"

	if sessionID == "" {

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

		msg := &pb.Message{
			ContextId: sessionID,
			Role:      pb.Role_ROLE_USER,
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{Text: input},
				},
			},
		}

		DisplayAgentPrompt(agentID)

		if streaming {

			streamChan, err := a2aClient.StreamMessage(ctx, agentID, msg)
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			for chunk := range streamChan {
				if msgChunk := chunk.GetMsg(); msgChunk != nil {
					DisplayMessage(msgChunk, "", showThinking)
				}
			}
			fmt.Println()
		} else {

			resp, err := a2aClient.SendMessage(ctx, agentID, msg)
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			if respMsg := resp.GetMsg(); respMsg != nil {
				DisplayMessageLine(respMsg, "", showThinking)
			} else if task := resp.GetTask(); task != nil {
				DisplayTask(task)
			}
		}
	}

	return nil
}

type ClientArgs interface {
	GetURL() string
	GetToken() string
	GetAgent() string
}

func (c *CallCmd) GetURL() string   { return c.URL }
func (c *CallCmd) GetToken() string { return c.Token }
func (c *CallCmd) GetAgent() string { return c.Agent }

func (c *ChatCmd) GetURL() string   { return c.URL }
func (c *ChatCmd) GetToken() string { return c.Token }
func (c *ChatCmd) GetAgent() string { return c.Agent }

func (t *TaskGetCmd) GetURL() string   { return CLI.Task.URL }
func (t *TaskGetCmd) GetToken() string { return CLI.Task.Token }
func (t *TaskGetCmd) GetAgent() string { return t.Agent }

func (t *TaskCancelCmd) GetURL() string   { return CLI.Task.URL }
func (t *TaskCancelCmd) GetToken() string { return CLI.Task.Token }
func (t *TaskCancelCmd) GetAgent() string { return t.Agent }

func createClient[T ClientArgs](args T, cfg *config.Config, mode CLIMode) (client.A2AClient, error) {
	switch mode {
	case ModeClient:
		// Use UniversalA2AClient for true A2A interoperability
		// Auto-discovers agent card, chooses transport (gRPC/REST/JSON-RPC)
		// Works with ANY A2A-compliant service (not just Hector)
		url := args.GetURL()
		if url == "" {
			return nil, fmt.Errorf("--url is required in client mode (provide agent card URL or service base URL)")
		}

		agentID := args.GetAgent()
		token := args.GetToken()

		// Create universal A2A client (discovers agent and chooses transport)
		a2aClient, err := client.NewUniversalA2AClient(url, agentID, token, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create A2A client: %w", err)
		}

		return a2aClient, nil

	case ModeLocalConfig:
		// Local mode with config file
		return runtime.NewWithConfig(cfg)
	case ModeLocalZeroConfig:
		// Zero-config local mode
		return runtime.NewWithConfig(cfg)

	default:
		return nil, fmt.Errorf("unsupported mode for client creation: %s", mode)
	}
}

//nolint:unused // Reserved for future multi-agent card support
func buildAgentCardsFromConfig(cfg *config.Config) []*pb.AgentCard {
	cards := make([]*pb.AgentCard, 0, len(cfg.Agents))

	for agentID := range cfg.Agents {
		card := buildAgentCardFromConfig(cfg, agentID)
		cards = append(cards, card)
	}

	return cards
}

func buildAgentCardFromConfig(cfg *config.Config, agentID string) *pb.AgentCard {
	agentCfg := cfg.Agents[agentID]

	description := agentCfg.Description
	if description == "" {

		if llmCfg, ok := cfg.LLMs[agentCfg.LLM]; ok {
			description = fmt.Sprintf("AI assistant powered by %s (%s)", llmCfg.Type, llmCfg.Model)
		} else {
			description = "AI assistant"
		}
	}

	supportsStreaming := agentCfg.Type != "a2a"

	return &pb.AgentCard{
		Name:        agentCfg.Name,
		Description: description,
		Version:     "1.0.0",
		Url:         fmt.Sprintf("local://%s", agentID),
		Capabilities: &pb.AgentCapabilities{
			Streaming: supportsStreaming,
		},
	}
}
