# Hector Benchmarking & Measurement Foundation

## 📊 Overview

This document explains how Hector measures and benchmarks itself for autonomous improvement.

## 🎯 What We Measure (4 KPI Categories)

### 1. **Functional Quality** (Correctness)

**What**: Does the code work correctly?

**How We Measure**:
```bash
# Run all tests
go test -v -cover ./...

# Output parsing:
# - Count: RUN, PASS, FAIL occurrences
# - Parse coverage: "coverage: 75.5% of statements"
```

**Metrics**:
- `TestsTotal`: Count of `RUN` in output
- `TestsPassed`: Count of `PASS:` in output  
- `TestsFailed`: Count of `FAIL:` in output
- `TestPassRate`: (Passed / Total) × 100
- `TestCoverage`: Extracted from coverage output

**Current Status**: ✅ **WORKING** - Parses actual test execution

---

### 2. **Performance** (Speed & Resources)

**What**: How fast and efficient is the code?

**How We Measure**:
```bash
# Run Go benchmarks
go test -bench=. -benchmem -run=^$ ./...

# Example output:
# BenchmarkDAGExecutor-8    1000    303ms/op    13400 B/op    38 allocs/op
```

**Metrics**:
- `AvgResponseTime`: Average ns/op converted to ms
- `P50/P95/P99Latency`: Percentile estimates from min/max
- `ThroughputOpsPerSec`: 1000 / avgResponseTime  
- `MemoryUsageAvg`: Average B/op across benchmarks
- `MemoryUsagePeak`: Current runtime memory stats
- `AllocsPerOp`: Average allocs/op

**Parsing Logic**:
```go
// Extract from lines like:
// BenchmarkName-8    1000    180ms/op    1340 B/op    38 allocs/op

fields := strings.Fields(line)
// fields[2] = "180ms/op" or "180ns/op"
// fields[3] = "1340" (if present)
// fields[4] = "B/op"
// fields[5] = "38"
// fields[6] = "allocs/op"
```

**Current Status**: ✅ **WORKING** - We have real benchmarks in `workflow/benchmark_test.go`

**Example Benchmarks**:
```go
// From workflow/benchmark_test.go
BenchmarkDAGExecutor/1_agent
BenchmarkDAGExecutor/2_agents
BenchmarkDAGExecutor/5_agents
BenchmarkDAGExecutor/10_agents
BenchmarkDAGExecutor/20_agents
BenchmarkAutonomousExecutor/1_agent
...
```

---

### 3. **Efficiency** (Token Usage & Cost)

**What**: How many tokens does the system use?

**How We Measure**:

**Current Implementation**:
```go
// Looks for test output like:
// "Tokens used: 1234"
// "tokens: 1234"

func parseTokenUsage(output string) []int {
    // Parse test logs for token counts
    // Tests need to log: fmt.Printf("Tokens used: %d\n", tokens)
}
```

**Metrics**:
- `AvgTokensPerRequest`: Mean of all token counts
- `MinTokensPerRequest`: Minimum tokens used
- `MaxTokensPerRequest`: Maximum tokens used  
- `TokenEfficiency`: Quality score / tokens (higher = better)
- `EstimatedCostPer1kReq`: Cost calculation based on GPT-4o-mini pricing

**Current Status**: ⚠️ **PARTIAL** - Framework exists, needs actual token tracking tests

**What's Missing**:
```go
// Need to add tests like this:
func TestTokenUsage(t *testing.T) {
    agent := createAgent()
    response, tokens := agent.Generate("test prompt")
    t.Logf("Tokens used: %d", tokens) // ← Benchmarker looks for this
}
```

---

### 4. **Code Quality** (Maintainability)

**What**: How clean and maintainable is the code?

**How We Measure**:

**Linting**:
```bash
# Try golangci-lint first
golangci-lint run --out-format=json ./...

# Fallback to go vet
go vet ./...

# Parse JSON output:
# - Count "Severity" occurrences
# - Count "error" vs "warning"
```

**Code Metrics**:
```bash
# Cyclomatic complexity
gocyclo -avg .

# Lines of code
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} +

# Comment lines  
grep -r --include='*.go' '^[[:space:]]*//' . | wc -l
```

**Metrics**:
- `LinterIssues`: Total issues from linter
- `CriticalIssues`: Count of "error" severity
- `WarningIssues`: Count of "warning" severity
- `CyclomaticComplexity`: From gocyclo (or estimated from LOC)
- `CodeDuplication`: Estimated percentage
- `LinesOfCode`: Total Go lines
- `CommentRatio`: Comment lines / total lines
- `TechnicalDebt`: Estimated minutes (issues × weights)

**Current Status**: ✅ **WORKING** - Uses go vet as fallback

---

## 🔧 How Benchmarking Works

### The Benchmark Runner (`dev/benchmarks.go`)

