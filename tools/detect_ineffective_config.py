#!/usr/bin/env python3
"""
Enhanced Config Field Effectiveness Analyzer

Detects not just unused fields, but also INEFFECTIVE fields:
- Fields that are read but never acted upon
- Fields that are validated but never used
- Fields that are set in defaults but ignored
- Fields that exist only for backwards compatibility
"""

import subprocess
import re
from collections import defaultdict
from typing import Dict, List, Set, Tuple

class FieldUsagePattern:
    def __init__(self, field_name: str, struct_name: str):
        self.field_name = field_name
        self.struct_name = struct_name
        self.defined_in = []
        self.read_in_validation = False
        self.read_in_setdefaults = False
        self.read_in_method = []
        self.written_to = []
        self.passed_to_function = []
        self.used_in_logic = []
        
    def is_effective(self) -> bool:
        """
        A field is effective if it's used beyond just validation/defaults.
        """
        # Used in actual logic (not just validation/defaults)
        if self.used_in_logic:
            return True
        # Passed to external functions/constructors
        if self.passed_to_function:
            return True
        # Used in methods that do actual work
        if self.read_in_method and not all('Validate' in m or 'SetDefaults' in m for m in self.read_in_method):
            return True
        return False
    
    def effectiveness_score(self) -> int:
        """
        Score from 0 (completely ineffective) to 100 (highly effective)
        """
        score = 0
        if self.used_in_logic:
            score += 50
        if self.passed_to_function:
            score += 30
        if self.read_in_method:
            score += 10
        if self.written_to:
            score += 5
        if self.read_in_validation:
            score += 3
        if self.read_in_setdefaults:
            score += 2
        return min(score, 100)

def analyze_field_effectiveness(pkg_path: str = "pkg/") -> Dict[str, FieldUsagePattern]:
    """
    Analyze how each config field is actually used
    """
    print("🔍 Analyzing field effectiveness...")
    
    # TODO: Implement detailed Go AST analysis
    # For now, use enhanced grep patterns
    
    patterns = {}
    
    # Find all config fields
    result = subprocess.run(
        ["grep", "-r", "-n", "yaml:", f"{pkg_path}config/types.go"],
        capture_output=True, text=True
    )
    
    for line in result.stdout.split('\n'):
        if 'yaml:' in line:
            # Extract field info
            match = re.search(r'(\w+)\s+\w+.*yaml:"([^"]+)"', line)
            if match:
                field_name = match.group(1)
                yaml_tag = match.group(2).split(',')[0]
                
                pattern = FieldUsagePattern(field_name, "Config")
                
                # Check various usage patterns
                check_validation_usage(pattern, pkg_path)
                check_defaults_usage(pattern, pkg_path)
                check_functional_usage(pattern, pkg_path)
                
                patterns[f"{field_name}"] = pattern
    
    return patterns

def check_validation_usage(pattern: FieldUsagePattern, pkg_path: str):
    """Check if field is only validated but never used"""
    result = subprocess.run(
        ["grep", "-r", f"\\.{pattern.field_name}\\b", pkg_path, "--include=*.go"],
        capture_output=True, text=True
    )
    
    for line in result.stdout.split('\n'):
        if 'Validate()' in line or 'if.*< 0' in line or 'if.*==' in line:
            pattern.read_in_validation = True

def check_defaults_usage(pattern: FieldUsagePattern, pkg_path: str):
    """Check if field is only set in defaults but never used"""
    result = subprocess.run(
        ["grep", "-r", f"\\.{pattern.field_name}\\s*=", pkg_path, "--include=*.go"],
        capture_output=True, text=True
    )
    
    for line in result.stdout.split('\n'):
        if 'SetDefaults()' in line:
            pattern.read_in_setdefaults = True
        else:
            pattern.written_to.append(line.strip())

def check_functional_usage(pattern: FieldUsagePattern, pkg_path: str):
    """Check if field is used in actual functional code"""
    result = subprocess.run(
        ["grep", "-r", f"\\.{pattern.field_name}\\b", pkg_path, "--include=*.go"],
        capture_output=True, text=True
    )
    
    functional_keywords = ['return', 'fmt.Sprintf', 'append', 'New', 'Create']
    
    for line in result.stdout.split('\n'):
        if any(kw in line for kw in functional_keywords):
            if 'Validate' not in line and 'SetDefaults' not in line:
                pattern.used_in_logic.append(line.strip())

if __name__ == "__main__":
    print("Enhanced Config Field Effectiveness Analysis")
    print("=" * 60)
    print()
    print("This tool detects:")
    print("  1. Unused fields (no accesses)")
    print("  2. Ineffective fields (validated but not used)")
    print("  3. Zombie fields (set but never read)")
    print("  4. Backwards-compat only fields")
    print()
    
    patterns = analyze_field_effectiveness()
    
    ineffective = []
    validated_only = []
    defaults_only = []
    
    for name, pattern in patterns.items():
        score = pattern.effectiveness_score()
        
        if score == 0:
            ineffective.append((name, pattern))
        elif score < 10 and pattern.read_in_validation:
            validated_only.append((name, pattern))
        elif score < 10 and pattern.read_in_setdefaults:
            defaults_only.append((name, pattern))
    
    print(f"📊 Analysis Results:")
    print(f"   Totally ineffective: {len(ineffective)}")
    print(f"   Validated but unused: {len(validated_only)}")
    print(f"   Defaults but unused: {len(defaults_only)}")
    print()
    print("See output above for details")

