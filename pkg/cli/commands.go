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
	"github.com/kadirpekel/hector/pkg/runtime"
)

// ListCommand lists all available agents
func ListCommand(args *CLIArgs) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// List agents
	agents, err := a2aClient.ListAgents(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	// Display
	mode := "Local Mode"
	if args.ServerURL != "" {
		mode = "Server Mode"
	}
	DisplayAgentList(agents, mode)

	return nil
}

// InfoCommand displays agent information
func InfoCommand(args *CLIArgs) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// Get agent card
	card, err := a2aClient.GetAgentCard(context.Background(), args.AgentID)
	if err != nil {
		return fmt.Errorf("failed to get agent card: %w", err)
	}

	// Display
	DisplayAgentCard(args.AgentID, card)

	return nil
}

// CallCommand sends a single message to an agent
func CallCommand(args *CLIArgs) error {
	// For config mode with agent, validate before expensive initialization
	if args.ConfigFile != "" && args.AgentID != "" {
		a2aClient, _, err := createClientWithValidation(args)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer a2aClient.Close()
		return executeCall(a2aClient, args)
	}

	// For other modes (zero-config, server), use regular client creation
	a2aClient, err := createClient(args)
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
	// For config mode with agent, validate before expensive initialization
	if args.ConfigFile != "" && args.AgentID != "" {
		a2aClient, _, err := createClientWithValidation(args)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer a2aClient.Close()
		return executeChat(a2aClient, args)
	}

	// For other modes (zero-config, server), use regular client creation
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()
	return executeChat(a2aClient, args)
}

func executeChat(a2aClient client.A2AClient, args *CLIArgs) error {

	// Display welcome message
	mode := "Local Mode"
	if args.ServerURL != "" {
		mode = "Server Mode"
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

// createClient creates the appropriate A2A client (HTTP or Local) based on args
func createClient(args *CLIArgs) (client.A2AClient, error) {
	if args.ServerURL != "" {
		// Server mode: use HTTP client
		return runtime.NewHTTPClient(args.ServerURL, args.Token), nil
	}

	// Local mode: use Runtime
	rt, err := runtime.New(runtime.Options{
		ConfigFile: args.ConfigFile,
		Provider:   args.Provider,
		APIKey:     args.APIKey,
		BaseURL:    args.BaseURL,
		Model:      args.Model,
		Tools:      args.Tools,
		MCPURL:     args.MCPURL,
		DocsFolder: args.DocsFolder,
		AgentName:  args.AgentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	return rt.Client(), nil
}

// createClientWithValidation creates a client and validates agent exists first (for config mode)
// This avoids expensive initialization if the agent doesn't exist
func createClientWithValidation(args *CLIArgs) (client.A2AClient, *runtime.Runtime, error) {
	if args.ServerURL != "" {
		// Server mode: use HTTP client (no validation needed)
		return runtime.NewHTTPClient(args.ServerURL, args.Token), nil, nil
	}

	// Local mode: load config first to validate agent before expensive initialization
	cfg, err := runtime.LoadConfigForValidation(args.ConfigFile, runtime.Options{
		Provider:   args.Provider,
		APIKey:     args.APIKey,
		BaseURL:    args.BaseURL,
		Model:      args.Model,
		Tools:      args.Tools,
		MCPURL:     args.MCPURL,
		DocsFolder: args.DocsFolder,
		AgentName:  args.AgentID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate agent exists before expensive initialization
	if err := cfg.ValidateAgent(args.AgentID); err != nil {
		return nil, nil, err
	}

	// Now create runtime with validated config
	rt, err := runtime.NewWithConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	return rt.Client(), rt, nil
}
