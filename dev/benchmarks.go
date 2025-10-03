// Package dev provides self-development capabilities for Hector
// Comprehensive benchmarking and KPI measurement
package dev

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// BENCHMARK RUNNER
// ============================================================================

// BenchmarkRunner orchestrates all benchmarks and collects KPIs
type BenchmarkRunner struct {
	ProjectRoot string
	Verbose     bool
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner(projectRoot string) *BenchmarkRunner {
	return &BenchmarkRunner{
		ProjectRoot: projectRoot,
		Verbose:     false,
	}
}

// RunAll runs all benchmarks and returns comprehensive KPIs
func (r *BenchmarkRunner) RunAll(ctx context.Context) (*HectorKPIs, error) {
	kpis := &HectorKPIs{
		Timestamp: time.Now(),
	}

	// Get git commit
	kpis.GitCommit = r.getGitCommit()

	// Run functional tests
	if err := r.measureFunctionalKPIs(ctx, kpis); err != nil {
		return nil, fmt.Errorf("functional KPIs failed: %w", err)
	}

	// Run performance benchmarks
	if err := r.measurePerformanceKPIs(ctx, kpis); err != nil {
		return nil, fmt.Errorf("performance KPIs failed: %w", err)
	}

	// Measure efficiency (requires test runs)
	if err := r.measureEfficiencyKPIs(ctx, kpis); err != nil {
		return nil, fmt.Errorf("efficiency KPIs failed: %w", err)
	}

	// Analyze code quality
	if err := r.measureQualityKPIs(ctx, kpis); err != nil {
		return nil, fmt.Errorf("quality KPIs failed: %w", err)
	}

	return kpis, nil
}

// ============================================================================
// FUNCTIONAL KPIs
// ============================================================================

func (r *BenchmarkRunner) measureFunctionalKPIs(ctx context.Context, kpis *HectorKPIs) error {
	// Run tests with coverage
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-cover", "-coverprofile=coverage.out", "./...")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if r.Verbose {
		fmt.Println(outputStr)
	}

	// Parse test results
	kpis.Functional.TestsTotal = countOccurrences(outputStr, "RUN")
	kpis.Functional.TestsPassed = countOccurrences(outputStr, "PASS:")
	kpis.Functional.TestsFailed = countOccurrences(outputStr, "FAIL:")

	if kpis.Functional.TestsTotal > 0 {
		kpis.Functional.TestPassRate = (float64(kpis.Functional.TestsPassed) / float64(kpis.Functional.TestsTotal)) * 100
	}

	// Parse coverage
	if coverage := r.parseCoverage(outputStr); coverage > 0 {
		kpis.Functional.TestCoverage = coverage
	}

	// Count benchmarks
	benchCmd := exec.CommandContext(ctx, "go", "test", "-list=Benchmark.*", "./...")
	benchCmd.Dir = r.ProjectRoot
	if benchOutput, err := benchCmd.Output(); err == nil {
		kpis.Functional.BenchmarksTotal = len(strings.Split(strings.TrimSpace(string(benchOutput)), "\n"))
	}

	if err != nil && kpis.Functional.TestsFailed > 0 {
		return fmt.Errorf("tests failed: %d failures", kpis.Functional.TestsFailed)
	}

	return nil
}

// ============================================================================
// PERFORMANCE KPIs
// ============================================================================

