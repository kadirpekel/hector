# 🎯 Dynamic Masking System - Complete Solution

## 📋 **Overview**

The dynamic masking system allows reasoning engines to **dynamically register** what content they want masked from LLM responses, rather than being limited to predefined tags. This provides maximum flexibility while maintaining clean separation between user-facing content and system processing data.

## 🏗️ **Architecture**

### **Core Components**

1. **`masking/interfaces.go`** - Core interfaces and standard processors
2. **`agent/dynamic_masking.go`** - Dynamic masking engine implementation  
3. **`examples/simple-dynamic-masking.go`** - Working example

### **Key Interfaces**

```go
// Reasoning engines implement this to register their masking needs
type MaskingCapable interface {
    RegisterMasking(contentType, pattern string, processor func(string) (string, string))
    RegisterRegexMasking(contentType, pattern string, processor func(string) (string, string), priority int)
    GetExtractedContent(contentType string) []MaskedContent
    GetAllExtractedContent() map[string][]MaskedContent
    ClearExtractedContent()
}

// Reasoning engines that use dynamic masking
type MaskingAwareReasoningEngine interface {
    InitializeMasking(llmService MaskingCapable)
    ProcessExtractedContent(extractedContent map[string][]MaskedContent) error
    Execute(ctx context.Context, query string) (<-chan string, error)
    GetName() string
    GetDescription() string
}
```

## 🚀 **How It Works**

### **1. Dynamic Registration**
Reasoning engines register their masking needs at runtime:

```go
func (engine *MyReasoningEngine) InitializeMasking(llmService MaskingCapable) {
    // Register simple markers
    llmService.RegisterMasking("tools", "TOOL_CALLS:", engine.processTools)
    llmService.RegisterMasking("thoughts", "INTERNAL_THOUGHTS:", engine.processThoughts)
    
    // Register regex patterns
    llmService.RegisterRegexMasking("api_calls", `API_CALL:\s*([A-Z]+\s+[^\n]+)`, engine.processAPI, 200)
    
    // Register custom patterns specific to this engine
    llmService.RegisterMasking("custom", "MY_CUSTOM_TAG:", engine.processCustom)
}
```

### **2. Flexible Processing**
Each content type has its own processor function:

```go
func (engine *MyReasoningEngine) processTools(content string) (userLabel string, rawData string) {
    // Extract tool label from JSON
    label := extractLabelFromJSON(content)
    return "\n🔧 " + label, content  // User sees label, system gets raw JSON
}

func (engine *MyReasoningEngine) processThoughts(content string) (userLabel string, rawData string) {
    return "", content  // Completely hidden from user
}

func (engine *MyReasoningEngine) processAPI(content string) (userLabel string, rawData string) {
    return "\n🌐 Making API request...", content
}
```

### **3. Content Retrieval**
After LLM response, reasoning engines get structured access to extracted content:

```go
func (engine *MyReasoningEngine) ProcessExtractedContent(contentMap map[string][]MaskedContent) error {
    // Process tool calls
    if tools, exists := contentMap["tools"]; exists {
        for _, tool := range tools {
            engine.executeTool(tool.RawData)  // Execute with raw JSON
        }
    }
    
    // Process internal thoughts (for debugging)
    if thoughts, exists := contentMap["thoughts"]; exists {
        for _, thought := range thoughts {
            engine.logThought(thought.RawData)  // Log internal reasoning
        }
    }
    
    // Process API calls
    if apiCalls, exists := contentMap["api_calls"]; exists {
        for _, api := range apiCalls {
            engine.makeAPICall(api.RawData)  // Execute API call
        }
    }
    
    return nil
}
```

## 🎯 **Key Benefits**

### **✅ Complete Flexibility**
- **No predefined tags** - reasoning engines define their own
- **Runtime registration** - patterns can be added dynamically
- **Custom processors** - each engine handles content its own way
- **Regex support** - complex pattern matching available

### **✅ Clean Separation**
- **User sees**: Meaningful progress indicators and labels
- **System gets**: Raw structured data for processing
- **No leakage**: Internal details never shown to users

### **✅ Extensible Design**
- **Multiple content types** per response
- **Priority-based processing** for overlapping patterns
- **Streaming support** with intelligent buffering
- **Standard processors** available for common use cases

