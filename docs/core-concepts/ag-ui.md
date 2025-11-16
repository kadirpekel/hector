---
title: AG-UI Protocol
description: AG-UI streaming event format and metadata schema for standardized agent UIs
---

# AG-UI Protocol Support

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

## AG-UI Metadata Schema

Hector enriches all A2A protocol parts with **AG-UI metadata** to make them natively compatible with AG-UI-based user interfaces while remaining 100% A2A compliant. This is achieved by embedding AG-UI event type hints in the A2A `Part.metadata` field, which is explicitly designed for protocol extensions.

### Why AG-UI Metadata in A2A Parts?

The **A2A (Agent-to-Agent) Protocol** is Hector's native protocol and intentionally does not prescribe specific contextual types like "thinking", "tool_call", or "task" in its core specification. Instead, A2A provides an optional `metadata` field on `Part` messages for custom extensions.

**AG-UI (Agent User Interaction) Protocol** is a standardized streaming event format for agent UIs that defines specific event types for rich user experiences (thinking blocks, tool calls, tasks, etc.).

By embedding AG-UI metadata hints in A2A parts, Hector achieves:

1. **A2A Native**: All parts remain valid A2A messages
2. **AG-UI Ready**: AG-UI clients can read metadata hints for proper UI rendering
3. **Zero Configuration**: No special configuration needed - AG-UI is a default capability
4. **Backward Compatible**: A2A-only clients simply ignore unfamiliar metadata

### Core Metadata Fields

All AG-UI-enriched A2A parts include these optional metadata fields:

| Field | Type | Description |
|-------|------|-------------|
| `agui_event_type` | string | The AG-UI event type: `"content_block"`, `"thinking"`, `"tool_call"`, `"task"`, `"error"`, or `"message"` |
| `agui_block_type` | string | For content blocks: `"text"`, `"thinking"`, or `"code"` |
| `agui_block_id` | string | Unique identifier for this content block |
| `agui_block_index` | integer | Sequential index of this block within the message |

### Tool Call Metadata Fields

For tool call and tool result parts:

| Field | Type | Description |
|-------|------|-------------|
| `agui_tool_call_id` | string | Unique identifier for the tool call |
| `agui_tool_name` | string | Name of the tool being called |
| `agui_is_error` | boolean | Whether this tool result represents an error |

### Example A2A Parts with AG-UI Metadata

**Text Content Part:**
```json
{
  "text": "Hello! How can I help you today?",
  "metadata": {
    "agui_event_type": "content_block",
    "agui_block_type": "text",
    "agui_block_id": "block-1234",
    "agui_block_index": 0
  }
}
```

**Thinking Block Part:**
```json
{
  "text": "[Thinking: Analyzing the user's request...]\n",
  "metadata": {
    "agui_event_type": "thinking",
    "agui_block_type": "thinking",
    "agui_block_id": "think-5678",
    "agui_block_index": 1
  }
}
```

**Tool Call Part:**
```json
{
  "data": {
    "data": {
      "id": "call-9abc",
      "name": "search_code",
      "arguments": {
        "query": "authentication logic"
      }
    }
  },
  "metadata": {
    "agui_event_type": "tool_call",
    "agui_tool_call_id": "call-9abc",
    "agui_tool_name": "search_code"
  }
}
```

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
      engine: "chain-of-thought"
      enable_streaming: true
      show_thinking: true  # Thinking will be converted to AG-UI thinking events
    
    tools:
      - web_search
```

Clients choose their preferred format at runtime via Accept header or query parameter.

---

## UI Implementation Guide

### Building an AG-UI Compatible Client

This section provides guidance for implementing AG-UI clients that consume Hector's AG-UI streaming events.

#### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Client Application                     │
│  ┌──────────────┬──────────────┬──────────────┐          │
│  │   Event      │    State     │     UI       │          │
│  │  Handler     │  Management  │  Rendering   │          │
│  └──────┬───────┴──────┬───────┴──────┬───────┘          │
│         │              │              │                   │
│         └──────────────┴──────────────┘                   │
│                    Event Stream                            │
└───────────────────────────┬───────────────────────────────┘
                            │
┌───────────────────────────▼───────────────────────────────┐
│              Hector AG-UI Event Stream                      │
│  (SSE: application/x-agui-events)                          │
└─────────────────────────────────────────────────────────────┘
```

#### Step 1: Connect to Event Stream

**JavaScript/TypeScript:**
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
```

**Python:**
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
```

#### Step 2: Handle Event Types

