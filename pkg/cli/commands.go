package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/runtime"
)

// ListCommand lists all available agents
func ListCommand(args Args) error {
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
func InfoCommand(args Args) error {
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
func CallCommand(args Args) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// Create message
	msg := &pb.Message{
		Role: pb.Role_ROLE_USER,
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
func ChatCommand(args Args) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// Display welcome message
	mode := "Local Mode"
	if args.ServerURL != "" {
		mode = "Server Mode"
	}

	streamInfo := ""
	if args.Stream {
		streamInfo = " (streaming)"
	}

	fmt.Printf("\n🤖 Chat with %s (%s)%s (type 'exit' to quit)\n\n", args.AgentID, mode, streamInfo)

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

		// Create message
		msg := &pb.Message{
			Role: pb.Role_ROLE_USER,
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

// createClient creates the appropriate A2A client (HTTP or Direct) based on args
func createClient(args Args) (client.A2AClient, error) {
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	return rt.Client(), nil
}
