package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirpekel/hector"
)

func main() {
	// Parse command line arguments
	configFile := flag.String("config", "", "YAML configuration file path")
	dynamicReasoning := flag.Bool("dynamic", false, "Enable dynamic reasoning mode")
	streaming := flag.Bool("stream", false, "Enable streaming output")
	flag.Parse()

	// Determine config file to use
	var configPath string
	var err error

	if *configFile != "" {
		configPath = *configFile
	} else if len(flag.Args()) > 0 {
		configName := flag.Args()[0]
		configPath, err = findNamedConfig(configName)
		if err != nil {
			fmt.Printf("Config '%s' not found\n", configName)
			listAvailableConfigs()
			os.Exit(1)
		}
	} else {
		configPath, err = findDefaultConfig()
		if err != nil {
			configPath = "" // Use zero-config
		}
	}

	// Build Agent
	var agent *hector.Agent
	if configPath == "" {
		agent, err = hector.NewAgentWithDefaults()
		if err != nil {
			fmt.Printf("Failed to initialize agent: %v\n", err)
			fmt.Println("Ensure Ollama and Qdrant are running:")
			fmt.Println("  ollama serve")
			fmt.Println("  docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant")
			os.Exit(1)
		}
	} else {
		agent, err = hector.LoadAgentFromFile(configPath)
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// Start interactive session
	startInteractiveChat(agent, *dynamicReasoning, *streaming)
}

// startInteractiveChat starts the interactive chat interface
func startInteractiveChat(agent *hector.Agent, dynamicReasoning bool, streaming bool) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Hector AI Agent")
	if dynamicReasoning {
		fmt.Println("🧠 Dynamic Reasoning Mode Enabled")
		if streaming {
			fmt.Println("📡 Streaming Output Enabled")
		}
	}
	fmt.Println("Type /help for commands or ask a question")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			handleCommand(agent, input, dynamicReasoning, streaming)
			continue
		}

		// Handle queries
		if dynamicReasoning {
			handleDynamicQuery(agent, input, streaming)
		} else {
			handleQuery(agent, input)
		}
	}
}

// handleCommand handles special commands
func handleCommand(agent *hector.Agent, input string, dynamicReasoning bool, streaming bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/help":
		showHelp()
	case "/tools":
		handleListTools(agent)
	case "/quit", "/exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", command)
	}
}

// handleListTools lists available tools from MCP infrastructure
func handleListTools(agent *hector.Agent) {
	mcp := agent.GetMCP()
	toolList := mcp.ListTools()

	if len(toolList) == 0 {
		fmt.Println("No tools available")
		return
	}

	fmt.Println("Available Tools:")
	for i, tool := range toolList {
		fmt.Printf("%d. %s - %s\n", i+1, tool.Name, tool.Description)
	}
}

// handleQuery handles user queries with simple single-shot responses
func handleQuery(agent *hector.Agent, query string) {
	fmt.Print("Processing... ")

	// Use the simple Query method for single-shot responses
	ctx := context.Background()
	response, err := agent.Query(ctx, query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println(response.Answer)

	// Show additional info if available
	if len(response.Context) > 0 {
		fmt.Printf("\n[Used %d context sources]\n", len(response.Context))
	}
	if len(response.ToolResults) > 0 {
		fmt.Printf("[Used %d tools]\n", len(response.ToolResults))
	}
	if response.TokensUsed > 0 {
		fmt.Printf("[Tokens: %d, Duration: %v]\n", response.TokensUsed, response.Duration)
	}
}

// handleDynamicQuery handles user queries with dynamic reasoning
func handleDynamicQuery(agent *hector.Agent, query string, streaming bool) {
	ctx := context.Background()

	// Get reasoning config from agent config or use defaults
	reasoningConfig := hector.ReasoningConfig{
		MaxIterations:        5,
		EnableSelfReflection: true,
		EnableMetaReasoning:  true,
		EnableDynamicTools:   true,
		EnableGoalEvolution:  false,
		QualityThreshold:     0.8,
		StreamingMode:        "all_steps",
	}

	if streaming {
		fmt.Println("🧠 Starting dynamic reasoning...")
		streamCh, err := agent.ExecuteQueryWithReasoningStreaming(ctx, query, reasoningConfig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		for chunk := range streamCh {
			fmt.Print(chunk)
		}
	} else {
		fmt.Print("🧠 Processing with dynamic reasoning... ")

		response, err := agent.ExecuteQueryWithReasoning(ctx, query, reasoningConfig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println()
		fmt.Println(response.Answer)

		// Show additional info
		if len(response.Context) > 0 {
			fmt.Printf("\n[Used %d context sources]\n", len(response.Context))
		}
		if len(response.ToolResults) > 0 {
			fmt.Printf("[Used %d tools]\n", len(response.ToolResults))
		}
		if response.TokensUsed > 0 {
			fmt.Printf("[Tokens: %d, Duration: %v, Confidence: %.2f]\n",
				response.TokensUsed, response.Duration, response.Confidence)
		}
	}
}

// showHelp shows the help message
func showHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help         - Show this help")
	fmt.Println("  /tools        - List available tools")
	fmt.Println("  /quit         - Exit")
	fmt.Println()
	fmt.Println("Command line flags:")
	fmt.Println("  --dynamic     - Enable dynamic reasoning mode")
	fmt.Println("  --stream      - Enable streaming output")
	fmt.Println("  --config      - Specify configuration file")
	fmt.Println()
	fmt.Println("Or just ask questions naturally!")
}

// findDefaultConfig looks for the default configuration file
func findDefaultConfig() (string, error) {
	configPath := "hector.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}
	return "", fmt.Errorf("default config 'hector.yaml' not found")
}

// findNamedConfig looks for named configuration files
func findNamedConfig(name string) (string, error) {
	configPath := fmt.Sprintf("configs/%s.yaml", name)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}
	return "", fmt.Errorf("config '%s' not found in configs/ directory", name)
}

// listAvailableConfigs lists all available configuration files
func listAvailableConfigs() {
	fmt.Println("Available configs:")

	// List default config
	if _, err := os.Stat("hector.yaml"); err == nil {
		fmt.Println("  default (hector.yaml)")
	}

	// List configs directory
	if _, err := os.Stat("configs"); err == nil {
		matches, err := filepath.Glob("configs/*.yaml")
		if err == nil {
			for _, match := range matches {
				baseName := strings.TrimSuffix(filepath.Base(match), ".yaml")
				fmt.Printf("  %s\n", baseName)
			}
		}
	}
}
