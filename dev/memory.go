// Package dev provides self-development capabilities for Hector
// Learning from past improvements via commit history
package dev

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// DEVELOPMENT MEMORY
// ============================================================================

// DevMemory stores and analyzes past development improvements
type DevMemory struct {
	gitManager *GitManager
	commits    []*DevCommit
	insights   *DevInsights
}

// NewDevMemory creates a new development memory system
func NewDevMemory(gitManager *GitManager) *DevMemory {
	return &DevMemory{
		gitManager: gitManager,
		commits:    make([]*DevCommit, 0),
	}
}

// Load loads recent commits and analyzes them
func (m *DevMemory) Load(commitCount int) error {
	commits, err := m.gitManager.GetRecentCommits(commitCount)
	if err != nil {
		return fmt.Errorf("failed to load commits: %w", err)
	}

	m.commits = commits
	m.insights = m.analyzeCommits()

	return nil
}

// ============================================================================
// INSIGHTS GENERATION
// ============================================================================

// DevInsights contains learned insights from past improvements
type DevInsights struct {
	TotalImprovements  int
	SuccessfulPatterns []ImprovementPattern
	FailedPatterns     []ImprovementPattern
	BestCategories     []CategoryStats
	WorstCategories    []CategoryStats
	AverageScore       float64
	TrendDirection     string // "improving", "stable", "declining"
	RecommendedFocus   []string
}

// ImprovementPattern represents a pattern of successful/failed improvements
type ImprovementPattern struct {
	Category       string
	Description    string
	AvgScore       float64
	Occurrences    int
	SuccessRate    float64
	TopMetrics     []string
	Recommendation string
}

// CategoryStats represents statistics for a category
type CategoryStats struct {
	Category        string
	Count           int
	AvgScore        float64
	TotalScore      float64
	TopImprovements []string
}

func (m *DevMemory) analyzeCommits() *DevInsights {
	insights := &DevInsights{
		SuccessfulPatterns: make([]ImprovementPattern, 0),
		FailedPatterns:     make([]ImprovementPattern, 0),
		BestCategories:     make([]CategoryStats, 0),
		WorstCategories:    make([]CategoryStats, 0),
		RecommendedFocus:   make([]string, 0),
	}

	if len(m.commits) == 0 {
		return insights
	}

	// Analyze by category
	categoryMap := m.groupByCategory()
	insights.BestCategories, insights.WorstCategories = m.rankCategories(categoryMap)

	// Identify successful patterns
	insights.SuccessfulPatterns = m.identifySuccessfulPatterns(categoryMap)

	// Identify failed patterns
	insights.FailedPatterns = m.identifyFailedPatterns(categoryMap)

	// Calculate average score
	totalScore := 0.0
	for _, commit := range m.commits {
		totalScore += commit.KPIScore
	}
	insights.AverageScore = totalScore / float64(len(m.commits))

	// Determine trend
	insights.TrendDirection = m.determineTrend()

	// Generate recommendations
	insights.RecommendedFocus = m.generateRecommendations(insights)

	insights.TotalImprovements = len(m.commits)

	return insights
}

func (m *DevMemory) groupByCategory() map[string][]*DevCommit {
	categoryMap := make(map[string][]*DevCommit)

	for _, commit := range m.commits {
		if commit.Category == "" {
			commit.Category = "uncategorized"
		}
		categoryMap[commit.Category] = append(categoryMap[commit.Category], commit)
	}

	return categoryMap
}

func (m *DevMemory) rankCategories(categoryMap map[string][]*DevCommit) ([]CategoryStats, []CategoryStats) {
	stats := make([]CategoryStats, 0)

	for category, commits := range categoryMap {
		totalScore := 0.0
		topImprovements := make(map[string]int)

		for _, commit := range commits {
			totalScore += commit.KPIScore

			// Track which metrics improved most
			for metric := range commit.Improvements {
				topImprovements[metric]++
			}
		}

		avgScore := totalScore / float64(len(commits))

		// Get top 3 metrics
		topMetrics := m.getTopMetrics(topImprovements, 3)

		stats = append(stats, CategoryStats{
			Category:        category,
			Count:           len(commits),
			AvgScore:        avgScore,
			TotalScore:      totalScore,
			TopImprovements: topMetrics,
		})
	}

	// Sort by average score
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].AvgScore > stats[j].AvgScore
	})

	// Split into best and worst
	midpoint := len(stats) / 2
	if midpoint == 0 {
		midpoint = 1
	}

	best := stats[:midpoint]
	worst := stats[midpoint:]

	return best, worst
}

