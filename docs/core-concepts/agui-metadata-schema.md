# AG-UI Metadata Schema

## Overview

Hector enriches all A2A protocol parts with **AG-UI (Agent User Interaction) metadata** to make them natively compatible with AG-UI-based user interfaces while remaining 100% A2A compliant. This is achieved by embedding AG-UI event type hints in the A2A `Part.metadata` field, which is explicitly designed for protocol extensions.

## Why AG-UI Metadata in A2A Parts?

The **A2A (Agent-to-Agent) Protocol** is Hector's native protocol and intentionally does not prescribe specific contextual types like "thinking", "tool_call", or "task" in its core specification. Instead, A2A provides an optional `metadata` field on `Part` messages for custom extensions.

**AG-UI (Agent User Interaction) Protocol** is a standardized streaming event format for agent UIs that defines specific event types for rich user experiences (thinking blocks, tool calls, tasks, etc.).

By embedding AG-UI metadata hints in A2A parts, Hector achieves:

1. **A2A Native**: All parts remain valid A2A messages
2. **AG-UI Ready**: AG-UI clients can read metadata hints for proper UI rendering
3. **Zero Configuration**: No special configuration needed - AG-UI is a default capability
4. **Backward Compatible**: A2A-only clients simply ignore unfamiliar metadata

## AG-UI Metadata Fields

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

## Example A2A Parts with AG-UI Metadata

### Text Content Part

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

### Thinking Block Part

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

### Tool Call Part

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

### Tool Result Part

```json
{
  "data": {
    "data": {
      "tool_call_id": "call-9abc",
      "content": "Found 5 files matching 'authentication'...",
      "error": ""
    }
  },
  "metadata": {
    "agui_event_type": "tool_call",
    "agui_tool_call_id": "call-9abc",
    "agui_is_error": false
  }
}
```

## Implementation Details

### Creating AG-UI-Enriched Parts

Hector provides helper functions in `pkg/protocol/agui_metadata.go` for creating AG-UI-enriched parts:

```go
// Create a text content part with AG-UI metadata
part := protocol.CreateTextPartWithAGUI(
    text,      // "Hello world"
    blockID,   // "block-uuid" or "" for auto-generation
    blockIndex // 0, 1, 2, ...
)

// Create a thinking block part
thinkingPart := protocol.CreateThinkingPart(
    thinkingText, // "[Thinking: reasoning process...]"
    blockID,      // "" for auto-generation
    blockIndex    // index within message
)

// Create a tool call part (automatically includes AG-UI metadata)
toolCallPart := protocol.CreateToolCallPart(&protocol.ToolCall{
    ID:   "call-123",
    Name: "search",
    Args: map[string]interface{}{"query": "example"},
})

// Create a tool result part (automatically includes AG-UI metadata)
toolResultPart := protocol.CreateToolResultPart(&protocol.ToolResult{
    ToolCallID: "call-123",
    Content:    "Search results...",
    Error:      "",
})
```

### Reading AG-UI Metadata

Both A2A and AG-UI clients can read metadata hints:

```go
// Check if a part is a thinking block
if protocol.IsThinkingPart(part) {
    // Display as thinking block in UI
}

// Get AG-UI event type
eventType := protocol.GetAGUIEventType(part)
// Returns: "thinking", "tool_call", "content_block", etc.

// Get AG-UI block ID
blockID := protocol.GetAGUIBlockID(part)

// Get AG-UI block type
blockType := protocol.GetAGUIBlockType(part)
// Returns: "text", "thinking", "code", etc.
```

### AG-UI Converter

When clients opt-in to AG-UI streaming (via `Accept: application/x-agui-events` header or `format=agui` query parameter), Hector's AG-UI converter (`pkg/agui/converter.go`) reads these metadata hints to generate proper AG-UI events:

```go
// The converter reads AG-UI metadata hints
aguiEventType := part.Metadata.Fields["agui_event_type"].GetStringValue()
aguiBlockType := part.Metadata.Fields["agui_block_type"].GetStringValue()
aguiBlockID := part.Metadata.Fields["agui_block_id"].GetStringValue()

// And generates appropriate AG-UI events
if aguiEventType == "thinking" {
    events = append(events, 
        NewThinkingStartEvent(aguiBlockID, ""),
        NewThinkingDeltaEvent(aguiBlockID, text),
        NewThinkingStopEvent(aguiBlockID, ""),
    )
}
```

## Client-Side Handling

### A2A Native UI (index.html)

Hector's built-in A2A UI recognizes AG-UI metadata to provide rich contextual display:

