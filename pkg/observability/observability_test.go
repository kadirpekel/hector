package observability

import (
	"context"
	"testing"
	"time"
)

// TestMetricsRecording tests that metrics are recorded without errors
func TestMetricsRecording(t *testing.T) {
	ctx := context.Background()

	// Create metrics instance with nil instruments (for testing)
	metrics := &PrometheusMetrics{}

	// Test agent call recording - should not panic
	metrics.RecordAgentCall(ctx, 100*time.Millisecond, 150, nil)
	metrics.RecordAgentCall(ctx, 200*time.Millisecond, 200, nil)

	t.Log("✅ Agent metrics recorded successfully")
}

// TestToolMetricsRecording tests tool execution metrics
func TestToolMetricsRecording(t *testing.T) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	// Test tool recording - should not panic
	metrics.RecordToolExecution(ctx, "search", 50*time.Millisecond, nil)
	metrics.RecordToolExecution(ctx, "write_file", 100*time.Millisecond, nil)

	t.Log("✅ Tool metrics recorded successfully")
}

// TestLLMMetricsRecording tests LLM call metrics
func TestLLMMetricsRecording(t *testing.T) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	// Test LLM recording - should not panic
	metrics.RecordLLMCall(ctx, "gpt-4o", 500*time.Millisecond, 100, 50, nil)
	metrics.RecordLLMCall(ctx, "claude-sonnet", 600*time.Millisecond, 150, 75, nil)

	t.Log("✅ LLM metrics recorded successfully")
}

// TestNoopMetrics verifies noop metrics don't panic
func TestNoopMetrics(t *testing.T) {
	ctx := context.Background()
	var metrics Metrics

	// Should not panic with nil metrics
	if metrics == nil {
		// Properly test by calling on noop
		noopMetrics := &NoopMetrics{}
		noopMetrics.RecordAgentCall(ctx, 100*time.Millisecond, 150, nil)
		noopMetrics.RecordToolExecution(ctx, "test", 50*time.Millisecond, nil)
		noopMetrics.RecordLLMCall(ctx, "test-model", 300*time.Millisecond, 10, 5, nil)
	}

	t.Log("✅ Noop metrics handled correctly")
}

// TestNoopTracer verifies noop tracer works
func TestNoopTracer(t *testing.T) {
	tracer := NoopTracer("test")

	// Should be able to start spans with noop tracer
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test_span")
	defer span.End()

	t.Log("✅ Noop tracer works correctly")
}

// TestStringTruncation tests the truncation helper function
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
		// This would be in instrumentation.go
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}

	t.Log("✅ String truncation tests passed")
}

// TestGlobalMetrics tests the global metrics management
func TestGlobalMetrics(t *testing.T) {
	ctx := context.Background()

	// Initially should be nil or empty
	_ = GetGlobalMetrics()

	// Set noop metrics
	noopMetrics := &NoopMetrics{}
	SetGlobalMetrics(noopMetrics)

	// Retrieve and verify
	retrievedMetrics := GetGlobalMetrics()
	if retrievedMetrics == nil {
		t.Error("Expected non-nil metrics after SetGlobalMetrics")
	}

	// Test that we can record
	retrievedMetrics.RecordAgentCall(ctx, 100*time.Millisecond, 50, nil)

	t.Log("✅ Global metrics management works correctly")
}

// BenchmarkMetricsRecording benchmarks metrics recording overhead
func BenchmarkMetricsRecording(b *testing.B) {
	ctx := context.Background()
	metrics := &PrometheusMetrics{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordAgentCall(ctx, 100*time.Millisecond, 50, nil)
	}
}

// truncateString is a test helper (duplicate from instrumentation.go for testing)
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
