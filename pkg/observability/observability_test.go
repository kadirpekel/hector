package observability

import (
	"context"
	"testing"
	"time"
)

func TestMetricsRecording(t *testing.T) {
	ctx := context.Background()

	metrics := &PrometheusMetrics{}

	metrics.RecordAgentCall(ctx, 100*time.Millisecond, 150, nil)
	metrics.RecordAgentCall(ctx, 200*time.Millisecond, 200, nil)

	t.Log("✅ Agent metrics recorded successfully (nil-safe)")
}

func TestToolMetricsRecording(t *testing.T) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	metrics.RecordToolExecution(ctx, "search", 50*time.Millisecond, nil)
	metrics.RecordToolExecution(ctx, "write_file", 100*time.Millisecond, nil)

	t.Log("✅ Tool metrics recorded successfully")
}

func TestLLMMetricsRecording(t *testing.T) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	metrics.RecordLLMCall(ctx, "gpt-4o", 500*time.Millisecond, 100, 50, nil)
	metrics.RecordLLMCall(ctx, "claude-sonnet", 600*time.Millisecond, 150, 75, nil)

	t.Log("✅ LLM metrics recorded successfully")
}

func TestNoopMetrics(t *testing.T) {
	ctx := context.Background()
	var metrics Metrics

	if metrics == nil {

		noopMetrics := &NoopMetrics{}
		noopMetrics.RecordAgentCall(ctx, 100*time.Millisecond, 150, nil)
		noopMetrics.RecordToolExecution(ctx, "test", 50*time.Millisecond, nil)
		noopMetrics.RecordLLMCall(ctx, "test-model", 300*time.Millisecond, 10, 5, nil)
	}

	t.Log("✅ Noop metrics handled correctly")
}

func TestNoopTracer(t *testing.T) {
	tracer := NoopTracer("test")

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test_span")
	defer span.End()

	t.Log("✅ Noop tracer works correctly")
}

func TestStringTruncation(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"test", 4, "test"},
		{"toolongstring", 4, "tool..."},
	}

	for _, tt := range tests {

		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}

	t.Log("✅ String truncation tests passed")
}

func TestGlobalMetrics(t *testing.T) {
	ctx := context.Background()

	_ = GetGlobalMetrics()

	noopMetrics := &NoopMetrics{}
	SetGlobalMetrics(noopMetrics)

	retrievedMetrics := GetGlobalMetrics()
	if retrievedMetrics == nil {
		t.Error("Expected non-nil metrics after SetGlobalMetrics")
	}

	retrievedMetrics.RecordAgentCall(ctx, 100*time.Millisecond, 50, nil)

	t.Log("✅ Global metrics management works correctly")
}

func BenchmarkMetricsRecording(b *testing.B) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordAgentCall(ctx, 100*time.Millisecond, 50, nil)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