**Message Lifecycle:**
```typescript
interface MessageState {
  messageId: string;
  role: 'user' | 'agent';
  blocks: Map<string, ContentBlock>;
  thinking: Map<string, ThinkingBlock>;
  toolCalls: Map<string, ToolCall>;
}

const messageState = new Map<string, MessageState>();

eventSource.addEventListener('message_start', (e) => {
  const event = JSON.parse(e.data);
  const msgId = event.messageStart.messageId;
  messageState.set(msgId, {
    messageId: msgId,
    role: event.messageStart.role,
    blocks: new Map(),
    thinking: new Map(),
    toolCalls: new Map()
  });
  // Render new message container in UI
  createMessageContainer(msgId);
});

eventSource.addEventListener('message_stop', (e) => {
  const event = JSON.parse(e.data);
  const msgId = event.messageStop.messageId;
  // Finalize message rendering
  finalizeMessage(msgId);
});
```

**Content Blocks:**
```typescript
eventSource.addEventListener('content_block_start', (e) => {
  const event = JSON.parse(e.data);
  const blockId = event.contentBlockStart.blockId;
  const blockType = event.contentBlockStart.blockType;
  
  // Create new content block in UI
  createContentBlock(blockId, blockType);
});

eventSource.addEventListener('content_block_delta', (e) => {
  const event = JSON.parse(e.data);
  const blockId = event.contentBlockDelta.blockId;
  const delta = event.contentBlockDelta.delta;
  
  // Append delta to content block (streaming text)
  appendToBlock(blockId, delta);
});

eventSource.addEventListener('content_block_stop', (e) => {
  const event = JSON.parse(e.data);
  const blockId = event.contentBlockStop.blockId;
  
  // Mark block as complete
  finalizeBlock(blockId);
});
```

**Tool Calls:**
```typescript
eventSource.addEventListener('tool_call_start', (e) => {
  const event = JSON.parse(e.data);
  const toolCallId = event.toolCallStart.toolCallId;
  const toolName = event.toolCallStart.toolName;
  const input = event.toolCallStart.input;
  
  // Show tool indicator in UI
  showToolCall(toolCallId, toolName, input);
});

eventSource.addEventListener('tool_call_stop', (e) => {
  const event = JSON.parse(e.data);
  const toolCallId = event.toolCallStop.toolCallId;
  const result = event.toolCallStop.result;
  const error = event.toolCallStop.error;
  
  // Update tool call with result
  updateToolCall(toolCallId, result, error);
});
```

**Thinking Blocks:**
```typescript
eventSource.addEventListener('thinking_start', (e) => {
  const event = JSON.parse(e.data);
  const thinkingId = event.thinkingStart.thinkingId;
  const title = event.thinkingStart.title;
  
  // Show thinking block (collapsible by default)
  showThinkingBlock(thinkingId, title);
});

eventSource.addEventListener('thinking_delta', (e) => {
  const event = JSON.parse(e.data);
  const thinkingId = event.thinkingDelta.thinkingId;
  const delta = event.thinkingDelta.delta;
  
  // Append thinking text
  appendThinking(thinkingId, delta);
});

eventSource.addEventListener('thinking_stop', (e) => {
  const event = JSON.parse(e.data);
  const thinkingId = event.thinkingStop.thinkingId;
  
  // Finalize thinking block
  finalizeThinking(thinkingId);
});
```

**Task Status:**
```typescript
eventSource.addEventListener('task_start', (e) => {
  const event = JSON.parse(e.data);
  const taskId = event.taskStart.taskId;
  
  // Show task indicator
  showTaskIndicator(taskId);
});

eventSource.addEventListener('task_update', (e) => {
  const event = JSON.parse(e.data);
  const taskId = event.taskUpdate.taskId;
  const status = event.taskUpdate.status;
  
  // Update task status
  updateTaskStatus(taskId, status);
});

eventSource.addEventListener('task_complete', (e) => {
  const event = JSON.parse(e.data);
  const taskId = event.taskComplete.taskId;
  
  // Mark task as complete
  completeTask(taskId);
});

eventSource.addEventListener('task_error', (e) => {
  const event = JSON.parse(e.data);
  const taskId = event.taskError.taskId;
  const error = event.taskError.error;
  
  // Show error
  showTaskError(taskId, error);
});
```

#### Step 3: State Management

Maintain state for each message to properly render streaming content:

