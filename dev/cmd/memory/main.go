// Development memory viewer for Hector self-development
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kadirpekel/hector/dev"
)

func main() {
	// Parse flags
	commitCount := flag.Int("commits", 20, "Number of recent commits to analyze")
	flag.Parse()

	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create memory system
	gitManager := dev.NewGitManager(projectRoot)
	memory := dev.NewDevMemory(gitManager)

	// Load and analyze commits
	fmt.Printf("🧠 Loading and analyzing last %d dev commits...\n", *commitCount)

	if err := memory.Load(*commitCount); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load commit history: %v\n", err)
		fmt.Println("\nℹ️  This is normal if no [hector-dev] commits exist yet.")
		fmt.Println("   Run a self-improvement workflow to create your first commit!")
		os.Exit(0)
	}

	// Print learnings
	fmt.Println(memory.FormatLearnings())

	// Show recent successful commits
	successful := memory.GetSuccessfulCommits()
	if len(successful) > 0 {
		fmt.Println("🏆 RECENT SUCCESSFUL COMMITS:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		count := 5
		if len(successful) < count {
			count = len(successful)
		}

		for i := 0; i < count; i++ {
			commit := successful[i]
			fmt.Printf("\n%d. %s [%s]\n", i+1, commit.Title, commit.Hash[:7])
			fmt.Printf("   Category: %s | Score: %.1f/100\n", commit.Category, commit.KPIScore)
			fmt.Printf("   Date: %s\n", commit.Timestamp.Format("2006-01-02 15:04"))

			if len(commit.Improvements) > 0 {
				fmt.Printf("   Improvements: ")
				first := true
				for metric := range commit.Improvements {
					if !first {
						fmt.Print(", ")
					}
					fmt.Print(metric)
					first = false
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	// Show recent trend
	recent := memory.GetRecentTrend(7)
	if len(recent) > 0 {
		fmt.Println("📈 LAST 7 DAYS TREND:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		totalScore := 0.0
		for _, commit := range recent {
			totalScore += commit.KPIScore
		}
		avgScore := totalScore / float64(len(recent))

		fmt.Printf("Commits: %d\n", len(recent))
		fmt.Printf("Average Score: %.1f/100\n", avgScore)
		fmt.Println()
	}

	fmt.Println("💡 TIP: Use these insights when running the self-improvement workflow")
	fmt.Println("   ./hector --config hector-dev.yaml --workflow self-improvement")
}

func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

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
