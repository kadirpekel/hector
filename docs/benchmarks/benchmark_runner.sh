#!/bin/bash
# Structured Output Features Benchmark Runner
# Tests all feature combinations across scenarios and providers

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Path to hector binary
HECTOR_BIN="../hector"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="$RESULTS_DIR/run_$TIMESTAMP"

# Create results directory
mkdir -p "$RUN_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Structured Output Features Benchmark${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Run ID: $TIMESTAMP"
echo "Results will be saved to: $RUN_DIR"
echo ""

# Parse command line arguments
PROVIDER="${1:-openai}"  # Default to OpenAI
SCENARIO="${2:-all}"     # Default to all scenarios

echo -e "${YELLOW}Testing with provider: $PROVIDER${NC}"
echo -e "${YELLOW}Testing scenarios: $SCENARIO${NC}"
echo ""

# Configurations to test
CONFIGS=(
    "baseline-$PROVIDER:Baseline (No Features)"
    "reflection-only-$PROVIDER:Reflection Only"
    "completion-only-$PROVIDER:Completion Only"
    "all-features-$PROVIDER:All Features"
)

# Add supervisor config if testing OpenAI
if [ "$PROVIDER" = "openai" ]; then
    CONFIGS+=("supervisor-$PROVIDER:Supervisor (Goals)")
fi

# Test scenarios
SCENARIOS=(
    "simple_math:Calculate 157 * 89 and tell me the result."
    "multi_step:Calculate (25 * 4) + (100 / 5), then multiply the result by 3. Show each step."
    "error_recovery:Calculate 50 / 0, then if that fails, calculate 50 / 5 instead."
    "complex_multi_step:Do the following: 1) Calculate 100 + 200, 2) Multiply the result by 3, 3) Divide by 2, 4) Add 50. Report all intermediate results."
    "incomplete_prone:Calculate three things: A) 25 * 4, B) 100 / 5, and C) the sum of A and B. Make sure to report all three results separately."
)

# Function to run a single test
run_test() {
    local config_name="$1"
    local config_label="$2"
    local scenario_name="$3"
    local scenario_prompt="$4"
    
    echo -e "${GREEN}Testing: $config_label - $scenario_name${NC}"
    
    local result_file="$RUN_DIR/${config_name}_${scenario_name}.txt"
    local metrics_file="$RUN_DIR/${config_name}_${scenario_name}_metrics.json"
    
    # Start server in background
    echo "  Starting server..."
    $HECTOR_BIN serve --config "configs/${config_name}.yaml" > "$RUN_DIR/${config_name}_${scenario_name}_server.log" 2>&1 &
    local server_pid=$!
    
    # Wait for server to start
    sleep 3
    
    # Run test and capture output with timing
    echo "  Running test..."
    local start_time=$(date +%s.%N)
    
    if $HECTOR_BIN call test_agent "$scenario_prompt" > "$result_file" 2>&1; then
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        
        # Extract metrics from output
        local iterations=$(grep -o "Iteration [0-9]*" "$result_file" | tail -1 | awk '{print $2}' || echo "0")
        local tokens=$(grep -o "Tokens: [0-9]*" "$result_file" | tail -1 | awk '{print $2}' || echo "0")
        
        # Create metrics JSON
        cat > "$metrics_file" <<EOF
{
    "config": "$config_label",
    "scenario": "$scenario_name",
    "status": "success",
    "duration_seconds": $duration,
    "iterations": $iterations,
    "tokens": $tokens,
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
        
        echo -e "  ${GREEN}✓ Success${NC} (${duration}s, ${iterations} iterations, ${tokens} tokens)"
    else
        echo -e "  ${RED}✗ Failed${NC}"
        cat > "$metrics_file" <<EOF
{
    "config": "$config_label",
    "scenario": "$scenario_name",
    "status": "failed",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    fi
    
    # Stop server
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    sleep 1
    
    echo ""
}

# Main test loop
echo -e "${BLUE}Starting benchmark tests...${NC}"
echo ""

for config in "${CONFIGS[@]}"; do
    IFS=':' read -r config_name config_label <<< "$config"
    
    echo -e "${YELLOW}Configuration: $config_label${NC}"
    echo "---"
    
    for scenario in "${SCENARIOS[@]}"; do
        IFS=':' read -r scenario_name scenario_prompt <<< "$scenario"
        run_test "$config_name" "$config_label" "$scenario_name" "$scenario_prompt"
    done
    
    echo ""
done

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Benchmark complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Results saved to: $RUN_DIR"
echo ""
echo "To analyze results, run:"
echo "  python3 testing-lab/analyze_results.py $RUN_DIR"
echo ""

