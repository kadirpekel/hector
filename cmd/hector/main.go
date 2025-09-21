package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
	case "/sync-model":
		handleSyncModel(agent, parts[1:])
	case "/sync-all":
		handleSyncAllModels(agent)
	case "/list-models":
		handleListModels(agent)
	case "/model-status":
		handleModelStatus(agent, parts[1:])
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
		"id":      uuid.New().String(),
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
		fmt.Println("Usage: /search <query> [model_name]")
		return
	}

	query := strings.Join(args, " ")

	// Determine which model to search
	modelName := "document" // Default
	if len(args) > 1 {
		// If multiple args, last one might be model name
		models := agent.ListModels()
		if len(models) > 0 {
			// Check if last arg is a model name
			lastArg := args[len(args)-1]
			for _, model := range models {
				if model == lastArg {
					modelName = lastArg
					query = strings.Join(args[:len(args)-1], " ")
					break
				}
			}
		}
	} else {
		// Use first available model if no model specified
		models := agent.ListModels()
		if len(models) > 0 {
			modelName = models[0]
		}
	}

	results, err := agent.SearchDocuments(query, modelName, 5)
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
		} else if filename, exists := result.Metadata["filename"]; exists {
			if filenameStr, ok := filename.(string); ok {
				title = filenameStr
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
		// Force flush to ensure real-time output
		os.Stdout.Sync()
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
	fmt.Println("  /help         - Show this help")
	fmt.Println("  /add          - Add a document")
	fmt.Println("  /search       - Search documents")
	fmt.Println("  /tools        - List available tools")
	fmt.Println("  /sync-model   - Sync a specific model")
	fmt.Println("  /sync-all     - Sync all models")
	fmt.Println("  /list-models  - List all models")
	fmt.Println("  /model-status - Show model status")
	fmt.Println("  /quit         - Exit")
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

// ============================================================================
// MODEL SYNC COMMAND HANDLERS
// ============================================================================

// handleSyncModel handles model synchronization
func handleSyncModel(agent *hector.Agent, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: /sync-model <model_name>")
		return
	}

	modelName := args[0]
	fmt.Printf("Syncing model '%s'...\n", modelName)

	err := agent.SyncModel(modelName)
	if err != nil {
		fmt.Printf("Failed to sync model '%s': %v\n", modelName, err)
	} else {
		fmt.Printf("Successfully synced model '%s'\n", modelName)
	}
}

// handleSyncAllModels handles syncing all models
func handleSyncAllModels(agent *hector.Agent) {
	fmt.Println("Syncing all models...")

	err := agent.SyncAllModels()
	if err != nil {
		fmt.Printf("Failed to sync all models: %v\n", err)
	} else {
		fmt.Println("Successfully synced all models")
	}
}

// handleListModels handles listing all models
func handleListModels(agent *hector.Agent) {
	models := agent.ListModels()

	if len(models) == 0 {
		fmt.Println("No models configured")
		return
	}

	fmt.Println("Available models:")
	for _, model := range models {
		fmt.Printf("  - %s\n", model)
	}
}

// handleModelStatus handles showing model status
func handleModelStatus(agent *hector.Agent, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: /model-status <model_name>")
		return
	}

	modelName := args[0]
	status, err := agent.GetModelStatus(modelName)
	if err != nil {
		fmt.Printf("Failed to get status for model '%s': %v\n", modelName, err)
		return
	}

	fmt.Printf("Model Status: %s\n", modelName)
	for key, value := range status {
		fmt.Printf("  %s: %v\n", key, value)
	}
}
