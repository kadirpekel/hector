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

package config

import "fmt"

// LoggerConfig configures logging behavior.
//
// Priority order (highest to lowest):
//  1. CLI flags (--log-level, --log-file, --log-format)
//  2. Environment variables (LOG_LEVEL, LOG_FILE, LOG_FORMAT)
//  3. Config file (logger section)
//  4. Defaults (info level, simple format, stderr)
//
// Example:
//
//	logger:
//	  level: info
//	  file: hector.log
//	  format: simple
type LoggerConfig struct {
	// Level specifies the log level (debug, info, warn, error).
	// Default: info
	Level string `yaml:"level,omitempty"`

	// File specifies the log file path.
	// If empty, logs go to stderr.
	// Default: empty (stderr)
	File string `yaml:"file,omitempty"`

	// Format specifies the log format.
	// Values: "simple" (level + message), "verbose" (time + level + message), or custom.
	// Default: simple
	Format string `yaml:"format,omitempty"`
}

// SetDefaults applies default values to LoggerConfig.
func (c *LoggerConfig) SetDefaults() {
	if c.Level == "" {
		c.Level = "info"
	}
	if c.Format == "" {
		c.Format = "simple"
	}
	// File defaults to empty (stderr) - no need to set
}

// Validate checks the logger configuration.
func (c *LoggerConfig) Validate() error {
	if c.Level != "" {
		validLevels := map[string]bool{
			"debug":   true,
			"info":    true,
			"warn":    true,
			"warning": true,
			"error":   true,
		}
		if !validLevels[c.Level] {
			return fmt.Errorf("invalid log level %q (valid: debug, info, warn, error)", c.Level)
		}
	}

	// Format can be "simple", "verbose", or any custom value
	// No validation needed - custom formats are allowed
	_ = c.Format

	return nil
}