```go
type BenchmarkRunner struct {
    ProjectRoot string
    Verbose     bool
}

func (r *BenchmarkRunner) RunAll(ctx context.Context) (*HectorKPIs, error) {
    kpis := &HectorKPIs{Timestamp: time.Now()}
    
    // 1. Functional: Run tests
    r.measureFunctionalKPIs(ctx, kpis)
    
    // 2. Performance: Run benchmarks
    r.measurePerformanceKPIs(ctx, kpis)
    
    // 3. Efficiency: Parse token usage
    r.measureEfficiencyKPIs(ctx, kpis)
    
    // 4. Quality: Run linters
    r.measureQualityKPIs(ctx, kpis)
    
    return kpis, nil
}
```

### Execution Flow

```
RunAll()
  │
  ├─► measureFunctionalKPIs()
  │     └─► exec: go test -v -cover ./...
  │         └─► Parse output for pass/fail/coverage
  │
  ├─► measurePerformanceKPIs()
  │     └─► exec: go test -bench=. -benchmem -run=^$
  │         └─► Parse: BenchmarkName-8  1000  180ms/op  ...
  │         └─► Calculate: avg, min, max, throughput
  │
  ├─► measureEfficiencyKPIs()
  │     └─► exec: go test -v -run=TestTokenUsage
  │         └─► Parse: "Tokens used: 1234"
  │         └─► Calculate: avg, min, max, cost
  │
  └─► measureQualityKPIs()
        ├─► exec: golangci-lint run (or go vet)
        ├─► exec: gocyclo -avg .
        ├─► exec: find + wc for LOC
        └─► Calculate: issues, complexity, debt
```

---

## 📈 Real Example: Performance Benchmarking

### Existing Benchmarks

We already have comprehensive benchmarks in `workflow/benchmark_test.go`:

```go
func BenchmarkDAGExecutor(b *testing.B) {
    testCases := []struct {
        name       string
        agentCount int
    }{
        {"1_agent", 1},
        {"2_agents", 2},
        {"5_agents", 5},
        {"10_agents", 10},
        {"20_agents", 20},
    }
    
    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            // Setup mock workflow
            workflow := createMockWorkflow(tc.agentCount)
            executor := NewDAGExecutor(workflow)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                executor.Execute(ctx, request)
            }
        })
    }
}
```

### Benchmark Output (Real)

```
BenchmarkDAGExecutor/1_agent-8         3949    303221 ns/op    13400 B/op     38 allocs/op
BenchmarkDAGExecutor/2_agents-8        1972    606442 ns/op    14000 B/op     55 allocs/op
BenchmarkDAGExecutor/5_agents-8         789   1512105 ns/op    15800 B/op    103 allocs/op
BenchmarkDAGExecutor/10_agents-8        394   3036210 ns/op    21300 B/op    191 allocs/op
BenchmarkDAGExecutor/20_agents-8        196   6072420 ns/op    35300 B/op    354 allocs/op
```

### How We Parse This

```go
func parseBenchmarkResults(output string) []BenchmarkResult {
    for _, line := range strings.Split(output, "\n") {
        if !strings.HasPrefix(line, "Benchmark") {
            continue
        }
        
        fields := strings.Fields(line)
        // fields[0] = "BenchmarkDAGExecutor/1_agent-8"
        // fields[1] = "3949" (iterations)
        // fields[2] = "303221" (ns/op)
        // fields[3] = "13400" (B/op)
        // fields[4] = "B/op"
        // fields[5] = "38" (allocs/op)
        
        result.NsPerOp = parseInt(fields[2])
        result.BytesPerOp = parseInt(fields[3])
        result.AllocsPerOp = parseInt(fields[5])
    }
}
```

### KPI Calculation

```go
// Average response time
totalTime := 0
for _, result := range results {
    totalTime += result.NsPerOp
}
avgNs := totalTime / len(results)
kpis.Performance.AvgResponseTime = avgNs / 1_000_000 // Convert to ms

// Throughput
kpis.Performance.ThroughputOpsPerSec = 1000.0 / float64(avgMs)

// Memory
avgMemory := totalMemory / len(results)
kpis.Performance.MemoryUsageAvg = avgMemory
```

---

## 🔍 What's Actually Working vs. What Needs Work

### ✅ **WORKING NOW**

1. **Test Execution & Parsing**
   - Runs `go test -v -cover ./...`
   - Parses pass/fail/coverage
   - Real test results

2. **Performance Benchmarking**
   - Runs `go test -bench=. -benchmem`
   - Parses benchmark output
   - **We have 6 real benchmarks** in `workflow/benchmark_test.go`
   - Calculates time, memory, allocations

3. **Code Quality Analysis**
   - Runs `go vet` (fallback)
   - Counts LOC
   - Estimates complexity

4. **KPI Comparison**
   - Detects improvements/regressions
   - Calculates overall score
   - Weighted metrics

