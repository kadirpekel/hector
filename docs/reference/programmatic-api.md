---
title: Programmatic API Reference
description: Complete reference for Hector's programmatic API builders
---

# Programmatic API Reference

Complete reference for building agents programmatically in Go.

---

## Package: `github.com/kadirpekel/hector/pkg/hector`

All programmatic API builders are in the `hector` package.

---

## Agent Builder

### `NewAgent(id string) *AgentBuilder`

Creates a new agent builder.

**Parameters:**
- `id` (string): Unique agent identifier (required)

**Returns:** `*AgentBuilder`

**Example:**
```go
builder := hector.NewAgent("my-agent")
```

### AgentBuilder Methods

#### Identity

```go
WithName(name string) *AgentBuilder
WithDescription(desc string) *AgentBuilder
```

#### Core Components

```go
WithLLMProvider(provider llms.LLMProvider) *AgentBuilder
WithReasoningStrategy(strategy reasoning.ReasoningStrategy) *AgentBuilder
WithWorkingMemory(strategy memory.WorkingMemoryStrategy) *AgentBuilder
WithLongTermMemory(strategy memory.LongTermMemoryStrategy, config memory.LongTermConfig) *AgentBuilder
```

#### Services

```go
WithContext(service reasoning.ContextService) *AgentBuilder
WithTask(service reasoning.TaskService) *AgentBuilder
WithSession(service reasoning.SessionService) *AgentBuilder
```

#### Prompts

```go
WithSystemPrompt(prompt string) *AgentBuilder
WithPromptSlots(slots *reasoning.PromptSlots) *AgentBuilder
```

#### Tools

```go
WithTool(tool tools.Tool) *AgentBuilder
WithTools(tools ...tools.Tool) *AgentBuilder
```

#### A2A Configuration

```go
WithA2ACard(card *config.A2ACardConfig) *AgentBuilder
WithPreferredTransport(transport string) *AgentBuilder
```

#### Security

```go
WithSecurity(security *config.SecurityConfig) *AgentBuilder
```

#### Other

```go
WithRegistry(registry *agent.AgentRegistry) *AgentBuilder
WithBaseURL(url string) *AgentBuilder
WithStructuredOutput(cfg *config.StructuredOutputConfig) *AgentBuilder
WithDocsFolder(folder string) *AgentBuilder
EnableTools(enabled bool) *AgentBuilder
```

#### Build

```go
Build() (*agent.Agent, error)
```

Builds the agent from the builder configuration.

---

## LLM Provider Builder

### `NewLLMProvider(providerType string) *LLMProviderBuilder`

Creates a new LLM provider builder.

**Supported Types:**
- `"openai"` - OpenAI API
- `"anthropic"` - Anthropic Claude API
- `"gemini"` - Google Gemini API
- `"ollama"` - Local Ollama instance

**Example:**
```go
builder := hector.NewLLMProvider("openai")
```

### LLMProviderBuilder Methods

```go
Model(model string) *LLMProviderBuilder
APIKey(key string) *LLMProviderBuilder
APIKeyFromEnv(envVar string) *LLMProviderBuilder
Host(host string) *LLMProviderBuilder
Temperature(temp float64) *LLMProviderBuilder  // 0.0-2.0
MaxTokens(max int) *LLMProviderBuilder
Timeout(seconds int) *LLMProviderBuilder
MaxRetries(max int) *LLMProviderBuilder
RetryDelay(seconds int) *LLMProviderBuilder
StructuredOutput(cfg *config.StructuredOutputConfig) *LLMProviderBuilder
WithStructuredOutput(builder *StructuredOutputBuilder) *LLMProviderBuilder
Build() (llms.LLMProvider, error)
```

---

## Reasoning Builder

### `NewReasoning(strategyType string) *ReasoningBuilder`

Creates a new reasoning strategy builder.

**Supported Types:**
- `"chain-of-thought"` - Single-agent iterative reasoning
- `"supervisor"` - Multi-agent orchestration

**Example:**
```go
builder := hector.NewReasoning("chain-of-thought")
```

### ReasoningBuilder Methods

```go
MaxIterations(max int) *ReasoningBuilder
EnableStreaming(enable bool) *ReasoningBuilder
ShowTools(show bool) *ReasoningBuilder
ShowThinking(show bool) *ReasoningBuilder
Build() (reasoning.ReasoningStrategy, error)
GetConfig() ReasoningConfig
```

