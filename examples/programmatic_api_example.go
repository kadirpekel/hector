package main

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/hector"
	"github.com/kadirpekel/hector/pkg/runtime"
	"github.com/kadirpekel/hector/pkg/tools"
)

func main() {
	// Example 1: Pure programmatic API - build agent from scratch
	fmt.Println("=== Example 1: Pure Programmatic API ===")

	// Build LLM provider
	llm, err := hector.NewLLMProvider("openai").
		Model("gpt-4o-mini").
		APIKeyFromEnv("OPENAI_API_KEY").
		Temperature(0.7).
		Build()
	if err != nil {
		fmt.Printf("Error building LLM: %v\n", err)
		return
	}

	// Build reasoning strategy
	reasoning, err := hector.NewReasoning("chain-of-thought").Build()
	if err != nil {
		fmt.Printf("Error building reasoning: %v\n", err)
		return
	}

	// Build working memory
	workingMemory, err := hector.NewWorkingMemory("summary_buffer").
		Budget(2000).
		Threshold(0.8).
		Target(0.6).
		WithLLMProvider(llm).
		Build()
	if err != nil {
		fmt.Printf("Error building working memory: %v\n", err)
		return
	}

	// Build agent
	agent, err := hector.NewAgent("assistant").
		WithName("Assistant").
		WithDescription("A helpful AI assistant").
		WithLLMProvider(llm).
		WithReasoningStrategy(reasoning).
		WithWorkingMemory(workingMemory).
		WithSystemPrompt("You are a helpful assistant.").
		WithTools(
			tools.NewFileWriterTool(nil),
			tools.NewReadFileTool(nil),
		).
		Build()
	if err != nil {
		fmt.Printf("Error building agent: %v\n", err)
		return
	}

	fmt.Printf("Built agent: %s (%s)\n", agent.GetID(), agent.GetName())

	// Example 2: Config bridge - build agent from config using programmatic API
	fmt.Println("\n=== Example 2: Config Bridge ===")

	// Load config
	cfg, err := config.LoadConfig(config.LoaderOptions{
		Path: "configs/weather-assistant.yaml",
	})
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Build agents from config using programmatic API
	builder, err := hector.NewConfigAgentBuilder(cfg)
	if err != nil {
		fmt.Printf("Error creating config builder: %v\n", err)
		return
	}

	agents, err := builder.BuildAllAgents()
	if err != nil {
		fmt.Printf("Error building agents: %v\n", err)
		return
	}

	fmt.Printf("Built %d agents from config\n", len(agents))
	for id := range agents {
		fmt.Printf("  - %s\n", id)
	}

	// Example 3: Build runtime programmatically (combining config and programmatic agents)
	fmt.Println("\n=== Example 3: Runtime Builder (Combined) ===")

	rt, err := runtime.NewRuntimeBuilder().
		WithAgents(agents). // From config (uses programmatic API internally)
		WithAgent(agent).   // Programmatic
		Start()
	if err != nil {
		fmt.Printf("Error building runtime: %v\n", err)
		return
	}

	fmt.Printf("Runtime created with %d agents\n", len(agents)+1)
	defer rt.Close()
}
