---
title: AG-UI Protocol
description: AG-UI streaming event format for standardized agent UIs
---

# AG-UI Protocol Support

> **💡 See Also:** [AG-UI Metadata Schema](agui-metadata-schema.md) for detailed information about how Hector enriches A2A parts with AG-UI metadata.

Hector implements **AG-UI streaming events** as an optional output format alongside the native A2A protocol. This enables compatibility with modern agentic UIs that have adopted this standardized streaming format.

!!! info "Implementation Scope"
    Hector implements **AG-UI Streaming Events Layer** - the 16 standardized event types for real-time UI updates. The full AG-UI specification includes additional features (bidirectional interaction, state management, human-in-the-loop) which are future considerations.

## Overview

### What is AG-UI?

**AG-UI (Agent User Interaction)** is a standardized protocol for agent-to-UI communication. The core component we implement is the **streaming events layer** - 16 event types that provide fine-grained updates for UI rendering.

### Implementation Level

```
┌─────────────────────────────────────────────┐
│  AG-UI Streaming Events (✅ Implemented)    │
├─────────────────────────────────────────────┤
│  • 16 event types                           │
│  • Lifecycle management (start/delta/stop)  │
│  • SSE/WebSocket transport                  │
│  • A2A → AG-UI conversion                   │
└─────────────────────────────────────────────┘
         ↓ Future considerations
┌─────────────────────────────────────────────┐
│  AG-UI Full Spec (Not Implemented)          │
├─────────────────────────────────────────────┤
│  • Bidirectional interaction                │
│  • State management                         │
│  • Human-in-the-loop workflows              │
│  • Multi-agent handoff                      │
└─────────────────────────────────────────────┘
```

### Key Principles

1. **A2A First**: Hector is 100% A2A compatible by default. A2A is the native protocol.
2. **AG-UI as Output Layer**: AG-UI is an optional streaming format for UI compatibility
3. **Opt-in at Runtime**: Clients choose format via Accept header
4. **No Configuration**: Works out-of-the-box with any agent

---

## Event Types (16 Total)

### Message Events (3)
- `message_start`: Message begins
- `message_delta`: Incremental message text
- `message_stop`: Message ends

### Content Block Events (3)
- `content_block_start`: Content section begins (with blockId, blockType)
- `content_block_delta`: Incremental content text
- `content_block_stop`: Content section ends

### Tool Call Events (3)
- `tool_call_start`: Tool invocation begins (with toolName, input)
- `tool_call_delta`: Streaming tool parameters (optional)
- `tool_call_stop`: Tool execution completes (with result, error)

### Thinking Events (3)
- `thinking_start`: Agent reasoning begins
- `thinking_delta`: Incremental reasoning text
- `thinking_stop`: Reasoning complete

### Task Events (4)
- `task_start`: Task created
- `task_update`: Task status change
- `task_complete`: Task finished successfully
- `task_error`: Task failed

---

## Enabling AG-UI Format

AG-UI is **enabled by default** - no configuration required!

### Method 1: Accept Header (Recommended)

```bash
curl -N http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: application/x-agui-events" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'
```

Supported Accept values:
- `application/x-agui-events` ⭐ (recommended)
- `application/agui+json`

### Method 2: Query Parameter

```bash
curl -N "http://localhost:8080/v1/agents/assistant/message:stream?format=agui" \
  -H "Content-Type: application/json" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'
```

### Supported Endpoints

✅ **REST SSE**: `/v1/agents/{agent}/message:stream`  
✅ **JSON-RPC Streaming**: `/v1/agents/{agent}/stream` (POST with `message/stream` method)

---

## Protocol Comparison

### A2A Native Format (Default)

```bash
curl -N http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'
```

**Response:**
```
event: message
data: {"message":{"messageId":"msg-123","parts":[{"text":"Hello"}]}}

event: message
data: {"message":{"messageId":"msg-123","parts":[{"text":" there!"}]}}

event: done
data: {}
```

### AG-UI Format (Opt-in)

```bash
curl -N http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Accept: application/x-agui-events" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'
```

