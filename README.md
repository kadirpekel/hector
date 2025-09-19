# Hector

```
 ██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
 ██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
 ███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
 ██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
 ██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
 ╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Declarative AI Agent Framework - Multi-step reasoning with integrated components**

Hector is a declarative AI agent framework that combines LLM providers, vector databases, MCP tools, and embedders into multi-step reasoning workflows. Define agent hierarchies through YAML configuration - from simple single-step agents to nested systems where each reasoning step contains its own specialized sub-agent.

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

> **Alpha Version**: Hector is currently in alpha. Core features work, but we're actively exploring and experimenting with the framework. Expect API changes as we refine the approach. Perfect for early adopters who want to experiment with declarative multi-step AI agents.

## Value Proposition

**Why Hector?** Unlike LangChain (Python-focused) or CrewAI (limited configurability), Hector offers:

- **Pure Declarative Approach**: Define entire agent workflows in YAML - no Python/JavaScript required
- **True Multi-Step Reasoning**: Sequential task decomposition with specialized sub-agents, not just tool chaining
- **Integrated Components**: LLM + VectorDB + MCP + Embedders work together seamlessly
- **Nested Agent Hierarchies**: Sub-agents can contain their own reasoning processes (unique capability)
- **Hot-Swappable Providers**: Switch between Ollama, OpenAI, TGI without code changes

Perfect for teams who want sophisticated AI agents without complex programming.

## Use Cases

### Research & Analysis
- **Market Research Agent**: Multi-step analysis with data collection → analysis → report generation
- **Document Analysis**: Extract insights from large document collections with specialized sub-agents
- **Competitive Intelligence**: Automated research workflows with nested reasoning steps

### Customer Support
- **Tiered Support Agent**: Route → analyze → escalate with specialized agents for each step
- **Knowledge Base Assistant**: Search → synthesize → respond with context-aware memory
- **Technical Troubleshooting**: Diagnose → test → resolve with tool integration

### Content & Operations
- **Content Creation Pipeline**: Research → draft → review → publish workflows
- **Data Processing**: Extract → transform → validate → store with specialized agents
- **Quality Assurance**: Test → analyze → report with nested validation agents

### Business Automation
- **Lead Qualification**: Analyze → score → route with specialized business logic
- **Invoice Processing**: Extract → validate → approve with multi-step verification
- **Compliance Monitoring**: Monitor → analyze → alert with regulatory expertise

## Key Features

- **Multi-Step Reasoning**: Sequential workflows with specialized agents for each step
- **Nested Agent Hierarchies**: Sub-agents with their own reasoning processes
- **Declarative Configuration**: Single YAML file defines entire agent workflows
- **Integrated Components**: LLM providers, vector databases, MCP tools, embedders
- **Hot-Swappable Providers**: Ollama, OpenAI, TGI, Qdrant with dynamic configuration
- **MCP Tool Integration**: Generic tool access through Model Context Protocol
- **Conversation Memory**: Persistent chat history and context awareness

## Quick Start

### Install & Run

**Option 1: CLI Binary (Recommended for users)**
```bash
# Install Hector CLI
go install github.com/kadirpekel/hector/cmd/hector@latest

# Run with basic configuration
hector --config configs/basic.yaml

# Or start with minimal setup (requires Ollama + Qdrant)
hector
```

**Option 2: Go Package (For developers)**
```bash
# Add to your Go project
go get github.com/kadirpekel/hector@v1.0.2

# Use in your code
import "github.com/kadirpekel/hector"
```

### Prerequisites
```bash
# For minimal setup mode
ollama serve
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

## Quick Demo

### Multi-Step Reasoning with Tool Integration
```yaml
# MCP tool integration + multi-step reasoning
mcp_servers:
  - name: "composio"
    url: "https://apollo.composio.dev/v3/mcp/..."
    description: "Weather and web search tools"

reasoning:
  strategy: "state_machine"
  steps:
    - name: "analyzer"
      type: "analyze"
      agent_config:
        llm:
          name: "openai"
          model: "gpt-4"
          temperature: 0.3
    
    - name: "executor"
      type: "execute"
      agent_config:
        llm:
          name: "openai"
          model: "gpt-4"
          temperature: 0.7
```

### Nested Agents
```yaml
reasoning:
  steps:
    - name: "research_agent"
      agent_config:
        reasoning:  # This sub-agent has its own reasoning steps
          strategy: "state_machine"
          steps:
            - name: "data_collection"
              type: "execute"
            - name: "analysis"
              type: "analyze"
```

## Configuration Examples

### Provider Configuration
```yaml
llm:
  name: "ollama"
  model: "llama3.2"
  temperature: 0.7

memory:
  name: "qdrant"
  collection: "documents"
  default_top_k: 5

embedder:
  name: "ollama"
  model: "nomic-embed-text"
```

### Conversation Memory & Context
```yaml
# Conversation memory with context preservation
reasoning:
  context:
    preserve_history: true
    max_history_steps: 10
    enable_context_share: true

# Agent-level configuration
agent:
  name: "research_assistant"
  description: "Specialized in domain research"
```

### Available Configurations
- **basic.yaml** - Simple single-step agent
- **tools.yaml** - Agent with MCP tool integration
- **stepped.yaml** - Multi-step reasoning workflow
- **nested.yaml** - Nested agent hierarchies

## Programmatic Usage

For developers who want to integrate Hector into their Go applications:

```bash
go get github.com/kadirpekel/hector@v1.0.2
```

```go
package main

import (
    "fmt"
    "github.com/kadirpekel/hector"
    "github.com/kadirpekel/hector/providers"
)

func main() {
    providers.RegisterDefaultProviders()
    agent, _ := hector.NewAgentWithDefaults()
    response, _ := agent.ExecuteQueryWithReasoning("Hello!")
    fmt.Println(response)
}
```

## Documentation

### Core API
```go
// Agent Creation
func NewAgentWithDefaults() (*Agent, error)
func NewAgentFromYAML(configPath string) (*Agent, error)
func NewAgent() *Agent

// Query Execution
func (a *Agent) ExecuteQueryWithReasoning(query string) (string, error)
func (a *Agent) ExecuteQuery(query string) (string, error)

// Memory Operations
func (a *Agent) AddDocument(content string, metadata map[string]interface{}) error
func (a *Agent) SearchDocuments(query string, topK int) ([]Document, error)
func (a *Agent) GetConversationHistory() []ConversationEntry

// Configuration
func (a *Agent) WithLLMConfig(config YAMLProviderConfig) *Agent
func (a *Agent) WithMemoryConfig(config YAMLProviderConfig) *Agent
func (a *Agent) WithEmbedderConfig(config YAMLProviderConfig) *Agent

// Tool Management
func (a *Agent) GetAvailableTools() []Tool
func (a *Agent) RegisterTool(tool Tool) error
func (a *Agent) ExecuteTool(toolName string, params map[string]interface{}) (interface{}, error)

// Provider Management
func (a *Agent) GetLLMProvider() LLMProvider
func (a *Agent) GetMemoryProvider() VectorDB
func (a *Agent) GetEmbedderProvider() EmbeddingProvider
```

### Configuration Structure
```go
type AgentConfig struct {
    Agent      AgentInfo           `yaml:"agent"`
    LLM        YAMLProviderConfig  `yaml:"llm"`
    Memory     YAMLProviderConfig  `yaml:"memory"`
    Embedder   YAMLProviderConfig  `yaml:"embedder"`
    Reasoning  ReasoningConfig     `yaml:"reasoning"`
}
```

## License

MIT License - see [LICENSE](LICENSE) file for details.