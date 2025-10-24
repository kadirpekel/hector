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

func ListCommand(args *ListCmd, cfg *config.Config, mode CLIMode) error {
	switch mode {
	case ModeClient:

		httpClient := runtime.NewHTTPClient(args.Server, args.Token)
		defer httpClient.Close()

		agents, err := httpClient.ListAgents(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		DisplayAgentList(agents, "Client Mode")
		return nil

	case ModeLocalConfig:

		agents := buildAgentCardsFromConfig(cfg)
		DisplayAgentList(agents, "Local Mode")
		return nil

	case ModeLocalZeroConfig:

		fmt.Println("\n📋 Available Agents (Local Mode)")
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

func InfoCommand(args *InfoCmd, cfg *config.Config, mode CLIMode) error {
	switch mode {
	case ModeClient:

		httpClient := runtime.NewHTTPClient(args.Server, args.Token)
		defer httpClient.Close()

		card, err := httpClient.GetAgentCard(context.Background(), args.Agent)
		if err != nil {
			return fmt.Errorf("failed to get agent card: %w", err)
		}

		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalConfig:

		if err := cfg.ValidateAgent(args.Agent); err != nil {
			return err
		}

		card := buildAgentCardFromConfig(cfg, args.Agent)
		DisplayAgentCard(args.Agent, card)
		return nil

	case ModeLocalZeroConfig:

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

	var agentID string
	if mode == ModeClient {

		if args.Agent == "" {
			return fmt.Errorf("agent name is required in client mode (use --agent flag)")
		}
		agentID = args.Agent
	} else if mode == ModeLocalConfig {

		if args.Agent == "" {

			agentNames := make([]string, 0, len(cfg.Agents))
			for name := range cfg.Agents {
				agentNames = append(agentNames, name)
			}
			return fmt.Errorf("agent name is required when using --config flag. Available agents: %v", agentNames)
		}
		agentID = args.Agent
	} else {

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
		Content: []*pb.Part{
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
				DisplayMessage(msgChunk, "")
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
			DisplayMessageLine(respMsg, "Agent: ")
		} else if task := resp.GetTask(); task != nil {
			DisplayTask(task)
		}
		return nil
	}
}

func ChatCommand(args *ChatCmd, cfg *config.Config, mode CLIMode) error {

	var agentID string
	if mode == ModeClient {

		if args.Agent == "" {
			return fmt.Errorf("agent name is required in client mode (use --agent flag)")
		}
		agentID = args.Agent
	} else if mode == ModeLocalConfig {

		if args.Agent == "" {

			agentNames := make([]string, 0, len(cfg.Agents))
			for name := range cfg.Agents {
				agentNames = append(agentNames, name)
			}
			return fmt.Errorf("agent name is required when using --config flag. Available agents: %v", agentNames)
		}
		agentID = args.Agent
	} else {

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
			Content: []*pb.Part{
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
					DisplayMessage(msgChunk, "")
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
				DisplayMessageLine(respMsg, "")
			} else if task := resp.GetTask(); task != nil {
				DisplayTask(task)
			}
		}
	}

	return nil
}

type ClientArgs interface {
	GetServer() string
	GetToken() string
	GetAgent() string
}

func (c *CallCmd) GetServer() string { return c.Server }
func (c *CallCmd) GetToken() string  { return c.Token }
func (c *CallCmd) GetAgent() string  { return c.Agent }

func (c *ChatCmd) GetServer() string { return c.Server }
func (c *ChatCmd) GetToken() string  { return c.Token }
func (c *ChatCmd) GetAgent() string  { return c.Agent }

func (t *TaskGetCmd) GetServer() string { return CLI.Task.Server }
func (t *TaskGetCmd) GetToken() string  { return CLI.Task.Token }
func (t *TaskGetCmd) GetAgent() string  { return t.Agent }

func (t *TaskCancelCmd) GetServer() string { return CLI.Task.Server }
func (t *TaskCancelCmd) GetToken() string  { return CLI.Task.Token }
func (t *TaskCancelCmd) GetAgent() string  { return t.Agent }

func createClient[T ClientArgs](args T, cfg *config.Config, mode CLIMode) (client.A2AClient, error) {
	switch mode {
	case ModeClient:

		return runtime.NewHTTPClient(args.GetServer(), args.GetToken()), nil

	case ModeLocalConfig:

		return runtime.NewWithConfig(cfg)
	case ModeLocalZeroConfig:
		return runtime.NewWithConfig(cfg)

	default:
		return nil, fmt.Errorf("unsupported mode for client creation: %s", mode)
	}
}

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