### ⚠️ **NEEDS ENHANCEMENT**

1. **Token Usage Tracking**
   - **Framework exists** but needs actual tests
   - Need to add token counting to LLM calls
   - Need tests that log: `"Tokens used: X"`

2. **Advanced Code Metrics**
   - Need `golangci-lint` for better linting
   - Need `gocyclo` for complexity
   - Need code duplication detection

3. **More Benchmarks**
   - Token usage benchmarks
   - End-to-end workflow benchmarks
   - Real LLM call benchmarks (expensive!)

---

## 🚀 How to Add Token Tracking

### Step 1: Modify LLM Providers

```go
// In llms/openai.go
func (p *OpenAIProvider) Generate(prompt string) (string, int, error) {
    response, err := p.makeRequest(request)
    
    tokens := response.Usage.TotalTokens // ← Already have this!
    
    return content, tokens, nil // ← Return tokens
}
```

### Step 2: Track in Agent

```go
// In agent/agent.go  
type Agent struct {
    // ...
    tokensUsed int
}

func (a *Agent) Generate(input string) (string, error) {
    response, tokens, err := a.llm.Generate(prompt)
    a.tokensUsed += tokens
    
    return response, err
}
```

### Step 3: Add Token Usage Tests

```go
// New file: utils/tokens_test.go
func TestTokenUsage(t *testing.T) {
    tests := []struct {
        name   string
        prompt string
    }{
        {"simple", "Hello"},
        {"complex", "Write a function..."},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := createTestAgent()
            _, tokens := agent.GenerateWithTokens(tt.prompt)
            
            t.Logf("Tokens used: %d", tokens) // ← Benchmarker finds this
        })
    }
}
```

### Step 4: Benchmark Token Efficiency

```go
func BenchmarkTokenEfficiency(b *testing.B) {
    agent := createAgent()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, tokens := agent.GenerateWithTokens("test prompt")
        b.ReportMetric(float64(tokens), "tokens/op")
    }
}
```

---

## 📊 Complete Benchmarking Workflow

```
┌─────────────────────────────────────────────────────────────┐
│  1. BASELINE: Run initial benchmarks                        │
│     $ go run dev/cmd/benchmark/main.go --output baseline.json │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  2. DEVELOP: Make code changes                              │
│     (either manually or via self-improvement workflow)       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  3. MEASURE: Run benchmarks again                           │
│     $ go run dev/cmd/benchmark/main.go --output after.json  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  4. COMPARE: Analyze improvements                           │
│     $ go run dev/cmd/compare/main.go \                      │
│         --before baseline.json --after after.json           │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  5. DECIDE: Is improvement significant?                     │
│     • Improvement > 5%? ✅ Commit                           │
│     • Regression? ❌ Reject                                  │
│     • Minimal change? ⚠️  Review                            │
└─────────────────────────────────────────────────────────────┘
```

---

## 🎯 Summary: What We Have

| Component | Status | Notes |
|-----------|--------|-------|
| Test Execution | ✅ Working | Real test parsing |
| Performance Benchmarks | ✅ Working | 6 real benchmarks in workflow |
| Memory Tracking | ✅ Working | From Go benchmarks |
| Code Quality | ✅ Working | go vet + LOC counting |
| Token Tracking | ⚠️ Framework Only | Need actual usage tests |
| KPI Comparison | ✅ Working | Full comparison logic |
| Benchmark CLI | ✅ Working | `dev/cmd/benchmark` |
| Compare CLI | ✅ Working | `dev/cmd/compare` |

---

## 💡 Recommendations

### Immediate Actions

1. **Add Token Usage Tests**
   ```bash
   # Create: utils/tokens_test.go
   # Add tests that log: "Tokens used: X"
   ```

2. **Install Optional Tools** (for better metrics)
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
   ```

3. **Run Baseline Benchmarks**
   ```bash
   go run dev/cmd/benchmark/main.go --output kpis-baseline.json
   ```

### Future Enhancements

1. **Real LLM Benchmarks** (careful: costs money!)
   - End-to-end workflow timing
   - Actual token usage with real LLMs
   - Quality metrics (response coherence)

2. **Profiling Integration**
   ```bash
   go test -cpuprofile=cpu.prof -bench=.
   go tool pprof cpu.prof
   ```

3. **Continuous Benchmarking**
   - Run on every commit
   - Track trends over time
   - Alert on regressions

---

## 🔗 Related Files

- `dev/benchmarks.go` - Benchmark runner implementation
- `dev/kpis.go` - KPI definitions & comparison
- `workflow/benchmark_test.go` - **Real performance benchmarks**
- `dev/cmd/benchmark/main.go` - CLI tool
- `dev/cmd/compare/main.go` - Comparison tool

---

**The foundation is solid.** We can measure performance, tests, and code quality **right now**. Token tracking just needs a few tests added, and then the full system will be operational!

