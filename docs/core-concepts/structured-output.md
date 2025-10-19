---
title: Structured Output
description: Provider-aware JSON/XML/Enum output for reliable data extraction
---

# Structured Output Guide

Hector provides comprehensive structured output capabilities across all supported LLM providers (OpenAI, Anthropic, Gemini). This guide explains how to leverage provider-specific optimizations for consistent, reliable structured responses.

---

## Overview

Structured output ensures that LLM responses conform to a specific format or schema. This is essential for:

- **Reliable parsing** - No need to parse free-form text
- **Type safety** - Guaranteed data types and structure
- **Downstream integration** - Direct use in APIs, databases, and other systems
- **Consistency** - Predictable outputs across multiple calls

## Provider Comparison

| Feature | OpenAI | Anthropic | Gemini |
|---------|--------|-----------|--------|
| Native JSON Schema | Yes | No | Yes |
| Strict Validation | Yes | No | Partial |
| Response Prefill | No | Yes | No |
| Property Ordering | No | No | Yes |
| Enum Support | Yes | Via prompt | Yes |

### Implementation Details

- **OpenAI**: Uses native `response_format` with JSON schema and strict validation
- **Anthropic**: Uses system prompt instructions + prefill technique for JSON output
- **Gemini**: Uses `responseMimeType` and `responseSchema` with optional property ordering

## Configuration

Structured output is configured via the `StructuredOutputConfig` struct:

```go
type StructuredOutputConfig struct {
    Format           string                 // "json" or "enum"
    Schema           interface{}            // JSON Schema (map[string]interface{})
    Enum             []string               // Enum values (for enum format)
    Prefill          string                 // Anthropic-specific: prefill response
    PropertyOrdering []string               // Gemini-specific: property order
}
```

## JSON Schema Output

### Basic JSON Schema

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "sentiment": map[string]interface{}{
                "type": "string",
                "enum": []string{"positive", "negative", "neutral"},
                "description": "The sentiment of the text",
            },
            "confidence": map[string]interface{}{
                "type": "number",
                "minimum": 0,
                "maximum": 1,
                "description": "Confidence score for the sentiment",
            },
            "reasoning": map[string]interface{}{
                "type": "string",
                "description": "Brief explanation of the sentiment",
            },
        },
        "required": []string{"sentiment", "confidence"},
    },
}

// Use with any provider
text, toolCalls, tokens, err := provider.GenerateStructured(messages, tools, config)
```

### Complex Nested Schema

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "person": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "name": map[string]interface{}{
                        "type": "string",
                    },
                    "age": map[string]interface{}{
                        "type": "number",
                    },
                    "address": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "street": map[string]interface{}{"type": "string"},
                            "city": map[string]interface{}{"type": "string"},
                            "zipcode": map[string]interface{}{"type": "string"},
                        },
                    },
                },
            },
            "skills": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": "string",
                },
            },
        },
        "required": []string{"person", "skills"},
    },
}
```

## Enum Output

For selecting from a fixed set of options:

```go
config := &llms.StructuredOutputConfig{
    Format: "enum",
    Enum:   []string{"Percussion", "String", "Woodwind", "Brass", "Keyboard"},
}

// Gemini will set responseMimeType to "text/x.enum"
text, toolCalls, tokens, err := provider.GenerateStructured(messages, tools, config)
```

## Provider-Specific Optimizations

### OpenAI: Strict JSON Mode

OpenAI's structured output uses strict JSON schema validation:

```go
// Hector automatically enables strict mode for OpenAI
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: yourSchema,
}

// Translates to:
// {
//   "response_format": {
//     "type": "json_schema",
//     "json_schema": {
//       "name": "response",
//       "schema": yourSchema,
//       "strict": true
//     }
//   }
// }
```

### Anthropic: Prefill Technique

Anthropic uses response prefilling to enforce JSON output:

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: yourSchema,
    Prefill: "{\"sentiment\":",  // Forces JSON start
}