---

## Working Memory Builder

### `NewWorkingMemory(strategy string) *WorkingMemoryBuilder`

Creates a new working memory builder.

**Supported Strategies:**
- `"buffer"` - Fixed-size buffer
- `"buffer_window"` - Fixed message count window
- `"sliding_window"` - Token-based sliding window
- `"summary_buffer"` - Automatic summarization

**Example:**
```go
builder := hector.NewWorkingMemory("summary_buffer")
```

### WorkingMemoryBuilder Methods

```go
WindowSize(size int) *WorkingMemoryBuilder
Budget(tokens int) *WorkingMemoryBuilder
Threshold(threshold float64) *WorkingMemoryBuilder
Target(target float64) *WorkingMemoryBuilder
WithLLMProvider(provider llms.LLMProvider) *WorkingMemoryBuilder
Build() (memory.WorkingMemoryStrategy, error)
```

---

## Long-Term Memory Builder

### `NewLongTermMemory() *LongTermMemoryBuilder`

Creates a new long-term memory builder.

**Example:**
```go
builder := hector.NewLongTermMemory()
```

### LongTermMemoryBuilder Methods

```go
Enabled(enabled bool) *LongTermMemoryBuilder
Collection(name string) *LongTermMemoryBuilder
StorageScope(scope memory.StorageScope) *LongTermMemoryBuilder
BatchSize(size int) *LongTermMemoryBuilder
AutoRecall(enabled bool) *LongTermMemoryBuilder
RecallLimit(limit int) *LongTermMemoryBuilder
WithDatabase(db databases.DatabaseProvider) *LongTermMemoryBuilder
WithEmbedder(embedder embedders.EmbedderProvider) *LongTermMemoryBuilder
Build() (memory.LongTermMemoryStrategy, memory.LongTermConfig, error)
```

---

## Context Service Builder (RAG)

### `NewContextService() *ContextServiceBuilder`

Creates a new context service builder for RAG.

**Example:**
```go
builder := hector.NewContextService()
```

### ContextServiceBuilder Methods

```go
WithDatabase(db databases.DatabaseProvider) *ContextServiceBuilder
WithEmbedder(embedder embedders.EmbedderProvider) *ContextServiceBuilder
TopK(k int) *ContextServiceBuilder
Threshold(threshold float64) *ContextServiceBuilder
PreserveCase(preserve bool) *ContextServiceBuilder
WithDocumentStores(stores ...*config.DocumentStoreConfig) *ContextServiceBuilder
IncludeContext(include bool) *ContextServiceBuilder
Build() (reasoning.ContextService, error)
GetIncludeContext() bool
```

---

## Task Service Builder

### `NewTaskService() *TaskServiceBuilder`

Creates a new task service builder.

**Example:**
```go
builder := hector.NewTaskService()
```

### TaskServiceBuilder Methods

#### Core Configuration

```go
Backend(backend string) *TaskServiceBuilder  // "memory" or "sql"
WorkerPool(size int) *TaskServiceBuilder
WithSQLConfig(cfg *config.TaskSQLConfig) *TaskServiceBuilder
InputTimeout(seconds int) *TaskServiceBuilder  // Timeout for INPUT_REQUIRED state
Timeout(seconds int) *TaskServiceBuilder      // Timeout for async task execution
Build() (reasoning.TaskService, error)
```

#### Human-in-the-Loop (HITL) Configuration

```go
WithHITL(cfg *config.HITLConfig) *TaskServiceBuilder
HITL() *HITLConfigBuilder
```

**HITLConfigBuilder Methods:**
```go
Mode(mode string) *HITLConfigBuilder  // "auto", "blocking", or "async"
Build() *config.HITLConfig
```

**Example:**
```go
taskBuilder := hector.NewTaskService().
    Backend("sql").
    WorkerPool(10).
    HITL().
        Mode("async").  // Enable async HITL (requires session_store)
        Build()
```

#### Checkpoint Configuration

```go
WithCheckpoint(cfg *config.CheckpointConfig) *TaskServiceBuilder
Checkpoint() *CheckpointConfigBuilder
```

**CheckpointConfigBuilder Methods:**
```go
Enabled(enabled bool) *CheckpointConfigBuilder
Strategy(strategy string) *CheckpointConfigBuilder  // "event", "interval", or "hybrid"
Interval() *CheckpointIntervalConfigBuilder
Recovery() *CheckpointRecoveryConfigBuilder
Build() *config.CheckpointConfig
```

