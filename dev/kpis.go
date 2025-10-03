// Package dev provides self-development capabilities for Hector
// KPI tracking, benchmarking, and autonomous improvement
package dev

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ============================================================================
// KPI DEFINITIONS
// ============================================================================

// HectorKPIs represents measurable performance indicators
type HectorKPIs struct {
	Timestamp time.Time `json:"timestamp"`
	GitCommit string    `json:"git_commit,omitempty"`

	// Functional Quality
	Functional FunctionalKPIs `json:"functional"`

	// Token Efficiency
	Efficiency EfficiencyKPIs `json:"efficiency"`

	// Performance (Speed)
	Performance PerformanceKPIs `json:"performance"`

	// Code Quality
	Quality QualityKPIs `json:"quality"`
}

// FunctionalKPIs measures correctness and reliability
type FunctionalKPIs struct {
	TestsTotal       int     `json:"tests_total"`
	TestsPassed      int     `json:"tests_passed"`
	TestsFailed      int     `json:"tests_failed"`
	TestPassRate     float64 `json:"test_pass_rate"` // Percentage
	TestCoverage     float64 `json:"test_coverage"`  // Percentage
	BenchmarksTotal  int     `json:"benchmarks_total"`
	BenchmarksPassed int     `json:"benchmarks_passed"`
}

// EfficiencyKPIs measures token usage and cost
type EfficiencyKPIs struct {
	AvgTokensPerRequest   int     `json:"avg_tokens_per_request"`
	MinTokensPerRequest   int     `json:"min_tokens_per_request"`
	MaxTokensPerRequest   int     `json:"max_tokens_per_request"`
	TokenEfficiency       float64 `json:"token_efficiency"`          // Output quality / tokens
	EstimatedCostPer1kReq float64 `json:"estimated_cost_per_1k_req"` // USD
}

// PerformanceKPIs measures speed and throughput
type PerformanceKPIs struct {
	AvgResponseTime     int     `json:"avg_response_time_ms"`
	P50Latency          int     `json:"p50_latency_ms"`
	P95Latency          int     `json:"p95_latency_ms"`
	P99Latency          int     `json:"p99_latency_ms"`
	ThroughputOpsPerSec float64 `json:"throughput_ops_per_sec"`
	MemoryUsageAvg      int64   `json:"memory_usage_avg_bytes"`
	MemoryUsagePeak     int64   `json:"memory_usage_peak_bytes"`
	AllocsPerOp         int64   `json:"allocs_per_op"`
}

// QualityKPIs measures code quality
type QualityKPIs struct {
	LinterIssues         int     `json:"linter_issues"`
	CriticalIssues       int     `json:"critical_issues"`
	WarningIssues        int     `json:"warning_issues"`
	CodeDuplication      float64 `json:"code_duplication_pct"`  // Percentage
	CyclomaticComplexity float64 `json:"cyclomatic_complexity"` // Average
	LinesOfCode          int     `json:"lines_of_code"`
	CommentRatio         float64 `json:"comment_ratio"`          // Comments / LOC
	TechnicalDebt        int     `json:"technical_debt_minutes"` // Estimated minutes
}

// ============================================================================
// KPI COMPARISON & IMPROVEMENT DETECTION
// ============================================================================

// KPIComparison represents the difference between two KPI snapshots
type KPIComparison struct {
	Before        *HectorKPIs        `json:"before"`
	After         *HectorKPIs        `json:"after"`
	Improvements  map[string]float64 `json:"improvements"` // Metric -> % improvement
	Regressions   map[string]float64 `json:"regressions"`  // Metric -> % regression
	IsSignificant bool               `json:"is_significant"`
	OverallScore  float64            `json:"overall_score"` // -100 to +100
}

// Compare compares two KPI snapshots and returns improvement analysis
func (k *HectorKPIs) Compare(other *HectorKPIs) *KPIComparison {
	if other == nil {
		return nil
	}

	comparison := &KPIComparison{
		Before:       k,
		After:        other,
		Improvements: make(map[string]float64),
		Regressions:  make(map[string]float64),
	}

	// Compare functional metrics
	comparison.compareFunctional()

	// Compare efficiency metrics
	comparison.compareEfficiency()

	// Compare performance metrics
	comparison.comparePerformance()

	// Compare quality metrics
	comparison.compareQuality()

	// Calculate overall score
	comparison.calculateOverallScore()

	// Determine significance (>5% improvement in any major metric)
	comparison.IsSignificant = comparison.OverallScore > 5.0

	return comparison
}

