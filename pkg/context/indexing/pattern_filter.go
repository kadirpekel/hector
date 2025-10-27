package indexing

import (
	"path/filepath"
	"strings"
)

// PatternFilter implements FileFilter using include/exclude patterns
type PatternFilter struct {
	sourcePath   string
	cache        *PatternCache
	includeCount int
}

// PatternCache provides fast pattern matching
type PatternCache struct {
	dirExcludes  map[string]bool
	extExcludes  map[string]bool
	dirIncludes  map[string]bool
	extIncludes  map[string]bool
	globExcludes []string
	globIncludes []string
}

// NewPatternFilter creates a new pattern-based filter
func NewPatternFilter(sourcePath string, includePatterns, excludePatterns []string) *PatternFilter {
	return &PatternFilter{
		sourcePath:   sourcePath,
		cache:        buildPatternCache(includePatterns, excludePatterns),
		includeCount: len(includePatterns),
	}
}

// ShouldInclude checks if a file matches include patterns
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

// ShouldExclude checks if a file matches exclude patterns
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

func buildPatternCache(includePatterns, excludePatterns []string) *PatternCache {
	cache := &PatternCache{
		dirExcludes: make(map[string]bool),
		extExcludes: make(map[string]bool),
		dirIncludes: make(map[string]bool),
		extIncludes: make(map[string]bool),
	}

	// Process exclude patterns
	for _, pattern := range excludePatterns {
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

	// Process include patterns
	for _, pattern := range includePatterns {
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

	return cache
}