func (m *DevMemory) identifySuccessfulPatterns(categoryMap map[string][]*DevCommit) []ImprovementPattern {
	patterns := make([]ImprovementPattern, 0)

	for category, commits := range categoryMap {
		// Only consider patterns with significant positive score
		successfulCommits := 0
		totalScore := 0.0
		metricImprovements := make(map[string]int)

		for _, commit := range commits {
			if commit.KPIScore > 5.0 { // Significant improvement threshold
				successfulCommits++
				totalScore += commit.KPIScore

				for metric := range commit.Improvements {
					metricImprovements[metric]++
				}
			}
		}

		if successfulCommits > 0 {
			avgScore := totalScore / float64(successfulCommits)
			successRate := float64(successfulCommits) / float64(len(commits)) * 100

			topMetrics := m.getTopMetrics(metricImprovements, 3)

			pattern := ImprovementPattern{
				Category:    category,
				Description: fmt.Sprintf("%s improvements", category),
				AvgScore:    avgScore,
				Occurrences: successfulCommits,
				SuccessRate: successRate,
				TopMetrics:  topMetrics,
			}

			// Generate recommendation
			if successRate > 70 {
				pattern.Recommendation = fmt.Sprintf("Continue focusing on %s - high success rate", category)
			} else if len(topMetrics) > 0 {
				pattern.Recommendation = fmt.Sprintf("In %s, focus on %s", category, strings.Join(topMetrics, ", "))
			}

			patterns = append(patterns, pattern)
		}
	}

	// Sort by average score
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].AvgScore > patterns[j].AvgScore
	})

	return patterns
}

func (m *DevMemory) identifyFailedPatterns(categoryMap map[string][]*DevCommit) []ImprovementPattern {
	patterns := make([]ImprovementPattern, 0)

	for category, commits := range categoryMap {
		// Look for patterns with negative scores or regressions
		failedCommits := 0
		totalScore := 0.0
		problemMetrics := make(map[string]int)

		for _, commit := range commits {
			if commit.KPIScore < 0 {
				failedCommits++
				totalScore += commit.KPIScore

				for metric := range commit.Regressions {
					problemMetrics[metric]++
				}
			}
		}

		if failedCommits > 0 {
			avgScore := totalScore / float64(failedCommits)
			failureRate := float64(failedCommits) / float64(len(commits)) * 100

			topProblemMetrics := m.getTopMetrics(problemMetrics, 3)

			pattern := ImprovementPattern{
				Category:    category,
				Description: fmt.Sprintf("%s attempts with issues", category),
				AvgScore:    avgScore,
				Occurrences: failedCommits,
				SuccessRate: 100 - failureRate,
				TopMetrics:  topProblemMetrics,
			}

			// Generate recommendation
			if failureRate > 50 {
				pattern.Recommendation = fmt.Sprintf("Avoid %s changes - high failure rate", category)
			} else if len(topProblemMetrics) > 0 {
				pattern.Recommendation = fmt.Sprintf("In %s, be careful with %s", category, strings.Join(topProblemMetrics, ", "))
			}

			patterns = append(patterns, pattern)
		}
	}

	return patterns
}

func (m *DevMemory) determineTrend() string {
	if len(m.commits) < 3 {
		return "insufficient_data"
	}

	// Compare recent vs older commits
	recentCount := len(m.commits) / 3
	if recentCount < 1 {
		recentCount = 1
	}

	recentScore := 0.0
	olderScore := 0.0

	for i := 0; i < recentCount; i++ {
		recentScore += m.commits[i].KPIScore
	}
	recentScore /= float64(recentCount)

	olderCount := 0
	for i := recentCount; i < len(m.commits); i++ {
		olderScore += m.commits[i].KPIScore
		olderCount++
	}
	if olderCount > 0 {
		olderScore /= float64(olderCount)
	}

	diff := recentScore - olderScore

	if diff > 5 {
		return "improving"
	} else if diff < -5 {
		return "declining"
	}

	return "stable"
}