func (r *BenchmarkRunner) measurePerformanceKPIs(ctx context.Context, kpis *HectorKPIs) error {
	// Run benchmarks
	cmd := exec.CommandContext(ctx, "go", "test", "-bench=.", "-benchmem", "-run=^$", "./...")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("benchmark execution failed: %w", err)
	}

	outputStr := string(output)
	if r.Verbose {
		fmt.Println(outputStr)
	}

	// Parse benchmark results
	results := r.parseBenchmarkResults(outputStr)
	if len(results) == 0 {
		return fmt.Errorf("no benchmark results found")
	}

	// Calculate aggregate metrics
	var totalTime int64
	var totalMemory int64
	var totalAllocs int64
	count := len(results)

	minTime := int64(^uint64(0) >> 1) // Max int64
	maxTime := int64(0)

	for _, result := range results {
		totalTime += result.NsPerOp
		totalMemory += result.BytesPerOp
		totalAllocs += result.AllocsPerOp

		if result.NsPerOp < minTime {
			minTime = result.NsPerOp
		}
		if result.NsPerOp > maxTime {
			maxTime = result.NsPerOp
		}
	}

	if count > 0 {
		kpis.Performance.AvgResponseTime = int(totalTime / int64(count) / 1_000_000) // Convert to ms
		kpis.Performance.MemoryUsageAvg = totalMemory / int64(count)
		kpis.Performance.AllocsPerOp = totalAllocs / int64(count)

		// Estimate percentiles (simplified)
		kpis.Performance.P50Latency = kpis.Performance.AvgResponseTime
		kpis.Performance.P95Latency = int(float64(maxTime) / 1_000_000 * 0.95)
		kpis.Performance.P99Latency = int(float64(maxTime) / 1_000_000 * 0.99)

		// Calculate throughput
		if kpis.Performance.AvgResponseTime > 0 {
			kpis.Performance.ThroughputOpsPerSec = 1000.0 / float64(kpis.Performance.AvgResponseTime)
		}
	}

	// Get current memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	kpis.Performance.MemoryUsagePeak = int64(m.Sys)

	return nil
}

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string
	Iterations  int
	NsPerOp     int64
	BytesPerOp  int64
	AllocsPerOp int64
}

func (r *BenchmarkRunner) parseBenchmarkResults(output string) []BenchmarkResult {
	var results []BenchmarkResult

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		result := BenchmarkResult{
			Name: fields[0],
		}

		// Parse iterations
		if iter, err := strconv.Atoi(fields[1]); err == nil {
			result.Iterations = iter
		}

		// Parse ns/op
		if nsPerOp, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], "ns/op"), 64); err == nil {
			result.NsPerOp = int64(nsPerOp)
		}

		// Parse B/op and allocs/op if present
		for i := 3; i < len(fields); i++ {
			if strings.HasSuffix(fields[i], "B/op") {
				if bytes, err := strconv.ParseInt(strings.TrimSuffix(fields[i], "B/op"), 10, 64); err == nil {
					result.BytesPerOp = bytes
				}
			}
			if strings.HasSuffix(fields[i], "allocs/op") {
				if allocs, err := strconv.ParseInt(strings.TrimSuffix(fields[i], "allocs/op"), 10, 64); err == nil {
					result.AllocsPerOp = allocs
				}
			}
		}

		results = append(results, result)
	}

	return results
}

// ============================================================================
// EFFICIENCY KPIs (Token Usage)
// ============================================================================

func (r *BenchmarkRunner) measureEfficiencyKPIs(ctx context.Context, kpis *HectorKPIs) error {
	// Run token usage tests if they exist
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-run=TestTokenUsage", "./utils/...")
	cmd.Dir = r.ProjectRoot

	output, _ := cmd.Output()
	outputStr := string(output)

	// Parse token metrics from test output
	// This assumes tests log token usage in a specific format
	tokens := r.parseTokenUsage(outputStr)

	if len(tokens) > 0 {
		total := 0
		min := tokens[0]
		max := tokens[0]

		for _, t := range tokens {
			total += t
			if t < min {
				min = t
			}
			if t > max {
				max = t
			}
		}

		kpis.Efficiency.AvgTokensPerRequest = total / len(tokens)
		kpis.Efficiency.MinTokensPerRequest = min
		kpis.Efficiency.MaxTokensPerRequest = max

		// Calculate efficiency score (output quality / tokens)
		// Higher is better - this is a simplified metric
		kpis.Efficiency.TokenEfficiency = 100.0 / float64(kpis.Efficiency.AvgTokensPerRequest)

		// Estimate cost (GPT-4o-mini pricing: $0.15/1M input, $0.60/1M output)
		avgCostPerRequest := (float64(kpis.Efficiency.AvgTokensPerRequest) / 1_000_000) * 0.375 // Average of input/output
		kpis.Efficiency.EstimatedCostPer1kReq = avgCostPerRequest * 1000
	} else {
		// Default values if no token tests exist
		kpis.Efficiency.AvgTokensPerRequest = 1000
		kpis.Efficiency.TokenEfficiency = 0.1
	}

	return nil
}

func (r *BenchmarkRunner) parseTokenUsage(output string) []int {
	var tokens []int

	// Look for lines like: "Tokens used: 1234"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Tokens used:") || strings.Contains(line, "tokens:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if (field == "Tokens" || field == "tokens:") && i+1 < len(fields) {
					if t, err := strconv.Atoi(strings.TrimSuffix(fields[i+1], ",")); err == nil {
						tokens = append(tokens, t)
					}
				}
			}
		}
	}

	return tokens
}