```typescript
class AGUIStateManager {
  private messages = new Map<string, MessageState>();
  
  handleEvent(eventType: string, eventData: any) {
    switch (eventType) {
      case 'message_start':
        this.startMessage(eventData);
        break;
      case 'content_block_delta':
        this.appendContent(eventData);
        break;
      case 'tool_call_start':
        this.startToolCall(eventData);
        break;
      // ... handle all event types
    }
  }
  
  getMessageState(messageId: string): MessageState | undefined {
    return this.messages.get(messageId);
  }
}
```

#### Step 4: UI Rendering Best Practices

1. **Progressive Rendering**: Update UI incrementally as deltas arrive
2. **Visual Indicators**: Show loading states for tool calls and thinking
3. **Error Handling**: Display errors from `tool_call_stop` and `task_error` events
4. **Collapsible Thinking**: Make thinking blocks collapsible by default
5. **Tool Call Visualization**: Show tool name, input, and result clearly
6. **Task Status**: Display task progress and status updates

#### Step 5: Complete Example

**React Component Example:**
```typescript
import React, { useEffect, useState } from 'react';

interface AGUIEvent {
  eventId: string;
  type: string;
  timestamp: string;
  [key: string]: any;
}

export function AGUIChat() {
  const [messages, setMessages] = useState<Map<string, any>>(new Map());
  
  useEffect(() => {
    const eventSource = new EventSource(
      'http://localhost:8080/v1/agents/assistant/message:stream',
      {
        headers: {
          'Accept': 'application/x-agui-events'
        }
      }
    );
    
    eventSource.addEventListener('message_start', (e) => {
      const event: AGUIEvent = JSON.parse(e.data);
      setMessages(prev => {
        const next = new Map(prev);
        next.set(event.messageStart.messageId, {
          id: event.messageStart.messageId,
          role: event.messageStart.role,
          content: '',
          toolCalls: [],
          thinking: []
        });
        return next;
      });
    });
    
    eventSource.addEventListener('content_block_delta', (e) => {
      const event: AGUIEvent = JSON.parse(e.data);
      setMessages(prev => {
        const next = new Map(prev);
        const msg = next.get(event.contentBlockDelta.blockId);
        if (msg) {
          msg.content += event.contentBlockDelta.delta;
        }
        return next;
      });
    });
    
    // ... handle other events
    
    return () => eventSource.close();
  }, []);
  
  return (
    <div className="chat-container">
      {Array.from(messages.values()).map(msg => (
        <div key={msg.id} className={`message ${msg.role}`}>
          <div className="content">{msg.content}</div>
          {msg.toolCalls.map(tc => (
            <div key={tc.id} className="tool-call">
              🔧 {tc.name}: {tc.result}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
```

### Using A2A Format with AG-UI Metadata

If you prefer A2A format but want AG-UI metadata hints:

```javascript
// A2A parts include AG-UI metadata automatically
function extractAGUIMetadata(part) {
  if (!part.metadata) return null;
  
  return {
    eventType: part.metadata.agui_event_type,
    blockType: part.metadata.agui_block_type,
    blockId: part.metadata.agui_block_id,
    toolCallId: part.metadata.agui_tool_call_id,
    toolName: part.metadata.agui_tool_name
  };
}

// Use metadata to render parts appropriately
function renderPart(part) {
  const metadata = extractAGUIMetadata(part);
  
  if (metadata?.eventType === 'thinking') {
    return renderThinkingBlock(part.text);
  } else if (metadata?.eventType === 'tool_call') {
    return renderToolCall(part.data.data);
  } else {
    return renderTextContent(part.text);
  }
}
```

---

## Client Implementation Examples

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

**✅ Implementation tested and verified!**

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
- ✅ AG-UI metadata in A2A parts

### What's Not Implemented (AG-UI Full Spec) ⚠️

The full AG-UI specification includes features beyond streaming events:

- ❌ **Bidirectional Communication**: Client → Agent tool invocations, interrupts
- ❌ **State Management**: Structured state, incremental updates, snapshots
- ❌ **Human-in-the-Loop**: Pause/modify/approve workflows (Note: Hector implements HITL via A2A protocol)
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
✅ **UI Implementation Guide**: Complete guidance for building AG-UI clients

The implementation successfully provides AG-UI streaming event compatibility while maintaining 100% A2A protocol fidelity.

!!! tip "When to Use Each Protocol"
    - **Use A2A** for: Full-featured agent integration, bidirectional communication, state management
    - **Use AG-UI** for: Streaming UI compatibility, display-only clients, framework integration
    - **Use Both** for: Maximum compatibility with diverse clients

