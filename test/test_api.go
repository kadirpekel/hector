package main

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector"
	"github.com/kadirpekel/hector/providers"
)

func main() {
	fmt.Println("Testing GPT-5 Nano API call...")

	// Register providers
	if err := providers.RegisterDefaultProviders(); err != nil {
		log.Fatalf("Failed to register providers: %v", err)
	}

	// Create agent from config
	agent, err := hector.NewAgentFromYAML("../local-configs/openai-minimal.yaml")
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	fmt.Println("Agent created successfully!")
	fmt.Println("Making API call...")

	// Test query
	query := "Hello! What model are you using? Please respond briefly."
	fmt.Printf("Query: %s\n", query)
	fmt.Println("---")

	response, err := agent.ExecuteQueryWithReasoning(query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("Response: %s\n", response)
	fmt.Println("---")
	fmt.Println("Test completed successfully!")
}