```javascript
// Check if part is a thinking block
function isThinkingPart(part) {
    if (!part || !part.metadata) return false;
    // Check AG-UI metadata
    return part.metadata.agui_event_type === 'thinking' || 
           part.metadata.agui_block_type === 'thinking';
}

// Extract text, excluding thinking blocks
function extractTextFromParts(parts) {
    return parts
        .filter(part => !isThinkingPart(part))
        .map(part => part.text || '')
        .join('\n');
}

// Extract thinking text separately
function extractThinkingFromParts(parts) {
    return parts
        .filter(isThinkingPart)
        .map(part => part.text || '')
        .join('\n');
}

// Extract tool calls with AG-UI metadata
function extractToolCallsFromParts(parts) {
    return parts
        .filter(part => {
            if (!part.metadata) return false;
            // Check AG-UI metadata (tool calls don't have agui_is_error)
            return part.metadata.agui_event_type === 'tool_call' && 
                   !part.metadata.hasOwnProperty('agui_is_error');
        })
        .map(part => {
            const data = part.data?.data || {};
            const metadata = part.metadata || {};
            return {
                id: metadata.agui_tool_call_id || data.id,
                name: metadata.agui_tool_name || data.name,
                args: data.arguments || {}
            };
        });
}
```

### AG-UI Native UIs

External AG-UI-native UIs can opt-in to receive AG-UI events directly:

```bash
# Request AG-UI format via Accept header
curl -H "Accept: application/x-agui-events" \
     -X POST http://localhost:8080/v1/agents/assistant/stream \
     -d '{"request": {"role": "user", "parts": [{"text": "Hello"}]}}'

# Or via query parameter
curl -X POST 'http://localhost:8080/v1/agents/assistant/stream?format=agui' \
     -d '{"request": {"role": "user", "parts": [{"text": "Hello"}]}}'
```

Theresponse will be AG-UI events generated from the A2A parts' metadata hints:

```
event: message_start
data: {"messageId":"msg-123","contextId":"ctx-456","role":"agent"}

event: thinking_start
data: {"thinkingId":"think-789"}

event: thinking_delta
data: {"thinkingId":"think-789","delta":"Analyzing request..."}

event: thinking_stop
data: {"thinkingId":"think-789"}

event: content_block_start
data: {"blockId":"block-abc","blockType":"text"}

event: content_block_delta
data: {"blockId":"block-abc","delta":"Hello! How can I help?"}

event: content_block_stop
data: {"blockId":"block-abc"}

event: message_stop
data: {"messageId":"msg-123"}
```

## Benefits

### For A2A Clients

- **Semantic Context**: Understand the meaning of parts (thinking vs. regular content)
- **Better UX**: Display thinking blocks, tool calls, and content distinctly
- **Protocol Compliant**: No special handling needed - just read metadata if available

### For AG-UI Clients

- **No Guesswork**: Event types are explicit in metadata
- **Simpler Conversion**: AG-UI converter reads hints directly
- **Consistent Experience**: Same semantics whether A2A or AG-UI format

### For Hector

- **Single Source of Truth**: Part semantics defined once at creation
- **A2A Native**: Core protocol remains A2A
- **AG-UI Compatible**: Trivial conversion to AG-UI events
- **Zero Configuration**: Works out of the box for all agents

## Specification Compliance

### A2A Compliance ✅

From the [A2A specification](https://a2a-protocol.org/latest/specification), Section 6.5 (Part Union Type):

> **Part Object:**
> - `metadata?: Record<string, any>` - Optional metadata associated with this part

The A2A spec explicitly provides the `metadata` field as an extension point without prescribing its contents. Hector's use of AG-UI metadata is fully compliant with A2A.

### AG-UI Event Types

Hector's AG-UI metadata hints map to the 16 standardized AG-UI event types:

1. **Message Events**: `message_start`, `message_delta`, `message_stop`
2. **Content Block Events**: `content_block_start`, `content_block_delta`, `content_block_stop`
3. **Tool Call Events**: `tool_call_start`, `tool_call_stop`
4. **Thinking Events**: `thinking_start`, `thinking_delta`, `thinking_stop`
5. **Task Events**: `task_start`, `task_update`, `task_complete`, `task_error`
6. **Error Events**: `error`

## Configuration

**No configuration required!** AG-UI metadata is automatically added to all A2A parts by default.

The only configuration is for clients opting-in to AG-UI event streaming:

```bash
# Via Accept header
Accept: application/x-agui-events

# Via query parameter
?format=agui
```

## Summary

- **A2A Native**: Hector uses A2A as its core protocol
- **AG-UI Compatible**: AG-UI metadata hints are embedded in A2A parts' `metadata` field
- **Zero Configuration**: AG-UI is a default capability for all agents
- **Backward Compatible**: A2A clients ignore unfamiliar metadata
- **Specification Compliant**: Fully compliant with A2A spec's extension mechanism
- **Rich Semantics**: Thinking blocks, tool calls, and content have explicit types

This approach makes Hector **truly A2A, MCP, and AG-UI native** without conflicting protocols or complex configuration.