**CheckpointIntervalConfigBuilder Methods:**
```go
EveryNIterations(n int) *CheckpointIntervalConfigBuilder
AfterToolCalls(enabled bool) *CheckpointIntervalConfigBuilder
BeforeLLMCalls(enabled bool) *CheckpointIntervalConfigBuilder
Build() *config.CheckpointIntervalConfig
```

**CheckpointRecoveryConfigBuilder Methods:**
```go
AutoResume(enabled bool) *CheckpointRecoveryConfigBuilder
AutoResumeHITL(enabled bool) *CheckpointRecoveryConfigBuilder
ResumeTimeout(seconds int) *CheckpointRecoveryConfigBuilder
Build() *config.CheckpointRecoveryConfig
```

**Example:**
```go
taskBuilder := hector.NewTaskService().
    Backend("sql").
    WorkerPool(10).
    Checkpoint().
        Enabled(true).
        Strategy("hybrid").
        Interval().
            EveryNIterations(5).
            AfterToolCalls(true).
            Build().
        Recovery().
            AutoResume(true).
            AutoResumeHITL(false).
            ResumeTimeout(3600).
            Build().
        Build()
```

**Complete Example with HITL and Checkpoint:**
```go
// Build session service (required for async HITL)
sessionService := hector.NewSessionService("my-agent").
    Backend("sql").
    SQLConfig().
        Driver("sqlite").
        Database("./sessions.db").
        Build().
    Build()

// Build task service with HITL and checkpoint
taskService := hector.NewTaskService().
    Backend("sql").
    WorkerPool(10).
    InputTimeout(600).
    Timeout(3600).
    SQLConfig().
        Driver("sqlite").
        Database("./tasks.db").
        Build().
    HITL().
        Mode("async").
        Build().
    Checkpoint().
        Enabled(true).
        Strategy("hybrid").
        Interval().
            EveryNIterations(5).
            Build().
        Recovery().
            AutoResume(true).
            ResumeTimeout(3600).
            Build().
        Build().
    Build()

// Build agent with task service
agent, err := hector.NewAgent("assistant").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    WithWorkingMemory(workingMemory).
    WithSession(sessionService).
    WithTask(taskService).
    Build()
```

---

## Session Service Builder

### `NewSessionService() *SessionServiceBuilder`

Creates a new session service builder.

**Example:**
```go
builder := hector.NewSessionService()
```

### SessionServiceBuilder Methods

```go
Backend(backend string) *SessionServiceBuilder  // "memory" or "sql"
WithSQLConfig(cfg *config.SQLConfig) *SessionServiceBuilder
WithRateLimit(config *config.RateLimitConfig) *SessionServiceBuilder
Build() (reasoning.SessionService, error)
```

---

## Database Builder

### `NewDatabase(dbType string) *DatabaseBuilder`

Creates a new database provider builder.

**Supported Types:**
- `"qdrant"` - Qdrant vector database

**Example:**
```go
builder := hector.NewDatabase("qdrant")
```

### DatabaseBuilder Methods

```go
Host(host string) *DatabaseBuilder
Port(port int) *DatabaseBuilder
APIKey(key string) *DatabaseBuilder
UseTLS(use bool) *DatabaseBuilder
Build() (databases.DatabaseProvider, error)
```

---

## Embedder Builder

### `NewEmbedder(embedderType string) *EmbedderBuilder`

Creates a new embedder provider builder.

**Supported Types:**
- `"openai"` - OpenAI embeddings
- `"ollama"` - Local Ollama embeddings

**Example:**
```go
builder := hector.NewEmbedder("openai")
```

### EmbedderBuilder Methods

```go
Model(model string) *EmbedderBuilder
APIKey(key string) *EmbedderBuilder
APIKeyFromEnv(envVar string) *EmbedderBuilder
Host(host string) *EmbedderBuilder
Build() (embedders.EmbedderProvider, error)
```

---

## Structured Output Builder

### `NewStructuredOutput() *StructuredOutputBuilder`

Creates a new structured output builder.

**Example:**
```go
builder := hector.NewStructuredOutput()
```

### StructuredOutputBuilder Methods

```go
Format(format string) *StructuredOutputBuilder  // "json", "yaml", etc.
Schema(schema map[string]interface{}) *StructuredOutputBuilder
Enum(values []string) *StructuredOutputBuilder
Prefill(value string) *StructuredOutputBuilder
PropertyOrdering(order []string) *StructuredOutputBuilder
Build() *config.StructuredOutputConfig
```

