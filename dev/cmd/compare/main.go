// KPI comparison tool for Hector self-development
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kadirpekel/hector/dev"
)

func main() {
	// Parse flags
	beforeFile := flag.String("before", "", "Before KPI JSON file (required)")
	afterFile := flag.String("after", "", "After KPI JSON file (required)")
	flag.Parse()

	if *beforeFile == "" || *afterFile == "" {
		fmt.Println("Usage: go run dev/cmd/compare/main.go --before <file> --after <file>")
		os.Exit(1)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║              HECTOR KPI COMPARISON                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load KPIs
	fmt.Printf("📖 Loading baseline from %s\n", *beforeFile)
	before, err := dev.LoadKPIsFromFile(*beforeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load before KPIs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📖 Loading current from %s\n", *afterFile)
	after, err := dev.LoadKPIsFromFile(*afterFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load after KPIs: %v\n", err)
		os.Exit(1)
	}

	// Compare
	fmt.Println("\n🔍 Analyzing changes...\n")
	comparison := before.Compare(after)

	// Print comparison
	fmt.Println(comparison.FormatSummary())

	// Print detailed metrics
	fmt.Println("\n📊 DETAILED METRICS:\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	printMetricComparison("Tests Pass Rate", before.Functional.TestPassRate, after.Functional.TestPassRate, "%.1f%%", true)
	printMetricComparison("Test Coverage", before.Functional.TestCoverage, after.Functional.TestCoverage, "%.1f%%", true)
	printMetricComparison("Avg Tokens/Request", float64(before.Efficiency.AvgTokensPerRequest), float64(after.Efficiency.AvgTokensPerRequest), "%.0f", false)
	printMetricComparison("Token Efficiency", before.Efficiency.TokenEfficiency, after.Efficiency.TokenEfficiency, "%.3f", true)
	printMetricComparison("Avg Response Time", float64(before.Performance.AvgResponseTime), float64(after.Performance.AvgResponseTime), "%.0fms", false)
	printMetricComparison("P95 Latency", float64(before.Performance.P95Latency), float64(after.Performance.P95Latency), "%.0fms", false)
	printMetricComparison("Throughput", before.Performance.ThroughputOpsPerSec, after.Performance.ThroughputOpsPerSec, "%.2f ops/s", true)
	printMetricComparison("Memory Usage", float64(before.Performance.MemoryUsageAvg)/(1024*1024), float64(after.Performance.MemoryUsageAvg)/(1024*1024), "%.2f MB", false)
	printMetricComparison("Linter Issues", float64(before.Quality.LinterIssues), float64(after.Quality.LinterIssues), "%.0f", false)
	printMetricComparison("Cyclomatic Complexity", before.Quality.CyclomaticComplexity, after.Quality.CyclomaticComplexity, "%.1f", false)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Verdict
	fmt.Println()
	if comparison.IsSignificant {
		if comparison.OverallScore > 0 {
			fmt.Printf("🎉 VERDICT: SIGNIFICANT IMPROVEMENT (%.1f/100)\n", comparison.OverallScore)
			fmt.Println("   This change should be committed!")
		} else {
			fmt.Printf("⚠️  VERDICT: REGRESSION (%.1f/100)\n", comparison.OverallScore)
			fmt.Println("   This change should be reviewed carefully")
		}
	} else {
		fmt.Printf("ℹ️  VERDICT: MINOR CHANGE (%.1f/100)\n", comparison.OverallScore)
		fmt.Println("   Not significant enough to warrant a commit")
	}
}

func printMetricComparison(name string, before, after float64, format string, higherIsBetter bool) {
	beforeStr := fmt.Sprintf(format, before)
	afterStr := fmt.Sprintf(format, after)

	var change string
	var indicator string

	if before == 0 {
		change = "N/A"
		indicator = "➡️ "
	} else {
		pct := ((after - before) / before) * 100
		if pct > 0 {
			change = fmt.Sprintf("+%.1f%%", pct)
			if higherIsBetter {
				indicator = "✅"
			} else {
				indicator = "❌"
			}
		} else if pct < 0 {
			change = fmt.Sprintf("%.1f%%", pct)
			if higherIsBetter {
				indicator = "❌"
			} else {
				indicator = "✅"
			}
		} else {
			change = "—"
			indicator = "➡️ "
		}
	}

	fmt.Printf("%-25s %12s → %-12s (%10s) %s\n", name+":", beforeStr, afterStr, change, indicator)
}
