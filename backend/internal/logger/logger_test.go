package logger

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"other":   slog.LevelInfo,
	}

	for input, expected := range tests {
		if got := parseLevel(input); got != expected {
			t.Fatalf("%s => %v, want %v", input, got, expected)
		}
	}
}

func TestNewAndWithRunContext(t *testing.T) {
	devLog := New(Config{Level: "debug", Env: "development"})
	if devLog == nil {
		t.Fatal("expected development logger")
	}

	prodLog := New(Config{Level: "info", Env: "production"})
	if prodLog == nil {
		t.Fatal("expected production logger")
	}

	enriched := WithRunContext(prodLog, "run-1", "claude", "Am", 122, 16, 42)
	if enriched == nil {
		t.Fatal("expected enriched logger")
	}
}
