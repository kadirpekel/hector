#!/usr/bin/env python3
"""
Structured Output Features Benchmark Results Analyzer
Analyzes benchmark results and generates comparison reports
"""

import json
import os
import sys
from pathlib import Path
from collections import defaultdict
from datetime import datetime

def load_metrics(results_dir):
    """Load all metrics files from results directory"""
    metrics = []
    for file in Path(results_dir).glob("*_metrics.json"):
        try:
            with open(file, 'r') as f:
                data = json.load(f)
                metrics.append(data)
        except Exception as e:
            print(f"Warning: Failed to load {file}: {e}")
    return metrics

def calculate_stats(metrics_list):
    """Calculate statistics for a list of metrics"""
    if not metrics_list:
        return None
    
    durations = [m['duration_seconds'] for m in metrics_list if 'duration_seconds' in m]
    iterations = [m['iterations'] for m in metrics_list if 'iterations' in m]
    tokens = [m['tokens'] for m in metrics_list if 'tokens' in m]
    
    return {
        'count': len(metrics_list),
        'avg_duration': sum(durations) / len(durations) if durations else 0,
        'avg_iterations': sum(iterations) / len(iterations) if iterations else 0,
        'avg_tokens': sum(tokens) / len(tokens) if tokens else 0,
        'total_tokens': sum(tokens),
    }

