package observability

const (
	// Semantic convention attribute names
	AttrServiceName     = "service.name"
	AttrServiceVersion  = "service.version"
	AttrAgentName       = "agent.name"
	AttrAgentLLM        = "agent.llm"
	AttrToolName        = "tool.name"
	AttrLLMModel        = "llm.model"
	AttrLLMTokensInput  = "llm.tokens.input"
	AttrLLMTokensOutput = "llm.tokens.output"
	AttrErrorType       = "error.type"
	AttrStatusCode      = "http.status_code"

	// Trace span names
	SpanAgentCall     = "agent.call"
	SpanLLMRequest    = "agent.llm_request"
	SpanToolExecution = "agent.tool_execution"
	SpanMemoryLookup  = "agent.memory_lookup"

	// Default service name
	DefaultServiceName = "hector"
)
