# Config Pointer Pattern Documentation

## Rule Statement

**Decision Rule for Config Struct Fields:**

```
If a nested config has IsEnabled() method or is semantically optional → use POINTER
If it's always-present with defaults → use VALUE
```

**Maps always use pointers:**
```
map[string]*Type  // Always
```

## Pattern Overview

### Always-Present Configs (Value Types)

These configs are **always initialized** with defaults and are passed to constructors by value:

- `PromptConfig` - Always present, has defaults
- `MemoryConfig` - Always present, has defaults  
- `ReasoningConfig` - Always present, has defaults
- `SearchConfig` - Always present, has defaults

**Usage:**
```go
// In AgentConfig struct:
Prompt   PromptConfig   `yaml:"prompt,omitempty"`
Memory   MemoryConfig  `yaml:"memory,omitempty"`
Reasoning ReasoningConfig `yaml:"reasoning,omitempty"`
Search   SearchConfig  `yaml:"search,omitempty"`

// Access (no nil check needed):
agentConfig.Prompt.SetDefaults()
agentConfig.Memory.SetDefaults()
```

### Optional Configs (Pointer Types)

These configs **may be absent** and require nil checks:

- `*TaskConfig` - Has `IsEnabled()` method → OPTIONAL
- `*SecurityConfig` - Has `IsEnabled()` method → OPTIONAL
- `*AgentCredentials` - Semantically optional
- `*StructuredOutputConfig` - Semantically optional  
- `*A2ACardConfig` - Semantically optional

**Usage:**
```go
// In AgentConfig struct:
Task             *TaskConfig              `yaml:"task,omitempty"`
Security         *SecurityConfig          `yaml:"security,omitempty"`
Credentials      *AgentCredentials        `yaml:"credentials,omitempty"`
StructuredOutput *StructuredOutputConfig  `yaml:"structured_output,omitempty"`
A2A              *A2ACardConfig         `yaml:"a2a,omitempty"`

// Access (nil check required):
if agentConfig.Task != nil && agentConfig.Task.IsEnabled() {
    // use task config
}
```

### Map Configs (Always Pointers)

All config maps use pointers to allow presence checking and mutability:

```go
// In Config struct:
LLMs           map[string]*LLMProviderConfig
Databases      map[string]*DatabaseProviderConfig
Embedders      map[string]*EmbedderProviderConfig
Agents         map[string]*AgentConfig
DocumentStores map[string]*DocumentStoreConfig
SessionStores  map[string]*SessionStoreConfig

// In ToolConfigs:
Tools          map[string]*ToolConfig

// In PluginConfigs:
LLMProviders        map[string]*PluginConfig
DatabaseProviders   map[string]*PluginConfig
EmbedderProviders   map[string]*PluginConfig
ToolProviders       map[string]*PluginConfig
ReasoningStrategies map[string]*PluginConfig

// In SecurityConfig:
Schemes map[string]*SecurityScheme
```

## Implementation Guidelines

### 1. Access Pattern for Values
```go
// No nil check needed
agentConfig.Prompt.SetDefaults()
agentConfig.Memory.Strategy = "buffer"
```

### 2. Access Pattern for Optional
```go
// Nil check required
if agentConfig.Task != nil && agentConfig.Task.IsEnabled() {
    // use task config
}

if agentConfig.Security != nil {
    agentConfig.Security.SetDefaults()
}
```

### 3. Map Access Pattern
```go
// Already pointers, no dereference needed
for name, llm := range cfg.LLMs {
    if llm != nil {
        llm.SetDefaults()
    }
}

// Direct assignment
cfg.LLMs["my-llm"] = &config.LLMProviderConfig{
    Type: "openai",
    // ...
}
```

### 4. Factory Function Signatures

Factory functions that create from configs should accept pointers:

```go
// ✓ CORRECT
func CreateProviderFromConfig(config *SomeConfig) (*Provider, error)

// ✗ WRONG
func CreateProviderFromConfig(config SomeConfig) (*Provider, error)
```

### 5. Passing Configs Between Functions

```go
// When passing value types (always-present):
func NewService(cfg PromptConfig) *Service { ... }

// When passing optional configs (pointers):
func NewTaskService(cfg *TaskConfig) (TaskService, error) {
    if cfg == nil {
        return nil, nil // or error based on requirements
    }
    // ...
}
```

## SetDefaults() Pattern

### Value Types
```go
func (c *AgentConfig) SetDefaults() {
    // Always called
    c.Prompt.SetDefaults()
    c.Memory.SetDefaults()
    c.Reasoning.SetDefaults()
    c.Search.SetDefaults()
}
```

### Optional Types
```go
func (c *AgentConfig) SetDefaults() {
    // Nil check required
    if c.Task != nil {
        c.Task.SetDefaults()
    }
    if c.Security != nil {
        c.Security.SetDefaults()
    }
}
```

## Validate() Pattern

