---
title: Extend Hector with Programmatic API
description: Build custom reasoning engines, tools, and components using Hector's programmatic API
---

# Extend Hector with Programmatic API

Hector is designed to be extended. Import it as a Go library and build custom reasoning engines, tools, memory strategies, and more.

**Time:** 30-60 minutes  
**Difficulty:** Intermediate

---

## What You'll Learn

- Import Hector as a Go library
- Create custom reasoning engines
- Build custom tools
- Implement custom memory strategies
- Integrate custom components into agents
- Best practices for extension development

---

## Why Extend Hector?

Common reasons to extend Hector:

1. **Custom Reasoning Engines** - Implement domain-specific reasoning strategies
2. **Custom Tools** - Build tools specific to your use case
3. **Custom Memory Strategies** - Create specialized memory management
4. **Custom LLM Providers** - Integrate proprietary or specialized LLMs
5. **Domain-Specific Agents** - Build agents tailored to your domain

---

## Getting Started

### Import Hector

```go
package main

import (
    "github.com/kadirpekel/hector/pkg/hector"
    "github.com/kadirpekel/hector/pkg/agent"
    "github.com/kadirpekel/hector/pkg/reasoning"
    "github.com/kadirpekel/hector/pkg/tools"
)
```

### Basic Agent with Custom Components

```go
// Build agent with custom components
agent, err := hector.NewAgent("my-agent").
    WithName("Custom Agent").
    WithLLMProvider(llm).
    WithReasoningStrategy(customReasoning).  // Your custom reasoning
    WithTools(customTool).                   // Your custom tool
    Build()
```

---

## Example 1: Custom Reasoning Engine

Create a custom reasoning engine that implements domain-specific logic.

### Step 1: Implement ReasoningStrategy Interface

```go
package main

import (
    "github.com/kadirpekel/hector/pkg/protocol"
    "github.com/kadirpekel/hector/pkg/reasoning"
)

// TreeSearchStrategy implements a tree-of-thoughts reasoning approach
type TreeSearchStrategy struct {
    maxDepth int
    branchingFactor int
}

func NewTreeSearchStrategy(maxDepth, branchingFactor int) *TreeSearchStrategy {
    return &TreeSearchStrategy{
        maxDepth: maxDepth,
        branchingFactor: branchingFactor,
    }
}

// GetName returns the strategy name
func (s *TreeSearchStrategy) GetName() string {
    return "tree-search"
}

// GetDescription returns a description
func (s *TreeSearchStrategy) GetDescription() string {
    return "Tree-of-thoughts reasoning with configurable depth and branching"
}

// PrepareIteration prepares the iteration
func (s *TreeSearchStrategy) PrepareIteration(iteration int, state *reasoning.ReasoningState) error {
    // Initialize tree search state
    if iteration == 1 {
        state.GetCustomState()["tree_depth"] = 0
        state.GetCustomState()["branches"] = make([]string, 0)
    }
    return nil
}

// ShouldStop determines if reasoning should stop
func (s *TreeSearchStrategy) ShouldStop(
    text string,
    toolCalls []*protocol.ToolCall,
    state *reasoning.ReasoningState,
) bool {
    // Stop if no tool calls and we have a final answer
    if len(toolCalls) == 0 && text != "" {
        return true
    }
    
    // Stop if we've exceeded max depth
    if depth, ok := state.GetCustomState()["tree_depth"].(int); ok {
        if depth >= s.maxDepth {
            return true
        }
    }
    
    return false
}

// AfterIteration processes results after each iteration
func (s *TreeSearchStrategy) AfterIteration(
    iteration int,
    text string,
    toolCalls []*protocol.ToolCall,
    results []reasoning.ToolResult,
    state *reasoning.ReasoningState,
) error {
    // Update tree depth
    if depth, ok := state.GetCustomState()["tree_depth"].(int); ok {
        state.GetCustomState()["tree_depth"] = depth + 1
    }
    
    // Track branches explored
    branches, _ := state.GetCustomState()["branches"].([]string)
    branches = append(branches, text)
    state.GetCustomState()["branches"] = branches
    
    return nil
}

// GetContextInjection returns context to inject into prompts
func (s *TreeSearchStrategy) GetContextInjection(state *reasoning.ReasoningState) string {
    branches, ok := state.GetCustomState()["branches"].([]string)
    if !ok || len(branches) == 0 {
        return ""
    }
    
    // Inject tree search context
    return "Current reasoning branches explored:\n" + 
        strings.Join(branches[len(branches)-s.branchingFactor:], "\n")
}

// GetPromptSlots returns custom prompt templates
func (s *TreeSearchStrategy) GetPromptSlots() reasoning.PromptSlots {
    return reasoning.PromptSlots{
        SystemRole: `You are an AI agent using tree-of-thoughts reasoning.
            Explore multiple reasoning paths before converging on a solution.
            Consider ${branchingFactor} alternative approaches at each step.`,
        Instructions: `
            <tree_search>
                - Generate ${branchingFactor} alternative reasoning paths
                - Evaluate each path before proceeding
                - Converge on the best path when confident
            </tree_search>
        `,
        UserGuidance: "",
    }
}

// GetRequiredTools returns tools required by this strategy
func (s *TreeSearchStrategy) GetRequiredTools() []reasoning.RequiredTool {
    return []reasoning.RequiredTool{
        {
            Name:        "todo_write",
            Type:        "todo",
            Description: "Required for tracking reasoning branches",
            AutoCreate:  true,
        },
    }
}
```

