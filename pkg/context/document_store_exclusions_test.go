package context

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context/indexing"
)

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		excludes []string
		path     string
		expected bool
	}{

		{
			name:     "Exclude node_modules directory",
			excludes: []string{"**/node_modules/**"},
			path:     "/project/node_modules/package.json",
			expected: true,
		},
		{
			name:     "Exclude .git directory",
			excludes: []string{"**/.git/**"},
			path:     "/project/.git/config",
			expected: true,
		},
		{
			name:     "Exclude vendor directory",
			excludes: []string{"**/vendor/**"},
			path:     "/project/vendor/module.go",
			expected: true,
		},
		{
			name:     "Exclude nested node_modules",
			excludes: []string{"**/node_modules/**"},
			path:     "/project/src/components/node_modules/react/index.js",
			expected: true,
		},

		{
			name:     "Exclude .log files",
			excludes: []string{"*.log"},
			path:     "/project/logs/app.log",
			expected: true,
		},
		{
			name:     "Exclude .pyc files",
			excludes: []string{"*.pyc"},
			path:     "/project/src/main.pyc",
			expected: true,
		},
		{
			name:     "Exclude .exe files",
			excludes: []string{"*.exe"},
			path:     "/project/build/app.exe",
			expected: true,
		},

		{
			name:     "Exclude .DS_Store",
			excludes: []string{"**/.DS_Store"},
			path:     "/project/src/.DS_Store",
			expected: true,
		},
		{
			name:     "Exclude package-lock.json",
			excludes: []string{"**/package-lock.json"},
			path:     "/project/package-lock.json",
			expected: true,
		},

		{
			name:     "Include .go file",
			excludes: []string{"**/node_modules/**", "*.log"},
			path:     "/project/src/main.go",
			expected: false,
		},
		{
			name:     "Include .js file not in node_modules",
			excludes: []string{"**/node_modules/**"},
			path:     "/project/src/app.js",
			expected: false,
		},
		{
			name:     "Include .md file",
			excludes: []string{"*.log", "*.pyc"},
			path:     "/project/README.md",
			expected: false,
		},

		{
			name:     "Multiple exclusions - match first",
			excludes: []string{"*.log", "*.tmp", "*.cache"},
			path:     "/project/temp/app.log",
			expected: true,
		},
		{
			name:     "Multiple exclusions - match last",
			excludes: []string{"*.log", "*.tmp", "*.cache"},
			path:     "/project/temp/app.cache",
			expected: true,
		},

		{
			name:     "Empty exclude list",
			excludes: []string{},
			path:     "/project/src/main.go",
			expected: false,
		},
		{
			name:     "Exclude subdirectory of build",
			excludes: []string{"**/build/**"},
			path:     "/project/build/dist/app.js",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a proper filter and file source
			filter := indexing.NewPatternFilter("/project", []string{}, tt.excludes)
			fileSource := indexing.NewDirectorySource("/project", filter, 5*1024*1024)

			ds := &DocumentStore{
				config: &config.DocumentStoreConfig{
					Path:            "/project",
					ExcludePatterns: tt.excludes,
				},
				sourcePath: "/project",
				fileSource: fileSource,
			}

			result := ds.shouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExclude() = %v, want %v for path %s", result, tt.expected, tt.path)
			}
		})
	}
}

func TestDefaultExcludePatterns(t *testing.T) {
	cfg := &config.DocumentStoreConfig{}
	cfg.SetDefaults()

	if len(cfg.ExcludePatterns) == 0 {
		t.Fatal("Expected default exclude patterns to be set")
	}

	expectedPatterns := []string{
		"**/node_modules/**",
		"**/.git/**",
		"**/vendor/**",
		"**/__pycache__/**",
		"*.pyc",
		"*.log",
		"**/.DS_Store",
	}

	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range cfg.ExcludePatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected default pattern %q not found in exclude patterns", expected)
		}
	}

	t.Logf("✅ Default exclude patterns count: %d", len(cfg.ExcludePatterns))
}

func TestEmptyFileSkipping(t *testing.T) {
	t.Skip("Empty file skipping is tested via integration test - requires file I/O")

}
