// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics provides Prometheus metrics collection for Hector.
type Metrics struct {
	config   *MetricsConfig
	registry *prometheus.Registry

	// Agent metrics
	agentCalls        *prometheus.CounterVec
	agentCallDuration *prometheus.HistogramVec
	agentErrors       *prometheus.CounterVec
	agentActiveRuns   *prometheus.GaugeVec

	// LLM metrics
	llmCalls        *prometheus.CounterVec
	llmCallDuration *prometheus.HistogramVec
	llmTokensInput  *prometheus.CounterVec
	llmTokensOutput *prometheus.CounterVec
	llmErrors       *prometheus.CounterVec

	// Tool metrics
	toolCalls        *prometheus.CounterVec
	toolCallDuration *prometheus.HistogramVec
	toolErrors       *prometheus.CounterVec

	// Memory/Index metrics
	memorySearches  *prometheus.CounterVec
	memorySearchDur *prometheus.HistogramVec
	memoryIndexed   *prometheus.CounterVec

	// Session metrics
	sessionsCreated    *prometheus.CounterVec
	sessionsActive     *prometheus.GaugeVec
	sessionEventsTotal *prometheus.CounterVec

	// HTTP metrics
	httpRequests     *prometheus.CounterVec
	httpDuration     *prometheus.HistogramVec
	httpRequestSize  *prometheus.HistogramVec
	httpResponseSize *prometheus.HistogramVec

	// RAG metrics
	ragDocsIndexed    *prometheus.CounterVec
	ragDocsSkipped    *prometheus.CounterVec
	ragDocsErrors     *prometheus.CounterVec
	ragIndexDuration  *prometheus.HistogramVec
	ragSearches       *prometheus.CounterVec
	ragSearchDuration *prometheus.HistogramVec
	ragSearchResults  *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance from configuration.
func NewMetrics(cfg *MetricsConfig) (*Metrics, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}

	cfg.SetDefaults()

	m := &Metrics{
		config:   cfg,
		registry: prometheus.NewRegistry(),
	}

	m.initAgentMetrics()
	m.initLLMMetrics()
	m.initToolMetrics()
	m.initMemoryMetrics()
	m.initSessionMetrics()
	m.initHTTPMetrics()
	m.initRAGMetrics()

	return m, nil
}

func (m *Metrics) initAgentMetrics() {
	m.agentCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "agent",
			Name:      "calls_total",
			Help:      "Total number of agent invocations",
		},
		[]string{"agent_name", "agent_type"},
	)

	m.agentCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "agent",
			Name:      "call_duration_seconds",
			Help:      "Agent invocation duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 15), // 10ms to 163s
		},
		[]string{"agent_name", "agent_type"},
	)

	m.agentErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "agent",
			Name:      "errors_total",
			Help:      "Total number of agent errors",
		},
		[]string{"agent_name", "agent_type", "error_type"},
	)

	m.agentActiveRuns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.config.Namespace,
			Subsystem: "agent",
			Name:      "active_runs",
			Help:      "Number of currently active agent runs",
		},
		[]string{"agent_name"},
	)

	m.registry.MustRegister(m.agentCalls, m.agentCallDuration, m.agentErrors, m.agentActiveRuns)
}

func (m *Metrics) initLLMMetrics() {
	m.llmCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "llm",
			Name:      "calls_total",
			Help:      "Total number of LLM API calls",
		},
		[]string{"model", "provider"},
	)

	m.llmCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "llm",
			Name:      "call_duration_seconds",
			Help:      "LLM API call duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 12), // 100ms to 204s
		},
		[]string{"model", "provider"},
	)

	m.llmTokensInput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "llm",
			Name:      "tokens_input_total",
			Help:      "Total number of input tokens consumed",
		},
		[]string{"model", "provider"},
	)

	m.llmTokensOutput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "llm",
			Name:      "tokens_output_total",
			Help:      "Total number of output tokens generated",
		},
		[]string{"model", "provider"},
	)

	m.llmErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "llm",
			Name:      "errors_total",
			Help:      "Total number of LLM API errors",
		},
		[]string{"model", "provider", "error_type"},
	)

	m.registry.MustRegister(m.llmCalls, m.llmCallDuration, m.llmTokensInput, m.llmTokensOutput, m.llmErrors)
}

func (m *Metrics) initToolMetrics() {
	m.toolCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "tool",
			Name:      "calls_total",
			Help:      "Total number of tool invocations",
		},
		[]string{"tool_name"},
	)

	m.toolCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "tool",
			Name:      "call_duration_seconds",
			Help:      "Tool execution duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{"tool_name"},
	)

	m.toolErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "tool",
			Name:      "errors_total",
			Help:      "Total number of tool errors",
		},
		[]string{"tool_name", "error_type"},
	)

	m.registry.MustRegister(m.toolCalls, m.toolCallDuration, m.toolErrors)
}

