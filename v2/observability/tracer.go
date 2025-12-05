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
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Tracer wraps the OpenTelemetry tracer with Hector-specific helpers.
type Tracer struct {
	provider       *sdktrace.TracerProvider
	tracer         trace.Tracer
	debugExporter  *DebugExporter
	capturePayload bool
	serviceName    string
}

// TracerOption configures the Tracer.
type TracerOption func(*Tracer)

// WithDebugExporter adds a debug exporter for web UI inspection.
func WithDebugExporter(exporter *DebugExporter) TracerOption {
	return func(t *Tracer) {
		t.debugExporter = exporter
	}
}

// WithCapturePayloads enables capturing full LLM request/response in spans.
func WithCapturePayloads(capture bool) TracerOption {
	return func(t *Tracer) {
		t.capturePayload = capture
	}
}

// NewTracer creates a new Tracer from configuration.
func NewTracer(ctx context.Context, cfg *TracingConfig, opts ...TracerOption) (*Tracer, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}

	cfg.SetDefaults()

	// Create exporter
	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String(AttrGenAISystem, "hector"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	sampler := sdktrace.TraceIDRatioBased(cfg.SamplingRate)

	// Create tracer provider
	providerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	}

	provider := sdktrace.NewTracerProvider(providerOpts...)

	// Set global tracer provider and propagator
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t := &Tracer{
		provider:    provider,
		tracer:      provider.Tracer(cfg.ServiceName),
		serviceName: cfg.ServiceName,
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// Register debug exporter if enabled
	if t.debugExporter != nil {
		provider.RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(t.debugExporter))
	}

	return t, nil
}

// createExporter creates the appropriate span exporter based on configuration.
func createExporter(ctx context.Context, cfg *TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "otlp":
		return createOTLPExporter(ctx, cfg)
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "jaeger", "zipkin":
		// For Jaeger and Zipkin, we still use OTLP as most modern collectors support it
		return createOTLPExporter(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported exporter: %s", cfg.Exporter)
	}
}

// createOTLPExporter creates an OTLP gRPC exporter.
func createOTLPExporter(ctx context.Context, cfg *TracingConfig) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.IsInsecure() {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// Start begins a new span with the given name.
func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t == nil || t.tracer == nil {
		return ctx, noopSpan()
	}
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartAgentRun begins a span for an agent invocation.
func (t *Tracer) StartAgentRun(ctx context.Context, agentName, agentType, sessionID, userID, invocationID string) (context.Context, trace.Span) {
	return t.Start(ctx, SpanAgentRun,
		trace.WithAttributes(
			attribute.String(AttrHectorAgentName, agentName),
			attribute.String(AttrHectorAgentType, agentType),
			attribute.String(AttrHectorSessionID, sessionID),
			attribute.String(AttrHectorUserID, userID),
			attribute.String(AttrHectorInvocationID, invocationID),
		),
	)
}

// StartLLMCall begins a span for an LLM API call.
func (t *Tracer) StartLLMCall(ctx context.Context, model string, maxTokens int, temperature, topP float64) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrGenAIOperationName, OpChat),
		attribute.String(AttrGenAIRequestModel, model),
	}

	if maxTokens > 0 {
		attrs = append(attrs, attribute.Int(AttrGenAIRequestMaxTokens, maxTokens))
	}
	if temperature > 0 {
		attrs = append(attrs, attribute.Float64(AttrGenAIRequestTemperature, temperature))
	}
	if topP > 0 {
		attrs = append(attrs, attribute.Float64(AttrGenAIRequestTopP, topP))
	}

	return t.Start(ctx, SpanLLMCall, trace.WithAttributes(attrs...))
}

// StartToolExecution begins a span for tool execution.
func (t *Tracer) StartToolExecution(ctx context.Context, toolName, toolDescription, callID string) (context.Context, trace.Span) {
	return t.Start(ctx, SpanToolExecution,
		trace.WithAttributes(
			attribute.String(AttrGenAIOperationName, OpToolCall),
			attribute.String(AttrGenAIToolName, toolName),
			attribute.String(AttrGenAIToolDescription, toolDescription),
			attribute.String(AttrGenAIToolCallID, callID),
		),
	)
}

// StartMemorySearch begins a span for memory search operations.
func (t *Tracer) StartMemorySearch(ctx context.Context, query string, limit int) (context.Context, trace.Span) {
	return t.Start(ctx, SpanMemorySearch,
		trace.WithAttributes(
			attribute.String("query", query),
			attribute.Int("limit", limit),
		),
	)
}