def generate_report(results_dir):
    """Generate comprehensive benchmark report"""
    metrics = load_metrics(results_dir)
    
    if not metrics:
        print("No metrics found in results directory")
        return
    
    print("=" * 80)
    print(" Structured Output Features Benchmark Results")
    print("=" * 80)
    print()
    
    # Group by configuration
    by_config = defaultdict(list)
    by_scenario = defaultdict(list)
    
    for m in metrics:
        if m.get('status') == 'success':
            by_config[m['config']].append(m)
            by_scenario[m['scenario']].append(m)
    
    # Configuration comparison
    print("📊 PERFORMANCE BY CONFIGURATION")
    print("-" * 80)
    print(f"{'Configuration':<30} {'Tests':<8} {'Avg Time':<12} {'Avg Iter':<10} {'Avg Tokens':<12} {'Total Tokens':<12}")
    print("-" * 80)
    
    baseline_tokens = 0
    config_stats = {}
    
    for config, config_metrics in sorted(by_config.items()):
        stats = calculate_stats(config_metrics)
        config_stats[config] = stats
        
        if "Baseline" in config:
            baseline_tokens = stats['avg_tokens']
        
        print(f"{config:<30} {stats['count']:<8} {stats['avg_duration']:<12.2f}s {stats['avg_iterations']:<10.1f} {stats['avg_tokens']:<12.0f} {stats['total_tokens']:<12.0f}")
    
    print()
    
    # Cost analysis
    if baseline_tokens > 0:
        print("💰 COST ANALYSIS (vs Baseline)")
        print("-" * 80)
        print(f"{'Configuration':<30} {'Extra Tokens':<15} {'Extra Cost/Task*':<20} {'% Increase':<12}")
        print("-" * 80)
        
        for config, stats in sorted(config_stats.items()):
            extra_tokens = stats['avg_tokens'] - baseline_tokens
            extra_cost_gpt4 = (extra_tokens / 1_000_000) * 30  # $30/1M tokens
            pct_increase = ((stats['avg_tokens'] / baseline_tokens) - 1) * 100 if baseline_tokens > 0 else 0
            
            print(f"{config:<30} {extra_tokens:>+14.0f} ${extra_cost_gpt4:>18.4f} {pct_increase:>11.1f}%")
        
        print()
        print("* Cost estimated at $30/1M tokens (GPT-4 input pricing)")
        print()
    
    # Scenario comparison
    print("🎯 PERFORMANCE BY SCENARIO")
    print("-" * 80)
    print(f"{'Scenario':<25} {'Tests':<8} {'Avg Time':<12} {'Avg Iter':<10} {'Avg Tokens':<12}")
    print("-" * 80)
    
    for scenario, scenario_metrics in sorted(by_scenario.items()):
        stats = calculate_stats(scenario_metrics)
        print(f"{scenario:<25} {stats['count']:<8} {stats['avg_duration']:<12.2f}s {stats['avg_iterations']:<10.1f} {stats['avg_tokens']:<12.0f}")
    
    print()
    
    # Feature impact matrix
    print("🔬 FEATURE IMPACT MATRIX")
    print("-" * 80)
    
    # Calculate feature impact
    baseline_stats = config_stats.get('Baseline (No Features)')
    reflection_stats = config_stats.get('Reflection Only')
    completion_stats = config_stats.get('Completion Only')
    all_features_stats = config_stats.get('All Features')
    
    if baseline_stats:
        print(f"Baseline Performance:")
        print(f"  - Avg Duration: {baseline_stats['avg_duration']:.2f}s")
        print(f"  - Avg Tokens: {baseline_stats['avg_tokens']:.0f}")
        print()
        
        if reflection_stats:
            reflection_overhead = reflection_stats['avg_tokens'] - baseline_stats['avg_tokens']
            print(f"Structured Reflection Impact:")
            print(f"  - Token Overhead: +{reflection_overhead:.0f} tokens ({(reflection_overhead/baseline_stats['avg_tokens']*100):.1f}%)")
            print(f"  - Time Impact: +{(reflection_stats['avg_duration'] - baseline_stats['avg_duration']):.2f}s")
            print()
        
        if completion_stats:
            completion_overhead = completion_stats['avg_tokens'] - baseline_stats['avg_tokens']
            print(f"Completion Verification Impact:")
            print(f"  - Token Overhead: +{completion_overhead:.0f} tokens ({(completion_overhead/baseline_stats['avg_tokens']*100):.1f}%)")
            print(f"  - Time Impact: +{(completion_stats['avg_duration'] - baseline_stats['avg_duration']):.2f}s")
            print()
        
        if all_features_stats:
            all_overhead = all_features_stats['avg_tokens'] - baseline_stats['avg_tokens']
            print(f"All Features Combined Impact:")
            print(f"  - Token Overhead: +{all_overhead:.0f} tokens ({(all_overhead/baseline_stats['avg_tokens']*100):.1f}%)")
            print(f"  - Time Impact: +{(all_features_stats['avg_duration'] - baseline_stats['avg_duration']):.2f}s")
            print()
    
    # Success rate
    print("✅ SUCCESS RATE")
    print("-" * 80)
    total_tests = len(metrics)
    successful_tests = len([m for m in metrics if m.get('status') == 'success'])
    print(f"Total Tests: {total_tests}")
    print(f"Successful: {successful_tests}")
    print(f"Success Rate: {(successful_tests/total_tests*100):.1f}%")
    print()
    
    # Recommendations
    print("💡 RECOMMENDATIONS")
    print("-" * 80)
    
    if baseline_stats and reflection_stats:
        reflection_overhead_pct = (reflection_stats['avg_tokens'] - baseline_stats['avg_tokens']) / baseline_stats['avg_tokens'] * 100
        if reflection_overhead_pct < 20:
            print("✓ Structured Reflection: LOW overhead (<20%), recommended for production")
        elif reflection_overhead_pct < 40:
            print("⚠ Structured Reflection: MODERATE overhead (20-40%), use for critical tasks")
        else:
            print("⚠ Structured Reflection: HIGH overhead (>40%), use selectively")
    
    if baseline_stats and completion_stats:
        completion_overhead_pct = (completion_stats['avg_tokens'] - baseline_stats['avg_tokens']) / baseline_stats['avg_tokens'] * 100
        if completion_overhead_pct < 15:
            print("✓ Completion Verification: LOW overhead (<15%), good for multi-step tasks")
        elif completion_overhead_pct < 30:
            print("⚠ Completion Verification: MODERATE overhead (15-30%), use for important tasks")
        else:
            print("⚠ Completion Verification: HIGH overhead (>30%), use very selectively")
    
    if baseline_stats and all_features_stats:
        all_overhead_pct = (all_features_stats['avg_tokens'] - baseline_stats['avg_tokens']) / baseline_stats['avg_tokens'] * 100
        if all_overhead_pct < 40:
            print("✓ All Features: ACCEPTABLE overhead, recommended for development/debug")
        else:
            print("⚠ All Features: HIGH overhead, use only when quality is critical")
    
    print()
    print("=" * 80)
    print()
    
    # Save summary JSON
    summary = {
        'timestamp': datetime.now().isoformat(),
        'results_dir': str(results_dir),
        'total_tests': total_tests,
        'successful_tests': successful_tests,
        'configurations': {k: v for k, v in config_stats.items()},
        'baseline_tokens': baseline_tokens,
    }
    
    summary_file = Path(results_dir) / 'summary.json'
    with open(summary_file, 'w') as f:
        json.dump(summary, f, indent=2)
    
    print(f"Summary saved to: {summary_file}")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python3 analyze_results.py <results_directory>")
        sys.exit(1)
    
    results_dir = sys.argv[1]
    if not os.path.isdir(results_dir):
        print(f"Error: Directory not found: {results_dir}")
        sys.exit(1)
    
    generate_report(results_dir)

