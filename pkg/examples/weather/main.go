// Example weather demonstrates using Hector with MCP tools.
//
// Prerequisites:
//   - Set ANTHROPIC_API_KEY environment variable
//   - Set MCP_URL environment variable (or run an MCP weather server)
//
// Run:
//
//	go run ./pkg/examples/weather
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/verikod/hector/pkg/model/anthropic"

	v2 "github.com/verikod/hector/pkg"
	"github.com/verikod/hector/pkg/config"
)

func main() {
	// Load .env file
	config.LoadDotEnv()

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Get MCP URL (optional)
	mcpURL := os.Getenv("MCP_URL")

	// Build options
	opts := []v2.Option{
		v2.WithAnthropic(anthropic.Config{APIKey: apiKey}),
		v2.WithInstruction(`You are a helpful weather assistant.
When asked about weather, use the available tools to get current conditions.
Be concise and friendly in your responses.`),
	}

	// Add MCP tool if URL is provided
	if mcpURL != "" {
		fmt.Printf("Using MCP server at %s\n", mcpURL)
		// Use HTTP transport for Composio-style endpoints
		opts = append(opts, v2.WithMCPToolHTTP("weather", mcpURL))
	} else {
		fmt.Println("Note: MCP_URL not set, running without weather tool")
	}

	// Create Hector instance
	h, err := v2.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create Hector: %v", err)
	}
	defer h.Close()

	// Test prompt
	prompt := "How is the weather like today in Berlin?"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	fmt.Printf("\n📝 Prompt: %s\n", prompt)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Generate response
	ctx := context.Background()
	response, err := h.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Printf("\n🤖 Response:\n%s\n", response)
}
