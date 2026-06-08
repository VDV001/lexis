// Package logging provides the application's structured logger (slog).
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// ParseLevel maps a LOG_LEVEL string to an slog.Level, defaulting to Info for
// empty or unrecognised values.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New builds a structured logger writing to w. In production it emits JSON
// (machine-parseable for log aggregation); otherwise it emits human-readable
// text. The level is taken from the LOG_LEVEL-style string.
func New(w io.Writer, level, env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: ParseLevel(level)}
	var h slog.Handler
	if env == "production" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}