---

## Security Builder

### `NewSecurityBuilder(cfg *config.SecurityConfig) *SecurityBuilder`

Creates a new security configuration builder.

**Example:**
```go
builder := hector.NewSecurityBuilder(nil)
```

### SecurityBuilder Methods

```go
JWKSURL(url string) *SecurityBuilder
Issuer(issuer string) *SecurityBuilder
Audience(audience string) *SecurityBuilder
WithScheme(name string, scheme *config.SecurityScheme) *SecurityBuilder
Scheme(name string) *SecuritySchemeBuilder
Require(requirement map[string][]string) *SecurityBuilder
Build() *config.SecurityConfig
```

---

## A2A Card Builder

### `NewA2ACardBuilder(cfg *config.A2ACardConfig) *A2ACardBuilder`

Creates a new A2A card builder.

**Example:**
```go
builder := hector.NewA2ACardBuilder(nil)
```

### A2ACardBuilder Methods

```go
Version(version string) *A2ACardBuilder
InputModes(modes []string) *A2ACardBuilder
AddInputMode(mode string) *A2ACardBuilder
OutputModes(modes []string) *A2ACardBuilder
AddOutputMode(mode string) *A2ACardBuilder
Skills(skills []config.A2ASkillConfig) *A2ACardBuilder
AddSkill(skill config.A2ASkillConfig) *A2ACardBuilder
Skill() *A2ASkillBuilder
Provider(provider *config.A2AProviderConfig) *A2ACardBuilder
PreferredTransport(transport string) *A2ACardBuilder
DocumentationURL(url string) *A2ACardBuilder
Build() *config.A2ACardConfig
```

---

## Document Store Builder

### `NewDocumentStore(name, source string) *DocumentStoreBuilder`

Creates a new document store builder. `source` must be one of: `"directory"`, `"sql"`, or `"api"`.

**Example:**
```go
// Directory source
builder := hector.NewDocumentStore("docs", "directory").
    Path("./docs").
    ChunkSize(800).
    ChunkOverlap(100).
    IncludePatterns([]string{"*.md", "*.txt"}).
    WatchChanges(true)

// SQL source
sqlBuilder := hector.NewDocumentStoreSQL("mydb").
    Driver("postgres").
    Host("localhost").
    Port(5432).
    Username("user").
    Password("pass")

tableBuilder := hector.NewDocumentStoreSQLTable("articles", []string{"title", "content"}, "id").
    UpdatedColumn("updated_at").
    MetadataColumns([]string{"author", "category"})

builder := hector.NewDocumentStore("articles", "sql").
    WithSQLBuilder(sqlBuilder).
    WithSQLTableBuilder(tableBuilder)

// API source
authBuilder := hector.NewDocumentStoreAPIAuth("bearer").
    Token("my-token")

endpointBuilder := hector.NewDocumentStoreAPIEndpoint("/api/articles").
    ContentField("content").
    IDField("id").
    WithAuthBuilder(authBuilder)

apiBuilder := hector.NewDocumentStoreAPI("https://api.example.com").
    WithAuthBuilder(authBuilder).
    WithEndpointBuilder(endpointBuilder)

builder := hector.NewDocumentStore("api-docs", "api").
    WithAPIBuilder(apiBuilder)
```

### DocumentStoreBuilder Methods

#### Core Configuration

```go
Path(path string) *DocumentStoreBuilder                    // Required for directory source
IncludePatterns(patterns []string) *DocumentStoreBuilder
ExcludePatterns(patterns []string) *DocumentStoreBuilder
AdditionalExcludes(patterns []string) *DocumentStoreBuilder
WatchChanges(watch bool) *DocumentStoreBuilder             // Directory source only
MaxFileSize(size int64) *DocumentStoreBuilder              // Directory source only
IncrementalIndexing(enabled bool) *DocumentStoreBuilder
```

#### Chunking Configuration

```go
ChunkSize(size int) *DocumentStoreBuilder
ChunkOverlap(overlap int) *DocumentStoreBuilder
ChunkStrategy(strategy string) *DocumentStoreBuilder       // "simple", "overlapping", "semantic"
```

#### Metadata Configuration

```go
ExtractMetadata(enabled bool) *DocumentStoreBuilder
MetadataLanguages(languages []string) *DocumentStoreBuilder
```