func (m *Metrics) initMemoryMetrics() {
	m.memorySearches = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "memory",
			Name:      "searches_total",
			Help:      "Total number of memory searches",
		},
		[]string{"index_type"},
	)

	m.memorySearchDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "memory",
			Name:      "search_duration_seconds",
			Help:      "Memory search duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 2s
		},
		[]string{"index_type"},
	)

	m.memoryIndexed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "memory",
			Name:      "indexed_total",
			Help:      "Total number of items indexed",
		},
		[]string{"index_type"},
	)

	m.registry.MustRegister(m.memorySearches, m.memorySearchDur, m.memoryIndexed)
}

func (m *Metrics) initSessionMetrics() {
	m.sessionsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "session",
			Name:      "created_total",
			Help:      "Total number of sessions created",
		},
		[]string{"app_name"},
	)

	m.sessionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.config.Namespace,
			Subsystem: "session",
			Name:      "active",
			Help:      "Number of currently active sessions",
		},
		[]string{"app_name"},
	)

	m.sessionEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "session",
			Name:      "events_total",
			Help:      "Total number of session events",
		},
		[]string{"app_name", "event_type"},
	)

	m.registry.MustRegister(m.sessionsCreated, m.sessionsActive, m.sessionEventsTotal)
}

func (m *Metrics) initHTTPMetrics() {
	m.httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "path"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "path"},
	)

	m.registry.MustRegister(m.httpRequests, m.httpDuration, m.httpRequestSize, m.httpResponseSize)
}

func (m *Metrics) initRAGMetrics() {
	m.ragDocsIndexed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "documents_indexed_total",
			Help:      "Total number of documents indexed",
		},
		[]string{"store_name"},
	)

	m.ragDocsSkipped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "documents_skipped_total",
			Help:      "Total number of documents skipped during indexing",
		},
		[]string{"store_name"},
	)

	m.ragDocsErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "documents_errors_total",
			Help:      "Total number of document indexing errors",
		},
		[]string{"store_name"},
	)

	m.ragIndexDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "index_duration_seconds",
			Help:      "Document indexing duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms to 20s
		},
		[]string{"store_name"},
	)

	m.ragSearches = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "searches_total",
			Help:      "Total number of RAG searches",
		},
		[]string{"store_name"},
	)

	m.ragSearchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "search_duration_seconds",
			Help:      "RAG search duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 2s
		},
		[]string{"store_name"},
	)

	m.ragSearchResults = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.config.Namespace,
			Subsystem: "rag",
			Name:      "search_results_count",
			Help:      "Number of results returned by RAG search",
			Buckets:   prometheus.LinearBuckets(0, 5, 11), // 0, 5, 10, ... 50
		},
		[]string{"store_name"},
	)

	m.registry.MustRegister(m.ragDocsIndexed, m.ragDocsSkipped, m.ragDocsErrors,
		m.ragIndexDuration, m.ragSearches, m.ragSearchDuration, m.ragSearchResults)
}

// =============================================================================
// Agent Metrics
// =============================================================================

// RecordAgentCall records an agent invocation.
func (m *Metrics) RecordAgentCall(agentName, agentType string, duration time.Duration) {
	if m == nil {
		return
	}
	m.agentCalls.WithLabelValues(agentName, agentType).Inc()
	m.agentCallDuration.WithLabelValues(agentName, agentType).Observe(duration.Seconds())
}

// RecordAgentError records an agent error.
func (m *Metrics) RecordAgentError(agentName, agentType, errorType string) {
	if m == nil {
		return
	}
	m.agentErrors.WithLabelValues(agentName, agentType, errorType).Inc()
}

// IncAgentActiveRuns increments the active runs counter.
func (m *Metrics) IncAgentActiveRuns(agentName string) {
	if m == nil {
		return
	}
	m.agentActiveRuns.WithLabelValues(agentName).Inc()
}

// DecAgentActiveRuns decrements the active runs counter.
func (m *Metrics) DecAgentActiveRuns(agentName string) {
	if m == nil {
		return
	}
	m.agentActiveRuns.WithLabelValues(agentName).Dec()
}

// =============================================================================
// LLM Metrics
// =============================================================================

// RecordLLMCall records an LLM API call.
func (m *Metrics) RecordLLMCall(model, provider string, duration time.Duration) {
	if m == nil {
		return
	}
	m.llmCalls.WithLabelValues(model, provider).Inc()
	m.llmCallDuration.WithLabelValues(model, provider).Observe(duration.Seconds())
}

