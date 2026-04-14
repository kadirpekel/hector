package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/verikod/hector/pkg/utils"
)

// DatabaseConfig holds configuration for SQL database connections.
// Supports DSN format: postgres://user:pass@host:port/db, mysql://..., sqlite://path
type DatabaseConfig struct {
	// RawDSN is the original DSN string provided by the user.
	RawDSN string `yaml:"-" json:"-"`

	// Parsed fields from DSN
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
	MaxConns int
	MaxIdle  int
	// Params holds additional connection parameters
	Params map[string]string `yaml:"-" json:"-"`
}

// DefaultDSN returns the default database connection string.
func DefaultDSN() string {
	return "sqlite://" + utils.DefaultDatabasePath()
}

// ParseDSN parses a database connection string and returns a DatabaseConfig.
// Supported formats:
//   - sqlite://.hector/hector.db
//   - sqlite://:memory:
//   - postgres://user:pass@host:5432/dbname?sslmode=disable
//   - mysql://user:pass@host:3306/dbname
func ParseDSN(dsn string) (*DatabaseConfig, error) {
	if dsn == "" {
		dsn = DefaultDSN()
	}

	cfg := &DatabaseConfig{
		RawDSN:   dsn,
		MaxConns: 25,
		MaxIdle:  5,
		Params:   make(map[string]string),
	}

	// Parse URL
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN format: %w", err)
	}

	// Extract driver from scheme
	cfg.Driver = strings.ToLower(u.Scheme)
	if cfg.Driver == "sqlite3" {
		cfg.Driver = "sqlite"
	} else if cfg.Driver == "postgresql" {
		cfg.Driver = "postgres"
	}

	// Populate params from query string
	for k, v := range u.Query() {
		if len(v) > 0 {
			cfg.Params[k] = v[0]
		}
	}

	switch cfg.Driver {
	case "sqlite":
		// sqlite://path or sqlite://:memory:
		cfg.Database = strings.TrimPrefix(dsn, "sqlite://")
		if cfg.Database == "" {
			cfg.Database = ".hector/hector.db"
		}

	case "postgres", "pgx":
		// Use pgx driver for postgres
		cfg.Driver = "pgx"
		cfg.Host = u.Hostname()
		if u.Port() != "" {
			if p, err := strconv.Atoi(u.Port()); err == nil {
				cfg.Port = p
			}
		}
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
		cfg.Database = strings.TrimPrefix(u.Path, "/")
		if u.User != nil {
			cfg.Username = u.User.Username()
			cfg.Password, _ = u.User.Password()
		}
		// Handle sslmode specifically as it's often critical
		if sslmode := cfg.Params["sslmode"]; sslmode != "" {
			cfg.SSLMode = sslmode
		} else {
			cfg.SSLMode = "disable"
		}

	case "mysql":
		cfg.Host = u.Hostname()
		if u.Port() != "" {
			if p, err := strconv.Atoi(u.Port()); err == nil {
				cfg.Port = p
			}
		}
		if cfg.Port == 0 {
			cfg.Port = 3306
		}
		cfg.Database = strings.TrimPrefix(u.Path, "/")
		if u.User != nil {
			cfg.Username = u.User.Username()
			cfg.Password, _ = u.User.Password()
		}

	default:
		return nil, fmt.Errorf("unsupported database driver: %s (valid: sqlite, postgres, pgx, mysql)", cfg.Driver)
	}

	return cfg, nil
}

// Validate checks the database configuration.
func (c *DatabaseConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}

	validDrivers := map[string]bool{
		"postgres": true,
		"pgx":      true,
		"mysql":    true,
		"sqlite":   true,
	}
	if !validDrivers[c.Driver] {
		return fmt.Errorf("invalid driver %q (valid: postgres, pgx, mysql, sqlite)", c.Driver)
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	// For non-SQLite, require host
	if c.Driver != "sqlite" {
		if c.Host == "" {
			return fmt.Errorf("host is required for %s", c.Driver)
		}
	}

	return nil
}

// DSN returns the data source name (connection string) for the database.
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case "postgres", "pgx":
		// Build PostgreSQL URL for pgx
		// Format: postgres://user:password@host:port/dbname?param1=value1
		u := url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
			Path:   c.Database,
		}

		if c.Username != "" {
			if c.Password != "" {
				u.User = url.UserPassword(c.Username, c.Password)
			} else {
				u.User = url.User(c.Username)
			}
		}

		q := u.Query()
		if c.SSLMode != "" {
			q.Set("sslmode", c.SSLMode)
		}

		// Add all other params
		for k, v := range c.Params {
			if k != "sslmode" { // Already handled
				q.Set(k, v)
			}
		}

		u.RawQuery = q.Encode()
		return u.String()

	case "mysql":
		// MySQL DSN format: [username[:password]@][protocol[(address)]]/dbname
		if c.Username != "" {
			return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				c.Username, c.Password, c.Host, c.Port, c.Database)
		}
		return fmt.Sprintf("tcp(%s:%d)/%s", c.Host, c.Port, c.Database)
	case "sqlite":
		return c.Database // For SQLite, database is the file path
	default:
		return ""
	}
}

// DriverName returns the normalized driver name for sql.Open().
func (c *DatabaseConfig) DriverName() string {
	return c.Driver
}

// Dialect returns the SQL dialect name for query building.
func (c *DatabaseConfig) Dialect() string {
	if c.Driver == "pgx" {
		return "postgres"
	}
	return c.Driver
}
