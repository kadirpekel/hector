# Natural Tool Execution Foundation

This document demonstrates the new ingenious tool execution foundation that allows LLMs to naturally embed tool execution claims in their responses.

## How It Works

### 1. LLM Response Format
LLMs can now embed tool calls naturally using the format: `[description]{tool_json}`

Example:
```
User: "Hello, how is the weather today in Berlin?"

LLM Response:
"Hi, let me check the weather for you [checking weather in Berlin]{"name": "weather", "parameters": {"location": "Berlin"}}. 

The weather is quite nice in Berlin with 20 degrees celsius and sunny skies."
```

### 2. Streaming Behavior
- **User sees**: "Hi, let me check the weather for you [checking weather in Berlin]. The weather is quite nice..."
- **Behind the scenes**: The JSON `{"name": "weather", "parameters": {"location": "Berlin"}}` is executed
- **Tool results**: Are made available for the follow-up LLM generation

### 3. Architecture Components

#### Natural Tool Claim Parser
- Detects `[description]{json}` patterns in LLM responses
- Extracts user-friendly descriptions and tool JSON
- Maintains position information for streaming

#### Streaming Tool Interceptor  
- Masks tool JSON during streaming
- Shows only user-friendly descriptions
- Executes tools behind the scenes

#### Two-Phase Response Generation
1. **Phase 1**: Initial response with natural tool claims
2. **Tool Execution**: Happens automatically when claims are detected
3. **Phase 2**: Follow-up response using tool results

### 4. Benefits

- **Natural UX**: Users see friendly descriptions instead of JSON
- **Streaming Compatible**: Works with both streaming and non-streaming
- **Minimal Complexity**: Simple format that LLMs can easily learn
- **Backward Compatible**: Falls back to legacy TOOL_EXEC tags
- **Clean Separation**: Agent services handle execution, reasoning focuses on reasoning

### 5. Example Flow

```
User: "What's the weather in Berlin and Paris?"

LLM Phase 1:
"I'll check the weather in both cities for you [checking Berlin weather]{"name": "weather", "parameters": {"location": "Berlin"}} and [checking Paris weather]{"name": "weather", "parameters": {"location": "Paris"}}."

Tool Execution:
- weather(Berlin) -> "20°C, sunny"  
- weather(Paris) -> "18°C, cloudy"

LLM Phase 2:
"Here are the current weather conditions:

**Berlin**: 20°C and sunny - perfect weather for outdoor activities!
**Paris**: 18°C and cloudy - might want to bring a light jacket.

Both cities have pleasant temperatures today!"
```

### 6. Implementation Details

- **Agent Services**: Handle all tool detection, parsing, and execution
- **Reasoning Engines**: Focus purely on reasoning logic
- **LLM Service**: Handles streaming with automatic tool masking
- **Prompt Instructions**: Guide LLMs to use the natural format

This foundation provides an elegant solution for tool execution that feels natural to users while maintaining clean architecture separation.