// ============================================================================
// CODE QUALITY KPIs
// ============================================================================

func (r *BenchmarkRunner) measureQualityKPIs(ctx context.Context, kpis *HectorKPIs) error {
	// Run linter
	if err := r.runLinter(ctx, kpis); err != nil {
		// Don't fail on linter errors, just log them
		if r.Verbose {
			fmt.Printf("Linter analysis: %v\n", err)
		}
	}

	// Count lines of code
	kpis.Quality.LinesOfCode = r.countLinesOfCode()

	// Calculate code metrics
	if err := r.analyzeCodeMetrics(ctx, kpis); err != nil {
		if r.Verbose {
			fmt.Printf("Code metrics: %v\n", err)
		}
	}

	return nil
}

func (r *BenchmarkRunner) runLinter(ctx context.Context, kpis *HectorKPIs) error {
	// Try golangci-lint first
	cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--out-format=json", "./...")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		// If golangci-lint not available, try go vet
		return r.runGoVet(ctx, kpis)
	}

	// Parse JSON output for issue counts
	outputStr := string(output)
	kpis.Quality.LinterIssues = countOccurrences(outputStr, `"Severity"`)
	kpis.Quality.CriticalIssues = countOccurrences(outputStr, `"error"`)
	kpis.Quality.WarningIssues = countOccurrences(outputStr, `"warning"`)

	return nil
}

func (r *BenchmarkRunner) runGoVet(ctx context.Context, kpis *HectorKPIs) error {
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = r.ProjectRoot

	output, _ := cmd.CombinedOutput()

	// Count issues in vet output
	lines := strings.Split(string(output), "\n")
	issueCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			issueCount++
		}
	}

	kpis.Quality.LinterIssues = issueCount
	kpis.Quality.CriticalIssues = issueCount / 2 // Rough estimate

	return nil
}

func (r *BenchmarkRunner) analyzeCodeMetrics(ctx context.Context, kpis *HectorKPIs) error {
	// Try gocyclo for complexity
	cmd := exec.CommandContext(ctx, "gocyclo", "-avg", ".")
	cmd.Dir = r.ProjectRoot

	if output, err := cmd.Output(); err == nil {
		// Parse average complexity
		outputStr := string(output)
		if strings.Contains(outputStr, "Average:") {
			fields := strings.Fields(outputStr)
			for i, field := range fields {
				if field == "Average:" && i+1 < len(fields) {
					if complexity, err := strconv.ParseFloat(fields[i+1], 64); err == nil {
						kpis.Quality.CyclomaticComplexity = complexity
					}
				}
			}
		}
	} else {
		// Fallback: estimate from LOC
		kpis.Quality.CyclomaticComplexity = float64(kpis.Quality.LinesOfCode) / 100.0
	}

	// Estimate code duplication (simplified)
	kpis.Quality.CodeDuplication = 5.0 // Default estimate

	// Calculate comment ratio
	commentLines := r.countCommentLines()
	if kpis.Quality.LinesOfCode > 0 {
		kpis.Quality.CommentRatio = float64(commentLines) / float64(kpis.Quality.LinesOfCode)
	}

	// Estimate technical debt
	kpis.Quality.TechnicalDebt = (kpis.Quality.LinterIssues * 5) +
		int(kpis.Quality.CyclomaticComplexity*10) +
		(kpis.Quality.CriticalIssues * 30)

	return nil
}

func (r *BenchmarkRunner) countLinesOfCode() int {
	cmd := exec.Command("find", ".", "-name", "*.go", "-not", "-path", "*/vendor/*", "-exec", "wc", "-l", "{}", "+")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	total := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			if count, err := strconv.Atoi(fields[0]); err == nil {
				total += count
			}
		}
	}

	return total
}

func (r *BenchmarkRunner) countCommentLines() int {
	cmd := exec.Command("sh", "-c", "grep -r --include='*.go' '^[[:space:]]*//' . | wc -l")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (r *BenchmarkRunner) getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = r.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

func (r *BenchmarkRunner) parseCoverage(output string) float64 {
	// Look for coverage percentage in output like: "coverage: 75.5% of statements"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") && strings.Contains(line, "%") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "coverage:" && i+1 < len(fields) {
					coverageStr := strings.TrimSuffix(fields[i+1], "%")
					if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
						return coverage
					}
				}
			}
		}
	}
	return 0
}

func countOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}