### Value Types
```go
func (c *AgentConfig) Validate() error {
    // Direct validation
    if err := c.Prompt.Validate(); err != nil {
        return err
    }
    // ...
}
```

### Optional Types
```go
func (c *AgentConfig) Validate() error {
    // Nil check before validation
    if c.Task != nil {
        if err := c.Task.Validate(); err != nil {
            return err
        }
    }
    // ...
}
```

## How to Determine If a Config Should Be Pointer

### Checklist

1. **Does it have an `IsEnabled()` method?**
   - ✓ YES → Use pointer
   - Example: `TaskConfig`, `SecurityConfig`

2. **Is it semantically optional?**
   - ✓ YES → Use pointer
   - Example: `Credentials` (only for A2A agents)

3. **Is it conditionally used?**
   - ✓ YES → Use pointer
   - Example: `StructuredOutput` (feature flag)

4. **Is it always initialized with defaults?**
   - ✓ YES → Use value
   - Example: `Prompt`, `Memory`, `Reasoning`

### Decision Tree

```
┌─────────────────────────────────┐
│ Is config in a MAP?             │
└────────┬────────────────────────┘
         │ YES
         └─────────→ Use POINTER
         
         NO
         │
         ▼
┌─────────────────────────────────┐
│ Does it have IsEnabled()?        │
└────────┬────────────────────────┘
         │ YES
         └─────────→ Use POINTER
         
         NO
         │
         ▼
┌─────────────────────────────────┐
│ Is it always present?            │
└────────┬────────────────────────┘
         │ YES
         └─────────→ Use VALUE
         
         NO
         │
         └─────────→ Use POINTER
```

## Current Implementation Status

### ✓ Correctly Implemented

- **Maps**: All maps use `map[string]*Type` ✓
- **Optional Struct Fields**: `Task`, `Security`, `Credentials`, `StructuredOutput`, `A2A` ✓
- **Always-Present Fields**: `Prompt`, `Memory`, `Reasoning`, `Search` ✓
- **Factory Functions**: All accept pointers ✓
- **SetDefaults()**: Proper nil checks for optional fields ✓
- **Validate()**: Proper nil checks for optional fields ✓

## Verification Commands

Run these to verify the pattern:

```bash
# Check for incorrect map declarations (should be empty)
grep -r "map\[string\](?!\*)config\." pkg/ --include="*.go" | grep -v "_test.go"

# Check for struct fields that should be pointers but aren't
grep -r "func.*IsEnabled" pkg/config/types.go -A 1 | grep -E "^\s+\w+\s+.*Config[^*]" | grep -v "*"

# Verify all access sites use nil checks for optional configs
grep -r "\.Task\." pkg/ --include="*.go" | grep -v "_test.go" | grep -v "!= nil"

# Should show all accesses are guarded
```

## Migration Guide

If adding a new config field:

1. **Determine if it's optional:**
   - Does it have `IsEnabled()`? → Pointer
   - Is it semantically optional? → Pointer
   - Always present? → Value

2. **Update struct:**
   ```go
   // Optional
   MyConfig *MyConfigType `yaml:"my_config,omitempty"`
   
   // Always present
   MyConfig MyConfigType `yaml:"my_config,omitempty"`
   ```

3. **Update SetDefaults():**
   ```go
   // Optional
   if c.MyConfig != nil {
       c.MyConfig.SetDefaults()
   }
   
   // Always present
   c.MyConfig.SetDefaults()
   ```

4. **Update Validate():**
   ```go
   // Optional
   if c.MyConfig != nil {
       if err := c.MyConfig.Validate(); err != nil {
           return err
       }
   }
   
   // Always present
   if err := c.MyConfig.Validate(); err != nil {
       return err
   }
   ```

5. **Update all access sites:**
   ```go
   // Optional
   if agentConfig.MyConfig != nil {
       // use config
   }
   
   // Always present
   // use config directly
   ```

## Examples

### Creating Config with Optional Fields

```go
// ✓ CORRECT
cfg := &config.AgentConfig{
    Name: "my-agent",
    Task: &config.TaskConfig{
        Backend: "memory",
    },
    Security: nil, // Not configured
}

// ✗ WRONG
cfg := &config.AgentConfig{
    Name: "my-agent",
    Task: config.TaskConfig{  // Missing &
        Backend: "memory",
    },
}
```

### Accessing Optional Configs

```go
// ✓ CORRECT
if agentConfig.Task != nil && agentConfig.Task.IsEnabled() {
    // use task
}

// ✗ WRONG
agentConfig.Task.IsEnabled()  // Panic if nil!
```

## Summary

The pattern creates a **consistent, type-safe configuration system** where:

1. **Type indicates presence**: Pointer = optional, Value = always present
2. **Compiler enforces**: Nil checks required for pointers
3. **Semantics match usage**: Optional configs are truly optional
4. **Consistent across codebase**: All follows the same rule

This follows Go's pointer conventions and creates self-documenting code.

