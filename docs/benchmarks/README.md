# Benchmarking & Testing

Tools for testing and optimizing Hector agent configurations.

---

## Quick Start

### Prerequisites

```bash
# Set API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GEMINI_API_KEY="AIza..."

# Build Hector
make build
```

### Run Benchmarks

```bash
cd docs/benchmarks

# Test single provider
./benchmark_runner.sh openai

# Test all providers
./run_all_benchmarks.sh

# Analyze results
python analyze_results.py results/*/summary.json
```

---

## Tools

### 1. Performance Benchmarking

**`benchmark_runner.sh <provider>`** - Test structured output features

Tests baseline vs. reflection vs. completion vs. all features.

**Expected results:**
- Reflection: +13% quality, +20% cost
- All features: +25% quality, +35% cost

### 2. Generic A/B Testing

**`generic_benchmark.py <test_definition.json>`** - Test any config changes

```json
{
  "test_name": "my_test",
  "variants": [
    {"name": "control", "config": "baseline.yaml"},
    {"name": "treatment", "config": "baseline.yaml", 
     "config_overrides": {"llms.main.temperature": 0.3}}
  ],
  "scenarios": [
    {"name": "test1", "prompt": "Calculate 2+2"}
  ]
}
```

Use cases:
- Prompt variations
- Temperature tuning
- Model comparison
- Any YAML parameter

**Examples:** See `tests/` directory

### 3. Prompt Quality Assessment

**`assess_prompts.py [results_dir]`** - Analyze prompt effectiveness

```bash
python assess_prompts.py results/
```

Output: Quality score (0-100), token efficiency, recommendations.

---

## Structure

```
docs/benchmarks/
├── README.md (this file)
│
├── Tools
│   ├── benchmark_runner.sh
│   ├── behavioral_benchmark.py
│   ├── generic_benchmark.py
│   ├── assess_prompts.py
│   ├── compare_providers.py
│   ├── analyze_results.py
│   └── run_all_benchmarks.sh
│
├── configs/        # Test agent configurations
├── tests/          # A/B test definitions
└── scenarios/      # Test scenarios
```

---

## Configuration

Test configurations in `configs/`:
- `baseline-<provider>.yaml` - No features
- `reflection-only-<provider>.yaml` - Structured reflection
- `completion-only-<provider>.yaml` - Completion verification
- `all-features-<provider>.yaml` - All features enabled

Supported providers: `openai`, `anthropic`, `gemini`

---

## Key Insights

**Smart Defaults (Current):**
- Structured reflection: **ENABLED** by default
- +13% quality, +20% cost
- Disable if needed: `enable_structured_reflection: false`

**Cost/Quality Tradeoffs:**
- Small model + structured output = Better than large model without
- Example: `gpt-4o-mini` + reflection > `gpt-4` baseline

**When to Use What:**
- **Baseline:** High-volume, cost-sensitive
- **Reflection:** General use (default)
- **Completion:** Multi-step workflows
- **All features:** Critical tasks

---

## Tips

1. **Start simple:** Test 2-3 variants with 3-5 scenarios
2. **One change at a time:** Isolate variables for clear results
3. **Run multiple iterations:** 3-5 runs for statistical significance
4. **Cost-optimize:** Use smaller models with structured features

---

## Troubleshooting

**Server won't start:**
- Check if port 8080 is free
- Verify API keys are set
- Check config syntax

**Tests timing out:**
- Increase timeout in scripts
- Check agent max_iterations setting

**Low quality scores:**
- Review prompts for clarity
- Check if scenarios are representative
- Consider enabling structured features

---

For configuration details, see [docs/CONFIGURATION.md](../CONFIGURATION.md).
