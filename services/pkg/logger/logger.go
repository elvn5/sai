// Package logger builds the structured JSON logger shared by all services.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog.Logger tagged with the service name. The level
// comes from LOG_LEVEL (debug, info, warn, error; default info).
func New(service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))})
	return slog.New(handler).With("service", service)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
