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

// DatabaseConfig holds configuration for SQL database connections.
// Supports PostgreSQL, MySQL, and SQLite.
type DatabaseConfig struct {
	// Driver specifies the database driver: "postgres", "mysql", or "sqlite"
	Driver string `yaml:"driver"`

	// Host is the database server hostname (not required for SQLite).
	Host string `yaml:"host,omitempty"`

	// Port is the database server port (not required for SQLite).
	Port int `yaml:"port,omitempty"`

	// Database is the database name (or file path for SQLite).
	Database string `yaml:"database"`

	// Username for database authentication (not required for SQLite).
	Username string `yaml:"username,omitempty"`

	// Password for database authentication (not required for SQLite).
	Password string `yaml:"password,omitempty"`

	// SSLMode for PostgreSQL connections.
	SSLMode string `yaml:"ssl_mode,omitempty"`

	// MaxConns is the maximum number of open connections.
	MaxConns int `yaml:"max_conns,omitempty"`

	// MaxIdle is the maximum number of idle connections.
	MaxIdle int `yaml:"max_idle,omitempty"`
}

// SetDefaults applies default values to the database config.
func (c *DatabaseConfig) SetDefaults() {
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MaxIdle == 0 {
		c.MaxIdle = 5
	}

	// Default ports per driver
	if c.Port == 0 {
		switch c.Driver {
		case "postgres":
			c.Port = 5432
		case "mysql":
			c.Port = 3306
		}
	}

	// Default SSL mode for PostgreSQL
	if c.Driver == "postgres" && c.SSLMode == "" {
		c.SSLMode = "disable"
	}
}

// Validate checks the database configuration.
func (c *DatabaseConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}

	validDrivers := map[string]bool{
		"postgres": true,
		"mysql":    true,
		"sqlite":   true,
		"sqlite3":  true,
	}
	if !validDrivers[c.Driver] {
		return fmt.Errorf("invalid driver %q (valid: postgres, mysql, sqlite)", c.Driver)
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	// For non-SQLite, require host
	if c.Driver != "sqlite" && c.Driver != "sqlite3" {
		if c.Host == "" {
			return fmt.Errorf("host is required for %s", c.Driver)
		}
	}

	if c.MaxConns < 0 {
		return fmt.Errorf("max_conns must be non-negative")
	}

	if c.MaxIdle < 0 {
		return fmt.Errorf("max_idle must be non-negative")
	}

	return nil
}

// DSN returns the data source name (connection string) for the database.
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case "postgres":
		// Build PostgreSQL DSN, only including credentials if provided
		dsn := fmt.Sprintf("host=%s port=%d dbname=%s", c.Host, c.Port, c.Database)
		if c.Username != "" {
			dsn += fmt.Sprintf(" user=%s", c.Username)
		}
		if c.Password != "" {
			dsn += fmt.Sprintf(" password=%s", c.Password)
		}
		if c.SSLMode != "" {
			dsn += fmt.Sprintf(" sslmode=%s", c.SSLMode)
		}
		return dsn
	case "mysql":
		// MySQL DSN format: [username[:password]@][protocol[(address)]]/dbname
		if c.Username != "" {
			return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				c.Username, c.Password, c.Host, c.Port, c.Database)
		}
		return fmt.Sprintf("tcp(%s:%d)/%s", c.Host, c.Port, c.Database)
	case "sqlite", "sqlite3":
		return c.Database // For SQLite, database is the file path
	default:
		return ""
	}
}

// DriverName returns the normalized driver name for sql.Open().
// Converts "sqlite" to "sqlite3" for the go-sqlite3 driver.
func (c *DatabaseConfig) DriverName() string {
	if c.Driver == "sqlite" {
		return "sqlite3"
	}
	return c.Driver
}

// Dialect returns the normalized SQL dialect name for query building.
// Converts "sqlite3" to "sqlite" for consistent dialect handling.
func (c *DatabaseConfig) Dialect() string {
	if c.Driver == "sqlite3" {
		return "sqlite"
	}
	return c.Driver
}
