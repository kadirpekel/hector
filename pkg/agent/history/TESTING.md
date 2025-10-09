# History Strategy Testing Foundation

Comprehensive testing suite for validating history management strategies.

---

## Overview

This testing foundation provides **deterministic, analytical tests** to verify that each history strategy works correctly. All tests are designed to be reproducible and verify exact behavior.

---

## Test Structure

### 1. Unit Tests

**Buffer Window Strategy** (`buffer_window_unit_test.go`)
- ✅ Basic operations (create, add, get, clear)
- ✅ Window size enforcement (exact size)
- ✅ FIFO behavior verification
- ✅ Session isolation
- ✅ Message ordering
- ✅ Role preservation
- ✅ Edge cases (window size = 1, empty sessions)

**Summary Buffer Strategy** (`summary_buffer_unit_test.go`)
- ✅ Basic operations (create with defaults/custom values)
- ✅ Token counting accuracy
- ✅ Threshold calculation (80%, 75%, 90%)
- ✅ Summarization triggering
- ✅ Minimum message threshold (< 10 messages)
- ✅ Session isolation
- ✅ Summary content verification
- ✅ Hierarchical summarization (summary of summaries)
- ✅ Error handling

### 2. Integration Tests (`integration_test.go`)

- ✅ Factory creates correct strategies
- ✅ Full workflow tests (end-to-end)
- ✅ Strategy comparison
- ✅ Session lifecycle (create → add → get → clear → restart)
- ✅ Default values application
- ✅ Concurrent sessions
- ✅ Error handling

### 3. Analytical Test Suite (`test_suite.go`)

Comprehensive test suite that validates all critical behaviors:
- ✅ Window size enforcement
- ✅ FIFO ordering
- ✅ Token budget enforcement
- ✅ Threshold triggering
- ✅ Summarization flow
- ✅ Session isolation
- ✅ Strategy factory
- ✅ Default values

---

## Running Tests

### Run All Tests

```bash
cd /Users/kadirpekel/hector
go test ./pkg/agent/history/... -v
```

### Run Specific Test Categories

**Unit Tests Only:**
```bash
go test ./pkg/agent/history/... -v -run "Unit"
```

**Integration Tests Only:**
```bash
go test ./pkg/agent/history/... -v -run "Integration"
```

**Analytical Test Suite:**
```bash
go test ./pkg/agent/history/... -v -run "TestSuite_All"
```

### Run Individual Tests

**Buffer Window Tests:**
```bash
go test ./pkg/agent/history/... -v -run "TestBufferWindow"
```

**Summary Buffer Tests:**
```bash
go test ./pkg/agent/history/... -v -run "TestSummaryBuffer"
```

### Run with Coverage

```bash
go test ./pkg/agent/history/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Test Determinism

All tests are **deterministic** - they produce the same results every time:

### 1. Deterministic Summarizer

The `DeterministicSummarizer` mock provides predictable output:

```go
type DeterministicSummarizer struct {
    CallCount    int
    SummaryCalls [][]llms.Message
}

// Creates predictable summaries: "SUMMARY#1[...]", "SUMMARY#2[...]"
func (d *DeterministicSummarizer) SummarizeConversation(...)
```

**Benefits:**
- Predictable output format
- Call tracking for verification
- No LLM API dependencies
- Fast test execution

### 2. Exact Token Counting

Tests use real `tiktoken` token counting (not estimates):

```go
// Accurate token counting
tokenCounter, _ := utils.NewTokenCounter("gpt-4o")
tokens := tokenCounter.CountMessages(messages)
```

**Benefits:**
- 100% accurate token counts
- Verifies real-world behavior
- No approximations

### 3. Controlled Inputs

All tests use controlled, predictable inputs:

```go
// Exact message content
messages := []string{"First", "Second", "Third"}

