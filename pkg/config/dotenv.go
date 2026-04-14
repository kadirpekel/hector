package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads environment variables from .env files.
//
// Search order (first found wins):
//  1. Explicit path if provided
//  2. .env in current directory
//  3. .env in config file's directory (if config path provided)
//  4. .env in home directory (~/.env)
//
// This function is idempotent and safe to call multiple times.
// Existing environment variables are NOT overwritten.
func LoadDotEnv(paths ...string) error {
	// Try explicit paths first
	for _, path := range paths {
		if path != "" {
			if err := loadIfExists(path); err != nil {
				return err
			}
		}
	}

	// Try .env in current directory
	if err := loadIfExists(".env"); err != nil {
		return err
	}

	// Try .env in home directory
	if home, err := os.UserHomeDir(); err == nil {
		if err := loadIfExists(filepath.Join(home, ".env")); err != nil {
			return err
		}
	}

	return nil
}

// LoadDotEnvForConfig loads .env from the config file's directory.
func LoadDotEnvForConfig(configPath string) error {
	if configPath == "" {
		return LoadDotEnv()
	}

	// Get absolute path of config
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return LoadDotEnv()
	}

	// Try .env in config's directory
	configDir := filepath.Dir(absPath)
	envPath := filepath.Join(configDir, ".env")

	return LoadDotEnv(envPath)
}

// loadIfExists loads a .env file if it exists.
// Does NOT overwrite existing environment variables.
func loadIfExists(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, not an error
	}

	// Load without overwriting existing vars
	if err := godotenv.Load(path); err != nil {
		// Log but don't fail - .env is optional
		slog.Debug("Failed to load .env file", "path", path, "error", err)
		return nil
	}

	slog.Debug("Loaded environment from .env", "path", path)
	return nil
}

// MustLoadDotEnv loads .env files and panics on error.
// Use this in init() or main() where errors should be fatal.
func MustLoadDotEnv(paths ...string) {
	if err := LoadDotEnv(paths...); err != nil {
		panic("failed to load .env: " + err.Error())
	}
}

// ReloadDotEnv reloads environment variables from .env files.
// Unlike LoadDotEnv, this OVERWRITES existing environment variables.
// Returns the map of loaded environment variables.
func ReloadDotEnv(paths ...string) (map[string]string, error) {
	loaded := make(map[string]string)
	for _, path := range paths {
		if path == "" {
			continue
		}
		vars, err := reloadIfExists(path)
		if err != nil {
			return nil, err
		}
		// Merge vars
		for k, v := range vars {
			loaded[k] = v
		}
	}
	return loaded, nil
}

// ReloadDotEnvForConfig reloads .env from the config file's directory.
// Returns the map of loaded environment variables.
func ReloadDotEnvForConfig(configPath string) (map[string]string, error) {
	if configPath == "" {
		return nil, nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, nil
	}

	configDir := filepath.Dir(absPath)
	envPath := filepath.Join(configDir, ".env")

	return ReloadDotEnv(envPath)
}

// reloadIfExists reloads a .env file, overwriting existing variables.
// Returns the map of loaded variables.
func reloadIfExists(path string) (map[string]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	// Read first to get the map
	envMap, err := godotenv.Read(path)
	if err != nil {
		slog.Debug("Failed to read .env file", "path", path, "error", err)
		return nil, nil
	}

	// Overload (apply to process env)
	if err := godotenv.Overload(path); err != nil {
		slog.Debug("Failed to reload .env file", "path", path, "error", err)
		return nil, nil
	}

	slog.Info("Reloaded environment from .env", "path", path)
	return envMap, nil
}
