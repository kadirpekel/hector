// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package utils provides utility functions for v2.
package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// Default paths for Hector files
const (
	// DefaultHectorDir is the default directory for Hector data
	DefaultHectorDir = ".hector"

	// DefaultConfigFileName is the default config file name
	DefaultConfigFileName = "config.yaml"

	// DefaultDatabaseFileName is the default database file name
	DefaultDatabaseFileName = "hector.db"
)

// DefaultConfigPath returns the default config file path: .hector/config.yaml
func DefaultConfigPath() string {
	return filepath.Join(DefaultHectorDir, DefaultConfigFileName)
}

// DefaultDatabasePath returns the default database file path: .hector/hector.db
func DefaultDatabasePath() string {
	return filepath.Join(DefaultHectorDir, DefaultDatabaseFileName)
}

// EnsureHectorDir ensures the .hector directory exists at the given base path.
// If basePath is empty or ".", it creates ./.hector in the current directory.
// Otherwise, it creates {basePath}/.hector.
//
// This is used by various facilities that need to store data in .hector:
// - Config file: ./.hector/config.yaml
// - Tasks database: ./.hector/hector.db
// - Document store index state: {sourcePath}/.hector/index_state_*.json
// - Checkpoints: {sourcePath}/.hector/checkpoints/
// - Vector stores: {sourcePath}/.hector/vectors/
//
// Returns the full path to the .hector directory and any error.
func EnsureHectorDir(basePath string) (string, error) {
	var hectorDir string
	if basePath == "" || basePath == "." {
		// Root-level .hector directory (for config, tasks.db, etc.)
		hectorDir = DefaultHectorDir
	} else {
		// Source-specific .hector directory (for document stores, checkpoints)
		hectorDir = filepath.Join(basePath, DefaultHectorDir)
	}

	if err := os.MkdirAll(hectorDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .hector directory at '%s': %w", hectorDir, err)
	}

	return hectorDir, nil
}
