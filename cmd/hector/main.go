package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirpekel/hector/agent"
	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/reasoning"
	"github.com/kadirpekel/hector/team"
	"github.com/kadirpekel/hector/workflow"
)

// ============================================================================
// TYPES AND CONSTANTS
// ============================================================================

// CLIArgs holds parsed command line arguments
type CLIArgs struct {
	ConfigFile   string
	WorkflowName string
	AgentName    string
	DebugMode    bool
}

const (
	DefaultConfigFile       = "hector.yaml"
	ConfigsDirectory        = "configs"
	DefaultMaxIterations    = 5
	DefaultQualityThreshold = 0.8
)

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	args := parseCommandLineArgs()

	hectorConfig, err := loadConfiguration(args)
	if err != nil {
		fatalf("Configuration error: %v", err)
	}

	componentManager, err := component.NewComponentManager(hectorConfig)
	if err != nil {
		fatalf("Component initialization failed: %v", err)
	}

	// Initialize document stores if configured
	if err := initializeDocumentStores(hectorConfig, componentManager); err != nil {
		fatalf("Document store initialization failed: %v", err)
	}

	routeExecution(args, hectorConfig, componentManager)
}

// ============================================================================
// DOCUMENT STORE INITIALIZATION
// ============================================================================

// initializeDocumentStores initializes document stores from configuration
func initializeDocumentStores(hectorConfig *config.Config, componentManager *component.ComponentManager) error {
	if len(hectorConfig.DocumentStores) == 0 {
		return nil // No document stores configured
	}

	// Find a database and embedder to use for document stores
	// Look for agents that have document stores configured to get their db/embedder
	var dbName, embedderName string
	for _, agentConfig := range hectorConfig.Agents {
		if len(agentConfig.DocumentStores) > 0 {
			dbName = agentConfig.Database
			embedderName = agentConfig.Embedder
			break
		}
	}

	// Fallback to defaults if not found
	if dbName == "" {
		// Try to find any configured database
		for name := range hectorConfig.Databases {
			dbName = name
			break
		}
	}
	if embedderName == "" {
		// Try to find any configured embedder
		for name := range hectorConfig.Embedders {
			embedderName = name
			break
		}
	}

	if dbName == "" || embedderName == "" {
		return fmt.Errorf("document stores require a database and embedder to be configured")
	}

	// Get database and embedder from component manager
	db, err := componentManager.GetDatabase(dbName)
	if err != nil {
		return fmt.Errorf("failed to get database '%s': %w", dbName, err)
	}

	embedder, err := componentManager.GetEmbedder(embedderName)
	if err != nil {
		return fmt.Errorf("failed to get embedder '%s': %w", embedderName, err)
	}

	// Create search engine with default config
	searchConfig := config.SearchConfig{}
	searchConfig.SetDefaults()
	searchEngine, err := hectorcontext.NewSearchEngine(db, embedder, searchConfig)
	if err != nil {
		return fmt.Errorf("failed to create search engine: %w", err)
	}

	// Convert document stores map to slice
	docStores := make([]config.DocumentStoreConfig, 0, len(hectorConfig.DocumentStores))
	for _, storeConfig := range hectorConfig.DocumentStores {
		docStores = append(docStores, storeConfig)
	}

	// Initialize document stores (this indexes synchronously and waits for completion)
	err = hectorcontext.InitializeDocumentStoresFromConfig(docStores, searchEngine)
	if err != nil {
		return fmt.Errorf("failed to initialize document stores: %w", err)
	}
	return nil
}

// ============================================================================
// COMMAND LINE & CONFIGURATION
// ============================================================================

// parseCommandLineArgs parses and returns command line arguments
func parseCommandLineArgs() *CLIArgs {
	args := &CLIArgs{}

	flag.StringVar(&args.ConfigFile, "config", "", "YAML configuration file path")
	flag.StringVar(&args.WorkflowName, "workflow", "", "Workflow name to execute from config")
	flag.StringVar(&args.AgentName, "agent", "", "Agent name to use (defaults to first agent)")
	flag.BoolVar(&args.DebugMode, "debug", false, "Show technical details and debug info")
	flag.Parse()

	return args
}

// loadConfiguration loads configuration based on CLI arguments
func loadConfiguration(args *CLIArgs) (*config.Config, error) {
	configPath, err := determineConfigPath(args)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// Use default configuration with built-in defaults
		hectorConfig := &config.Config{}
		hectorConfig.SetDefaults()
		return hectorConfig, nil
	}

	hectorConfig, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	return hectorConfig, nil
}