// RecordLLMTokens records token usage.
func (m *Metrics) RecordLLMTokens(model, provider string, inputTokens, outputTokens int) {
	if m == nil {
		return
	}
	m.llmTokensInput.WithLabelValues(model, provider).Add(float64(inputTokens))
	m.llmTokensOutput.WithLabelValues(model, provider).Add(float64(outputTokens))
}

// RecordLLMError records an LLM error.
func (m *Metrics) RecordLLMError(model, provider, errorType string) {
	if m == nil {
		return
	}
	m.llmErrors.WithLabelValues(model, provider, errorType).Inc()
}

// =============================================================================
// Tool Metrics
// =============================================================================

// RecordToolCall records a tool invocation.
func (m *Metrics) RecordToolCall(toolName string, duration time.Duration) {
	if m == nil {
		return
	}
	m.toolCalls.WithLabelValues(toolName).Inc()
	m.toolCallDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}

// RecordToolError records a tool error.
func (m *Metrics) RecordToolError(toolName, errorType string) {
	if m == nil {
		return
	}
	m.toolErrors.WithLabelValues(toolName, errorType).Inc()
}

// =============================================================================
// Memory Metrics
// =============================================================================

// RecordMemorySearch records a memory search operation.
func (m *Metrics) RecordMemorySearch(indexType string, duration time.Duration) {
	if m == nil {
		return
	}
	m.memorySearches.WithLabelValues(indexType).Inc()
	m.memorySearchDur.WithLabelValues(indexType).Observe(duration.Seconds())
}

// RecordMemoryIndexed records items being indexed.
func (m *Metrics) RecordMemoryIndexed(indexType string, count int) {
	if m == nil {
		return
	}
	m.memoryIndexed.WithLabelValues(indexType).Add(float64(count))
}

// =============================================================================
// Session Metrics
// =============================================================================

// RecordSessionCreated records a session creation.
func (m *Metrics) RecordSessionCreated(appName string) {
	if m == nil {
		return
	}
	m.sessionsCreated.WithLabelValues(appName).Inc()
}

// SetSessionsActive sets the number of active sessions.
func (m *Metrics) SetSessionsActive(appName string, count int) {
	if m == nil {
		return
	}
	m.sessionsActive.WithLabelValues(appName).Set(float64(count))
}

// RecordSessionEvent records a session event.
func (m *Metrics) RecordSessionEvent(appName, eventType string) {
	if m == nil {
		return
	}
	m.sessionEventsTotal.WithLabelValues(appName, eventType).Inc()
}

// =============================================================================
// HTTP Metrics
// =============================================================================

// RecordHTTPRequest records an HTTP request.
func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, reqSize, respSize int64) {
	if m == nil {
		return
	}
	status := statusCodeLabel(statusCode)
	m.httpRequests.WithLabelValues(method, path, status).Inc()
	m.httpDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	if reqSize > 0 {
		m.httpRequestSize.WithLabelValues(method, path).Observe(float64(reqSize))
	}
	if respSize > 0 {
		m.httpResponseSize.WithLabelValues(method, path).Observe(float64(respSize))
	}
}

// statusCodeLabel converts a status code to a label string.
func statusCodeLabel(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// =============================================================================
// RAG Metrics
// =============================================================================

// RecordRAGDocIndexed records a document being indexed.
func (m *Metrics) RecordRAGDocIndexed(storeName string, duration time.Duration) {
	if m == nil {
		return
	}
	m.ragDocsIndexed.WithLabelValues(storeName).Inc()
	m.ragIndexDuration.WithLabelValues(storeName).Observe(duration.Seconds())
}

// RecordRAGDocSkipped records a document being skipped.
func (m *Metrics) RecordRAGDocSkipped(storeName string) {
	if m == nil {
		return
	}
	m.ragDocsSkipped.WithLabelValues(storeName).Inc()
}

// RecordRAGDocError records a document indexing error.
func (m *Metrics) RecordRAGDocError(storeName string) {
	if m == nil {
		return
	}
	m.ragDocsErrors.WithLabelValues(storeName).Inc()
}

// RecordRAGSearch records a RAG search operation.
func (m *Metrics) RecordRAGSearch(storeName string, duration time.Duration, resultCount int) {
	if m == nil {
		return
	}
	m.ragSearches.WithLabelValues(storeName).Inc()
	m.ragSearchDuration.WithLabelValues(storeName).Observe(duration.Seconds())
	m.ragSearchResults.WithLabelValues(storeName).Observe(float64(resultCount))
}

// =============================================================================
// HTTP Handler
// =============================================================================

// Handler returns an HTTP handler for the Prometheus metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Registry returns the Prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}
	return m.registry
}
