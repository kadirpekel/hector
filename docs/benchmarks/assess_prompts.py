#!/usr/bin/env python3
"""
Prompt Quality Assessment Script

Analyzes benchmark results to evaluate the quality and efficiency of default prompts.
Extracts metrics to determine if prompts are performing well or need tuning.

Usage:
    python assess_prompts.py [results_dir]
    
Example:
    python assess_prompts.py results/
"""

import json
import os
import sys
from pathlib import Path
from collections import defaultdict
from dataclasses import dataclass
from typing import Dict, List, Optional


@dataclass
class PromptMetrics:
    """Metrics for evaluating prompt quality"""
    avg_tokens_per_task: float
    avg_iterations: float
    success_rate: float
    token_efficiency: float  # tokens per successful task
    avg_tokens_per_iteration: float
    failure_patterns: List[str]
    cost_per_success: float  # estimated cost per successful task


class PromptAssessor:
    """Analyzes prompt performance from benchmark results"""
    
    # Cost per 1M tokens (input + output blended estimate)
    COST_PER_1M_TOKENS = {
        'openai': {'gpt-4o': 5.0, 'gpt-4o-mini': 0.5},
        'anthropic': {'claude-3-5-sonnet': 7.5, 'claude-3-haiku': 1.0},
        'gemini': {'gemini-2.0-flash': 0.3, 'gemini-1.5-pro': 2.5}
    }
    
    def __init__(self, results_dir: str):
        self.results_dir = Path(results_dir)
        self.all_results = []
        
    def load_results(self) -> bool:
        """Load all result JSON files from results directory"""
        if not self.results_dir.exists():
            print(f"❌ Results directory not found: {self.results_dir}")
            return False
        
        for provider_dir in self.results_dir.iterdir():
            if not provider_dir.is_dir():
                continue
                
            for run_dir in provider_dir.iterdir():
                if not run_dir.is_dir() or not run_dir.name.startswith('run_'):
                    continue
                
                result_files = list(run_dir.glob('*.json'))
                for result_file in result_files:
                    if result_file.name == 'summary.json':
                        continue
                    
                    try:
                        with open(result_file) as f:
                            result = json.load(f)
                            result['provider'] = provider_dir.name
                            result['run'] = run_dir.name
                            self.all_results.append(result)
                    except Exception as e:
                        print(f"⚠️  Failed to load {result_file}: {e}")
        
        print(f"✅ Loaded {len(self.all_results)} results from {self.results_dir}")
        return len(self.all_results) > 0
    
    def calculate_metrics(self, config_name: str, provider: Optional[str] = None) -> PromptMetrics:
        """Calculate prompt metrics for a specific configuration"""
        # Filter results
        filtered = [
            r for r in self.all_results 
            if r['config'] == config_name and (provider is None or r['provider'] == provider)
        ]
        
        if not filtered:
            return PromptMetrics(0, 0, 0, 0, 0, [], 0)
        
        # Calculate metrics
        total_tokens = sum(r.get('total_tokens', 0) for r in filtered)
        total_iterations = sum(r.get('iterations', 0) for r in filtered)
        successes = sum(1 for r in filtered if r.get('success', False))
        
        avg_tokens = total_tokens / len(filtered) if filtered else 0
        avg_iterations = total_iterations / len(filtered) if filtered else 0
        success_rate = successes / len(filtered) if filtered else 0
        token_efficiency = total_tokens / successes if successes > 0 else float('inf')
        avg_tokens_per_iter = avg_tokens / avg_iterations if avg_iterations > 0 else 0
        
        # Identify failure patterns
        failures = [r for r in filtered if not r.get('success', False)]
        failure_patterns = [r.get('scenario', 'unknown') for r in failures]
        
        # Estimate cost (using blended average)
        avg_cost_per_token = 5.0 / 1_000_000  # Conservative estimate
        cost_per_success = (total_tokens / successes * avg_cost_per_token) if successes > 0 else 0
        
        return PromptMetrics(
            avg_tokens_per_task=avg_tokens,
            avg_iterations=avg_iterations,
            success_rate=success_rate,
            token_efficiency=token_efficiency,
            avg_tokens_per_iteration=avg_tokens_per_iter,
            failure_patterns=failure_patterns,
            cost_per_success=cost_per_success
        )
    
    def assess_baseline_quality(self) -> Dict[str, any]:
        """Assess quality of baseline (default) prompts"""
        print("\n" + "="*80)
        print("PROMPT QUALITY ASSESSMENT: BASELINE (DEFAULT PROMPTS)")
        print("="*80 + "\n")
        
        baseline_metrics = self.calculate_metrics('baseline-openai')
        
        # Quality thresholds (based on empirical data)
        QUALITY_THRESHOLDS = {
            'success_rate': 0.90,          # 90%+ is good
            'avg_tokens_per_task': 5000,   # Under 5k tokens is efficient
            'avg_iterations': 5,           # Under 5 iterations is focused
            'token_efficiency': 5500,      # Under 5.5k tokens per success
        }
        
        assessment = {
            'metrics': baseline_metrics,
            'quality_score': 0,
            'issues': [],
            'recommendations': []
        }
        
        # Assess success rate
        if baseline_metrics.success_rate >= QUALITY_THRESHOLDS['success_rate']:
            assessment['quality_score'] += 25
            print(f"✅ Success Rate: {baseline_metrics.success_rate:.1%} (Excellent)")
        elif baseline_metrics.success_rate >= 0.80:
            assessment['quality_score'] += 20
            print(f"✅ Success Rate: {baseline_metrics.success_rate:.1%} (Good)")
        else:
            print(f"⚠️  Success Rate: {baseline_metrics.success_rate:.1%} (Needs Improvement)")
            assessment['issues'].append(f"Low success rate ({baseline_metrics.success_rate:.1%})")
            assessment['recommendations'].append("Review failure patterns and add error handling guidance to prompts")
        
        # Assess token efficiency
        if baseline_metrics.avg_tokens_per_task <= QUALITY_THRESHOLDS['avg_tokens_per_task']:
            assessment['quality_score'] += 25
            print(f"✅ Token Usage: {baseline_metrics.avg_tokens_per_task:.0f} tokens/task (Efficient)")
        else:
            print(f"⚠️  Token Usage: {baseline_metrics.avg_tokens_per_task:.0f} tokens/task (High)")
            assessment['issues'].append(f"High token usage ({baseline_metrics.avg_tokens_per_task:.0f} tokens/task)")
            assessment['recommendations'].append("Consider more concise reasoning instructions or reduce max iterations")
        
        # Assess iteration count
        if baseline_metrics.avg_iterations <= QUALITY_THRESHOLDS['avg_iterations']:
            assessment['quality_score'] += 25
            print(f"✅ Iterations: {baseline_metrics.avg_iterations:.1f} avg (Focused)")
        else:
            print(f"⚠️  Iterations: {baseline_metrics.avg_iterations:.1f} avg (High)")
            assessment['issues'].append(f"High iteration count ({baseline_metrics.avg_iterations:.1f})")
            assessment['recommendations'].append("Add clearer task completion criteria to prompts")
        
        # Assess overall token efficiency
        if baseline_metrics.token_efficiency <= QUALITY_THRESHOLDS['token_efficiency']:
            assessment['quality_score'] += 25
            print(f"✅ Efficiency: {baseline_metrics.token_efficiency:.0f} tokens/success (Good)")
        else:
            print(f"⚠️  Efficiency: {baseline_metrics.token_efficiency:.0f} tokens/success (Suboptimal)")
            assessment['issues'].append(f"Suboptimal token efficiency ({baseline_metrics.token_efficiency:.0f})")
            assessment['recommendations'].append("Refine prompts to be more direct and reduce redundancy")
        
        print(f"\n📊 Overall Quality Score: {assessment['quality_score']}/100")
        
        if assessment['quality_score'] >= 90:
            print("🎉 Prompts are performing excellently!")
        elif assessment['quality_score'] >= 75:
            print("✅ Prompts are performing well.")
        elif assessment['quality_score'] >= 60:
            print("⚠️  Prompts are adequate but have room for improvement.")
        else:
            print("❌ Prompts need significant tuning.")
        
        # Failure analysis
        if baseline_metrics.failure_patterns:
            print(f"\n⚠️  Failure Patterns ({len(baseline_metrics.failure_patterns)} failures):")
            failure_counts = defaultdict(int)
            for pattern in baseline_metrics.failure_patterns:
                failure_counts[pattern] += 1
            
            for scenario, count in sorted(failure_counts.items(), key=lambda x: -x[1]):
                print(f"   - {scenario}: {count} failure(s)")
                assessment['recommendations'].append(f"Improve prompt handling for '{scenario}' scenario")
        
        return assessment
    
    def compare_strategies(self):
        """Compare prompt effectiveness across reasoning strategies"""
        print("\n" + "="*80)
        print("STRATEGY COMPARISON")
        print("="*80 + "\n")
        
        strategies = ['baseline', 'reflection-only', 'completion-only', 'all-features']
        
        print(f"{'Strategy':<20} {'Tokens':<12} {'Iterations':<12} {'Success':<10} {'Cost/Success':<12}")
        print("-" * 80)
        
        for strategy in strategies:
            config_name = f"{strategy}-openai"
            metrics = self.calculate_metrics(config_name)
            
            if metrics.avg_tokens_per_task > 0:
                print(f"{strategy:<20} {metrics.avg_tokens_per_task:<12.0f} "
                      f"{metrics.avg_iterations:<12.1f} {metrics.success_rate:<10.1%} "
                      f"${metrics.cost_per_success:<11.4f}")
        
        print("\n💡 Insight: Compare baseline metrics to enhanced strategies.")
        print("   If baseline is competitive, prompts are already well-tuned.")
        print("   If enhancements show major gains, prompts may need refinement.")
    
    def provider_comparison(self):
        """Compare prompt performance across LLM providers"""
        print("\n" + "="*80)
        print("CROSS-PROVIDER PROMPT ANALYSIS")
        print("="*80 + "\n")
        
        providers = ['openai', 'anthropic', 'gemini']
        
        print(f"{'Provider':<15} {'Tokens':<12} {'Iterations':<12} {'Success':<10} {'Efficiency':<12}")
        print("-" * 80)
        
        for provider in providers:
            metrics = self.calculate_metrics('baseline-openai', provider=provider)
            
            if metrics.avg_tokens_per_task > 0:
                print(f"{provider:<15} {metrics.avg_tokens_per_task:<12.0f} "
                      f"{metrics.avg_iterations:<12.1f} {metrics.success_rate:<10.1%} "
                      f"{metrics.token_efficiency:<12.0f}")
        
        print("\n💡 Insight: If metrics vary significantly across providers,")
        print("   prompts may need provider-specific tuning (e.g., Claude prefill).")
    
    def generate_recommendations(self, assessment: Dict) -> List[str]:
        """Generate actionable recommendations"""
        recommendations = assessment.get('recommendations', [])
        
        # Add general recommendations
        if assessment['quality_score'] < 90:
            recommendations.append("Run A/B tests with prompt variations to identify improvements")
            recommendations.append("Use the generic testing framework to benchmark prompt changes")
        
        return list(set(recommendations))  # Deduplicate
    
    def export_report(self, assessment: Dict):
        """Export detailed assessment report"""
        report_path = self.results_dir / 'prompt_assessment_report.json'
        
        with open(report_path, 'w') as f:
            json.dump({
                'quality_score': assessment['quality_score'],
                'issues': assessment['issues'],
                'recommendations': self.generate_recommendations(assessment),
                'metrics': {
                    'avg_tokens_per_task': assessment['metrics'].avg_tokens_per_task,
                    'avg_iterations': assessment['metrics'].avg_iterations,
                    'success_rate': assessment['metrics'].success_rate,
                    'token_efficiency': assessment['metrics'].token_efficiency,
                    'cost_per_success': assessment['metrics'].cost_per_success,
                    'failure_count': len(assessment['metrics'].failure_patterns)
                }
            }, f, indent=2)
        
        print(f"\n📄 Detailed report exported: {report_path}")
    
    def run_assessment(self):
        """Run complete prompt assessment"""
        if not self.load_results():
            return
        
        # Main assessment
        assessment = self.assess_baseline_quality()
        
        # Comparisons
        self.compare_strategies()
        self.provider_comparison()
        
        # Recommendations
        print("\n" + "="*80)
        print("RECOMMENDATIONS")
        print("="*80 + "\n")
        
        recommendations = self.generate_recommendations(assessment)
        for i, rec in enumerate(recommendations, 1):
            print(f"{i}. {rec}")
        
        # Export
        self.export_report(assessment)
        
        # Summary
        print("\n" + "="*80)
        print("SUMMARY")
        print("="*80 + "\n")
        
        if assessment['quality_score'] >= 75:
            print("✅ **Current default prompts are performing well.**")
            print("   No immediate tuning required, but continuous monitoring recommended.")
        else:
            print("⚠️  **Current default prompts need improvement.**")
            print("   Implement recommendations and re-run assessment.")
        
        print(f"\n📊 Quality Score: {assessment['quality_score']}/100")
        print(f"📋 Issues Found: {len(assessment['issues'])}")
        print(f"💡 Recommendations: {len(recommendations)}")
        
        print("\n🔬 Next Steps:")
        print("   1. Review recommendations above")
        print("   2. Use generic framework to test prompt variations")
        print("   3. Re-run assessment after changes")
        print("   4. See docs/benchmarks/AB_TESTING.md for A/B testing guide")


def main():
    if len(sys.argv) > 1:
        results_dir = sys.argv[1]
    else:
        # Default to results directory
        script_dir = Path(__file__).parent
        results_dir = script_dir / 'results'
    
    assessor = PromptAssessor(results_dir)
    assessor.run_assessment()


if __name__ == '__main__':
    main()