**Response:**
```
event: task_start
data: {"eventId":"evt-1","type":"AGUI_EVENT_TYPE_TASK_START","timestamp":"2025-11-04T10:52:22Z","taskStart":{"taskId":"task-123"}}

event: message_start
data: {"eventId":"evt-2","type":"AGUI_EVENT_TYPE_MESSAGE_START","timestamp":"2025-11-04T10:52:22Z","messageStart":{"messageId":"msg-123","role":"agent"}}

event: content_block_start
data: {"eventId":"evt-3","type":"AGUI_EVENT_TYPE_CONTENT_BLOCK_START","timestamp":"2025-11-04T10:52:22Z","contentBlockStart":{"blockId":"blk-1","blockType":"text"}}

event: content_block_delta
data: {"eventId":"evt-4","type":"AGUI_EVENT_TYPE_CONTENT_BLOCK_DELTA","timestamp":"2025-11-04T10:52:22Z","contentBlockDelta":{"blockId":"blk-1","delta":"Hello"}}

event: content_block_delta
data: {"eventId":"evt-5","type":"AGUI_EVENT_TYPE_CONTENT_BLOCK_DELTA","timestamp":"2025-11-04T10:52:22Z","contentBlockDelta":{"blockId":"blk-1","delta":" there!"}}

event: content_block_stop
data: {"eventId":"evt-6","type":"AGUI_EVENT_TYPE_CONTENT_BLOCK_STOP","timestamp":"2025-11-04T10:52:22Z","contentBlockStop":{"blockId":"blk-1"}}

event: message_stop
data: {"eventId":"evt-7","type":"AGUI_EVENT_TYPE_MESSAGE_STOP","timestamp":"2025-11-04T10:52:22Z","messageStop":{"messageId":"msg-123"}}

event: task_complete
data: {"eventId":"evt-8","type":"AGUI_EVENT_TYPE_TASK_COMPLETE","timestamp":"2025-11-04T10:52:22Z","taskComplete":{"taskId":"task-123"}}
```

---

## Event Mapping

### A2A → AG-UI Conversion

Hector automatically converts A2A events to AG-UI events at the transport layer:

| A2A Event | AG-UI Events | Details |
|-----------|--------------|---------|
| Task (SUBMITTED) | `task_start` → `message_start` | Task creation triggers message lifecycle |
| Message (text part) | `content_block_start` → `content_block_delta` → `content_block_stop` | Text wrapped in content blocks |
| Message (tool_call part) | `tool_call_start` → `tool_call_stop` | Tool invocation lifecycle |
| Message (thinking part) | `thinking_start` → `thinking_delta` → `thinking_stop` | Reasoning process exposed |
| Task (WORKING) | `task_update` | Status change notification |
| Task (COMPLETED) | `message_stop` → `task_complete` | Message and task completion |
| StatusUpdate | `task_update` | Incremental status changes |
| ArtifactUpdate | Content blocks | Artifacts as content |

### Example: Tool Call

**A2A Format:**
```json
{
  "message": {
    "parts": [{
      "data": {
        "data": {"name": "web_search", "input": "{\"query\":\"weather\"}"}
      },
      "metadata": {"part_type": "tool_call", "tool_call_id": "tc-123"}
    }]
  }
}
```

**AG-UI Format:**
```json
{
  "eventId": "evt-1",
  "type": "AGUI_EVENT_TYPE_TOOL_CALL_START",
  "timestamp": "2025-11-04T10:52:22Z",
  "toolCallStart": {
    "toolCallId": "tc-123",
    "toolName": "web_search",
    "input": {"query": "weather"}
  }
}
```

### Example: Thinking

When `show_thinking: true` in reasoning config:

**A2A Format:**
```json
{
  "message": {
    "parts": [{
      "text": "Let me analyze this step by step...",
      "metadata": {"part_type": "thinking", "title": "Planning"}
    }]
  }
}
```

**AG-UI Format:**
```json
[
  {
    "type": "AGUI_EVENT_TYPE_THINKING_START",
    "thinkingStart": {"thinkingId": "think-123", "title": "Planning"}
  },
  {
    "type": "AGUI_EVENT_TYPE_THINKING_DELTA",
    "thinkingDelta": {"thinkingId": "think-123", "delta": "Let me analyze this step by step..."}
  },
  {
    "type": "AGUI_EVENT_TYPE_THINKING_STOP",
    "thinkingStop": {"thinkingId": "think-123"}
  }
]
```

---

## Agent Card Declaration

AG-UI is advertised in the agent card as an optional extension:

```bash
curl http://localhost:8080/v1/agents/assistant/ | jq
```

