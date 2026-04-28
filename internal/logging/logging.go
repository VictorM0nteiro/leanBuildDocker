// Package logging configures the global slog logger for lbd.
//
// All packages should use slog directly (slog.Info, slog.Debug, etc.)
// rather than holding a logger reference. Configure() must be called
// once at startup before any logging happens.
package logging

import (
	"log/slog"
	"os"
)

// Configure sets up the default slog logger.
// verbose=true enables DEBUG level; otherwise INFO and above.
// Logs are always written to stderr.
func Configure(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}