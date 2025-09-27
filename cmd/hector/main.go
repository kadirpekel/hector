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
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/executors"
	"github.com/kadirpekel/hector/interfaces"
)

func main() {
	// Parse command line arguments
	configFile := flag.String("config", "", "YAML configuration file path")
	workflowFile := flag.String("workflow", "", "Multi-agent workflow YAML file path")
	debugMode := flag.Bool("debug", false, "Show technical details and debug info")
	noStreaming := flag.Bool("no-stream", false, "Disable streaming output (streaming is default)")
	flag.Parse()

	// Check if this is a multi-agent workflow execution
	if *workflowFile != "" {
		executeWorkflow(*workflowFile, *debugMode)
		return
	}

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
	streaming := !*noStreaming // Streaming is default, unless disabled
	startInteractiveChat(agent, streaming, *debugMode)
}

// startInteractiveChat starts the interactive chat interface
func startInteractiveChat(agent *hector.Agent, streaming bool, debugMode bool) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Hector AI Agent")
	if debugMode {
		fmt.Println("🐛 Debug Mode: Technical details enabled")
	}
	if streaming {
		fmt.Println("📡 Streaming: Real-time reasoning enabled")
	} else {
		fmt.Println("⚡ Non-streaming: Direct answers")
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
			handleCommand(agent, input, streaming)
			continue
		}

		// Handle queries with AI reasoning
		handleQuery(agent, input, streaming, debugMode)
	}
}

