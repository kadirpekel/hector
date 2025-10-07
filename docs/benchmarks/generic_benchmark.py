#!/usr/bin/env python3
"""
Generic Benchmark Framework

Runs A/B tests and benchmarks for any configuration changes based on test definitions.

Usage:
    python generic_benchmark.py <test_definition.json>
    
Example:
    python generic_benchmark.py tests/prompt_variations.json
"""

import json
import os
import sys
import subprocess
import time
import yaml
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional


class GenericBenchmark:
    """Generic framework for running A/B tests and benchmarks"""
    
    def __init__(self, test_def_path: str):
        self.test_def_path = Path(test_def_path)
        self.test_def = None
        self.results_dir = Path('results') / 'generic'
        self.hector_bin = Path('../hector').resolve()
        
    def load_test_definition(self) -> bool:
        """Load and validate test definition"""
        if not self.test_def_path.exists():
            print(f"❌ Test definition not found: {self.test_def_path}")
            return False
        
        try:
            with open(self.test_def_path) as f:
                self.test_def = json.load(f)
            
            # Validate required fields
            required = ['test_name', 'description', 'variants', 'scenarios']
            for field in required:
                if field not in self.test_def:
                    print(f"❌ Missing required field: {field}")
                    return False
            
            print(f"✅ Loaded test definition: {self.test_def['test_name']}")
            print(f"   Description: {self.test_def['description']}")
            print(f"   Variants: {len(self.test_def['variants'])}")
            print(f"   Scenarios: {len(self.test_def['scenarios'])}")
            return True
            
        except Exception as e:
            print(f"❌ Failed to load test definition: {e}")
            return False
    
    def prepare_variant_config(self, variant: Dict, provider: str) -> Optional[Path]:
        """Prepare configuration for a variant"""
        config_path = variant.get('config')
        
        # If config is a string, it's a file path
        if isinstance(config_path, str):
            config_file = Path(config_path)
            if not config_file.exists():
                print(f"⚠️  Config file not found: {config_file}")
                return None
            
            # Apply overrides if specified
            if 'config_overrides' in variant:
                with open(config_file) as f:
                    config = yaml.safe_load(f)
                
                # Apply overrides
                for key, value in variant['config_overrides'].items():
                    self._set_nested_key(config, key, value)
                
                # Write modified config
                temp_config = self.results_dir / f"{variant['name']}_{provider}_temp.yaml"
                with open(temp_config, 'w') as f:
                    yaml.safe_dump(config, f)
                
                return temp_config
            
            return config_file
        
        # If config is an object, generate config file
        elif isinstance(config_path, dict):
            config = config_path
            temp_config = self.results_dir / f"{variant['name']}_{provider}_temp.yaml"
            with open(temp_config, 'w') as f:
                yaml.safe_dump(config, f)
            return temp_config
        
        return None
    
    def _set_nested_key(self, d: dict, key_path: str, value):
        """Set a nested dictionary key using dot notation"""
        keys = key_path.split('.')
        for key in keys[:-1]:
            d = d.setdefault(key, {})
        d[keys[-1]] = value
    
    def run_scenario(self, variant: Dict, scenario: Dict, config_path: Path, agent_name: str) -> Dict:
        """Run a single scenario against a variant"""
        scenario_name = scenario['name']
        prompt = scenario['prompt']
        
        print(f"   Running scenario: {scenario_name}")
        
        # Start server
        server_log = self.results_dir / f"{variant['name']}_{scenario_name}_server.log"
        server_proc = subprocess.Popen(
            [str(self.hector_bin), 'serve', '--config', str(config_path)],
            stdout=open(server_log, 'w'),
            stderr=subprocess.STDOUT,
            cwd='.'
        )
        time.sleep(3)  # Wait for server to start
        
        try:
            # Run test
            start_time = time.time()
            result = subprocess.run(
                [str(self.hector_bin), 'call', agent_name, prompt],
                capture_output=True,
                text=True,
                timeout=60,
                cwd='.'
            )
            elapsed_time = time.time() - start_time
            
            # Parse output for metrics
            output = result.stdout
            success = result.returncode == 0
            
            # Extract metrics from output
            tokens = self._extract_metric(output, 'tokens')
            iterations = self._extract_metric(output, 'iteration')
            
            # Check success criteria
            if 'expected_output_keywords' in scenario:
                for keyword in scenario['expected_output_keywords']:
                    if keyword.lower() not in output.lower():
                        success = False
                        break
            
            return {
                'scenario': scenario_name,
                'variant': variant['name'],
                'success': success,
                'elapsed_time': elapsed_time,
                'tokens': tokens,
                'iterations': iterations,
                'output': output[:500]  # Truncate for storage
            }
        
        except subprocess.TimeoutExpired:
            return {
                'scenario': scenario_name,
                'variant': variant['name'],
                'success': False,
                'error': 'Timeout',
                'elapsed_time': 60
            }
        
        finally:
            server_proc.terminate()
            server_proc.wait()
    
    def _extract_metric(self, output: str, metric_name: str) -> int:
        """Extract a metric value from agent output"""
        # Simple pattern matching - can be enhanced
        import re
        
        patterns = {
            'tokens': r'(\d+)\s*tokens?',
            'iteration': r'iteration\s*(\d+)',
        }
        
        pattern = patterns.get(metric_name)
        if pattern:
            match = re.search(pattern, output, re.IGNORECASE)
            if match:
                return int(match.group(1))
        
        return 0
    
    def run_benchmark(self):
        """Run complete benchmark based on test definition"""
        if not self.load_test_definition():
            return
        
        # Create results directory
        test_name = self.test_def['test_name']
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        run_dir = self.results_dir / test_name / f"run_{timestamp}"
        run_dir.mkdir(parents=True, exist_ok=True)
        self.results_dir = run_dir
        
        print(f"\n{'='*80}")
        print(f"RUNNING TEST: {test_name}")
        print(f"{'='*80}\n")
        
        all_results = []
        providers = self.test_def.get('providers', ['openai'])
        iterations = self.test_def.get('iterations', 1)
        
        # Run each variant
        for variant in self.test_def['variants']:
            print(f"\n🧪 Testing variant: {variant['name']}")
            if 'description' in variant:
                print(f"   {variant['description']}")
            
            for provider in providers:
                print(f"   Provider: {provider}")
                
                # Prepare config
                config_path = self.prepare_variant_config(variant, provider)
                if not config_path:
                    print(f"   ⚠️  Failed to prepare config, skipping")
                    continue
                
                # Determine agent name from config
                with open(config_path) as f:
                    config = yaml.safe_load(f)
                    agents = config.get('agents', {})
                    agent_name = list(agents.keys())[0] if agents else 'test_agent'
                
                # Run scenarios
                for scenario in self.test_def['scenarios']:
                    for i in range(iterations):
                        result = self.run_scenario(variant, scenario, config_path, agent_name)
                        result['provider'] = provider
                        result['iteration'] = i + 1
                        all_results.append(result)
                        
                        status = "✅" if result.get('success') else "❌"
                        print(f"      {status} {scenario['name']} ({result.get('elapsed_time', 0):.1f}s)")
        
        # Save results
        results_file = run_dir / 'results.json'
        with open(results_file, 'w') as f:
            json.dump(all_results, f, indent=2)
        
        print(f"\n✅ Results saved: {results_file}")
        
        # Generate summary
        self.generate_summary(all_results, run_dir)
    
    def generate_summary(self, results: List[Dict], run_dir: Path):
        """Generate summary report"""
        print(f"\n{'='*80}")
        print("SUMMARY")
        print(f"{'='*80}\n")
        
        variants = list(set(r['variant'] for r in results))
        
        print(f"{'Variant':<20} {'Success Rate':<15} {'Avg Tokens':<15} {'Avg Time':<15}")
        print("-" * 80)
        
        summary = {}
        for variant in variants:
            variant_results = [r for r in results if r['variant'] == variant]
            
            success_count = sum(1 for r in variant_results if r.get('success'))
            success_rate = success_count / len(variant_results) if variant_results else 0
            
            avg_tokens = sum(r.get('tokens', 0) for r in variant_results) / len(variant_results) if variant_results else 0
            avg_time = sum(r.get('elapsed_time', 0) for r in variant_results) / len(variant_results) if variant_results else 0
            
            summary[variant] = {
                'success_rate': success_rate,
                'avg_tokens': avg_tokens,
                'avg_time': avg_time,
                'total_tests': len(variant_results)
            }
            
            print(f"{variant:<20} {success_rate:<15.1%} {avg_tokens:<15.0f} {avg_time:<15.1f}s")
        
        # Save summary
        summary_file = run_dir / 'summary.json'
        with open(summary_file, 'w') as f:
            json.dump(summary, f, indent=2)
        
        print(f"\n📊 Summary saved: {summary_file}")
        
        # Recommendations
        print("\n💡 Recommendations:")
        
        # Find best variant by primary metric
        primary_metric = self.test_def.get('metrics', {}).get('primary', 'quality')
        
        if primary_metric == 'quality':
            best = max(variants, key=lambda v: summary[v]['success_rate'])
            print(f"   - Best quality: {best} ({summary[best]['success_rate']:.1%} success)")
        elif primary_metric == 'cost':
            best = min(variants, key=lambda v: summary[v]['avg_tokens'])
            print(f"   - Most cost-effective: {best} ({summary[best]['avg_tokens']:.0f} avg tokens)")
        elif primary_metric == 'speed':
            best = min(variants, key=lambda v: summary[v]['avg_time'])
            print(f"   - Fastest: {best} ({summary[best]['avg_time']:.1f}s avg)")


def main():
    if len(sys.argv) < 2:
        print("Usage: python generic_benchmark.py <test_definition.json>")
        print("\nExample:")
        print("  python generic_benchmark.py tests/prompt_variations.json")
        sys.exit(1)
    
    test_def_path = sys.argv[1]
    benchmark = GenericBenchmark(test_def_path)
    benchmark.run_benchmark()


if __name__ == '__main__':
    main()

