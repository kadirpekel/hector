package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Init initializes the logger with the specified level
func Init(level slog.Level, output *os.File) {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(output, opts)
	defaultLogger = slog.New(handler)

	// Set as default logger
	slog.SetDefault(defaultLogger)
}

// GetLogger returns the default slog logger
func GetLogger() *slog.Logger {
	if defaultLogger == nil {
		// Initialize with default level if not already done
		Init(slog.LevelInfo, os.Stderr)
	}
	return defaultLogger
}
