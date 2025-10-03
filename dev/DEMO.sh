#!/bin/bash
# Demo script for Hector self-development system

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║     HECTOR SELF-DEVELOPMENT SYSTEM - DEMO                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Please run this script from the Hector project root"
    exit 1
fi

# Check for API key
if [ -z "$OPENAI_API_KEY" ] && [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "⚠️  No API key found!"
    echo "   Please set OPENAI_API_KEY or ANTHROPIC_API_KEY"
    echo ""
    echo "   Example: export OPENAI_API_KEY='your-key-here'"
    exit 1
fi

echo "🎯 DEMO OVERVIEW"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "This demo will show you the Hector self-development system:"
echo "  1. Run comprehensive benchmarks"
echo "  2. View development memory (if any past commits exist)"
echo "  3. Show how to run the self-improvement workflow"
echo ""
read -p "Press Enter to continue..."

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 1: Running Comprehensive Benchmarks"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 This will measure:"
echo "   • Functional Quality (tests, coverage)"
echo "   • Performance (speed, memory)"
echo "   • Efficiency (token usage)"
echo "   • Code Quality (linting, complexity)"
echo ""
read -p "Press Enter to run benchmarks..."

# Run benchmarks
echo ""
go run dev/cmd/benchmark/main.go --output kpis-baseline.json

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 2: Development Memory & Learnings"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🧠 Analyzing past improvements from git history..."
echo ""
read -p "Press Enter to view learnings..."

# View memory
echo ""
go run dev/cmd/memory/main.go --commits 20

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STEP 3: Self-Improvement Workflow"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🤖 The self-improvement workflow runs 6 specialized agents:"
echo ""
echo "   1. Code Analyzer - Finds improvement opportunities"
echo "   2. Architect - Designs solution"
echo "   3. Implementer - Writes code"
echo "   4. Tester - Runs tests & benchmarks"
echo "   5. Reviewer - Quality gate"
echo "   6. Git Manager - Commits with KPI tracking"
echo ""
echo "To run the self-improvement workflow:"
echo ""
echo "  echo \"Improve token efficiency in prompt building\" | \\"
echo "    ./hector --config hector-dev.yaml --workflow self-improvement"
echo ""
echo "⚠️  Note: This is a live demo script, so we won't actually run the"
echo "   full workflow now (it takes 5-10 minutes and modifies code)."
echo ""
echo "When you DO run it, it will:"
echo "  ✅ Analyze your codebase"
echo "  ✅ Propose specific improvements"
echo "  ✅ Implement changes"
echo "  ✅ Test & benchmark"
echo "  ✅ Commit to dev/* branch with full KPI data"
echo "  ✅ Wait for your review before merging"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BONUS: KPI Comparison"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "After the workflow makes changes, compare KPIs:"
echo ""
echo "  # Run benchmarks again"
echo "  go run dev/cmd/benchmark/main.go --output kpis-after.json"
echo ""
echo "  # Compare before & after"
echo "  go run dev/cmd/compare/main.go \\"
echo "    --before kpis-baseline.json \\"
echo "    --after kpis-after.json"
echo ""
echo "This will show:"
echo "  • Improvements in each metric"
echo "  • Overall improvement score"
echo "  • Whether the change is significant"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ DEMO COMPLETE!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📚 Learn more:"
echo "   • Read: dev/README.md"
echo "   • Configuration: hector-dev.yaml"
echo "   • Examples: dev/cmd/"
echo ""
echo "🚀 Ready to improve Hector?"
echo "   Just run the self-improvement workflow with your improvement goal!"
echo ""
echo "Example categories to try:"
echo "   • Performance: \"Reduce average response time\""
echo "   • Efficiency: \"Optimize token usage in prompts\""
echo "   • Reasoning: \"Improve chain-of-thought clarity\""
echo "   • Quality: \"Reduce code complexity\""
echo "   • Architecture: \"Refactor agent services\""
echo ""
echo "Happy self-improving! 🎉"

