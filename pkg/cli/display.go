package cli

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// DisplayAgentList displays a formatted list of agents
func DisplayAgentList(agents []client.AgentInfo, mode string) {
	fmt.Printf("\n📋 Available Agents (%s)\n\n", mode)
	fmt.Printf("Found %d agent(s):\n\n", len(agents))

	for _, agent := range agents {
		fmt.Printf("• %s", agent.Name)
		if agent.ID != agent.Name {
			fmt.Printf(" (%s)", agent.ID)
		}
		fmt.Println()

		if agent.Description != "" {
			fmt.Printf("  Description: %s\n", agent.Description)
		}
		if agent.Endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", agent.Endpoint)
		}
		fmt.Println()
	}

	fmt.Println("💡 Use 'hector info <agent>' for detailed information")
	fmt.Println("💡 Use 'hector call <agent> \"prompt\"' to interact with an agent")
}

// DisplayAgentCard displays a formatted agent card
func DisplayAgentCard(agentID string, card *pb.AgentCard) {
	fmt.Printf("\n📋 Agent Information: %s\n\n", agentID)
	fmt.Printf("Name: %s\n", card.Name)
	if card.Description != "" {
		fmt.Printf("Description: %s\n", card.Description)
	}
	if card.Version != "" {
		fmt.Printf("Version: %s\n", card.Version)
	}
	if card.Capabilities != nil {
		fmt.Printf("Streaming: %v\n", card.Capabilities.Streaming)
	}
}

// DisplayMessage displays a message response
func DisplayMessage(msg *pb.Message, prefix string) {
	if msg == nil {
		return
	}

	if prefix != "" {
		fmt.Print(prefix)
	}

	for _, part := range msg.Content {
		if text := part.GetText(); text != "" {
			fmt.Print(text)
		}
	}
}

// DisplayMessageLine displays a message response with newline
func DisplayMessageLine(msg *pb.Message, prefix string) {
	DisplayMessage(msg, prefix)
	fmt.Println()
}

// DisplayTask displays a task response
func DisplayTask(task *pb.Task) {
	fmt.Printf("Task created: %s (status: %s)\n", task.Id, task.Status)
}

// DisplayError displays an error message
func DisplayError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

// DisplayStreamingStart displays a streaming start message
func DisplayStreamingStart(agentID, mode string) {
	streamInfo := ""
	if mode != "" {
		streamInfo = fmt.Sprintf(" (%s)", mode)
	}
	fmt.Printf("\n🤖 Chat with %s%s (streaming) (type 'exit' to quit)\n\n", agentID, streamInfo)
}

// DisplayChatPrompt displays a chat input prompt
func DisplayChatPrompt() {
	fmt.Print("You: ")
}

// DisplayAgentPrompt displays an agent response prompt
func DisplayAgentPrompt(agentID string) {
	fmt.Printf("\n%s: ", agentID)
}

// DisplayGoodbye displays a goodbye message
func DisplayGoodbye() {
	fmt.Println("👋 Goodbye!")
}