// determineConfigPath determines which configuration file to use
func determineConfigPath(args *CLIArgs) (string, error) {
	if args.ConfigFile != "" {
		return args.ConfigFile, nil
	}

	if len(flag.Args()) > 0 {
		configName := flag.Args()[0]
		configPath, err := findNamedConfig(configName)
		if err != nil {
			fmt.Printf("Config '%s' not found\n", configName)
			listAvailableConfigs()
			return "", err
		}
		return configPath, nil
	}

	configPath, err := findDefaultConfig()
	if err != nil {
		return "", nil // Use zero-config
	}

	return configPath, nil
}

// routeExecution routes to the appropriate execution mode
func routeExecution(args *CLIArgs, hectorConfig *config.Config, componentManager *component.ComponentManager) {
	if args.WorkflowName != "" {
		executeWorkflowMode(hectorConfig, args.WorkflowName, args.DebugMode, componentManager)
	} else {
		executeSingleAgentMode(hectorConfig, args.AgentName, args.DebugMode, componentManager)
	}
}

// ============================================================================
// EXECUTION MODES
// ============================================================================

// executeSingleAgentMode runs a single agent in interactive mode
func executeSingleAgentMode(hectorConfig *config.Config, agentName string, debugMode bool, componentManager *component.ComponentManager) {
	selectedAgent := selectAgent(hectorConfig, agentName)

	agentConfig := hectorConfig.Agents[selectedAgent]
	agentInstance, err := agent.NewAgent(&agentConfig, componentManager)
	if err != nil {
		fatalf("Failed to create agent '%s': %v", selectedAgent, err)
	}

	if debugMode {
		printAgentInfo(&agentConfig, hectorConfig)
	}

	startInteractiveChat(agentInstance, debugMode)
}

// executeWorkflowMode runs a workflow from the unified configuration
func executeWorkflowMode(hectorConfig *config.Config, workflowName string, debugMode bool, componentManager *component.ComponentManager) {
	workflow := selectWorkflow(hectorConfig, workflowName)

	printWorkflowHeader(workflow, debugMode)

	team := createTeamFromWorkflow(workflow, hectorConfig, componentManager)

	initializeTeam(team, debugMode)

	input := getUserInput()

	executeWorkflowAndPrintResults(team, input, debugMode)
}

// ============================================================================
// INTERACTIVE CHAT INTERFACE
// ============================================================================

// startInteractiveChat starts the interactive chat interface
func startInteractiveChat(agentInstance *agent.Agent, debugMode bool) {
	scanner := bufio.NewScanner(os.Stdin)

	printChatWelcome(debugMode)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			handleCommand(agentInstance, input)
			continue
		}

		handleQuery(agentInstance, input, debugMode)
	}
}

// handleCommand handles special commands
func handleCommand(agentInstance *agent.Agent, input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	switch command {
	case "/help":
		showHelp()
	case "/quit", "/exit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", command)
	}
}

// handleQuery handles user queries with AI reasoning
func handleQuery(agentInstance *agent.Agent, query string, debugMode bool) {
	ctx := context.Background()

	// Always use streaming based on config
	handleStreamingQuery(agentInstance, ctx, query, debugMode)
}

// ============================================================================
// WORKFLOW EXECUTION HELPERS
// ============================================================================

// selectAgent determines which agent to use
func selectAgent(hectorConfig *config.Config, agentName string) string {
	if agentName != "" {
		if _, exists := hectorConfig.Agents[agentName]; !exists {
			fmt.Printf("Agent '%s' not found in configuration\n", agentName)
			listAvailableAgents(hectorConfig)
			os.Exit(1)
		}
		return agentName
	}

	// Use first available agent
	for name := range hectorConfig.Agents {
		return name
	}

	fatalf("No agents defined in configuration")
	return "" // unreachable
}

// selectWorkflow finds and returns the specified workflow
func selectWorkflow(hectorConfig *config.Config, workflowName string) *config.WorkflowConfig {
	workflow, exists := hectorConfig.Workflows[workflowName]
	if !exists {
		fmt.Printf("Workflow '%s' not found in configuration\n", workflowName)
		listAvailableWorkflows(hectorConfig)
		os.Exit(1)
	}
	return &workflow
}