// StartRAGSearch begins a span for RAG search operations.
func (t *Tracer) StartRAGSearch(ctx context.Context, storeName, query string, topK int, hydeEnabled, rerankEnabled, multiQueryEnabled bool) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGSearch,
		trace.WithAttributes(
			attribute.String(AttrRAGStoreName, storeName),
			attribute.String(AttrRAGQuery, query),
			attribute.Int(AttrRAGTopK, topK),
			attribute.Bool(AttrRAGHyDEEnabled, hydeEnabled),
			attribute.Bool(AttrRAGRerankEnabled, rerankEnabled),
			attribute.Bool(AttrRAGMultiQueryEnabled, multiQueryEnabled),
		),
	)
}

// StartRAGIndex begins a span for RAG indexing operations.
func (t *Tracer) StartRAGIndex(ctx context.Context, storeName, sourceType string, documentCount int) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGIndex,
		trace.WithAttributes(
			attribute.String(AttrRAGStoreName, storeName),
			attribute.String(AttrRAGSourceType, sourceType),
			attribute.Int(AttrRAGDocumentCount, documentCount),
		),
	)
}

// StartRAGEmbed begins a span for embedding generation.
func (t *Tracer) StartRAGEmbed(ctx context.Context, model string, textLength int) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGEmbed,
		trace.WithAttributes(
			attribute.String(AttrGenAIOperationName, OpEmbeddings),
			attribute.String(AttrRAGEmbeddingModel, model),
			attribute.Int("text_length", textLength),
		),
	)
}

// StartRAGChunk begins a span for document chunking.
func (t *Tracer) StartRAGChunk(ctx context.Context, strategy string, documentSize int) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGChunk,
		trace.WithAttributes(
			attribute.String(AttrRAGChunkerStrategy, strategy),
			attribute.Int("document_size", documentSize),
		),
	)
}

// StartRAGRerank begins a span for result reranking.
func (t *Tracer) StartRAGRerank(ctx context.Context, inputCount int) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGRerank,
		trace.WithAttributes(
			attribute.Int("input_count", inputCount),
		),
	)
}

// StartRAGHyDE begins a span for HyDE hypothetical document generation.
func (t *Tracer) StartRAGHyDE(ctx context.Context, query string) (context.Context, trace.Span) {
	return t.Start(ctx, SpanRAGHyDE,
		trace.WithAttributes(
			attribute.String(AttrRAGQuery, query),
		),
	)
}

// AddRAGSearchResults adds search result count to a span.
func (t *Tracer) AddRAGSearchResults(span trace.Span, resultCount int) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.Int(AttrRAGResultCount, resultCount))
}

// AddRAGIndexStats adds indexing statistics to a span.
func (t *Tracer) AddRAGIndexStats(span trace.Span, chunkCount int) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.Int(AttrRAGChunkCount, chunkCount))
}

// AddLLMUsage adds token usage information to a span.
func (t *Tracer) AddLLMUsage(span trace.Span, inputTokens, outputTokens int) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.Int(AttrGenAIUsageInputTokens, inputTokens),
		attribute.Int(AttrGenAIUsageOutputTokens, outputTokens),
	)
}

// AddLLMFinishReason adds the finish reason to a span.
func (t *Tracer) AddLLMFinishReason(span trace.Span, reason string) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.String(AttrGenAIResponseFinishReason, reason))
}

// AddPayload adds serialized request/response to a span (if capture is enabled).
func (t *Tracer) AddPayload(span trace.Span, request, response string) {
	if span == nil || !t.capturePayload {
		return
	}
	if request != "" {
		span.SetAttributes(attribute.String(AttrHectorLLMRequest, request))
	}
	if response != "" {
		span.SetAttributes(attribute.String(AttrHectorLLMResponse, response))
	}
}

// AddToolPayload adds serialized tool args/response to a span (if capture is enabled).
func (t *Tracer) AddToolPayload(span trace.Span, args, response string) {
	if span == nil || !t.capturePayload {
		return
	}
	if args != "" {
		span.SetAttributes(attribute.String(AttrHectorToolArgs, args))
	}
	if response != "" {
		span.SetAttributes(attribute.String(AttrHectorToolResponse, response))
	}
}

// RecordError records an error on a span.
func (t *Tracer) RecordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetAttributes(
		attribute.String(AttrErrorType, fmt.Sprintf("%T", err)),
		attribute.String(AttrErrorMessage, err.Error()),
	)
}

// DebugExporter returns the debug exporter if configured.
func (t *Tracer) DebugExporter() *DebugExporter {
	if t == nil {
		return nil
	}
	return t.debugExporter
}

// Shutdown gracefully shuts down the tracer.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t == nil || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// noopSpan returns a no-op span that satisfies the trace.Span interface.
// We use OTel's built-in noop tracer which handles interface requirements properly.
func noopSpan() trace.Span {
	_, span := trace.NewNoopTracerProvider().Tracer("noop").Start(context.Background(), "noop")
	return span
}