func (c *KPIComparison) compareFunctional() {
	before := c.Before.Functional
	after := c.After.Functional

	if improvement := calculateImprovement(before.TestPassRate, after.TestPassRate); improvement != 0 {
		if improvement > 0 {
			c.Improvements["test_pass_rate"] = improvement
		} else {
			c.Regressions["test_pass_rate"] = -improvement
		}
	}

	if improvement := calculateImprovement(before.TestCoverage, after.TestCoverage); improvement != 0 {
		if improvement > 0 {
			c.Improvements["test_coverage"] = improvement
		} else {
			c.Regressions["test_coverage"] = -improvement
		}
	}
}

func (c *KPIComparison) compareEfficiency() {
	before := c.Before.Efficiency
	after := c.After.Efficiency

	// Lower is better for token usage
	if improvement := calculateImprovement(float64(before.AvgTokensPerRequest), float64(after.AvgTokensPerRequest)); improvement < 0 {
		c.Improvements["token_efficiency"] = -improvement
	} else if improvement > 0 {
		c.Regressions["token_efficiency"] = improvement
	}

	// Higher is better for efficiency score
	if improvement := calculateImprovement(before.TokenEfficiency, after.TokenEfficiency); improvement > 0 {
		c.Improvements["token_efficiency_score"] = improvement
	} else if improvement < 0 {
		c.Regressions["token_efficiency_score"] = -improvement
	}
}

func (c *KPIComparison) comparePerformance() {
	before := c.Before.Performance
	after := c.After.Performance

	// Lower is better for latency
	metrics := map[string][2]int{
		"avg_response_time": {before.AvgResponseTime, after.AvgResponseTime},
		"p95_latency":       {before.P95Latency, after.P95Latency},
		"p99_latency":       {before.P99Latency, after.P99Latency},
	}

	for name, values := range metrics {
		if improvement := calculateImprovement(float64(values[0]), float64(values[1])); improvement < 0 {
			c.Improvements[name] = -improvement
		} else if improvement > 0 {
			c.Regressions[name] = improvement
		}
	}

	// Higher is better for throughput
	if improvement := calculateImprovement(before.ThroughputOpsPerSec, after.ThroughputOpsPerSec); improvement > 0 {
		c.Improvements["throughput"] = improvement
	} else if improvement < 0 {
		c.Regressions["throughput"] = -improvement
	}

	// Lower is better for memory
	if improvement := calculateImprovement(float64(before.MemoryUsageAvg), float64(after.MemoryUsageAvg)); improvement < 0 {
		c.Improvements["memory_usage"] = -improvement
	} else if improvement > 0 {
		c.Regressions["memory_usage"] = improvement
	}
}

func (c *KPIComparison) compareQuality() {
	before := c.Before.Quality
	after := c.After.Quality

	// Lower is better for issues
	issueMetrics := map[string][2]int{
		"linter_issues":   {before.LinterIssues, after.LinterIssues},
		"critical_issues": {before.CriticalIssues, after.CriticalIssues},
	}

	for name, values := range issueMetrics {
		if improvement := calculateImprovement(float64(values[0]), float64(values[1])); improvement < 0 {
			c.Improvements[name] = -improvement
		} else if improvement > 0 {
			c.Regressions[name] = improvement
		}
	}

	// Lower is better for complexity and duplication
	if improvement := calculateImprovement(before.CyclomaticComplexity, after.CyclomaticComplexity); improvement < 0 {
		c.Improvements["complexity"] = -improvement
	} else if improvement > 0 {
		c.Regressions["complexity"] = improvement
	}
}