// createTeamFromWorkflow creates a team from workflow configuration
func createTeamFromWorkflow(workflow *config.WorkflowConfig, globalConfig *config.Config, componentManager *component.ComponentManager) *team.Team {
	teamInstance, err := team.NewTeam(workflow, globalConfig, componentManager)
	if err != nil {
		fatalf("Error creating team: %v", err)
	}
	return teamInstance
}

// initializeTeam initializes the team
func initializeTeam(teamInstance *team.Team, debugMode bool) {
	ctx := context.Background()
	if err := teamInstance.Initialize(ctx); err != nil {
		fatalf("Failed to initialize team: %v", err)
	}

	// Check for any errors during initialization
	if errors := teamInstance.GetErrors(); len(errors) > 0 {
		fmt.Println("⚠️  Errors during team initialization:")
		for _, err := range errors {
			fmt.Printf("   - %v\n", err)
		}
		fatalf("Team initialization had %d error(s)", len(errors))
	}

	if debugMode {
		fmt.Println("✅ Team initialized successfully")
		fmt.Println()
	}
}

// getUserInput gets input from user or pipe
func getUserInput() string {
	var input string

	if isInputFromPipe() {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fatalf("Error reading input: %v", err)
		}
	} else {
		fmt.Print("💬 Enter your request: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
	}

	if strings.TrimSpace(input) == "" {
		fatalf("No input provided")
	}

	return input
}