### Step 2: Use Custom Reasoning Engine

```go
func main() {
    // Build LLM provider
    llm, err := hector.NewLLMProvider("openai").
        Model("gpt-4o").
        APIKeyFromEnv("OPENAI_API_KEY").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Create custom reasoning engine
    customReasoning := NewTreeSearchStrategy(maxDepth: 5, branchingFactor: 3)

    // Build agent with custom reasoning
    agent, err := hector.NewAgent("tree-search-agent").
        WithName("Tree Search Agent").
        WithLLMProvider(llm).
        WithReasoningStrategy(customReasoning).
        WithSystemPrompt("You are an expert problem solver using tree-of-thoughts.").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Use the agent
    runtime, err := runtime.NewRuntimeBuilder().
        WithAgent(agent).
        Start()
    if err != nil {
        log.Fatal(err)
    }
    defer runtime.Close()
}
```

---

## Example 2: Custom Tool

Create a custom tool that implements domain-specific functionality.

### Step 1: Implement Tool Interface

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/kadirpekel/hector/pkg/tools"
)

// DatabaseQueryTool allows agents to query databases
type DatabaseQueryTool struct {
    dbConnection string
}

func NewDatabaseQueryTool(dbConnection string) *DatabaseQueryTool {
    return &DatabaseQueryTool{
        dbConnection: dbConnection,
    }
}

// GetName returns the tool name
func (t *DatabaseQueryTool) GetName() string {
    return "database_query"
}

// GetDescription returns tool description
func (t *DatabaseQueryTool) GetDescription() string {
    return "Execute SQL queries against the database. Use for data retrieval and analysis."
}

// GetInfo returns tool metadata
func (t *DatabaseQueryTool) GetInfo() tools.ToolInfo {
    return tools.ToolInfo{
        Name:        t.GetName(),
        Description: t.GetDescription(),
        Parameters: []tools.ToolParameter{
            {
                Name:        "query",
                Type:        "string",
                Description: "SQL query to execute (SELECT only, no DML/DDL)",
                Required:    true,
            },
            {
                Name:        "limit",
                Type:        "integer",
                Description: "Maximum number of rows to return (default: 100)",
                Required:    false,
                Default:     100,
            },
        },
    }
}