func (c *KPIComparison) calculateOverallScore() {
	improvementScore := 0.0
	regressionScore := 0.0

	// Weight different metrics
	weights := map[string]float64{
		"test_pass_rate":    10.0,
		"test_coverage":     5.0,
		"token_efficiency":  8.0,
		"avg_response_time": 8.0,
		"p95_latency":       7.0,
		"throughput":        6.0,
		"linter_issues":     5.0,
		"critical_issues":   10.0,
		"complexity":        4.0,
		"memory_usage":      5.0,
	}

	for metric, improvement := range c.Improvements {
		weight := weights[metric]
		if weight == 0 {
			weight = 1.0
		}
		improvementScore += improvement * weight
	}

	for metric, regression := range c.Regressions {
		weight := weights[metric]
		if weight == 0 {
			weight = 1.0
		}
		regressionScore += regression * weight
	}

	// Normalize to -100 to +100 scale
	maxScore := 100.0
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight > 0 {
		c.OverallScore = ((improvementScore - regressionScore) / totalWeight) * 100
		if c.OverallScore > maxScore {
			c.OverallScore = maxScore
		} else if c.OverallScore < -maxScore {
			c.OverallScore = -maxScore
		}
	}
}

// calculateImprovement calculates percentage improvement (positive = better)
func calculateImprovement(before, after float64) float64 {
	if before == 0 {
		return 0
	}
	return ((after - before) / before) * 100
}

// ============================================================================
// KPI PERSISTENCE
// ============================================================================

// SaveToFile saves KPIs to a JSON file
func (k *HectorKPIs) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal KPIs: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write KPIs file: %w", err)
	}

	return nil
}

// LoadFromFile loads KPIs from a JSON file
func LoadKPIsFromFile(filepath string) (*HectorKPIs, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read KPIs file: %w", err)
	}

	var kpis HectorKPIs
	if err := json.Unmarshal(data, &kpis); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KPIs: %w", err)
	}

	return &kpis, nil
}

// ============================================================================
// KPI FORMATTING
// ============================================================================

// FormatSummary returns a human-readable summary of KPIs
func (k *HectorKPIs) FormatSummary() string {
	return fmt.Sprintf(`
Hector KPI Summary
==================
Timestamp: %s
Git Commit: %s

Functional Quality:
  Tests: %d/%d passed (%.1f%%)
  Coverage: %.1f%%

Efficiency:
  Avg Tokens/Request: %d
  Token Efficiency: %.2f

Performance:
  Avg Response Time: %dms
  P95 Latency: %dms
  Throughput: %.2f ops/sec
  Memory Usage: %.2f MB

Code Quality:
  Linter Issues: %d (%d critical)
  Cyclomatic Complexity: %.1f
  Code Duplication: %.1f%%
`,
		k.Timestamp.Format(time.RFC3339),
		k.GitCommit,
		k.Functional.TestsPassed, k.Functional.TestsTotal, k.Functional.TestPassRate,
		k.Functional.TestCoverage,
		k.Efficiency.AvgTokensPerRequest,
		k.Efficiency.TokenEfficiency,
		k.Performance.AvgResponseTime,
		k.Performance.P95Latency,
		k.Performance.ThroughputOpsPerSec,
		float64(k.Performance.MemoryUsageAvg)/(1024*1024),
		k.Quality.LinterIssues, k.Quality.CriticalIssues,
		k.Quality.CyclomaticComplexity,
		k.Quality.CodeDuplication,
	)
}

// FormatComparisonSummary returns a human-readable comparison summary
func (c *KPIComparison) FormatSummary() string {
	summary := fmt.Sprintf(`
KPI Comparison
==============
Overall Score: %.1f/100 (%s)

Improvements:
`, c.OverallScore, c.getScoreLabel())

	if len(c.Improvements) == 0 {
		summary += "  None\n"
	} else {
		for metric, improvement := range c.Improvements {
			summary += fmt.Sprintf("  ✅ %s: +%.1f%%\n", metric, improvement)
		}
	}

	summary += "\nRegressions:\n"
	if len(c.Regressions) == 0 {
		summary += "  None\n"
	} else {
		for metric, regression := range c.Regressions {
			summary += fmt.Sprintf("  ❌ %s: -%.1f%%\n", metric, regression)
		}
	}

	return summary
}

func (c *KPIComparison) getScoreLabel() string {
	score := c.OverallScore
	switch {
	case score >= 20:
		return "Excellent"
	case score >= 10:
		return "Great"
	case score >= 5:
		return "Good"
	case score >= 0:
		return "Minor Improvement"
	case score >= -5:
		return "Minor Regression"
	case score >= -10:
		return "Regression"
	default:
		return "Significant Regression"
	}
}
