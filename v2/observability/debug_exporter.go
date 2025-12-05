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

package observability

import (
	"context"
	"sync"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// DebugExporter is a custom SpanExporter that stores span data in memory
// for web UI inspection and debugging.
//
// Inspired by adk-go's APIServerSpanExporter, this exporter captures
// relevant span attributes (LLM calls, tool executions, agent runs)
// keyed by event ID for easy lookup.
//
// Thread-safe for concurrent reads and writes.
type DebugExporter struct {
	mu      sync.RWMutex
	spans   map[string]*DebugSpan // Keyed by span ID
	byEvent map[string]*DebugSpan // Keyed by hector.event_id for quick lookup
	maxSize int                   // Maximum number of spans to retain
}

// DebugSpan contains captured span information for debugging.
type DebugSpan struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Name         string            `json:"name"`
	StartTime    int64             `json:"start_time_unix_nano"`
	EndTime      int64             `json:"end_time_unix_nano"`
	DurationMs   float64           `json:"duration_ms"`
	Attributes   map[string]string `json:"attributes"`
	Events       []SpanEvent       `json:"events,omitempty"`
	Status       string            `json:"status"`
	StatusMsg    string            `json:"status_message,omitempty"`
}

// SpanEvent represents an event recorded on a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	TimeUnix   int64             `json:"time_unix_nano"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// NewDebugExporter creates a new DebugExporter.
func NewDebugExporter() *DebugExporter {
	return &DebugExporter{
		spans:   make(map[string]*DebugSpan),
		byEvent: make(map[string]*DebugSpan),
		maxSize: 1000, // Default: retain last 1000 spans
	}
}

// WithMaxSize sets the maximum number of spans to retain.
func (e *DebugExporter) WithMaxSize(size int) *DebugExporter {
	e.maxSize = size
	return e
}

// ExportSpans implements sdktrace.SpanExporter.
// It captures span data for relevant spans (LLM calls, tool executions, etc.).
func (e *DebugExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, span := range spans {
		// Only capture spans we're interested in
		if !e.shouldCapture(span.Name()) {
			continue
		}

		debugSpan := e.convertSpan(span)
		e.spans[debugSpan.SpanID] = debugSpan

		// Also index by event ID if present
		if eventID, ok := debugSpan.Attributes[AttrHectorEventID]; ok && eventID != "" {
			e.byEvent[eventID] = debugSpan
		}

		// Evict old spans if we're over the limit
		e.evictOldest()
	}

	return nil
}

// shouldCapture returns true if we should capture this span type.
func (e *DebugExporter) shouldCapture(name string) bool {
	switch name {
	case SpanAgentRun, SpanLLMCall, SpanToolExecution, SpanMemorySearch:
		return true
	default:
		return false
	}
}

// convertSpan converts an OpenTelemetry span to our DebugSpan format.
func (e *DebugExporter) convertSpan(span sdktrace.ReadOnlySpan) *DebugSpan {
	startTime := span.StartTime().UnixNano()
	endTime := span.EndTime().UnixNano()
	durationMs := float64(endTime-startTime) / 1e6

	ds := &DebugSpan{
		TraceID:    span.SpanContext().TraceID().String(),
		SpanID:     span.SpanContext().SpanID().String(),
		Name:       span.Name(),
		StartTime:  startTime,
		EndTime:    endTime,
		DurationMs: durationMs,
		Attributes: make(map[string]string),
		Status:     span.Status().Code.String(),
		StatusMsg:  span.Status().Description,
	}

	// Convert parent span ID if present
	if span.Parent().HasSpanID() {
		ds.ParentSpanID = span.Parent().SpanID().String()
	}

	// Convert attributes
	for _, attr := range span.Attributes() {
		key := string(attr.Key)
		ds.Attributes[key] = attr.Value.AsString()
	}

	// Convert events
	for _, event := range span.Events() {
		se := SpanEvent{
			Name:       event.Name,
			TimeUnix:   event.Time.UnixNano(),
			Attributes: make(map[string]string),
		}
		for _, attr := range event.Attributes {
			se.Attributes[string(attr.Key)] = attr.Value.AsString()
		}
		ds.Events = append(ds.Events, se)
	}

	return ds
}

// evictOldest removes the oldest spans if we're over the limit.
// Caller must hold the write lock.
func (e *DebugExporter) evictOldest() {
	if len(e.spans) <= e.maxSize {
		return
	}

	// Simple eviction: just remove excess (not strictly oldest, but efficient)
	excess := len(e.spans) - e.maxSize
	removed := 0
	for id := range e.spans {
		if removed >= excess {
			break
		}
		delete(e.spans, id)
		removed++
	}
}

// Shutdown implements sdktrace.SpanExporter.
func (e *DebugExporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = make(map[string]*DebugSpan)
	e.byEvent = make(map[string]*DebugSpan)
	return nil
}

// =============================================================================
// Query Methods
// =============================================================================

// GetSpan returns a span by its span ID.
func (e *DebugExporter) GetSpan(spanID string) *DebugSpan {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.spans[spanID]
}

// GetByEventID returns a span by its hector.event_id attribute.
func (e *DebugExporter) GetByEventID(eventID string) *DebugSpan {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.byEvent[eventID]
}

// GetAllSpans returns all captured spans.
func (e *DebugExporter) GetAllSpans() []*DebugSpan {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*DebugSpan, 0, len(e.spans))
	for _, span := range e.spans {
		result = append(result, span)
	}
	return result
}

// GetSpansByName returns all spans with the given name.
func (e *DebugExporter) GetSpansByName(name string) []*DebugSpan {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*DebugSpan
	for _, span := range e.spans {
		if span.Name == name {
			result = append(result, span)
		}
	}
	return result
}

// GetSpansByTrace returns all spans for a given trace ID.
func (e *DebugExporter) GetSpansByTrace(traceID string) []*DebugSpan {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*DebugSpan
	for _, span := range e.spans {
		if span.TraceID == traceID {
			result = append(result, span)
		}
	}
	return result
}

// Clear removes all captured spans.
func (e *DebugExporter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = make(map[string]*DebugSpan)
	e.byEvent = make(map[string]*DebugSpan)
}

// Count returns the number of captured spans.
func (e *DebugExporter) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.spans)
}

// Ensure DebugExporter implements SpanExporter.
var _ sdktrace.SpanExporter = (*DebugExporter)(nil)
