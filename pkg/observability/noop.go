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
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// =============================================================================
// No-op Manager
// =============================================================================

// NoopManager returns a no-operation Manager that does nothing.
// Use this when observability is completely disabled.
func NoopManager() *Manager {
	return &Manager{}
}

// =============================================================================
// No-op Tracer
// =============================================================================

// NoopTracer returns a no-operation Tracer.
type NoopTracer struct{}

// Start returns a no-op span.
func (NoopTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, noopSpan()
}

// StartAgentRun returns a no-op span.
func (NoopTracer) StartAgentRun(ctx context.Context, _, _, _, _, _ string) (context.Context, trace.Span) {
	return ctx, noopSpan()
}

// StartLLMCall returns a no-op span.
func (NoopTracer) StartLLMCall(ctx context.Context, _ string, _ int, _, _ float64) (context.Context, trace.Span) {
	return ctx, noopSpan()
}

// StartToolExecution returns a no-op span.
func (NoopTracer) StartToolExecution(ctx context.Context, _, _, _ string) (context.Context, trace.Span) {
	return ctx, noopSpan()
}

// StartMemorySearch returns a no-op span.
func (NoopTracer) StartMemorySearch(ctx context.Context, _ string, _ int) (context.Context, trace.Span) {
	return ctx, noopSpan()
}

// AddLLMUsage is a no-op.
func (NoopTracer) AddLLMUsage(_ trace.Span, _, _ int) {}

// AddLLMFinishReason is a no-op.
func (NoopTracer) AddLLMFinishReason(_ trace.Span, _ string) {}

// AddPayload is a no-op.
func (NoopTracer) AddPayload(_ trace.Span, _, _ string) {}

// AddToolPayload is a no-op.
func (NoopTracer) AddToolPayload(_ trace.Span, _, _ string) {}

// RecordError is a no-op.
func (NoopTracer) RecordError(_ trace.Span, _ error) {}

// DebugExporter returns nil.
func (NoopTracer) DebugExporter() *DebugExporter { return nil }

// Shutdown is a no-op.
func (NoopTracer) Shutdown(_ context.Context) error { return nil }

// =============================================================================
// No-op Metrics
// =============================================================================

// NoopMetrics is a metrics implementation that does nothing.
type NoopMetrics struct{}

// Agent metrics - no-op
func (NoopMetrics) RecordAgentCall(_, _ string, _ time.Duration) {}
func (NoopMetrics) RecordAgentError(_, _, _ string)              {}
func (NoopMetrics) IncAgentActiveRuns(_ string)                  {}
func (NoopMetrics) DecAgentActiveRuns(_ string)                  {}

// LLM metrics - no-op
func (NoopMetrics) RecordLLMCall(_, _ string, _ time.Duration) {}
func (NoopMetrics) RecordLLMTokens(_, _ string, _, _ int)      {}
func (NoopMetrics) RecordLLMError(_, _, _ string)              {}

// Tool metrics - no-op
func (NoopMetrics) RecordToolCall(_ string, _ time.Duration) {}
func (NoopMetrics) RecordToolError(_, _ string)              {}

// Memory metrics - no-op
func (NoopMetrics) RecordMemorySearch(_ string, _ time.Duration) {}
func (NoopMetrics) RecordMemoryIndexed(_ string, _ int)          {}

// Session metrics - no-op
func (NoopMetrics) RecordSessionCreated(_ string)     {}
func (NoopMetrics) SetSessionsActive(_ string, _ int) {}
func (NoopMetrics) RecordSessionEvent(_, _ string)    {}

// HTTP metrics - no-op
func (NoopMetrics) RecordHTTPRequest(_, _ string, _ int, _ time.Duration, _, _ int64) {}

// RAG metrics - no-op
func (NoopMetrics) RecordRAGDocIndexed(_ string, _ time.Duration)    {}
func (NoopMetrics) RecordRAGDocSkipped(_ string)                     {}
func (NoopMetrics) RecordRAGDocError(_ string)                       {}
func (NoopMetrics) RecordRAGSearch(_ string, _ time.Duration, _ int) {}

// Handler returns a handler that returns 503 Service Unavailable.
func (NoopMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("metrics not enabled"))
	})
}

// =============================================================================
// Recorder Interface
// =============================================================================

// Recorder defines the interface for recording metrics.
// This allows for dependency injection and easier testing.
type Recorder interface {
	// Agent metrics
	RecordAgentCall(agentName, agentType string, duration time.Duration)
	RecordAgentError(agentName, agentType, errorType string)
	IncAgentActiveRuns(agentName string)
	DecAgentActiveRuns(agentName string)

	// LLM metrics
	RecordLLMCall(model, provider string, duration time.Duration)
	RecordLLMTokens(model, provider string, inputTokens, outputTokens int)
	RecordLLMError(model, provider, errorType string)

	// Tool metrics
	RecordToolCall(toolName string, duration time.Duration)
	RecordToolError(toolName, errorType string)

	// Memory metrics
	RecordMemorySearch(indexType string, duration time.Duration)
	RecordMemoryIndexed(indexType string, count int)

	// Session metrics
	RecordSessionCreated(appName string)
	SetSessionsActive(appName string, count int)
	RecordSessionEvent(appName, eventType string)

	// HTTP metrics
	RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, reqSize, respSize int64)

	// RAG metrics
	RecordRAGDocIndexed(storeName string, duration time.Duration)
	RecordRAGDocSkipped(storeName string)
	RecordRAGDocError(storeName string)
	RecordRAGSearch(storeName string, duration time.Duration, resultCount int)
}

// Ensure implementations satisfy the interface.
var (
	_ Recorder = (*Metrics)(nil)
	_ Recorder = NoopMetrics{}
)
