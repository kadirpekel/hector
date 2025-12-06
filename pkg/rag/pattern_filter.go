// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PatternFilter implements FileFilter using include/exclude patterns.
//
// Direct port from legacy pkg/context/indexing/pattern_filter.go
type PatternFilter struct {
	sourcePath   string
	cache        *PatternCache
	includeCount int
}

// PatternCache provides fast pattern matching.
//
// Direct port from legacy pkg/context/indexing/pattern_filter.go
type PatternCache struct {
	dirExcludes  map[string]bool
	extExcludes  map[string]bool
	dirIncludes  map[string]bool
	extIncludes  map[string]bool
	globExcludes []string
	globIncludes []string
}

// NewPatternFilter creates a new pattern-based filter with validation.
//
// Direct port from legacy pkg/context/indexing/pattern_filter.go
func NewPatternFilter(sourcePath string, includePatterns, excludePatterns []string) (*PatternFilter, error) {
	cache, err := buildPatternCache(includePatterns, excludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to build pattern cache: %w", err)
	}

	return &PatternFilter{
		sourcePath:   sourcePath,
		cache:        cache,
		includeCount: len(includePatterns),
	}, nil
}

// validatePattern checks if a glob pattern is valid.
func validatePattern(pattern string) error {
	// Empty pattern is invalid
	if pattern == "" {
		return fmt.Errorf("empty pattern")
	}

	// Normalize the pattern
	normalizedPattern := filepath.ToSlash(pattern)

	// Test the pattern with filepath.Match
	// Use a simple test path to check pattern syntax
	testPath := "test/path/file.txt"
	_, err := filepath.Match(normalizedPattern, testPath)
	if err != nil {
		return fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
	}

	// Check for common mistakes
	if strings.Contains(pattern, "**") && !strings.HasPrefix(normalizedPattern, "**/") {
		// ** should only be used at the beginning
		return fmt.Errorf("pattern '%s': '**' is only supported at the beginning (e.g., '**/dir/**')", pattern)
	}

	return nil
}

// ShouldInclude checks if a file matches include patterns.
func (pf *PatternFilter) ShouldInclude(path string) bool {
	if pf.includeCount == 0 {
		return true
	}

	relPath, err := filepath.Rel(pf.sourcePath, path)
	if err != nil {
		relPath = path
	}

	normalizedPath := filepath.ToSlash(relPath)

	// Fast path: Check extension inclusions
	ext := filepath.Ext(normalizedPath)
	if ext != "" && pf.cache.extIncludes[ext] {
		return true
	}

	// Fast path: Check directory name inclusions
	pathParts := strings.Split(normalizedPath, "/")
	for _, part := range pathParts {
		if pf.cache.dirIncludes[part] {
			return true
		}
	}

	// Slow path: Check glob patterns
	for _, pattern := range pf.cache.globIncludes {
		if pattern == "*" {
			return true
		}

		if matched, err := filepath.Match(pattern, normalizedPath); err == nil && matched {
			return true
		}

		if strings.HasPrefix(pattern, "**/") {
			simplePattern := strings.TrimPrefix(pattern, "**/")
			if matched, err := filepath.Match(simplePattern, filepath.Base(normalizedPath)); err == nil && matched {
				return true
			}
		}
	}

	return false
}

// ShouldExclude checks if a file matches exclude patterns.
func (pf *PatternFilter) ShouldExclude(path string) bool {
	relPath, err := filepath.Rel(pf.sourcePath, path)
	if err != nil {
		relPath = path
	}

	normalizedPath := filepath.ToSlash(relPath)

	// Fast path: Check extension exclusions
	ext := filepath.Ext(normalizedPath)
	if ext != "" && pf.cache.extExcludes[ext] {
		return true
	}

	// Fast path: Check directory name exclusions
	pathParts := strings.Split(normalizedPath, "/")
	for _, part := range pathParts {
		if pf.cache.dirExcludes[part] {
			return true
		}
	}

	// Slow path: Check glob patterns
	for _, pattern := range pf.cache.globExcludes {
		if matched, err := filepath.Match(pattern, normalizedPath); err == nil && matched {
			return true
		}

		if strings.HasPrefix(pattern, "**/") {
			simplePattern := strings.TrimPrefix(pattern, "**/")
			if matched, err := filepath.Match(simplePattern, filepath.Base(normalizedPath)); err == nil && matched {
				return true
			}
		}
	}

	return false
}

// buildPatternCache builds the pattern cache from include/exclude patterns.
func buildPatternCache(includePatterns, excludePatterns []string) (*PatternCache, error) {
	cache := &PatternCache{
		dirExcludes: make(map[string]bool),
		extExcludes: make(map[string]bool),
		dirIncludes: make(map[string]bool),
		extIncludes: make(map[string]bool),
	}

	// Validate and process exclude patterns
	for _, pattern := range excludePatterns {
		if err := validatePattern(pattern); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern: %w", err)
		}

		normalizedPattern := filepath.ToSlash(pattern)

		if strings.HasPrefix(normalizedPattern, "**/") && strings.HasSuffix(normalizedPattern, "/**") {
			dirName := strings.Trim(normalizedPattern, "*/")
			cache.dirExcludes[dirName] = true
		} else if strings.HasPrefix(normalizedPattern, "*.") {
			ext := strings.TrimPrefix(normalizedPattern, "*")
			cache.extExcludes[ext] = true
		} else if strings.HasPrefix(normalizedPattern, ".") && !strings.Contains(normalizedPattern, "/") {
			cache.extExcludes[normalizedPattern] = true
		} else if !strings.Contains(normalizedPattern, "*") {
			cache.dirExcludes[normalizedPattern] = true
		} else {
			cache.globExcludes = append(cache.globExcludes, normalizedPattern)
		}
	}

	// Validate and process include patterns
	for _, pattern := range includePatterns {
		if err := validatePattern(pattern); err != nil {
			return nil, fmt.Errorf("invalid include pattern: %w", err)
		}

		normalizedPattern := filepath.ToSlash(pattern)

		if strings.HasPrefix(normalizedPattern, "**/") && strings.HasSuffix(normalizedPattern, "/**") {
			dirName := strings.Trim(normalizedPattern, "*/")
			cache.dirIncludes[dirName] = true
		} else if strings.HasPrefix(normalizedPattern, "*.") {
			ext := strings.TrimPrefix(normalizedPattern, "*")
			cache.extIncludes[ext] = true
		} else if strings.HasPrefix(normalizedPattern, ".") && !strings.Contains(normalizedPattern, "/") {
			cache.extIncludes[normalizedPattern] = true
		} else if !strings.Contains(normalizedPattern, "*") {
			cache.dirIncludes[normalizedPattern] = true
		} else {
			cache.globIncludes = append(cache.globIncludes, normalizedPattern)
		}
	}

	return cache, nil
}

// Ensure PatternFilter implements FileFilter.
var _ FileFilter = (*PatternFilter)(nil)