// executeWorkflowAndPrintResults executes workflow with streaming and prints results
func executeWorkflowAndPrintResults(teamInstance *team.Team, input string, debugMode bool) {
	ctx := context.Background()
	eventCh, err := teamInstance.ExecuteStreaming(ctx, input)

	if err != nil {
		fmt.Printf("❌ Workflow failed to start: %v\n", err)
		os.Exit(1)
	}

	// Stream workflow events in real-time
	for event := range eventCh {
		switch event.EventType {
		case workflow.EventWorkflowStart:
			fmt.Printf("\n%s\n", event.Content)
		case workflow.EventAgentStart:
			fmt.Printf("\n%s\n", event.Content)
		case workflow.EventAgentOutput:
			fmt.Print(event.Content)
		case workflow.EventAgentComplete:
			fmt.Printf("\n%s\n", event.Content)
		case workflow.EventProgress:
			if debugMode && event.Progress != nil {
				fmt.Printf("\n📊 Progress: %.1f%% (%d/%d steps)\n",
					event.Progress.PercentComplete,
					event.Progress.CompletedSteps,
					event.Progress.TotalSteps)
			}
		case workflow.EventAgentError:
			fmt.Printf("\n❌ Error: %s\n", event.Content)
		case workflow.EventWorkflowEnd:
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("\n%s\n", event.Content)
			if debugMode && event.Metadata != nil {
				fmt.Println("\nWorkflow Metadata:")
				for k, v := range event.Metadata {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
		}
	}

	fmt.Println()
}

// ============================================================================
// QUERY HANDLING HELPERS
// ============================================================================

// handleStreamingQuery handles streaming queries
func handleStreamingQuery(agentInstance *agent.Agent, ctx context.Context, query string, debugMode bool) {
	streamCh, err := agentInstance.QueryStreaming(ctx, query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Stream the response
	for chunk := range streamCh {
		fmt.Print(chunk)
	}

	// Ensure we end with a newline for proper prompt formatting
	fmt.Println()
}

// ============================================================================
// PRINTING AND DISPLAY HELPERS
// ============================================================================

// printChatWelcome prints the welcome message for interactive chat
func printChatWelcome(debugMode bool) {
	fmt.Println("AI Agent")
	if debugMode {
		fmt.Println("🐛 Debug Mode: Technical details enabled")
	}
	fmt.Println("📡 Streaming: Real-time reasoning")
	fmt.Println("Type /help for commands or ask a question")
	fmt.Println()
}

// printAgentInfo prints agent information in debug mode
func printAgentInfo(agentConfig *config.AgentConfig, hectorConfig *config.Config) {
	fmt.Printf("🤖 Using agent: %s (%s)\n", agentConfig.Name, agentConfig.Description)
	fmt.Printf("📋 Configuration: %s\n", hectorConfig.Name)
	fmt.Println()
}

// printWorkflowHeader prints workflow header information
func printWorkflowHeader(workflow *config.WorkflowConfig, debugMode bool) {
	// Header is now printed by the workflow itself (team/team.go)
	// Removed duplicate header to clean up output
}

// printWorkflowResults prints workflow execution results
func printWorkflowResults(result *team.WorkflowResult, debugMode bool) {
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

// printQueryDebugInfo prints debug information for query responses
func printQueryDebugInfo(response *reasoning.ReasoningResponse, debugMode bool) {
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

// printWorkflowDebugInfo prints detailed workflow execution information
func printWorkflowDebugInfo(result *team.WorkflowResult) {
	fmt.Println("🔍 Debug Information:")
	fmt.Printf("   Status: %s\n", result.Status)

	if len(result.Results) > 0 {
		fmt.Println("   Step Results:")
		for stepName, stepResult := range result.Results {
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

	// Print shared context information
	fmt.Println("   Shared Context:")
	if len(result.SharedContext.Variables) > 0 {
		fmt.Println("     Variables:")
		for key, value := range result.SharedContext.Variables {
			fmt.Printf("       %s: %s\n", key, value)
		}
	}
	if len(result.SharedContext.Metadata) > 0 {
		fmt.Println("     Metadata:")
		for key, value := range result.SharedContext.Metadata {
			fmt.Printf("       %s: %s\n", key, value)
		}
	}
	if len(result.SharedContext.Artifacts) > 0 {
		fmt.Println("     Artifacts:")
		for key, artifact := range result.SharedContext.Artifacts {
			fmt.Printf("       %s: %s (%d bytes)\n", key, artifact.Type, artifact.Size)
		}
	}

	fmt.Println()
}

// ============================================================================
// COMMAND HANDLING
// ============================================================================

// showHelp shows the help message
func showHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help         - Show this help")
	fmt.Println("  /quit         - Exit")
	fmt.Println()
	fmt.Println("Command line flags:")
	fmt.Println("  --config FILE       - Specify configuration file")
	fmt.Println("  --agent NAME        - Use specific agent (defaults to first)")
	fmt.Println("  --workflow NAME     - Execute workflow instead of single agent")
	fmt.Println("  --debug             - Show technical details and debug info")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hector                           # Single agent with streaming")
	fmt.Println("  hector --agent main-agent        # Use specific agent")
	fmt.Println("  hector --workflow research-flow  # Execute workflow")
	fmt.Println("  hector --debug                   # Show technical details")
	fmt.Println()
	fmt.Println("Or just ask questions naturally!")
}

// ============================================================================
// CONFIGURATION DISCOVERY
// ============================================================================

// findDefaultConfig looks for the default configuration file
func findDefaultConfig() (string, error) {
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		return DefaultConfigFile, nil
	}
	return "", fmt.Errorf("default config '%s' not found", DefaultConfigFile)
}

// findNamedConfig looks for named configuration files
func findNamedConfig(name string) (string, error) {
	configPath := fmt.Sprintf("%s/%s.yaml", ConfigsDirectory, name)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}
	return "", fmt.Errorf("config '%s' not found in %s/ directory", name, ConfigsDirectory)
}

// listAvailableConfigs lists all available configuration files
func listAvailableConfigs() {
	fmt.Println("Available configs:")

	// List default config
	if _, err := os.Stat(DefaultConfigFile); err == nil {
		fmt.Printf("  default (%s)\n", DefaultConfigFile)
	}

	// List configs directory
	if _, err := os.Stat(ConfigsDirectory); err == nil {
		matches, err := filepath.Glob(ConfigsDirectory + "/*.yaml")
		if err == nil {
			for _, match := range matches {
				baseName := strings.TrimSuffix(filepath.Base(match), ".yaml")
				fmt.Printf("  %s\n", baseName)
			}
		}
	}
}

// listAvailableAgents lists all available agents in the configuration
func listAvailableAgents(hectorConfig *config.Config) {
	fmt.Println("Available agents:")
	for name, agentConfig := range hectorConfig.Agents {
		fmt.Printf("  %s - %s\n", name, agentConfig.Description)
	}
}

// listAvailableWorkflows lists all available workflows in the configuration
func listAvailableWorkflows(hectorConfig *config.Config) {
	fmt.Println("Available workflows:")
	for name, workflow := range hectorConfig.Workflows {
		fmt.Printf("  %s - %s\n", name, workflow.Description)
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// fatalf prints an error message and exits the program
func fatalf(format string, args ...interface{}) {
	fmt.Printf("Error: "+format+"\n", args...)
	os.Exit(1)
}

// isInputFromPipe checks if input is coming from a pipe
func isInputFromPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
