// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package observability provides OpenTelemetry tracing and Prometheus metrics.
//
// This package combines the production-tested foundation from legacy Hector
// with GenAI semantic conventions from adk-go for ecosystem compatibility.
//
// # Architecture
//
// The observability system has three main components:
//
//  1. Tracing: OpenTelemetry spans with OTLP export
//  2. Metrics: Prometheus counters and histograms
//  3. Debug: In-memory span capture for web UI inspection
//
// # Configuration
//
// Configure observability in your hector.yaml:
//
//	server:
//	  observability:
//	    tracing:
//	      enabled: true
//	      exporter: otlp
//	      endpoint: localhost:4317
//	      sampling_rate: 1.0
//	      service_name: my-agent
//	    metrics:
//	      enabled: true
//	      endpoint: /metrics
package observability

// =============================================================================
// Service Attributes (OpenTelemetry Semantic Conventions)
// =============================================================================

const (
	// AttrServiceName is the logical name of the service.
	AttrServiceName = "service.name"

	// AttrServiceVersion is the version of the service.
	AttrServiceVersion = "service.version"

	// AttrServiceInstance is the instance ID of the service.
	AttrServiceInstance = "service.instance.id"
)

// =============================================================================
// GenAI Semantic Conventions (OpenTelemetry GenAI SIG / ADK-Go aligned)
// =============================================================================

const (
	// AttrGenAISystem identifies the GenAI system (e.g., "hector", "openai").
	AttrGenAISystem = "gen_ai.system"

	// AttrGenAIOperationName is the operation being performed.
	// Values: "chat", "text_completion", "embeddings"
	AttrGenAIOperationName = "gen_ai.operation.name"

	// AttrGenAIRequestModel is the name of the model being used.
	AttrGenAIRequestModel = "gen_ai.request.model"

	// AttrGenAIRequestTemperature is the temperature parameter.
	AttrGenAIRequestTemperature = "gen_ai.request.temperature"

	// AttrGenAIRequestTopP is the top_p parameter.
	AttrGenAIRequestTopP = "gen_ai.request.top_p"

	// AttrGenAIRequestMaxTokens is the maximum tokens requested.
	AttrGenAIRequestMaxTokens = "gen_ai.request.max_tokens"

	// AttrGenAIResponseFinishReason is why generation stopped.
	// Values: "stop", "length", "tool_calls", "content_filter"
	AttrGenAIResponseFinishReason = "gen_ai.response.finish_reason"

	// AttrGenAIUsageInputTokens is the number of input tokens.
	AttrGenAIUsageInputTokens = "gen_ai.usage.input_tokens"

	// AttrGenAIUsageOutputTokens is the number of output tokens.
	AttrGenAIUsageOutputTokens = "gen_ai.usage.output_tokens"

	// AttrGenAIToolName is the name of the tool being called.
	AttrGenAIToolName = "gen_ai.tool.name"

	// AttrGenAIToolDescription is the description of the tool.
	AttrGenAIToolDescription = "gen_ai.tool.description"

	// AttrGenAIToolCallID is the unique ID of the tool call.
	AttrGenAIToolCallID = "gen_ai.tool.call.id"
)

// =============================================================================
// Hector-Specific Attributes
// =============================================================================

const (
	// AttrHectorAgentName is the name of the agent.
	AttrHectorAgentName = "hector.agent.name"

	// AttrHectorAgentType is the type of agent (llm, workflow, remote).
	AttrHectorAgentType = "hector.agent.type"

	// AttrHectorInvocationID is the unique ID for this agent invocation.
	AttrHectorInvocationID = "hector.invocation_id"

	// AttrHectorSessionID is the session ID.
	AttrHectorSessionID = "hector.session_id"

	// AttrHectorUserID is the user ID.
	AttrHectorUserID = "hector.user_id"

	// AttrHectorEventID is the event ID within a session.
	AttrHectorEventID = "hector.event_id"

	// AttrHectorLLMRequest is the serialized LLM request (optional, for debugging).
	AttrHectorLLMRequest = "hector.llm.request"

	// AttrHectorLLMResponse is the serialized LLM response (optional, for debugging).
	AttrHectorLLMResponse = "hector.llm.response"

	// AttrHectorToolArgs is the serialized tool arguments (optional, for debugging).
	AttrHectorToolArgs = "hector.tool.args"

	// AttrHectorToolResponse is the serialized tool response (optional, for debugging).
	AttrHectorToolResponse = "hector.tool.response"

	// AttrHectorThinkingBlocks is the number of thinking blocks in response.
	AttrHectorThinkingBlocks = "hector.llm.thinking.blocks"

	// AttrHectorThinkingLength is the total length of thinking content (chars).
	AttrHectorThinkingLength = "hector.llm.thinking.length"
)

// =============================================================================
// HTTP Attributes
// =============================================================================