func (m *DevMemory) generateRecommendations(insights *DevInsights) []string {
	recommendations := make([]string, 0)

	// Recommend based on successful patterns
	if len(insights.SuccessfulPatterns) > 0 {
		best := insights.SuccessfulPatterns[0]
		recommendations = append(recommendations,
			fmt.Sprintf("Focus on %s improvements (%.1f avg score)", best.Category, best.AvgScore))
	}

	// Recommend based on trend
	switch insights.TrendDirection {
	case "improving":
		recommendations = append(recommendations, "Continue current approach - showing positive trend")
	case "declining":
		recommendations = append(recommendations, "Review recent changes - performance declining")
	case "stable":
		recommendations = append(recommendations, "Try new optimization approaches")
	}

	// Recommend avoiding failed patterns
	if len(insights.FailedPatterns) > 0 {
		worst := insights.FailedPatterns[0]
		recommendations = append(recommendations,
			fmt.Sprintf("Exercise caution with %s changes", worst.Category))
	}

	// Recommend unexplored categories
	exploredCategories := make(map[string]bool)
	for _, commit := range m.commits {
		exploredCategories[commit.Category] = true
	}

	allCategories := []string{"performance", "efficiency", "reasoning", "architecture", "quality"}
	for _, category := range allCategories {
		if !exploredCategories[category] {
			recommendations = append(recommendations,
				fmt.Sprintf("Consider exploring %s improvements", category))
			break // Only suggest one unexplored area
		}
	}

	return recommendations
}

func (m *DevMemory) getTopMetrics(metrics map[string]int, limit int) []string {
	type metricCount struct {
		name  string
		count int
	}

	counts := make([]metricCount, 0)
	for name, count := range metrics {
		counts = append(counts, metricCount{name, count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	result := make([]string, 0)
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, counts[i].name)
	}

	return result
}

// ============================================================================
// QUERY INTERFACE
// ============================================================================

// GetInsights returns the analyzed insights
func (m *DevMemory) GetInsights() *DevInsights {
	return m.insights
}

// GetSuccessfulCommits returns commits with positive scores
func (m *DevMemory) GetSuccessfulCommits() []*DevCommit {
	successful := make([]*DevCommit, 0)
	for _, commit := range m.commits {
		if commit.KPIScore > 5.0 {
			successful = append(successful, commit)
		}
	}
	return successful
}

// GetRecentTrend returns commits from the last N days
func (m *DevMemory) GetRecentTrend(days int) []*DevCommit {
	cutoff := time.Now().AddDate(0, 0, -days)
	recent := make([]*DevCommit, 0)

	for _, commit := range m.commits {
		if commit.Timestamp.After(cutoff) {
			recent = append(recent, commit)
		}
	}

	return recent
}

// FormatLearnings returns a human-readable summary of learnings
func (m *DevMemory) FormatLearnings() string {
	if m.insights == nil {
		return "No insights available - load commit history first"
	}

	var output strings.Builder

	output.WriteString("\n╔═══════════════════════════════════════════════════════════╗\n")
	output.WriteString("║           HECTOR DEVELOPMENT LEARNINGS                    ║\n")
	output.WriteString("╚═══════════════════════════════════════════════════════════╝\n\n")

	output.WriteString(fmt.Sprintf("📊 Total Improvements Attempted: %d\n", m.insights.TotalImprovements))
	output.WriteString(fmt.Sprintf("📈 Average Score: %.1f/100\n", m.insights.AverageScore))
	output.WriteString(fmt.Sprintf("🎯 Trend: %s\n\n", m.insights.TrendDirection))

	// Successful patterns
	if len(m.insights.SuccessfulPatterns) > 0 {
		output.WriteString("✅ SUCCESSFUL PATTERNS:\n")
		output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, pattern := range m.insights.SuccessfulPatterns {
			if i >= 3 {
				break // Top 3
			}
			output.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, pattern.Category))
			output.WriteString(fmt.Sprintf("   Score: %.1f/100 | Success Rate: %.1f%% | Count: %d\n",
				pattern.AvgScore, pattern.SuccessRate, pattern.Occurrences))
			if len(pattern.TopMetrics) > 0 {
				output.WriteString(fmt.Sprintf("   Top Metrics: %s\n", strings.Join(pattern.TopMetrics, ", ")))
			}
			if pattern.Recommendation != "" {
				output.WriteString(fmt.Sprintf("   💡 %s\n", pattern.Recommendation))
			}
		}
		output.WriteString("\n")
	}

	// Best categories
	if len(m.insights.BestCategories) > 0 {
		output.WriteString("🏆 TOP PERFORMING CATEGORIES:\n")
		output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, cat := range m.insights.BestCategories {
			if i >= 3 {
				break
			}
			output.WriteString(fmt.Sprintf("%d. %s (%.1f avg score, %d attempts)\n",
				i+1, cat.Category, cat.AvgScore, cat.Count))
		}
		output.WriteString("\n")
	}

	// Recommendations
	if len(m.insights.RecommendedFocus) > 0 {
		output.WriteString("💡 RECOMMENDATIONS:\n")
		output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, rec := range m.insights.RecommendedFocus {
			output.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		output.WriteString("\n")
	}

	return output.String()
}
