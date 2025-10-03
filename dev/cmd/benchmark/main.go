// Benchmark runner for Hector self-development
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kadirpekel/hector/dev"
)

func main() {
	// Parse flags
	outputFile := flag.String("output", "", "Output file for KPI JSON (optional)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║        HECTOR COMPREHENSIVE BENCHMARK SUITE               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create benchmark runner
	runner := dev.NewBenchmarkRunner(projectRoot)
	runner.Verbose = *verbose

	// Run all benchmarks
	fmt.Println("🚀 Running comprehensive benchmarks...")
	fmt.Println("   This may take a few minutes...\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	kpis, err := runner.RunAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Benchmark failed: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println(kpis.FormatSummary())

	// Save to file if requested
	if *outputFile != "" {
		if err := kpis.SaveToFile(*outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to save KPIs: %v\n", err)
		} else {
			fmt.Printf("✅ KPIs saved to %s\n", *outputFile)
		}
	}

	fmt.Println("\n✅ Benchmark complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  • Compare with baseline: go run dev/cmd/compare/main.go --before baseline.json --after", *outputFile)
	fmt.Println("  • Run improvements: ./hector --config hector-dev.yaml --workflow self-improvement")
}

func findProjectRoot() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}
