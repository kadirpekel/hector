package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureHectorDir ensures the .hector directory exists at the given base path.
// If basePath is empty or ".", it creates ./.hector in the current directory.
// Otherwise, it creates {basePath}/.hector.
//
// This is used by various facilities that need to store data in .hector:
// - Tasks database: ./.hector/tasks.db
// - Document store index state: {sourcePath}/.hector/index_state_*.json
// - Checkpoints: {sourcePath}/.hector/checkpoints/
//
// Returns the full path to the .hector directory and any error.
func EnsureHectorDir(basePath string) (string, error) {
	var hectorDir string
	if basePath == "" || basePath == "." {
		// Root-level .hector directory (for tasks.db, etc.)
		hectorDir = ".hector"
	} else {
		// Source-specific .hector directory (for document stores, checkpoints)
		hectorDir = filepath.Join(basePath, ".hector")
	}

	if err := os.MkdirAll(hectorDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .hector directory at '%s': %w", hectorDir, err)
	}

	return hectorDir, nil
}
