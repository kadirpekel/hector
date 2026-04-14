// Package app provides multi-app (multi-tenant) management for Hector.
//
// Each app represents an isolated workspace with its own:
//   - Configuration (agents, tools, LLMs)
//   - Data (sessions, tasks, memory)
//   - Authentication (JWT tokens)
//
// Apps are stored in the database and managed via the Admin API.
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// App represents a tenant/workspace in the multi-app system.
type App struct {
	// ID is the unique identifier for the app (also used as app_name/tenant_id)
	ID string `json:"id"`

	// Name is the human-readable display name
	Name string `json:"name"`

	// ConfigJSON stores the full hector configuration as JSON/YAML
	ConfigJSON string `json:"config_json,omitempty"`

	// CreatedAt is when the app was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the app was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// Store provides CRUD operations for apps.
type Store interface {
	// Create creates a new app and returns it with generated ID if not provided.
	Create(ctx context.Context, app *App) (*App, error)

	// Get retrieves an app by ID.
	Get(ctx context.Context, id string) (*App, error)

	// List returns all apps.
	List(ctx context.Context) ([]*App, error)

	// Update updates an existing app's configuration.
	Update(ctx context.Context, app *App) error

	// Delete removes an app by ID.
	Delete(ctx context.Context, id string) error

	// Exists checks if an app with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)
}

// SQLStore implements Store using a SQL database.
type SQLStore struct {
	db      *sql.DB
	dialect string
}

// Schema creation SQL
const createAppsSchemaSQL = `
CREATE TABLE IF NOT EXISTS apps (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    config_json TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
)`

const createAppsNameIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_apps_name ON apps(name)`

// NewSQLStore creates a new SQL-based app store.
func NewSQLStore(db *sql.DB, dialect string) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Normalize dialect
	normalizedDialect := dialect
	if dialect == "sqlite3" {
		normalizedDialect = "sqlite"
	}

	switch normalizedDialect {
	case "postgres", "mysql", "sqlite", "pgx":
		// Valid dialects
	default:
		return nil, fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", dialect)
	}

	s := &SQLStore{
		db:      db,
		dialect: normalizedDialect,
	}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize apps schema: %w", err)
	}

	return s, nil
}

// initSchema creates the required tables if they don't exist.
func (s *SQLStore) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statements := []string{
		createAppsSchemaSQL,
		createAppsNameIndexSQL,
	}

	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	slog.Debug("Apps schema initialized")
	return nil
}

// Create creates a new app.
func (s *SQLStore) Create(ctx context.Context, app *App) (*App, error) {
	if app == nil {
		return nil, fmt.Errorf("app is required")
	}

	// Generate ID if not provided
	if app.ID == "" {
		app.ID = uuid.NewString()
	}

	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now

	query := `INSERT INTO apps (id, name, config_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	if s.dialect == "postgres" || s.dialect == "pgx" {
		query = `INSERT INTO apps (id, name, config_json, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	}

	_, err := s.db.ExecContext(ctx, query, app.ID, app.Name, app.ConfigJSON, app.CreatedAt, app.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	slog.Info("App created", "id", app.ID, "name", app.Name)
	return app, nil
}

// Get retrieves an app by ID.
func (s *SQLStore) Get(ctx context.Context, id string) (*App, error) {
	query := `SELECT id, name, config_json, created_at, updated_at FROM apps WHERE id = ?`
	if s.dialect == "postgres" || s.dialect == "pgx" {
		query = `SELECT id, name, config_json, created_at, updated_at FROM apps WHERE id = $1`
	}

	var app App
	var configJSON sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID, &app.Name, &configJSON, &app.CreatedAt, &app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	if configJSON.Valid {
		app.ConfigJSON = configJSON.String
	}

	return &app, nil
}

// List returns all apps.
func (s *SQLStore) List(ctx context.Context) ([]*App, error) {
	query := `SELECT id, name, config_json, created_at, updated_at FROM apps ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		var app App
		var configJSON sql.NullString
		if err := rows.Scan(&app.ID, &app.Name, &configJSON, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan app: %w", err)
		}
		if configJSON.Valid {
			app.ConfigJSON = configJSON.String
		}
		apps = append(apps, &app)
	}

	return apps, nil
}

// Update updates an existing app.
func (s *SQLStore) Update(ctx context.Context, app *App) error {
	if app == nil || app.ID == "" {
		return fmt.Errorf("app with ID is required")
	}

	app.UpdatedAt = time.Now()

	query := `UPDATE apps SET name = ?, config_json = ?, updated_at = ? WHERE id = ?`
	if s.dialect == "postgres" || s.dialect == "pgx" {
		query = `UPDATE apps SET name = $1, config_json = $2, updated_at = $3 WHERE id = $4`
	}

	result, err := s.db.ExecContext(ctx, query, app.Name, app.ConfigJSON, app.UpdatedAt, app.ID)
	if err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAppNotFound
	}

	slog.Info("App updated", "id", app.ID)
	return nil
}

// Delete removes an app by ID.
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM apps WHERE id = ?`
	if s.dialect == "postgres" || s.dialect == "pgx" {
		query = `DELETE FROM apps WHERE id = $1`
	}

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAppNotFound
	}

	slog.Info("App deleted", "id", id)
	return nil
}

// Exists checks if an app exists.
func (s *SQLStore) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM apps WHERE id = ?`
	if s.dialect == "postgres" || s.dialect == "pgx" {
		query = `SELECT 1 FROM apps WHERE id = $1`
	}

	var exists int
	err := s.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check app existence: %w", err)
	}

	return true, nil
}

// ErrAppNotFound is returned when an app is not found.
var ErrAppNotFound = fmt.Errorf("app not found")

// ParseConfig parses the ConfigJSON field into a config.Config struct.
// This is a forward declaration - actual implementation depends on config package.
func (a *App) ParseConfig() (map[string]any, error) {
	if a.ConfigJSON == "" {
		return nil, nil
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(a.ConfigJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// Compile-time interface check
var _ Store = (*SQLStore)(nil)
