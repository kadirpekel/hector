package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kadirpekel/hector"
)

func main() {
	// Parse command line arguments
	configFile := flag.String("config", "", "YAML configuration file path")
	flag.Parse()

	// Determine config file to use
	var configPath string
	var err error

	if *configFile != "" {
		// Explicit config file provided
		configPath = *configFile
	} else if len(flag.Args()) > 0 {
		// Named config provided as argument
		configName := flag.Args()[0]
		configPath, err = findNamedConfig(configName)
		if err != nil {
			fmt.Printf("Failed to find config '%s': %v\n", configName, err)
			fmt.Println("Available configs:")
			listAvailableConfigs()
			os.Exit(1)
		}
	} else {
		// Look for default config, or use zero-config if none found
		configPath, err = findDefaultConfig()
		if err != nil {
			fmt.Println("No config file found. Starting with zero configuration...")
			fmt.Println("   Assumes Ollama (localhost:11434) and Qdrant (localhost:6334) are running")
			configPath = "" // Signal for zero-config
		}
	}

	fmt.Printf("Config: %s\n", configPath)
	fmt.Println()

	// Build Agent from YAML config or zero-config
	var agent *hector.Agent
	if configPath == "" {
		// Zero-config startup
		agent, err = hector.NewAgentWithDefaults()
		if err != nil {
			fmt.Printf("Failed to create agent with defaults: %v\n", err)
			fmt.Println("Make sure Ollama and Qdrant are running:")
			fmt.Println("  ollama serve")
			fmt.Println("  docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant")
			os.Exit(1)
		}
		fmt.Println("Agent created with zero configuration")
	} else {
		// Config file startup
		agent, err = hector.NewAgentFromYAML(configPath)
		if err != nil {
			fmt.Printf("Failed to load agent from config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Agent created from config: %s\n", configPath)
	}

	fmt.Println("Hector AI Agent system initialized successfully!")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  /help       - Show this help")
	fmt.Println("  /add        - Add a document")
	fmt.Println("  /search     - Search documents")
	fmt.Println("  /tools      - List available tools")
	fmt.Println("  /quit       - Exit")
	fmt.Println()
	fmt.Println("Start chatting! Ask any questions.")
	fmt.Println("Note: All queries use agent capabilities (tools, memory, conversation history)")
	fmt.Println()
	fmt.Println("Tip: Add documents with /add for context-aware responses!")
	fmt.Println()

	// Start interactive chat
	startInteractiveChat(agent)
}

// startInteractiveChat starts the interactive chat interface
func startInteractiveChat(agent *hector.Agent) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("💬 You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			handleCommand(agent, input)
			continue
		}

		// Handle all queries as agent queries (unified experience)
		handleAgentQuery(agent, []string{input})
	}
}

// handleCommand handles special commands
func handleCommand(h *hector.Agent, input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/help":
		showHelp()
	case "/add":
		handleAddDocument(h, parts[1:])
	case "/search":
		handleSearchDocuments(h, parts[1:])
	case "/tools":
		handleListTools(h)
	case "/quit", "/exit":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("❓ Unknown command: %s. Type /help for available commands.\n", command)
	}
}

// handleAddDocument handles document addition
func handleAddDocument(h *hector.Agent, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: /add <title> <content>")
		return
	}

	title := args[0]
	content := strings.Join(args[1:], " ")

	doc := map[string]interface{}{
		"id":      fmt.Sprintf("doc-%d", time.Now().Unix()),
		"title":   title,
		"content": content,
		"source":  "manual-input",
	}

	err := h.UpsertDocument("document", doc["id"].(string), doc)
	if err != nil {
		fmt.Printf("Failed to add document: %v\n", err)
		return
	}

	fmt.Printf("Added document: %s\n", title)
}

// handleSearchDocuments handles document search
func handleSearchDocuments(h *hector.Agent, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: /search <query>")
		return
	}

	query := strings.Join(args, " ")

	results, err := h.SearchDocuments(query, "document", 5)
	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
		return
	}

	fmt.Printf("Found %d documents:\n", len(results))
	for i, result := range results {
		title := "Unknown"
		if t, exists := result.Metadata["title"]; exists {
			if titleStr, ok := t.(string); ok {
				title = titleStr
			}
		}
		fmt.Printf("  %d. %s (score: %.3f)\n", i+1, title, result.Score)
	}
	fmt.Println()
}

