package wstun

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		// case-insensitive
		{"DEBUG", slog.LevelDebug},
		{"Info", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"Error", slog.LevelError},
		// mixed case
		{"DeBuG", slog.LevelDebug},
		// empty / unknown -> default info
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"trace", slog.LevelInfo},
		// whitespace
		{" info ", slog.LevelInfo},
		{" debug", slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