### **✅ Type Safety**
- **Structured content** with metadata (position, type, etc.)
- **Type-specific retrieval** - get only what you need
- **Error handling** for malformed patterns

## 📊 **Example Output**

### **Original LLM Response:**
```
I'll help you with that.

INTERNAL_THOUGHTS:
This is complex. I need to search first.

TOOL_CALLS:
{"tool": "search", "params": {"query": "optimization"}, "label": "🔍 Searching..."}

Based on my search results.

API_CALL: GET /api/metrics
Authorization: Bearer token123
```

### **User Sees (Masked):**
```
I'll help you with that.

🔍 Searching...

Based on my search results.

🌐 Making API request...
```

### **System Gets (Structured):**
```go
contentMap = {
    "thoughts": [{"RawData": "This is complex. I need to search first.", "UserLabel": "", ...}],
    "tools": [{"RawData": "{\"tool\": \"search\", ...}", "UserLabel": "🔍 Searching...", ...}],
    "api_calls": [{"RawData": "GET /api/metrics\nAuthorization: ...", "UserLabel": "🌐 Making API request...", ...}]
}
```

## 🔧 **Integration Steps**

### **1. Update LLM Service**
```go
type MaskingAwareLLMService struct {
    llmProvider   LLMProvider
    maskingEngine *agent.DynamicMaskingEngine
    lastExtracted []masking.MaskedContent
}

func (s *MaskingAwareLLMService) RegisterMasking(contentType, pattern string, processor func(string) (string, string)) {
    s.maskingEngine.RegisterSimpleMarker(contentType, pattern, processor)
}

func (s *MaskingAwareLLMService) GenerateLLM(prompt string) (string, int, error) {
    rawResponse, tokens, err := s.llmProvider.Generate(prompt)
    if err != nil {
        return "", tokens, err
    }
    
    maskedResponse, extracted := s.maskingEngine.ProcessResponse(rawResponse)
    s.lastExtracted = extracted
    
    return maskedResponse, tokens, nil
}
```

### **2. Update Reasoning Engine**
```go
type MyReasoningEngine struct {
    // ... existing fields
    processors *masking.StandardMaskingProcessors
    patterns   *masking.CommonMaskingPatterns
}

func (e *MyReasoningEngine) Execute(ctx context.Context, query string) (<-chan string, error) {
    // 1. Initialize masking if not done
    if maskingService, ok := e.llmService.(masking.MaskingCapable); ok {
        e.InitializeMasking(maskingService)
    }
    
    // 2. Generate LLM response (automatically masked)
    response, err := e.llmService.GenerateLLM(prompt)
    
    // 3. Process extracted content
    if maskingService, ok := e.llmService.(masking.MaskingCapable); ok {
        extractedContent := maskingService.GetAllExtractedContent()
        e.ProcessExtractedContent(extractedContent)
    }
    
    return outputChannel, nil
}
```

## 🎛️ **Standard Patterns Available**

The system includes standard patterns that reasoning engines can use:

```go
patterns := masking.NewCommonMaskingPatterns()
processors := masking.NewStandardMaskingProcessors()

// Standard markers
patterns.ToolCallsPattern()        // "TOOL_CALLS:"
patterns.InternalThoughtsPattern() // "INTERNAL_THOUGHTS:"
patterns.ActionPlanPattern()       // "ACTION_PLAN:"
patterns.MemoryOpsPattern()        // "MEMORY_OPS:"

// Standard processors
processors.ToolCallProcessor       // Extracts JSON labels
processors.ThoughtProcessor        // Hides completely
processors.APICallProcessor        // Shows "Making API request..."
processors.CustomProcessor("🎯")   // Custom label
```

## 🚀 **Next Steps**

1. **Integrate with existing LLM service** - Add `MaskingCapable` interface
2. **Update reasoning engines** - Implement `MaskingAwareReasoningEngine`
3. **Test with real responses** - Verify masking works with actual LLM outputs
4. **Add configuration** - Allow YAML-based masking rule configuration
5. **Performance optimization** - Optimize for large responses with many patterns

The dynamic masking system provides the flexibility you requested - reasoning engines can register whatever masking patterns they need at runtime, and retrieve the extracted content in a structured way! 🎯