#### Performance Configuration

```go
MaxConcurrentFiles(max int) *DocumentStoreBuilder
ShowProgress(show bool) *DocumentStoreBuilder
VerboseProgress(verbose bool) *DocumentStoreBuilder
EnableCheckpoints(enabled bool) *DocumentStoreBuilder
QuietMode(quiet bool) *DocumentStoreBuilder
```

#### SQL Configuration

```go
WithSQLConfig(sqlConfig *config.DocumentStoreSQLConfig) *DocumentStoreBuilder
WithSQLBuilder(sqlBuilder *DocumentStoreSQLBuilder) *DocumentStoreBuilder
SQLMaxRows(maxRows int) *DocumentStoreBuilder
WithSQLTable(tableConfig *config.DocumentStoreSQLTableConfig) *DocumentStoreBuilder
WithSQLTableBuilder(tableBuilder *DocumentStoreSQLTableBuilder) *DocumentStoreBuilder
```

#### API Configuration

```go
WithAPIConfig(apiConfig *config.DocumentStoreAPIConfig) *DocumentStoreBuilder
WithAPIBuilder(apiBuilder *DocumentStoreAPIBuilder) *DocumentStoreBuilder
```

#### Build

```go
Build() (*config.DocumentStoreConfig, error)
```

### DocumentStoreSQLBuilder Methods

```go
NewDocumentStoreSQL(database string) *DocumentStoreSQLBuilder
Driver(driver string) *DocumentStoreSQLBuilder              // "postgres", "mysql", "sqlite3"
Host(host string) *DocumentStoreSQLBuilder
Port(port int) *DocumentStoreSQLBuilder
Username(username string) *DocumentStoreSQLBuilder
Password(password string) *DocumentStoreSQLBuilder
SSLMode(mode string) *DocumentStoreSQLBuilder
Build() (*config.DocumentStoreSQLConfig, error)
```

### DocumentStoreSQLTableBuilder Methods

```go
NewDocumentStoreSQLTable(table string, columns []string, idColumn string) *DocumentStoreSQLTableBuilder
UpdatedColumn(column string) *DocumentStoreSQLTableBuilder
WhereClause(clause string) *DocumentStoreSQLTableBuilder
MetadataColumns(columns []string) *DocumentStoreSQLTableBuilder
Build() (*config.DocumentStoreSQLTableConfig, error)
```

### DocumentStoreAPIBuilder Methods

```go
NewDocumentStoreAPI(baseURL string) *DocumentStoreAPIBuilder
WithAuth(authConfig *config.DocumentStoreAPIAuthConfig) *DocumentStoreAPIBuilder
WithAuthBuilder(authBuilder *DocumentStoreAPIAuthBuilder) *DocumentStoreAPIBuilder
WithEndpoint(endpointConfig *config.DocumentStoreAPIEndpointConfig) *DocumentStoreAPIBuilder
WithEndpointBuilder(endpointBuilder *DocumentStoreAPIEndpointBuilder) *DocumentStoreAPIBuilder
Build() (*config.DocumentStoreAPIConfig, error)
```

### DocumentStoreAPIAuthBuilder Methods

```go
NewDocumentStoreAPIAuth(authType string) *DocumentStoreAPIAuthBuilder  // "bearer", "basic", "apikey"
Token(token string) *DocumentStoreAPIAuthBuilder                      // For bearer auth
Username(username string) *DocumentStoreAPIAuthBuilder                // For basic auth
Password(password string) *DocumentStoreAPIAuthBuilder                 // For basic auth
Header(header string) *DocumentStoreAPIAuthBuilder                    // For apikey auth
Extra(extra map[string]string) *DocumentStoreAPIAuthBuilder
Build() (*config.DocumentStoreAPIAuthConfig, error)
```

### DocumentStoreAPIEndpointBuilder Methods