// The assistant's response will begin with the prefill,
// ensuring JSON output from the start
```

**Best prefills:**
- `{` - Generic JSON object
- `{"field_name":` - Specific first field
- `[` - JSON array
- `{"type": "` - When type is first field

### Gemini: Property Ordering

Gemini supports property ordering for consistent output:

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: yourSchema,
    PropertyOrdering: []string{"name", "age", "email", "phone"},
}

// Properties will appear in this exact order in the response
```

## Examples

### Example 1: Sentiment Analysis

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "sentiment": map[string]interface{}{
                "type": "string",
                "enum": []string{"positive", "negative", "neutral"},
            },
            "score": map[string]interface{}{
                "type": "number",
                "minimum": -1,
                "maximum": 1,
            },
            "key_phrases": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{"type": "string"},
            },
        },
        "required": []string{"sentiment", "score"},
    },
}

messages := []llms.Message{
    {Role: "user", Content: "I absolutely love this product! It's amazing!"},
}

text, _, _, err := provider.GenerateStructured(messages, nil, config)
// text: {"sentiment": "positive", "score": 0.95, "key_phrases": ["love", "amazing"]}
```

### Example 2: Data Extraction

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "company": map[string]interface{}{
                "type": "string",
            },
            "position": map[string]interface{}{
                "type": "string",
            },
            "salary_range": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "min": map[string]interface{}{"type": "number"},
                    "max": map[string]interface{}{"type": "number"},
                    "currency": map[string]interface{}{"type": "string"},
                },
            },
            "requirements": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{"type": "string"},
            },
        },
        "required": []string{"company", "position"},
    },
}
```

### Example 3: Classification with Streaming

```go
config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "category": map[string]interface{}{
                "type": "string",
                "enum": []string{"bug", "feature", "question", "documentation"},
            },
            "priority": map[string]interface{}{
                "type": "string",
                "enum": []string{"low", "medium", "high", "critical"},
            },
            "tags": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{"type": "string"},
            },
        },
        "required": []string{"category", "priority"},
    },
}

// Streaming works with structured output
chunks, err := provider.GenerateStructuredStreaming(messages, nil, config)
for chunk := range chunks {
    switch chunk.Type {
    case "text":
        fmt.Print(chunk.Text)  // Incremental JSON
    case "done":
        fmt.Printf("\nTokens: %d\n", chunk.Tokens)
    case "error":
        fmt.Printf("Error: %v\n", chunk.Error)
    }
}
```

### Example 4: Multi-Turn with Structured Output

```go
messages := []llms.Message{
    {Role: "user", Content: "Extract key information from this resume: John Doe, 5 years exp..."},
    {Role: "assistant", Content: `{"name": "John Doe", "experience_years": 5, ...}`},
    {Role: "user", Content: "Now add a relevance score for a software engineer position"},
}

config := &llms.StructuredOutputConfig{
    Format: "json",
    Schema: resumeWithScoreSchema,
}

text, _, _, err := provider.GenerateStructured(messages, nil, config)
```

## Configuration via YAML

You can configure structured output in agent configs:

```yaml
agents:
  - name: sentiment_analyzer
    description: Analyzes sentiment with structured output
    llm:
      type: openai
      model: gpt-4
      structured_output:
        format: json
        schema:
          type: object
          properties:
            sentiment:
              type: string
              enum: ["positive", "negative", "neutral"]
            confidence:
              type: number
              minimum: 0
              maximum: 1
          required: ["sentiment", "confidence"]
```

## Best Practices

### Schema Design

1. **Keep schemas simple** - Complex nested schemas can be harder for LLMs to follow
2. **Use descriptive field names** - Clear names help the LLM understand what to extract
3. **Include required fields** - Specify which fields are mandatory
4. **Add constraints** - Use min/max for numbers, enums for strings

### Provider Selection

1. **OpenAI** - Best for strict JSON validation and complex schemas
2. **Anthropic** - Good for JSON with prefill technique, excellent reasoning
3. **Gemini** - Best for property ordering and enum outputs

### Error Handling

```go
text, _, _, err := provider.GenerateStructured(messages, nil, config)
if err != nil {
    // Handle provider-specific errors
    log.Printf("Structured output failed: %v", err)
    return
}

// Validate JSON before using
var result map[string]interface{}
if err := json.Unmarshal([]byte(text), &result); err != nil {
    log.Printf("Invalid JSON response: %v", err)
    return
}
```

### Performance Considerations

1. **Use streaming for long responses** - Better user experience
2. **Cache schemas** - Avoid recreating complex schemas
3. **Choose appropriate providers** - Match provider strengths to your use case
4. **Track token usage** - Structured output may use more tokens than free-form text