```json
{
  "name": "Assistant",
  "capabilities": {
    "streaming": true,
    "extensions": [
      {
        "uri": "https://ag-ui.org/protocol/v1",
        "description": "AG-UI Protocol - Standardized streaming event format for agent UIs. Clients can opt-in via Accept header 'application/x-agui-events' or query parameter 'format=agui'.",
        "required": false
      }
    ]
  },
  "defaultOutputModes": [
    "text/plain",
    "application/json",
    "application/x-agui-events"
  ]
}
```

The `required: false` indicates that AG-UI is optional - clients can use A2A or AG-UI.

---

## Configuration

**No configuration required!** AG-UI is enabled by default as an optional output format.

Every agent automatically supports both A2A and AG-UI with these default output modes:
- `text/plain`
- `application/json`
- `application/x-agui-events` ✨

Simply use any existing agent configuration:

```yaml
agents:
  assistant:
    name: "Assistant"
    llm: default-llm
    
    reasoning:
      engine: "react"
      enable_streaming: true
      show_thinking: true  # Thinking will be converted to AG-UI thinking events
    
    tools:
      - web_search
```

Clients choose their preferred format at runtime via Accept header or query parameter.

---

## Client Implementation

### JavaScript/TypeScript Example

```typescript
const eventSource = new EventSource(
  'http://localhost:8080/v1/agents/assistant/message:stream',
  {
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/x-agui-events'  // Opt-in to AG-UI
    }
  }
);

// Handle different event types
eventSource.addEventListener('message_start', (e) => {
  const event = JSON.parse(e.data);
  console.log('Message started:', event.messageStart.messageId);
});

eventSource.addEventListener('content_block_delta', (e) => {
  const event = JSON.parse(e.data);
  const delta = event.contentBlockDelta.delta;
  // Render delta to UI
  appendToMessage(delta);
});

eventSource.addEventListener('tool_call_start', (e) => {
  const event = JSON.parse(e.data);
  const toolName = event.toolCallStart.toolName;
  // Show tool indicator
  showToolIndicator(toolName);
});

eventSource.addEventListener('thinking_delta', (e) => {
  const event = JSON.parse(e.data);
  const thinkingText = event.thinkingDelta.delta;
  // Show thinking process
  renderThinking(thinkingText);
});

eventSource.addEventListener('task_complete', (e) => {
  const event = JSON.parse(e.data);
  console.log('Task completed:', event.taskComplete.taskId);
  eventSource.close();
});
```

### Python Example

```python
import requests
import json

url = "http://localhost:8080/v1/agents/assistant/message:stream"
headers = {
    "Content-Type": "application/json",
    "Accept": "application/x-agui-events"  # Opt-in to AG-UI
}
data = {"message": {"parts": [{"text": "Hello"}]}}

response = requests.post(url, headers=headers, json=data, stream=True)

for line in response.iter_lines():
    if line:
        line_str = line.decode('utf-8')
        if line_str.startswith('event:'):
            event_type = line_str.split(':', 1)[1].strip()
        elif line_str.startswith('data:'):
            event_data = json.loads(line_str.split(':', 1)[1].strip())
            
            if event_type == 'content_block_delta':
                delta = event_data['contentBlockDelta']['delta']
                print(delta, end='', flush=True)
            elif event_type == 'tool_call_start':
                tool_name = event_data['toolCallStart']['toolName']
                print(f"\n🔧 {tool_name}", flush=True)
```

---

## Testing

**✅ Implementation tested and verified!** See `TEST_RESULTS_AGUI.md` for detailed test results.

Start any Hector agent (AG-UI is always available):

```bash
# Zero-config mode (easiest)
./hector serve --tools

# Or use any existing config
./hector serve --config configs/coding.yaml
```

Test with curl:

```bash
# A2A format (default) - ✅ Tested
curl -N http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'

# AG-UI format - ✅ Tested
curl -N http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: application/x-agui-events" \
  -d '{"message": {"parts": [{"text": "Hello"}]}}'

# Check agent card - ✅ Tested
curl http://localhost:8080/v1/agents/assistant/ | jq '.capabilities.extensions'
```

**Expected Results:**
- A2A format returns `event: message` with A2A Message structure
- AG-UI format returns `event: message_start`, `event: content_block_delta`, etc.
- Agent card shows AG-UI extension at `https://ag-ui.org/protocol/v1`

---

## Benefits

### For UI Developers