const (
	// AttrHTTPMethod is the HTTP method.
	AttrHTTPMethod = "http.method"

	// AttrHTTPPath is the HTTP path (route pattern, not raw path).
	AttrHTTPPath = "http.route"

	// AttrHTTPStatusCode is the HTTP response status code.
	AttrHTTPStatusCode = "http.status_code"

	// AttrHTTPRequestSize is the request body size in bytes.
	AttrHTTPRequestSize = "http.request.body.size"

	// AttrHTTPResponseSize is the response body size in bytes.
	AttrHTTPResponseSize = "http.response.body.size"
)

// =============================================================================
// Error Attributes
// =============================================================================

const (
	// AttrErrorType is the type of error that occurred.
	AttrErrorType = "error.type"

	// AttrErrorMessage is the error message.
	AttrErrorMessage = "error.message"
)

// =============================================================================
// RAG/Document Store Attributes
// =============================================================================

const (
	// AttrRAGStoreName is the name of the document store.
	AttrRAGStoreName = "hector.rag.store.name"

	// AttrRAGQuery is the search query.
	AttrRAGQuery = "hector.rag.query"

	// AttrRAGResultCount is the number of search results.
	AttrRAGResultCount = "hector.rag.result_count"

	// AttrRAGTopK is the requested number of results.
	AttrRAGTopK = "hector.rag.top_k"

	// AttrRAGDocumentCount is the number of documents indexed.
	AttrRAGDocumentCount = "hector.rag.document_count"

	// AttrRAGChunkCount is the number of chunks indexed.
	AttrRAGChunkCount = "hector.rag.chunk_count"

	// AttrRAGSourceType is the data source type (directory, sql, api).
	AttrRAGSourceType = "hector.rag.source_type"

	// AttrRAGChunkerStrategy is the chunking strategy used.
	AttrRAGChunkerStrategy = "hector.rag.chunker_strategy"

	// AttrRAGHyDEEnabled indicates if HyDE was used.
	AttrRAGHyDEEnabled = "hector.rag.hyde_enabled"

	// AttrRAGRerankEnabled indicates if reranking was used.
	AttrRAGRerankEnabled = "hector.rag.rerank_enabled"

	// AttrRAGMultiQueryEnabled indicates if multi-query was used.
	AttrRAGMultiQueryEnabled = "hector.rag.multiquery_enabled"

	// AttrRAGEmbeddingModel is the embedding model used.
	AttrRAGEmbeddingModel = "hector.rag.embedding_model"

	// AttrRAGVectorProvider is the vector database provider.
	AttrRAGVectorProvider = "hector.rag.vector_provider"
)

// =============================================================================
// Span Names
// =============================================================================

const (
	// SpanAgentRun is the top-level span for an agent invocation.
	SpanAgentRun = "hector.agent.run"

	// SpanLLMCall is a span for an LLM API call.
	SpanLLMCall = "hector.llm.call"

	// SpanToolExecution is a span for tool execution.
	SpanToolExecution = "hector.tool.execute"

	// SpanMemorySearch is a span for memory/index search.
	SpanMemorySearch = "hector.memory.search"

	// SpanSessionLoad is a span for loading session data.
	SpanSessionLoad = "hector.session.load"

	// SpanHTTPRequest is a span for HTTP request handling.
	SpanHTTPRequest = "hector.http.request"

	// SpanRAGSearch is a span for RAG search operations.
	SpanRAGSearch = "hector.rag.search"

	// SpanRAGIndex is a span for RAG indexing operations.
	SpanRAGIndex = "hector.rag.index"

	// SpanRAGEmbed is a span for embedding generation.
	SpanRAGEmbed = "hector.rag.embed"

	// SpanRAGChunk is a span for document chunking.
	SpanRAGChunk = "hector.rag.chunk"

	// SpanRAGRerank is a span for result reranking.
	SpanRAGRerank = "hector.rag.rerank"

	// SpanRAGHyDE is a span for HyDE hypothetical document generation.
	SpanRAGHyDE = "hector.rag.hyde"
)

// =============================================================================
// Default Values
// =============================================================================

const (
	// DefaultServiceName is the default service name for tracing.
	DefaultServiceName = "hector"

	// DefaultSamplingRate is the default trace sampling rate.
	DefaultSamplingRate = 1.0

	// DefaultOTLPEndpoint is the default OTLP endpoint.
	DefaultOTLPEndpoint = "localhost:4317"

	// DefaultMetricsPath is the default Prometheus metrics endpoint.
	DefaultMetricsPath = "/metrics"
)

// =============================================================================
// GenAI Operation Names (for AttrGenAIOperationName)
// =============================================================================

const (
	// OpChat is a chat completion operation.
	OpChat = "chat"

	// OpTextCompletion is a text completion operation.
	OpTextCompletion = "text_completion"

	// OpEmbeddings is an embeddings generation operation.
	OpEmbeddings = "embeddings"

	// OpToolCall is a tool execution operation.
	OpToolCall = "execute_tool"
)
