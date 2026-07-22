package wstun

import (
	"log/slog"
	"strings"
)

// ParseLogLevel maps a configuration log level string to a slog.Level.
// Accepted values (case-insensitive): "debug", "info", "warn", "error".
// Returns slog.LevelInfo for empty or unrecognised input.
func ParseLogLevel(s string) slog.Level {
	var l slog.Level
	l.UnmarshalText([]byte(strings.TrimSpace(s)))
	return l
}

// SlogLevel returns the slog.Level corresponding to the LogLevel string in ServerOptions.
func (o *ServerOptions) SlogLevel() slog.Level {
	return ParseLogLevel(o.LogLevel)
}

// SlogLevel returns the slog.Level corresponding to the LogLevel string in ClientOptions.
func (o *ClientOptions) SlogLevel() slog.Level {
	return ParseLogLevel(o.LogLevel)
}