```go
NewDocumentStoreAPIEndpoint(path string) *DocumentStoreAPIEndpointBuilder
Method(method string) *DocumentStoreAPIEndpointBuilder
Params(params map[string]string) *DocumentStoreAPIEndpointBuilder
Headers(headers map[string]string) *DocumentStoreAPIEndpointBuilder
Body(body string) *DocumentStoreAPIEndpointBuilder
WithAuth(authConfig *config.DocumentStoreAPIAuthConfig) *DocumentStoreAPIEndpointBuilder
WithAuthBuilder(authBuilder *DocumentStoreAPIAuthBuilder) *DocumentStoreAPIEndpointBuilder
IDField(field string) *DocumentStoreAPIEndpointBuilder
ContentField(field string) *DocumentStoreAPIEndpointBuilder
MetadataFields(fields []string) *DocumentStoreAPIEndpointBuilder
UpdatedField(field string) *DocumentStoreAPIEndpointBuilder
WithPagination(paginationConfig *config.DocumentStoreAPIPaginationConfig) *DocumentStoreAPIEndpointBuilder
WithPaginationBuilder(paginationBuilder *DocumentStoreAPIPaginationBuilder) *DocumentStoreAPIEndpointBuilder
Build() (*config.DocumentStoreAPIEndpointConfig, error)
```

### DocumentStoreAPIPaginationBuilder Methods

```go
NewDocumentStoreAPIPagination(paginationType string) *DocumentStoreAPIPaginationBuilder  // "offset", "cursor", "page", "link"
PageParam(param string) *DocumentStoreAPIPaginationBuilder
SizeParam(param string) *DocumentStoreAPIPaginationBuilder
MaxPages(maxPages int) *DocumentStoreAPIPaginationBuilder
PageSize(size int) *DocumentStoreAPIPaginationBuilder
NextField(field string) *DocumentStoreAPIPaginationBuilder
DataField(field string) *DocumentStoreAPIPaginationBuilder
Build() (*config.DocumentStoreAPIPaginationConfig, error)
```

---

## Observability Builder

### `NewObservability() *ObservabilityBuilder`

Creates a new observability builder for configuring tracing and metrics.

**Example:**
```go
obsConfig, _ := hector.NewObservability().
    EnableMetrics(true).
    WithTracing(hector.NewTracing().
        Enable(true).
        EndpointURL("http://jaeger:4317").
        SamplingRate(0.1).
        ServiceName("my-app")).
    Build()

// Initialize observability manager
obsMgr := observability.NewManager(obsConfig)
if err := obsMgr.Initialize(context.Background()); err != nil {
    log.Fatal(err)
}
```

### ObservabilityBuilder Methods

```go
EnableMetrics(enabled bool) *ObservabilityBuilder
WithTracing(tracingBuilder *TracingBuilder) *ObservabilityBuilder
Build() (observability.Config, error)
```

### TracingBuilder Methods

```go
NewTracing() *TracingBuilder
Enable(enabled bool) *TracingBuilder
ExporterType(exporterType string) *TracingBuilder
EndpointURL(url string) *TracingBuilder
SamplingRate(rate float64) *TracingBuilder                  // 0.0 to 1.0
ServiceName(name string) *TracingBuilder
Build() (observability.TracerConfig, error)
```

---

## Config Agent Builder

### `NewConfigAgentBuilder(cfg *config.Config) (*ConfigAgentBuilder, error)`

Creates a builder that converts configuration to agents using the programmatic API.

**Example:**
```go
cfg, _ := config.LoadConfig(config.LoaderOptions{Path: "agents.yaml"})
builder, _ := hector.NewConfigAgentBuilder(cfg)
```

### ConfigAgentBuilder Methods

```go
BuildAgent(agentID string) (*agent.Agent, error)
BuildAllAgents() (map[string]*agent.Agent, error)
AgentRegistry() *agent.AgentRegistry
ComponentManager() *component.ComponentManager
Config() *config.Config
```

---

## Runtime Builder

### `runtime.NewRuntimeBuilder() *RuntimeBuilder`

Creates a new runtime builder (from `pkg/runtime` package).

**Example:**
```go
builder := runtime.NewRuntimeBuilder()
```

### RuntimeBuilder Methods

```go
WithAgent(agent *agent.Agent) *RuntimeBuilder
WithAgents(agents map[string]*agent.Agent) *RuntimeBuilder
Start() (*Runtime, error)
```

---

## Helper Functions

### `boolPtr(b bool) *bool`

Returns a pointer to the bool value.

### `boolValue(b *bool, defaultValue bool) bool`

Returns the bool value or default if nil.

---

## Type Definitions

### `ReasoningConfig`

```go
type ReasoningConfig struct {
    Engine          string
    MaxIterations   int
    EnableStreaming *bool
    ShowTools       *bool
    ShowThinking    *bool
}
```

---

## Examples

See [Programmatic API Guide](../core-concepts/programmatic-api.md) for complete examples.

