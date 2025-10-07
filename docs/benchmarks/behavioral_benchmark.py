#!/usr/bin/env python3
"""
Behavioral and Functional Testing for Structured Output Features
Tests quality and correctness, not just performance
"""

import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path
from collections import defaultdict
from datetime import datetime

class BehavioralTester:
    def __init__(self, results_dir):
        self.results_dir = Path(results_dir)
        self.results_dir.mkdir(parents=True, exist_ok=True)
        
    def load_behavioral_tests(self):
        """Load behavioral test definitions"""
        with open('behavioral_tests.json', 'r') as f:
            data = json.load(f)
        return data['behavioral_tests']
    
    def start_server(self, config_path, timeout=5):
        """Start Hector server with given config"""
        cmd = ['../hector', 'serve', '--config', config_path]
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            cwd='.'
        )
        time.sleep(timeout)
        return proc
    
    def stop_server(self, proc):
        """Stop Hector server"""
        if proc:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
    
    def run_test(self, agent_name, prompt):
        """Execute a test and return output"""
        cmd = ['../hector', 'call', agent_name, prompt]
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=60,
                cwd='.'
            )
            return result.stdout, result.returncode == 0
        except subprocess.TimeoutExpired:
            return "TIMEOUT", False
        except Exception as e:
            return f"ERROR: {str(e)}", False
    
    def check_criterion(self, output, criterion):
        """Check if a success criterion is met"""
        pattern = criterion['check_pattern']
        try:
            # Case-insensitive search
            match = re.search(pattern, output, re.IGNORECASE | re.DOTALL)
            return match is not None
        except Exception as e:
            print(f"  Warning: Pattern error: {e}")
            return False
    
    def evaluate_test(self, test_def, output):
        """Evaluate a single test against success criteria"""
        results = {
            'test_id': test_def['id'],
            'test_name': test_def['name'],
            'criteria_results': [],
            'passed_criteria': 0,
            'total_criteria': len(test_def['success_criteria']),
            'passed': False,
            'output_length': len(output)
        }
        
        for criterion in test_def['success_criteria']:
            passed = self.check_criterion(output, criterion)
            results['criteria_results'].append({
                'criterion': criterion['criterion'],
                'description': criterion['description'],
                'passed': passed
            })
            if passed:
                results['passed_criteria'] += 1
        
        # Test passes if all criteria are met
        results['passed'] = results['passed_criteria'] == results['total_criteria']
        results['score'] = results['passed_criteria'] / results['total_criteria'] if results['total_criteria'] > 0 else 0
        
        return results
    
    def run_behavioral_suite(self, config_name, config_path, agent_name='test_agent_baseline'):
        """Run full behavioral test suite for a configuration"""
        print(f"\n{'='*80}")
        print(f"Testing Configuration: {config_name}")
        print(f"{'='*80}\n")
        
        tests = self.load_behavioral_tests()
        results = []
        
        # Start server
        print("Starting server...")
        server_proc = self.start_server(config_path)
        
        try:
            for test_def in tests:
                print(f"\n  Running: {test_def['name']}")
                print(f"  Feature: {test_def['tests_feature']}")
                print(f"  Prompt: {test_def['prompt'][:70]}...")
                
                # Run test
                output, success = self.run_test(agent_name, test_def['prompt'])
                
                if not success:
                    print(f"  ❌ Test execution failed")
                    results.append({
                        'test_id': test_def['id'],
                        'test_name': test_def['name'],
                        'execution_failed': True,
                        'passed': False,
                        'score': 0.0
                    })
                    continue
                
                # Evaluate
                evaluation = self.evaluate_test(test_def, output)
                results.append(evaluation)
                
                # Display results
                print(f"  Score: {evaluation['passed_criteria']}/{evaluation['total_criteria']} criteria met")
                for cr in evaluation['criteria_results']:
                    status = "✓" if cr['passed'] else "✗"
                    print(f"    {status} {cr['description']}")
                
                if evaluation['passed']:
                    print(f"  ✅ Test PASSED")
                else:
                    print(f"  ⚠️  Test PARTIAL ({evaluation['score']*100:.0f}%)")
                
                # Save detailed output
                output_file = self.results_dir / f"{config_name}_{test_def['id']}.txt"
                with open(output_file, 'w') as f:
                    f.write(output)
                
        finally:
            # Stop server
            print("\nStopping server...")
            self.stop_server(server_proc)
        
        return results
    
    def generate_comparison_report(self, all_results):
        """Generate comparison report across configurations"""
        print("\n" + "="*80)
        print(" BEHAVIORAL TESTING RESULTS - COMPARISON")
        print("="*80 + "\n")
        
        # Group by configuration
        by_config = {}
        for config_name, results in all_results.items():
            by_config[config_name] = results
        
        # Calculate summary stats
        print("📊 OVERALL SCORES BY CONFIGURATION")
        print("-"*80)
        print(f"{'Configuration':<30} {'Tests':<8} {'Passed':<8} {'Partial':<8} {'Failed':<8} {'Avg Score':<10}")
        print("-"*80)
        
        config_summaries = {}
        for config_name, results in by_config.items():
            total = len(results)
            passed = len([r for r in results if r.get('passed', False)])
            partial = len([r for r in results if not r.get('passed', False) and r.get('score', 0) > 0])
            failed = len([r for r in results if r.get('score', 0) == 0])
            avg_score = sum(r.get('score', 0) for r in results) / total if total > 0 else 0
            pass_rate = (passed / total * 100) if total > 0 else 0
            
            config_summaries[config_name] = {
                'tests_total': total,
                'tests_passed': passed,
                'tests_partial': partial,
                'tests_failed': failed,
                'avg_quality_score': avg_score * 100,  # As percentage
                'pass_rate': pass_rate
            }
            
            print(f"{config_name:<30} {total:<8} {passed:<8} {partial:<8} {failed:<8} {avg_score*100:<9.1f}%")
        
        print()
        
        # Feature-specific analysis
        print("🎯 FEATURE-SPECIFIC ANALYSIS")
        print("-"*80)
        
        # Load test definitions to get feature mappings
        tests = self.load_behavioral_tests()
        feature_map = {test['id']: test['tests_feature'] for test in tests}
        
        # Group results by feature
        by_feature = defaultdict(lambda: defaultdict(list))
        for config_name, results in by_config.items():
            for result in results:
                test_id = result['test_id']
                feature = feature_map.get(test_id, 'unknown')
                by_feature[feature][config_name].append(result.get('score', 0))
        
        for feature in ['reflection', 'completion', 'goals', 'all']:
            if feature not in by_feature:
                continue
            
            print(f"\n{feature.upper()} Feature Tests:")
            print(f"{'Configuration':<30} {'Tests':<8} {'Avg Score':<10}")
            print("-"*50)
            
            for config_name in ['Baseline', 'Reflection Only', 'Completion Only', 'All Features', 'Supervisor']:
                if config_name in by_feature[feature]:
                    scores = by_feature[feature][config_name]
                    avg = sum(scores) / len(scores) if scores else 0
                    print(f"{config_name:<30} {len(scores):<8} {avg*100:<9.1f}%")
        
        print()
        
        # Improvement analysis
        print("📈 IMPROVEMENT ANALYSIS (vs Baseline)")
        print("-"*80)
        
        baseline_summary = config_summaries.get('Baseline', {})
        baseline_score = baseline_summary.get('avg_score', 0)
        
        if baseline_score > 0:
            print(f"{'Configuration':<30} {'Score Improvement':<20} {'Pass Rate':<15}")
            print("-"*80)
            
            for config_name, summary in config_summaries.items():
                if config_name == 'Baseline':
                    continue
                
                score_improvement = summary['avg_score'] - baseline_score
                pass_rate = summary['passed'] / summary['total'] * 100 if summary['total'] > 0 else 0
                
                improvement_str = f"{score_improvement*100:+.1f}%"
                print(f"{config_name:<30} {improvement_str:<20} {pass_rate:<14.1f}%")
        
        print()
        
        # Recommendations
        print("💡 BEHAVIORAL RECOMMENDATIONS")
        print("-"*80)
        
        # Check reflection improvement
        reflection_config = config_summaries.get('Reflection Only', {})
        if reflection_config and baseline_summary:
            reflection_improvement = (reflection_config['avg_score'] - baseline_summary['avg_score']) * 100
            if reflection_improvement > 10:
                print(f"✅ Structured Reflection shows {reflection_improvement:.1f}% improvement")
                print("   Recommended for: Error recovery, iterative refinement, quality tasks")
            elif reflection_improvement > 5:
                print(f"⚠️  Structured Reflection shows modest {reflection_improvement:.1f}% improvement")
                print("   Consider for: Critical tasks where quality matters")
            else:
                print(f"⚠️  Structured Reflection shows minimal improvement ({reflection_improvement:.1f}%)")
                print("   May not justify cost for your use case")
        
        # Check completion improvement
        completion_config = config_summaries.get('Completion Only', {})
        if completion_config and baseline_summary:
            completion_improvement = (completion_config['avg_score'] - baseline_summary['avg_score']) * 100
            if completion_improvement > 10:
                print(f"\n✅ Completion Verification shows {completion_improvement:.1f}% improvement")
                print("   Recommended for: Multi-step tasks, complex workflows, reports")
            elif completion_improvement > 5:
                print(f"\n⚠️  Completion Verification shows modest {completion_improvement:.1f}% improvement")
                print("   Consider for: Tasks where completeness is critical")
            else:
                print(f"\n⚠️  Completion Verification shows minimal improvement ({completion_improvement:.1f}%)")
                print("   Baseline may be sufficient for your tasks")
        
        # Check if combined is better
        all_features_config = config_summaries.get('All Features', {})
        if all_features_config and baseline_summary:
            all_improvement = (all_features_config['avg_score'] - baseline_summary['avg_score']) * 100
            if all_improvement > 15:
                print(f"\n✅ All Features combined: {all_improvement:.1f}% improvement")
                print("   Strong synergy between features - recommended for quality-critical production")
            elif all_improvement > 10:
                print(f"\n⚠️  All Features combined: {all_improvement:.1f}% improvement")
                print("   Moderate synergy - use for important workflows")
        
        print()
        print("="*80)
        
        # Save summary with standardized structure for cross-provider comparison
        summary = {
            'timestamp': datetime.now().isoformat(),
            'configurations': {
                config_name.lower().replace(' ', '_'): config_data 
                for config_name, config_data in config_summaries.items()
            },
            'feature_analysis': dict(by_feature),
            'baseline_score': baseline_score
        }
        
        summary_file = self.results_dir / 'summary.json'
        with open(summary_file, 'w') as f:
            json.dump(summary, f, indent=2)
        
        print(f"\nSummary saved to: {summary_file}")

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 behavioral_benchmark.py <provider> [results_dir]")
        print("Example: python3 behavioral_benchmark.py openai")
        print("Example: python3 behavioral_benchmark.py openai results/custom_dir")
        sys.exit(1)
    
    provider = sys.argv[1]
    
    # Allow custom results directory
    if len(sys.argv) >= 3:
        results_dir = sys.argv[2]
    else:
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        results_dir = f"results/behavioral_{provider}_{timestamp}"
    
    print("="*80)
    print(" Behavioral & Functional Testing Suite")
    print("="*80)
    print(f"\nProvider: {provider}")
    print(f"Results directory: {results_dir}")
    print()
    
    tester = BehavioralTester(results_dir)
    
    # Configurations to test
    configs = [
        ('Baseline', f'configs/baseline-{provider}.yaml'),
        ('Reflection Only', f'configs/reflection-only-{provider}.yaml'),
        ('Completion Only', f'configs/completion-only-{provider}.yaml'),
        ('All Features', f'configs/all-features-{provider}.yaml'),
        ('Supervisor', f'configs/supervisor-{provider}.yaml'),
    ]
    
    # Run tests for each configuration
    all_results = {}
    agent_names = {
        'Baseline': 'test_agent_baseline',
        'Reflection Only': 'test_agent_reflection',
        'Completion Only': 'test_agent_completion',
        'All Features': 'test_agent_all',
        'Supervisor': 'test_agent_supervisor'
    }
    
    for config_name, config_path in configs:
        agent_name = agent_names.get(config_name, 'test_agent_baseline')
        results = tester.run_behavioral_suite(config_name, config_path, agent_name)
        all_results[config_name] = results
        
        # Save individual results
        results_file = Path(results_dir) / f'{config_name.lower().replace(" ", "_")}_results.json'
        with open(results_file, 'w') as f:
            json.dump(results, f, indent=2)
    
    # Generate comparison report
    tester.generate_comparison_report(all_results)
    
    print(f"\n✅ Behavioral testing complete!")
    print(f"Results saved to: {results_dir}")

if __name__ == '__main__':
    main()

