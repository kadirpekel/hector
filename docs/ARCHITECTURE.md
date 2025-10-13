# Hector Architecture

**100% A2A Protocol Native • Multi-Transport • Production-Ready**

---

## Table of Contents

- [Overview](#overview)
- [A2A Protocol Native Architecture](#a2a-protocol-native-architecture)
- [Transport Layer](#transport-layer)
- [Client Architecture](#client-architecture)
- [Server Architecture](#server-architecture)
- [Runtime System](#runtime-system)
- [Core Components](#core-components)
- [Multi-Agent Orchestration](#multi-agent-orchestration)
- [Security & Authentication](#security--authentication)
- [Extension Points](#extension-points)

---

## Overview

Hector is a **100% A2A Protocol Native** AI agent platform built from the ground up with the [Agent-to-Agent (A2A) Protocol](https://a2a-protocol.org/) at its core.

### Design Principles

1. **Protocol Native**: Every component speaks A2A natively using protobuf types - zero abstraction layers
2. **Multi-Transport**: gRPC (native), REST (grpc-gateway), JSON-RPC (custom adapter)
3. **Clean Architecture**: Clear separation of concerns (transport, runtime, client, server, agents)
4. **Interface-Based**: Dependency injection and strategy patterns throughout
5. **Production-Ready**: Authentication, discovery, streaming, task management

### Key Features

- ✅ **100% Protobuf-Based**: All message types use `pb.*` (protobuf) directly
- ✅ **Multi-Transport**: Single codebase, three protocols (gRPC, REST, JSON-RPC)
- ✅ **Spec-Compliant**: Fully compliant with [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- ✅ **Discovery**: RFC 8615 `.well-known` endpoints for agent discovery
- ✅ **Authentication**: JWT-based security with configurable schemes
- ✅ **Streaming**: Real-time response streaming via gRPC streams and SSE
- ✅ **Task Management**: Async task processing with status tracking
- ✅ **External Agents**: Native support for calling remote A2A agents

---

## A2A Protocol Native Architecture

Hector is built with **genuine A2A-native architecture**. Unlike platforms that add A2A as an afterthought, Hector uses A2A protocol types directly throughout the entire stack.

### What Makes Hector A2A Native?

```go
// ✅ NATIVE: Direct use of protobuf types everywhere
import "github.com/kadirpekel/hector/pkg/a2a/pb"

// Agent interface uses pb.* types directly
func (a *Agent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error)
func (a *Agent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error
func (a *Agent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error)

// ❌ NOT NATIVE: Internal types with conversion layers
type InternalMessage struct { ... }
func convertToA2A(internal InternalMessage) A2AMessage { ... } // Extra layer!
```

### Complete A2A Stack

```
┌──────────────────────────────────────────────────────────────┐
│                      APPLICATION                             │
│                 (Your Agents & Logic)                        │
├──────────────────────────────────────────────────────────────┤
│                     HECTOR RUNTIME                           │
│  • Configuration Loading  • Agent Initialization             │
│  • Component Management   • Lifecycle Management             │
├──────────────────────────────────────────────────────────────┤
│                      CLIENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │         A2AClient Interface (Protocol Native)           ││
│  ├─────────────────────────────────────────────────────────┤│
│  │  HTTPClient           │          DirectClient           ││
│  │  • Remote agents      │          • In-process agents    ││
│  │  • Uses protojson     │          • No network calls     ││
│  │  • Multi-transport    │          • Direct protobuf      ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                      TRANSPORT LAYER                         │
│  ┌──────────────┬──────────────────┬─────────────────────┐ │
│  │  gRPC (Core) │  REST (Gateway)  │  JSON-RPC (Adapter) │ │
│  │  • Native    │  • Auto-gen      │  • Custom HTTP      │ │
│  │  • Binary    │  • JSON          │  • Simple RPC       │ │
│  │  • Streaming │  • SSE           │  • JSON             │ │
│  └──────────────┴──────────────────┴─────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                      SERVER LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │            RegistryService (Multi-Agent Hub)            ││
│  │  • Agent registration    • Request routing              ││
│  │  • Metadata management   • Discovery endpoints          ││
│  │  • Authentication        • Well-known endpoints         ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                       AGENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Agent (pb.A2AServiceServer interface)                  ││
│  │  • SendMessage          • GetAgentCard                  ││
│  │  • SendStreamingMessage • GetTask/CancelTask            ││
│  │  • Task subscriptions   • Push notifications            ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                        CORE SERVICES                         │
│  ┌───────────┬──────────┬──────────┬──────────┬──────────┐ │
│  │    LLM    │   Tools  │   Memory │    RAG   │   Tasks  │ │
│  │  • OpenAI │ • Local  │ • Buffer │ • Qdrant │ • Async  │ │
│  │• Anthropic│ • MCP    │ • Summary│ • Search │ • Status │ │
│  │  • Gemini │ • Plugin │ • Session│ • Embed  │ • Track  │ │
│  └───────────┴──────────┴──────────┴──────────┴──────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### Protobuf Message Flow

Every message in Hector uses protobuf types:

```go
// 1. Client creates message using pb types
message := &pb.Message{
    Role: pb.Role_ROLE_USER,
    Content: []*pb.Part{
        {Part: &pb.Part_Text{Text: "Hello"}},
    },
}

// 2. Serialized with protojson for HTTP/REST
jsonData, _ := protojson.Marshal(&pb.SendMessageRequest{
    Request: message,
})

// 3. Server receives and processes as pb types
func (a *Agent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
    // req.Request is already *pb.Message
    // No conversion needed!
}

// 4. Response uses pb types
return &pb.SendMessageResponse{
    Payload: &pb.SendMessageResponse_Msg{
        Msg: &pb.Message{
            Role: pb.Role_ROLE_AGENT,
            Content: []*pb.Part{
                {Part: &pb.Part_Text{Text: "Hi!"}},
            },
        },
    },
}, nil
```

---

## Transport Layer

Hector provides three transport protocols, all serving the same A2A protocol specification:

### 1. gRPC (Native)

**Status**: Core transport, auto-generated from `.proto` files

**Features**:
- ✅ Binary protocol (protobuf)
- ✅ Built-in streaming (bidirectional)
- ✅ HTTP/2 multiplexing
- ✅ Efficient for high-throughput
- ✅ Strong typing via protobuf
- ✅ Cross-language support

**Endpoints**:
```protobuf
service A2AService {
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  rpc SendStreamingMessage(SendMessageRequest) returns (stream StreamResponse);
  rpc GetTask(GetTaskRequest) returns (Task);
  rpc CancelTask(CancelTaskRequest) returns (Task);
  rpc TaskSubscription(TaskSubscriptionRequest) returns (stream StreamResponse);
  rpc GetAgentCard(GetAgentCardRequest) returns (AgentCard);
  // ... more methods
}
```

**Usage**:
```bash
# gRPC port (default: 50051)
grpcurl -plaintext \
  -d '{"request":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}' \
  localhost:50051 \
  a2a.v1.A2AService/SendMessage
```

### 2. REST (grpc-gateway)

**Status**: Auto-generated from protobuf annotations, zero custom code

**Features**:
- ✅ JSON over HTTP
- ✅ RESTful semantics
- ✅ Server-Sent Events (SSE) for streaming
- ✅ Browser-friendly
- ✅ 100% generated from proto definitions

**Key Endpoints**:
```
# Agent Discovery
GET    /.well-known/agent-card.json          # Service-level card
GET    /v1/agents                            # List all agents
GET    /v1/agents/{agent_id}/.well-known/agent-card.json  # Agent card

# Messaging
POST   /v1/agents/{agent_id}/message:send    # Non-streaming
POST   /v1/agents/{agent_id}/message:stream  # Streaming (SSE)

# Task Management
GET    /v1/tasks/{task_id}                   # Get task status
POST   /v1/tasks/{task_id}:cancel            # Cancel task
GET    /v1/tasks/{task_id}:subscribe         # Subscribe to updates (SSE)
```

**Example**:
```bash
# Send message
curl -X POST http://localhost:50052/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Hello"}]
    }
  }'

# Streaming
curl -N -X POST http://localhost:50052/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Tell me a story"}]
    }
  }'

# Output (SSE format):
# event: message
# data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":"Once"}]}}}
#
# event: message  
# data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":" upon"}]}}}
```

### 3. JSON-RPC

**Status**: Custom adapter over gRPC

**Features**:
- ✅ Simple RPC over HTTP POST
- ✅ JSON-RPC 2.0 compliant
- ✅ Single endpoint for all methods
- ✅ Easy integration for simple clients

**Endpoint**:
```
POST /rpc   # All methods
```

**Methods**:
- `SendMessage` - Send a message (non-streaming)
- `GetAgentCard` - Get agent metadata
- `GetTask` - Get task status
- `CancelTask` - Cancel a task

**Example**:
```bash
curl -X POST http://localhost:50053/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "SendMessage",
    "params": {
      "agentId": "assistant",
      "message": {
        "role": "ROLE_USER",
        "content": [{"text": "Hello"}]
      }
    },
    "id": 1
  }'

# Response:
{
  "jsonrpc": "2.0",
  "result": {
    "message": {
      "role": "ROLE_AGENT",
      "content": [{"text": "Hi there!"}]
    }
  },
  "id": 1
}
```

### Transport Comparison

| Feature | gRPC | REST | JSON-RPC |
|---------|------|------|----------|
| **Protocol** | HTTP/2 Binary | HTTP/1.1 JSON | HTTP/1.1 JSON |
| **Streaming** | Bidirectional | SSE (server→client) | Not supported |
| **Performance** | Highest | Medium | Medium |
| **Browser Support** | Via grpc-web | Native | Native |
| **Simplicity** | Medium | Medium | Highest |
| **Type Safety** | Strongest | Strong | Strong |
| **Code Generation** | Full | Full (grpc-gateway) | Partial |
| **Best For** | Services, high-throughput | Web apps, APIs | Simple integrations |

### Port Configuration

```yaml
# Default ports
global:
  a2a_server:
    grpc_port: 50051      # gRPC
    rest_port: 50052      # REST/HTTP
    jsonrpc_port: 50053   # JSON-RPC
```

---

## Client Architecture

The client layer provides a unified interface for interacting with A2A agents, abstracting away transport details.

### A2AClient Interface

```go
// pkg/a2a/client/client.go
type A2AClient interface {
    // Agent Discovery
    ListAgents(ctx context.Context) ([]AgentInfo, error)
    GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error)
    
    // Messaging
    SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error)
    StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error)
    
    // Lifecycle
    Close() error
}
```

### Client Implementations

#### 1. HTTPClient (Remote Agents)

**Purpose**: Connect to remote A2A servers over HTTP/REST

**Features**:
- ✅ Uses `protojson` for proper protobuf↔JSON serialization
- ✅ Supports streaming via SSE
- ✅ Authentication via bearer tokens
- ✅ Configurable timeouts

**Implementation**:
```go
// pkg/a2a/client/http.go
type HTTPClient struct {
    baseURL string
    token   string
    client  *http.Client
}

func (c *HTTPClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
    // Build request using protojson (NOT json.Marshal!)
    reqProto := &pb.SendMessageRequest{Request: message}
    jsonData, _ := protojson.Marshal(reqProto)  // ✅ Correct protobuf serialization
    
    // Make HTTP request
    resp, _ := c.client.Post(url, "application/json", bytes.NewReader(jsonData))
    
    // Parse streaming responses with protojson
    for scanner.Scan() {
        var streamResp pb.StreamResponse
        protojson.Unmarshal(line, &streamResp)  // ✅ Correct deserialization
        streamChan <- &streamResp
    }
}
```

**Usage**:
```go
// Connect to remote server
client := client.NewHTTPClient("http://localhost:50052", "token")

// Send message
response, err := client.SendMessage(ctx, "assistant", message)

// Stream responses
stream, err := client.StreamMessage(ctx, "assistant", message)
for chunk := range stream {
    if msg := chunk.GetMsg(); msg != nil {
        fmt.Print(msg.Content[0].GetText())
    }
}
```

#### 2. DirectClient (In-Process Agents)

**Purpose**: Execute agents in the same process (zero network overhead)

**Features**:
- ✅ No serialization overhead
- ✅ Direct protobuf type usage
- ✅ Useful for embedded scenarios
- ✅ Development and testing

**Implementation**:
```go
// pkg/a2a/client/direct.go
type DirectClient struct {
    config     *config.Config
    components *component.ComponentManager
    registry   *agent.AgentRegistry
}

func (c *DirectClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
    // Get agent from local registry
    agentEntry, _ := c.registry.Get(agentID)
    
    // Call directly (no network!)
    return agentEntry.Agent.SendMessage(ctx, &pb.SendMessageRequest{
        Request: message,
    })
}
```

**Usage**:
```go
// Create direct client
client, err := client.NewDirectClient(config)

// Same interface as HTTP client!
response, err := client.SendMessage(ctx, "assistant", message)
```

### Client Selection Logic

The CLI automatically chooses the right client:

```go
// pkg/cli/commands.go
func createClient(args Args) (client.A2AClient, error) {
    if args.ServerURL != "" {
        // Server mode: use HTTP client
        return runtime.NewHTTPClient(args.ServerURL, args.Token), nil
    }
    
    // Direct mode: use in-process client
    rt, err := runtime.New(runtime.Options{
        ConfigFile: args.ConfigFile,
        Provider:   args.Provider,
        APIKey:     args.APIKey,
        // ... more options
    })
    return rt.Client(), nil
}
```

---

## Server Architecture

The server layer hosts multiple agents and routes requests using A2A protocol.

### RegistryService (Multi-Agent Hub)

**Purpose**: Central registry and router for all agents

```go
// pkg/transport/registry_server.go
type RegistryService struct {
    pb.UnimplementedA2AServiceServer
    registry *agent.AgentRegistry
}
```

**Responsibilities**:
1. **Agent Registration**: Register native and remote agents
2. **Request Routing**: Route requests to correct agent based on `agent-name` header
3. **Discovery**: Provide agent listing and metadata
4. **Authentication**: Apply security policies
5. **Streaming**: Handle streaming responses

**Key Methods**:
```go
// Register an agent
func (s *RegistryService) RegisterAgent(agentID string, agent pb.A2AServiceServer, config *config.AgentConfig) error

// List all agents
func (s *RegistryService) ListAgents() []string

// Get agent metadata
func (s *RegistryService) GetAgentMetadata(agentID string) (*AgentMetadata, error)

// Route message to agent
func (s *RegistryService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
    // Extract agent ID from metadata
    md, _ := metadata.FromIncomingContext(ctx)
    agentID := md.Get("agent-name")[0]
    
    // Get agent
    agentEntry, _ := s.registry.Get(agentID)
    
    // Forward request
    return agentEntry.Agent.SendMessage(ctx, req)
}
```

### Agent Discovery Endpoints

Hector implements [RFC 8615](https://tools.ietf.org/html/rfc8615) well-known URIs for agent discovery:

```
# Service-level discovery
GET /.well-known/agent-card.json
Response: {
  "name": "Hector A2A Server",
  "description": "Multi-agent AI platform",
  "version": "1.0.0",
  "agents": [
    {"id": "assistant", "name": "Assistant", "agent_card_url": "/v1/agents/assistant/.well-known/agent-card.json"},
    {"id": "researcher", "name": "Researcher", "agent_card_url": "/v1/agents/researcher/.well-known/agent-card.json"}
  ]
}

# Agent-specific discovery
GET /v1/agents/{agent_id}/.well-known/agent-card.json
Response: {
  "name": "Assistant",
  "description": "Helpful AI assistant",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "task_tracking": true,
    "session_support": true
  },
  "security_schemes": {
    "bearer": {
      "type": "http",
      "scheme": "bearer"
    }
  }
}

# List all agents
GET /v1/agents
Response: {
  "agents": [
    {"id": "assistant", "name": "Assistant", "description": "...", "agent_card_url": "..."},
    {"id": "researcher", "name": "Researcher", "description": "...", "agent_card_url": "..."}
  ]
}
```

### Server Lifecycle

```go
// pkg/a2a/server/server.go
type HectorServer struct {
    grpc    *transport.Server           // gRPC server
    rest    *transport.RESTGateway      // REST gateway
    jsonrpc *transport.JSONRPCHandler   // JSON-RPC handler
}

// Start all transports
func (s *HectorServer) Start(ctx context.Context) error {
    // Start gRPC (port 50051)
    go s.grpc.Start()
    
    // Start REST gateway (port 50052)  
    go s.rest.Start(ctx)
    
    // Start JSON-RPC (port 50053)
    go s.jsonrpc.Start()
    
    // Wait for shutdown signal
    <-ctx.Done()
    return s.Stop(ctx)
}

// Graceful shutdown
func (s *HectorServer) Stop(ctx context.Context) error {
    s.grpc.Stop(ctx)
    s.rest.Stop(ctx)
    s.jsonrpc.Stop(ctx)
    return nil
}
```

**Bootstrap Process**:
```go
// pkg/a2a/server/bootstrap.go
func Bootstrap(opts BootstrapOptions) (*HectorServer, error) {
    // 1. Create component manager
    componentManager, _ := component.NewComponentManager(config)
    
    // 2. Create agent registry
    agentRegistry := agent.NewAgentRegistry()
    registryService := transport.NewRegistryService(agentRegistry)
    
    // 3. Register all agents
    for agentID, agentCfg := range config.Agents {
        if agentCfg.Type == "a2a" {
            // External A2A agent
            agent, _ := agent.NewExternalA2AAgent(&agentCfg)
        } else {
            // Native agent
            agent, _ := agent.NewAgent(&agentCfg, componentManager, agentRegistry)
        }
        registryService.RegisterAgent(agentID, agent, &agentCfg)
    }
    
    // 4. Create transports
    grpcServer := transport.NewServer(registryService, grpcConfig)
    restGateway := transport.NewRESTGateway(restConfig)
    jsonrpcHandler := transport.NewJSONRPCHandler(jsonrpcConfig, registryService)
    
    // 5. Apply authentication
    if config.Global.Auth.Enabled {
        jwtValidator, _ := auth.NewJWTValidator(...)
        grpcServer.SetInterceptors(jwtValidator)
        restGateway.SetAuth(&transport.AuthConfig{Validator: jwtValidator})
        jsonrpcHandler.SetAuth(&transport.AuthConfig{Validator: jwtValidator})
    }
    
    return NewHectorServer(grpcServer, restGateway, jsonrpcHandler), nil
}
```

---

## Runtime System

The runtime system manages configuration loading, initialization, and client creation.

### Runtime Package

```go
// pkg/runtime/runtime.go
type Runtime struct {
    config *config.Config
    client client.A2AClient
}

// Initialize runtime
func New(opts Options) (*Runtime, error) {
    // 1. Load or create config
    cfg, _ := loadOrCreateConfig(opts)
    
    // 2. Create appropriate client
    var a2aClient client.A2AClient
    if opts.ConfigFile != "" {
        // Use HTTPClient for server mode
        a2aClient = client.NewHTTPClient(serverURL, token)
    } else {
        // Use DirectClient for zero-config mode
        a2aClient, _ = client.NewDirectClient(cfg)
    }
    
    return &Runtime{
        config: cfg,
        client: a2aClient,
    }, nil
}
```

### Configuration Loading

```go
func loadOrCreateConfig(opts Options) (*config.Config, error) {
    // Try to load from file
    if fileExists(opts.ConfigFile) {
        cfg, _ := config.LoadConfig(opts.ConfigFile)
        cfg.SetDefaults()
        cfg.Validate()
        return cfg, nil
    }
    
    // Create zero-config
    return config.CreateZeroConfig(config.ZeroConfigOptions{
        Provider:   opts.Provider,   // openai, anthropic, gemini
        APIKey:     opts.APIKey,
        BaseURL:    opts.BaseURL,
        Model:      opts.Model,
        EnableTools: opts.Tools,
        MCPURL:     opts.MCPURL,
        DocsFolder: opts.DocsFolder,
    }), nil
}
```

### Zero-Config Mode

Hector supports zero-configuration mode for quick starts:

```bash
# Just provide API key, everything else is automatic
export OPENAI_API_KEY=sk-...
hector chat assistant --tools

# Or specify provider explicitly
hector chat assistant --provider anthropic --api-key sk-ant-... --tools
```

**Generated Configuration**:
```go
func CreateZeroConfig(opts ZeroConfigOptions) *Config {
    // Provider-specific defaults
    switch opts.Provider {
    case "openai":
        opts.BaseURL = "https://api.openai.com/v1"
        opts.Model = "gpt-4o-mini"
    case "anthropic":
        opts.BaseURL = "https://api.anthropic.com"
        opts.Model = "claude-3-5-sonnet-20241022"
    case "gemini":
        opts.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
        opts.Model = "gemini-2.0-flash-exp"
    }
    
    // Create config with single agent
    return &Config{
        Agents: map[string]AgentConfig{
            "assistant": {
                Name: "Assistant",
                LLM:  opts.Provider,
                // ... tool config if opts.EnableTools
            },
        },
        LLMs: map[string]LLMProviderConfig{
            opts.Provider: {
                Type:   opts.Provider,
                Model:  opts.Model,
                APIKey: opts.APIKey,
                Host:   opts.BaseURL,
            },
        },
    }
}
```

---

## Core Components

### 1. Agent

**File**: `pkg/agent/agent.go`

**Purpose**: Execute reasoning tasks and implement A2A protocol

**Interface Implementation**:
```go
type Agent struct {
    config   *config.AgentConfig
    llm      llms.LLMProvider
    tools    tools.ToolRegistry
    memory   memory.MemoryService
    services *AgentServices
}

// A2A Protocol Methods (pb.A2AServiceServer interface)
func (a *Agent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error)
func (a *Agent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error
func (a *Agent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error)
func (a *Agent) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error)
func (a *Agent) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error)
func (a *Agent) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error
```

### 2. LLM Providers

**File**: `pkg/llms/*.go`

**Providers**:
- `openai.go` - OpenAI (GPT-4o, GPT-4, GPT-3.5)
- `anthropic.go` - Anthropic (Claude 3.5 Sonnet, Opus, Haiku)
- `gemini.go` - Google Gemini (Gemini 2.0 Flash, 1.5 Pro)

**Interface**:
```go
type LLMProvider interface {
    Generate(messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error)
    GenerateStreaming(messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error)
    Close() error
}
```

### 3. Tool System

**File**: `pkg/tools/*.go`

**Tool Sources**:
- `local.go` - Built-in tools (command, file_writer, search_replace, todo)
- `mcp.go` - Model Context Protocol tools
- `registry.go` - Tool registration and discovery

**Built-in Tools**:
```go
// Command execution
type CommandTool struct {
    allowedCommands []string
    workingDir      string
}

// File operations
type FileWriterTool struct {
    allowedPaths []string
    maxFileSize  int64
}

// Find and replace
type SearchReplaceTool struct {
    workingDir string
}

// Task management
type TodoTool struct {
    todos []*TodoItem
}
```

### 4. Memory System

**File**: `pkg/memory/*.go`

**Strategies**:
- `buffer_window.go` - Fixed-size message window
- `summary_buffer.go` - Automatic summarization
- `session_service.go` - Session management

**Interface**:
```go
type MemoryStrategy interface {
    AddToHistory(ctx context.Context, message *pb.Message) error
    GetHistory(ctx context.Context) ([]*pb.Message, error)
    Clear(ctx context.Context) error
}
```

### 5. Task Service

**File**: `pkg/agent/task_service.go`

**Purpose**: Manage async task processing and tracking

**Implementations**:
- `memory_task_service.go` - In-memory task storage
- `sql_task_service.go` - SQL-based task storage (PostgreSQL, MySQL, SQLite)

**Interface**:
```go
type TaskService interface {
    CreateTask(ctx context.Context, agentID string) (*pb.Task, error)
    GetTask(ctx context.Context, taskID string) (*pb.Task, error)
    UpdateTask(ctx context.Context, task *pb.Task) error
    ListTasks(ctx context.Context, agentID string) ([]*pb.Task, error)
    DeleteTask(ctx context.Context, taskID string) error
}
```

---

## Multi-Agent Orchestration

Hector supports sophisticated multi-agent workflows via the A2A protocol.

### Orchestrator Pattern

```
┌────────────────────────────────────────────────────────────┐
│                       USER / CLIENT                        │
│                  (CLI, API, External System)               │
└─────────────────────────┬──────────────────────────────────┘
                          │ A2A Protocol
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  ORCHESTRATOR AGENT                         │
│  • Task decomposition                                       │
│  • Agent selection (LLM-driven)                             │
│  • Result synthesis                                         │
│  • Tool: agent_call(agent_id, task)                         │
└─────────────────┬───────────────────────────────────────────┘
                  │
         ┌────────┴────────┬────────────┐
         │                 │            │
         ▼                 ▼            ▼
┌─────────────────┐ ┌─────────────┐ ┌────────────────┐
│  Weather Agent  │ │Travel Agent │ │  Search Agent  │
│  (External A2A) │ │  (Native)   │ │    (Native)    │
└─────────────────┘ └─────────────┘ └────────────────┘
```

### Configuration Example

```yaml
# Orchestrator agent
agents:
  orchestrator:
    name: "Orchestrator"
    description: "Coordinates multiple specialist agents"
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"  # Supervisor reasoning for orchestration
      max_iterations: 20
    tools:
      - "agent_call"  # Built-in tool for calling other agents

  # Specialist agents
  weather:
    type: "a2a"  # External A2A agent
    url: "https://weather-agent.example.com"
    credentials:
      type: "bearer"
      token: "${WEATHER_API_TOKEN}"
  
  travel:
    name: "Travel Specialist"
    llm: "gpt-4o"
    document_stores:
      - "travel_docs"
  
  search:
    name: "Web Researcher"
    llm: "gpt-4o"
    tools:
      - "search"
```

### Agent Call Tool

**Built-in tool for agent-to-agent communication**:

```yaml
# Automatically available when multiple agents configured
tools:
  agent_call:
    type: "agent_call"
    enabled: true
```

**Usage by orchestrator**:
```
User: "Plan a trip to Paris and check the weather"

Orchestrator:
1. Breaking down task...
   - Subtask 1: Get weather for Paris
   - Subtask 2: Plan travel itinerary

2. Calling weather agent...
   Tool: agent_call(agent_id="weather", task="What's the weather in Paris next week?")
   Result: "Sunny, 20°C average"

3. Calling travel agent...
   Tool: agent_call(agent_id="travel", task="Plan 3-day Paris itinerary")
   Result: "Day 1: Eiffel Tower, Day 2: Louvre..."

4. Synthesizing results...
   "Here's your Paris trip plan with weather forecast..."
```

---

## Security & Authentication

Hector implements comprehensive security based on A2A protocol recommendations.

### JWT Authentication

**Configuration**:
```yaml
global:
  auth:
    jwks_url: "https://auth.example.com/.well-known/jwks.json"
    issuer: "https://auth.example.com"
    audience: "hector-api"
```

**Supported Schemes**:
- **Bearer Token** (JWT)
- **API Key** (via HTTP headers)
- **Basic Auth** (for development)

**Implementation**:
```go
// pkg/auth/middleware.go
type JWTValidator struct {
    jwks     *keyfunc.JWKS
    issuer   string
    audience string
}

// Validate JWT token
func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
    token, err := jwt.Parse(tokenString, v.jwks.Keyfunc)
    // Extract claims
    claims := &Claims{
        Subject:  token.Claims["sub"].(string),
        Issuer:   token.Claims["iss"].(string),
        Audience: token.Claims["aud"].(string),
        Roles:    token.Claims["roles"].([]string),
        TenantID: token.Claims["tenant_id"].(string),
    }
    return claims, nil
}

// HTTP Middleware
func (v *JWTValidator) HTTPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        auth := r.Header.Get("Authorization")
        token := strings.TrimPrefix(auth, "Bearer ")
        
        // Validate
        claims, err := v.ValidateToken(r.Context(), token)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add claims to context
        ctx := context.WithValue(r.Context(), "claims", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// gRPC Interceptors
func (v *JWTValidator) UnaryServerInterceptor() grpc.UnaryServerInterceptor
func (v *JWTValidator) StreamServerInterceptor() grpc.StreamServerInterceptor
```

### Agent-to-Agent Authentication

For external A2A agents:

```yaml
agents:
  external_agent:
    type: "a2a"
    url: "https://external-agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_AGENT_TOKEN}"
      # OR
      type: "api_key"
      key: "${EXTERNAL_API_KEY}"
      # OR
      type: "basic"
      username: "${USERNAME}"
      password: "${PASSWORD}"
```

**Client-side authentication**:
```go
// pkg/agent/a2a_client.go
func NewExternalA2AAgent(cfg *config.AgentConfig) (pb.A2AServiceServer, error) {
    // Create authenticated gRPC connection
    conn, err := auth.NewAuthenticatedClientConn(
        cfg.URL,
        auth.NewTokenProviderFromCredentials(cfg.Credentials),
    )
    
    // Create A2A client
    client := pb.NewA2AServiceClient(conn)
    
    return &ExternalA2AAgent{
        client: client,
        config: cfg,
    }, nil
}
```

---

## Extension Points

Hector is designed for extensibility via plugins and configuration.

### 1. Plugin System

**gRPC-based plugins** for custom providers:

```
┌────────────────────────────────────────┐
│         HECTOR CORE                    │
├────────────────────────────────────────┤
│  Plugin Registry                       │
│  • Discovery      • Lifecycle          │
│  • Hot-reload     • Sandboxing         │
└─────────┬──────────────────────────────┘
          │ gRPC
     ┌────┴────┬────────┬────────┐
     ▼         ▼        ▼        ▼
┌─────────┐ ┌────────┐ ┌──────┐ ┌──────┐
│LLM      │ │Database│ │Tool  │ │Other │
│Plugin   │ │Plugin  │ │Plugin│ │Plugin│
└─────────┘ └────────┘ └──────┘ └──────┘
```

**Configuration**:
```yaml
plugins:
  llm_providers:
    my_custom_llm:
      type: "grpc"
      path: "./plugins/my-llm"
      enabled: true
      config:
        custom_param: "value"
```

### 2. Tool Extensions

**MCP (Model Context Protocol)**:
```yaml
tools:
  mcp_tools:
    type: "mcp"
    url: "http://localhost:3000"
    enabled: true
```

**Custom Local Tools**:
```go
// Implement tools.Tool interface
type CustomTool struct{}

func (t *CustomTool) GetInfo() tools.ToolInfo {
    return tools.ToolInfo{
        Name: "custom_tool",
        Description: "Does something custom",
        InputSchema: {...},
    }
}

func (t *CustomTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Implementation
}

// Register
toolRegistry.RegisterTool(source, &CustomTool{})
```

### 3. Memory Strategies

**Custom Memory Strategy**:
```go
type CustomMemoryStrategy struct{}

func (m *CustomMemoryStrategy) AddToHistory(ctx context.Context, message *pb.Message) error
func (m *CustomMemoryStrategy) GetHistory(ctx context.Context) ([]*pb.Message, error)
func (m *CustomMemoryStrategy) Clear(ctx context.Context) error

// Register
memoryRegistry.Register("custom", &CustomMemoryStrategy{})
```

**Usage**:
```yaml
agents:
  my_agent:
    memory:
      strategy: "custom"
      config:
        custom_param: "value"
```

---

## Performance & Scalability

### Streaming Performance

- **gRPC**: Native bidirectional streaming with HTTP/2 multiplexing
- **REST/SSE**: Server-Sent Events for efficient server→client streaming
- **Chunked Transfer**: Minimal latency for first token

### Task Management

- **Async Processing**: Background workers for long-running tasks
- **Status Tracking**: Real-time task status updates
- **Push Notifications**: Optional webhooks for task completion

### Resource Management

```yaml
global:
  performance:
    max_concurrent_requests: 100
    request_timeout: "120s"
    task_workers: 10  # Async task processing workers
```

---

## Best Practices

### 1. Choose the Right Transport

- **gRPC**: Internal services, high-performance needs
- **REST**: Web applications, browser clients
- **JSON-RPC**: Simple integrations, minimal setup

### 2. Use Direct Client for Development

```bash
# Zero-config direct mode
hector chat assistant --tools --provider openai --api-key sk-...
```

### 3. Enable Authentication in Production

```yaml
global:
  auth:
    jwks_url: "https://your-auth-provider/.well-known/jwks.json"
    issuer: "https://your-auth-provider"
    audience: "your-api"
```

### 4. Configure Task Storage

```yaml
# In-memory (development)
agents:
  my_agent:
    task:
      backend: "memory"

# SQL (production)
agents:
  my_agent:
    task:
      backend: "sql"
      driver: "postgres"
      dsn: "${DATABASE_URL}"
```

### 5. Monitor and Log

```yaml
global:
  logging:
    level: "info"
    format: "json"
    output: "stdout"
```

---

## Summary

Hector provides a **complete, production-ready A2A protocol implementation** with:

✅ **100% Protobuf Native** - No abstraction layers  
✅ **Multi-Transport** - gRPC, REST, JSON-RPC  
✅ **Spec-Compliant** - Full A2A protocol support  
✅ **Extensible** - Plugins, tools, memory strategies  
✅ **Secure** - JWT authentication, role-based access  
✅ **Scalable** - Async tasks, streaming, clustering  
✅ **Developer-Friendly** - Zero-config mode, clear APIs  

**Start building A2A-native agents today!** 🚀