// Exact configuration
config := HistoryConfig{
    Budget:    100,
    Threshold: 0.8,
    Target:    0.6,
}
```

---

## Analytical Validation

### What Gets Validated

#### Buffer Window Strategy

1. **Exact Window Size**
   - Add 10 messages with window size 5
   - Verify exactly ≤5 messages kept
   - Validate no overflow

2. **FIFO Ordering**
   - Add messages: "First", "Second", "Third", "Fourth", "Fifth"
   - Verify oldest messages dropped first
   - Verify order maintained

3. **Session Isolation**
   - Create sessions A, B, C
   - Add different messages to each
   - Verify no cross-contamination

#### Summary Buffer Strategy

1. **Token Budget Enforcement**
   - Set budget to 100 tokens
   - Add messages exceeding budget
   - Verify budget respected (via summarization or truncation)

2. **Threshold Triggering**
   - Set threshold to 80%
   - Add messages until 80% exceeded
   - Verify summarization triggered

3. **Summarization Flow**
   - Trigger summarization
   - Verify summary is first message
   - Verify summary role = "system"
   - Verify summary content format

4. **Hierarchical Summarization**
   - Trigger multiple summarizations
   - Verify summaries are themselves summarized
   - Validate compression cascade

---

## Example Test Output

```
=== RUN   TestSuite_All
=== RUN   TestSuite_All/Analytical_Tests
=== RUN   TestSuite_All/Analytical_Tests/1._Buffer_Window_-_Window_Size_Enforcement
=== RUN   TestSuite_All/Analytical_Tests/2._Buffer_Window_-_FIFO_Ordering
=== RUN   TestSuite_All/Analytical_Tests/3._Summary_Buffer_-_Token_Budget_Enforcement
=== RUN   TestSuite_All/Analytical_Tests/4._Summary_Buffer_-_Threshold_Triggering
=== RUN   TestSuite_All/Analytical_Tests/5._Summary_Buffer_-_Summarization_Flow
=== RUN   TestSuite_All/Analytical_Tests/6._Session_Isolation
=== RUN   TestSuite_All/Analytical_Tests/7._Strategy_Factory
=== RUN   TestSuite_All/Analytical_Tests/8._Default_Values
======================================================================
ANALYTICAL TEST SUITE SUMMARY
======================================================================
✅ PASS | Buffer Window respects window size
         Added 10 messages, kept 5 (max: 5)
✅ PASS | Buffer Window maintains FIFO order
         FIFO maintained, kept last 3 messages
✅ PASS | Summary Buffer respects token budget
         Budget: 100, Summarizations: 2
✅ PASS | Summary Buffer triggers at threshold
         Threshold: 80%, Summarizations triggered: 2
✅ PASS | Summary Buffer creates valid summary
         Summary created, 5 messages in history
✅ PASS | Sessions are completely isolated
         Session A: 2 msgs, Session B: 1 msgs
✅ PASS | Factory creates correct strategies
         BW: buffer_window, SB: summary_buffer, Default: summary_buffer
✅ PASS | Default values applied correctly
         BW size: 20, SB budget: 2000, threshold: 0.8, target: 0.6
======================================================================
TOTAL: 8 tests | PASSED: 8 | FAILED: 0
======================================================================
--- PASS: TestSuite_All (0.12s)
PASS
ok      github.com/kadirpekel/hector/pkg/agent/history  0.368s
```

---

## Test Coverage Goals

| Component | Target | Current |
|-----------|--------|---------|
| Buffer Window Strategy | 90%+ | ✅ |
| Summary Buffer Strategy | 90%+ | ✅ |
| Factory | 100% | ✅ |
| Integration | 85%+ | ✅ |

---

## Adding New Tests

### 1. Unit Tests

Add to `*_unit_test.go`:

```go
func TestBufferWindow_Unit_YourFeature(t *testing.T) {
    // DETERMINISTIC TEST: What you're testing
    strategy, err := NewHistoryStrategy(HistoryConfig{
        Strategy: "buffer_window",
        WindowSize: 5,
    })
    require.NoError(t, err)

    // Your test logic
    // Assert exact behavior
}
```

### 2. Integration Tests

Add to `integration_test.go`:

```go
func TestIntegration_YourScenario(t *testing.T) {
    // Test end-to-end workflow
    // Verify multiple components work together
}
```

### 3. Analytical Tests

Add to `test_suite.go`:

```go
func (s *TestSuite) TestYourAnalysis(t *testing.T) {
    // Perform analytical validation
    // Track result
    s.AddResult("Your test name", passed, notes)
}
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run History Strategy Tests
  run: |
    go test ./pkg/agent/history/... -v -coverprofile=coverage.out
    go tool cover -func=coverage.out
```

### Pre-commit Hook

```bash
#!/bin/bash
go test ./pkg/agent/history/... -v
if [ $? -ne 0 ]; then
    echo "❌ History strategy tests failed"
    exit 1
fi
```

---

## Debugging Failed Tests

### 1. View Detailed Output

```bash
go test ./pkg/agent/history/... -v -run "TestName"
```

### 2. Enable Debug Logging

Tests log important events:
- Summarization triggers
- Token counts
- Session operations

### 3. Check Summarizer Call History

```go
if summarizer.CallCount > 0 {
    for i, call := range summarizer.SummaryCalls {
        t.Logf("Summarization #%d: %d messages", i+1, len(call))
    }
}
```

---

## Performance Benchmarks

Run benchmarks to verify performance:

```bash
go test ./pkg/agent/history/... -bench=. -benchmem
```

Expected performance:
- Buffer Window: < 1µs per operation
- Summary Buffer (no summarization): < 10µs per operation
- Summary Buffer (with summarization): < 100ms per summarization

---

## Conclusion

This testing foundation provides:
- ✅ **Deterministic** - Same results every time
- ✅ **Analytical** - Validates exact behavior
- ✅ **Comprehensive** - Covers all scenarios
- ✅ **Fast** - No external API dependencies
- ✅ **Maintainable** - Clear structure and documentation

Run `go test ./pkg/agent/history/... -v` to validate everything works correctly!

