package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds logger configuration.
type Config struct {
	Level string // debug, info, warn, error
	Env   string // development, production
}

// New creates a new structured logger based on environment.
// In development: text format for readability.
// In production: JSON format for parsing/correlation.
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.Env == "development" {
		// Text format with color for dev mode
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		// JSON format for production / CI / API
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// parseLevel converts string level to slog.Level.
func parseLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRunContext enriches a logger with run-specific fields.
func WithRunContext(log *slog.Logger, runID, provider, key string, bpm float64, bars int, seed uint64) *slog.Logger {
	return log.With(
		slog.String("run_id", runID),
		slog.String("provider", provider),
		slog.String("key", key),
		slog.Float64("bpm", bpm),
		slog.Int("bars", bars),
		slog.Uint64("seed", seed),
	)
}