// Execute runs the tool
func (t *DatabaseQueryTool) Execute(
    ctx context.Context,
    args map[string]interface{},
) (tools.ToolResult, error) {
    startTime := time.Now()

    // Extract arguments
    query, ok := args["query"].(string)
    if !ok || query == "" {
        return tools.ToolResult{
            Success:  false,
            Error:    "query parameter is required",
            ToolName: t.GetName(),
        }, fmt.Errorf("query parameter is required")
    }

    limit := 100
    if l, ok := args["limit"].(float64); ok {
        limit = int(l)
    }

    // Validate query (only SELECT allowed)
    if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
        return tools.ToolResult{
            Success:  false,
            Error:    "only SELECT queries are allowed",
            ToolName: t.GetName(),
        }, fmt.Errorf("only SELECT queries are allowed")
    }

    // Execute query (pseudo-code - implement with your DB driver)
    results, err := t.executeQuery(ctx, query, limit)
    if err != nil {
        return tools.ToolResult{
            Success:       false,
            Error:         err.Error(),
            ToolName:      t.GetName(),
            ExecutionTime: time.Since(startTime),
        }, err
    }

    // Format results as JSON
    jsonResults, err := json.Marshal(results)
    if err != nil {
        return tools.ToolResult{
            Success:       false,
            Error:         "failed to serialize results",
            ToolName:      t.GetName(),
            ExecutionTime: time.Since(startTime),
        }, err
    }

    return tools.ToolResult{
        Success:       true,
        Content:       string(jsonResults),
        ToolName:      t.GetName(),
        ExecutionTime: time.Since(startTime),
        Metadata: map[string]interface{}{
            "rows_returned": len(results),
            "query":         query,
        },
    }, nil
}

// executeQuery executes the SQL query (implement with your DB driver)
func (t *DatabaseQueryTool) executeQuery(
    ctx context.Context,
    query string,
    limit int,
) ([]map[string]interface{}, error) {
    // TODO: Implement actual database query
    // Example:
    // db, err := sql.Open("postgres", t.dbConnection)
    // rows, err := db.QueryContext(ctx, query + " LIMIT $1", limit)
    // ... process rows ...
    
    // Mock implementation
    return []map[string]interface{}{
        {"id": 1, "name": "Example", "value": 42},
    }, nil
}
```

### Step 2: Use Custom Tool

```go
func main() {
    // Build LLM provider
    llm, err := hector.NewLLMProvider("openai").
        Model("gpt-4o").
        APIKeyFromEnv("OPENAI_API_KEY").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Create custom tool
    dbTool := NewDatabaseQueryTool("postgres://user:pass@localhost/db")

    // Build agent with custom tool
    agent, err := hector.NewAgent("data-analyst").
        WithName("Data Analyst").
        WithLLMProvider(llm).
        WithReasoningStrategy(reasoning).
        WithTools(dbTool).
        WithSystemPrompt("You are a data analyst. Use database_query to analyze data.").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Use the agent
    runtime, err := runtime.NewRuntimeBuilder().
        WithAgent(agent).
        Start()
    if err != nil {
        log.Fatal(err)
    }
    defer runtime.Close()
}
```

---

## Example 3: Custom Memory Strategy

Create a custom working memory strategy.

### Step 1: Implement WorkingMemoryStrategy Interface

```go
package main

