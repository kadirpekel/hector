#!/bin/bash

# Run All Provider Benchmarks
# This script runs performance and behavioral benchmarks for all LLM providers

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║    Hector Structured Output Features - Full Benchmark    ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check for required environment variables
check_env_var() {
    local var_name=$1
    if [ -z "${!var_name}" ]; then
        echo "⚠️  Warning: $var_name not set. Skipping $(echo $var_name | cut -d_ -f1 | tr '[:upper:]' '[:lower:]') provider."
        return 1
    fi
    return 0
}

PROVIDERS=()
if check_env_var "OPENAI_API_KEY"; then
    PROVIDERS+=("openai")
fi
if check_env_var "ANTHROPIC_API_KEY"; then
    PROVIDERS+=("anthropic")
fi
if check_env_var "GEMINI_API_KEY"; then
    PROVIDERS+=("gemini")
fi

if [ ${#PROVIDERS[@]} -eq 0 ]; then
    echo "❌ Error: No API keys found. Please set at least one of:"
    echo "   - OPENAI_API_KEY"
    echo "   - ANTHROPIC_API_KEY"
    echo "   - GEMINI_API_KEY"
    exit 1
fi

echo "🎯 Testing Providers: ${PROVIDERS[*]}"
echo ""

# Create results directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="results/full_run_${TIMESTAMP}"
mkdir -p "$RESULTS_DIR"

echo "📁 Results will be saved to: $RESULTS_DIR"
echo ""

# Run benchmarks for each provider
for PROVIDER in "${PROVIDERS[@]}"; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🚀 Running benchmarks for: $(echo $PROVIDER | tr '[:lower:]' '[:upper:]')"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Performance benchmarks
    echo "📊 [1/2] Performance Benchmarks..."
    ./benchmark_runner.sh "$PROVIDER" "$RESULTS_DIR/${PROVIDER}_performance"
    
    # Behavioral benchmarks
    echo ""
    echo "🎭 [2/2] Behavioral Benchmarks..."
    python3 behavioral_benchmark.py "$PROVIDER" "$RESULTS_DIR/${PROVIDER}_behavioral"
    
    echo ""
    echo "✅ Completed $PROVIDER benchmarks"
    echo ""
done

# Generate cross-provider comparison report
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📈 Generating Cross-Provider Analysis..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

python3 compare_providers.py "$RESULTS_DIR"

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                  🎉 ALL BENCHMARKS COMPLETE               ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "📂 Results Location: $RESULTS_DIR"
echo ""
echo "📄 Available Reports:"
echo "   - ${RESULTS_DIR}/cross_provider_comparison.md"
echo "   - ${RESULTS_DIR}/executive_summary.md"
for PROVIDER in "${PROVIDERS[@]}"; do
    echo "   - ${RESULTS_DIR}/${PROVIDER}_performance/summary.json"
    echo "   - ${RESULTS_DIR}/${PROVIDER}_behavioral/summary.json"
done
echo ""
echo "🔍 Next Steps:"
echo "   1. Review cross_provider_comparison.md for insights"
echo "   2. Review executive_summary.md for recommendations"
echo "   3. Make deployment decisions based on findings"
echo ""