1. **Standardized Events**: Use the same event handlers across different agent platforms
2. **Fine-Grained Control**: Separate events for different UI elements (thinking, tools, content)
3. **Progressive Rendering**: Stream content as it's generated with delta events
4. **Better UX**: Show tool calls, thinking process, and task status separately

### For Hector Users

1. **Backward Compatibility**: Existing A2A clients continue to work
2. **Future-Proof**: Compatible with emerging agentic UI frameworks
3. **No Configuration**: Works out of the box with opt-in mechanism
4. **Dual Protocol**: Support both A2A agents and AG-UI UIs simultaneously

---

## Architecture

### Conversion Layer

The conversion from A2A to AG-UI happens at the transport layer (REST gateway):

```
┌─────────────────────────────────────────────────────────┐
│                    Client Request                        │
│  (Accept: application/x-agui-events or ?format=agui)    │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│                    REST Gateway                          │
│  • Detects AG-UI format preference                      │
│  • Creates restStreamWrapper with useAGUI=true          │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│                    Agent (A2A Native)                    │
│  • Generates A2A StreamResponse events                  │
│  • Task, Message, StatusUpdate, ArtifactUpdate          │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│          restStreamWrapper.sendAsAGUI()                 │
│  • Converts A2A events to AG-UI events                  │
│  • Maintains state (message, blocks, tools, thinking)   │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│                   SSE Response                           │
│  event: message_start                                    │
│  data: {...}                                             │
│                                                          │
│  event: content_block_delta                              │
│  data: {...}                                             │
└──────────────────────────────────────────────────────────┘
```

### Key Components

1. **`pkg/agui/proto/agui.proto`**: Protocol buffer definition for AG-UI events
2. **`pkg/agui/converter.go`**: A2A Part → AG-UI events converter
3. **`pkg/agui/events.go`**: AG-UI event builder functions
4. **`pkg/transport/rest_gateway.go`**: HTTP/SSE handler with AG-UI support
5. **`pkg/cli/agui_handler.go`**: Reference implementation for displaying AG-UI events

---

## Limitations & Future Work

### What's Implemented ✅

- ✅ All 16 AG-UI streaming event types
- ✅ SSE transport with proper event names
- ✅ A2A → AG-UI conversion at transport layer
- ✅ Opt-in mechanism (Accept header / query param)
- ✅ Agent card declaration
- ✅ Zero configuration

### What's Not Implemented (AG-UI Full Spec) ⚠️

The full AG-UI specification includes features beyond streaming events:

- ❌ **Bidirectional Communication**: Client → Agent tool invocations, interrupts
- ❌ **State Management**: Structured state, incremental updates, snapshots
- ❌ **Human-in-the-Loop**: Pause/modify/approve workflows
- ❌ **Multi-Agent Handoff**: Agent-to-agent collaboration protocol
- ❌ **Client-Defined Tools**: Front-end defines tools for agent to call

These features are future considerations and would require:
1. Extended A2A protocol support
2. Bidirectional WebSocket transport
3. State synchronization mechanisms
4. Workflow management APIs

### Current Implementation Scope

We implement **AG-UI Streaming Events** - the event format layer that enables compatibility with AG-UI-aware display UIs. This is valuable for:

1. **UI Framework Compatibility**: Works with streaming chat UIs (Vercel AI SDK, etc.)
2. **Progressive Rendering**: Fine-grained events for smooth UX
3. **Tool/Thinking Visibility**: Separate events for different interaction types
4. **Standardization**: Common format across agent platforms

For full AG-UI spec features, consider:
- Using A2A protocol directly (bidirectional, state management)
- MCP for client-defined tools
- A2A multi-agent collaboration features

---

## Summary

✅ **A2A Native**: Default protocol, always works  
✅ **AG-UI Optional**: Opt-in streaming event format  
✅ **Zero Configuration**: Works out-of-the-box  
✅ **Full Compatibility**: Both protocols coexist seamlessly  
✅ **Tested & Verified**: All features working correctly  

The implementation successfully provides AG-UI streaming event compatibility while maintaining 100% A2A protocol fidelity.

!!! tip "When to Use Each Protocol"
    - **Use A2A** for: Full-featured agent integration, bidirectional communication, state management
    - **Use AG-UI** for: Streaming UI compatibility, display-only clients, framework integration
    - **Use Both** for: Maximum compatibility with diverse clients