// handleListTools lists available tools from MCP infrastructure
func handleListTools(h *hector.Agent) {
	mcp := h.GetMCP()
	toolList := mcp.ListTools()

	fmt.Println("Available Tools:")
	fmt.Println("==================")

	if len(toolList) == 0 {
		fmt.Println("No tools discovered from MCP servers.")
		return
	}

	for i, tool := range toolList {
		fmt.Printf("%d. %s\n", i+1, tool.Name)
		fmt.Printf("   Description: %s\n", tool.Description)

		params := tool.Parameters
		if len(params) > 0 {
			fmt.Printf("   Parameters:\n")
			for _, param := range params {
				required := ""
				if param.Required {
					required = " (required)"
				}
				fmt.Printf("     - %s (%s): %s%s\n", param.Name, param.Type, param.Description, required)
			}
		}

		fmt.Printf("   Capabilities: make requests\n")
		fmt.Println()
	}
}

// handleAgentQuery handles agent queries with tool usage (now used for all queries)
func handleAgentQuery(h *hector.Agent, args []string) {
	if len(args) == 0 {
		fmt.Println("Please provide a query")
		fmt.Println("Usage: Just type your question naturally!")
		fmt.Println("Example: what is Go programming?")
		return
	}

	query := strings.Join(args, " ")
	fmt.Println("🤖 Hector: Thinking...")

	// Try streaming first, fall back to regular if not supported
	if err := handleStreamingAgentQuery(h, query); err != nil {
		// Fall back to regular agent query
		handleRegularAgentQuery(h, query)
	}
}

// handleStreamingAgentQuery handles streaming agent queries
func handleStreamingAgentQuery(h *hector.Agent, query string) error {
	fmt.Printf("🔍 handleStreamingAgentQuery called with query: %s\n", query)

	// Show loading indicator
	showLoadingIndicator("Analyzing query")

	// Clear loading indicator before reasoning to show step-by-step output
	clearLoadingIndicator()

	// Start streaming with reasoning
	fmt.Printf("🔍 About to call ExecuteQueryWithReasoning\n")
	response, err := h.ExecuteQueryWithReasoning(query)
	if err != nil {
		return err
	}

	// Show response
	fmt.Printf("🤖 Hector: %s\n", response.Answer)

	if len(response.ToolResults) > 0 {
		fmt.Println("\nTool Results:")
		for toolName, result := range response.ToolResults {
			fmt.Printf("  %s: %s\n", toolName, result.Content)
		}
	}

	if len(response.Sources) > 0 {
		fmt.Println("\n📚 Sources:")
		for i, source := range response.Sources {
			fmt.Printf("  %d. %s\n", i+1, source)
		}
	}

	if response.Confidence > 0 {
		fmt.Printf("\n🎯 Confidence: %.1f%%\n", response.Confidence*100)
	}

	if response.TokensUsed > 0 {
		fmt.Printf("🔢 Tokens used: %d\n", response.TokensUsed)
	}

	return nil
}

// handleRegularAgentQuery handles regular (non-streaming) agent queries
func handleRegularAgentQuery(h *hector.Agent, query string) {
	fmt.Printf("🔍 handleRegularAgentQuery called with query: %s\n", query)

	// Show loading indicator
	showLoadingIndicator("Processing query")

	// Clear loading indicator before reasoning to show step-by-step output
	clearLoadingIndicator()

	// Execute agent query with reasoning
	fmt.Printf("About to call ExecuteQueryWithReasoning\n")
	response, err := h.ExecuteQueryWithReasoning(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Show response
	fmt.Printf("Hector: %s\n", response.Answer)

	if len(response.ToolResults) > 0 {
		fmt.Println("\nTool Results:")
		for toolName, result := range response.ToolResults {
			fmt.Printf("  %s: %s\n", toolName, result.Content)
			if result.ExecutionTime > 0 {
				fmt.Printf("    (executed in %dms)\n", result.ExecutionTime)
			}
		}
	}

	fmt.Printf("📊 Confidence: %.2f\n", response.Confidence)
	if len(response.Sources) > 0 {
		fmt.Printf("📚 Sources: %s\n", strings.Join(response.Sources, ", "))
	}
	fmt.Println()
}

// showHelp shows the help message
func showHelp() {
	fmt.Println("📚 AI Agent System Commands:")
	fmt.Println("  /help     - Show this help")
	fmt.Println("  /add <title> <content> - Add a document")
	fmt.Println("  /search <query> - Search documents")
	fmt.Println("  /tools    - List available tools")
	fmt.Println("  /quit     - Exit")
	fmt.Println()
	fmt.Println("💬 Or just ask questions naturally!")
	fmt.Println("🤖 All queries use agent capabilities: tools, memory, conversation history")
	fmt.Println()
}

// ============================================================================
// CONFIG DISCOVERY FUNCTIONS
// ============================================================================

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

// showLoadingIndicator shows a loading indicator with a message
func showLoadingIndicator(message string) {
	fmt.Printf("🔄 %s...\r", message)
}

// clearLoadingIndicator clears the loading indicator
func clearLoadingIndicator() {
	fmt.Print("\033[2K\r") // Clear line and return to beginning
}
