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
		agent, err = hector.NewAgentFromYAML(configPath)
		if err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// Start interactive session
	startInteractiveChat(agent)
}

// startInteractiveChat starts the interactive chat interface
func startInteractiveChat(agent *hector.Agent) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Hector AI Agent")
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
			handleCommand(agent, input)
			continue
		}

		// Handle queries
		handleQuery(agent, input)
	}
}

// handleCommand handles special commands
func handleCommand(agent *hector.Agent, input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/help":
		showHelp()
	case "/add":
		handleAddDocument(agent, parts[1:])
	case "/search":
		handleSearchDocuments(agent, parts[1:])
	case "/tools":
		handleListTools(agent)
	case "/quit", "/exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", command)
	}
}

// handleAddDocument handles document addition
func handleAddDocument(agent *hector.Agent, args []string) {
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

	err := agent.UpsertDocument("document", doc["id"].(string), doc)
	if err != nil {
		fmt.Printf("Failed to add document: %v\n", err)
		return
	}

	fmt.Printf("Added document: %s\n", title)
}

// handleSearchDocuments handles document search
func handleSearchDocuments(agent *hector.Agent, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: /search <query>")
		return
	}

	query := strings.Join(args, " ")
	results, err := agent.SearchDocuments(query, "document", 5)
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

// handleQuery handles user queries
func handleQuery(agent *hector.Agent, query string) {
	fmt.Print("Processing... ")

	// Try streaming first, fall back to regular if not supported
	if err := handleStreamingQuery(agent, query); err != nil {
		handleRegularQuery(agent, query)
	}
}

// handleStreamingQuery handles streaming queries
func handleStreamingQuery(agent *hector.Agent, query string) error {
	streamCh, err := agent.ExecuteQueryWithReasoningStreaming(query)
	if err != nil {
		return err
	}

	fmt.Println()
	for chunk := range streamCh {
		fmt.Print(chunk)
	}
	fmt.Println()
	return nil
}

// handleRegularQuery handles regular queries
func handleRegularQuery(agent *hector.Agent, query string) {
	response, err := agent.ExecuteQueryWithReasoning(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println(response.Answer)
}

// showHelp shows the help message
func showHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help     - Show this help")
	fmt.Println("  /add      - Add a document")
	fmt.Println("  /search   - Search documents")
	fmt.Println("  /tools    - List available tools")
	fmt.Println("  /quit     - Exit")
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
