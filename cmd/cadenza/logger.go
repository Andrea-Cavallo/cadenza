package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/config"
)

func setupLogger(outputDir string, appCfg *config.AppConfig) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		slog.Error("cannot create output dir for log", "err", err)
		return
	}
	logFile := appCfg.Logging.File
	if logFile == "" {
		logFile = "cadenza.log"
	}
	logPath := filepath.Join(outputDir, logFile)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("cannot open log file", "path", logPath, "err", err)
		return
	}

	level := parseSlogLevel(appCfg.Logging.Level)
	mw := io.MultiWriter(os.Stderr, f)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if appCfg.Logging.Format == "json" || appCfg.App.Env == "production" {
		handler = slog.NewJSONHandler(mw, opts)
	} else {
		handler = slog.NewTextHandler(mw, opts)
	}

	slog.SetDefault(slog.New(handler))
	slog.Info("cadenza started", "version", version, "log", logPath, "time", time.Now().Format(time.RFC3339))
}

func parseSlogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
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
