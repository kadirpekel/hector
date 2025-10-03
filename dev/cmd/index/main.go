// Code indexing tool for Hector self-development
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kadirpekel/hector/dev"
)

func main() {
	// Parse flags
	outputFile := flag.String("output", "", "Output JSON file (optional)")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	// Get project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║              HECTOR CODE INDEXER                          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Directories to index
	directories := []string{
		"agent",
		"config",
		"context",
		"databases",
		"embedders",
		"llms",
		"reasoning",
		"tools",
		"workflow",
		"team",
		"component",
		"dev",
		"cmd/hector",
	}

	// Create indexer
	indexer := dev.NewCodeIndexer(projectRoot)
	indexer.Verbose = *verbose

	// Index code
	fmt.Println("🔍 Indexing Go codebase...")
	result, err := indexer.Index(directories)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Indexing failed: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println(result.FormatSummary())

	// Save to file if requested
	if *outputFile != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to marshal results: %v\n", err)
		} else if err := os.WriteFile(*outputFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to save results: %v\n", err)
		} else {
			fmt.Printf("✅ Results saved to %s\n\n", *outputFile)
		}
	}

	// Show sample symbols
	if len(result.Symbols) > 0 {
		fmt.Println("📝 SAMPLE INDEXED SYMBOLS:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		count := 5
		if len(result.Symbols) < count {
			count = len(result.Symbols)
		}

		for i := 0; i < count; i++ {
			symbol := result.Symbols[i]
			fmt.Printf("\n%d. [%s] %s.%s\n", i+1, symbol.Type, symbol.Package, symbol.Name)
			fmt.Printf("   File: %s:%d\n", filepath.Base(symbol.File), symbol.Line)
			if symbol.Signature != "" {
				fmt.Printf("   Signature: %s\n", symbol.Signature)
			}
			if symbol.Doc != "" {
				doc := symbol.Doc
				if len(doc) > 80 {
					doc = doc[:80] + "..."
				}
				fmt.Printf("   Doc: %s\n", doc)
			}
		}
		fmt.Println()
	}

	fmt.Println("✅ Indexing complete!")
	fmt.Println("\n💡 Next steps:")
	fmt.Println("  • Use this data to populate your vector database")
	fmt.Println("  • Enable semantic code search in agents")
	fmt.Println("  • Run: ./hector --config hector-dev.yaml --workflow self-improvement")
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
