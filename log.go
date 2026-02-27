package wstun

import (
	"log/slog"
	"strings"
)

// ParseLogLevel maps a configuration log level string to a slog.Level.
// Accepted values (case-insensitive): "debug", "info", "warn", "error".
// Returns slog.LevelInfo for empty or unrecognised input.
func ParseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
