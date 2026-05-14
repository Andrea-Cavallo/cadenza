package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func setupDesktopLogger(outputDir string, ctx context.Context) (string, *os.File, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create output dir: %w", err)
	}

	logPath := filepath.Join(outputDir, "logs", "Cadenza.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", nil, fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", nil, fmt.Errorf("open log file: %w", err)
	}

	writers := []io.Writer{os.Stderr, f}
	if shouldEmitWailsEvents(ctx) {
		writers = append(writers, newEventLineWriter(ctx, "log"))
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	slog.Info("Cadenza desktop started", "log", logPath, "time", time.Now().Format(time.RFC3339))
	return logPath, f, nil
}

type eventLineWriter struct {
	ctx   context.Context
	event string

	mu  sync.Mutex
	buf bytes.Buffer
	ch  chan string
}

func newEventLineWriter(ctx context.Context, event string) *eventLineWriter {
	w := &eventLineWriter{ctx: ctx, event: event, ch: make(chan string, 128)}
	go w.worker()
	return w
}

func (w *eventLineWriter) worker() {
	for line := range w.ch {
		wailsruntime.EventsEmit(w.ctx, w.event, line)
	}
}

func shouldEmitWailsEvents(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	switch fmt.Sprintf("%T", ctx) {
	case "context.backgroundCtx", "context.todoCtx":
		return false
	default:
		return true
	}
}

func (w *eventLineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.buf.Write(p); err != nil {
		return 0, err
	}

	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return 0, err
			}
			if line != "" {
				w.buf.Reset()
				_, _ = w.buf.WriteString(line)
			}
			break
		}
		w.emit(strings.TrimSpace(line))
	}

	return len(p), nil
}

func (w *eventLineWriter) emit(line string) {
	if line == "" || w.ctx == nil {
		return
	}
	select {
	case w.ch <- line:
	default:
		// Channel full — drop oldest to make room
		select {
		case <-w.ch:
		default:
		}
		w.ch <- line
	}
}
