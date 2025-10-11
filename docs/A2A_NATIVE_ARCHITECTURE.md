---
layout: default
title: A2A Native Architecture
nav_order: 1
parent: Advanced
description: "Deep A2A protocol integration - Native message types, rich content, and zero abstraction layers"
---

# A2A Native Architecture

## 🏆 **Deep A2A Protocol Integration**

Hector is built with **genuine A2A-native architecture** - every component uses A2A protocol types directly, with zero abstraction layers or compatibility wrappers.

---

## **🎯 What Makes Hector Truly A2A-Native**

### **✅ Native Message Types Throughout**
```go
// Every component uses a2a.Message directly
type LLMService interface {
    Generate(messages []a2a.Message, tools []ToolDefinition) (string, []a2a.ToolCall, int, error)
}

type MemoryService interface {
    GetRecentHistory(sessionID string) ([]a2a.Message, error)
    AddToHistory(sessionID string, msg a2a.Message) error
}

type Agent interface {
    ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error)
}
```

### **✅ Rich Content Support**
```go
// A2A Message with rich content parts
message := a2a.Message{
    Role: a2a.MessageRoleUser,
    Parts: []a2a.Part{
        {Type: a2a.PartTypeText, Text: "Hello"},
        {Type: a2a.PartTypeFile, File: "document.pdf"},
        {Type: a2a.PartTypeData, Data: jsonData},
    },
    ToolCalls: []a2a.ToolCall{...},
}
```

### **✅ Native Tool Call Integration**
```go
// Tool calls are native A2A types
toolCall := a2a.ToolCall{
    ID:        "call_123",
    Name:      "search_web",
    Arguments: map[string]interface{}{"query": "AI agents"},
    RawArgs:   `{"query": "AI agents"}`,
}
```

---

## **🔄 Complete Message Flow**

### **Native A2A Flow:**
```
HTTP Request → A2A Server → Agent → Reasoning → Memory → LLM Providers
     ↓              ↓         ↓         ↓         ↓           ↓
a2a.Message → a2a.Task → a2a.Message → a2a.Message → a2a.Message → API Format
```

**Every step uses native A2A types - NO conversions, NO aliases, NO abstractions!**

---

## **🏗️ Architecture Components**

### **1. A2A Server (HTTP+JSON Transport)**
```go
// Native A2A protocol implementation
type Server struct {
    agents map[string]Agent // Pure a2a.Agent interface
    tasks  map[string]*a2a.Task // Native A2A tasks
}

// Handle A2A message/send endpoint
func (s *Server) handleMessageSend(w http.ResponseWriter, r *http.Request, agentID string) {
    var params a2a.MessageSendParams
    json.NewDecoder(r.Body).Decode(&params)
    
    task := &a2a.Task{
        Messages: []a2a.Message{params.Message}, // Native A2A Message
    }
    
    agent.ExecuteTask(ctx, task) // Native A2A execution
}
```

### **2. Agent Implementation**
```go
// Native A2A Agent interface implementation
func (a *Agent) ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
    // Extract user message using native A2A helpers
    userText := a2a.ExtractTextFromTask(task)
    
    // Execute reasoning with native A2A Message types
    streamCh, err := a.execute(ctx, userText, strategy)
    
    // Add response using native A2A Message creation
    task.Messages = append(task.Messages, a2a.CreateAssistantMessage(response))
    
    return task, nil
}
```

### **3. Memory Services**
```go
// Native A2A Message storage and retrieval
func (s *MemoryService) AddToHistory(sessionID string, msg a2a.Message) error {
    // Store native A2A Message directly
    return s.workingMemory.AddMessage(session, msg)
}

func (s *MemoryService) GetRecentHistory(sessionID string) ([]a2a.Message, error) {
    // Return native A2A Messages directly
    return s.workingMemory.GetMessages(session)
}
```

### **4. LLM Providers**
```go
// Native A2A Message processing
func (p *OpenAIProvider) Generate(messages []a2a.Message, tools []ToolDefinition) (string, []a2a.ToolCall, int, error) {
    // Convert A2A Messages to OpenAI format
    for _, msg := range messages {
        content := a2a.ExtractTextFromMessage(msg) // Native A2A helper
        toolCalls := a2a.ExtractToolCallsFromMessage(msg) // Native A2A helper
        
        openaiMsg := OpenAIMessage{
            Role:    string(msg.Role), // Native A2A role
            Content: &content,
        }
    }
}
```

---

## **🎉 Benefits of Native A2A Architecture**

### **1. True Protocol Compliance**
- **No custom message types** - Uses A2A protocol types exclusively
- **No abstraction layers** - Direct A2A type usage throughout
- **No compatibility wrappers** - Pure A2A implementation

### **2. Rich Content Support**
- **Multi-part messages** - Text, files, data in single message
- **Tool call integration** - Native tool call support
- **Future-ready** - Ready for multi-modal agents

### **3. Performance & Maintainability**
- **No conversion overhead** - Direct A2A type usage
- **Type safety** - Impossible to mix up message types
- **Clean codebase** - Single source of truth for message types

### **4. Interoperability**
- **True A2A compliance** - Works with any A2A agent
- **Protocol evolution** - Easy to adopt new A2A features
- **Standards-based** - No proprietary deviations

---

## **🔍 Technical Deep Dive**

### **Message Type Hierarchy**
```
a2a.Message (Native A2A Protocol)
├── Role: MessageRole (user, assistant, system, tool)
├── Parts: []Part (text, files, data)
├── ToolCalls: []ToolCall (native tool calls)
├── ToolCallID: string (tool result reference)
└── Name: string (tool name)
```

### **Helper Functions**
```go
// Native A2A helper functions
a2a.CreateUserMessage("Hello")           // Creates user message
a2a.CreateAssistantMessage("Hi!")        // Creates assistant message
a2a.ExtractTextFromMessage(msg)          // Extracts text content
a2a.ExtractToolCallsFromMessage(msg)     // Extracts tool calls
a2a.ExtractTextFromTask(task)            // Extracts user input from task
```

### **Type Safety**
```go
// All interfaces use native A2A types
type LLMService interface {
    Generate(messages []a2a.Message, tools []ToolDefinition) (string, []a2a.ToolCall, int, error)
}

type MemoryService interface {
    GetRecentHistory(sessionID string) ([]a2a.Message, error)
    AddToHistory(sessionID string, msg a2a.Message) error
}

type Agent interface {
    ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error)
}
```

---

## **🚀 Future-Ready Architecture**

### **Multi-Modal Support**
```go
// Ready for rich content
message := a2a.Message{
    Parts: []a2a.Part{
        {Type: a2a.PartTypeText, Text: "Analyze this image"},
        {Type: a2a.PartTypeFile, File: "chart.png"},
        {Type: a2a.PartTypeData, Data: jsonData},
    },
}
```

### **Protocol Evolution**
- **Easy to extend** - Add new A2A features without breaking changes
- **Backward compatible** - A2A protocol ensures compatibility
- **Standards-based** - Follows A2A specification evolution

---

## **🎯 Conclusion**

Hector's **deep A2A native integration** ensures:

1. **✅ True protocol compliance** - No custom deviations
2. **✅ Future-ready architecture** - Ready for rich content and multi-modal agents
3. **✅ Performance optimized** - No conversion overhead
4. **✅ Maintainable codebase** - Single source of truth for message types
5. **✅ Interoperable** - Works with any A2A-compliant agent

**This is what makes Hector a genuine A2A-native platform!** 🚀