import (
    "fmt"
    "sort"
    
    "github.com/kadirpekel/hector/pkg/memory"
    "github.com/kadirpekel/hector/pkg/a2a/pb"
    hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

// PriorityBufferMemory keeps most important messages based on priority scores
type PriorityBufferMemory struct {
    maxSize int
    priorities map[string]float64
    notifier memory.StatusNotifier
}

func NewPriorityBufferMemory(maxSize int) *PriorityBufferMemory {
    return &PriorityBufferMemory{
        maxSize: maxSize,
        priorities: make(map[string]float64),
    }
}

// Name returns the strategy name
func (m *PriorityBufferMemory) Name() string {
    return "priority_buffer"
}

// AddMessage adds a message with priority
func (m *PriorityBufferMemory) AddMessage(
    session *hectorcontext.ConversationHistory,
    message *pb.Message,
) error {
    // Calculate priority (example: based on message length, role, etc.)
    priority := m.calculatePriority(message)
    
    msgID := generateMessageID(message)
    m.priorities[msgID] = priority
    
    // Add to session
    session.Messages = append(session.Messages, message)
    
    // Keep only top priority messages
    return m.trimToMaxSize(session)
}

// CheckAndSummarize checks if summarization is needed
func (m *PriorityBufferMemory) CheckAndSummarize(
    session *hectorcontext.ConversationHistory,
) ([]*pb.Message, error) {
    // Custom summarization logic based on priority
    // Return messages to summarize
    return []*pb.Message{}, nil
}

// GetMessages returns messages sorted by priority
func (m *PriorityBufferMemory) GetMessages(
    session *hectorcontext.ConversationHistory,
) ([]*pb.Message, error) {
    // Sort messages by priority and return top maxSize
    messages := make([]*pb.Message, len(session.Messages))
    copy(messages, session.Messages)
    
    // Sort by priority (implement sorting logic)
    sort.Slice(messages, func(i, j int) bool {
        idI := generateMessageID(messages[i])
        idJ := generateMessageID(messages[j])
        return m.priorities[idI] > m.priorities[idJ]
    })
    
    if len(messages) > m.maxSize {
        return messages[:m.maxSize], nil
    }
    
    return messages, nil
}

// SetStatusNotifier sets the status notifier
func (m *PriorityBufferMemory) SetStatusNotifier(notifier memory.StatusNotifier) {
    m.notifier = notifier
}

// LoadState loads state from session service
func (m *PriorityBufferMemory) LoadState(
    sessionID string,
    sessionService interface{},
) (*hectorcontext.ConversationHistory, error) {
    // Load conversation history from session service
    // Implementation depends on your session service
    return &hectorcontext.ConversationHistory{
        Messages: make([]*pb.Message, 0),
    }, nil
}

// Helper methods
func (m *PriorityBufferMemory) calculatePriority(msg *pb.Message) float64 {
    // Example: prioritize longer messages and assistant responses
    priority := float64(len(msg.Parts)) * 0.1
    if msg.Role == "assistant" {
        priority += 0.5
    }
    return priority
}

func (m *PriorityBufferMemory) trimToMaxSize(
    session *hectorcontext.ConversationHistory,
) error {
    if len(session.Messages) <= m.maxSize {
        return nil
    }
    
    // Sort and keep top priority messages
    messages, _ := m.GetMessages(session)
    session.Messages = messages
    return nil
}

func generateMessageID(msg *pb.Message) string {
    // Generate unique ID for message
    return fmt.Sprintf("%s-%d", msg.Role, len(msg.Parts))
}
```

### Step 2: Use Custom Memory Strategy

```go
func main() {
    // Build LLM provider
    llm, err := hector.NewLLMProvider("openai").
        Model("gpt-4o").
        APIKeyFromEnv("OPENAI_API_KEY").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // Create custom memory strategy
    customMemory := NewPriorityBufferMemory(maxSize: 20)

    // Build agent with custom memory
    // Note: You'll need to use agent.NewAgentDirect for custom memory strategies
    // as the builder currently supports built-in strategies
    agent, err := agent.NewAgentDirect(agent.AgentBuilderOptions{
        ID: "custom-memory-agent",
        Name: "Custom Memory Agent",
        LLMProvider: llm,
        ReasoningStrategy: reasoning,
        WorkingMemory: customMemory,  // Your custom strategy
        SystemPrompt: "You are an agent with priority-based memory.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Use the agent
    runtime, err := runtime.NewRuntimeBuilder().
        WithAgent(agent).
        Start()
    if err != nil {
        log.Fatal(err)
    }
    defer runtime.Close()
}
```

---

## Example 4: Complete Custom Agent

Build a complete agent with multiple custom components:

```go
package main

import (
    "log"
    
    "github.com/kadirpekel/hector/pkg/hector"
    "github.com/kadirpekel/hector/pkg/runtime"
    "github.com/kadirpekel/hector/pkg/reasoning"
)

func main() {
    // 1. Build LLM provider
    llm, err := hector.NewLLMProvider("openai").
        Model("gpt-4o").
        APIKeyFromEnv("OPENAI_API_KEY").
        Temperature(0.7).
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create custom reasoning engine
    customReasoning := NewTreeSearchStrategy(maxDepth: 5, branchingFactor: 3)

    // 3. Create custom tools
    dbTool := NewDatabaseQueryTool("postgres://localhost/db")
    customTool := NewCustomDomainTool()

    // 4. Build working memory (using built-in)
    workingMemory, err := hector.NewWorkingMemory("summary_buffer").
        Budget(8000).      // Default: 8000
        Threshold(0.85).    // Default: 0.85
        WithLLMProvider(llm).
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // 5. Build agent with all custom components
    agent, err := hector.NewAgent("custom-agent").
        WithName("Custom Domain Agent").
        WithDescription("Agent with custom reasoning and tools").
        WithLLMProvider(llm).
        WithReasoningStrategy(customReasoning).
        WithWorkingMemory(workingMemory).
        WithTools(dbTool, customTool).
        WithSystemPrompt("You are a specialized domain expert.").
        Build()
    if err != nil {
        log.Fatal(err)
    }

    // 6. Create runtime
    rt, err := runtime.NewRuntimeBuilder().
        WithAgent(agent).
        Start()
    if err != nil {
        log.Fatal(err)
    }
    defer rt.Close()

    // 7. Use the agent
    // ... interact with agent via runtime ...
}
```

---

## Best Practices

### 1. Interface Compliance

Always implement interfaces completely:

```go
// ✅ Good: Implements all interface methods
type MyStrategy struct {}
func (s *MyStrategy) GetName() string { return "my-strategy" }
func (s *MyStrategy) GetDescription() string { return "..." }
func (s *MyStrategy) PrepareIteration(...) error { return nil }
// ... implement all methods

// ❌ Bad: Missing methods
type MyStrategy struct {}
func (s *MyStrategy) GetName() string { return "my-strategy" }
// Missing other required methods - won't compile
```

### 2. Error Handling

Always return proper errors:

```go
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
    // Validate inputs
    if arg, ok := args["required_arg"].(string); !ok || arg == "" {
        return tools.ToolResult{
            Success:  false,
            Error:    "required_arg is required",
            ToolName: t.GetName(),
        }, fmt.Errorf("required_arg is required")
    }
    
    // Handle errors gracefully
    result, err := doWork(arg)
    if err != nil {
        return tools.ToolResult{
            Success:  false,
            Error:    err.Error(),
            ToolName: t.GetName(),
        }, err
    }
    
    return tools.ToolResult{
        Success:  true,
        Content:  result,
        ToolName: t.GetName(),
    }, nil
}
```

### 3. State Management

Use ReasoningState for custom state:

```go
func (s *MyStrategy) PrepareIteration(iteration int, state *reasoning.ReasoningState) error {
    // Initialize custom state
    if iteration == 1 {
        state.GetCustomState()["my_key"] = initialValue
    }
    
    // Access existing state
    if value, ok := state.GetCustomState()["my_key"]; ok {
        // Use value
    }
    
    return nil
}
```

### 4. Testing Custom Components

Test your custom components:

```go
func TestCustomReasoning(t *testing.T) {
    strategy := NewTreeSearchStrategy(5, 3)
    
    assert.Equal(t, "tree-search", strategy.GetName())
    assert.NotEmpty(t, strategy.GetDescription())
    
    state := reasoning.NewReasoningState()
    err := strategy.PrepareIteration(1, state)
    assert.NoError(t, err)
    
    shouldStop := strategy.ShouldStop("answer", nil, state)
    assert.True(t, shouldStop)
}
```

### 5. Documentation

Document your custom components:

```go
// TreeSearchStrategy implements tree-of-thoughts reasoning.
//
// It explores multiple reasoning paths before converging on a solution.
// Configure with maxDepth (maximum tree depth) and branchingFactor
// (number of branches to explore at each level).
//
// Example:
//   strategy := NewTreeSearchStrategy(maxDepth: 5, branchingFactor: 3)
//   agent := hector.NewAgent("agent").WithReasoningStrategy(strategy)
type TreeSearchStrategy struct {
    maxDepth int
    branchingFactor int
}
```

---

## Integration Patterns

### Pattern 1: Wrapper Around Existing Strategy

```go
// EnhancedChainOfThought wraps ChainOfThought with additional features
type EnhancedChainOfThought struct {
    *reasoning.ChainOfThoughtStrategy
    customFeature bool
}

func NewEnhancedChainOfThought() *EnhancedChainOfThought {
    return &EnhancedChainOfThought{
        ChainOfThoughtStrategy: reasoning.NewChainOfThoughtStrategy(),
        customFeature: true,
    }
}

// Override specific methods
func (s *EnhancedChainOfThought) PrepareIteration(iteration int, state *reasoning.ReasoningState) error {
    // Custom logic
    if s.customFeature {
        // Do something custom
    }
    
    // Call parent
    return s.ChainOfThoughtStrategy.PrepareIteration(iteration, state)
}
```

### Pattern 2: Composition

```go
// MultiStageReasoning composes multiple strategies
type MultiStageReasoning struct {
    stage1 reasoning.ReasoningStrategy
    stage2 reasoning.ReasoningStrategy
    currentStage int
}

func (s *MultiStageReasoning) PrepareIteration(iteration int, state *reasoning.ReasoningState) error {
    if s.currentStage == 1 {
        return s.stage1.PrepareIteration(iteration, state)
    }
    return s.stage2.PrepareIteration(iteration, state)
}
```

---

## Real-World Example: Domain-Specific Agent

Build a financial analysis agent with custom components:

```go
package main

import (
    "github.com/kadirpekel/hector/pkg/hector"
    "github.com/kadirpekel/hector/pkg/runtime"
)

func BuildFinancialAgent() (*agent.Agent, error) {
    // Custom reasoning for financial analysis
    financialReasoning := NewFinancialAnalysisStrategy()
    
    // Custom tools
    marketDataTool := NewMarketDataTool(apiKey)
    portfolioTool := NewPortfolioAnalysisTool(dbConnection)
    riskTool := NewRiskAnalysisTool()
    
    // Build agent
    agent, err := hector.NewAgent("financial-analyst").
        WithName("Financial Analyst").
        WithLLMProvider(llm).
        WithReasoningStrategy(financialReasoning).
        WithTools(marketDataTool, portfolioTool, riskTool).
        WithSystemPrompt("You are a financial analyst. Analyze markets and portfolios.").
        Build()
    
    return agent, err
}
```

---

## Summary

Hector's programmatic API enables you to:

✅ **Import as Library** - Use Hector in your Go applications  
✅ **Custom Reasoning Engines** - Implement domain-specific reasoning strategies  
✅ **Custom Tools** - Build tools for your specific use cases  
✅ **Custom Memory Strategies** - Create specialized memory management  
✅ **Complete Control** - Build agents exactly as you need them  

### Key Interfaces

- **`reasoning.ReasoningStrategy`** - For custom reasoning engines
- **`tools.Tool`** - For custom tools
- **`memory.WorkingMemoryStrategy`** - For custom working memory
- **`memory.LongTermMemoryStrategy`** - For custom long-term memory
- **`llms.LLMProvider`** - For custom LLM providers (advanced)

### When to Extend vs Use Built-in

**Use Built-in Components When:**
- Standard reasoning (chain-of-thought, supervisor) fits your needs
- Built-in tools (file operations, search, etc.) are sufficient
- Standard memory strategies work for your use case

**Extend with Custom Components When:**
- You need domain-specific reasoning logic
- You require specialized tools not available built-in
- You need custom memory management strategies
- You're building a domain-specific agent platform

---

## Next Steps

- **[Programmatic API Reference](../reference/programmatic-api.md)** - Complete API documentation
- **[Programmatic API Guide](../core-concepts/programmatic-api.md)** - Core concepts and patterns
- **[Add Custom Tools](add-custom-tools.md)** - MCP-based tool development
- **[Plugin Development](plugins.md)** - gRPC plugins for advanced extensions

---

## Related Topics

- **[Reasoning Strategies](../core-concepts/reasoning.md)** - Understanding reasoning engines
- **[Tools System](../core-concepts/tools.md)** - Tool architecture
- **[Memory Management](../core-concepts/memory.md)** - Memory strategies
- **[Agent Architecture](../reference/architecture/agent-lifecycle.md)** - How agents work internally