// handleCommand handles special commands
func handleCommand(agent *hector.Agent, input string, streaming bool) {
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

// handleListTools lists available tools from tool registry
func handleListTools(agent *hector.Agent) {
	registry := agent.GetToolRegistryInstance()
	if registry == nil {
		fmt.Println("No tool registry available")
		return
	}

	toolList := registry.ListTools()

	if len(toolList) == 0 {
		fmt.Println("No tools available")
		return
	}

	fmt.Println("Available Tools:")
	for i, tool := range toolList {
		source, _ := registry.GetToolSource(tool.Name)
		fmt.Printf("%d. %s - %s (from %s)\n", i+1, tool.Name, tool.Description, source)
	}
}

// handleDynamicQuery handles user queries with dynamic reasoning
func handleQuery(agent *hector.Agent, query string, streaming bool, debugMode bool) {
	ctx := context.Background()

	if streaming {
		// For streaming, use ExecuteQueryWithReasoningStreaming with debug config
		reasoningConfig := config.ReasoningConfig{
			MaxIterations:        5,
			EnableSelfReflection: true,
			EnableMetaReasoning:  true,
			EnableDynamicTools:   true,
			EnableGoalEvolution:  false,
			QualityThreshold:     0.8,
			ShowDebugInfo:        debugMode,
			EnableStreaming:      true,
		}

		streamCh, err := agent.ExecuteQueryWithReasoningStreaming(ctx, query, reasoningConfig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		for chunk := range streamCh {
			fmt.Print(chunk)
		}
	} else {
		// For non-streaming, show a simple thinking message
		if debugMode {
			fmt.Print("🧠 Processing with AI reasoning... ")
		} else {
			fmt.Print("💭 Let me think about this... ")
		}

		response, err := agent.Query(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println()
		fmt.Println(response.Answer)

		// Show additional info if debug mode or if context was used
		if debugMode || len(response.Context) > 0 || len(response.ToolResults) > 0 {
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
}

// showHelp shows the help message
func showHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help         - Show this help")
	fmt.Println("  /tools        - List available tools")
	fmt.Println("  /quit         - Exit")
	fmt.Println()
	fmt.Println("Command line flags:")
	fmt.Println("  --debug             - Show technical details and debug info")
	fmt.Println("  --no-stream         - Disable streaming output (streaming is default)")
	fmt.Println("  --config FILE       - Specify configuration file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hector                    # Natural AI reasoning with streaming")
	fmt.Println("  hector --debug            # Show technical details and debug info")
	fmt.Println("  hector --no-stream        # Direct answers without streaming")
	fmt.Println("  hector --debug --no-stream # Debug mode without streaming")
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

// executeWorkflow runs a multi-agent workflow
func executeWorkflow(workflowFile string, debugMode bool) {
	fmt.Printf("🤖 Starting Multi-Agent Workflow: %s\n", workflowFile)
	fmt.Println(strings.Repeat("=", 60))

	// Load workflow definition
	workflow, err := hector.LoadWorkflowDefinition(workflowFile)
	if err != nil {
		fmt.Printf("❌ Failed to load workflow: %v\n", err)
		os.Exit(1)
	}

	if debugMode {
		fmt.Printf("📋 Workflow: %s\n", workflow.Name)
		fmt.Printf("📝 Description: %s\n", workflow.Description)
		fmt.Printf("⚙️  Mode: %s\n", workflow.Mode)
		fmt.Printf("👥 Agents: %d types, %d total instances\n",
			len(workflow.Agents), getTotalInstances(workflow.Agents))
		fmt.Println()
	}

	// Create team
	team, err := hector.NewTeam(workflow)
	if err != nil {
		fmt.Printf("Error creating team: %v\n", err)
		os.Exit(1)
	}

	// Initialize executor registry
	executorRegistry := initializeExecutorRegistry()
	if err := team.SetExecutorRegistry(executorRegistry); err != nil {
		fmt.Printf("Error setting executor registry: %v\n", err)
		os.Exit(1)
	}

	// Initialize the team
	ctx := context.Background()
	if err := team.Initialize(ctx); err != nil {
		fmt.Printf("❌ Failed to initialize team: %v\n", err)
		os.Exit(1)
	}

	if debugMode {
		fmt.Println("✅ Team initialized successfully")
		fmt.Println()
	}

	// Get user input
	var input string
	if isInputFromPipe() {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print("💬 Enter your request: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
	}

	if strings.TrimSpace(input) == "" {
		fmt.Println("❌ No input provided")
		os.Exit(1)
	}

	fmt.Printf("🚀 Executing workflow with input: %s\n", input)
	fmt.Println(strings.Repeat("-", 60))

	// Execute workflow
	result, err := team.Execute(ctx, input)

	fmt.Println(strings.Repeat("-", 60))

	if err != nil {
		fmt.Printf("❌ Workflow failed: %v\n", err)
		if debugMode && result != nil {
			printWorkflowDebugInfo(result)
		}
		os.Exit(1)
	}

	// Print results
	fmt.Printf("✅ Workflow completed successfully!\n")
	fmt.Printf("⏱️  Execution time: %v\n", result.ExecutionTime)
	fmt.Printf("🔢 Steps executed: %d\n", result.StepsExecuted)
	fmt.Printf("🤖 Agents used: %s\n", strings.Join(result.AgentsUsed, ", "))
	fmt.Printf("🎯 Total tokens: %d\n", result.TotalTokens)
	fmt.Println()

	if debugMode {
		printWorkflowDebugInfo(result)
	}

	fmt.Println("📄 Final Output:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(result.FinalOutput)
	fmt.Println(strings.Repeat("=", 60))
}

// getTotalInstances calculates total agent instances
func getTotalInstances(agents []string) int {
	// In the new config system, agents are just references
	// For now, assume 1 instance per agent reference
	return len(agents)
}

// printWorkflowDebugInfo prints detailed workflow execution information
func printWorkflowDebugInfo(result *hector.WorkflowResult) {
	fmt.Println("🔍 Debug Information:")
	fmt.Printf("   Status: %s\n", result.Status)

	if len(result.AgentResults) > 0 {
		fmt.Println("   Step Results:")
		for stepName, stepResult := range result.AgentResults {
			status := "✅"
			if !stepResult.Success {
				status = "❌"
			}
			fmt.Printf("     %s %s (%s): %v tokens, %v duration\n",
				status, stepName, stepResult.AgentName,
				stepResult.TokensUsed, stepResult.Duration)

			if stepResult.Error != "" {
				fmt.Printf("       Error: %s\n", stepResult.Error)
			}
		}
	}

	if len(result.SharedContext) > 0 {
		fmt.Println("   Shared Context:")
		for key, value := range result.SharedContext {
			if str, ok := value.(string); ok && len(str) < 100 {
				fmt.Printf("     %s: %s\n", key, str)
			} else {
				fmt.Printf("     %s: [%T]\n", key, value)
			}
		}
	}

	fmt.Println()
}

// isInputFromPipe checks if input is coming from a pipe
func isInputFromPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// initializeExecutorRegistry sets up the executor registry with default executors
func initializeExecutorRegistry() interfaces.ExecutorRegistry {
	return executors.InitializeDefaultExecutors()
}
